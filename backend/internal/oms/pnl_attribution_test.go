package oms

import (
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/internal/costsvc"
)

func decFAttr(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

func decCloseAttr(a, b decimal.Decimal) bool {
	return a.Sub(b).Abs().LessThan(decFAttr(0.02))
}

func TestPnLAttribution_BuyProfitable(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0950), decFAttr(1.0), decFAttr(100000), decFAttr(1))

	// Gross P&L should be positive
	if a.GrossPnL.LessThanOrEqual(decimal.Zero) {
		t.Fatalf("profitable buy: GrossPnL=%s, want >0", a.GrossPnL.String())
	}
	// All cost dimensions should be non-zero
	if a.SpreadCost.LessThanOrEqual(decimal.Zero) {
		t.Errorf("SpreadCost=%s, want >0", a.SpreadCost.String())
	}
	if a.Commission.LessThanOrEqual(decimal.Zero) {
		t.Errorf("Commission=%s, want >0", a.Commission.String())
	}
	if a.SlippageCost.LessThanOrEqual(decimal.Zero) {
		t.Errorf("SlippageCost=%s, want >0", a.SlippageCost.String())
	}
	// Swap may be zero for short holding, but for 1 day it should exist
	if a.SwapCost.GreaterThanOrEqual(decimal.Zero) {
		t.Logf("SwapCost=%s (long swap is negative for EURUSD default)", a.SwapCost.String())
	}

	// Net = Gross - Execution - Holding
	net := a.NetPnL()
	expected := a.GrossPnL.Sub(a.ExecutionCost()).Sub(a.HoldingCost())
	if !decCloseAttr(net, expected) {
		t.Errorf("NetPnL=%s, want %s", net.String(), expected.String())
	}

	// Execution cost should be positive (costs reduce P&L)
	if a.ExecutionCost().LessThanOrEqual(decimal.Zero) {
		t.Errorf("ExecutionCost=%s, want >0", a.ExecutionCost().String())
	}

	// Validate identity
	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}

	t.Logf("Buy: Gross=%s Exec=%s Hold=%s Net=%s | Signal=%sbps Exec=%sbps Hold=%sbps Net=%sbps",
		a.GrossPnL.String(), a.ExecutionCost().String(), a.HoldingCost().String(), a.NetPnL().String(),
		a.SignalBps().String(), a.ExecutionBps().String(), a.HoldingBps().String(), a.NetBps().String())
}

func TestPnLAttribution_SellProfitable(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("sell", decFAttr(1.0950), decFAttr(1.0850), decFAttr(1.0), decFAttr(100000), decFAttr(1))

	if a.GrossPnL.LessThanOrEqual(decimal.Zero) {
		t.Fatalf("profitable sell: GrossPnL=%s, want >0", a.GrossPnL.String())
	}
	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}

	// Net should be less than Gross (costs eat into profit)
	if a.NetPnL().GreaterThanOrEqual(a.GrossPnL) {
		t.Errorf("Net %s >= Gross %s — costs not applied", a.NetPnL().String(), a.GrossPnL.String())
	}

	t.Logf("Sell: Gross=%s Exec=%s Hold=%s Net=%s",
		a.GrossPnL.String(), a.ExecutionCost().String(), a.HoldingCost().String(), a.NetPnL().String())
}

func TestPnLAttribution_LosingTrade(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("buy", decFAttr(1.0950), decFAttr(1.0850), decFAttr(1.0), decFAttr(100000), decFAttr(1))

	if a.GrossPnL.GreaterThanOrEqual(decimal.Zero) {
		t.Fatalf("losing trade: GrossPnL=%s, want <0", a.GrossPnL.String())
	}
	// Net loss should be larger (more negative) than gross
	if a.NetPnL().GreaterThanOrEqual(a.GrossPnL) {
		t.Errorf("Net %s >= Gross %s — costs should deepen the loss", a.NetPnL().String(), a.GrossPnL.String())
	}
	// Signal bps should be negative for losing trade
	if a.SignalBps().GreaterThanOrEqual(decimal.Zero) {
		t.Errorf("SignalBps=%s, want <0", a.SignalBps().String())
	}
	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}

	t.Logf("Loss: Gross=%s Exec=%s Hold=%s Net=%s | Signal=%sbps Net=%sbps",
		a.GrossPnL.String(), a.ExecutionCost().String(), a.HoldingCost().String(), a.NetPnL().String(),
		a.SignalBps().String(), a.NetBps().String())
}

func TestPnLAttribution_FlatTrade(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	// Open and close at same price
	a := attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0850), decFAttr(1.0), decFAttr(100000), decFAttr(0))

	if a.GrossPnL.Abs().GreaterThan(decFAttr(0.01)) {
		t.Errorf("flat trade: GrossPnL=%s, want 0", a.GrossPnL.String())
	}
	// Costs still exist even on flat trade
	if a.ExecutionCost().LessThanOrEqual(decimal.Zero) {
		t.Errorf("flat trade: ExecutionCost=%s, want >0", a.ExecutionCost().String())
	}
	// Net should be negative (costs without alpha)
	if a.NetPnL().GreaterThanOrEqual(decimal.Zero) {
		t.Errorf("flat trade: NetPnL=%s, want <0", a.NetPnL().String())
	}
	// Signal bps should be near zero
	if a.SignalBps().Abs().GreaterThan(decFAttr(0.1)) {
		t.Errorf("flat trade: SignalBps=%s, want ~0", a.SignalBps().String())
	}
	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}

	t.Logf("Flat: Gross=%s Exec=%s Hold=%s Net=%s",
		a.GrossPnL.String(), a.ExecutionCost().String(), a.HoldingCost().String(), a.NetPnL().String())
}

func TestPnLAttribution_ValidateIdentityHolds(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	cases := []struct {
		name string
		a    PnLAttribution
	}{
		{"profitable buy", attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0950), decFAttr(1.0), decFAttr(100000), decFAttr(1))},
		{"profitable sell", attr.Attribute("sell", decFAttr(1.0950), decFAttr(1.0850), decFAttr(1.0), decFAttr(100000), decFAttr(1))},
		{"losing buy", attr.Attribute("buy", decFAttr(1.0950), decFAttr(1.0850), decFAttr(1.0), decFAttr(100000), decFAttr(1))},
		{"flat trade", attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0850), decFAttr(1.0), decFAttr(100000), decFAttr(0))},
		{"micro size", attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0950), decFAttr(0.01), decFAttr(100000), decFAttr(1))},
		{"long hold", attr.Attribute("sell", decFAttr(1.0850), decFAttr(1.0900), decFAttr(2.0), decFAttr(100000), decFAttr(30))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.a.Validate(); err != nil {
				t.Errorf("Validate failed: %v", err)
			}
		})
	}
}

func TestPnLAttribution_AddAggregation(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a1 := attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0900), decFAttr(0.5), decFAttr(100000), decFAttr(1))
	a2 := attr.Attribute("buy", decFAttr(1.0900), decFAttr(1.0950), decFAttr(0.5), decFAttr(100000), decFAttr(1))

	combined := a1.Add(a2)

	if !decCloseAttr(combined.GrossPnL, a1.GrossPnL.Add(a2.GrossPnL)) {
		t.Errorf("GrossPnL: combined=%s, sum=%s", combined.GrossPnL.String(), a1.GrossPnL.Add(a2.GrossPnL).String())
	}
	if !decCloseAttr(combined.Commission, a1.Commission.Add(a2.Commission)) {
		t.Errorf("Commission: combined=%s, sum=%s", combined.Commission.String(), a1.Commission.Add(a2.Commission).String())
	}
	if !decCloseAttr(combined.SlippageCost, a1.SlippageCost.Add(a2.SlippageCost)) {
		t.Errorf("SlippageCost: combined=%s, sum=%s", combined.SlippageCost.String(), a1.SlippageCost.Add(a2.SlippageCost).String())
	}
	if !decCloseAttr(combined.SpreadCost, a1.SpreadCost.Add(a2.SpreadCost)) {
		t.Errorf("SpreadCost: combined=%s, sum=%s", combined.SpreadCost.String(), a1.SpreadCost.Add(a2.SpreadCost).String())
	}

	// Validate the combined result
	if err := combined.Validate(); err != nil {
		t.Errorf("combined Validate failed: %v", err)
	}

	t.Logf("Combined: Gross=%s Net=%s", combined.GrossPnL.String(), combined.NetPnL().String())
}

func TestPnLAttribution_ThreeDimensionsIndependent(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0950), decFAttr(1.0), decFAttr(100000), decFAttr(5))

	// Each dimension must be independently non-zero for a typical trade
	dims := map[string]decimal.Decimal{
		"Signal":    a.SignalPnL(),
		"Execution": a.ExecutionCost(),
		"Holding":   a.HoldingCost(),
	}
	for name, val := range dims {
		if val.Abs().LessThan(decFAttr(0.001)) {
			t.Errorf("dimension %s is zero — each dimension should be independently measurable", name)
		}
	}

	// Holding cost must grow with holding days
	a1 := attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0950), decFAttr(1.0), decFAttr(100000), decFAttr(1))
	a5 := attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0950), decFAttr(1.0), decFAttr(100000), decFAttr(5))

	if decCloseAttr(a1.HoldingCost(), a5.HoldingCost()) {
		t.Errorf("HoldingCost should differ by day count: day1=%s day5=%s", a1.HoldingCost().String(), a5.HoldingCost().String())
	}

	// Execution costs should be independent of holding days
	if !decCloseAttr(a1.ExecutionCost(), a5.ExecutionCost()) {
		t.Errorf("ExecutionCost should be independent of holding: day1=%s day5=%s",
			a1.ExecutionCost().String(), a5.ExecutionCost().String())
	}

	t.Logf("Signal=%s Exec=%s Hold=%s Net=%s",
		a.SignalPnL().String(), a.ExecutionCost().String(), a.HoldingCost().String(), a.NetPnL().String())
}

func TestPnLAttribution_SwapScalesWithHoldingDays(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	var prevSwap decimal.Decimal
	for _, days := range []float64{0, 1, 3, 7} {
		a := attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0950), decFAttr(1.0), decFAttr(100000), decFAttr(days))
		if days > 0 {
			// Swap cost should grow (in absolute value) with more days
			if a.SwapCost.Abs().LessThanOrEqual(prevSwap.Abs()) && !prevSwap.Equal(decimal.Zero) {
				t.Errorf("day %.0f: |SwapCost|=%s should exceed previous |%s|", days, a.SwapCost.Abs().String(), prevSwap.Abs().String())
			}
		}
		prevSwap = a.SwapCost
	}
}

func TestPnLAttribution_ZeroHoldingDays(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0900), decFAttr(1.0), decFAttr(100000), decFAttr(0))

	if a.SwapCost.Abs().GreaterThan(decFAttr(0.01)) {
		t.Errorf("zero holding: SwapCost=%s, want 0", a.SwapCost.String())
	}
	// Execution and commission still apply (entry + exit)
	if a.ExecutionCost().LessThanOrEqual(decimal.Zero) {
		t.Errorf("Execution should be non-zero even with zero holding")
	}
	if a.Commission.LessThanOrEqual(decimal.Zero) {
		t.Errorf("Commission should be non-zero (entry + exit)")
	}
	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}
}

func TestPnLAttribution_BpsConsistency(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0950), decFAttr(1.0), decFAttr(100000), decFAttr(1))

	// NetBps = SignalBps - ExecutionBps - HoldingBps
	expected := a.SignalBps().Sub(a.ExecutionBps()).Sub(a.HoldingBps())
	if !decCloseAttr(a.NetBps(), expected) {
		t.Errorf("NetBps=%s, Signal-Exec-Hold=%s", a.NetBps().String(), expected.String())
	}
}

func TestPnLAttribution_SmallSize(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	// 0.01 lot micro trade
	a := attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0950), decFAttr(0.01), decFAttr(100000), decFAttr(1))

	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed on micro trade: %v", err)
	}
	// Costs should be proportionally smaller
	if a.ExecutionCost().LessThanOrEqual(decimal.Zero) {
		t.Errorf("micro trade: ExecutionCost=%s, want >0", a.ExecutionCost().String())
	}
	t.Logf("Micro: Gross=%s Exec=%s Hold=%s Net=%s | NetBps=%s",
		a.GrossPnL.String(), a.ExecutionCost().String(), a.HoldingCost().String(), a.NetPnL().String(), a.NetBps().String())
}

func TestPnLAttribution_ValidateAllCostsNonNegative(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0950), decFAttr(1.0), decFAttr(100000), decFAttr(1))

	// Spread, commission, slippage must never be negative
	for name, val := range map[string]decimal.Decimal{
		"SpreadCost":   a.SpreadCost,
		"Commission":   a.Commission,
		"SlippageCost": a.SlippageCost,
	} {
		if val.LessThan(decimal.Zero) {
			t.Errorf("%s=%s, costs must be non-negative", name, val.String())
		}
	}
	// Swap and Funding can be negative (you receive it)
	// GrossPnL can be negative (loss)
}

func TestPnLAttribution_LongHoldingSwap(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	// 30-day hold — swap cost should dominate holding dimension
	a := attr.Attribute("sell", decFAttr(1.0850), decFAttr(1.0900), decFAttr(1.0), decFAttr(100000), decFAttr(30))

	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}
	// Holding cost for 30 days should be significant
	if a.HoldingCost().Abs().LessThan(decFAttr(1.0)) {
		t.Errorf("30-day hold: HoldingCost=%s, want >1.0", a.HoldingCost().String())
	}
	t.Logf("30d hold: Gross=%s Exec=%s Hold=%s (swap=%s) Net=%s",
		a.GrossPnL.String(), a.ExecutionCost().String(), a.HoldingCost().String(), a.SwapCost.String(), a.NetPnL().String())
}

func TestPnLAttribution_SideIsPreserved(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	aBuy := attr.Attribute("buy", decFAttr(1.0850), decFAttr(1.0950), decFAttr(1.0), decFAttr(100000), decFAttr(1))
	if aBuy.Side != "buy" {
		t.Errorf("Side = %s, want buy", aBuy.Side)
	}

	aSell := attr.Attribute("sell", decFAttr(1.0950), decFAttr(1.0850), decFAttr(1.0), decFAttr(100000), decFAttr(1))
	if aSell.Side != "sell" {
		t.Errorf("Side = %s, want sell", aSell.Side)
	}
}

func TestPnLAttribution_CostModelAccessor(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	retrieved := attr.CostModel()
	if retrieved == nil {
		t.Fatal("CostModel() returned nil")
	}
	if retrieved.Symbol != cm.Symbol {
		t.Errorf("Symbol = %s, want %s", retrieved.Symbol, cm.Symbol)
	}
}

func TestPnLAttribution_ValidateErrorFormat(t *testing.T) {
	t.Parallel()
	// Manually construct an attribution to verify Validate passes
	// and the arithmetic identity holds.
	a := PnLAttribution{
		GrossPnL:     decFAttr(100.0),
		SlippageCost: decFAttr(5.0),
		SpreadCost:   decFAttr(5.0),
		Commission:   decFAttr(7.0),
		SwapCost:     decFAttr(-3.5),
		Notional:     decFAttr(100000),
		Side:         "buy",
	}
	// Net = 100 - 10 - 3.5 = 86.5
	expectedNet := decFAttr(86.5)
	if !decCloseAttr(a.NetPnL(), expectedNet) {
		t.Errorf("NetPnL() = %s, want %s", a.NetPnL().String(), expectedNet.String())
	}
	if err := a.Validate(); err != nil {
		t.Errorf("Validate should pass for consistent attribution: %v", err)
	}
}
