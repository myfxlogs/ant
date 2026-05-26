// Package ai provides Deflated Sharpe Ratio (M10-BASE-E3).
//
// DSR adjusts the Sharpe ratio for multiple testing bias using the
// López de Prado (2014) formula:
//
//	DSR = SR * sqrt[(1 - gamma*ln(N)) / (1 - skew*SR + (kurt-1)/4*SR^2)]
//
// Where:
//   - SR = annualized Sharpe ratio
//   - N = number of strategy variations the user has attempted
//   - gamma = Euler-Mascheroni constant approximation (0.5772)
//   - skew = return distribution skewness
//   - kurt = return distribution excess kurtosis
//
// DSR < 0.95 (95% confidence) → reject the strategy.
// A raw SR of 0.5 with N=100 attempts gets deflated below the rejection threshold.

package ai

import "math"

// DeflatedSharpeConfig holds parameters for DSR calculation.
type DeflatedSharpeConfig struct {
	NumAttempts       int     // N: number of strategy variations attempted (default 1)
	ConfidenceLevel   float64 // rejection threshold (default 0.95 for 95%)
	Gamma             float64 // Euler-Mascheroni constant approximation
}

// DefaultDeflatedSharpeConfig returns standard DSR parameters.
func DefaultDeflatedSharpeConfig() DeflatedSharpeConfig {
	return DeflatedSharpeConfig{
		NumAttempts:     1,
		ConfidenceLevel: 0.95,
		Gamma:           0.5772156649, // Euler-Mascheroni constant
	}
}

// ReturnMoments holds the first four moments of a return distribution.
type ReturnMoments struct {
	Mean       float64 // average daily return
	StdDev     float64 // standard deviation of daily returns
	Skewness   float64 // skewness (third moment)
	ExcessKurtosis float64 // excess kurtosis (fourth moment - 3)
	SharpeRatio float64 // annualized Sharpe ratio
}

// ComputeReturnMoments calculates the first four moments from daily returns.
func ComputeReturnMoments(dailyReturns []float64) ReturnMoments {
	n := len(dailyReturns)
	if n < 4 {
		return ReturnMoments{}
	}

	// Mean.
	var sum float64
	for _, r := range dailyReturns {
		sum += r
	}
	mean := sum / float64(n)

	// Standard deviation.
	var sumSqDiff float64
	for _, r := range dailyReturns {
		d := r - mean
		sumSqDiff += d * d
	}
	variance := sumSqDiff / float64(n)
	stdDev := math.Sqrt(variance)

	// Skewness and kurtosis.
	var sumCube, sumQuad float64
	for _, r := range dailyReturns {
		d := (r - mean) / stdDev
		sumCube += d * d * d
		sumQuad += d * d * d * d
	}
	skew := sumCube / float64(n)
	kurt := sumQuad/float64(n) - 3.0 // excess kurtosis

	// Annualized Sharpe.
	sharpe := 0.0
	if stdDev > 0 {
		sharpe = (mean / stdDev) * math.Sqrt(252)
	}

	return ReturnMoments{
		Mean:           mean,
		StdDev:         stdDev,
		Skewness:       skew,
		ExcessKurtosis: kurt,
		SharpeRatio:    sharpe,
	}
}

// DeflatedSharpe computes the Deflated Sharpe Ratio per López de Prado (2014).
//
// Formula: DSR = SR * sqrt[(1 - gamma*ln(N)) / (1 - skew*SR + (kurt-1)/4*SR^2)]
//
// Returns (DSR, passed) where passed = DSR >= confidenceLevel.
func DeflatedSharpe(moments ReturnMoments, cfg DeflatedSharpeConfig) (float64, bool) {
	if cfg.NumAttempts <= 0 {
		cfg.NumAttempts = 1
	}
	if cfg.Gamma <= 0 {
		cfg.Gamma = 0.5772156649
	}
	if cfg.ConfidenceLevel <= 0 {
		cfg.ConfidenceLevel = 0.95
	}

	SR := moments.SharpeRatio
	if SR <= 0 {
		return 0, false
	}

	N := float64(cfg.NumAttempts)
	skew := moments.Skewness
	kurt := moments.ExcessKurtosis

	// Numerator: 1 - gamma * ln(N)
	// When N=1, ln(1)=0 → numerator=1.
	numerator := 1.0 - cfg.Gamma*math.Log(N)
	if numerator <= 0 {
		return 0, false // too many attempts degrade SR to zero
	}

	// Denominator: 1 - skew*SR + (kurt+2)/4*SR²
	// Note: excess kurtosis + 3 = regular kurtosis; the formula uses (kurt-1)/4.
	// When denominator is non-positive, the Edgeworth expansion is invalid.
	// Fall back to using only the multiple-testing correction (numerator).
	denominator := 1.0 - skew*SR + (kurt-1.0)/4.0*SR*SR
	if denominator <= 0 {
		// Edgeworth expansion breaks down for extreme distributions.
		// Use only the multiple-testing deflation.
		deflationFactor := math.Sqrt(math.Max(numerator, 0))
		DSR := SR * deflationFactor
		passed := DSR >= cfg.ConfidenceLevel
		return DSR, passed
	}

	deflationFactor := math.Sqrt(numerator / denominator)
	DSR := SR * deflationFactor

	passed := DSR >= cfg.ConfidenceLevel
	return DSR, passed
}

// DeflatedSharpeFromReturns is a convenience function that computes DSR
// directly from daily returns.
func DeflatedSharpeFromReturns(dailyReturns []float64, numAttempts int) (float64, bool) {
	moments := ComputeReturnMoments(dailyReturns)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = numAttempts
	return DeflatedSharpe(moments, cfg)
}
