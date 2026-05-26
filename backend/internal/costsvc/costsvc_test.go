package costsvc

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestSpreadCost_Forex(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	cost := m.SpreadCost(1.0) // 1 standard lot
	// half spread: 0.5 pips * $10/pip = $5
	expected := 5.0
	if math.Abs(cost-expected) > 0.01 {
		t.Fatalf("spread cost = %.4f, want %.4f", cost, expected)
	}
}

func TestSpreadCost_ZeroSpread(t *testing.T) {
	m := &CostModel{Symbol: "TEST", PipValue: 10.0, SpreadPips: 0}
	cost := m.SpreadCost(1.0)
	if cost != 0 {
		t.Fatalf("zero spread should give 0 cost, got %.4f", cost)
	}
}

func TestCommission_PerLot(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	cost := m.Commission(1.0, 100000)
	// $7 per lot
	if math.Abs(cost-7.0) > 0.01 {
		t.Fatalf("commission per lot = %.4f, want 7.00", cost)
	}
}

func TestCommission_PerNotional(t *testing.T) {
	m := DefaultCryptoModel("BTCUSD")
	// CommissionBps=10, notional=50000, cost = 50000 * 10/10000 = 50
	cost := m.Commission(1.0, 50000)
	if math.Abs(cost-50.0) > 0.01 {
		t.Fatalf("commission bps = %.4f, want 50.00", cost)
	}
}

func TestCommission_MinFloor(t *testing.T) {
	m := &CostModel{Symbol: "TEST", MinCommission: 5.0}
	cost := m.Commission(0.01, 1000)
	if math.Abs(cost-5.0) > 0.01 {
		t.Fatalf("min commission floor = %.4f, want 5.00", cost)
	}
}

func TestSwapCost_Long(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	// SwapLong=-3.5, 1 lot, holding 3 days → -3.5 * 1 * 3 = -10.50
	cost := m.SwapCost("buy", 1.0, 1.0850, 100000, 3)
	if math.Abs(cost-(-10.50)) > 0.01 {
		t.Fatalf("swap long = %.4f, want -10.50", cost)
	}
}

func TestSwapCost_Short(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	// SwapShort=0.5, 1 lot, 3 days → 0.5 * 1 * 3 = 1.50
	cost := m.SwapCost("sell", 1.0, 1.0850, 100000, 3)
	if math.Abs(cost-1.50) > 0.01 {
		t.Fatalf("swap short = %.4f, want 1.50", cost)
	}
}

func TestSwapCost_ZeroHolding(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	cost := m.SwapCost("buy", 1.0, 1.0850, 100000, 0)
	if cost != 0 {
		t.Fatalf("zero holding days should give 0 swap, got %.4f", cost)
	}
}

func TestFundingCost(t *testing.T) {
	m := DefaultCryptoModel("BTCUSD")
	// funding rate 0.0001, interval 8h, notional 50000
	// holding 24h → 3 intervals → 0.0001 * 50000 * 3 = 15
	cost := m.FundingCost(1.0, 50000, 1.0, 24*time.Hour)
	if math.Abs(cost-15.0) > 0.01 {
		t.Fatalf("funding cost = %.4f, want 15.00", cost)
	}
}

func TestSlippageCost(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	// SlippageBps=0.5, notional=108500, cost = 108500*0.5/10000 = 5.425
	cost := m.SlippageCost(1.0, 1.0850, 100000)
	if math.Abs(cost-5.425) > 0.01 {
		t.Fatalf("slippage cost = %.4f, want 5.425", cost)
	}
}

func TestEstimate_Forex_Buy(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	est := m.Estimate(EstimateParams{
		Side: "buy", Lots: 1.0, Price: 1.0850, ContractSize: 100000, HoldingDays: 1,
	})
	// spread=5 + commission=7 + slippage=5.425 + swap=-3.5 + funding=0 = 13.925
	if math.Abs(est.TotalCost-13.925) > 0.1 {
		t.Fatalf("total cost = %.4f, want ~13.925", est.TotalCost)
	}
	if est.CostBps <= 0 {
		t.Fatalf("cost_bps should be positive, got %.2f", est.CostBps)
	}
	t.Logf("Forex buy estimate: total=%.4f bps=%.2f spread=%.2f comm=%.2f slip=%.2f swap=%.2f",
		est.TotalCost, est.CostBps, est.SpreadCost, est.Commission, est.SlippageCost, est.SwapCost)
}

func TestEstimate_Crypto(t *testing.T) {
	m := DefaultCryptoModel("BTCUSD")
	est := m.Estimate(EstimateParams{
		Side: "buy", Lots: 1.0, Price: 50000, ContractSize: 1, HoldingDuration: 24 * time.Hour,
	})
	// spread = 5 pips * 1 * 1 = 5
	// commission = 50000 * 10/10000 = 50
	// slippage = 50000 * 2/10000 = 10
	// funding = 0.0001 * 50000 * 3 = 15
	// total = 5 + 50 + 10 + 15 = 80
	if est.TotalCost < 70 || est.TotalCost > 90 {
		t.Fatalf("total cost = %.4f, want ~80", est.TotalCost)
	}
	t.Logf("Crypto estimate: total=%.4f bps=%.2f", est.TotalCost, est.CostBps)
}

func TestGrossToNetFillPrice_Buy(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	net := m.GrossToNetFillPrice(1.0850, EstimateParams{
		Side: "buy", Lots: 1.0, Price: 1.0850, ContractSize: 100000, HoldingDays: 0,
	})
	// Gross=1.0850, costs=spread+commission+slippage = 5+7+5.425 = 17.425
	// cost per unit = 17.425 / 100000 = 0.00017425
	// net = 1.0850 + 0.00017425 = 1.08517425
	if net <= 1.0850 {
		t.Fatalf("buy net fill should be > gross (costs add to price), got %.6f", net)
	}
}

func TestGrossToNetFillPrice_Sell(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	net := m.GrossToNetFillPrice(1.0850, EstimateParams{
		Side: "sell", Lots: 1.0, Price: 1.0850, ContractSize: 100000, HoldingDays: 0,
	})
	// net = gross - cost per unit, so net < gross for sell
	if net >= 1.0850 {
		t.Fatalf("sell net fill should be < gross (costs reduce proceeds), got %.6f", net)
	}
}

func TestSnapshot_Roundtrip(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	snap := m.Snapshot()
	if snap.Symbol != "EURUSD" {
		t.Fatalf("snapshot symbol: %s", snap.Symbol)
	}
	if snap.SpreadPips != m.SpreadPips {
		t.Fatalf("snapshot spread mismatch")
	}
	if snap.FrozenAt.IsZero() {
		t.Fatal("frozen_at should be set")
	}
}

func TestCostModel_ZeroNotional(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	est := m.Estimate(EstimateParams{Side: "buy", Lots: 0, Price: 1.0850})
	if est.TotalCost != 0 {
		t.Fatalf("zero lots should give zero total cost, got %.4f", est.TotalCost)
	}
}

func TestEffectiveSwapDays_NoWednesday(t *testing.T) {
	// Monday + 1 day (Mon→Tue) = 1 swap charge
	mon := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) // Monday
	eff := EffectiveSwapDays(mon, 1)
	if eff != 1 {
		t.Fatalf("Mon→Tue: want 1, got %d", eff)
	}
}

func TestEffectiveSwapDays_Wednesday(t *testing.T) {
	// Tuesday + 2 days (Tue→Wed→Thu): Wed = 3 charges, total = 4
	tue := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC) // Tuesday
	eff := EffectiveSwapDays(tue, 2)
	if eff != 4 {
		t.Fatalf("Tue→Thu (spanning Wed): want 4, got %d", eff)
	}
}

func TestEffectiveSwapDays_Zero(t *testing.T) {
	eff := EffectiveSwapDays(time.Now(), 0)
	if eff != 0 {
		t.Fatalf("zero days: want 0, got %d", eff)
	}
}

func TestSwapCostDate_TripleWednesday(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	// Tuesday start, 2 day hold (spans Wed) → effective 4 swap charges
	tue := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	cost := m.SwapCostDate("sell", 1.0, tue, 2)
	// SwapShort=0.5, effective days=4, lots=1 → 0.5 * 1 * 4 = 2.0
	if math.Abs(cost-2.0) > 0.01 {
		t.Fatalf("triple Wed swap: want 2.00, got %.4f", cost)
	}
}

func TestSnapshot_FrozenAtClock(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	snap := m.Snapshot()
	if snap.FrozenAt.IsZero() {
		t.Fatal("frozen_at should be set")
	}
	// Should be within 1s of now.
	if time.Since(snap.FrozenAt) > time.Second {
		t.Fatalf("frozen_at too far from now: %v", snap.FrozenAt)
	}
}

func TestSnapshotConfig(t *testing.T) {
	models := map[string]*CostModel{
		"EURUSD": DefaultForexModel("EURUSD"),
		"BTCUSD": DefaultCryptoModel("BTCUSD"),
	}
	data, err := SnapshotConfig("test_broker", models)
	if err != nil {
		t.Fatalf("SnapshotConfig: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("snapshot data should not be empty")
	}
	if !strings.Contains(string(data), "EURUSD") {
		t.Fatal("snapshot should contain EURUSD")
	}
	if !strings.Contains(string(data), "test_broker") {
		t.Fatal("snapshot should contain broker name")
	}
}

func TestSnapshotFromList(t *testing.T) {
	models := []*CostModel{DefaultForexModel("EURUSD"), DefaultCryptoModel("BTCUSD")}
	data, err := SnapshotFromList("test_broker", models)
	if err != nil {
		t.Fatalf("SnapshotFromList: %v", err)
	}
	if !strings.Contains(string(data), "BTCUSD") {
		t.Fatal("snapshot should contain BTCUSD")
	}
}

func TestStaticEstimator(t *testing.T) {
	m := DefaultForexModel("EURUSD")
	est := &StaticEstimator{Model: m}
	breakdown := est.Estimate(t.Context(), EstimateParams{
		Side: "buy", Lots: 1.0, Price: 1.0850, ContractSize: 100000,
	})
	if breakdown.TotalCost <= 0 {
		t.Fatalf("static estimator should return costs, got %.4f", breakdown.TotalCost)
	}
}

func TestMultiModelEstimator_KnownSymbol(t *testing.T) {
	est := &MultiModelEstimator{
		Models:  map[string]*CostModel{"EURUSD": DefaultForexModel("EURUSD")},
		Default: nil,
	}
	breakdown := est.Estimate(t.Context(), EstimateParams{
		Symbol: "EURUSD", Side: "buy", Lots: 1.0, Price: 1.0850, ContractSize: 100000,
	})
	if breakdown.TotalCost <= 0 {
		t.Fatalf("known symbol should return costs, got %.4f", breakdown.TotalCost)
	}
}

func TestMultiModelEstimator_FallbackDefault(t *testing.T) {
	est := &MultiModelEstimator{
		Models:  map[string]*CostModel{},
		Default: DefaultForexModel("EURUSD"),
	}
	breakdown := est.Estimate(t.Context(), EstimateParams{
		Symbol: "UNKNOWN", Side: "buy", Lots: 1.0, Price: 1.0850, ContractSize: 100000,
	})
	if breakdown.TotalCost <= 0 {
		t.Fatalf("fallback default should return costs, got %.4f", breakdown.TotalCost)
	}
}

func TestMultiModelEstimator_NoModel(t *testing.T) {
	est := &MultiModelEstimator{
		Models:  map[string]*CostModel{},
		Default: nil,
	}
	breakdown := est.Estimate(t.Context(), EstimateParams{
		Symbol: "UNKNOWN", Side: "buy", Lots: 1.0, Price: 1.0850,
	})
	if breakdown.TotalCost != 0 {
		t.Fatalf("no model should return zero cost, got %.4f", breakdown.TotalCost)
	}
}
