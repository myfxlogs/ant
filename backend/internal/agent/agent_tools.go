package agent

import (
	"context"

	"anttrader/internal/ai"
	connectai "anttrader/internal/connect/ai"
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
	t.result.PythonSource = code
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
func buildPythonToolRegistry(result *generateState) *connectai.ToolRegistry {
	reg := connectai.NewEmptyToolRegistry()
	reg.AddPreTool(&compilePythonTool{result: result})
	reg.AddPreTool(&runBacktestTool{result: result})
	reg.AddPreTool(&readCurrentCodeTool{result: result})
	reg.AddPreTool(&editCodeTool{result: result})
	return reg
}
