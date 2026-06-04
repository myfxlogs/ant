// Package ai — E3 Deflated Sharpe Ratio tests.
package ai

import "testing"

func TestDeflatedSharpe_PositiveEdge(t *testing.T) {
	t.Parallel()
	returns := []float64{
		0.002, -0.003, 0.005, -0.001, 0.004, -0.006, 0.003, 0.001, -0.002, 0.004,
		-0.001, 0.003, -0.005, 0.002, 0.004, -0.002, 0.001, -0.003, 0.006, -0.001,
	}
	moments := ComputeReturnMoments(returns)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = 1
	dsr, passed := DeflatedSharpe(moments, cfg)
	t.Logf("SR=%.4f DSR=%.4f Passed=%v", moments.SharpeRatio, dsr, passed)
	if !passed {
		t.Fatal("positive edge with N=1 should pass")
	}
}

func TestDeflatedSharpe_N100_Deflates(t *testing.T) {
	t.Parallel()
	returns := []float64{0.003, -0.002, 0.001, 0.004, -0.001, 0.002, -0.003, 0.001, 0.000, 0.002}
	moments := ComputeReturnMoments(returns)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = 100
	dsr, _ := DeflatedSharpe(moments, cfg)
	t.Logf("N=100: SR=%.4f DSR=%.4f", moments.SharpeRatio, dsr)
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
	returns := []float64{0.002, -0.003, 0.001, 0.004, -0.001, 0.002, -0.002, 0.003, -0.001, 0.001}
	moments := ComputeReturnMoments(returns)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = 1
	dsr, _ := DeflatedSharpe(moments, cfg)
	t.Logf("N=1: SR=%.4f DSR=%.4f", moments.SharpeRatio, dsr)
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
	returns := []float64{0.002, -0.003, 0.001, 0.004, -0.001, 0.002, -0.002, 0.003, -0.001, 0.001}
	dsr, passed := DeflatedSharpeFromReturns(returns, 1)
	t.Logf("DSR from returns: %.4f passed=%v", dsr, passed)
	if dsr <= 0 {
		t.Fatal("positive returns should yield positive DSR")
	}
}
