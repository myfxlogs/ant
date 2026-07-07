package agent

import (
	"context"
	"fmt"

	connectai "anttrader/internal/connect/ai"
	systemai "anttrader/internal/service/systemai"
	"anttrader/tools/mql2go"
)

// runBacktestTool compiles and validates the current Python strategy code.
// Phase 2: will run a real backtest when btRepo + market data are wired.
type runBacktestTool struct {
	result *generateState
}

func (t *runBacktestTool) Name() string { return "run_backtest" }

func (t *runBacktestTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name: "run_backtest",
			Description: "对当前Python策略代码编译并验证。返回编译结果和覆盖度。编译通过=策略结构正确。后续将支持真实回测。",
			Parameters: map[string]any{"type": "object", "properties": map[string]any{}},
		},
	}
}

func (t *runBacktestTool) Run(ctx context.Context, in connectai.ToolInput) connectai.ToolOutput {
	code := in.Code
	if code == "" {
		code = t.result.PythonSource
	}
	if code == "" {
		return connectai.ToolOutput{Success: false, Error: "no code to backtest"}
	}
	_, cov, err := mql2go.CompilePythonWithCoverage(code)
	if err != nil {
		return connectai.ToolOutput{Success: false, Error: fmt.Sprintf("compile failed: %v", err)}
	}
	return connectai.ToolOutput{
		Success: true,
		Output: map[string]string{
			"status":   "compiled",
			"coverage": fmt.Sprintf("%.1f%%", cov.Score*100),
		},
	}
}
