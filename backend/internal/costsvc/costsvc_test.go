package costsvc

import (
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func decClose(a, b decimal.Decimal) bool {
	return a.Sub(b).Abs().LessThan(decF(0.01))
}

func TestSpreadCost_Forex(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	cost := m.SpreadCost(decF(1.0)) // 1 standard lot
	// half spread: 0.5 pips * $10/pip = $5
	expected := decF(5.0)
	if !decClose(cost, expected) {
		t.Fatalf("spread cost = %s, want %s", cost.String(), expected.String())
	}
}

func TestSpreadCost_ZeroSpread(t *testing.T) {
	t.Parallel()
	m := &CostModel{Symbol: "TEST", PipValue: decF(10.0), SpreadPips: decimal.Zero}
	cost := m.SpreadCost(decF(1.0))
	if !cost.Equal(decimal.Zero) {
		t.Fatalf("zero spread should give 0 cost, got %s", cost.String())
	}
}

func TestCommission_PerLot(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	cost := m.Commission(decF(1.0), decF(100000))
	// $7 per lot
	if !decClose(cost, decF(7.0)) {
		t.Fatalf("commission per lot = %s, want 7.00", cost.String())
	}
}

func TestCommission_PerNotional(t *testing.T) {
	t.Parallel()
	m := DefaultCryptoModel("BTCUSD")
	// CommissionBps=10, notional=50000, cost = 50000 * 10/10000 = 50
	cost := m.Commission(decF(1.0), decF(50000))
	if !decClose(cost, decF(50.0)) {
		t.Fatalf("commission bps = %s, want 50.00", cost.String())
	}
}

func TestCommission_MinFloor(t *testing.T) {
	t.Parallel()
	m := &CostModel{Symbol: "TEST", MinCommission: decF(5.0)}
	cost := m.Commission(decF(0.01), decF(1000))
	if !decClose(cost, decF(5.0)) {
		t.Fatalf("min commission floor = %s, want 5.00", cost.String())
	}
}

func TestSwapCost_Long(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	// SwapLong=-3.5, 1 lot, holding 3 days → -3.5 * 1 * 3 = -10.50
	cost := m.SwapCost("buy", decF(1.0), decF(1.0850), decF(100000), decF(3))
	if !decClose(cost, decF(-10.50)) {
		t.Fatalf("swap long = %s, want -10.50", cost.String())
	}
}

func TestSwapCost_Short(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	// SwapShort=0.5, 1 lot, 3 days → 0.5 * 1 * 3 = 1.50
	cost := m.SwapCost("sell", decF(1.0), decF(1.0850), decF(100000), decF(3))
	if !decClose(cost, decF(1.50)) {
		t.Fatalf("swap short = %s, want 1.50", cost.String())
	}
}

func TestSwapCost_ZeroHolding(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	cost := m.SwapCost("buy", decF(1.0), decF(1.0850), decF(100000), decimal.Zero)
	if !cost.Equal(decimal.Zero) {
		t.Fatalf("zero holding days should give 0 swap, got %s", cost.String())
	}
}

func TestFundingCost(t *testing.T) {
	t.Parallel()
	m := DefaultCryptoModel("BTCUSD")
	// funding rate 0.0001, interval 8h, notional 50000
	// holding 24h → 3 intervals → 0.0001 * 50000 * 3 = 15
	cost := m.FundingCost(decF(1.0), decF(50000), decF(1.0), 24*time.Hour)
	if !decClose(cost, decF(15.0)) {
		t.Fatalf("funding cost = %s, want 15.00", cost.String())
	}
}

func TestSlippageCost(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	// SlippageBps=0.5, notional=108500, cost = 108500*0.5/10000 = 5.425
	cost := m.SlippageCost(decF(1.0), decF(1.0850), decF(100000))
	if !decClose(cost, decF(5.425)) {
		t.Fatalf("slippage cost = %s, want 5.425", cost.String())
	}
}

func TestEstimate_Forex_Buy(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	est := m.Estimate(EstimateParams{
		Side: "buy", Lots: decF(1.0), Price: decF(1.0850), ContractSize: decF(100000), HoldingDays: decF(1),
	})
	// spread=5 + commission=7 + slippage=5.425 + swap=-3.5 + funding=0 = 13.925
	if !decClose(est.TotalCost, decF(13.925)) {
		t.Fatalf("total cost = %s, want ~13.925", est.TotalCost.String())
	}
	if !est.CostBps.GreaterThan(decimal.Zero) {
		t.Fatalf("cost_bps should be positive, got %s", est.CostBps.String())
	}
	t.Logf("Forex buy estimate: total=%s bps=%s spread=%s comm=%s slip=%s swap=%s",
		est.TotalCost.String(), est.CostBps.String(), est.SpreadCost.String(), est.Commission.String(), est.SlippageCost.String(), est.SwapCost.String())
}

func TestEstimate_Crypto(t *testing.T) {
	t.Parallel()
	m := DefaultCryptoModel("BTCUSD")
	est := m.Estimate(EstimateParams{
		Side: "buy", Lots: decF(1.0), Price: decF(50000), ContractSize: decF(1), HoldingDuration: 24 * time.Hour,
	})
	// spread = 5 pips * 1 * 1 = 5
	// commission = 50000 * 10/10000 = 50
	// slippage = 50000 * 2/10000 = 10
	// funding = 0.0001 * 50000 * 3 = 15
	// total = 5 + 50 + 10 + 15 = 80
	if est.TotalCost.LessThan(decF(70)) || est.TotalCost.GreaterThan(decF(90)) {
		t.Fatalf("total cost = %s, want ~80", est.TotalCost.String())
	}
	t.Logf("Crypto estimate: total=%s bps=%s", est.TotalCost.String(), est.CostBps.String())
}

func TestGrossToNetFillPrice_Buy(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	net := m.GrossToNetFillPrice(decF(1.0850), EstimateParams{
		Side: "buy", Lots: decF(1.0), Price: decF(1.0850), ContractSize: decF(100000), HoldingDays: decimal.Zero,
	})
	// Gross=1.0850, costs=spread+commission+slippage = 5+7+5.425 = 17.425
	// cost per unit = 17.425 / 100000 = 0.00017425
	// net = 1.0850 + 0.00017425 = 1.08517425
	if !net.GreaterThan(decF(1.0850)) {
		t.Fatalf("buy net fill should be > gross (costs add to price), got %s", net.String())
	}
}

func TestGrossToNetFillPrice_Sell(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	net := m.GrossToNetFillPrice(decF(1.0850), EstimateParams{
		Side: "sell", Lots: decF(1.0), Price: decF(1.0850), ContractSize: decF(100000), HoldingDays: decimal.Zero,
	})
	// net = gross - cost per unit, so net < gross for sell
	if !net.LessThan(decF(1.0850)) {
		t.Fatalf("sell net fill should be < gross (costs reduce proceeds), got %s", net.String())
	}
}

func TestSnapshot_Roundtrip(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	snap := m.Snapshot()
	if snap.Symbol != "EURUSD" {
		t.Fatalf("snapshot symbol: %s", snap.Symbol)
	}
	if !snap.SpreadPips.Equal(m.SpreadPips) {
		t.Fatalf("snapshot spread mismatch")
	}
	if snap.FrozenAt.IsZero() {
		t.Fatal("frozen_at should be set")
	}
}

func TestCostModel_ZeroNotional(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	est := m.Estimate(EstimateParams{Side: "buy", Lots: decimal.Zero, Price: decF(1.0850)})
	if !est.TotalCost.Equal(decimal.Zero) {
		t.Fatalf("zero lots should give zero total cost, got %s", est.TotalCost.String())
	}
}

func TestEffectiveSwapDays_NoWednesday(t *testing.T) {
	t.Parallel()
	// Monday + 1 day (Mon→Tue) = 1 swap charge
	mon := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC) // Monday
	eff := EffectiveSwapDays(mon, 1)
	if eff != 1 {
		t.Fatalf("Mon→Tue: want 1, got %d", eff)
	}
}

func TestEffectiveSwapDays_Wednesday(t *testing.T) {
	t.Parallel()
	// Tuesday + 2 days (Tue→Wed→Thu): Wed = 3 charges, total = 4
	tue := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC) // Tuesday
	eff := EffectiveSwapDays(tue, 2)
	if eff != 4 {
		t.Fatalf("Tue→Thu (spanning Wed): want 4, got %d", eff)
	}
}

func TestEffectiveSwapDays_Zero(t *testing.T) {
	t.Parallel()
	eff := EffectiveSwapDays(time.Now(), 0)
	if eff != 0 {
		t.Fatalf("zero days: want 0, got %d", eff)
	}
}

func TestSwapCostDate_TripleWednesday(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	// Tuesday start, 2 day hold (spans Wed) → effective 4 swap charges
	tue := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	cost := m.SwapCostDate("sell", decF(1.0), tue, 2)
	// SwapShort=0.5, effective days=4, lots=1 → 0.5 * 1 * 4 = 2.0
	if !decClose(cost, decF(2.0)) {
		t.Fatalf("triple Wed swap: want 2.00, got %s", cost.String())
	}
}

func TestSnapshot_FrozenAtClock(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	var m antv1.CostSnapshotMap
	if err := proto.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal CostSnapshotMap: %v", err)
	}
	if _, ok := m.Entries["EURUSD"]; !ok {
		t.Fatal("snapshot should contain EURUSD")
	}
	if m.Entries["EURUSD"].Broker != "test_broker" {
		t.Fatal("snapshot should contain broker name")
	}
}

func TestSnapshotFromList(t *testing.T) {
	t.Parallel()
	models := []*CostModel{DefaultForexModel("EURUSD"), DefaultCryptoModel("BTCUSD")}
	data, err := SnapshotFromList("test_broker", models)
	if err != nil {
		t.Fatalf("SnapshotFromList: %v", err)
	}
	var m antv1.CostSnapshotMap
	if err := proto.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal CostSnapshotMap: %v", err)
	}
	if _, ok := m.Entries["BTCUSD"]; !ok {
		t.Fatal("snapshot should contain BTCUSD")
	}
}

func TestStaticEstimator(t *testing.T) {
	t.Parallel()
	m := DefaultForexModel("EURUSD")
	est := &StaticEstimator{Model: m}
	breakdown := est.Estimate(t.Context(), EstimateParams{
		Side: "buy", Lots: decF(1.0), Price: decF(1.0850), ContractSize: decF(100000),
	})
	if !breakdown.TotalCost.GreaterThan(decimal.Zero) {
		t.Fatalf("static estimator should return costs, got %s", breakdown.TotalCost.String())
	}
}

func TestMultiModelEstimator_KnownSymbol(t *testing.T) {
	t.Parallel()
	est := &MultiModelEstimator{
		Models:  map[string]*CostModel{"EURUSD": DefaultForexModel("EURUSD")},
		Default: nil,
	}
	breakdown := est.Estimate(t.Context(), EstimateParams{
		Symbol: "EURUSD", Side: "buy", Lots: decF(1.0), Price: decF(1.0850), ContractSize: decF(100000),
	})
	if !breakdown.TotalCost.GreaterThan(decimal.Zero) {
		t.Fatalf("known symbol should return costs, got %s", breakdown.TotalCost.String())
	}
}

func TestMultiModelEstimator_FallbackDefault(t *testing.T) {
	t.Parallel()
	est := &MultiModelEstimator{
		Models:  map[string]*CostModel{},
		Default: DefaultForexModel("EURUSD"),
	}
	breakdown := est.Estimate(t.Context(), EstimateParams{
		Symbol: "UNKNOWN", Side: "buy", Lots: decF(1.0), Price: decF(1.0850), ContractSize: decF(100000),
	})
	if !breakdown.TotalCost.GreaterThan(decimal.Zero) {
		t.Fatalf("fallback default should return costs, got %s", breakdown.TotalCost.String())
	}
}

func TestMultiModelEstimator_NoModel(t *testing.T) {
	t.Parallel()
	est := &MultiModelEstimator{
		Models:  map[string]*CostModel{},
		Default: nil,
	}
	breakdown := est.Estimate(t.Context(), EstimateParams{
		Symbol: "UNKNOWN", Side: "buy", Lots: decF(1.0), Price: decF(1.0850),
	})
	if !breakdown.TotalCost.Equal(decimal.Zero) {
		t.Fatalf("no model should return zero cost, got %s", breakdown.TotalCost.String())
	}
}

var _ = math.Abs
