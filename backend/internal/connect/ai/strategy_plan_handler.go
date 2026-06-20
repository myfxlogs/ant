package ai

import (
	"context"
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

	// ── Tool pipeline (registry-driven, extensible) ──
	NewToolRegistry(s.backtestRepo).Execute(ctx, ToolInput{
		Code: code, Symbol: m.Symbol, Timeframe: m.Timeframe, UserID: userID,
	}, func(chunk *antv1.ExecutePlanChunk) error {
		return stream.Send(chunk)
	})

	s.persistExchange(ctx, userID, m.ConversationId, m.Plan, code, m.FeedbackMessage)
	return stream.Send(&antv1.ExecutePlanChunk{Phase: "done", Code: code, PreviousCode: m.PreviousCode, Analysis: analysis})
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
		return "你是一个量化策略规划师。根据用户需求，输出一个编号的执行计划。每行一个步骤，用 1. 2. 3. 开头。包含：策略类型、关键指标、入场/出场逻辑、风控措施。直接输出编号列表，不要任何其他文字。例如：\n1. 使用 EMA20/EMA50 判断趋势方向\n2. RSI < 30 时入场做多\n3. 设置 2% 止损和 4% 止盈"
	case "zh-tw":
		return "你是一個量化策略規劃師。根據用戶需求，輸出一個編號的執行計畫。每行一個步驟，用 1. 2. 3. 開頭。包含：策略類型、關鍵指標、入場/出場邏輯、風控措施。直接輸出編號列表，不要任何其他文字。"
	default:
		return "You are a quantitative strategy planner. Output a numbered execution plan. One step per line, starting with 1. 2. 3. Include: strategy type, key indicators, entry/exit logic, risk controls. Output ONLY the numbered list, nothing else. Example:\n1. Use EMA20/EMA50 for trend direction\n2. Enter long when RSI < 30\n3. Set 2% stop loss and 4% take profit"
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

func diagnoseAndFixPrompt(lang string) string {
	switch lang {
	case "zh":
		return "你是策略迭代助手。用户会发送后续消息——可能是问题、反馈或修改要求。\n\n" +
			"规则：\n" +
			"1. 先判断用户意图：纯问题 → 只回答，不生成代码。修改要求 → 先解释，再给新代码。\n" +
			"2. 如果只是问问题（如\"回测结果是什么\"），直接回答问题即可。不要生成代码。\n" +
			"3. 如果要求修改，在现有代码基础上改，不要完全重写。\n" +
			"4. 如果要生成代码，先写分析/解释，然后输出代码。代码不要有 markdown 格式。"
	case "zh-tw":
		return "你是策略迭代助手。用戶會發送後續訊息——可能是問題、回饋或修改要求。\n\n" +
			"規則：\n" +
			"1. 先判斷用戶意圖：純問題 → 只回答，不生成程式碼。修改要求 → 先解釋，再給新程式碼。\n" +
			"2. 如果只是問問題（如\"回測結果是什麼\"），直接回答問題即可。不要生成程式碼。\n" +
			"3. 如果要求修改，在現有程式碼基礎上改，不要完全重寫。\n" +
			"4. 如果要生成程式碼，先寫分析/解釋，然後輸出程式碼。程式碼不要有 markdown 格式。"
	default:
		return "You are a strategy iteration assistant. The user sends follow-up messages — questions, feedback, or change requests.\n\n" +
			"Rules:\n" +
			"1. First classify the intent: pure question → just answer, no code. Change request → explain then give new code.\n" +
			"2. If it's just a question (e.g. 'what are the backtest results?'), answer it directly. Do NOT generate code.\n" +
			"3. If it's a change request, modify the existing code — do not rewrite from scratch.\n" +
			"4. If you generate code, write analysis/explanation first, then output code. Code must not have markdown formatting."
	}
}
