// Package ai provides Monte Carlo simulation for strategy robustness testing.
//
// Uses bootstrap resampling of daily returns to estimate:
//   - Confidence intervals for Sharpe ratio
//   - Distribution of max drawdown
//   - Probability of ruin (max DD exceeding threshold)
//   - Median survival metric
//
// Reference: López de Prado, "Advances in Financial Machine Learning" (2018), Ch. 11-12.

package ai

import (
	"math"
	"math/rand"
	"sort"
	"time"
)

// MonteCarloConfig holds parameters for Monte Carlo simulation.
type MonteCarloConfig struct {
	NumSimulations  int     // number of bootstrap resamples (default 1000)
	ConfidenceLevel float64 // confidence level for CI (default 0.95)
	MaxDDLimit      float64 // ruin threshold for max drawdown (default 0.30 = 30%)
	Seed            int64   // RNG seed; 0 = random (production), non-zero = reproducible (tests)
}

// DefaultMonteCarloConfig returns standard parameters.
func DefaultMonteCarloConfig() MonteCarloConfig {
	return MonteCarloConfig{
		NumSimulations:  1000,
		ConfidenceLevel: 0.95,
		MaxDDLimit:      0.30,
		Seed:            0, // 0 = random seed (production); tests set explicit seed
	}
}

// MonteCarloResult holds the output of a Monte Carlo simulation.
type MonteCarloResult struct {
	SharpeMedian    float64  // median Sharpe across simulations
	SharpeP5        float64  // 5th percentile Sharpe (lower CI bound)
	SharpeP95       float64  // 95th percentile Sharpe (upper CI bound)
	MaxDDMedian     float64  // median max drawdown
	MaxDDP95        float64  // 95th percentile max drawdown (worst-case)
	ProbRuin        float64  // P(max DD > MaxDDLimit)
	ProbPositive     float64  // P(Sharpe > 0)
	Passed          bool    // true if ProbPositive >= ConfidenceLevel and ProbRuin < 0.05
	Reason          string  // explanation if failed
}

// MonteCarlo runs bootstrap resampling on daily returns to estimate
// the statistical robustness of a strategy's performance.
//
// The method:
//  1. Resample daily returns with replacement N times
//  2. Compute Sharpe and max DD for each resample
//  3. Calculate percentile-based confidence intervals
//  4. Estimate probability of ruin and positive returns
func MonteCarlo(dailyReturns []float64, cfg MonteCarloConfig) MonteCarloResult {
	if len(dailyReturns) < 5 {
		return MonteCarloResult{
			Passed: false,
			Reason: "insufficient data for Monte Carlo (need >= 5 daily returns)",
		}
	}

	if cfg.NumSimulations < 100 {
		cfg.NumSimulations = 1000
	}
	if cfg.ConfidenceLevel <= 0 || cfg.ConfidenceLevel >= 1 {
		cfg.ConfidenceLevel = 0.95
	}
	if cfg.MaxDDLimit <= 0 {
		cfg.MaxDDLimit = 0.30
	}

	seed := cfg.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(seed))
	n := len(dailyReturns)

	sharpeSamples := make([]float64, 0, cfg.NumSimulations)
	ddSamples := make([]float64, 0, cfg.NumSimulations)

	for i := 0; i < cfg.NumSimulations; i++ {
		resampled := make([]float64, n)
		for j := 0; j < n; j++ {
			resampled[j] = dailyReturns[rng.Intn(n)]
		}
		sharpeSamples = append(sharpeSamples, computeSharpe(resampled))
		ddSamples = append(ddSamples, computeMaxDD(resampled))
	}

	sort.Float64s(sharpeSamples)
	sort.Float64s(ddSamples)

	result := MonteCarloResult{
		SharpeMedian: percentile(sharpeSamples, 0.50),
		SharpeP5:     percentile(sharpeSamples, 0.05),
		SharpeP95:    percentile(sharpeSamples, 0.95),
		MaxDDMedian:  percentile(ddSamples, 0.50),
		MaxDDP95:     percentile(ddSamples, 0.95),
	}

	// Probability of ruin: P(max DD > limit)
	ruinCount := 0
	for _, dd := range ddSamples {
		if dd > cfg.MaxDDLimit {
			ruinCount++
		}
	}
	result.ProbRuin = float64(ruinCount) / float64(cfg.NumSimulations)

	// Probability of positive Sharpe
	positiveCount := 0
	for _, sr := range sharpeSamples {
		if sr > 0 {
			positiveCount++
		}
	}
	result.ProbPositive = float64(positiveCount) / float64(cfg.NumSimulations)

	// Pass criteria: high probability of positive returns and low probability of ruin
	result.Passed = result.ProbPositive >= cfg.ConfidenceLevel && result.ProbRuin < 0.05
	if !result.Passed {
		if result.ProbPositive < cfg.ConfidenceLevel {
			result.Reason = "Monte Carlo: probability of positive returns below confidence threshold"
		} else {
			result.Reason = "Monte Carlo: probability of ruin exceeds 5%"
		}
	}

	return result
}

// percentile returns the value at the given percentile from a sorted slice.
// p must be in [0, 1].
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := p * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper {
		return sorted[lower]
	}
	weight := idx - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
