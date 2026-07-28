// Package ai provides Sharpe ratio and max drawdown computation utilities.
package ai

import "math"

// computeSharpe calculates the annualized Sharpe ratio from daily returns.
// Assumes 252 trading days/year. Uses population standard deviation.
func computeSharpe(dailyReturns []float64) float64 {
	if len(dailyReturns) < 2 {
		return 0
	}
	// Reject NaN/Inf inputs — they silently bypass downstream checks.
	for _, r := range dailyReturns {
		if math.IsNaN(r) || math.IsInf(r, 0) {
			return 0
		}
	}
	var sum, sumSq float64
	for _, r := range dailyReturns {
		sum += r
		sumSq += r * r
	}
	n := float64(len(dailyReturns))
	mean := sum / n
	variance := sumSq/n - mean*mean
	if variance <= 0 {
		return 0
	}
	dailySD := math.Sqrt(variance)
	if dailySD == 0 {
		return 0
	}
	return (mean / dailySD) * math.Sqrt(252)
}

// computeMaxDD calculates the maximum drawdown as a fraction of peak equity.
// Returns a value in [0, 1] where 0.30 = 30% drawdown.
// For underwater strategies (cumulative return never positive), returns 1.0
// (100% drawdown — the strategy never recovered above its starting point).
func computeMaxDD(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	peak := 0.0
	maxDD := 0.0
	cumulative := 0.0
	for _, r := range returns {
		cumulative += r
		if cumulative > peak {
			peak = cumulative
		}
		if peak > 0 {
			dd := (peak - cumulative) / peak
			if dd > maxDD {
				maxDD = dd
			}
		}
	}
	// Underwater strategy: cumulative return never positive → 100% drawdown.
	if peak == 0 {
		return 1.0
	}
	return maxDD
}
