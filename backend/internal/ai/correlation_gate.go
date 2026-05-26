// Package ai provides Correlation Gate (M10-BASE-E5).
//
// Computes signal direction correlation between a new strategy and existing
// live strategies. If correlation >= 0.7, the new strategy is considered
// redundant (same effective signal) and should be rejected.
//
// This prevents the "20 strategies that are essentially the same thing" problem.

package ai

import "math"

// CorrelationGateConfig holds parameters for signal correlation checking.
type CorrelationGateConfig struct {
	MaxCorrelation  float64 // maximum allowed signal correlation (default 0.7)
	MinObservations int     // minimum signal pairs to compute correlation (default 20)
}

// DefaultCorrelationGateConfig returns standard parameters.
func DefaultCorrelationGateConfig() CorrelationGateConfig {
	return CorrelationGateConfig{
		MaxCorrelation:  0.7,
		MinObservations: 20,
	}
}

// SignalDirection represents the direction of a strategy's signal at a point in time.
// +1 = long, -1 = short, 0 = neutral/no signal.
type SignalDirection struct {
	Timestamp int64   `json:"ts_unix_ms"`
	Direction float64 `json:"direction"` // +1, -1, or 0
}

// CorrelationGateResult is the outcome of the correlation gate check.
type CorrelationGateResult struct {
	Passed             bool    `json:"passed"`
	MaxCorrelation     float64 `json:"max_correlation"`     // highest correlation found
	CorrelatedStrategy string  `json:"correlated_strategy,omitempty"` // which existing strategy
	Reason             string  `json:"reason,omitempty"`
}

// CorrelationGate checks if a new strategy's signals are too correlated with existing ones.
func CorrelationGate(newSignals []SignalDirection, existing map[string][]SignalDirection, cfg CorrelationGateConfig) CorrelationGateResult {
	result := CorrelationGateResult{Passed: true}

	if len(newSignals) < cfg.MinObservations {
		result.Passed = false
		result.Reason = "insufficient signal observations"
		return result
	}

	// Align signals by timestamp and compute direction correlation with each existing strategy.
	for name, existingSignals := range existing {
		if len(existingSignals) < cfg.MinObservations {
			continue
		}

		corr := computeDirectionCorrelation(newSignals, existingSignals)
		if corr > result.MaxCorrelation {
			result.MaxCorrelation = corr
		}

		if corr >= cfg.MaxCorrelation {
			result.Passed = false
			result.CorrelatedStrategy = name
			result.Reason = "signal correlation too high with existing strategy"
			return result // fail fast on first correlated strategy
		}
	}

	// If we got through all existing strategies without failing, check if any correlation exceeded.
	if !result.Passed {
		return result
	}

	return result
}

// computeDirectionCorrelation calculates the Pearson correlation between two signal direction series.
// Aligns by timestamp (takes the intersection of timestamps).
func computeDirectionCorrelation(a, b []SignalDirection) float64 {
	// Build timestamp index for b.
	bIdx := make(map[int64]float64, len(b))
	for _, s := range b {
		bIdx[s.Timestamp] = s.Direction
	}

	// Collect paired observations.
	var x, y []float64
	for _, sa := range a {
		if sb, ok := bIdx[sa.Timestamp]; ok {
			x = append(x, sa.Direction)
			y = append(y, sb)
		}
	}
	if len(x) < 2 {
		return 0
	}

	return pearsonCorrelation(x, y)
}

// pearsonCorrelation computes the Pearson correlation coefficient.
func pearsonCorrelation(x, y []float64) float64 {
	n := float64(len(x))
	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	num := n*sumXY - sumX*sumY
	denX := n*sumX2 - sumX*sumX
	denY := n*sumY2 - sumY*sumY

	if denX <= 0 || denY <= 0 {
		return 0
	}

	r := num / math.Sqrt(denX*denY)
	if math.IsNaN(r) {
		return 0
	}
	if r > 1 {
		r = 1
	}
	if r < -1 {
		r = -1
	}
	return r
}
