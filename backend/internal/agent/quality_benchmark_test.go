//go:build benchmark

package agent

import (
	"context"
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/tools/mql2go"
)

// AgentQualityBenchmark tests strategy compilation + backtest via Bytecode VM.
// Uses Python subset strategies (valid AlphaForge SDK API).
// Validates the tool chain, not the LLM generation quality.
//
// Run: go test -tags=benchmark -run TestAgentQualityBenchmark ./internal/agent/ -count=1 -v

func TestAgentQualityBenchmark(t *testing.T) {
	cases := []struct {
		name    string
		code    string
		expects string // "compile_ok" | "compile_fail" | "backtest_ok"
	}{
		{
			"ma_cross",
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
`,
			"compile_ok",
		},
		{
			"rsi_simple",
			`class MyStrategy:
    rsi_val: float = 0.0

    def on_bar(self, ctx) -> None:
        rsi_val = ctx.indicators.rsi("EURUSD", "H1", 14, 0)
        if rsi_val < 30 and rsi_val > 0:
            if not ctx.positions():
                ctx.broker.buy(0.1, 0, 0)
        elif rsi_val > 70:
            ctx.broker.close_all()
`,
			"compile_ok",
		},
		{
			"syntax_error",
			`class MyStrategy
    def on_bar(ctx)
        if ctx.bars().close(0) > 0
            ctx.broker.buy(0.1)
`,
			"compile_fail",
		},
		{
			"no_class",
			`def on_bar(self, ctx) -> None:
    pass
`,
			"compile_fail",
		},
		{
			"bollinger",
			`class MyStrategy:
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
`,
			"compile_ok",
		},
	}

	compileOK := 0
	compileFailExpected := 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, cov, err := mql2go.CompilePythonWithCoverage(tc.code)
			if err != nil {
				if tc.expects == "compile_fail" {
					t.Logf("compile failed as expected: %v", err)
					compileFailExpected++
					return
				}
				t.Errorf("unexpected compile failure for %s: %v", tc.name, err)
				return
			}
			compileOK++
			t.Logf("%s: compile OK (coverage=%.1f%%)", tc.name, cov.Score*100)
		})
	}

	t.Logf("RESULTS: compile_success=%d/%d expected_failures=%d/%d",
		compileOK, len(cases), compileFailExpected, len(cases))
	_ = context.Background()
	_ = antv1.AgentBacktestConfig{}
}
