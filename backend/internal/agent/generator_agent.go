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
	"anttrader/strategy/sdk"
	"anttrader/tools/mql2go"
)

// generatorAgentSystemPrompt is the system prompt for the Python strategy agent.
// It instructs the LLM to generate Python subset code and use tools to compile/backtest.
const generatorAgentSystemPrompt = `You are a quantitative strategy generator on the AntTrader platform.
Your task is to generate Python trading strategies from natural language descriptions.

` + pythonSubsetRules + `

## Workflow

You have access to two tools:
- **compile_python** — Compile your Python strategy code. Returns success + coverage score, or a specific error message.
- **run_backtest** — Run a backtest on your compiled strategy. Returns key metrics (total return, max drawdown, Sharpe ratio, win rate, trade count, etc.).

Follow this workflow:
1. **Discuss the plan first.** Analyze the user's strategy request, propose a concrete execution plan (numbered 1. 2. 3.), and confirm with the user.
2. **Think before acting.** Before ANY code generation, tool call, or analysis, wrap your reasoning in [THINK]...[/THINK]: (a) what is the current state? (b) what am I about to do? (c) why this approach?
3. **Generate the Python code.** Output complete, compilable Python subset code in a markdown code block. Do NOT use TODO or pass as placeholders.
4. **Self-verify.** Before calling compile_python, run through this checklist silently and fix any issues:
   □ Every __init__ param has type annotation AND default value?
   □ __init__ has -> None return type?
   □ Every method has a return type annotation?
   □ Every local variable has a type annotation?
   □ All prices/volumes/P&L use Decimal (not float)?
   □ Only import is "from decimal import Decimal"?
   □ No forbidden syntax (lambda, try/except, f-strings, list comprehensions)?
5. **Compile.** Call compile_python to check for errors.
6. **Fix if needed.** If compilation fails, read the error carefully, understand the root cause, fix the code, self-verify again, and compile again. Do NOT blindly guess — read the error message.
7. **Backtest.** Once compilation passes, call run_backtest to see the performance.
8. **Analyze.** Interpret the backtest results. Explain what the metrics mean and suggest improvements. If results are poor (Sharpe < 0.5, drawdown > 20%, < 5 trades), diagnose likely causes.

## Thinking Discipline (CRITICAL)

Before EVERY significant action (generating code, calling a tool, analyzing results), you MUST output a [THINK] block:

[THINK]
1. Current state: (what just happened? what do I know?)
2. Next action: (what am I about to do?)
3. Reason: (why this specific action?)
[/THINK]

Then immediately take the action. This prevents impulsive decisions and helps you catch mistakes before they happen.

## Error Memory

Common mistakes that cause compile failures — check these FIRST before generating code:
- FORGETTING -> None on __init__ method
- FORGETTING type annotation on local variables (e.g., ema_fast: float = ...)
- Using float for stop_loss/take_profit/price instead of Decimal
- Missing -> None on on_deinit method
- Importing anything other than Decimal
- Using f-strings or list comprehensions

If you just fixed a compile error, remember what caused it. Do NOT repeat the same mistake.

## Rules
- Output ONLY Python code in markdown fences — no explanations mixed with code.
- After calling a tool, STOP and wait for the result. Do not predict tool output.
- If the user provides a confirmed plan, follow it precisely.`

// runAgentLoop replaces the hardcoded generate→compile→backtest→retry pipeline
// with an LLM-driven agent loop. The LLM decides when to generate code, compile,
// backtest, fix errors, and iterate — using native tool_use via compile_python
// and run_backtest tools.
func (g *Generator) runAgentLoop(
	ctx context.Context,
	userID uuid.UUID,
	msg *antv1.AgentGenerateStrategyRequest,
	bars []sdk.Bar,
	btCfg *antv1.AgentBacktestConfig,
	preProfile *antv1.StrategyProfile,
	sessionMem *SessionMemory,
	confirmedPlan *antv1.StrategyPlan,
	streamOrAbort func(*antv1.AgentGenerateStrategyChunk) error,
) error {
	result := &generateState{}

	// ── Build Python tools ──
	gtCtx := &genToolContext{
		bars:   bars,
		btCfg:  btCfg,
		params: msg.Params,
	}
	registry := buildPythonToolRegistry(gtCtx, result)

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
	userPrompt.WriteString("\n\nDiscuss the plan briefly, then generate the Python strategy code, compile it, and run the backtest.")

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
		case "run_backtest":
			chunk := &antv1.AgentGenerateStrategyChunk{Phase: "backtesting"}
			if !tr.Success {
				chunk.BacktestError = tr.Error
			}
			return streamOrAbort(chunk)
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
	loop.SetCurrentCode("") // starts empty — LLM generates code, tools use it

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
		pythonSource = raw // fallback: use entire response as code
	}
	result.PythonSource = pythonSource

	// If the LLM never called compile_python (no errors set), compile manually.
	if result.CompileError == "" && result.BacktestError == "" {
		runner, coverage, compileErr := mql2go.CompilePythonWithCoverage(pythonSource)
		if compileErr != nil {
			result.CompileError = compileErr.Error()
		} else {
			// Run backtest if LLM didn't call run_backtest.
			btResult, btErr := runVMBacktest(ctx, runner, btCfg, bars, msg.Params)
			if btErr != nil {
				result.BacktestError = btErr.Error()
			} else {
				// Success — send final result.
				btProto := buildBacktestResultProto(btResult)
				chunk := &antv1.AgentGenerateStrategyChunk{
					Phase:         "done",
					PythonSource:  pythonSource,
					Result:        btProto,
					CoverageScore: coverage.Score,
				}
				for _, bs := range coverage.BlindSpots {
					chunk.BlindSpots = append(chunk.BlindSpots, &antv1.AgentBlindSpot{
						Builtin: bs.Builtin, Severity: bs.Severity, Count: int32(bs.Count),
					})
				}
				_ = streamOrAbort(chunk)
				return nil
			}
		}
	}

	// ── Send final chunk with whatever state we have ──
	finalChunk := &antv1.AgentGenerateStrategyChunk{
		Phase:         "done",
		PythonSource:  result.PythonSource,
		CompileError:  result.CompileError,
		BacktestError: result.BacktestError,
	}
	_ = streamOrAbort(finalChunk)
	return nil
}
