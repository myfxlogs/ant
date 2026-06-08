package analysis

import "math"

// --- Math helpers (mirror ai/regime.go for package independence) ---

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
	sum := 0.0
	for _, v := range tr[len(tr)-period:] {
		sum += v
	}
	atr := sum / float64(period)
	if closes[len(closes)-1] == 0 {
		return 1.0
	}
	return atr / closes[len(closes)-1] * 100
}

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

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}
