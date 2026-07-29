package ai

import (
	"context"

	"alphaforge/internal/repository"
	systemai "alphaforge/internal/service/systemai"
	"alphaforge/tools/mql2go"
)

// ── read_backtest_log tool ──

type ReadBacktestLogTool struct{ repo *repository.BacktestRunRepository }

func NewReadBacktestLogTool(repo *repository.BacktestRunRepository) *ReadBacktestLogTool {
	return &ReadBacktestLogTool{repo: repo}
}

func (t *ReadBacktestLogTool) Name() string { return "read_backtest_log" }
func (t *ReadBacktestLogTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: toolTypeFunction,
		Function: systemai.ToolDefFunction{
			Name:        "read_backtest_log",
			Description: "读取最近一次回测的状态和错误信息。用于回测失败后查看具体原因。无需参数。",
			Parameters: map[string]any{
				schemaKeyType:       schemaTypeObject,
				schemaKeyProperties: map[string]any{},
			},
		},
	}
}
func (t *ReadBacktestLogTool) Run(ctx context.Context, in ToolInput) ToolOutput {
	runs, err := t.repo.ListByUser(ctx, in.UserID, nil, nil, 1, 0)
	if err != nil || len(runs) == 0 {
		return ToolOutput{Success: false, Error: "no recent backtest runs found"}
	}
	run := runs[0]
	out := map[string]any{
		"run_id": run.ID.String(), "symbol": run.Symbol, "timeframe": run.Timeframe,
		"status": run.Status,
	}
	if run.Error != "" {
		out["error"] = run.Error
	}
	return ToolOutput{Success: true, Output: out}
}

// ── compile_python tool ──

type compilePythonChatTool struct{}

func (t *compilePythonChatTool) Name() string { return "compile_python" }
func (t *compilePythonChatTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: toolTypeFunction,
		Function: systemai.ToolDefFunction{
			Name:        "compile_python",
			Description: "编译当前的 Python 策略代码。代码必须是符合 Python 子集规范的完整策略。编译成功返回覆盖度评分；编译失败返回具体错误信息。",
			Parameters: map[string]any{
				schemaKeyType:       schemaTypeObject,
				schemaKeyProperties: map[string]any{},
			},
		},
	}
}
func (t *compilePythonChatTool) Run(_ context.Context, in ToolInput) ToolOutput {
	if in.Code == "" {
		return ToolOutput{Success: false, Error: "no Python code to compile"}
	}
	_, coverage, err := mql2go.CompilePythonWithCoverage(in.Code)
	if err != nil {
		return ToolOutput{Success: false, Error: err.Error()}
	}
	return ToolOutput{
		Success: true,
		Output: map[string]any{
			"compiles":       true,
			"coverage_score": coverage.Score,
		},
	}
}
