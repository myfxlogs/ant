// Package ai — E3 Deflated Sharpe Ratio tests.
package ai

import "testing"

func TestDeflatedSharpe_PositiveEdge(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 200)
	for i := range returns {
		returns[i] = 0.002 + float64(i%10)*0.0003
	}
	moments := ComputeReturnMoments(returns)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = 1
	dsr, passed := DeflatedSharpe(moments, cfg)
	t.Logf("SR=%.4f DSR=%.4f Passed=%v T=%d", moments.SharpeRatio, dsr, passed, moments.NumObservations)
	if !passed {
		t.Fatal("positive edge with N=1 and T=200 should pass")
	}
}

func TestDeflatedSharpe_N100_Deflates(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 200)
	for i := range returns {
		returns[i] = 0.001 + float64(i%10)*0.0002
	}
	moments := ComputeReturnMoments(returns)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = 100
	dsr, _ := DeflatedSharpe(moments, cfg)
	t.Logf("N=100: SR=%.4f DSR=%.4f T=%d", moments.SharpeRatio, dsr, moments.NumObservations)
	if dsr >= moments.SharpeRatio {
		t.Fatal("DSR should be lower than raw SR when N > 1")
	}
}

func TestDeflatedSharpe_ZeroSharpe(t *testing.T) {
	t.Parallel()
	moments := ReturnMoments{SharpeRatio: 0}
	dsr, passed := DeflatedSharpe(moments, DefaultDeflatedSharpeConfig())
	if dsr != 0 || passed {
		t.Fatalf("zero SR: want dsr=0 passed=false, got dsr=%.4f passed=%v", dsr, passed)
	}
}

func TestDeflatedSharpe_NegativeSharpe(t *testing.T) {
	t.Parallel()
	moments := ReturnMoments{SharpeRatio: -0.5}
	dsr, passed := DeflatedSharpe(moments, DefaultDeflatedSharpeConfig())
	if dsr != 0 || passed {
		t.Fatalf("negative SR: want dsr=0 passed=false, got dsr=%.4f passed=%v", dsr, passed)
	}
}

func TestDeflatedSharpe_N1_unchanged(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 200)
	for i := range returns {
		returns[i] = 0.002 + float64(i%10)*0.0003
	}
	moments := ComputeReturnMoments(returns)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = 1
	dsr, _ := DeflatedSharpe(moments, cfg)
	t.Logf("N=1: SR=%.4f DSR=%.4f T=%d", moments.SharpeRatio, dsr, moments.NumObservations)
	if dsr <= 0 {
		t.Fatal("N=1: DSR should be positive for a positive-edge strategy")
	}
	if dsr > moments.SharpeRatio {
		t.Fatalf("N=1: DSR %.4f should not exceed SR %.4f (deflation only reduces)", dsr, moments.SharpeRatio)
	}
}

func TestComputeReturnMoments(t *testing.T) {
	t.Parallel()
	returns := []float64{0.01, -0.005, 0.02, -0.01, 0.015}
	moments := ComputeReturnMoments(returns)
	if moments.SharpeRatio == 0 {
		t.Fatal("should have non-zero Sharpe")
	}
	t.Logf("Moments: mean=%.4f std=%.4f skew=%.4f kurt=%.4f SR=%.4f",
		moments.Mean, moments.StdDev, moments.Skewness, moments.ExcessKurtosis, moments.SharpeRatio)
}

func TestDeflatedSharpeFromReturns(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 200)
	for i := range returns {
		returns[i] = 0.002 + float64(i%10)*0.0003
	}
	dsr, passed := DeflatedSharpeFromReturns(returns, 1)
	t.Logf("DSR from returns: %.4f passed=%v T=%d", dsr, passed, len(returns))
	if dsr <= 0 {
		t.Fatal("positive returns should yield positive DSR")
	}
}

// TestDeflatedSharpe_NoHardCutoffAtN6 verifies the critical fix: the old formula
// had a hard cutoff at N≥6 (numerator 1-γ*ln(N) goes negative). The new additive
// formula has no such cutoff — DSR decreases gradually with N but doesn't zero out.
func TestDeflatedSharpe_NoHardCutoffAtN6(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 500)
	for i := range returns {
		returns[i] = 0.5 + float64(i%10)*0.1
	}
	moments := ComputeReturnMoments(returns)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = 6
	dsr6, passed6 := DeflatedSharpe(moments, cfg)
	t.Logf("N=6: SR=%.4f DSR=%.4f passed=%v T=%d", moments.SharpeRatio, dsr6, passed6, moments.NumObservations)
	if dsr6 <= 0 {
		t.Fatal("N=6 should not hard-cutoff to 0 — additive formula deflates gradually")
	}
	if !passed6 {
		t.Fatal("strong strategy with T=500 should still pass at N=6")
	}
}

// TestDeflatedSharpe_LargeNStillPassesWithLargeT verifies that with enough data,
// a strong strategy passes even with many attempts.
func TestDeflatedSharpe_LargeNStillPassesWithLargeT(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 500)
	for i := range returns {
		returns[i] = 0.5 + float64(i%10)*0.1
	}
	moments := ComputeReturnMoments(returns)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = 50
	dsr50, _ := DeflatedSharpe(moments, cfg)
	t.Logf("N=50 T=500: SR=%.4f DSR=%.4f", moments.SharpeRatio, dsr50)
	if dsr50 <= 0 {
		t.Fatal("strong strategy with T=500 should still have positive DSR at N=50")
	}
}

// TestDeflatedSharpe_SampleLengthMatters verifies that more observations
// lead to less deflation (higher DSR) for the same strategy.
func TestDeflatedSharpe_SampleLengthMatters(t *testing.T) {
	t.Parallel()
	returnsShort := make([]float64, 30)
	returnsLong := make([]float64, 500)
	for i := range returnsShort {
		returnsShort[i] = 0.002 + float64(i%10)*0.0003
	}
	for i := range returnsLong {
		returnsLong[i] = 0.002 + float64(i%10)*0.0003
	}
	momentsShort := ComputeReturnMoments(returnsShort)
	momentsLong := ComputeReturnMoments(returnsLong)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = 10
	dsrShort, _ := DeflatedSharpe(momentsShort, cfg)
	dsrLong, _ := DeflatedSharpe(momentsLong, cfg)
	t.Logf("T=30: DSR=%.4f, T=500: DSR=%.4f", dsrShort, dsrLong)
	if dsrLong <= dsrShort {
		t.Fatal("larger T should yield higher DSR (less deflation with more data)")
	}
}
