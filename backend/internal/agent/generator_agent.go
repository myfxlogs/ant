package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	connectai "anttrader/internal/connect/ai"
	systemai "anttrader/internal/service/systemai"
)

// generatorDiscipline defines the thinking and verification discipline for the
// Generator agent. Separate from the compilation contract (pythonSubsetRules)
// because it's a prompt-engineering concern — it can evolve independently.
const generatorDiscipline = `
## Thinking Discipline (CRITICAL)

Before EVERY significant action (generating code, calling a tool, analyzing results), you MUST output a [THINK] block:

[THINK]
1. Current state: (what just happened? what do I know?)
2. Next action: (what am I about to do?)
3. Reason: (why this specific action?)
[/THINK]

Then immediately take the action. This prevents impulsive decisions and helps you catch mistakes before they happen.

## Pre-Generation Syntax Check (MANDATORY)

Before outputting ANY Python code, mentally verify:
□ Does every function/method have a colon after its definition line?
□ Are all parentheses and brackets properly matched?
□ Is indentation consistent (4 spaces per level)?
□ Are all string literals properly closed?
□ No stray characters, incomplete lines, or copy-paste artifacts?

Code that fails tree-sitter parsing (HasError=true) will be REJECTED immediately. Check basic syntax BEFORE outputting.

## Pre-Compile Self-Verification (MANDATORY)

Before calling compile_python, silently run through this checklist. If ANY item fails, fix the code first — do NOT call compile_python with known issues:

□ Every __init__ param has type annotation AND default value?
□ __init__ has -> None return type?
□ Every method has a return type annotation?
□ Every local variable has a type annotation (e.g., ema_fast: float = ...)?
□ All prices/volumes/P&L/stop-loss/take-profit use Decimal (not float)?
□ Only import is "from decimal import Decimal"?
□ No forbidden syntax (lambda, try/except, f-strings, list comprehensions)?

## Error Memory — Common Mistakes

These are the most frequent compile errors. Check these FIRST before generating code:
- FORGETTING -> None on __init__ method
- FORGETTING type annotation on local variables
- Using float for stop_loss/take_profit/price instead of Decimal
- Missing -> None on on_deinit method
- Importing anything other than Decimal
- Using f-strings or list comprehensions
- Missing colon after def line, causing tree-sitter parse failure

If you just fixed a compile error, REMEMBER what caused it. Do NOT repeat the same mistake.`

// generatorAgentSystemPrompt is the system prompt for the Python strategy agent.
const generatorAgentSystemPrompt = `You are a quantitative strategy generator on the AntTrader platform.
Your task is to generate Python trading strategies from natural language descriptions.

` + pythonSubsetRules + `

## CRITICAL: Tool Usage Rules (VIOLATION = FAILURE)

**Never ask "should I use tool X?" Just call it. The workspace has symbol and timeframe.**

| User says | You do |
|-----------|--------|
| "什么行情?" "图表显示?" | → call read_kline IMMEDIATELY |
| "帮我编译" "验证代码" | → call compile_python IMMEDIATELY |
| "记住我偏好..." | → call remember IMMEDIATELY |
| "我之前用什么参数?" | → call recall IMMEDIATELY |
| "写一个策略" "生成代码" | → discuss plan FIRST, then generate |

**Only strategy generation needs discussion. Everything else: tool first, talk later.**

## Workflow

You have tools available:
- **read_kline** — Returns current price, EMA20/50, trend direction, volatility, recent OHLC bars.
- **compile_python** — Compile your Python code. Only call when the user explicitly asks you to verify.

Follow this workflow:
1. **Discuss the plan first.** Analyze the user's strategy request, propose a concrete execution plan (numbered 1. 2. 3.), and confirm with the user.
2. **[THINK]** Before generating code, think through the strategy logic silently.
3. **Generate the Python code.** Output complete, compilable Python subset code in a markdown code block. Do NOT use TODO or pass as placeholders.
4. **Present the code to the user.** After generating code, STOP. Show the code and tell the user: "Here's your strategy code. You can save it, or ask me to compile and verify it."
5. **Wait for user instruction.** Do NOT call compile_python automatically. Wait for the user to explicitly ask you to compile, modify, or explain the code.
6. **Compile only when asked.** If the user asks you to verify: call compile_python. If it fails, [THINK] read the error, understand the root cause, fix the specific issue, and compile again.

**IMPORTANT**: Never call compile_python without the user asking. Never run backtest — the user does that manually via the UI buttons. Your job is to generate clean, correct code and present it.

` + generatorDiscipline + `

## Rules
- Output ONLY Python code in markdown fences — no explanations mixed with code.
- After calling a tool, STOP and wait for the result. Do not predict tool output.
- If the user provides a confirmed plan, follow it precisely.`

// runAgentLoop runs the LLM-driven agent loop for the Generator.
// After the loop completes, code is extracted and sent to the frontend.
func (g *Generator) runAgentLoop(
	ctx context.Context,
	userID uuid.UUID,
	msg *antv1.AgentGenerateStrategyRequest,
	preProfile *antv1.StrategyProfile,
	sessionMem *SessionMemory,
	confirmedPlan *antv1.StrategyPlan,
	streamOrAbort func(*antv1.AgentGenerateStrategyChunk) error,
) error {
	result := &generateState{}

// ── Build tool registry — same tools as Conversate path
	registry := buildPythonToolRegistry(result)
	if g.mkt != nil {
		registry.AddPreTool(&connectai.ReadKlineTool{})
	}
	if g.btRepo != nil {
		registry.AddPreTool(&connectai.ReadBacktestLogTool{})
	}
	if g.dbExec != nil && g.dbQuery != nil {
		registry.WireMemoryDB(g.dbExec, g.dbQuery)
	}

	// ── Build system prompt ──
	sysPrompt := generatorAgentSystemPrompt
	if msg.Symbol != "" || msg.Timeframe != "" {
		sysPrompt += fmt.Sprintf("\n\n## Current Workspace\nSymbol: %s\nTimeframe: %s", msg.Symbol, msg.Timeframe)
	}

	// ── Build user prompt ──
	var userPrompt strings.Builder
	userPrompt.WriteString("## Strategy Request\n")
	userPrompt.WriteString(msg.Message)
	if preProfile != nil {
		userPrompt.WriteString("\n\n## Strategy Profile (AI-generated guidance)\n")
		userPrompt.WriteString(fmt.Sprintf("Type: %s\n", preProfile.StrategyType))
		userPrompt.WriteString(fmt.Sprintf("Description: %s\n", preProfile.Description))
		if len(preProfile.IndicatorsUsed) > 0 {
			userPrompt.WriteString(fmt.Sprintf("Indicators: %s\n", strings.Join(preProfile.IndicatorsUsed, ", ")))
	}
		userPrompt.WriteString(fmt.Sprintf("Entry: %s\n", preProfile.EntryLogic))
		userPrompt.WriteString(fmt.Sprintf("Exit: %s\n", preProfile.ExitLogic))
		userPrompt.WriteString(fmt.Sprintf("Risk: %s\n", preProfile.RiskManagement))
	}
	if confirmedPlan != nil {
		userPrompt.WriteString("\n\n## Confirmed Plan (follow precisely)\n")
		userPrompt.WriteString(fmt.Sprintf("Type: %s\n", confirmedPlan.Type))
		userPrompt.WriteString(fmt.Sprintf("Entry: %s\n", confirmedPlan.Entry))
		userPrompt.WriteString(fmt.Sprintf("Exit: %s\n", confirmedPlan.Exit))
		userPrompt.WriteString(fmt.Sprintf("Risk: %s\n", confirmedPlan.Risk))
		userPrompt.WriteString(fmt.Sprintf("Market: %s\n", confirmedPlan.Market))
	}
	if sessionMem != nil {
		sessionMem.InjectIntoPrompt(&userPrompt)
	}
	userPrompt.WriteString("\n\nDiscuss the plan briefly, then generate the Python strategy code and present it to the user. Do NOT compile — the user will ask if they want verification.")

	// ── Stream callbacks — map AgentLoop events to Generator chunk format ──
	streamChunk := func(delta string) error {
		return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "generating", Delta: delta})
	}
	toolStream := func(tc *antv1.ToolCall, tr *antv1.ToolResult) error {
		switch tc.Name {
		case "compile_python":
			chunk := &antv1.AgentGenerateStrategyChunk{Phase: "compiling", PythonSource: result.PythonSource}
			if !tr.Success {
				chunk.CompileError = tr.Error
			}
			return streamOrAbort(chunk)
		case "read_kline":
			return streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "planning"})
		}
		return nil
	}

	// ── Create and run the AgentLoop ──
	loop := connectai.NewAgentLoop(registry,
		func(llmCtx context.Context, messages []systemai.ChatMessage, tools []systemai.ToolDefinition, onChunk func(systemai.ChatStreamChunk) error) error {
			return g.aiSvc.ChatCompletionStreamWithTools(llmCtx, userID, messages, tools, onChunk)
	},
		streamChunk, toolStream,
	)
	loop.SetCurrentCode("")

	raw, loopErr := loop.Run(ctx, sysPrompt, userPrompt.String(), userID)
	if loopErr != nil {
		g.log.Warn("generator: agent loop failed", zap.Error(loopErr))
		_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{
			Phase: "done",
			Error: fmt.Sprintf("agent loop: %v", loopErr),
	})
		return nil
	}

	// ── Extract code from the final LLM response ──
	pythonSource := stripMarkdownFences(connectai.ExtractCode(raw))
	if pythonSource == "" {
		pythonSource = raw
	}
	result.PythonSource = pythonSource

	// ── Send final chunk with the generated code ──
	_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{
		Phase:        "done",
		PythonSource: pythonSource,
	})
	return nil
}
