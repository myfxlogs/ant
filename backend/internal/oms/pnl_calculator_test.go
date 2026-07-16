package oms

import (
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/internal/costsvc"
)

func decFCalc(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

func decCloseCalc(a, b decimal.Decimal) bool {
	return a.Sub(b).Abs().LessThan(decFCalc(0.01))
}

func TestPnLCalculator_Buy_Profitable(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	calc := NewPnLCalculator(fm)

	result := calc.Calculate("buy", decFCalc(1.0850), decFCalc(1.0950), decFCalc(1.0), decFCalc(100000), decFCalc(1))

	// Gross P&L: (1.0950 - 1.0850) * 100000 / 1.0850 = 921.66
	if result.GrossPnL.LessThanOrEqual(decimal.Zero) {
		t.Fatalf("profitable buy should have positive gross P&L, got %s", result.GrossPnL.String())
	}
	// Net should be lower than gross due to costs.
	if result.NetPnL.GreaterThanOrEqual(result.GrossPnL) {
		t.Fatalf("net P&L %s should be < gross P&L %s", result.NetPnL.String(), result.GrossPnL.String())
	}
	// Both must be reported.
	if result.GrossPnL.Equal(decimal.Zero) || result.NetPnL.Equal(decimal.Zero) {
		t.Fatal("dual-track: both gross and net P&L must be non-zero")
	}

	t.Logf("Buy trade: gross=%s net=%s (costs: spread=%s comm=%s swap=%s slip=%s)",
		result.GrossPnL.String(), result.NetPnL.String(), result.SpreadCost.String(), result.Commission.String(), result.SwapCost.String(), result.SlippageCost.String())
}

func TestPnLCalculator_Sell_Profitable(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	calc := NewPnLCalculator(fm)

	// Sell at 1.0950, buy back at 1.0850 = profit
	result := calc.Calculate("sell", decFCalc(1.0950), decFCalc(1.0850), decFCalc(1.0), decFCalc(100000), decFCalc(1))

	if result.GrossPnL.LessThanOrEqual(decimal.Zero) {
		t.Fatalf("profitable sell should have positive gross P&L, got %s", result.GrossPnL.String())
	}
	if result.NetPnL.GreaterThanOrEqual(result.GrossPnL) {
		t.Fatalf("net P&L %s should be < gross P&L %s", result.NetPnL.String(), result.GrossPnL.String())
	}
}

func TestPnLCalculator_Losing(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	calc := NewPnLCalculator(fm)

	result := calc.Calculate("buy", decFCalc(1.0950), decFCalc(1.0850), decFCalc(1.0), decFCalc(100000), decFCalc(1))

	if result.GrossPnL.GreaterThanOrEqual(decimal.Zero) {
		t.Fatalf("losing trade should have negative gross P&L, got %s", result.GrossPnL.String())
	}
	// Net loss should be larger (more negative) than gross due to costs.
	if result.NetPnL.GreaterThanOrEqual(result.GrossPnL) {
		t.Fatalf("net should be more negative than gross for losing trade")
	}
}

func TestPnLCalculator_ZeroHoldingDays(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	calc := NewPnLCalculator(fm)

	result := calc.Calculate("buy", decFCalc(1.0850), decFCalc(1.0900), decFCalc(1.0), decFCalc(100000), decFCalc(0))

	if result.SwapCost.Abs().GreaterThan(decFCalc(0.01)) {
		t.Fatalf("zero holding days: swap cost should be 0, got %s", result.SwapCost.String())
	}
	if result.GrossPnL.LessThanOrEqual(decimal.Zero) {
		t.Fatal("should be profitable")
	}
}

func TestPnLCalculator_GrossNetSeparation(t *testing.T) {
	t.Parallel()
	// Verify that Net = Gross - Spread - Commission - Swap - Slippage.
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	calc := NewPnLCalculator(fm)

	result := calc.Calculate("buy", decFCalc(1.0850), decFCalc(1.0900), decFCalc(0.5), decFCalc(100000), decFCalc(2))

	calculatedNet := result.GrossPnL.Sub(result.SpreadCost).Sub(result.Commission).Sub(result.SwapCost).Sub(result.SlippageCost)
	if !decCloseCalc(calculatedNet, result.NetPnL) {
		t.Fatalf("Net = Gross - costs: calculated=%s actual=%s", calculatedNet.String(), result.NetPnL.String())
	}
}

func TestDualTrackPnL(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	calc := NewPnLCalculator(fm)

	result := calc.Calculate("buy", decFCalc(1.0850), decFCalc(1.0950), decFCalc(1.0), decFCalc(100000), decFCalc(1))

	// Both Gross and Net P&L must be present.
	if result.GrossPnL.Equal(result.NetPnL) {
		t.Fatal("Gross P&L must differ from Net P&L due to trading costs")
	}
	// Net P&L = Gross - all costs
	expectedNet := result.GrossPnL.Sub(result.SpreadCost).Sub(result.Commission).Sub(result.SwapCost).Sub(result.SlippageCost)
	if !decCloseCalc(expectedNet, result.NetPnL) {
		t.Fatalf("Net P&L mismatch: expected %s got %s", expectedNet.String(), result.NetPnL.String())
	}
	t.Logf("Gross=%s Net=%s", result.GrossPnL.String(), result.NetPnL.String())
}
