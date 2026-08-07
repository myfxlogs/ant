// TestAgentQualityBenchmark_CompileOnly is a fast CI subset (no benchmark tag needed).
// Tests compilation only — validates the Python→VM compiler handles common strategy patterns.
// Full benchmark (compile + backtest + metrics) is in quality_benchmark_test.go with -tags=benchmark.

package agent

import (
	"testing"

	"alphaforge/tools/mql2go"
)

func TestAgentQualityBenchmark_CompileOnly(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		code       string
		expectFail bool
	}{
		{"ma_cross", `class MyStrategy:
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
`, false},
		{"rsi_simple", `class MyStrategy:
    rsi_val: float = 0.0

    def on_bar(self, ctx) -> None:
        rsi_val = ctx.indicators.rsi("EURUSD", "H1", 14, 0)
        if rsi_val < 30 and rsi_val > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif rsi_val > 70:
            ctx.broker.close_all()
`, false},
		{"bollinger", `class MyStrategy:
    upper: float = 0.0
    lower: float = 0.0
    price: float = 0.0

    def on_bar(self, ctx) -> None:
        upper = ctx.indicators.ibands("EURUSD", "H1", 20, 2, 0, 0, "upper")
        lower = ctx.indicators.ibands("EURUSD", "H1", 20, 2, 0, 0, "lower")
        price = ctx.bars().close(0)
        if price > upper and upper > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, lower, upper)
        elif price < lower and lower > 0:
            ctx.broker.close_all()
`, false},
		{"macd_trend", `class MyStrategy:
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
`, false},
		{"syntax_error", `class MyStrategy
    def on_bar(ctx)
        if ctx.bars().close(0) > 0
            ctx.broker.buy(0.1)
`, true},
	}

	compileOK := 0
	compileExpectedFail := 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := mql2go.CompilePythonWithCoverage(tc.code)
			if tc.expectFail {
				if err == nil {
					t.Error("expected compile failure but got nil")
				}
				compileExpectedFail++
				return
			}
			if err != nil {
				t.Errorf("unexpected compile failure: %v", err)
				return
			}
			compileOK++
		})
	}

	validCases := len(cases) - compileExpectedFail
	pctVal := 0.0
	if validCases > 0 {
		pctVal = float64(compileOK) / float64(validCases) * 100
	}
	t.Logf("Compile OK: %d/%d (%.0f%%)  target ≥90%%", compileOK, validCases, pctVal)
}
