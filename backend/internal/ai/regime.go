// regime.go: market regime detection from OHLC price bars.
// Classifies markets into 5 states for regime-aware strategy scoring.
// Adapted from QuantDinger experiment/regime.py.

package ai

import "math"

// MarketRegime represents a classified market state.
type MarketRegime int

const (
	RegimeBullTrend       MarketRegime = iota // strong uptrend
	RegimeBearTrend                           // strong downtrend
	RegimeHighVolatility                      // elevated volatility
	RegimeRangeCompression                    // narrow range, low activity
	RegimeTransition                          // everything else
)

// String returns the regime name.
func (r MarketRegime) String() string {
	switch r {
	case RegimeBullTrend:
		return "bull_trend"
	case RegimeBearTrend:
		return "bear_trend"
	case RegimeHighVolatility:
		return "high_volatility"
	case RegimeRangeCompression:
		return "range_compression"
	default:
		return "transition"
	}
}

// OHLCBar represents a single price bar for regime detection.
type OHLCBar struct {
	Open, High, Low, Close float64
	Volume                 float64
}

// RegimeResult contains the detected regime and associated metrics.
type RegimeResult struct {
	Regime     MarketRegime
	Confidence float64
	Features   RegimeFeatures
}

// RegimeFeatures holds the computed market features.
type RegimeFeatures struct {
	PriceChangePct       float64
	EMAGapPct            float64
	RealizedVolPct       float64
	ATRPct               float64
	DirectionalEfficiency float64
	VolumeRatio          float64
}

// DetectRegime classifies the market using at least 30 OHLC bars.
// Returns transition regime with low confidence if insufficient data.
func DetectRegime(bars []OHLCBar) *RegimeResult {
	if len(bars) < 30 {
		return &RegimeResult{Regime: RegimeTransition, Confidence: 0.3}
	}
	f := computeFeatures(bars)

	switch {
	case f.EMAGapPct >= 1.0 && f.DirectionalEfficiency >= 0.55:
		if f.PriceChangePct > 1.0 {
			return &RegimeResult{
				Regime: RegimeBullTrend,
				Confidence: clamp01(0.55 + f.EMAGapPct*0.12 + f.DirectionalEfficiency*0.3),
				Features: f,
			}
		}
		if f.PriceChangePct < -1.0 {
			return &RegimeResult{
				Regime: RegimeBearTrend,
				Confidence: clamp01(0.55 + f.EMAGapPct*0.12 + f.DirectionalEfficiency*0.3),
				Features: f,
			}
		}
	case f.RealizedVolPct >= 4.5 || f.ATRPct >= 3.5:
		return &RegimeResult{
			Regime:     RegimeHighVolatility,
			Confidence: clamp01(0.5 + f.RealizedVolPct/10 + f.ATRPct/7),
			Features:   f,
		}
	case f.EMAGapPct <= 0.45 && f.DirectionalEfficiency <= 0.38 && f.ATRPct <= 2.0:
		return &RegimeResult{
			Regime: RegimeRangeCompression,
			Confidence: clamp01(0.52 + (0.45-f.EMAGapPct)*0.35 +
				(0.38-f.DirectionalEfficiency)*0.25),
			Features: f,
		}
	}
	return &RegimeResult{Regime: RegimeTransition, Confidence: 0.55, Features: f}
}

// computeFeatures calculates all 6 market features from price bars.
func computeFeatures(bars []OHLCBar) RegimeFeatures {
	n := len(bars)
	closes := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	volumes := make([]float64, n)
	for i, b := range bars {
		closes[i] = b.Close
		highs[i] = b.High
		lows[i] = b.Low
		volumes[i] = b.Volume
	}

	// 1. Price change %
	fChange := (closes[n-1]/closes[0] - 1) * 100

	// 2. EMA gap (10 vs 30)
	ema10 := ema(closes, 10)
	ema30 := ema(closes, 30)
	fEMAGap := (ema10/ema30 - 1) * 100

	// 3. Realized volatility (std dev of daily returns, annualized)
	returns := make([]float64, n-1)
	for i := 1; i < n; i++ {
		returns[i-1] = (closes[i]/closes[i-1] - 1) * 100
	}
	fVol := stdDev(returns) * math.Sqrt(252)

	// 4. ATR % (14-period)
	fATR := atrPct(highs, lows, closes, 14)

	// 5. Directional efficiency
	fEff := directionalEfficiency(closes)

	// 6. Volume ratio (latest / 20-period avg)
	avgVol := mean(volumes[n-min(20, n):])
	fVolRatio := volumes[n-1] / avgVol

	return RegimeFeatures{
		PriceChangePct:       math.Round(fChange*100) / 100,
		EMAGapPct:            math.Round(fEMAGap*100) / 100,
		RealizedVolPct:       math.Round(fVol*100) / 100,
		ATRPct:               math.Round(fATR*100) / 100,
		DirectionalEfficiency: math.Round(fEff*1000) / 1000,
		VolumeRatio:          math.Round(fVolRatio*100) / 100,
	}
}

// ema computes the Exponential Moving Average for a given period.
func ema(data []float64, period int) float64 {
	if len(data) < period {
		return data[len(data)-1]
	}
	k := 2.0 / float64(period+1)
	result := data[0]
	for _, v := range data[1:] {
		result = v*k + result*(1-k)
	}
	return result
}

// atrPct computes Average True Range as percentage of close.
func atrPct(highs, lows, closes []float64, period int) float64 {
	if len(highs) < period+1 {
		return 1.0
	}
	tr := make([]float64, len(highs)-1)
	for i := 1; i < len(highs); i++ {
		h := math.Abs(highs[i] - lows[i])
		c := math.Abs(closes[i] - closes[i-1])
		l := math.Abs(lows[i] - closes[i-1])
		tr[i-1] = math.Max(h, math.Max(c, l))
	}
	atr := mean(tr[len(tr)-period:])
	if closes[len(closes)-1] == 0 {
		return 1.0
	}
	return atr / closes[len(closes)-1] * 100
}

// directionalEfficiency measures how much of total path length is net progress.
func directionalEfficiency(closes []float64) float64 {
	n := len(closes)
	if n < 2 {
		return 0
	}
	net := math.Abs(closes[n-1] - closes[0])
	total := 0.0
	for i := 1; i < n; i++ {
		total += math.Abs(closes[i] - closes[i-1])
	}
	if total == 0 {
		return 0
	}
	return net / total
}

func mean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func stdDev(data []float64) float64 {
	if len(data) < 2 {
		return 0
	}
	m := mean(data)
	sumSq := 0.0
	for _, v := range data {
		d := v - m
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(data)-1))
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }
