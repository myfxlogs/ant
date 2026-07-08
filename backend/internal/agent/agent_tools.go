package agent

import (
	"context"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	connectai "anttrader/internal/connect/ai"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
	"anttrader/tools/mql2go"
)

// ── compile_python tool ──

type compilePythonTool struct {
	result *generateState // mutated in-place so the caller can read final source/errors
}

func (t *compilePythonTool) Name() string { return "compile_python" }
func (t *compilePythonTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name:        "compile_python",
			Description: "编译当前的 Python 策略代码。代码必须是符合 Python 子集规范的完整策略（类名 MyStrategy，包含 on_bar 方法）。编译成功后返回覆盖度评分；编译失败返回具体错误信息。\n\n" + ai.PythonSubsetRules,
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}
func (t *compilePythonTool) Run(_ context.Context, in connectai.ToolInput) connectai.ToolOutput {
	code := in.Code
	// I1: only write_strategy sets PythonSource. compile_python just verifies.
	_, coverage, err := mql2go.CompilePythonWithCoverage(code)
	if err != nil {
		t.result.CompileError = err.Error()
		t.result.BacktestError = ""
		return connectai.ToolOutput{Success: false, Error: err.Error()}
	}
	t.result.CompileError = ""
	return connectai.ToolOutput{
		Success: true,
		Output: map[string]any{
			"compiles":       true,
			"coverage_score": coverage.Score,
		},
	}
}

// buildPythonToolRegistry creates a ToolRegistry with all agent tools.
// NEW: mkt+cfg injected for write_strategy real backtest (I2).
func buildPythonToolRegistry(result *generateState, mkt repository.MarketDataStore, cfg *antv1.AgentBacktestConfig) *connectai.ToolRegistry {
	reg := connectai.NewEmptyToolRegistry()
	// PRIMARY: write_strategy — the only way to submit final code (I1).
	// Compile + real backtest happen inside this tool (I2).
	reg.AddPreTool(&writeStrategyTool{result: result, mkt: mkt, cfg: cfg})
	// Support tools.
	reg.AddPreTool(&readCurrentCodeTool{result: result})
	reg.AddPreTool(&editCodeTool{result: result})
	reg.AddPreTool(&updatePlanTool{})
	// compile_python removed: write_strategy already does compile + backtest.
	// Its presence confuses LLM into picking it over write_strategy (native function calling).
	return reg
}
