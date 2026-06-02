package indicator

import (
	"math"

	"github.com/shopspring/decimal"
)

// shared math utilities used by indicator implementations.
// All functions operate on decimal.Decimal for precision.

// sma computes the simple moving average of the last n values.
func sma(values []decimal.Decimal, n int) []decimal.Decimal {
	out := make([]decimal.Decimal, len(values))
	if n <= 0 || len(values) == 0 {
		return out
	}
	var sum decimal.Decimal
	for i, v := range values {
		sum = sum.Add(v)
		if i >= n {
			sum = sum.Sub(values[i-n])
		}
		divisor := decimal.NewFromInt(int64(minInt(i+1, n)))
		if !divisor.IsZero() {
			out[i] = sum.Div(divisor)
		}
	}
	return out
}

// ema computes the exponential moving average (α = 2/(n+1)).
func ema(values []decimal.Decimal, n int) []decimal.Decimal {
	out := make([]decimal.Decimal, len(values))
	if n <= 0 || len(values) == 0 {
		return out
	}
	alpha := decimal.NewFromFloat(2.0 / float64(n+1))
	oneMinusAlpha := decimal.NewFromInt(1).Sub(alpha)
	// Seed with SMA of first n values, or first available value.
	seed := safeFirst(values)
	if len(values) >= n {
		seed = sma(values[:n], n)[n-1]
	}
	prev := seed
	for i := range values {
		if i == 0 {
			out[i] = seed
			continue
		}
		emaVal := values[i].Mul(alpha).Add(prev.Mul(oneMinusAlpha))
		out[i] = emaVal
		prev = emaVal
	}
	return out
}

// stddev returns the population standard deviation of values over window n.
func stddev(values []decimal.Decimal, n int) []decimal.Decimal {
	out := make([]decimal.Decimal, len(values))
	if n <= 1 || len(values) == 0 {
		return out
	}
	for i := n - 1; i < len(values); i++ {
		slice := values[i-n+1 : i+1]
		avg := sma(slice, n)[n-1]
		var sumSq decimal.Decimal
		for _, v := range slice {
			diff := v.Sub(avg)
			sumSq = sumSq.Add(diff.Mul(diff))
		}
		variance := sumSq.Div(decimal.NewFromInt(int64(n)))
		f, _ := variance.Float64()
		out[i] = decimal.NewFromFloat(math.Sqrt(f))
	}
	return out
}

// highest returns the rolling maximum over window n.
func highest(values []decimal.Decimal, n int) []decimal.Decimal {
	return rollingExtreme(values, n, true)
}

// lowest returns the rolling minimum over window n.
func lowest(values []decimal.Decimal, n int) []decimal.Decimal {
	return rollingExtreme(values, n, false)
}

func rollingExtreme(values []decimal.Decimal, n int, isMax bool) []decimal.Decimal {
	out := make([]decimal.Decimal, len(values))
	if n <= 0 || len(values) == 0 {
		return out
	}
	for i := range values {
		start := i - n + 1
		if start < 0 {
			start = 0
		}
		extreme := values[start]
		for j := start + 1; j <= i; j++ {
			if (isMax && values[j].GreaterThan(extreme)) || (!isMax && values[j].LessThan(extreme)) {
				extreme = values[j]
			}
		}
		out[i] = extreme
	}
	return out
}

// trueRange computes the true range series from bars.
func trueRange(bars []BarOHLCV) []decimal.Decimal {
	out := make([]decimal.Decimal, len(bars))
	if len(bars) == 0 {
		return out
	}
	out[0] = bars[0].High.Sub(bars[0].Low)
	for i := 1; i < len(bars); i++ {
		hl := bars[i].High.Sub(bars[i].Low)
		hc := bars[i].High.Sub(bars[i-1].Close).Abs()
		lc := bars[i].Low.Sub(bars[i-1].Close).Abs()
		out[i] = decimalMax(hl, decimalMax(hc, lc))
	}
	return out
}

// BarOHLCV is a simplified OHLCV view used by indicators.
type BarOHLCV struct {
	Open, High, Low, Close decimal.Decimal
	Volume                 float64
}

func barsToOHLCV(bars []BarOHLCV) (opens, highs, lows, closes, volumes []decimal.Decimal) {
	for _, b := range bars {
		opens = append(opens, b.Open)
		highs = append(highs, b.High)
		lows = append(lows, b.Low)
		closes = append(closes, b.Close)
		volumes = append(volumes, decimal.NewFromFloat(b.Volume))
	}
	return
}

func safeFirst(v []decimal.Decimal) decimal.Decimal {
	if len(v) == 0 {
		return decimal.Zero
	}
	return v[0]
}

func decimalMax(a, b decimal.Decimal) decimal.Decimal {
	if a.GreaterThan(b) {
		return a
	}
	return b
}

func decimalMin(a, b decimal.Decimal) decimal.Decimal {
	if a.LessThan(b) {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
