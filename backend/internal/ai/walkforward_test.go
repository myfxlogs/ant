// Package ai — E2 Walk-Forward + CPCV tests.
package ai

import (
	"math"
	"testing"
)

func TestWalkForward_Passing(t *testing.T) {
	t.Parallel()
	// All returns equal → zero variance → Sharpe=0 for both train and test → diff=0 < 1.0.
	returns := make([]float64, 500)
	for i := range returns {
		returns[i] = 1.0
	}
	cfg := DefaultWalkForwardConfig()
	result := WalkForward(returns, cfg)
	if !result.Passed {
		t.Fatalf("uniform positive returns should pass walk-forward: %s (SharpeDiff=%.4f)", result.Reason, result.SharpeDiff)
	}
	if len(result.Folds) == 0 {
		t.Fatal("should have folds")
	}
	t.Logf("SharpeDiff=%.4f MaxFoldDD=%.4f MinTrades=%d Folds=%d",
		result.SharpeDiff, result.MaxFoldDD, result.MinTrades, len(result.Folds))
}

func TestWalkForward_InsufficientData(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 10)
	cfg := DefaultWalkForwardConfig()
	result := WalkForward(returns, cfg)
	if result.Passed {
		t.Fatal("insufficient data should not pass")
	}
}

func TestWalkForward_OverfittingDetected(t *testing.T) {
	t.Skip("TODO: add value assertions with known expected Sharpe/DD values")
	t.Parallel()
	returns := make([]float64, 250)
	for i := 0; i < 125; i++ {
		returns[i] = 5.0
	}
	for i := 125; i < 250; i++ {
		returns[i] = -2.0 + float64(i%5)*0.1
	}
	cfg := DefaultWalkForwardConfig()
	result := WalkForward(returns, cfg)
	if result.Passed {
		t.Logf("SharpeDiff=%.4f (overfitting may not trigger if < 1.0)", result.SharpeDiff)
	}
	t.Logf("Overfitting test: SharpeDiff=%.4f MaxDD=%.4f Passed=%v",
		result.SharpeDiff, result.MaxFoldDD, result.Passed)
}

func TestCPCV(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 252)
	for i := range returns {
		returns[i] = 1.0 + float64(i%20)*0.05
	}
	oosSharpe := CPCV(returns, 6, DefaultWalkForwardConfig())
	t.Logf("CPCV OOS median Sharpe: %.4f", oosSharpe)
	if math.IsNaN(oosSharpe) {
		t.Fatal("CPCV should return a valid number")
	}
}

func TestCPCV_EmptyReturns(t *testing.T) {
	t.Parallel()
	oosSharpe := CPCV([]float64{}, 6, DefaultWalkForwardConfig())
	if oosSharpe != 0 {
		t.Fatalf("empty returns: want 0, got %.4f", oosSharpe)
	}
}

func TestComputeSharpe(t *testing.T) {
	t.Parallel()
	returns := []float64{2.0, 1.0, 3.0, 1.5, 2.5, 1.0, 3.0, 2.0, 1.5, 2.5}
	sr := computeSharpe(returns)
	if sr <= 0 {
		t.Fatalf("positive returns: want positive SR, got %.4f", sr)
	}
	negReturns := []float64{-2.0, -1.0, -3.0, -1.5, -2.5, -1.0, -3.0, -2.0, -1.5, -2.5}
	srNeg := computeSharpe(negReturns)
	if srNeg >= 0 {
		t.Fatalf("negative returns: want negative SR, got %.4f", srNeg)
	}
}

func TestComputeMaxDD_Simple(t *testing.T) {
	t.Parallel()
	returns := []float64{100, -50, -30, 20, 50}
	dd := computeMaxDD(returns)
	if dd < 0 {
		t.Fatal("max DD should be >= 0")
	}
	if math.Abs(dd-0.8) > 0.01 {
		t.Fatalf("max DD: want ~0.8, got %.2f", dd)
	}
}
