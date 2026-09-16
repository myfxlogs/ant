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
// causes IsReliable=false in the backtest response — AND that this is the
// ONLY reason IsReliable is false (not the <10 trades rule in assessRisk).
//
// Design (VM-HONESTY-3-REVIEW):
//   - MA crossover (MAPeriod=3, 200 oscillating bars) produces ≥10 closed trades
//     → assessRisk sets IsReliable=true.
//   - iNonExistentIndicator is placed in a dead branch if(1==0) so static
//     coverage still detects it (SeverityFatal) but runtime never calls it
//     → no interference with trading.
//   - If the HONESTY-3 fatal-severity loop is removed, IsReliable stays true
//     (trades≥10) → test RED. This is a true adversarial proof.
//
// Adversarial proof: comment out the fatal-severity loop in
// backtest_worker_vm.go:344-349 → IsReliable=true (trades≥10) → RED.
func TestHONESTY3_FatalBlindSpotSetsUnreliable(t *testing.T) {
	source := `
extern int MagicNumber = 50001;
extern double LotSize = 0.1;
extern int MAPeriod = 3;
int OnInit() { return 0; }
void OnBar()
{
    double ma = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    double maPrev = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 2);
    if (ma > maPrev && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "T", MagicNumber, 0, clrGreen);
    if (ma < maPrev && OrdersTotal() > 0)
    {
        if (OrderSelect(0, SELECT_BY_POS, MODE_TRADES))
            OrderClose(OrderTicket(), LotSize, Bid, 5, clrRed);
    }
    // Dead branch: static coverage detects iNonExistentIndicator (iXxx → SeverityFatal),
    // but runtime never executes it → no interference with trading.
    if (1 == 0)
    {
        double v = iNonExistentIndicator(Symbol(), 0, 14, 0, 0);
    }
}`
	runner, cov, err := mql2go.CompileMQLWithCoverage(source)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	// 1. Verify the unknown indicator produces a fatal coverage blind spot
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
		t.Fatal("expected at least one fatal coverage blind spot from iNonExistentIndicator in dead branch")
	}

	// Run backtest with 200 oscillating bars → MA crossover produces ≥10 trades
	bars := makeE2EBars(200)
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

	// 2. KEY: ≥10 trades → assessRisk sets IsReliable=true.
	//    If IsReliable=false, it can ONLY come from the fatal-severity loop.
	t.Logf("TotalTrades=%d", result.Metrics.TotalTrades)
	if result.Metrics.TotalTrades < 10 {
		t.Fatalf("TotalTrades=%d, need ≥10 so assessRisk sets IsReliable=true (got <10 — adjust MAPeriod/bars)", result.Metrics.TotalTrades)
	}

	params := backtestParams{
		code:           source,
		initialCapital: "10000",
		commission:     "0.001",
		slippage:       "0",
		leverage:       "1",
		tradeDir:       antv1.TradeDirection_TRADE_DIRECTION_BOTH,
		strictMode:     true,
	}
	resp, _, _, _ := buildBacktestResponse(result, cfg, params, runner)

	// 3. HONESTY-3: fatal blind spots MUST set IsReliable=false
	if resp.Risk == nil {
		t.Fatal("resp.Risk is nil — expected non-nil with IsReliable=false")
	}
	if resp.Risk.IsReliable {
		t.Error("IsReliable=true but fatal coverage blind spots present and trades≥10 — HONESTY-3 fatal loop not firing")
	}

	// 4. Verify at least one blind spot in the response has fatal severity
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
// (warning severity, e.g. R06 OrderSelect+MODE_HISTORY) do NOT set
// IsReliable=false — the HONESTY-3 fatal-severity loop must not误伤 them.
//
// Design (VM-HONESTY-3-REVIEW):
//   - Same MA crossover (≥10 trades) → assessRisk sets IsReliable=true.
//   - Dead branch if(1==0) contains OrderSelect(0,SELECT_BY_POS,MODE_HISTORY)
//     → R06 rule (source text scan) fires → SeverityWarning blind spot.
//   - No fatal blind spots (all builtins implemented, no iXxx).
//   - Strong assertion: IsReliable must be true.
//
// Adversarial proof: change the fatal loop condition from
// `bs.Severity == interp.SeverityFatal` to `bs.Severity != interp.SeverityInfo`
// → warning blind spots also flip IsReliable → RED.
func TestHONESTY3_NonFatalBlindSpotKeepsReliable(t *testing.T) {
	source := `
extern int MagicNumber = 50003;
extern double LotSize = 0.1;
extern int MAPeriod = 3;
int OnInit() { return 0; }
void OnBar()
{
    double ma = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    double maPrev = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 2);
    if (ma > maPrev && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "T", MagicNumber, 0, clrGreen);
    if (ma < maPrev && OrdersTotal() > 0)
    {
        if (OrderSelect(0, SELECT_BY_POS, MODE_TRADES))
            OrderClose(OrderTicket(), LotSize, Bid, 5, clrRed);
    }
    // Dead branch: R06 rule scans source text for ORDERSELECT + MODE_HISTORY
    // → SeverityWarning. No runtime execution needed (text-based rule).
    if (1 == 0)
    {
        OrderSelect(0, SELECT_BY_POS, MODE_HISTORY);
    }
}`
	runner, _, err := mql2go.CompileMQLWithCoverage(source)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	bars := makeE2EBars(200)
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

	// 1. ≥10 trades → assessRisk sets IsReliable=true
	t.Logf("TotalTrades=%d", result.Metrics.TotalTrades)
	if result.Metrics.TotalTrades < 10 {
		t.Fatalf("TotalTrades=%d, need ≥10 so assessRisk sets IsReliable=true (got <10 — adjust MAPeriod/bars)", result.Metrics.TotalTrades)
	}

	params := backtestParams{
		code:           source,
		initialCapital: "10000",
		commission:     "0.001",
		slippage:       "0",
		leverage:       "1",
		tradeDir:       antv1.TradeDirection_TRADE_DIRECTION_BOTH,
		strictMode:     true,
	}
	resp, _, _, _ := buildBacktestResponse(result, cfg, params, runner)

	// 2. No fatal blind spots in response
	for _, bs := range resp.BlindSpots {
		t.Logf("response blind spot: id=%s severity=%s", bs.Id, bs.Severity)
		if bs.Severity == interp.SeverityFatal {
			t.Fatalf("unexpected fatal blind spot in response: %s — test source should have none", bs.Id)
		}
	}

	// 3. At least one warning blind spot (R06_orderselect_history) — proves
	//    non-fatal blind spot exists and passes through the fatal loop
	warningFound := false
	for _, bs := range resp.BlindSpots {
		if bs.Severity == interp.SeverityWarning {
			warningFound = true
		}
	}
	if !warningFound {
		t.Fatal("expected at least one SeverityWarning blind spot (R06_orderselect_history) — dead branch OrderSelect+MODE_HISTORY should trigger R06")
	}

	// 4. STRONG assertion: IsReliable must be true.
	//    Fatal loop sees only warning → must not flip.
	if resp.Risk == nil {
		t.Fatal("resp.Risk is nil — expected non-nil with IsReliable=true")
	}
	if !resp.Risk.IsReliable {
		t.Error("IsReliable=false but no fatal blind spots and trades≥10 — HONESTY-3 fatal loop误伤 warning blind spots")
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
