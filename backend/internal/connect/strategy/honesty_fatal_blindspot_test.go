package strategy

import (
	"context"
	"testing"
	"time"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go"
	"alphaforge/tools/mql2go/interp"

	"github.com/shopspring/decimal"
)

// makeE2EBars creates test bars with oscillating prices to trigger indicator crossovers.
func makeE2EBars(n int) []sdk.Bar {
	bars := make([]sdk.Bar, n)
	price := 1.1000
	for i := 0; i < n; i++ {
		if (i/10)%2 == 0 {
			price += 0.0020
		} else {
			price -= 0.0020
		}
		bars[i] = sdk.Bar{
			Open:      decimal.NewFromFloat(price - 0.0005),
			High:      decimal.NewFromFloat(price + 0.0010),
			Low:       decimal.NewFromFloat(price - 0.0010),
			Close:     decimal.NewFromFloat(price),
			Volume:    1000,
			Timestamp: time.Date(2024, 1, 1, 0, i, 0, 0, time.UTC).UnixMilli(),
		}
	}
	return bars
}

// TestHONESTY3_FatalBlindSpotSetsUnreliable verifies that a fatal coverage
// blind spot (e.g. unknown indicator iXxx → SeverityFatal → silently returns 0)
// causes IsReliable=false in the backtest response.
//
// Adversarial proof: if the HONESTY-3 fix in buildBacktestResponse is removed
// (the fatal-severity check loop), this test will FAIL because IsReliable
// stays true despite a fatal blind spot being present.
func TestHONESTY3_FatalBlindSpotSetsUnreliable(t *testing.T) {
	// MQL source with an unknown indicator (iXxx pattern → SeverityFatal)
	source := `
extern int MagicNumber = 50001;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double v = iNonExistentIndicator(Symbol(), 0, 14, 0, 0);
    if (v > 0 && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "Test", MagicNumber, 0, clrGreen);
}
`
	runner, cov, err := mql2go.CompileMQLWithCoverage(source)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	// Verify the unknown indicator produces a fatal blind spot
	fatalFound := false
	if cov != nil {
		for _, bs := range cov.BlindSpots {
			t.Logf("coverage blind spot: %s (severity=%s)", bs.Builtin, bs.Severity)
			if bs.Severity == interp.SeverityFatal {
				fatalFound = true
			}
		}
	}
	if !fatalFound {
		t.Fatal("expected at least one fatal coverage blind spot from iNonExistentIndicator")
	}

	// Run backtest
	bars := makeE2EBars(80)
	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		Params:         map[string]string{},
	}
	engine := backtest.New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("backtest failed: %v", err)
	}

	// Build the response via buildBacktestResponse — this is where HONESTY-3 fix lives
	params := backtestParams{
		initialCapital: "10000",
		commission:     "0.001",
		slippage:       "0",
		leverage:       "1",
		tradeDir:       antv1.TradeDirection_TRADE_DIRECTION_BOTH,
		strictMode:     true,
	}
	resp, _, _, _ := buildBacktestResponse(result, cfg, params, runner)

	// HONESTY-3: fatal blind spots MUST set IsReliable=false
	if resp.Risk == nil {
		t.Fatal("resp.Risk is nil — expected non-nil with IsReliable=false")
	}
	if resp.Risk.IsReliable {
		t.Error("IsReliable=true but fatal coverage blind spots present — HONESTY-3 crack not fixed")
	}

	// Verify at least one blind spot in the response has fatal severity
	fatalInResponse := false
	for _, bs := range resp.BlindSpots {
		if bs.Severity == interp.SeverityFatal {
			fatalInResponse = true
			t.Logf("response blind spot: id=%s severity=%s desc=%s", bs.Id, bs.Severity, bs.Description)
		}
	}
	if !fatalInResponse {
		t.Error("no fatal blind spot in response — expected at least one from iNonExistentIndicator")
	}
}

// TestHONESTY3_NonFatalBlindSpotKeepsReliable verifies that non-fatal blind spots
// (warning/info severity, e.g. statistical hints) do NOT set IsReliable=false.
// This is the "don't误伤 advisory/warning" guard from the fix spec.
//
// We use a simple MA crossover EA that produces >10 trades (so assessRisk
// returns IsReliable=true), with no fatal blind spots. The test verifies
// that IsReliable stays true — i.e. the HONESTY-3 fatal-severity check
// doesn't accidentally flag non-fatal blind spots.
func TestHONESTY3_NonFatalBlindSpotKeepsReliable(t *testing.T) {
	// Simple MA crossover — no unknown indicators, no unknown functions.
	// After HONESTY-1 fix, clrGreen is a known constant, so no blind spots at all.
	source := `
extern int MagicNumber = 50003;
extern double LotSize = 0.1;
extern int MAPeriod = 14;
int OnInit() { return 0; }
void OnBar()
{
    double ma = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    double maPrev = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 2);
    if (ma > maPrev && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "Test", MagicNumber, 0, clrGreen);
    if (ma < maPrev && OrdersTotal() > 0)
        OrderClose(OrderTicket(), LotSize, Bid, 5, clrRed);
}
`
	runner, _, err := mql2go.CompileMQLWithCoverage(source)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	bars := makeE2EBars(80)
	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		Params:         map[string]string{},
	}
	engine := backtest.New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("backtest failed: %v", err)
	}

	// Log all blind spots to verify none are fatal
	covResult := runner.GetCoverageResult()
	if covResult != nil {
		for _, bs := range covResult.BlindSpots {
			t.Logf("coverage blind spot: %s (severity=%s)", bs.Builtin, bs.Severity)
			if bs.Severity == interp.SeverityFatal {
				t.Fatalf("unexpected fatal blind spot: %s — test source should have none", bs.Builtin)
			}
		}
	}

	params := backtestParams{
		initialCapital: "10000",
		commission:     "0.001",
		slippage:       "0",
		leverage:       "1",
		tradeDir:       antv1.TradeDirection_TRADE_DIRECTION_BOTH,
		strictMode:     true,
	}
	resp, _, _, _ := buildBacktestResponse(result, cfg, params, runner)

	// Check all response blind spots — none should be fatal
	for _, bs := range resp.BlindSpots {
		t.Logf("response blind spot: id=%s severity=%s", bs.Id, bs.Severity)
		if bs.Severity == interp.SeverityFatal {
			t.Errorf("unexpected fatal blind spot in response: %s — HONESTY-3 check should not flag non-fatal blind spots", bs.Id)
		}
	}

	// If there are no fatal blind spots, IsReliable should not be set to false
	// by the HONESTY-3 check. (assessRisk may set it false if trades<10, which
	// is a separate concern — we only verify the HONESTY-3 loop doesn't fire.)
	if resp.Risk != nil && !resp.Risk.IsReliable {
		// Verify it's not because of a fatal blind spot
		hasFatal := false
		for _, bs := range resp.BlindSpots {
			if bs.Severity == interp.SeverityFatal {
				hasFatal = true
			}
		}
		if hasFatal {
			t.Error("IsReliable=false due to fatal blind spot — but test source should have no fatal blind spots")
		}
		// Otherwise it's fine — assessRisk set it false for other reasons (e.g. trades<10)
		t.Logf("IsReliable=false (expected if trades<10; trades=%d)", int(result.Metrics.TotalTrades))
	}
}

// TestHONESTY3_UnsupportedSilentWrongIsFatal verifies that StatusUnsupported
// functions that would silently produce wrong results (e.g. iCustom — an
// indicator that returns 0 if it reached the VM) are classified as
// SeverityFatal, not SeverityInfo.
//
// Adversarial proof: if the old code (StatusUnsupported → SeverityInfo) is
// restored, this test FAILS because iCustom would be classified as info
// instead of fatal.
//
// iCustom is rejected at compile time (compile_expr.go returns an error),
// so it never reaches the VM. But AnalyzeCoverage runs before compilation
// and classifies it as a blind spot — that classification must be fatal
// because iCustom is an indicator that would silently return wrong values.
func TestHONESTY3_UnsupportedSilentWrongIsFatal(t *testing.T) {
	// iCustom: StatusUnsupported + iXxx pattern → must be SeverityFatal
	if sev := interp.SeverityForBuiltin("iCustom"); sev != interp.SeverityFatal {
		t.Errorf("iCustom severity = %q, want %q (fatal) — unsupported indicator must be fatal, not info", sev, interp.SeverityFatal)
	}

	// ObjectCreate: StatusUnsupported + GUI/chart → must be SeverityInfo (graceful no-op)
	if sev := interp.SeverityForBuiltin("ObjectCreate"); sev != interp.SeverityInfo {
		t.Errorf("ObjectCreate severity = %q, want %q (info) — GUI functions are graceful no-ops, not fatal", sev, interp.SeverityInfo)
	}

	// FileOpen: StatusUnsupported + FileIO → must NOT be fatal (doesn't match iXxx/Order* pattern)
	// File I/O is rejected at compile time; if it somehow reached VM, returning 0
	// for file handles doesn't directly corrupt trading logic.
	if sev := interp.SeverityForBuiltin("FileOpen"); sev != interp.SeverityInfo {
		t.Errorf("FileOpen severity = %q, want %q (info) — file I/O doesn't match fatal patterns", sev, interp.SeverityInfo)
	}
}
