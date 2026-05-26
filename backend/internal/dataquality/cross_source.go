package dataquality

import "math"

// CrossSourceValidator compares price data across multiple sources (brokers/feeds).
// It detects price divergence that may indicate a bad feed or manipulated data.
type CrossSourceValidator struct {
	// MaxPriceDeviation is the maximum allowed price difference ratio between sources (default 0.01 = 1%).
	MaxPriceDeviation float64
	sources           map[string]*sourceState
}

type sourceState struct {
	lastBid  float64
	lastAsk  float64
	lastMid  float64
}

// NewCrossSourceValidator creates a validator with the given max deviation.
func NewCrossSourceValidator(maxDeviation float64) *CrossSourceValidator {
	if maxDeviation <= 0 {
		maxDeviation = 0.01 // 1% default
	}
	return &CrossSourceValidator{
		MaxPriceDeviation: maxDeviation,
		sources:           make(map[string]*sourceState),
	}
}

// Observe records a price tick from a source.
// source is the broker/feed identifier (e.g., "mt5-icmarkets", "mt4-pepperstone").
func (v *CrossSourceValidator) Observe(source string, bid, ask float64) {
	if bid <= 0 || ask <= 0 || bid > ask {
		return // invalid tick
	}
	v.sources[source] = &sourceState{
		lastBid: bid,
		lastAsk: ask,
		lastMid: (bid + ask) / 2.0,
	}
}

// Validate checks price consistency across all sources.
// Returns (valid, maxDeviation, deviationDetail).
func (v *CrossSourceValidator) Validate() (valid bool, maxDeviation float64, sourceCount int) {
	if len(v.sources) < 2 {
		return true, 0, len(v.sources)
	}

	// Compute max pairwise deviation between mid prices
	valid = true
	sourceCount = len(v.sources)

	mids := make([]float64, 0, len(v.sources))
	for _, s := range v.sources {
		mids = append(mids, s.lastMid)
	}

	for i := 0; i < len(mids); i++ {
		for j := i + 1; j < len(mids); j++ {
			if mids[i] <= 0 || mids[j] <= 0 {
				continue
			}
			deviation := math.Abs(mids[i]-mids[j]) / math.Min(mids[i], mids[j])
			if deviation > maxDeviation {
				maxDeviation = deviation
			}
			if deviation > v.MaxPriceDeviation {
				valid = false
			}
		}
	}

	return
}

// SourceCount returns the number of tracked sources.
func (v *CrossSourceValidator) SourceCount() int { return len(v.sources) }
