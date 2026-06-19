package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
)

// StrategyPlanServer implements ant.v1.StrategyPlanServiceHandler (both AnalyzePlan and ExecutePlan).
type StrategyPlanServer struct {
	systemSvc      *systemai.Service
	templatesRepo  *repository.AIStrategyTemplatesRepository
	backtestRepo   *repository.BacktestRunRepository
	convRepo       *repository.AIConversationRepository
	intentAnalyzer *ai.IntentAnalyzer
	log            *zap.Logger
}

var _ antv1c.StrategyPlanServiceHandler = (*StrategyPlanServer)(nil)

func NewStrategyPlanServer(
	systemSvc *systemai.Service,
	templatesRepo *repository.AIStrategyTemplatesRepository,
	backtestRepo *repository.BacktestRunRepository,
	convRepo *repository.AIConversationRepository,
	log *zap.Logger,
) *StrategyPlanServer {
	analyzer := ai.NewIntentAnalyzer(func(ctx context.Context, userID uuid.UUID, messages []ai.ChatMessage, _ string) (string, error) {
		sysMsgs := make([]systemai.ChatMessage, len(messages))
		for i, m := range messages {
			sysMsgs[i] = systemai.ChatMessage{Role: m.Role, Content: m.Content}
		}
		return systemSvc.ChatCompletion(ctx, userID, sysMsgs)
	})
	return &StrategyPlanServer{
		systemSvc: systemSvc, templatesRepo: templatesRepo, backtestRepo: backtestRepo,
		convRepo: convRepo, intentAnalyzer: analyzer, log: log,
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

	sysPrompt := planSystemPrompt(lang)
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

	sysPrompt := codeFromPlanPrompt(lang)
	userPrompt := buildExecuteUserPrompt(m)
	if m.FeedbackMessage != "" {
		sysPrompt = diagnoseAndFixPrompt(lang)
	}

	var fullBuf strings.Builder
	isFeedback := m.FeedbackMessage != ""
	err = s.systemSvc.ChatCompletionStream(ctx, userID,
		[]systemai.ChatMessage{{Role: "system", Content: sysPrompt}, {Role: "user", Content: userPrompt}},
		func(chunk systemai.ChatStreamChunk) error {
			fullBuf.WriteString(chunk.Content)
			return stream.Send(&antv1.ExecutePlanChunk{Phase: "generating", Delta: chunk.Content})
		})
	if err != nil {
		return stream.Send(&antv1.ExecutePlanChunk{Phase: "error", Error: systemai.FriendlyError(err)})
	}

	raw := fullBuf.String()
	code := ExtractCode(raw)
	var analysis string
	if isFeedback && code != "" && raw != code {
		// Split AI's explanation from the code block
		idx := strings.Index(raw, code)
		if idx > 0 {
			analysis = strings.TrimSpace(raw[:idx])
		}
	}

	// ToolCall: compliance_check
	call1 := "call_compliance"
	_ = stream.Send(&antv1.ExecutePlanChunk{
		Phase: "tool_call", ToolCall: &antv1.ToolCall{CallId: call1, Name: "compliance_check", ParamsJson: `{}`},
	})
	blocks, warns := ai.NewCodeComplianceScanner().Scan(code)
	allIssues := append(blocks, warns...)
	var issueProtos []*antv1.ComplianceIssue
	for _, iss := range allIssues {
		issueProtos = append(issueProtos, &antv1.ComplianceIssue{
			Rule: iss.RuleName, Message: iss.Message, Severity: iss.Severity, Line: int32(iss.Line),
		})
	}
	res1, _ := json.Marshal(&antv1.ComplianceResult{Passed: len(blocks) == 0, Issues: issueProtos})
	_ = stream.Send(&antv1.ExecutePlanChunk{
		Phase: "tool_result", ToolResult: &antv1.ToolResult{CallId: call1, Name: "compliance_check", Success: len(blocks) == 0, OutputJson: string(res1)},
	})

	// ToolCall: backtest
	call2 := "call_backtest"
	_ = stream.Send(&antv1.ExecutePlanChunk{
		Phase: "tool_call", ToolCall: &antv1.ToolCall{CallId: call2, Name: "backtest", ParamsJson: `{}`},
	})
	runID, btErr := s.triggerBacktest(ctx, userID, code, m.Symbol, m.Timeframe)
	if btErr == "" {
		outJSON, _ := json.Marshal(map[string]string{"run_id": runID})
		_ = stream.Send(&antv1.ExecutePlanChunk{
			Phase: "tool_result", ToolResult: &antv1.ToolResult{CallId: call2, Name: "backtest", Success: true, OutputJson: string(outJSON)},
		})
	} else {
		_ = stream.Send(&antv1.ExecutePlanChunk{
			Phase: "tool_result", ToolResult: &antv1.ToolResult{CallId: call2, Name: "backtest", Success: false, Error: btErr},
		})
	}

	s.persistExchange(ctx, userID, m.ConversationId, m.Plan, code, m.FeedbackMessage)
	return stream.Send(&antv1.ExecutePlanChunk{Phase: "done", Code: code, PreviousCode: m.PreviousCode, Analysis: analysis})
}

func (s *StrategyPlanServer) triggerBacktest(ctx context.Context, userID uuid.UUID, code, symbol, timeframe string) (string, string) {
	if symbol == "" {
		symbol = "EURUSD"
	}
	if timeframe == "" {
		timeframe = "1h"
	}
	runID, err := CreateBacktestRun(ctx, s.backtestRepo, userID, code, symbol, timeframe)
	if err != nil {
		return "", err.Error()
	}
	return runID, ""
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
		_, _ = s.convRepo.AddMessage(ctx, userID, cid, "user", "[EXECUTED]\n"+plan)
	}
	_, _ = s.convRepo.AddMessage(ctx, userID, cid, "assistant", "[CODE]\n"+code)
	_ = s.convRepo.Touch(ctx, cid, userID)
}

func planSystemPrompt(lang string) string {
	switch lang {
	case "zh":
		return "你是一个量化策略规划师。根据用户需求和已有分析，用简洁的中文写出1-2句话的执行计划。包含：策略类型、关键指标、入场/出场逻辑、风控措施。直接输出计划，不要有任何前缀。"
	case "zh-tw":
		return "你是一個量化策略規劃師。根據用戶需求和已有分析，用簡潔的繁體中文寫出1-2句話的執行計劃。包含：策略類型、關鍵指標、入場/出場邏輯、風控措施。直接輸出計劃，不要有任何前綴。"
	default:
		return "You are a quantitative strategy planner. Based on the user's request and analysis, write a concise 1-2 sentence execution plan in English. Include: strategy type, key indicators, entry/exit logic, risk controls. Output the plan directly, no prefix."
	}
}

func codeFromPlanPrompt(lang string) string {
	switch lang {
	case "zh":
		return "你是一个专业的量化策略实现工程师。根据执行计划生成完整的 Python 策略代码。只输出代码，不要有任何解释或 markdown 格式。使用 run_context 模式。包含完整的止损止盈逻辑和仓位管理。"
	case "zh-tw":
		return "你是一個專業的量化策略實現工程師。根據執行計劃生成完整的 Python 策略程式碼。只輸出程式碼，不要有任何解釋或 markdown 格式。使用 run_context 模式。包含完整的止損止盈邏輯和倉位管理。"
	default:
		return "You are a professional quantitative strategy engineer. Generate complete Python strategy code from the execution plan. Output ONLY code, no explanations or markdown formatting. Use run_context mode. Include full stop-loss/take-profit logic and position sizing."
	}
}

func buildFallbackPlan(intent *ai.IntentResult) string {
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
		p += "## 用户的问题或反馈\n" + m.FeedbackMessage + "\n\n"
		if m.PreviousCode != "" {
			p += "## 当前的策略代码（请在此基础修改，不要完全重写）\n```python\n" + m.PreviousCode + "\n```\n\n"
		}
		if m.BacktestMetricsJson != "" {
			p += "## 回测数据\n" + m.BacktestMetricsJson + "\n"
		}
		p += "\n请先直接回答用户的问题（解释原因），然后给出修改后的完整代码。不要完全重写，只修改需要改的部分。"
		return p
	}
	return m.Plan
}

func diagnoseAndFixPrompt(lang string) string {
	switch lang {
	case "zh":
		return "你是一个专业的量化策略诊断专家。用户会提出问题或修改要求，并提供当前代码和回测数据。你必须：1) 先直接回答用户的问题，用简洁的中文解释原因 2) 然后给出修改后的完整 Python 代码。重要：在现有代码基础上修改，不要完全重写。对于用户的问题（如\"为什么拿不到回测数据\"），先解释原因再给方案。代码不要有任何 markdown 格式。"
	case "zh-tw":
		return "你是一個專業的量化策略診斷專家。用戶會提出問題或修改要求，並提供當前程式碼和回測數據。你必須：1) 先直接回答用戶的問題，用簡潔的繁體中文解釋原因 2) 然後給出修改後的完整 Python 程式碼。重要：在現有程式碼基礎上修改，不要完全重寫。對於用戶的問題（如「為什麼拿不到回測數據」），先解釋原因再給方案。程式碼不要有任何 markdown 格式。"
	default:
		return "You are a professional quantitative strategy diagnostician. The user will ask questions or request changes, providing current code and backtest data. You MUST: 1) First directly answer the user's question in concise English 2) Then provide the modified complete Python code. IMPORTANT: modify the existing code, do NOT rewrite from scratch. For questions like 'why can't I get backtest data', explain the reason first then propose the fix. Code must not have markdown formatting."
	}
}
