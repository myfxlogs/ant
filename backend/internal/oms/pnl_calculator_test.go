package oms

import (
	"math"
	"testing"

	"anttrader/internal/costsvc"
)

func TestPnLCalculator_Buy_Profitable(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	calc := NewPnLCalculator(fm)

	result := calc.Calculate("buy", 1.0850, 1.0950, 1.0, 100000, 1)

	// Gross P&L: (1.0950 - 1.0850) * 100000 / 1.0850 = 921.66
	if result.GrossPnL <= 0 {
		t.Fatalf("profitable buy should have positive gross P&L, got %.2f", result.GrossPnL)
	}
	// Net should be lower than gross due to costs.
	if result.NetPnL >= result.GrossPnL {
		t.Fatalf("net P&L %.2f should be < gross P&L %.2f", result.NetPnL, result.GrossPnL)
	}
	// Both must be reported.
	if result.GrossPnL == 0 || result.NetPnL == 0 {
		t.Fatal("dual-track: both gross and net P&L must be non-zero")
	}

	t.Logf("Buy trade: gross=%.2f net=%.2f (costs: spread=%.2f comm=%.2f swap=%.2f slip=%.2f)",
		result.GrossPnL, result.NetPnL, result.SpreadCost, result.Commission, result.SwapCost, result.SlippageCost)
}

func TestPnLCalculator_Sell_Profitable(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	calc := NewPnLCalculator(fm)

	// Sell at 1.0950, buy back at 1.0850 = profit
	result := calc.Calculate("sell", 1.0950, 1.0850, 1.0, 100000, 1)

	if result.GrossPnL <= 0 {
		t.Fatalf("profitable sell should have positive gross P&L, got %.2f", result.GrossPnL)
	}
	if result.NetPnL >= result.GrossPnL {
		t.Fatalf("net P&L %.2f should be < gross P&L %.2f", result.NetPnL, result.GrossPnL)
	}
}

func TestPnLCalculator_Losing(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	calc := NewPnLCalculator(fm)

	result := calc.Calculate("buy", 1.0950, 1.0850, 1.0, 100000, 1)

	if result.GrossPnL >= 0 {
		t.Fatalf("losing trade should have negative gross P&L, got %.2f", result.GrossPnL)
	}
	// Net loss should be larger (more negative) than gross due to costs.
	if result.NetPnL >= result.GrossPnL {
		t.Fatalf("net should be more negative than gross for losing trade")
	}
}

func TestPnLCalculator_ZeroHoldingDays(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	calc := NewPnLCalculator(fm)

	result := calc.Calculate("buy", 1.0850, 1.0900, 1.0, 100000, 0)

	if math.Abs(result.SwapCost) > 0.01 {
		t.Fatalf("zero holding days: swap cost should be 0, got %.4f", result.SwapCost)
	}
	if result.GrossPnL <= 0 {
		t.Fatal("should be profitable")
	}
}

func TestPnLCalculator_GrossNetSeparation(t *testing.T) {
	t.Parallel()
	// Verify that Net = Gross - Spread - Commission - Swap - Slippage.
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	calc := NewPnLCalculator(fm)

	result := calc.Calculate("buy", 1.0850, 1.0900, 0.5, 100000, 2)

	calculatedNet := result.GrossPnL - result.SpreadCost - result.Commission - result.SwapCost - result.SlippageCost
	if math.Abs(calculatedNet-result.NetPnL) > 0.01 {
		t.Fatalf("Net = Gross - costs: calculated=%.2f actual=%.2f", calculatedNet, result.NetPnL)
	}
}

func TestDualTrackPnL(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)
	calc := NewPnLCalculator(fm)

	result := calc.Calculate("buy", 1.0850, 1.0950, 1.0, 100000, 1)

	// Both Gross and Net P&L must be present.
	if result.GrossPnL == result.NetPnL {
		t.Fatal("Gross P&L must differ from Net P&L due to trading costs")
	}
	// Net P&L = Gross - all costs
	expectedNet := result.GrossPnL - result.SpreadCost - result.Commission - result.SwapCost - result.SlippageCost
	if math.Abs(expectedNet-result.NetPnL) > 0.01 {
		t.Fatalf("Net P&L mismatch: expected %.2f got %.2f", expectedNet, result.NetPnL)
	}
	t.Logf("Gross=%.2f Net=%.2f", result.GrossPnL, result.NetPnL)
}
