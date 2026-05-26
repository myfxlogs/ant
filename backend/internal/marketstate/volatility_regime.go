package marketstate

import "math"

// VolatilityClassifier determines the current volatility regime from price data.
type VolatilityClassifier struct {
	prices   []float64 // rolling close prices
	returns  []float64 // rolling log returns
	capacity int
	index    int
	count    int

	// Thresholds
	calmVolAnnual    float64 // annual vol below this = calm (default 0.08)
	volatileVolAnnual float64 // annual vol above this = volatile (default 0.35)
}

// NewVolatilityClassifier creates a classifier with the given window size.
func NewVolatilityClassifier(windowSize int) *VolatilityClassifier {
	if windowSize <= 0 {
		windowSize = 100
	}
	return &VolatilityClassifier{
		prices:            make([]float64, windowSize),
		returns:           make([]float64, windowSize),
		capacity:          windowSize,
		calmVolAnnual:     0.08,
		volatileVolAnnual: 0.35,
	}
}

// Observe records a new price observation.
func (vc *VolatilityClassifier) Observe(price float64) {
	if vc.count > 0 {
		prev := vc.prices[(vc.index-1+vc.capacity)%vc.capacity]
		if prev > 0 {
			vc.returns[vc.index] = math.Log(price / prev)
		}
	}
	vc.prices[vc.index] = price
	vc.index = (vc.index + 1) % vc.capacity
	if vc.count < vc.capacity {
		vc.count++
	}
}

// Classify determines the current volatility regime.
func (vc *VolatilityClassifier) Classify() Regime {
	if vc.count < 20 {
		return RegimeUnknown
	}

	annualVol := vc.AnnualVolatility()
	_ = vc.trendStrength() // reserved for future trend detection refinement

	switch {
	case annualVol < vc.calmVolAnnual:
		return RegimeCalm
	case annualVol > vc.volatileVolAnnual:
		return RegimeVolatile
	default:
		// Distinguish ranging vs trending using price efficiency
		if vc.isTrending() {
			return RegimeTrending
		}
		return RegimeRanging
	}
}

// AnnualVolatility computes the annualized volatility from rolling returns.
func (vc *VolatilityClassifier) AnnualVolatility() float64 {
	if vc.count < 2 {
		return 0
	}
	n := vc.count - 1
	if n < 1 {
		n = 1
	}

	mean := 0.0
	for i := 0; i < n; i++ {
		mean += vc.returns[i]
	}
	mean /= float64(n)

	variance := 0.0
	for i := 0; i < n; i++ {
		d := vc.returns[i] - mean
		variance += d * d
	}
	variance /= float64(n - 1)
	if variance < 0 {
		variance = 0
	}

	return math.Sqrt(variance) * math.Sqrt(252)
}

// VolPercentile returns the current volatility as a fraction (0-1) vs typical ranges.
func (vc *VolatilityClassifier) VolPercentile() float64 {
	vol := vc.AnnualVolatility()
	if vol >= vc.volatileVolAnnual {
		return 1.0
	}
	if vol <= vc.calmVolAnnual {
		return 0.0
	}
	return (vol - vc.calmVolAnnual) / (vc.volatileVolAnnual - vc.calmVolAnnual)
}

func (vc *VolatilityClassifier) isTrending() bool {
	if vc.count < 20 {
		return false
	}
	// Price efficiency: net change / total path length
	n := vc.count
	start := vc.prices[(vc.index-n+vc.capacity)%vc.capacity]
	end := vc.prices[(vc.index-1+vc.capacity)%vc.capacity]
	if start <= 0 {
		return false
	}

	netChange := math.Abs(end - start)
	totalPath := 0.0
	for i := 1; i < n; i++ {
		idx := (vc.index - i + vc.capacity) % vc.capacity
		prev := (vc.index - i - 1 + vc.capacity) % vc.capacity
		totalPath += math.Abs(vc.prices[idx] - vc.prices[prev])
	}
	if totalPath <= 0 {
		return false
	}
	// > 0.4 efficiency = trending (directional), < 0.4 = ranging (choppy)
	return netChange/totalPath > 0.4
}

func (vc *VolatilityClassifier) trendStrength() float64 {
	if vc.count < 20 {
		return 0
	}
	n := vc.count
	start := vc.prices[(vc.index-n+vc.capacity)%vc.capacity]
	end := vc.prices[(vc.index-1+vc.capacity)%vc.capacity]
	if start <= 0 {
		return 0
	}
	totalPath := 0.0
	for i := 1; i < n; i++ {
		idx := (vc.index - i + vc.capacity) % vc.capacity
		prev := (vc.index - i - 1 + vc.capacity) % vc.capacity
		totalPath += math.Abs(vc.prices[idx] - vc.prices[prev])
	}
	if totalPath <= 0 {
		return 0
	}
	return math.Abs(end-start) / totalPath
}
