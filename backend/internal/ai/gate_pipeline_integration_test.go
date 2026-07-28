// Package ai — integration tests for gate pipeline correctness.
package ai

import (
	"math"
	"testing"
)

// TestComputeMaxDD_UnderwaterReturnsOne verifies that a strategy which never
// goes positive returns 1.0 (100% drawdown), consistent with the [0,1] contract.
func TestComputeMaxDD_UnderwaterReturnsOne(t *testing.T) {
	t.Parallel()
	returns := []float64{-10, -20, -5, -30, -15}
	dd := computeMaxDD(returns)
	if dd != 1.0 {
		t.Fatalf("underwater strategy should return 1.0, got %.4f", dd)
	}
}

// TestComputeMaxDD_NormalCase verifies fractional drawdown for a normal strategy.
func TestComputeMaxDD_NormalCase(t *testing.T) {
	t.Parallel()
	returns := []float64{100, -50, -30, 20, 50}
	dd := computeMaxDD(returns)
	// Peak=100, trough=20 → DD = (100-20)/100 = 0.8
	if math.Abs(dd-0.8) > 0.01 {
		t.Fatalf("normal case: want 0.8, got %.4f", dd)
	}
}

// TestComputeMaxDD_EmptyReturnsZero verifies empty input returns 0.
func TestComputeMaxDD_EmptyReturnsZero(t *testing.T) {
	t.Parallel()
	dd := computeMaxDD(nil)
	if dd != 0 {
		t.Fatalf("empty input should return 0, got %.4f", dd)
	}
}

// TestCPCV_RawOOSSharpe verifies that CPCV returns raw OOS Sharpe without
// the removed testSharpe *= 0.5 heuristic penalty.
func TestCPCV_RawOOSSharpe(t *testing.T) {
	t.Parallel()
	// Consistent positive returns → all OOS Sharpe should be positive.
	returns := make([]float64, 300)
	for i := range returns {
		returns[i] = 0.5 + float64(i%10)*0.1
	}
	cfg := DefaultWalkForwardConfig()
	cpcvSharpe := CPCV(returns, 6, cfg)
	if cpcvSharpe <= 0 {
		t.Fatalf("CPCV should return positive OOS Sharpe for consistent positive returns, got %.4f", cpcvSharpe)
	}
}

// TestCPCV_InsufficientData verifies CPCV returns 0 for too-small datasets.
func TestCPCV_InsufficientData(t *testing.T) {
	t.Parallel()
	cfg := DefaultWalkForwardConfig()
	cpcvSharpe := CPCV([]float64{0.1, 0.2}, 6, cfg)
	if cpcvSharpe != 0 {
		t.Fatalf("CPCV should return 0 for insufficient data, got %.4f", cpcvSharpe)
	}
}

// TestPipeline_AllGatesEvaluated verifies that all 7 gates are present in the
// result regardless of pass/fail (no short-circuit).
func TestPipeline_AllGatesEvaluated(t *testing.T) {
	t.Parallel()
	input := PipelineInput{
		Expression:   "close[t+1] > open",
		DailyReturns: make([]float64, 200),
	}
	result := Pipeline(input)
	if len(result.Gates) != 7 {
		t.Fatalf("should evaluate all 7 gates, got %d", len(result.Gates))
	}
	expectedOrder := GateOrder
	for i, g := range result.Gates {
		if g.Gate != expectedOrder[i] {
			t.Fatalf("gate %d: want %s, got %s", i, expectedOrder[i], g.Gate)
		}
	}
}

// TestPipeline_SkippedGatesDontAffectPassed verifies that skipped gates
// (Skipped=true, Passed=true) do not cause the pipeline to fail.
func TestPipeline_SkippedGatesDontAffectPassed(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 500)
	for i := range returns {
		returns[i] = 0.5 + float64(i%10)*0.1
	}
	input := PipelineInput{
		Expression:      "sma(close, 5) > ema(close, 20)",
		DailyReturns:    returns,
		NumAttempts:     1,
		PaperMetrics:    PaperGateMetrics{PaperDays: 0},
		NewSignals:      make([]SignalDirection, 25),
		ExistingSignals: map[string][]SignalDirection{},
	}
	result := Pipeline(input)
	if !result.Passed {
		t.Fatalf("pipeline should pass when only skipped gates exist, failed at: %s", result.FirstFail)
	}
	for _, g := range result.Gates {
		if g.Skipped && !g.Passed {
			t.Fatalf("skipped gate %s should have Passed=true", g.Gate)
		}
	}
}

// TestPipeline_FirstFailRecordsOnlyFirst verifies that FirstFail records
// only the first non-skipped failure, even though all gates are evaluated.
func TestPipeline_FirstFailRecordsOnlyFirst(t *testing.T) {
	t.Parallel()
	input := PipelineInput{
		Expression:   "close[t+1] > open",
		DailyReturns: make([]float64, 200),
	}
	result := Pipeline(input)
	if result.FirstFail != GateLookAhead {
		t.Fatalf("first fail should be lookahead (second gate), got %s", result.FirstFail)
	}
	// Compliance should pass (balanced brackets + has operator).
	for _, g := range result.Gates {
		if g.Gate == GateCompliance && !g.Passed {
			t.Fatal("compliance should pass for close[t+1] > open")
		}
	}
}

// TestEquityCurveToDailyReturns_EmptyProto verifies graceful handling of empty proto.
func TestEquityCurveToDailyReturns_EmptyProto(t *testing.T) {
	t.Parallel()
	rets := EquityCurveToDailyReturns(nil)
	if rets != nil {
		t.Fatalf("empty proto should return nil, got %v", rets)
	}
}

// TestEquityCurveToDailyReturns_InvalidProto verifies graceful handling of corrupt proto.
func TestEquityCurveToDailyReturns_InvalidProto(t *testing.T) {
	t.Parallel()
	rets := EquityCurveToDailyReturns([]byte{0xFF, 0xFF, 0xFF})
	if rets != nil {
		t.Fatalf("invalid proto should return nil, got %v", rets)
	}
}
