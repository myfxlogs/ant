package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	connectai "alphaforge/internal/connect/ai"
	"alphaforge/internal/repository"
	systemai "alphaforge/internal/service/systemai"
	"alphaforge/tools/mql2go"
)

// writeStrategyTool is the single entry point for strategy code submission (I1).
// Code is a REQUIRED parameter — the LLM cannot deliver code via free text (§3.1).
// Internally: compile → real backtest → write to generateState.PythonSource (§3.2).
// Returns structured result with inputs for transparency (I4).
//
// REUSE: mql2go.CompilePythonWithCoverage @ tools/mql2go
// REUSE: runVMBacktest @ backtest_helpers.go:18
// REUSE: buildBacktestResultProto @ backtest_helpers.go:60
// REUSE: full flow reference @ gateway.go:114-156
type writeStrategyTool struct {
	result *generateState
	mkt    repository.MarketDataStore // from Generator; nil if unavailable
	cfg    *antv1.AgentBacktestConfig // workspace config; nil if unavailable
}

func (t *writeStrategyTool) Name() string { return "write_strategy" }

func (t *writeStrategyTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name: "write_strategy",
			Description: "提交完整的交易策略代码。这是提交策略的唯一方式——不要在聊天文本中粘贴代码。\n\n此工具自动执行：\n1. 编译验证（语法检查）\n2. 真实市场数据回测\n3. 返回编译状态 + 回测指标（胜率、收益率、最大回撤、夏普比率）\n\n提交前请确认：代码完整可运行、无语法错误、无未来函数、仓位管理合理。",
			Parameters: map[string]any{
				schemaKeyType:     schemaTypeObject,
				"required": []string{"code"},
				schemaKeyProperties: map[string]any{
					"code": map[string]any{
						schemaKeyType:        schemaTypeString,
						"description": "完整的 Python 策略代码（class MyStrategy, on_bar 方法）",
					},
				},
			},
		},
	}
}

func (t *writeStrategyTool) Run(ctx context.Context, in connectai.ToolInput) connectai.ToolOutput {
	code, ok := in.RawArgs["code"].(string)
	if !ok || code == "" {
		return connectai.ToolOutput{Success: false, Error: "code is required — pass the complete strategy code as a string 'code' parameter"}
	}

	// I1: write_strategy is the ONLY source of truth for the deliverable code.
	t.result.PythonSource = code
	t.result.CompileError = ""
	t.result.BacktestError = ""

	// Step 1: Compile (REUSE: gateway.go:126).
	runner, cov, compileErr := mql2go.CompilePythonWithCoverage(code)
	if compileErr != nil {
		t.result.CompileError = compileErr.Error()
		return connectai.ToolOutput{
			Success: false,
			Error:   fmt.Sprintf("compile failed: %v", compileErr),
			Output: map[string]string{
				"compiled": "false",
				"error":    compileErr.Error(),
			},
		}
	}

	result := map[string]string{
		"compiled":  "true",
		"coverage":  fmt.Sprintf("%.1f%%", cov.Score*100),
	}

	// Step 2: Real backtest (REUSE: gateway.go:141-163).
	// I2a: smoke test (compile + execute on bars). I2b: performance (with full params).
	// I4: inputs must be included alongside results (§3.2c).
	if t.mkt != nil && t.cfg != nil && t.cfg.Symbol != "" {
		btSummary, tier, btErr := t.runBacktest(ctx, runner)
		if btErr != nil {
			result["backtest_error"] = btErr.Error()
			result["tier"] = "compile_only"
		} else {
			t.result.LastBacktest = btSummary // captured for persistent cross-conversation memory
			result["tier"] = tier
			result["total_trades"] = fmt.Sprintf("%d", btSummary.TotalTrades)
			result["win_rate"] = fmt.Sprintf("%s%%", btSummary.WinRate)
			result["total_return"] = fmt.Sprintf("%s%%", btSummary.TotalReturn)
			result["max_drawdown"] = fmt.Sprintf("%s%%", btSummary.MaxDrawdown)
			result["sharpe"] = btSummary.SharpeRatio
			// I4: transparent inputs alongside results (§3.2c).
			result["symbol"] = t.cfg.Symbol
			result["timeframe"] = t.cfg.Timeframe
			result["initial_capital"] = t.cfg.InitialCapital
			result["commission"] = t.cfg.Commission
			if t.cfg.StartDateMs > 0 {
				result["date_range"] = fmt.Sprintf("%s → %s",
					time.UnixMilli(t.cfg.StartDateMs).Format("2006-01-02"),
					time.UnixMilli(t.cfg.EndDateMs).Format("2006-01-02"))
			} else {
				result["date_range"] = "recent (auto)"
			}
		}
	} else {
		result["tier"] = "compile_only"
		result["backtest_note"] = "smoke test skipped — no symbol configured in workspace"
	}

	return connectai.ToolOutput{Success: true, Output: result}
}

// runBacktest REUSEs: gateway.go:141-163 flow (fetchBars → runVMBacktest → buildBacktestResultProto).
// Returns smoke tier (I2a) when config is partial, performance tier (I2b) when full.
func (t *writeStrategyTool) runBacktest(ctx context.Context, runner *mql2go.VMRunner) (*backtestSummary, string, error) {
	// Step 2a: Fetch bars (REUSE: gateway.go:141-153).
	bars, err := FetchBarsForBacktest(ctx, t.mkt, t.cfg)
	if err != nil {
		return nil, "", fmt.Errorf("fetch bars: %w", err)
	}
	if len(bars) < 2 {
		return nil, "", fmt.Errorf("insufficient market data: %d bars (need ≥2)", len(bars))
	}

	// Step 2b: Run VM backtest (runner already compiled above) (REUSE: runVMBacktest @ backtest_helpers.go:18).
	btResult, btErr := runVMBacktest(ctx, runner, t.cfg, bars, nil)

	// I2a vs I2b: determine tier.
	tier := "smoke"
	if t.cfg.Symbol != "" {
		if initialCap, err := decimal.NewFromString(t.cfg.InitialCapital); err == nil && initialCap.IsPositive() {
			tier = "performance"
		}
	}

	if btErr != nil {
		return nil, tier, fmt.Errorf("backtest: %w", btErr)
	}

	// Step 2d: Convert result (REUSE: buildBacktestResultProto @ backtest_helpers.go:60).
	btProto := buildBacktestResultProto(btResult)
	summary := &backtestSummary{
		TotalTrades: int(btProto.TotalTrades),
		WinRate:     btProto.WinRate,
		TotalReturn: btProto.TotalReturn,
		MaxDrawdown: btProto.MaxDrawdown,
		SharpeRatio: btProto.SharpeRatio,
	}
	return summary, tier, nil
}

type backtestSummary struct {
	TotalTrades int
	WinRate     string
	TotalReturn string
	MaxDrawdown string
	SharpeRatio string
}

