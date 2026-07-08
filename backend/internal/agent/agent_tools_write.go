package agent

import (
	"context"
	"fmt"
	"time"

	antv1 "anttrader/gen/proto/ant/v1"
	connectai "anttrader/internal/connect/ai"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
	sdk "anttrader/strategy/sdk"
	"anttrader/tools/mql2go"
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
			Description: "提交完整的 Python 策略代码。code 参数必填。内部自动编译→真实回测。这是提交最终代码的唯一方式——不要在聊天文本中粘贴代码。",
			Parameters: map[string]any{
				"type":     "object",
				"required": []string{"code"},
				"properties": map[string]any{
					"code": map[string]any{
						"type":        "string",
						"description": "完整的 Python 策略代码（class MyStrategy, on_bar 方法）",
					},
				},
			},
		},
	}
}

func (t *writeStrategyTool) Run(ctx context.Context, in connectai.ToolInput) connectai.ToolOutput {
	code, _ := in.RawArgs["code"].(string)
	if code == "" {
		code = in.Code
	}
	if code == "" {
		return connectai.ToolOutput{Success: false, Error: "code is required — pass the complete Python strategy as the 'code' parameter"}
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
			result["tier"] = tier
			result["total_trades"] = fmt.Sprintf("%d", btSummary.TotalTrades)
			result["win_rate"] = fmt.Sprintf("%.1f%%", btSummary.WinRate)
			result["total_return"] = fmt.Sprintf("%.2f%%", btSummary.TotalReturn)
			result["max_drawdown"] = fmt.Sprintf("%.2f%%", btSummary.MaxDrawdown)
			result["sharpe"] = fmt.Sprintf("%.2f", btSummary.SharpeRatio)
			// I4: transparent inputs alongside results.
			result["symbol"] = t.cfg.Symbol
			result["timeframe"] = t.cfg.Timeframe
			result["initial_capital"] = t.cfg.InitialCapital
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
	bars, err := fetchBarsForBacktest(ctx, t.mkt, t.cfg)
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
	if t.cfg.Symbol != "" && t.cfg.InitialCapital != "" {
		tier = "performance"
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

// fetchBarsForBacktest REUSEs the pattern from gateway.go:298-327.
func fetchBarsForBacktest(ctx context.Context, mkt repository.MarketDataStore, cfg *antv1.AgentBacktestConfig) ([]sdk.Bar, error) {
	if mkt == nil {
		return nil, fmt.Errorf("market data store not available")
	}
	var from, to *time.Time
	if cfg.StartDateMs > 0 {
		t := time.UnixMilli(cfg.StartDateMs)
		from = &t
	}
	if cfg.EndDateMs > 0 {
		t := time.UnixMilli(cfg.EndDateMs)
		to = &t
	}
	chBars, err := mkt.GetKlines(ctx, cfg.Symbol, "", cfg.Timeframe, from, to, 100000)
	if err != nil {
		return nil, err
	}
	bars := make([]sdk.Bar, 0, len(chBars))
	for i := len(chBars) - 1; i >= 0; i-- {
		b := chBars[i]
		bars = append(bars, sdk.Bar{
			Open:      b.Open,
			High:      b.High,
			Low:       b.Low,
			Close:     b.Close,
			Volume:    int64(b.Volume),
			Timestamp: int64(b.OpenTsUnixMs),
		})
	}
	return bars, nil
}

type backtestSummary struct {
	TotalTrades int
	WinRate     float64
	TotalReturn float64
	MaxDrawdown float64
	SharpeRatio float64
}

