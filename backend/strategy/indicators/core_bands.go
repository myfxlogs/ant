package indicators

import (
	"math"

	"github.com/shopspring/decimal"
)

// ── Bands, StdDev, OBV, SAR indicators ──────────────────────────────

// BollingerBands returns upper, middle, lower bands.
// MT4/MT5: uses population standard deviation (divide by period, not period-1).
// appliedPrice: 1=close, 2=open, 3=high, 4=low, 5=median, 6=typical, 7=weighted.
func BollingerBands(src BarSource, period int, deviation decimal.Decimal, shift int, appliedPrice int) (upper, middle, lower decimal.Decimal) {
	src = withAppliedPrice(src, appliedPrice)
	if src.Len() < period+shift {
		return decimal.Zero, decimal.Zero, decimal.Zero
	}
	var sum float64
	for i := shift; i < shift+period; i++ {
		c, _ := src.Close(i).Float64()
		sum += c
	}
	mid := sum / float64(period)

	var variance float64
	for i := shift; i < shift+period; i++ {
		c, _ := src.Close(i).Float64()
		diff := c - mid
		variance += diff * diff
	}
	// Population standard deviation: divide by period, not period-1
	std := math.Sqrt(variance / float64(period))

	dev, _ := deviation.Float64()
	sd := std * dev
	midf := decimal.NewFromFloat(mid)
	sdf := decimal.NewFromFloat(sd)
	return midf.Add(sdf), midf, midf.Sub(sdf)
}

// StdDev returns the standard deviation.
// MT4/MT5: StdDev = sqrt(sum((close - MA)^2) / period) where MA uses the given method.
// Uses population standard deviation (divide by period, not period-1).
// appliedPrice: 1=close, 2=open, 3=high, 4=low, 5=median, 6=typical, 7=weighted.
func StdDev(src BarSource, period, shift int, method string, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	if src.Len() < period+shift {
		return decimal.Zero
	}
	// Compute MA values for each bar in the window using the given method
	maFunc := sma
	switch method {
	case "EMA", "ema":
		maFunc = ema
	case "SMMA", "smma":
		maFunc = smma
	case "LWMA", "lwma":
		maFunc = lwma
	}
	var variance float64
	for i := shift; i < shift+period; i++ {
		c, _ := src.Close(i).Float64()
		maVal := maFunc(src, period, i)
		diff := c - maVal
		variance += diff * diff
	}
	return decimal.NewFromFloat(math.Sqrt(variance / float64(period)))
}

// OBV returns On-Balance Volume.
// MT4/MT5: cumulative sum: if close > prevClose, +volume; if close < prevClose, -volume.
// appliedPrice: 1=close, 2=open, 3=high, 4=low, 5=median, 6=typical, 7=weighted.
func OBV(src BarSource, shift int, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	n := src.Len()
	if n < shift+2 {
		return decimal.Zero
	}
	var obv float64
	for i := n - 1; i > shift; i-- {
		curr, _ := src.Close(i - 1).Float64()
		prev, _ := src.Close(i).Float64()
		vol := float64(src.Volume(i - 1))
		if curr > prev {
			obv += vol
		} else if curr < prev {
			obv -= vol
		}
	}
	return decimal.NewFromFloat(obv)
}

// SAR returns the Parabolic SAR.
// MT4/MT5: recursive tracking of EP (extreme point) and AF (acceleration factor).
func SAR(src BarSource, step, maximum decimal.Decimal, shift int) decimal.Decimal {
	n := src.Len()
	if n < shift+2 {
		return decimal.Zero
	}

	accel, _ := step.Float64()
	maxVal, _ := maximum.Float64()

	// Start from the oldest bar and iterate forward to the requested shift.
	// We need to track trend direction, SAR, EP, and AF.

	// Determine initial direction from the two oldest bars in our window.
	oldest := n - 1
	if oldest < 1 {
		return decimal.Zero
	}

	// Start with an uptrend (SAR below price)
	isUp := true
	prevHigh, _ := src.High(oldest).Float64()
	prevLow, _ := src.Low(oldest).Float64()
	sar := prevLow
	ep := prevHigh
	af := accel

	// Iterate from oldest-1 down to shift
	for i := oldest - 1; i >= shift; i-- {
		high, _ := src.High(i).Float64()
		low, _ := src.Low(i).Float64()

		if isUp {
			// SAR for this bar
			newSAR := sar + af*(ep-sar)

			// SAR cannot exceed the previous two bars' lows
			if i+1 < n {
				prevLow2, _ := src.Low(i + 1).Float64()
				if newSAR > prevLow2 {
					newSAR = prevLow2
				}
			}
			if i+2 < n {
				prevLow3, _ := src.Low(i + 2).Float64()
				if newSAR > prevLow3 {
					newSAR = prevLow3
				}
			}

			// Check for reversal
			if low < newSAR {
				// Reverse to downtrend
				isUp = false
				sar = ep // SAR jumps to the old EP
				ep = low
				af = accel
			} else {
				sar = newSAR
				if high > ep {
					ep = high
					if af+accel <= maxVal {
						af += accel
					} else {
						af = maxVal
					}
				}
			}
		} else {
			// Downtrend: SAR above price
			newSAR := sar + af*(ep-sar)

			// SAR cannot be below the previous two bars' highs
			if i+1 < n {
				prevHigh2, _ := src.High(i + 1).Float64()
				if newSAR < prevHigh2 {
					newSAR = prevHigh2
				}
			}
			if i+2 < n {
				prevHigh3, _ := src.High(i + 2).Float64()
				if newSAR < prevHigh3 {
					newSAR = prevHigh3
				}
			}

			// Check for reversal
			if high > newSAR {
				// Reverse to uptrend
				isUp = true
				sar = ep
				ep = high
				af = accel
			} else {
				sar = newSAR
				if low < ep {
					ep = low
					if af+accel <= maxVal {
						af += accel
					} else {
						af = maxVal
					}
				}
			}
		}
	}

	return decimal.NewFromFloat(sar)
}
