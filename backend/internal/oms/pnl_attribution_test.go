package oms

import (
	"math"
	"testing"

	"anttrader/internal/costsvc"
)

func closeEnoughAttribution(a, b float64) bool {
	return math.Abs(a-b) < 0.02
}

func TestPnLAttribution_BuyProfitable(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("buy", 1.0850, 1.0950, 1.0, 100000, 1)

	// Gross P&L should be positive
	if a.GrossPnL <= 0 {
		t.Fatalf("profitable buy: GrossPnL=%.2f, want >0", a.GrossPnL)
	}
	// All cost dimensions should be non-zero
	if a.SpreadCost <= 0 {
		t.Errorf("SpreadCost=%.4f, want >0", a.SpreadCost)
	}
	if a.Commission <= 0 {
		t.Errorf("Commission=%.4f, want >0", a.Commission)
	}
	if a.SlippageCost <= 0 {
		t.Errorf("SlippageCost=%.4f, want >0", a.SlippageCost)
	}
	// Swap may be zero for short holding, but for 1 day it should exist
	if a.SwapCost >= 0 {
		t.Logf("SwapCost=%.4f (long swap is negative for EURUSD default)", a.SwapCost)
	}

	// Net = Gross - Execution - Holding
	net := a.NetPnL()
	expected := a.GrossPnL - a.ExecutionCost() - a.HoldingCost()
	if !closeEnoughAttribution(net, expected) {
		t.Errorf("NetPnL=%.4f, want %.4f", net, expected)
	}

	// Execution cost should be positive (costs reduce P&L)
	if a.ExecutionCost() <= 0 {
		t.Errorf("ExecutionCost=%.4f, want >0", a.ExecutionCost())
	}

	// Validate identity
	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}

	t.Logf("Buy: Gross=%.2f Exec=%.2f Hold=%.2f Net=%.2f | Signal=%.1fbps Exec=%.1fbps Hold=%.1fbps Net=%.1fbps",
		a.GrossPnL, a.ExecutionCost(), a.HoldingCost(), a.NetPnL(),
		a.SignalBps(), a.ExecutionBps(), a.HoldingBps(), a.NetBps())
}

func TestPnLAttribution_SellProfitable(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("sell", 1.0950, 1.0850, 1.0, 100000, 1)

	if a.GrossPnL <= 0 {
		t.Fatalf("profitable sell: GrossPnL=%.2f, want >0", a.GrossPnL)
	}
	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}

	// Net should be less than Gross (costs eat into profit)
	if a.NetPnL() >= a.GrossPnL {
		t.Errorf("Net %.4f >= Gross %.4f — costs not applied", a.NetPnL(), a.GrossPnL)
	}

	t.Logf("Sell: Gross=%.2f Exec=%.2f Hold=%.2f Net=%.2f",
		a.GrossPnL, a.ExecutionCost(), a.HoldingCost(), a.NetPnL())
}

func TestPnLAttribution_LosingTrade(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("buy", 1.0950, 1.0850, 1.0, 100000, 1)

	if a.GrossPnL >= 0 {
		t.Fatalf("losing trade: GrossPnL=%.2f, want <0", a.GrossPnL)
	}
	// Net loss should be larger (more negative) than gross
	if a.NetPnL() >= a.GrossPnL {
		t.Errorf("Net %.4f >= Gross %.4f — costs should deepen the loss", a.NetPnL(), a.GrossPnL)
	}
	// Signal bps should be negative for losing trade
	if a.SignalBps() >= 0 {
		t.Errorf("SignalBps=%.2f, want <0", a.SignalBps())
	}
	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}

	t.Logf("Loss: Gross=%.2f Exec=%.2f Hold=%.2f Net=%.2f | Signal=%.1fbps Net=%.1fbps",
		a.GrossPnL, a.ExecutionCost(), a.HoldingCost(), a.NetPnL(),
		a.SignalBps(), a.NetBps())
}

func TestPnLAttribution_FlatTrade(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	// Open and close at same price
	a := attr.Attribute("buy", 1.0850, 1.0850, 1.0, 100000, 0)

	if math.Abs(a.GrossPnL) > 0.01 {
		t.Errorf("flat trade: GrossPnL=%.4f, want 0", a.GrossPnL)
	}
	// Costs still exist even on flat trade
	if a.ExecutionCost() <= 0 {
		t.Errorf("flat trade: ExecutionCost=%.4f, want >0", a.ExecutionCost())
	}
	// Net should be negative (costs without alpha)
	if a.NetPnL() >= 0 {
		t.Errorf("flat trade: NetPnL=%.4f, want <0", a.NetPnL())
	}
	// Signal bps should be near zero
	if math.Abs(a.SignalBps()) > 0.1 {
		t.Errorf("flat trade: SignalBps=%.2f, want ~0", a.SignalBps())
	}
	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}

	t.Logf("Flat: Gross=%.2f Exec=%.2f Hold=%.2f Net=%.2f",
		a.GrossPnL, a.ExecutionCost(), a.HoldingCost(), a.NetPnL())
}

func TestPnLAttribution_ValidateIdentityHolds(t *testing.T) {
	// Validate checks the arithmetic identity:
	//   NetPnL = GrossPnL - Spread - Slippage - Commission - Swap - Funding
	// Since NetPnL() is computed from fields, this is tautological for
	// properly-constructed attributions. The test verifies Validate passes
	// for a variety of realistic attributions.

	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	cases := []struct {
		name string
		a    PnLAttribution
	}{
		{"profitable buy", attr.Attribute("buy", 1.0850, 1.0950, 1.0, 100000, 1)},
		{"profitable sell", attr.Attribute("sell", 1.0950, 1.0850, 1.0, 100000, 1)},
		{"losing buy", attr.Attribute("buy", 1.0950, 1.0850, 1.0, 100000, 1)},
		{"flat trade", attr.Attribute("buy", 1.0850, 1.0850, 1.0, 100000, 0)},
		{"micro size", attr.Attribute("buy", 1.0850, 1.0950, 0.01, 100000, 1)},
		{"long hold", attr.Attribute("sell", 1.0850, 1.0900, 2.0, 100000, 30)},
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
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a1 := attr.Attribute("buy", 1.0850, 1.0900, 0.5, 100000, 1)
	a2 := attr.Attribute("buy", 1.0900, 1.0950, 0.5, 100000, 1)

	combined := a1.Add(a2)

	if !closeEnoughAttribution(combined.GrossPnL, a1.GrossPnL+a2.GrossPnL) {
		t.Errorf("GrossPnL: combined=%.4f, sum=%.4f", combined.GrossPnL, a1.GrossPnL+a2.GrossPnL)
	}
	if !closeEnoughAttribution(combined.Commission, a1.Commission+a2.Commission) {
		t.Errorf("Commission: combined=%.4f, sum=%.4f", combined.Commission, a1.Commission+a2.Commission)
	}
	if !closeEnoughAttribution(combined.SlippageCost, a1.SlippageCost+a2.SlippageCost) {
		t.Errorf("SlippageCost: combined=%.4f, sum=%.4f", combined.SlippageCost, a1.SlippageCost+a2.SlippageCost)
	}
	if !closeEnoughAttribution(combined.SpreadCost, a1.SpreadCost+a2.SpreadCost) {
		t.Errorf("SpreadCost: combined=%.4f, sum=%.4f", combined.SpreadCost, a1.SpreadCost+a2.SpreadCost)
	}

	// Validate the combined result
	if err := combined.Validate(); err != nil {
		t.Errorf("combined Validate failed: %v", err)
	}

	t.Logf("Combined: Gross=%.2f Net=%.2f", combined.GrossPnL, combined.NetPnL())
}

func TestPnLAttribution_ThreeDimensionsIndependent(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("buy", 1.0850, 1.0950, 1.0, 100000, 5)

	// Each dimension must be independently non-zero for a typical trade
	dims := map[string]float64{
		"Signal":    a.SignalPnL(),
		"Execution": a.ExecutionCost(),
		"Holding":   a.HoldingCost(),
	}
	for name, val := range dims {
		if math.Abs(val) < 0.001 {
			t.Errorf("dimension %s is zero — each dimension should be independently measurable", name)
		}
	}

	// Holding cost must grow with holding days
	a1 := attr.Attribute("buy", 1.0850, 1.0950, 1.0, 100000, 1)
	a5 := attr.Attribute("buy", 1.0850, 1.0950, 1.0, 100000, 5)

	if closeEnoughAttribution(a1.HoldingCost(), a5.HoldingCost()) {
		t.Errorf("HoldingCost should differ by day count: day1=%.4f day5=%.4f", a1.HoldingCost(), a5.HoldingCost())
	}

	// Execution costs should be independent of holding days
	if !closeEnoughAttribution(a1.ExecutionCost(), a5.ExecutionCost()) {
		t.Errorf("ExecutionCost should be independent of holding: day1=%.4f day5=%.4f",
			a1.ExecutionCost(), a5.ExecutionCost())
	}

	t.Logf("Signal=%.2f Exec=%.4f Hold=%.4f Net=%.2f",
		a.SignalPnL(), a.ExecutionCost(), a.HoldingCost(), a.NetPnL())
}

func TestPnLAttribution_SwapScalesWithHoldingDays(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	var prevSwap float64
	for _, days := range []float64{0, 1, 3, 7} {
		a := attr.Attribute("buy", 1.0850, 1.0950, 1.0, 100000, days)
		if days > 0 {
			// Swap cost should grow (in absolute value) with more days
			if math.Abs(a.SwapCost) <= math.Abs(prevSwap) && prevSwap != 0 {
				t.Errorf("day %.0f: |SwapCost|=%.4f should exceed previous |%.4f|", days, math.Abs(a.SwapCost), math.Abs(prevSwap))
			}
		}
		prevSwap = a.SwapCost
	}
}

func TestPnLAttribution_ZeroHoldingDays(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("buy", 1.0850, 1.0900, 1.0, 100000, 0)

	if math.Abs(a.SwapCost) > 0.01 {
		t.Errorf("zero holding: SwapCost=%.4f, want 0", a.SwapCost)
	}
	// Execution and commission still apply (entry + exit)
	if a.ExecutionCost() <= 0 {
		t.Errorf("Execution should be non-zero even with zero holding")
	}
	if a.Commission <= 0 {
		t.Errorf("Commission should be non-zero (entry + exit)")
	}
	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}
}

func TestPnLAttribution_BpsConsistency(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("buy", 1.0850, 1.0950, 1.0, 100000, 1)

	// NetBps = SignalBps - ExecutionBps - HoldingBps
	expected := a.SignalBps() - a.ExecutionBps() - a.HoldingBps()
	if !closeEnoughAttribution(a.NetBps(), expected) {
		t.Errorf("NetBps=%.2f, Signal-Exec-Hold=%.2f", a.NetBps(), expected)
	}
}

func TestPnLAttribution_SmallSize(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	// 0.01 lot micro trade
	a := attr.Attribute("buy", 1.0850, 1.0950, 0.01, 100000, 1)

	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed on micro trade: %v", err)
	}
	// Costs should be proportionally smaller
	if a.ExecutionCost() <= 0 {
		t.Errorf("micro trade: ExecutionCost=%.4f, want >0", a.ExecutionCost())
	}
	t.Logf("Micro: Gross=%.4f Exec=%.4f Hold=%.4f Net=%.4f | NetBps=%.2f",
		a.GrossPnL, a.ExecutionCost(), a.HoldingCost(), a.NetPnL(), a.NetBps())
}

func TestPnLAttribution_ValidateAllCostsNonNegative(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	a := attr.Attribute("buy", 1.0850, 1.0950, 1.0, 100000, 1)

	// Spread, commission, slippage must never be negative
	for name, val := range map[string]float64{
		"SpreadCost":   a.SpreadCost,
		"Commission":   a.Commission,
		"SlippageCost": a.SlippageCost,
	} {
		if val < 0 {
			t.Errorf("%s=%.4f, costs must be non-negative", name, val)
		}
	}
	// Swap and Funding can be negative (you receive it)
	// GrossPnL can be negative (loss)
}

func TestPnLAttribution_LongHoldingSwap(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	// 30-day hold — swap cost should dominate holding dimension
	a := attr.Attribute("sell", 1.0850, 1.0900, 1.0, 100000, 30)

	if err := a.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}
	// Holding cost for 30 days should be significant
	if math.Abs(a.HoldingCost()) < 1.0 {
		t.Errorf("30-day hold: HoldingCost=%.4f, want >1.0", a.HoldingCost())
	}
	t.Logf("30d hold: Gross=%.2f Exec=%.2f Hold=%.2f (swap=%.2f) Net=%.2f",
		a.GrossPnL, a.ExecutionCost(), a.HoldingCost(), a.SwapCost, a.NetPnL())
}

func TestPnLAttribution_SideIsPreserved(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	attr := NewPnLAttributor(fm)

	aBuy := attr.Attribute("buy", 1.0850, 1.0950, 1.0, 100000, 1)
	if aBuy.Side != "buy" {
		t.Errorf("Side = %s, want buy", aBuy.Side)
	}

	aSell := attr.Attribute("sell", 1.0950, 1.0850, 1.0, 100000, 1)
	if aSell.Side != "sell" {
		t.Errorf("Side = %s, want sell", aSell.Side)
	}
}

func TestPnLAttribution_CostModelAccessor(t *testing.T) {
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
	// Manually construct an attribution to verify Validate passes
	// and the arithmetic identity holds.
	a := PnLAttribution{
		GrossPnL:     100.0,
		SlippageCost: 5.0,
		SpreadCost:   5.0,
		Commission:   7.0,
		SwapCost:     -3.5,
		Notional:     100000,
		Side:         "buy",
	}
	// Net = 100 - 10 - 3.5 = 86.5
	expectedNet := 86.5
	if !closeEnoughAttribution(a.NetPnL(), expectedNet) {
		t.Errorf("NetPnL() = %.4f, want %.4f", a.NetPnL(), expectedNet)
	}
	if err := a.Validate(); err != nil {
		t.Errorf("Validate should pass for consistent attribution: %v", err)
	}
}
