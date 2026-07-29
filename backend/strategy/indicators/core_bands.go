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
	case maEMA, maEma:
		maFunc = ema
	case maSMMA, maSmma:
		maFunc = smma
	case maLWMA, maLwma:
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

	oldest := n - 1
	if oldest < 1 {
		return decimal.Zero
	}

	isUp := true
	prevHigh, _ := src.High(oldest).Float64()
	prevLow, _ := src.Low(oldest).Float64()
	sar := prevLow
	ep := prevHigh
	af := accel

	for i := oldest - 1; i >= shift; i-- {
		high, _ := src.High(i).Float64()
		low, _ := src.Low(i).Float64()

		if isUp {
			isUp, sar, ep, af = sarUptrendStep(src, i, n, sar, ep, af, accel, maxVal, high, low)
		} else {
			isUp, sar, ep, af = sarDowntrendStep(src, i, n, sar, ep, af, accel, maxVal, high, low)
		}
	}

	return decimal.NewFromFloat(sar)
}

func sarUptrendStep(src BarSource, i, n int, sar, ep, af, accel, maxVal, high, low float64) (isUp bool, newSAR, newEP, newAF float64) {
	newSAR = sar + af*(ep-sar)
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
	if low < newSAR {
		return false, ep, low, accel
	}
	if high > ep {
		newEP = high
	} else {
		newEP = ep
	}
	if newEP != ep || high > ep {
		if af+accel <= maxVal {
			newAF = af + accel
		} else {
			newAF = maxVal
		}
	} else {
		newAF = af
	}
	return true, newSAR, newEP, newAF
}

func sarDowntrendStep(src BarSource, i, n int, sar, ep, af, accel, maxVal, high, low float64) (isUp bool, newSAR, newEP, newAF float64) {
	newSAR = sar + af*(ep-sar)
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
	if high > newSAR {
		return true, ep, high, accel
	}
	if low < ep {
		newEP = low
	} else {
		newEP = ep
	}
	if newEP != ep || low < ep {
		if af+accel <= maxVal {
			newAF = af + accel
		} else {
			newAF = maxVal
		}
	} else {
		newAF = af
	}
	return false, newSAR, newEP, newAF
}
