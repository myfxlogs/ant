package marketstate

import "time"

// Aggregator combines spread, swap, and volatility detectors into a single MarketState.
type Aggregator struct {
	SpreadTracker  *SpreadTracker
	SwapDetector   *SwapWindowDetector
	VolClassifier  *VolatilityClassifier
}

// NewAggregator creates a market state aggregator with default detectors.
func NewAggregator() *Aggregator {
	return &Aggregator{
		SpreadTracker: NewSpreadTracker(200),
		SwapDetector:  NewSwapWindowDetector(),
		VolClassifier: NewVolatilityClassifier(200),
	}
}

// Observe records a new tick (price, spread in pips) for the aggregator's detectors.
func (a *Aggregator) Observe(price, spreadPips float64) {
	a.SpreadTracker.Observe(spreadPips)
	a.VolClassifier.Observe(price)
}

// Snapshot returns the current MarketState for the given symbol.
func (a *Aggregator) Snapshot(symbol string) *MarketState {
	now := time.Now()
	spreadPips := 0.0
	if a.SpreadTracker.Count() > 0 {
		spreadPips = a.SpreadTracker.Mean()
	}
	spreadZ := 0.0
	if a.SpreadTracker.Count() >= 10 {
		_, _, spreadZ = a.SpreadTracker.Stats(spreadPips)
	}

	regime := a.VolClassifier.Classify()
	annualVol := a.VolClassifier.AnnualVolatility()
	volPct := a.VolClassifier.VolPercentile()

	inRollover := a.SwapDetector.InRolloverWindow(now)
	minsToRollover := a.SwapDetector.MinutesToRollover(now)
	isTripleSwap := a.SwapDetector.IsTripleSwapDay(now)

	// Build warnings
	warnings := make([]string, 0)
	if spreadZ > 2.0 {
		warnings = append(warnings, "spread widening detected: z-score > 2")
	}
	if isTripleSwap {
		warnings = append(warnings, "triple swap day — elevated holding costs")
	}
	if inRollover {
		warnings = append(warnings, "within rollover window — spreads may be elevated")
	}
	if regime == RegimeVolatile {
		warnings = append(warnings, "volatile regime — reduced position sizing recommended")
	}

	// Quality score: 1.0 = perfect trading conditions, 0.0 = avoid at all costs
	score := 1.0
	if spreadZ > 2.0 {
		score -= 0.3
	} else if spreadZ > 1.5 {
		score -= 0.15
	}
	if inRollover {
		score -= 0.15
	}
	if isTripleSwap {
		score -= 0.05
	}
	if regime == RegimeVolatile {
		score -= 0.2
	} else if regime == RegimeUnknown {
		score -= 0.3
	}
	if score < 0 {
		score = 0
	}

	tradable := score >= 0.5 && spreadZ < 3.0 && regime != RegimeUnknown

	return &MarketState{
		Symbol:            symbol,
		Timestamp:         now,
		SpreadPips:        spreadPips,
		SpreadZScore:      spreadZ,
		SpreadWidening:    spreadZ > 2.0,
		IsTripleSwapDay:   isTripleSwap,
		InRolloverWindow:  inRollover,
		MinutesToRollover: minsToRollover,
		Regime:            regime,
		AnnualVol:         annualVol,
		VolPercentile:     volPct,
		QualityScore:      score,
		Tradable:          tradable,
		Warnings:          warnings,
	}
}
