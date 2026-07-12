package ai

import (
	"math"
	"testing"
)

func TestMonteCarlo_PositiveEdge(t *testing.T) {
	t.Parallel()
	// Consistent positive returns → should pass
	returns := make([]float64, 252)
	for i := range returns {
		returns[i] = 0.001 + float64(i%10)*0.0001
	}
	cfg := DefaultMonteCarloConfig()
	result := MonteCarlo(returns, cfg)
	if !result.Passed {
		t.Fatalf("positive edge should pass: %s (P(+)=%.2f, P(ruin)=%.4f)",
			result.Reason, result.ProbPositive, result.ProbRuin)
	}
	if result.SharpeMedian <= 0 {
		t.Errorf("median Sharpe should be positive, got %.4f", result.SharpeMedian)
	}
	if result.SharpeP5 > result.SharpeMedian {
		t.Errorf("P5 (%.4f) should be <= median (%.4f)", result.SharpeP5, result.SharpeMedian)
	}
	if result.SharpeP95 < result.SharpeMedian {
		t.Errorf("P95 (%.4f) should be >= median (%.4f)", result.SharpeP95, result.SharpeMedian)
	}
}

func TestMonteCarlo_NegativeEdge(t *testing.T) {
	t.Parallel()
	// Consistent negative returns → should fail
	returns := make([]float64, 252)
	for i := range returns {
		returns[i] = -0.001 - float64(i%10)*0.0001
	}
	cfg := DefaultMonteCarloConfig()
	result := MonteCarlo(returns, cfg)
	if result.Passed {
		t.Fatal("negative edge should NOT pass")
	}
	if result.ProbPositive > 0.5 {
		t.Errorf("P(positive) should be < 0.5 for negative returns, got %.4f", result.ProbPositive)
	}
}

func TestMonteCarlo_InsufficientData(t *testing.T) {
	t.Parallel()
	result := MonteCarlo([]float64{0.01, 0.02}, DefaultMonteCarloConfig())
	if result.Passed {
		t.Fatal("insufficient data should not pass")
	}
}

func TestMonteCarlo_HighDrawdown(t *testing.T) {
	t.Parallel()
	// Alternating large gains/losses → high drawdown probability
	returns := make([]float64, 252)
	for i := range returns {
		if i%2 == 0 {
			returns[i] = 0.05
		} else {
			returns[i] = -0.04
		}
	}
	cfg := DefaultMonteCarloConfig()
	cfg.MaxDDLimit = 0.10 // strict threshold
	result := MonteCarlo(returns, cfg)
	if result.ProbRuin < 0.5 {
		t.Errorf("expected high P(ruin) with strict DD limit, got %.4f", result.ProbRuin)
	}
}

func TestMonteCarlo_Reproducible(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 100)
	for i := range returns {
		returns[i] = 0.001
	}
	cfg := DefaultMonteCarloConfig()
	cfg.Seed = 12345
	r1 := MonteCarlo(returns, cfg)
	r2 := MonteCarlo(returns, cfg)
	if r1.SharpeMedian != r2.SharpeMedian {
		t.Fatalf("same seed should produce same result: %.6f vs %.6f",
			r1.SharpeMedian, r2.SharpeMedian)
	}
}

func TestPercentile(t *testing.T) {
	t.Parallel()
	sorted := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if p := percentile(sorted, 0.0); p != 1 {
		t.Errorf("p=0: want 1, got %f", p)
	}
	if p := percentile(sorted, 1.0); p != 10 {
		t.Errorf("p=1: want 10, got %f", p)
	}
	if p := percentile(sorted, 0.5); p != 5.5 {
		t.Errorf("p=0.5: want 5.5, got %f", p)
	}
}

func TestPercentile_Single(t *testing.T) {
	t.Parallel()
	if p := percentile([]float64{42}, 0.5); p != 42 {
		t.Errorf("single element: want 42, got %f", p)
	}
	if p := percentile([]float64{}, 0.5); p != 0 {
		t.Errorf("empty: want 0, got %f", p)
	}
}

func TestMonteCarlo_NoNaN(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 100)
	for i := range returns {
		returns[i] = 0.001
	}
	result := MonteCarlo(returns, DefaultMonteCarloConfig())
	if math.IsNaN(result.SharpeMedian) || math.IsNaN(result.MaxDDMedian) {
		t.Fatal("result should not contain NaN")
	}
	if math.IsNaN(result.ProbRuin) || math.IsNaN(result.ProbPositive) {
		t.Fatal("probabilities should not be NaN")
	}
}
