// Package marketstate provides market condition quality indicators (M10-BASE-F1/F2/F3).
//
// Three core detectors:
//   - SpreadZScore: detects abnormal spread widening via rolling z-score
//   - SwapWindowDetector: identifies triple-swap days and rollover danger zones
//   - VolatilityRegime: classifies market as calm/ranging/trending/volatile
//
// A MarketState aggregates all detectors into a single market quality snapshot.

package marketstate

import "time"

// Regime classifies the current market volatility regime.
type Regime int

const (
	RegimeUnknown  Regime = 0
	RegimeCalm     Regime = 1 // low volatility, tight spreads
	RegimeRanging  Regime = 2 // sideways, mean-reverting
	RegimeTrending Regime = 3 // directional move
	RegimeVolatile Regime = 4 // high volatility, wide spreads
)

func (r Regime) String() string {
	switch r {
	case RegimeCalm:
		return "calm"
	case RegimeRanging:
		return "ranging"
	case RegimeTrending:
		return "trending"
	case RegimeVolatile:
		return "volatile"
	default:
		return "unknown"
	}
}

// MarketState is a snapshot of market quality for one symbol.
type MarketState struct {
	Symbol       string    `json:"symbol"`
	Timestamp    time.Time `json:"timestamp"`

	// Spread metrics
	SpreadPips       float64 `json:"spread_pips"`
	SpreadZScore     float64 `json:"spread_zscore"`      // >2 = widening alert
	SpreadWidening   bool    `json:"spread_widening"`    // alert active

	// Swap/rollover
	IsTripleSwapDay  bool    `json:"is_triple_swap_day"`
	InRolloverWindow bool    `json:"in_rollover_window"` // within N min of rollover
	MinutesToRollover float64 `json:"minutes_to_rollover"`

	// Volatility
	Regime           Regime  `json:"regime"`
	AnnualVol        float64 `json:"annual_vol"`         // annualized volatility estimate
	VolPercentile    float64 `json:"vol_percentile"`     // 0-1 vs historical

	// Overall
	QualityScore     float64 `json:"quality_score"`      // 0-1, higher = better to trade
	Tradable         bool    `json:"tradable"`           // conditions acceptable for trading
	Warnings         []string `json:"warnings"`
}

// DefaultMarketState returns a neutral market state.
func DefaultMarketState(symbol string) *MarketState {
	return &MarketState{
		Symbol:    symbol,
		Timestamp: time.Now(),
		Regime:    RegimeUnknown,
	}
}
