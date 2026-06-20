package ai

import (
	internalai "anttrader/internal/ai"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
)

// StrategyPlanServer implements ant.v1.StrategyPlanServiceHandler (both AnalyzePlan and ExecutePlan).
type StrategyPlanServer struct {
	systemSvc      *systemai.Service
	templatesRepo  *repository.AIStrategyTemplatesRepository
	backtestRepo   *repository.BacktestRunRepository
	convRepo       *repository.AIConversationRepository
	marketDataRepo repository.MarketDataStore
	memoryExec     func(ctx context.Context, sql string, args ...any) error
	memoryQuery    func(ctx context.Context, sql string, args ...any) (string, error)
	intentAnalyzer *internalai.IntentAnalyzer
	log            *zap.Logger
}

var _ antv1c.StrategyPlanServiceHandler = (*StrategyPlanServer)(nil)

// SetPoolAdapter wires the PG pool for memory tools (remember/recall).
// Call this after construction to enable memory persistence.
func (s *StrategyPlanServer) SetPoolAdapter(execFn func(ctx context.Context, sql string, args ...any) error, queryFn func(ctx context.Context, sql string, args ...any) (string, error)) {
	s.memoryExec = execFn
	s.memoryQuery = queryFn
}

func NewStrategyPlanServer(
	systemSvc *systemai.Service,
	templatesRepo *repository.AIStrategyTemplatesRepository,
	backtestRepo *repository.BacktestRunRepository,
	convRepo *repository.AIConversationRepository,
	marketDataRepo repository.MarketDataStore,
	log *zap.Logger,
) *StrategyPlanServer {
	analyzer := internalai.NewIntentAnalyzer(func(ctx context.Context, userID uuid.UUID, messages []internalai.ChatMessage, _ string) (string, error) {
		sysMsgs := make([]systemai.ChatMessage, len(messages))
		for i, m := range messages {
			sysMsgs[i] = systemai.ChatMessage{Role: m.Role, Content: m.Content}
		}
		return systemSvc.ChatCompletion(ctx, userID, sysMsgs)
	})
	return &StrategyPlanServer{
		systemSvc: systemSvc, templatesRepo: templatesRepo, backtestRepo: backtestRepo,
		convRepo: convRepo, marketDataRepo: marketDataRepo, intentAnalyzer: analyzer, log: log,
		// memoryExec and memoryQuery are set via SetPoolAdapter after construction
	}
}

// ── AnalyzePlan: understanding phase ──

func (s *StrategyPlanServer) AnalyzePlan(
	ctx context.Context,
	req *connect.Request[antv1.AnalyzePlanRequest],
	stream *connect.ServerStream[antv1.AnalyzePlanChunk],
) error {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return err
	}
	m := req.Msg
	lang := LangFromAccept(req.Header().Get("Accept-Language"))

	intent, err := s.intentAnalyzer.Analyze(ctx, userID, m.Message, m.Symbol, m.Timeframe, lang, clarifyLangDirective(lang))
	if err != nil {
		s.log.Warn("plan analysis failed", zap.Error(err))
		return stream.Send(&antv1.AnalyzePlanChunk{Phase: "error", Error: systemai.FriendlyError(err)})
	}

	plan := intent.Plan
	if plan == "" {
		plan = buildFallbackPlan(intent)
	}

	sysPrompt := internalai.AgentPrompt(lang) + "\n\n## 当前任务：制定执行计划（绝对不要生成代码！）\n分析用户需求，输出一个纯文本的执行计划。每行一个步骤，用 1. 2. 3. 开头。只讨论策略逻辑和方案，不要写任何代码。"
	userPrompt := fmt.Sprintf("用户需求: %s\n\n分析结果: 策略类型=%s, 方向=%s, 风险=%s\n请用1-2句话生成一个简明的执行计划。",
		m.Message, intent.StrategyFamily, intent.TradeDirection, intent.RiskLevel)

	var fullPlan strings.Builder
	err = s.systemSvc.ChatCompletionStream(ctx, userID,
		[]systemai.ChatMessage{{Role: "system", Content: sysPrompt}, {Role: "user", Content: userPrompt}},
		func(chunk systemai.ChatStreamChunk) error {
			fullPlan.WriteString(chunk.Content)
			send := &antv1.AnalyzePlanChunk{Phase: "analyzing", Delta: chunk.Content}
			if chunk.Done {
				send.Phase = "plan_ready"
				send.Plan = fullPlan.String()
			}
			return stream.Send(send)
		})
	if err != nil {
		return stream.Send(&antv1.AnalyzePlanChunk{Phase: "error", Error: systemai.FriendlyError(err)})
	}

	s.persistPlan(ctx, userID, m.ConversationId, m.Message, fullPlan.String())
	return nil
}

// ── Conversate: unified agent conversation (Claude Code style) ──

func (s *StrategyPlanServer) Conversate(
	ctx context.Context,
	req *connect.Request[antv1.ConversateRequest],
	stream *connect.ServerStream[antv1.ConversateChunk],
) error {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return err
	}
	m := req.Msg
	lang := LangFromAccept(req.Header().Get("Accept-Language"))

	registry := NewToolRegistry(s.backtestRepo, s.marketDataRepo)
	registry.WireMemoryDB(s.memoryExec, s.memoryQuery)

	sysPrompt := internalai.AgentPrompt(lang)

	history := s.loadHistory(ctx, userID, m.ConversationId, 20)

	// Inject workspace context into the user prompt
	ctxInfo := fmt.Sprintf("[当前工作区: 品种=%s, 周期=%s]", m.Symbol, m.Timeframe)
	if m.Symbol != "" && m.Timeframe != "" {
		ctxInfo = fmt.Sprintf("[当前工作区: 品种=%s, 周期=%s。你可以直接使用这些信息，无需询问用户。]", m.Symbol, m.Timeframe)
	}
	userPrompt := ctxInfo + " " + m.Message
	oldCode := m.CurrentCode

	chunk := func(delta string) error {
		return stream.Send(&antv1.ConversateChunk{Phase: "thinking", Delta: delta})
	}
	toolEvt := func(tc *antv1.ToolCall, tr *antv1.ToolResult) error {
		return stream.Send(&antv1.ConversateChunk{Phase: "tool_result", ToolCall: tc, ToolResult: tr})
	}

	loop := NewAgentLoop(registry,
		func(ctx context.Context, msgs []systemai.ChatMessage, onChunk func(systemai.ChatStreamChunk) error) error {
			return s.systemSvc.ChatCompletionStream(ctx, userID, msgs, onChunk)
		},
		chunk, toolEvt,
	)

	raw, err := loop.RunWithHistory(ctx, sysPrompt, userPrompt, history, userID)
	if err != nil {
		return stream.Send(&antv1.ConversateChunk{Phase: "done", Error: systemai.FriendlyError(err)})
	}

	// Extract plan/code from the response
	plan := extractPlan(raw)
	code := ExtractCode(raw)
	// If less than 30% of response, it's discussion, not code output
	if code != "" && len(code) < len(raw)/3 {
		code = ""
	}

	// Compliance check → auto-fix loop (Claude Code style: try → fail → fix → retry)
	passed := false
	if code != "" {
		const maxRetries = 3
		for retry := 0; retry < maxRetries; retry++ {
			compliant := &complianceTool{}
			result := compliant.Run(ctx, ToolInput{Code: code})
			_ = stream.Send(&antv1.ConversateChunk{
				Phase: "tool_result",
				ToolResult: &antv1.ToolResult{CallId: "call_compliance", Name: "compliance_check", Success: result.Success, Error: result.Error},
			})

			if result.Success {
				passed = true
				break
			}

			if retry < maxRetries-1 {
				fixPrompt := fmt.Sprintf("[SYSTEM] 合规检查失败: %s\n请修复以上问题并重新输出完整代码。", result.Error)
				fixed, fixErr := loop.RunWithHistory(ctx, sysPrompt, fixPrompt, append(history,
					systemai.ChatMessage{Role: "assistant", Content: raw},
				), userID)
				if fixErr != nil { break }
				code = ExtractCode(fixed)
				if code == "" { break }
			}
		}

		// Backtest only if compliance passed
		if passed && code != "" {
			bt := &backtestTool{repo: s.backtestRepo}
			btResult := bt.Run(ctx, ToolInput{Code: code, Symbol: m.Symbol, Timeframe: m.Timeframe, UserID: userID})
			outJSON, _ := json.Marshal(btResult.Output)
			_ = stream.Send(&antv1.ConversateChunk{
				Phase: "tool_result",
				ToolResult: &antv1.ToolResult{CallId: "call_backtest", Name: "backtest", Success: btResult.Success, OutputJson: string(outJSON), Error: btResult.Error},
			})
		}
	}

	s.persistExchange(ctx, userID, m.ConversationId, plan, code, m.Message)

	return stream.Send(&antv1.ConversateChunk{Phase: "done", Code: code, Plan: plan, PreviousCode: oldCode})
}

// extractPlan pulls the first non-code text that looks like a plan from the response.
func extractPlan(raw string) string {
	code := ExtractCode(raw)
	if code == "" {
		return raw
	}
	if idx := strings.Index(raw, code); idx > 0 {
		return strings.TrimSpace(raw[:idx])
	}
	return ""
}

// ── ExecutePlan: execution phase ──

func (s *StrategyPlanServer) ExecutePlan(
	ctx context.Context,
	req *connect.Request[antv1.ExecutePlanRequest],
	stream *connect.ServerStream[antv1.ExecutePlanChunk],
) error {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return err
	}
	m := req.Msg
	lang := LangFromAccept(req.Header().Get("Accept-Language"))

	_ = stream.Send(&antv1.ExecutePlanChunk{Phase: "generating"})

	registry := NewToolRegistry(s.backtestRepo, s.marketDataRepo)
	registry.WireMemoryDB(s.memoryExec, s.memoryQuery)
	sysPrompt := internalai.AgentPrompt(lang) + "\n\n## 当前任务：生成或修改策略代码\n根据执行计划和用户的最新消息，输出完整的 Python 策略代码。你可以使用 [TOOL: name args] 来查询信息。"
	userPrompt := buildExecuteUserPrompt(m)

	// Agent Loop: LLM ↔ Tools (Claude Code / OpenAI Agents SDK pattern)
	loop := NewAgentLoop(registry,
		func(ctx context.Context, messages []systemai.ChatMessage, onChunk func(systemai.ChatStreamChunk) error) error {
			return s.systemSvc.ChatCompletionStream(ctx, userID, messages, onChunk)
		},
		func(delta string) error {
			return stream.Send(&antv1.ExecutePlanChunk{Phase: "generating", Delta: delta})
		},
		func(tc *antv1.ToolCall, tr *antv1.ToolResult) error {
			return stream.Send(&antv1.ExecutePlanChunk{Phase: "tool_result", ToolCall: tc, ToolResult: tr})
		},
	)

	raw, err := loop.Run(ctx, sysPrompt, userPrompt, userID)
	if err != nil {
		return stream.Send(&antv1.ExecutePlanChunk{Phase: "error", Error: systemai.FriendlyError(err)})
	}

	code := ExtractCode(raw)
	// If code is less than 30% of the response, it's likely just code snippets
	// in a discussion — treat the whole response as analysis, not code output.
	if code != "" && len(code) < len(raw)/3 {
		code = ""
	}
	var analysis string
	if m.FeedbackMessage != "" && code != "" && raw != code {
		if idx := strings.Index(raw, code); idx > 0 {
			analysis = strings.TrimSpace(raw[:idx])
		}
	}

	// ── Auto-run tools after code generation ──
	registry.Execute(ctx, ToolInput{Code: code, Symbol: m.Symbol, Timeframe: m.Timeframe, UserID: userID},
		func(chunk *antv1.ExecutePlanChunk) error { return stream.Send(chunk) })

	s.persistExchange(ctx, userID, m.ConversationId, m.Plan, code, m.FeedbackMessage)
	return stream.Send(&antv1.ExecutePlanChunk{Phase: "done", Code: code, PreviousCode: m.PreviousCode, Analysis: analysis})
}


// loadHistory loads recent conversation messages as context for the AgentLoop.
func (s *StrategyPlanServer) loadHistory(ctx context.Context, userID uuid.UUID, convID string, limit int) []systemai.ChatMessage {
	if convID == "" {
		return nil
	}
	cid, err := uuid.Parse(convID)
	if err != nil {
		return nil
	}
	msgs, err := s.convRepo.GetMessages(ctx, userID, cid)
	if err != nil || len(msgs) == 0 {
		return nil
	}
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	out := make([]systemai.ChatMessage, len(msgs))
	for i, m := range msgs {
		role := m.Role
		if role == "assistant" {
			role = "assistant"
		}
		out[i] = systemai.ChatMessage{Role: role, Content: m.Content}
	}
	return out
}

func (s *StrategyPlanServer) persistPlan(ctx context.Context, userID uuid.UUID, convID, userMsg, plan string) {
	if convID == "" || userMsg == "" || plan == "" {
		return
	}
	cid, err := uuid.Parse(convID)
	if err != nil {
		return
	}
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", userMsg)
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "assistant", "[PLAN]\n"+plan)
	_ = s.convRepo.Touch(ctx, cid, userID)
}

func (s *StrategyPlanServer) persistExchange(ctx context.Context, userID uuid.UUID, convID, plan, code, feedback string) {
	if convID == "" {
		return
	}
	cid, err := uuid.Parse(convID)
	if err != nil {
		return
	}
	if feedback != "" {
		_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", feedback)
	} else {
		_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", plan)
	}
	tag := "[DISCUSSION]"
	if code != "" {
		tag = "[CODE]"
	} else if plan != "" {
		tag = "[PLAN]"
	}
	content := code
	if content == "" {
		content = plan
	}
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "assistant", tag+"\n"+content)
	_ = s.convRepo.Touch(ctx, cid, userID)
}

// ── Diagnose: analysis only, no code generation ──

func (s *StrategyPlanServer) Diagnose(
	ctx context.Context,
	req *connect.Request[antv1.DiagnoseRequest],
	stream *connect.ServerStream[antv1.AnalyzePlanChunk],
) error {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return err
	}
	m := req.Msg
	lang := LangFromAccept(req.Header().Get("Accept-Language"))

	sysPrompt := internalai.AgentPrompt(lang) + "\n\n## 当前任务：诊断问题，提出建议\n分析用户的反馈和回测数据，给出具体的修改建议。每行一个建议，用 - 开头。不要生成代码，只输出诊断分析和建议。"
	userPrompt := fmt.Sprintf(
		"## 执行计划\n%s\n\n## 当前代码\n```python\n%s\n```\n\n## 回测数据\n%s\n\n## 用户反馈\n%s\n\n请诊断问题并给出修改建议。每行一个建议，用 - 开头。",
		m.Plan, m.CurrentCode, m.BacktestMetricsJson, m.FeedbackMessage,
	)

	var fullBuf strings.Builder
	err = s.systemSvc.ChatCompletionStream(ctx, userID,
		[]systemai.ChatMessage{{Role: "system", Content: sysPrompt}, {Role: "user", Content: userPrompt}},
		func(chunk systemai.ChatStreamChunk) error {
			fullBuf.WriteString(chunk.Content)
			send := &antv1.AnalyzePlanChunk{Phase: "analyzing", Delta: chunk.Content}
			if chunk.Done {
				send.Phase = "plan_ready"
				send.Plan = fullBuf.String()
			}
			return stream.Send(send)
		})
	if err != nil {
		return stream.Send(&antv1.AnalyzePlanChunk{Phase: "error", Error: systemai.FriendlyError(err)})
	}

	s.persistDiagnose(ctx, userID, m.ConversationId, m.FeedbackMessage, fullBuf.String())
	return nil
}

func (s *StrategyPlanServer) persistDiagnose(ctx context.Context, userID uuid.UUID, convID, question, answer string) {
	if convID == "" {
		return
	}
	cid, err := uuid.Parse(convID)
	if err != nil {
		return
	}
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", question)
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "assistant", "[DIAGNOSIS]\n"+answer)
	_ = s.convRepo.Touch(ctx, cid, userID)
}






func buildFallbackPlan(intent *internalai.IntentResult) string {
	s := "Strategy Plan:\n"
	if intent.StrategyFamily != "" && intent.StrategyFamily != "unknown" {
		s += "- Type: " + intent.StrategyFamily + "\n"
	}
	if intent.TradeDirection != "" && intent.TradeDirection != "unknown" {
		s += "- Direction: " + intent.TradeDirection + "\n"
	}
	if intent.RiskLevel != "" && intent.RiskLevel != "unknown" {
		s += "- Risk: " + intent.RiskLevel + "\n"
	}
	if len(intent.EntrySignals) > 0 {
		s += "- Entry: " + strings.Join(intent.EntrySignals, ", ") + "\n"
	}
	if len(intent.ExitSignals) > 0 {
		s += "- Exit: " + strings.Join(intent.ExitSignals, ", ") + "\n"
	}
	s += "- Default position sizing and 2% stop loss will be applied."
	return s
}

func buildExecuteUserPrompt(m *antv1.ExecutePlanRequest) string {
	if m.FeedbackMessage != "" {
		p := "## 执行计划\n" + m.Plan + "\n\n"
		p += "## 用户的后续消息\n" + m.FeedbackMessage + "\n\n"
		if m.PreviousCode != "" {
			p += "## 当前的策略代码\n```python\n" + m.PreviousCode + "\n```\n\n"
		}
		if m.BacktestMetricsJson != "" {
			p += "## 回测数据\n" + m.BacktestMetricsJson + "\n"
		}
		return p
	}
	return m.Plan
}

