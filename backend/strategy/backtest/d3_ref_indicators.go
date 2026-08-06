package backtest

// D3 Semantic Layer — Independent Reference Indicator Implementations
//
// These implementations are deliberately independent from strategy/indicators/.
// They use float64 slices and different code structure to catch wiring bugs
// in the VM → SDK → indicators pipeline.
//
// Bar indexing: chronological (oldest first), shift counts back from the end.
// shift=0 → last element (current bar), shift=1 → second-to-last, etc.
//
// All formulas follow MQL4 documentation:
// https://docs.mql4.com/indicators

// refBar holds OHLC data for reference indicator calculations.
type refBar struct {
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// refClose extracts close prices as a float64 slice (chronological order).
func refClose(bars []refBar) []float64 {
	out := make([]float64, len(bars))
	for i, b := range bars {
		out[i] = b.Close
	}
	return out
}

// refHigh extracts high prices.
func refHigh(bars []refBar) []float64 {
	out := make([]float64, len(bars))
	for i, b := range bars {
		out[i] = b.High
	}
	return out
}

// refLow extracts low prices.
func refLow(bars []refBar) []float64 {
	out := make([]float64, len(bars))
	for i, b := range bars {
		out[i] = b.Low
	}
	return out
}

// refTypical returns typical price (H+L+C)/3 for each bar.
func refTypical(bars []refBar) []float64 {
	out := make([]float64, len(bars))
	for i, b := range bars {
		out[i] = (b.High + b.Low + b.Close) / 3
	}
	return out
}

// ── Moving Averages ──────────────────────────────────────────────────

// refSMA computes Simple Moving Average at the given shift.
func refSMA(data []float64, period, shift int) float64 {
	n := len(data)
	idx := n - 1 - shift
	if idx < period-1 {
		return 0
	}
	sum := 0.0
	for i := idx - period + 1; i <= idx; i++ {
		sum += data[i]
	}
	return sum / float64(period)
}

// refEMA computes Exponential Moving Average at the given shift.
// Uses SMA seed for the first value (matching MQL4 behavior).
func refEMA(data []float64, period, shift int) float64 {
	n := len(data)
	if n < period {
		return 0
	}
	k := 2.0 / float64(period+1)
	// Compute EMA for all bars up to the target shift.
	// Seed with SMA of first `period` bars.
	seed := 0.0
	for i := 0; i < period; i++ {
		seed += data[i]
	}
	seed /= float64(period)

	ema := seed
	for i := period; i < n-shift; i++ {
		ema = data[i]*k + ema*(1-k)
	}
	return ema
}

// refSMMA computes Smoothed Moving Average (Wilder's smoothing) at the given shift.
func refSMMA(data []float64, period, shift int) float64 {
	n := len(data)
	if n < period {
		return 0
	}
	// Seed with SMA of first `period` bars.
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += data[i]
	}
	smma := sum / float64(period)
	for i := period; i < n-shift; i++ {
		smma = (smma*float64(period-1) + data[i]) / float64(period)
	}
	return smma
}

// refLWMA computes Linear Weighted Moving Average at the given shift.
// Weight: period for the newest bar, 1 for the oldest (MQL4 convention).
func refLWMA(data []float64, period, shift int) float64 {
	n := len(data)
	idx := n - 1 - shift
	if idx < period-1 {
		return 0
	}
	num := 0.0
	den := 0.0
	for i := 0; i < period; i++ {
		w := float64(period - i) // newest (idx-i) gets weight=period, oldest gets weight=1
		num += data[idx-i] * w
		den += w
	}
	return num / den
}

// refMA dispatches to the correct MA method.
// method: 0=SMA, 1=EMA, 2=SMMA, 3=LWMA (matching MQL4 MODE_SMA etc.)
func refMA(data []float64, period, shift, method int) float64 {
	switch method {
	case 0:
		return refSMA(data, period, shift)
	case 1:
		return refEMA(data, period, shift)
	case 2:
		return refSMMA(data, period, shift)
	case 3:
		return refLWMA(data, period, shift)
	default:
		return refSMA(data, period, shift)
	}
}
