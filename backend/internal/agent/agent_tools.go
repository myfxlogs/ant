package agent

import (
	"context"

	antv1 "anttrader/gen/proto/ant/v1"
	connectai "anttrader/internal/connect/ai"
	systemai "anttrader/internal/service/systemai"
	"anttrader/strategy/sdk"
	"anttrader/tools/mql2go"
)

// genToolContext holds shared state that Python agent tools need access to.
type genToolContext struct {
	bars   []sdk.Bar
	btCfg  *antv1.AgentBacktestConfig
	params map[string]string
}

// ── compile_python tool ──

type compilePythonTool struct {
	ctx    *genToolContext
	result *generateState // mutated in-place so the caller can read final source/errors
}

func (t *compilePythonTool) Name() string { return "compile_python" }
func (t *compilePythonTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name:        "compile_python",
			Description: "编译当前的 Python 策略代码。代码必须是符合 Python 子集规范的完整策略（类名 MyStrategy，包含 on_bar 方法）。编译成功后返回覆盖度评分；编译失败返回具体错误信息。",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}
func (t *compilePythonTool) Run(_ context.Context, in connectai.ToolInput) connectai.ToolOutput {
	code := in.Code
	if code == "" {
		return connectai.ToolOutput{Success: false, Error: "no Python code to compile — generate code first"}
	}
	t.result.PythonSource = code
	_, coverage, err := mql2go.CompilePythonWithCoverage(code)
	if err != nil {
		t.result.CompileError = err.Error()
		t.result.BacktestError = ""
		return connectai.ToolOutput{
			Success: false,
			Error:   err.Error(),
		}
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

// ── run_backtest tool ──

type runBacktestTool struct {
	ctx    *genToolContext
	result *generateState
}

func (t *runBacktestTool) Name() string { return "run_backtest" }
func (t *runBacktestTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name:        "run_backtest",
			Description: "对当前已编译通过的 Python 策略代码执行回测。回测在历史 K 线数据上运行，使用配置的初始资金、杠杆、手续费和滑点。返回关键指标：总收益率、最大回撤、夏普比率、胜率、交易次数等。如果代码尚未编译或编译失败，请先使用 compile_python 修复编译错误。",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}
func (t *runBacktestTool) Run(ctx context.Context, in connectai.ToolInput) connectai.ToolOutput {
	code := in.Code
	if code == "" {
		return connectai.ToolOutput{Success: false, Error: "no Python code to backtest — generate and compile code first"}
	}
	t.result.PythonSource = code
	runner, coverage, compileErr := mql2go.CompilePythonWithCoverage(code)
	if compileErr != nil {
		t.result.CompileError = compileErr.Error()
		return connectai.ToolOutput{
			Success: false,
			Error:   "compile failed before backtest: " + compileErr.Error(),
		}
	}
	t.result.CompileError = ""

	btResult, btErr := runVMBacktest(ctx, runner, t.ctx.btCfg, t.ctx.bars, t.ctx.params)
	if btErr != nil {
		t.result.BacktestError = btErr.Error()
		return connectai.ToolOutput{
			Success: false,
			Error:   btErr.Error(),
		}
	}
	t.result.BacktestError = ""

	out := map[string]any{
		"success":       true,
		"coverage_score": coverage.Score,
	}
	if btResult.Metrics != nil {
		out["total_return"] = btResult.Metrics.TotalReturn
		out["annual_return"] = btResult.Metrics.AnnualReturn
		out["max_drawdown"] = btResult.Metrics.MaxDrawdown
		out["sharpe_ratio"] = btResult.Metrics.SharpeRatio
		out["win_rate"] = btResult.Metrics.WinRate
		out["profit_factor"] = btResult.Metrics.ProfitFactor
		out["total_trades"] = btResult.Metrics.TotalTrades
		out["winning_trades"] = btResult.Metrics.WinningTrades
		out["losing_trades"] = btResult.Metrics.LosingTrades
	}
	// Include first 10 trades for the LLM to analyze.
	if len(btResult.Trades) > 0 {
		end := len(btResult.Trades)
		if end > 10 {
			end = 10
		}
		trades := make([]map[string]string, 0, end)
		for _, tr := range btResult.Trades[:end] {
			trades = append(trades, map[string]string{
				"pnl":    tr.Profit.String(),
				"volume": tr.Volume.String(),
				"reason": tr.Comment,
			})
		}
		out["trades"] = trades
	}
	return connectai.ToolOutput{Success: true, Output: out}
}

// buildPythonToolRegistry creates a ToolRegistry with Python-specific tools for strategy generation.
func buildPythonToolRegistry(gtCtx *genToolContext, result *generateState) *connectai.ToolRegistry {
	reg := connectai.NewEmptyToolRegistry()
	reg.AddPreTool(&compilePythonTool{ctx: gtCtx, result: result})
	reg.AddPreTool(&runBacktestTool{ctx: gtCtx, result: result})
	return reg
}
