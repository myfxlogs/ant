// Package ai provides Deflated Sharpe Ratio (M10-BASE-E3).
//
// DSR adjusts the Sharpe ratio for multiple testing bias using the
// López de Prado (2014) formula:
//
//	DSR = SR - [Z_c * sqrt(1 - skew*SR + (kurt+2)/4*SR^2) + gamma*ln(N)] / sqrt(T-1)
//
// Where:
//   - SR = annualized Sharpe ratio
//   - T = number of daily return observations (sample length)
//   - N = number of strategy variations the user has attempted
//   - Z_c = inverse normal CDF at confidence level c (e.g. 1.645 for 95%)
//   - gamma = Euler-Mascheroni constant (0.5772)
//   - skew = return distribution skewness
//   - kurt = return distribution excess kurtosis (raw kurtosis - 3)
//
// The term (kurt+2)/4 uses excess kurtosis, equivalent to (raw_kurt-1)/4 in the paper.
// DSR > 0 → strategy still has positive expected returns after all bias corrections.

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
	Mean           float64 // average daily return
	StdDev         float64 // standard deviation of daily returns
	Skewness       float64 // skewness (third moment)
	ExcessKurtosis float64 // excess kurtosis (fourth moment - 3)
	SharpeRatio    float64 // annualized Sharpe ratio
	NumObservations int    // T: number of daily return observations
}

// calcDistributionMoments computes skewness and excess kurtosis from standardized returns.
// Uses population moments (n denominator) — consistent with DSR formula expectations.
func calcDistributionMoments(returns []float64, mean, stdDev float64) (skewness, excessKurtosis float64) {
	n := float64(len(returns))
	var sumCube, sumQuad float64
	for _, r := range returns {
		d := (r - mean) / stdDev
		sumCube += d * d * d
		sumQuad += d * d * d * d
	}
	return sumCube / n, sumQuad/n - 3.0
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

	// Standard deviation (population moments, n denominator).
	var sumSqDiff float64
	for _, r := range dailyReturns {
		d := r - mean
		sumSqDiff += d * d
	}
	stdDev := math.Sqrt(sumSqDiff / float64(n))

	// Zero variance: skewness and kurtosis are undefined.
	if stdDev == 0 {
		return ReturnMoments{Mean: mean, StdDev: 0, SharpeRatio: 0, NumObservations: n}
	}

	skew, kurt := calcDistributionMoments(dailyReturns, mean, stdDev)
	sharpe := 0.0
	if stdDev > 0 {
		sharpe = (mean / stdDev) * math.Sqrt(252)
	}

	return ReturnMoments{
		Mean: mean, StdDev: stdDev,
		Skewness: skew, ExcessKurtosis: kurt,
		SharpeRatio: sharpe, NumObservations: n,
	}
}

// normInv computes the inverse of the standard normal CDF using Acklam's algorithm.
// Accuracy: relative error < 1.15e-9 across the full range.
func normInv(p float64) float64 {
	const (
		a1 = -3.969683028665376e+01
		a2 =  2.209460984245205e+02
		a3 = -2.759285104469687e+02
		a4 =  1.383577518672690e+02
		a5 = -3.066479806614716e+01
		a6 =  2.506628277459239e+00

		b1 = -5.447609879822406e+01
		b2 =  1.615858368580409e+02
		b3 = -1.556989798598866e+02
		b4 =  6.680131188771972e+01
		b5 = -1.328068155288572e+01

		c1 = -7.784894002430293e-03
		c2 = -3.223964580411365e-01
		c3 = -2.400758277161838e+00
		c4 = -2.549726590733709e+00

		d1 =  7.784695709041462e-03
		d2 =  3.224671290700398e-01
		d3 =  2.445134137142996e+00

		pLow  = 0.02425
		pHigh = 1 - pLow
	)

	if p < 0 || p > 1 {
		return 0
	}
	if p == 0 {
		return math.Inf(-1)
	}
	if p == 1 {
		return math.Inf(1)
	}

	var q, x float64

	if p < pLow {
		q = math.Sqrt(-2 * math.Log(p))
		x = (((((c1*q+c2)*q+c3)*q+c4)*q+1)) / ((((d1*q+d2)*q+d3)*q+1)*q)
	} else if p <= pHigh {
		q = p - 0.5
		r := q * q
		x = (((((a1*r+a2)*r+a3)*r+a4)*r+a5)*r+a6) * q /
			(((((b1*r+b2)*r+b3)*r+b4)*r+b5)*r+1)
	} else {
		q = math.Sqrt(-2 * math.Log(1-p))
		x = -(((((c1*q+c2)*q+c3)*q+c4)*q+1)) / ((((d1*q+d2)*q+d3)*q+1)*q)
	}

	return x
}

// DeflatedSharpe computes the Deflated Sharpe Ratio per López de Prado (2014).
//
// Formula: DSR = SR - [Z_c * sqrt(1 - skew*SR + (kurt+2)/4*SR^2) + gamma*ln(N)] / sqrt(T-1)
//
// Where Z_c = normInv(confidenceLevel), T = NumObservations, N = NumAttempts.
// The term (kurt+2)/4 uses excess kurtosis (equivalent to (raw_kurt-1)/4 in the paper).
//
// Returns (DSR, passed) where passed = DSR > 0 (strategy still profitable after all corrections).
func DeflatedSharpe(moments ReturnMoments, cfg DeflatedSharpeConfig) (float64, bool) {
	if cfg.NumAttempts <= 0 {
		cfg.NumAttempts = 1
	}
	if cfg.Gamma <= 0 {
		cfg.Gamma = 0.5772156649
	}
	if cfg.ConfidenceLevel <= 0 || cfg.ConfidenceLevel >= 1 {
		cfg.ConfidenceLevel = 0.95
	}

	SR := moments.SharpeRatio
	if SR <= 0 {
		return 0, false
	}

	T := float64(moments.NumObservations)
	if T < 2 {
		return 0, false
	}

	N := float64(cfg.NumAttempts)
	skew := moments.Skewness
	kurt := moments.ExcessKurtosis

	// PSR variance term: 1 - skew*SR + (kurt+2)/4*SR²
	// (kurt+2)/4 with excess kurtosis = (raw_kurt-1)/4 in the paper.
	psrVar := 1.0 - skew*SR + (kurt+2.0)/4.0*SR*SR
	if psrVar < 0 {
		psrVar = 0
	}

	// Z_c = inverse normal CDF at confidence level (e.g. 1.645 for 95%).
	Z_c := normInv(cfg.ConfidenceLevel)

	// DSR = SR - [Z_c * sqrt(psrVar) + gamma*ln(N)] / sqrt(T-1)
	// When N=1, ln(1)=0 → no multiple-testing penalty.
	// When T is large, the penalty shrinks (more data = more confidence).
	penalty := (Z_c*math.Sqrt(psrVar) + cfg.Gamma*math.Log(N)) / math.Sqrt(T-1)
	DSR := SR - penalty

	passed := DSR > 0
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
