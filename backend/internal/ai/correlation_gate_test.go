// Package ai — E5 Correlation Gate tests.
package ai

import (
	"math"
	"testing"
)

func TestCorrelationGate_LowCorrelation(t *testing.T) {
	t.Parallel()
	newSignals := []SignalDirection{
		{Timestamp: 1, Direction: 1},
		{Timestamp: 2, Direction: -1},
		{Timestamp: 3, Direction: 1},
		{Timestamp: 4, Direction: 1},
		{Timestamp: 5, Direction: -1},
	}
	existing := map[string][]SignalDirection{
		"strat_a": {
			{Timestamp: 1, Direction: -1},
			{Timestamp: 2, Direction: 1},
			{Timestamp: 3, Direction: -1},
			{Timestamp: 4, Direction: -1},
			{Timestamp: 5, Direction: 1},
		},
	}
	cfg := DefaultCorrelationGateConfig()
	cfg.MinObservations = 5
	result := CorrelationGate(newSignals, existing, cfg)
	if !result.Passed {
		t.Fatalf("opposite strategies should pass: %s (corr=%.4f)", result.Reason, result.MaxCorrelation)
	}
	t.Logf("Low correlation: max=%.4f", result.MaxCorrelation)
}

func TestCorrelationGate_HighCorrelation(t *testing.T) {
	t.Parallel()
	signals := make([]SignalDirection, 50)
	for i := range signals {
		dir := 1.0
		if i%7 == 0 {
			dir = -1.0
		}
		signals[i] = SignalDirection{Timestamp: int64(i), Direction: dir}
	}
	copy2 := make([]SignalDirection, 50)
	copy(copy2, signals)
	copy2[0] = SignalDirection{Timestamp: 0, Direction: 1.0}
	existing := map[string][]SignalDirection{
		"strat_similar": copy2,
	}
	cfg := DefaultCorrelationGateConfig()
	cfg.MinObservations = 20
	result := CorrelationGate(signals, existing, cfg)
	if result.Passed {
		t.Fatal("highly correlated signals should be rejected")
	}
	t.Logf("High correlation: max=%.4f strategy=%s", result.MaxCorrelation, result.CorrelatedStrategy)
}

func TestCorrelationGate_InsufficientObservations(t *testing.T) {
	t.Parallel()
	signals := make([]SignalDirection, 10)
	existing := map[string][]SignalDirection{}
	cfg := DefaultCorrelationGateConfig()
	cfg.MinObservations = 20
	result := CorrelationGate(signals, existing, cfg)
	if result.Passed {
		t.Fatal("insufficient observations should not pass")
	}
}

func TestPearsonCorrelation(t *testing.T) {
	t.Parallel()
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{2, 4, 6, 8, 10}
	r := pearsonCorrelation(x, y)
	if math.Abs(r-1.0) > 0.001 {
		t.Fatalf("perfect positive correlation: want 1.0, got %.4f", r)
	}

	y2 := []float64{10, 8, 6, 4, 2}
	r2 := pearsonCorrelation(x, y2)
	if math.Abs(r2-(-1.0)) > 0.001 {
		t.Fatalf("perfect negative correlation: want -1.0, got %.4f", r2)
	}
}
