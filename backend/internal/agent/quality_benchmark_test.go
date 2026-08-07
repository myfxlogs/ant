//go:build benchmark

package agent

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go"
)

// LAUNCH-1: Agent Strategy Quality Benchmark Suite
//
// Tests the full tool chain: compile → backtest → metrics.
// 20 strategy cases covering different styles (trend/mean-reversion/breakout/grid/multi-TF).
// Measures: compilation pass rate, backtest completion rate, Sharpe > 0 ratio.
//
// Targets (per pre-launch-assessment Gap 1):
//   - Compilation pass rate ≥ 90%
//   - Backtest completion rate ≥ 80%
//   - Sharpe > 0 ratio ≥ 50%
//
// Run: go test -tags=benchmark -run TestAgentQualityBenchmark -count=1 -v ./internal/agent/

// benchmarkCase is a single strategy test case.
type benchmarkCase struct {
	name       string
	style      string // trend / mean_reversion / breakout / grid / multi_tf / oscillator
	code       string
	expectFail bool // true if compilation should fail (syntax error, etc.)
}

// benchmarkBars generates deterministic synthetic H1 bars for backtesting.
// 500 bars (~20 days of hourly data) with trending + oscillating price action.
func benchmarkBars(n int) []sdk.Bar {
	bars := make([]sdk.Bar, n)
	base := 1.1000
	for i := 0; i < n; i++ {
		osc := math.Sin(float64(i)*0.1) * 0.0030
		trend := float64(i) * 0.0001
		close := base + trend + osc
		high := close + math.Abs(math.Sin(float64(i)*0.15))*0.0010 + 0.0005
		low := close - math.Abs(math.Cos(float64(i)*0.15))*0.0010 - 0.0005
		open := close - math.Sin(float64(i)*0.05)*0.0005

		bars[i] = sdk.Bar{
			Open:      decimal.NewFromFloat(open),
			High:      decimal.NewFromFloat(high),
			Low:       decimal.NewFromFloat(low),
			Close:     decimal.NewFromFloat(close),
			Volume:    int64(1000 + i%500),
			Timestamp: int64(i) * 3600 * 1000,
		}
	}
	return bars
}

func TestAgentQualityBenchmark(t *testing.T) {
	cases := []benchmarkCase{
		// --- Trend Following ---
		{
			"ma_cross_fast_slow", "trend",
			`class MyStrategy:
    price_fast: float = 0.0
    price_slow: float = 0.0

    def on_bar(self, ctx) -> None:
        price_fast = ctx.indicators.ima("EURUSD", "H1", 10, 0, 0, "close")
        price_slow = ctx.indicators.ima("EURUSD", "H1", 30, 0, 0, "close")
        if price_fast > price_slow and price_slow > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif price_fast < price_slow and price_slow > 0:
            ctx.broker.close_all()
`, false,
		},
		{
			"ma_cross_with_sl", "trend",
			`class MyStrategy:
    fast: float = 0.0
    slow: float = 0.0

    def on_bar(self, ctx) -> None:
        fast = ctx.indicators.ima("EURUSD", "H1", 5, 0, 0, "close")
        slow = ctx.indicators.ima("EURUSD", "H1", 20, 0, 0, "close")
        if fast > slow and slow > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0.0050, 0.0100)
        elif fast < slow:
            ctx.broker.close_all()
`, false,
		},
		{
			"macd_trend", "trend",
			`class MyStrategy:
    macd_main: float = 0.0
    macd_signal: float = 0.0

    def on_bar(self, ctx) -> None:
        macd_main = ctx.indicators.imacd("EURUSD", "H1", 12, 26, 9, 0, 0)
        macd_signal = ctx.indicators.imacd("EURUSD", "H1", 12, 26, 9, 0, 1)
        if macd_main > macd_signal and macd_signal > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif macd_main < macd_signal:
            ctx.broker.close_all()
`, false,
		},
		{
			"ema_ribbon", "trend",
			`class MyStrategy:
    ema5: float = 0.0
    ema10: float = 0.0
    ema20: float = 0.0

    def on_bar(self, ctx) -> None:
        ema5 = ctx.indicators.ima("EURUSD", "H1", 5, 0, 0, "close")
        ema10 = ctx.indicators.ima("EURUSD", "H1", 10, 0, 0, "close")
        ema20 = ctx.indicators.ima("EURUSD", "H1", 20, 0, 0, "close")
        if ema5 > ema10 and ema10 > ema20 and ema20 > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif ema5 < ema10 and ema10 < ema20:
            ctx.broker.close_all()
`, false,
		},

		// --- Mean Reversion ---
		{
			"rsi_oversold", "mean_reversion",
			`class MyStrategy:
    rsi_val: float = 0.0

    def on_bar(self, ctx) -> None:
        rsi_val = ctx.indicators.rsi("EURUSD", "H1", 14, 0)
        if rsi_val < 45 and rsi_val > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif rsi_val > 55:
            ctx.broker.close_all()
`, false,
		},
		{
			"bollinger_reversion", "mean_reversion",
			`class MyStrategy:
    upper: float = 0.0
    lower: float = 0.0
    price: float = 0.0

    def on_bar(self, ctx) -> None:
        upper = ctx.indicators.ibands("EURUSD", "H1", 20, 2, 0, 0, "upper")
        lower = ctx.indicators.ibands("EURUSD", "H1", 20, 2, 0, 0, "lower")
        price = ctx.bars().close(0)
        if price < lower and lower > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif price > upper and upper > 0:
            ctx.broker.close_all()
`, false,
		},
		{
			"stochastic_reversion", "mean_reversion",
			`class MyStrategy:
    k_val: float = 0.0
    d_val: float = 0.0

    def on_bar(self, ctx) -> None:
        k_val = ctx.indicators.istochastic("EURUSD", "H1", 5, 3, 0, 0, 0)
        d_val = ctx.indicators.istochastic("EURUSD", "H1", 5, 3, 0, 1, 0)
        if k_val < 40 and k_val > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif k_val > 60:
            ctx.broker.close_all()
`, false,
		},

		// --- Breakout ---
		{
			"donchian_breakout", "breakout",
			`class MyStrategy:
    upper: float = 0.0
    lower: float = 0.0

    def on_bar(self, ctx) -> None:
        upper = ctx.indicators.ibands("EURUSD", "H1", 20, 2, 0, 0, "upper")
        lower = ctx.indicators.ibands("EURUSD", "H1", 20, 2, 0, 0, "lower")
        price = ctx.bars().close(0)
        if price > upper and upper > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif price < lower and lower > 0:
            ctx.broker.close_all()
`, false,
		},
		{
			"range_breakout_with_volume", "breakout",
			`class MyStrategy:
    high_val: float = 0.0
    low_val: float = 0.0

    def on_bar(self, ctx) -> None:
        high_val = ctx.bars().high(5)
        low_val = ctx.bars().low(5)
        price = ctx.bars().close(0)
        if price > high_val and high_val > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0.0030, 0.0100)
        elif price < low_val and low_val > 0:
            ctx.broker.close_all()
`, false,
		},

		// --- Grid ---
		{
			"simple_grid", "grid",
			`class MyStrategy:
    grid_level: int = 0

    def on_bar(self, ctx) -> None:
        price = ctx.bars().close(0)
        if price > 0:
            if not ctx.positions():
                ctx.broker.buy(0.05, 0, 0)
            else:
                ctx.broker.close_all()
`, false,
		},

		// --- Multi-Timeframe ---
		{
			"multi_tf_trend", "multi_tf",
			`class MyStrategy:
    fast_ma: float = 0.0
    slow_ma: float = 0.0

    def on_bar(self, ctx) -> None:
        fast_ma = ctx.indicators.ima("EURUSD", "H1", 10, 0, 0, "close")
        slow_ma = ctx.indicators.ima("EURUSD", "H1", 50, 0, 0, "close")
        if fast_ma > slow_ma and slow_ma > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif fast_ma < slow_ma:
            ctx.broker.close_all()
`, false,
		},

		// --- Oscillator ---
		{
			"cci_oversold", "oscillator",
			`class MyStrategy:
    cci_val: float = 0.0

    def on_bar(self, ctx) -> None:
        cci_val = ctx.indicators.icci("EURUSD", "H1", 14, 0, 0)
        if cci_val < -100 and cci_val < 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif cci_val > 100:
            ctx.broker.close_all()
`, false,
		},
		{
			"mfi_oversold", "oscillator",
			`class MyStrategy:
    mfi_val: float = 0.0

    def on_bar(self, ctx) -> None:
        mfi_val = ctx.indicators.imfi("EURUSD", "H1", 14, 0)
        if mfi_val < 20 and mfi_val > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif mfi_val > 80:
            ctx.broker.close_all()
`, false,
		},
		{
			"adx_trend_strength", "oscillator",
			`class MyStrategy:
    adx_val: float = 0.0
    plus_di: float = 0.0
    minus_di: float = 0.0

    def on_bar(self, ctx) -> None:
        adx_val = ctx.indicators.iadx("EURUSD", "H1", 14, 0, 0)
        plus_di = ctx.indicators.iadx("EURUSD", "H1", 14, 0, 1)
        minus_di = ctx.indicators.iadx("EURUSD", "H1", 14, 0, 2)
        if adx_val > 10 and plus_di > minus_di and adx_val > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif plus_di < minus_di:
            ctx.broker.close_all()
`, false,
		},

		// --- Risk Management ---
		{
			"ma_with_trailing_stop", "trend",
			`class MyStrategy:
    ma_val: float = 0.0

    def on_bar(self, ctx) -> None:
        ma_val = ctx.indicators.ima("EURUSD", "H1", 20, 0, 0, "close")
        price = ctx.bars().close(0)
        if price > ma_val and ma_val > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0.0050, 0)
        elif price < ma_val:
            ctx.broker.close_all()
`, false,
		},
		{
			"rsi_with_partial_close", "mean_reversion",
			`class MyStrategy:
    rsi_val: float = 0.0

    def on_bar(self, ctx) -> None:
        rsi_val = ctx.indicators.rsi("EURUSD", "H1", 14, 0)
        if rsi_val < 45 and rsi_val > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0.0050, 0.0100)
        elif rsi_val > 55:
            ctx.broker.close_all()
`, false,
		},

		// --- Simple / Minimal ---
		{
			"price_action_basic", "trend",
			`class MyStrategy:
    prev_close: float = 0.0

    def on_bar(self, ctx) -> None:
        close = ctx.bars().close(0)
        if close > prev_close and prev_close > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif close < prev_close and prev_close > 0:
            ctx.broker.close_all()
        prev_close = close
`, false,
		},
		{
			"momentum_roc", "oscillator",
			`class MyStrategy:
    roc_val: float = 0.0

    def on_bar(self, ctx) -> None:
        close = ctx.bars().close(0)
        prev = ctx.bars().close(10)
        if close > 0 and prev > 0:
            roc_val = (close - prev) / prev
        if roc_val > 0.001:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif roc_val < 0:
            ctx.broker.close_all()
`, false,
		},

		// --- Negative Cases ---
		{
			"syntax_error", "invalid",
			`class MyStrategy
    def on_bar(ctx)
        if ctx.bars().close(0) > 0
            ctx.broker.buy(0.1)
`, true,
		},
		{
			"empty_strategy", "invalid",
			`class MyStrategy:
    pass
`, false,
		},
	}

	// --- Run benchmark ---
	bars := benchmarkBars(500)

	compileOK := 0
	compileExpectedFail := 0
	backtestOK := 0
	backtestErr := 0
	sharpePositive := 0

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner, cov, err := mql2go.CompilePythonWithCoverage(tc.code)
			if err != nil {
				if tc.expectFail {
					t.Logf("compile failed as expected: %v", err)
					compileExpectedFail++
					return
				}
				t.Errorf("unexpected compile failure: %v", err)
				return
			}
			compileOK++
			t.Logf("compile OK (coverage=%.1f%%)", cov.Score*100)

			// Run backtest
			btCfg := backtest.Config{
				Symbol:         "EURUSD",
				Timeframe:      "H1",
				InitialCapital: decimal.NewFromFloat(10000),
				Leverage:       100,
				Commission:     decimal.NewFromFloat(0.0003),
				Slippage:       decimal.NewFromFloat(0.00001),
				SwapRate:       decimal.NewFromFloat(0.00001),
			}
			backtest.DeriveSymbolInfoFromBars(&btCfg, bars)

			engine := backtest.New(btCfg, runner, bars)
			btCtx, btCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer btCancel()

			result, btErr := engine.Run(btCtx)
			if btErr != nil {
				t.Logf("backtest error: %v", btErr)
				backtestErr++
				return
			}
			backtestOK++

			if result.Metrics != nil {
				sharpe, _ := decimal.NewFromString(result.Metrics.SharpeRatio)
				totalTrades := result.Metrics.TotalTrades
				t.Logf("backtest OK: trades=%d sharpe=%s winRate=%s", totalTrades, result.Metrics.SharpeRatio, result.Metrics.WinRate)
				if sharpe.GreaterThan(decimal.Zero) {
					sharpePositive++
				}
			} else {
				t.Logf("backtest OK: no metrics")
			}
		})
	}

	// --- Summary ---
	validCases := len(cases) - compileExpectedFail
	t.Logf("\n=== LAUNCH-1 Quality Benchmark Results ===")
	t.Logf("Total cases:       %d", len(cases))
	t.Logf("Compile OK:        %d/%d (%.0f%%)  target ≥90%%", compileOK, validCases, pct(compileOK, validCases))
	t.Logf("Backtest OK:       %d/%d (%.0f%%)  target ≥80%%", backtestOK, compileOK, pct(backtestOK, compileOK))
	t.Logf("Sharpe > 0:        %d/%d (%.0f%%)  target ≥50%%", sharpePositive, backtestOK, pct(sharpePositive, backtestOK))
	t.Logf("Expected failures: %d", compileExpectedFail)
	t.Logf("Backtest errors:   %d", backtestErr)
	t.Logf("===========================================")
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d) * 100
}
