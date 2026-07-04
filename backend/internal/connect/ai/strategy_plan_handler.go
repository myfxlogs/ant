package ai

import (
	"anttrader/internal/ai"
	"context"
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

// chatDiscipline defines the thinking and verification discipline for the chat agent.
// Separate from the compilation contract (PythonSubsetRules) — prompt engineering
// can evolve independently for chat vs. generator contexts.
const chatDiscipline = `
## Thinking Discipline (CRITICAL)

Before EVERY significant action (generating code, calling a tool, analyzing results), you MUST output a [THINK] block:

[THINK]
1. Current state: (what just happened? what do I know?)
2. Next action: (what am I about to do?)
3. Reason: (why this specific action?)
[/THINK]

Then immediately take the action. This prevents impulsive decisions and helps you catch mistakes before they happen.

## Pre-Compile Self-Verification (MANDATORY)

Before calling compile_python, silently run through this checklist. If ANY item fails, fix the code first:

□ Every __init__ param has type annotation AND default value?
□ __init__ has -> None return type?
□ Every method has a return type annotation?
□ Every local variable has a type annotation?
□ All prices/volumes/P&L use Decimal (not float)?
□ Only import is "from decimal import Decimal"?
□ No forbidden syntax (lambda, try/except, f-strings, list comprehensions)?

## Error Memory — Common Mistakes

These are the most frequent compile errors. Check FIRST before generating code:
- FORGETTING -> None on __init__
- FORGETTING type annotation on local variables
- Using float for prices instead of Decimal
- Missing -> None on on_deinit
- Importing anything other than Decimal
- Using f-strings or list comprehensions

If you just fixed a compile error, REMEMBER what caused it. Do NOT repeat the same mistake.`

// pythonAgentPrompt is the system prompt for the Python strategy agent in chat context.
const pythonAgentPrompt = `You are a quantitative strategy developer on the AntTrader platform.
Your task is to generate Python trading strategies from natural language descriptions.

` + ai.PythonSubsetRules + `
` + chatDiscipline + `

## Workflow

You have tools available (the system will invoke them automatically):
- **compile_python** — Compile your Python code. Returns success + coverage score, or a specific error.
- **read_kline** — Query K-line data statistics for a symbol/timeframe.
- **read_backtest_log** — Read the most recent backtest status and errors.
- **remember / recall** — Store and retrieve user preferences.

Follow this workflow:
1. **Discuss first.** Analyze the strategy request, confirm understanding, propose a numbered plan.
2. **[THINK]** Before generating code, think through the strategy logic.
3. **Generate code.** Output complete Python code in a markdown code block.
4. **Self-verify.** Run through the pre-compile checklist. Fix any issues silently.
5. **Compile.** Call compile_python to verify.
6. **Fix if needed.** If compilation fails: [THINK] read the error, understand the root cause, fix the specific issue, self-verify again, compile again. Do NOT blindly guess.
7. The user will run backtest manually — interpret the results when they appear.

## When to Use Tools (CRITICAL — VIOLATION = FAILURE)

**General rule: NEVER ask "should I use tool X?" or "how many bars do you want?" Just call the tool. The workspace has symbol/timeframe. You have the tools. Use them.**

| User says | You do |
|-----------|--------|
| "什么行情?" "图表显示?" "帮我看看盘" | → call read_kline IMMEDIATELY |
| "回测结果?" "为什么回测失败?" | → call read_backtest_log IMMEDIATELY |
| "帮我编译" "验证一下代码" | → call compile_python IMMEDIATELY |
| "记住我偏好..." "保存这个参数" | → call remember IMMEDIATELY |
| "我之前用什么参数?" | → call recall IMMEDIATELY |
| "我有哪些策略?" | → call list_strategies IMMEDIATELY |
| "写一个策略" "生成代码" | → discuss plan FIRST, then generate |

**Only strategy generation needs discussion. Everything else: tool first, talk later.**

## Conversation Rules
- [THINK] before acting.
- Tool first, talk later. Only strategy generation needs discussion.
- Be honest about limitations.
- Explain your reasoning for indicator choices and parameter values.
- Use sensible defaults for unspecified parameters.
- Iterate on existing code rather than rewriting from scratch.
- Be honest about limitations — if something is infeasible, say so.
- After calling a tool, wait for the real result. Do not predict tool output.`

// StrategyPlanServer implements ant.v1.StrategyPlanServiceHandler (both AnalyzePlan and ExecutePlan).
type StrategyPlanServer struct {
	systemSvc      *systemai.Service
	backtestRepo   *repository.BacktestRunRepository
	convRepo       *repository.AIConversationRepository
	marketDataRepo repository.MarketDataStore
	memoryExec     func(ctx context.Context, sql string, args ...any) error
	memoryQuery    func(ctx context.Context, sql string, args ...any) (string, error)
	intentAnalyzer *ai.IntentAnalyzer
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
	backtestRepo *repository.BacktestRunRepository,
	convRepo *repository.AIConversationRepository,
	marketDataRepo repository.MarketDataStore,
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
		systemSvc: systemSvc, backtestRepo: backtestRepo,
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

	sysPrompt := ai.AgentPrompt(lang) + "\n\n## 当前任务：制定执行计划（绝对不要生成代码！）\n分析用户需求，输出一个纯文本的执行计划。每行一个步骤，用 1. 2. 3. 开头。只讨论策略逻辑和方案，不要写任何代码。"
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

// ── Conversate: unified Python strategy agent conversation ──

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

	registry := NewEmptyToolRegistry()
	registry.AddPreTool(&ReadKlineTool{repo: s.marketDataRepo})
	registry.AddPreTool(&ReadBacktestLogTool{repo: s.backtestRepo})
	registry.AddPreTool(&compilePythonChatTool{})
	registry.WireMemoryDB(s.memoryExec, s.memoryQuery)

	// Build Python agent system prompt with workspace context.
	sysPrompt := pythonAgentPrompt
	if m.Symbol != "" && m.Timeframe != "" {
		sysPrompt += fmt.Sprintf("\n\n## Current Workspace\nSymbol: %s\nTimeframe: %s", m.Symbol, m.Timeframe)
	}

	history := s.loadHistory(ctx, userID, m.ConversationId, 5)

	// Inject workspace context into the user prompt.
	ctxInfo := fmt.Sprintf("[当前工作区: 品种=%s, 周期=%s]", m.Symbol, m.Timeframe)
	if m.CurrentCode != "" {
		ctxInfo += "\n\n## 当前策略代码\n```python\n" + m.CurrentCode + "\n```"
	}
	userPrompt := ctxInfo + " " + m.Message

	chunk := func(delta string) error {
		return stream.Send(&antv1.ConversateChunk{Phase: "thinking", Delta: delta})
	}
	toolEvt := func(tc *antv1.ToolCall, tr *antv1.ToolResult) error {
		return stream.Send(&antv1.ConversateChunk{Phase: "tool_result", ToolCall: tc, ToolResult: tr})
	}

	loop := NewAgentLoop(registry,
		func(llmCtx context.Context, msgs []systemai.ChatMessage, tools []systemai.ToolDefinition, onChunk func(systemai.ChatStreamChunk) error) error {
			return s.systemSvc.ChatCompletionStreamWithTools(llmCtx, userID, msgs, tools, onChunk)
		},
		chunk, toolEvt,
	)
	loop.SetCurrentCode(m.CurrentCode)

	raw, err := loop.RunWithHistory(ctx, sysPrompt, userPrompt, history, userID)
	if err != nil {
		return stream.Send(&antv1.ConversateChunk{Phase: "done", Error: systemai.FriendlyError(err)})
	}

	// Extract Python code from the response.
	code := ExtractCode(raw)
	if code != "" && len(code) < len(raw)/3 {
		code = "" // discussion, not code output
	}

	s.persistExchange(ctx, userID, m.ConversationId, "", code, m.Message)

	return stream.Send(&antv1.ConversateChunk{Phase: "done", Code: code})
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

	_ = stream.Send(&antv1.ExecutePlanChunk{Phase: "generating"})

	registry := NewEmptyToolRegistry()
	registry.AddPreTool(&ReadKlineTool{repo: s.marketDataRepo})
	registry.AddPreTool(&ReadBacktestLogTool{repo: s.backtestRepo})
	registry.WireMemoryDB(s.memoryExec, s.memoryQuery)

	// Build Python agent system prompt with task instruction.
	sysPrompt := pythonAgentPrompt
	if m.Symbol != "" && m.Timeframe != "" {
		sysPrompt += fmt.Sprintf("\n\n## Current Workspace\nSymbol: %s\nTimeframe: %s", m.Symbol, m.Timeframe)
	}
	sysPrompt += "\n\n## 当前任务：生成或修改 Python 策略代码\n根据执行计划和用户的最新消息，输出完整的 Python 子集策略代码。生成后调用 compile_python 验证编译。"

	userPrompt := buildExecuteUserPrompt(m)

	loop := NewAgentLoop(registry,
		func(llmCtx context.Context, messages []systemai.ChatMessage, tools []systemai.ToolDefinition, onChunk func(systemai.ChatStreamChunk) error) error {
			return s.systemSvc.ChatCompletionStreamWithTools(llmCtx, userID, messages, tools, onChunk)
		},
		func(delta string) error {
			return stream.Send(&antv1.ExecutePlanChunk{Phase: "generating", Delta: delta})
		},
		func(tc *antv1.ToolCall, tr *antv1.ToolResult) error {
			return stream.Send(&antv1.ExecutePlanChunk{Phase: "tool_result", ToolCall: tc, ToolResult: tr})
		},
	)
	loop.SetCurrentCode(m.PreviousCode)

	raw, err := loop.Run(ctx, sysPrompt, userPrompt, userID)
	if err != nil {
		return stream.Send(&antv1.ExecutePlanChunk{Phase: "error", Error: systemai.FriendlyError(err)})
	}

	code := ExtractCode(raw)
	if code != "" && len(code) < len(raw)/3 {
		code = ""
	}

	s.persistExchange(ctx, userID, m.ConversationId, m.Plan, code, m.FeedbackMessage)
	return stream.Send(&antv1.ExecutePlanChunk{Phase: "done", Code: code, PreviousCode: m.PreviousCode})
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
			p += "## 当前的策略代码\n```go\n" + m.PreviousCode + "\n```\n\n"
		}
		if m.BacktestMetricsJson != "" {
			p += "## 回测数据\n" + m.BacktestMetricsJson + "\n"
		}
		return p
	}
	return m.Plan
}

