package ai

import (
	"context"
	"fmt"

	"alphaforge/internal/repository"
	systemai "alphaforge/internal/service/systemai"
	"alphaforge/tools/mql2go"
)

// ── read_backtest_log tool ──

type ReadBacktestLogTool struct {
	repo *repository.BacktestRunRepository
}

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

// ── analyze_mql tool ──

type analyzeMQLChatTool struct{}

func (t *analyzeMQLChatTool) Name() string { return "analyze_mql" }
func (t *analyzeMQLChatTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: toolTypeFunction,
		Function: systemai.ToolDefFunction{
			Name:        "analyze_mql",
			Description: "分析用户导入的 MQL4/MQL5 代码能否在本平台 VM 上编译运行：返回编译状态、覆盖度评分、盲区列表与建议。用户粘贴/导入 MQL 代码时必须先调用本工具做覆盖度分析，基于结果给出方案（可编译部分保留原逻辑；盲区部分按「盲区桥接」翻译为 Python 子集并向用户说明），禁止不经分析就断言平台是否支持。",
			Parameters: map[string]any{
				schemaKeyType:       schemaTypeObject,
				schemaKeyProperties: map[string]any{},
			},
		},
	}
}

// Run treats a compile failure as a successful ANALYSIS: the compiler error
// names the blind spot the agent must bridge.
func (t *analyzeMQLChatTool) Run(_ context.Context, in ToolInput) ToolOutput {
	if in.Code == "" {
		return ToolOutput{Success: false, Error: "no MQL code to analyze"}
	}
	_, cov, err := mql2go.CompileMQLWithCoverage(in.Code)
	if err != nil {
		return ToolOutput{Success: true, Output: map[string]any{
			"compiles": false,
			"error":    err.Error(),
			"advice":   "编译不支持的盲区需按「盲区桥接」机制翻译为 Python 子集实现，并向用户说明哪些部分是翻译实现",
		}}
	}
	out := map[string]any{
		"compiles":       true,
		"coverage_score": cov.Score,
	}
	if len(cov.BlindSpots) > 0 {
		spots := make([]string, 0, len(cov.BlindSpots))
		for _, b := range cov.BlindSpots {
			spots = append(spots, fmt.Sprintf("%s(x%d, severity=%s)", b.Builtin, b.Count, b.Severity))
		}
		out["blind_spots"] = spots
		out["advice"] = "存在运行期盲区：直接编译可用，但盲区调用需按「盲区桥接」翻译为 Python 子集"
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
