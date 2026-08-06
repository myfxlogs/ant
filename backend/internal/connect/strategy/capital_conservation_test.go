package strategy

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// helper: build a backtest.Trade with explicit Profit, Commission, Swap.
func makeTradeWithPCS(profit, commission, swap string) backtest.Trade {
	return backtest.Trade{
		Symbol:     "EURUSD",
		Side:       sdk.SideBuy,
		EntryTime:  time.UnixMilli(1000),
		ExitTime:   time.UnixMilli(2000),
		EntryPrice: decimal.NewFromFloat(1.1),
		ExitPrice:  decimal.NewFromFloat(1.11),
		Volume:     decimal.NewFromFloat(0.1),
		Profit:     parseDecimal(profit),
		Commission: parseDecimal(commission),
		Swap:       parseDecimal(swap),
		Comment:    "test",
	}
}

// helper: build a backtest.Result with the given finalBalance, trades, and initial capital.
func makeResult(finalBalance decimal.Decimal, trades []backtest.Trade, initialCapital decimal.Decimal) *backtest.Result {
	return &backtest.Result{
		Config: backtest.Config{
			InitialCapital: initialCapital,
		},
		Metrics:      makeMetrics(int32(len(trades))),
		FinalBalance: finalBalance,
		Trades:       trades,
	}
}

// --- checkCapitalConservation unit tests ---

// Positive: FinalBalance = 本金 + ΣProfit − ΣCommission − ΣSwap → invariant passes (nil).
func TestCheckCapitalConservation_Conserved(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := []backtest.Trade{
		makeTradeWithPCS("100", "5", "1"),
		makeTradeWithPCS("-50", "5", "0.5"),
		makeTradeWithPCS("200", "5", "2"),
	}
	// expected = 10000 + (100-50+200) - (5+5+5) - (1+0.5+2) = 10000 + 250 - 15 - 3.5 = 10231.5
	expected := decimal.NewFromFloat(10231.5)
	result := makeResult(expected, trades, initialCapital)

	bs := checkCapitalConservation(result)
	if bs != nil {
		t.Fatalf("expected nil BlindSpot for conserved capital, got %+v", bs)
	}
}

// Negative: FinalBalance deviates beyond tolerance → invariant triggers.
func TestCheckCapitalConservation_NotConserved(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := []backtest.Trade{
		makeTradeWithPCS("100", "5", "1"),
	}
	// expected = 10000 + 100 - 5 - 1 = 10094
	// actual = 10100 → diff = 6 > tolerance (max(0.01, 10000*1e-4=1) = 1)
	actual := decimal.NewFromInt(10100)
	result := makeResult(actual, trades, initialCapital)

	bs := checkCapitalConservation(result)
	if bs == nil {
		t.Fatal("expected BlindSpot for non-conserved capital, got nil")
	}
	if bs.Id != "capital_not_conserved" {
		t.Errorf("expected Id=capital_not_conserved, got %s", bs.Id)
	}
	if bs.Category != "invariant" {
		t.Errorf("expected Category=invariant, got %s", bs.Category)
	}
	if bs.Severity != interp.SeverityFatal {
		t.Errorf("expected Severity=%s, got %s", interp.SeverityFatal, bs.Severity)
	}
	if bs.Description == "" {
		t.Error("expected non-empty Description")
	}
}

// Edge: diff exactly == tolerance → violation (>= tolerance triggers).
func TestCheckCapitalConservation_DiffEqualsTolerance(t *testing.T) {
	// initialCapital = 100 → tolerance = max(0.01, 100*1e-4=0.01) = 0.01
	initialCapital := decimal.NewFromInt(100)
	trades := []backtest.Trade{} // no trades → expected = 100
	// actual = 100.01 → diff = 0.01 == tolerance → violation (>=)
	actual := decimal.New(10001, -2)
	result := makeResult(actual, trades, initialCapital)

	bs := checkCapitalConservation(result)
	if bs == nil {
		t.Fatal("expected BlindSpot when diff == tolerance (boundary: >= triggers)")
	}
}

// Edge: diff just below tolerance → passes.
func TestCheckCapitalConservation_DiffJustBelowTolerance(t *testing.T) {
	// initialCapital = 100 → tolerance = 0.01
	initialCapital := decimal.NewFromInt(100)
	trades := []backtest.Trade{}
	// actual = 100.00999 → diff = 0.00999 < 0.01 → passes
	actual := decimal.New(10000999, -5)
	result := makeResult(actual, trades, initialCapital)

	bs := checkCapitalConservation(result)
	if bs != nil {
		t.Fatalf("expected nil when diff < tolerance, got %+v", bs)
	}
}

// Edge: single trade, perfectly conserved → passes.
func TestCheckCapitalConservation_SingleTradeConserved(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := []backtest.Trade{
		makeTradeWithPCS("500", "10", "2"),
	}
	// expected = 10000 + 500 - 10 - 2 = 10488
	expected := decimal.NewFromInt(10488)
	result := makeResult(expected, trades, initialCapital)

	bs := checkCapitalConservation(result)
	if bs != nil {
		t.Fatalf("expected nil for single conserved trade, got %+v", bs)
	}
}

// Edge: zero trades, FinalBalance == initialCapital → passes.
func TestCheckCapitalConservation_ZeroTradesConserved(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := []backtest.Trade{}
	result := makeResult(initialCapital, trades, initialCapital)

	bs := checkCapitalConservation(result)
	if bs != nil {
		t.Fatalf("expected nil for zero trades with FinalBalance == initial, got %+v", bs)
	}
}

// Edge: tolerance scales with large initial capital.
// initialCapital = 1,000,000 → tolerance = max(0.01, 100) = 100
func TestCheckCapitalConservation_LargeCapitalToleranceScaling(t *testing.T) {
	initialCapital := decimal.NewFromInt(1000000)
	trades := []backtest.Trade{
		makeTradeWithPCS("1000", "50", "10"),
	}
	// expected = 1000000 + 1000 - 50 - 10 = 1000940
	// actual = 1000990 → diff = 50 < 100 (tolerance) → passes
	actual := decimal.NewFromInt(1000990)
	result := makeResult(actual, trades, initialCapital)

	bs := checkCapitalConservation(result)
	if bs != nil {
		t.Fatalf("expected nil when diff < scaled tolerance, got %+v", bs)
	}

	// Now diff = 100 == tolerance → violation
	actual2 := decimal.NewFromInt(1001040)
	result2 := makeResult(actual2, trades, initialCapital)
	bs2 := checkCapitalConservation(result2)
	if bs2 == nil {
		t.Fatal("expected BlindSpot when diff == scaled tolerance (boundary)")
	}
}

// Edge: negative profit (losing trade), still conserved → passes.
func TestCheckCapitalConservation_NegativeProfitConserved(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := []backtest.Trade{
		makeTradeWithPCS("-300", "5", "1"),
	}
	// expected = 10000 - 300 - 5 - 1 = 9694
	expected := decimal.NewFromInt(9694)
	result := makeResult(expected, trades, initialCapital)

	bs := checkCapitalConservation(result)
	if bs != nil {
		t.Fatalf("expected nil for conserved negative profit, got %+v", bs)
	}
}

// Edge: zero initial capital → tolerance = max(0.01, 0) = 0.01.
func TestCheckCapitalConservation_ZeroInitialCapital(t *testing.T) {
	initialCapital := decimal.Zero
	trades := []backtest.Trade{
		makeTradeWithPCS("100", "5", "1"),
	}
	// expected = 0 + 100 - 5 - 1 = 94
	expected := decimal.NewFromInt(94)
	result := makeResult(expected, trades, initialCapital)

	bs := checkCapitalConservation(result)
	if bs != nil {
		t.Fatalf("expected nil for zero initial capital conserved, got %+v", bs)
	}

	// Violation: actual = 95 → diff = 1 > 0.01
	actual := decimal.NewFromInt(95)
	result2 := makeResult(actual, trades, initialCapital)
	bs2 := checkCapitalConservation(result2)
	if bs2 == nil {
		t.Fatal("expected BlindSpot for zero initial capital not conserved")
	}
}

// 【关键新增】期末有大额未平仓浮盈，FinalBalance 仍守恒 → 必须返回 nil（不误报）。
// 这是本次修正的核心：旧实现用 Equity（= balance + 浮盈）对已实现等式，
// 期末有 5000 浮盈时 diff = 5000 → 误报。改用 FinalBalance 后浮盈无关。
func TestCheckCapitalConservation_UnrealizedProfitNotFlagged(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := []backtest.Trade{
		makeTradeWithPCS("100", "5", "1"),
		makeTradeWithPCS("200", "5", "2"),
	}
	// expected FinalBalance = 10000 + 300 - 10 - 3 = 10287
	finalBalance := decimal.NewFromInt(10287)
	result := makeResult(finalBalance, trades, initialCapital)

	// Simulate unrealized floating PnL of 5000 — this would have caused
	// a false positive with the old Equity-based check (diff = 5000 >> tolerance).
	// With FinalBalance, floating PnL is irrelevant — invariant must pass.
	result.Equity = []backtest.EquityPoint{
		{Time: time.UnixMilli(0), Equity: initialCapital, Bar: 0},
		{Time: time.UnixMilli(1000), Equity: finalBalance.Add(decimal.NewFromInt(5000)), Bar: 1},
	}

	bs := checkCapitalConservation(result)
	if bs != nil {
		t.Fatalf("expected nil with unrealized floating PnL (FinalBalance conserved), got %+v", bs)
	}
}

// 【关键新增变体】期末有大额未平仓浮亏，FinalBalance 仍守恒 → 必须返回 nil。
func TestCheckCapitalConservation_UnrealizedLossNotFlagged(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := []backtest.Trade{
		makeTradeWithPCS("100", "5", "1"),
	}
	// expected FinalBalance = 10000 + 100 - 5 - 1 = 10094
	finalBalance := decimal.NewFromInt(10094)
	result := makeResult(finalBalance, trades, initialCapital)

	// Unrealized floating loss of 8000 — Equity = 10094 - 8000 = 2094
	result.Equity = []backtest.EquityPoint{
		{Time: time.UnixMilli(0), Equity: initialCapital, Bar: 0},
		{Time: time.UnixMilli(1000), Equity: finalBalance.Sub(decimal.NewFromInt(8000)), Bar: 1},
	}

	bs := checkCapitalConservation(result)
	if bs != nil {
		t.Fatalf("expected nil with unrealized floating loss (FinalBalance conserved), got %+v", bs)
	}
}

// --- Integration: buildBacktestResponse with capital conservation ---

// Integration positive: capital conserved → no capital_not_conserved blind spot.
func TestBuildBacktestResponse_CapitalConservationPass(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := make([]backtest.Trade, 12)
	for i := range trades {
		trades[i] = makeTradeWithPCS("100", "5", "1")
		trades[i].Volume = decimal.NewFromFloat(0.1)
	}
	// expected FinalBalance = 10000 + 12*100 - 12*5 - 12*1 = 10000 + 1200 - 60 - 12 = 11128
	expected := decimal.NewFromInt(11128)
	result := makeResult(expected, trades, initialCapital)

	cfg := backtest.Config{
		InitialCapital: initialCapital,
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	for _, bs := range resp.BlindSpots {
		if bs.Id == "capital_not_conserved" {
			t.Fatal("expected no capital_not_conserved blind spot when capital is conserved")
		}
	}
}

// Integration negative: capital not conserved → IsReliable=false + blind spot present.
func TestBuildBacktestResponse_CapitalConservationFail(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := make([]backtest.Trade, 12)
	for i := range trades {
		trades[i] = makeTradeWithPCS("100", "5", "1")
		trades[i].Volume = decimal.NewFromFloat(0.1)
	}
	// expected = 11128, but we set 12000 → diff = 872 >> tolerance (1)
	actual := decimal.NewFromInt(12000)
	result := makeResult(actual, trades, initialCapital)

	cfg := backtest.Config{
		InitialCapital: initialCapital,
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	if resp.Risk.IsReliable {
		t.Error("expected IsReliable=false when capital is not conserved")
	}

	found := false
	for _, bs := range resp.BlindSpots {
		if bs.Id == "capital_not_conserved" {
			found = true
			if bs.Severity != interp.SeverityFatal {
				t.Errorf("expected Severity=%s, got %s", interp.SeverityFatal, bs.Severity)
			}
			if bs.Description == "" {
				t.Error("expected non-empty Description for capital_not_conserved blind spot")
			}
		}
	}
	if !found {
		t.Fatal("expected capital_not_conserved blind spot in response")
	}
}

// Integration: unrealized floating PnL at backtest end does not trigger false positive.
func TestBuildBacktestResponse_CapitalConservationWithUnrealizedPnL(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := make([]backtest.Trade, 12)
	for i := range trades {
		trades[i] = makeTradeWithPCS("100", "5", "1")
		trades[i].Volume = decimal.NewFromFloat(0.1)
	}
	// FinalBalance = 11128 (conserved), but Equity has large unrealized profit
	finalBalance := decimal.NewFromInt(11128)
	result := makeResult(finalBalance, trades, initialCapital)
	// Equity includes 8000 unrealized floating profit — old impl would flag this
	result.Equity = []backtest.EquityPoint{
		{Time: time.UnixMilli(0), Equity: initialCapital, Bar: 0},
		{Time: time.UnixMilli(1000), Equity: finalBalance.Add(decimal.NewFromInt(8000)), Bar: 1},
	}

	cfg := backtest.Config{
		InitialCapital: initialCapital,
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	for _, bs := range resp.BlindSpots {
		if bs.Id == "capital_not_conserved" {
			t.Fatal("expected no capital_not_conserved blind spot with unrealized PnL (FinalBalance conserved)")
		}
	}
}

// Integration edge: both volume and capital invariants violated → both blind spots present.
func TestBuildBacktestResponse_BothInvariantsViolated(t *testing.T) {
	initialCapital := decimal.NewFromInt(10000)
	trades := make([]backtest.Trade, 12)
	for i := range trades {
		trades[i] = makeTradeWithPCS("100", "5", "1")
		trades[i].Volume = decimal.NewFromFloat(0.1)
	}
	trades[3].Volume = decimal.Zero // volume invariant violation
	// expected = 11128, actual = 12000 → capital invariant violation
	actual := decimal.NewFromInt(12000)
	result := makeResult(actual, trades, initialCapital)

	cfg := backtest.Config{
		InitialCapital: initialCapital,
	}
	params := backtestParams{}
	vmRunner := newMinimalVMRunner(t)

	resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

	if resp.Risk.IsReliable {
		t.Error("expected IsReliable=false when both invariants are violated")
	}

	foundVolume := false
	foundCapital := false
	for _, bs := range resp.BlindSpots {
		if bs.Id == "zero_volume_trade" {
			foundVolume = true
		}
		if bs.Id == "capital_not_conserved" {
			foundCapital = true
		}
	}
	if !foundVolume {
		t.Error("expected zero_volume_trade blind spot when volume invariant is violated")
	}
	if !foundCapital {
		t.Error("expected capital_not_conserved blind spot when capital invariant is violated")
	}
}
