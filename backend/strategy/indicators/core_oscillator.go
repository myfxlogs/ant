package indicators

import (
	"math"

	"github.com/shopspring/decimal"
)

// ── Oscillator indicators (RSI, ATR, ADX, Stochastic, MFI, WPR, Momentum, CCI) ──

// RSI returns the Relative Strength Index using Wilder's smoothing.
// MT4/MT5: avgGain/avgLoss use Wilder smoothing, not simple average.
// appliedPrice: 1=close, 2=open, 3=high, 4=low, 5=median, 6=typical, 7=weighted.
func RSI(src BarSource, period, shift int, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	return decimal.NewFromFloat(rsiWilder(src, period, shift))
}

// rsiWilder computes RSI with Wilder's smoothing method.
// Full history: seed from oldest bars, smooth down to shift.
func rsiWilder(src BarSource, period, shift int) float64 {
	n := src.Len()
	if n < period+shift+1 {
		return 0
	}

	// Initial averages: simple average of oldest `period` gains/losses
	// Oldest price change is between bar n-1 and n-2 (n-1 has no prior)
	// MT4: first `period` changes are from bar n-period-1 vs n-period, ..., n-2 vs n-1
	var sumGain, sumLoss float64
	for i := n - 2; i >= n-period-1; i-- {
		if i < 0 {
			break
		}
		curr, _ := src.Close(i).Float64()
		prev, _ := src.Close(i + 1).Float64()
		diff := curr - prev
		if diff > 0 {
			sumGain += diff
		} else {
			sumLoss -= diff
		}
	}
	avgGain := sumGain / float64(period)
	avgLoss := sumLoss / float64(period)

	// Wilder smoothing from n-period-2 down to shift
	for i := n - period - 2; i >= shift; i-- {
		if i < 0 || i+1 >= n {
			continue
		}
		curr, _ := src.Close(i).Float64()
		prev, _ := src.Close(i + 1).Float64()
		diff := curr - prev
		gain := 0.0
		loss := 0.0
		if diff > 0 {
			gain = diff
		} else {
			loss = -diff
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
	}

	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// ATR returns the Average True Range using Wilder's smoothing.
// MT4/MT5: ATR uses Wilder smoothing, not simple average.
func ATR(src BarSource, period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(atrWilder(src, period, shift))
}

func atrWilder(src BarSource, period, shift int) float64 {
	n := src.Len()
	if n < period+shift+1 {
		return 0
	}

	// trueRange computes the True Range for bar at index i
	tr := func(i int) float64 {
		if i+1 >= n {
			h, _ := src.High(i).Float64()
			l, _ := src.Low(i).Float64()
			return h - l
		}
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		pc, _ := src.Close(i + 1).Float64()
		tr1 := h - l
		tr2 := h - pc
		if tr2 < 0 {
			tr2 = -tr2
		}
		tr3 := pc - l
		if tr3 < 0 {
			tr3 = -tr3
		}
		max := tr1
		if tr2 > max {
			max = tr2
		}
		if tr3 > max {
			max = tr3
		}
		return max
	}

	// Initial ATR: simple average of oldest `period` TRs (bars n-1 down to n-period)
	var sumTR float64
	for i := n - 1; i >= n-period; i-- {
		sumTR += tr(i)
	}
	atr := sumTR / float64(period)

	// Wilder smoothing from n-period-1 down to shift
	for i := n - period - 1; i >= shift; i-- {
		if i < 0 {
			break
		}
		atr = (atr*float64(period-1) + tr(i)) / float64(period)
	}
	return atr
}

// ADX returns the Average Directional Index using Wilder's smoothing.
// MT4/MT5: +DI, -DI, and ADX all use Wilder smoothing.
func ADX(src BarSource, period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(adxWilder(src, period, shift))
}

func adxWilder(src BarSource, period, shift int) float64 {
	n := src.Len()
	if n < period*2+shift {
		return 0
	}

	// dmTR computes +DM, -DM, and TR for bar at index i (needs bar i+1)
	dmTR := func(i int) (plusDM, minusDM, tr float64) {
		if i+1 >= n || i < 0 {
			return 0, 0, 0
		}
		h1, _ := src.High(i).Float64()
		h2, _ := src.High(i + 1).Float64()
		l1, _ := src.Low(i).Float64()
		l2, _ := src.Low(i + 1).Float64()
		pc, _ := src.Close(i + 1).Float64()

		pdm := h1 - h2
		mdm := l2 - l1
		if pdm < 0 {
			pdm = 0
		}
		if mdm < 0 {
			mdm = 0
		}
		if pdm > mdm {
			mdm = 0
		} else {
			pdm = 0
		}

		tr1 := h1 - l1
		tr2 := h1 - pc
		if tr2 < 0 {
			tr2 = -tr2
		}
		tr3 := pc - l1
		if tr3 < 0 {
			tr3 = -tr3
		}
		tr = tr1
		if tr2 > tr {
			tr = tr2
		}
		if tr3 > tr {
			tr = tr3
		}
		return pdm, mdm, tr
	}

	// Initial sums: simple sum of oldest `period` values (bars n-2 down to n-period-1)
	var sumPlusDM, sumMinusDM, sumTR float64
	for i := n - 2; i >= n-period-1; i-- {
		pdm, mdm, tr := dmTR(i)
		sumPlusDM += pdm
		sumMinusDM += mdm
		sumTR += tr
	}

	// Wilder smoothing from n-period-2 down to shift, collecting DX values
	var dxValues []float64
	for i := n - period - 2; i >= shift; i-- {
		pdm, mdm, tr := dmTR(i)
		sumTR = sumTR - sumTR/float64(period) + tr
		sumPlusDM = sumPlusDM - sumPlusDM/float64(period) + pdm
		sumMinusDM = sumMinusDM - sumMinusDM/float64(period) + mdm

		if sumTR == 0 {
			dxValues = append(dxValues, 0)
			continue
		}
		plusDI := 100 * sumPlusDM / sumTR
		minusDI := 100 * sumMinusDM / sumTR
		dx := math.Abs(plusDI - minusDI)
		dxValues = append(dxValues, dx)
	}

	if len(dxValues) < period {
		if len(dxValues) > 0 {
			return dxValues[len(dxValues)-1]
		}
		return 0
	}

	// ADX = Wilder smoothed average of DX
	// dxValues[0] = oldest DX, dxValues[len-1] = newest DX
	// Initial ADX: simple average of oldest `period` DX values (front of slice)
	var sumDX float64
	for i := 0; i < period && i < len(dxValues); i++ {
		sumDX += dxValues[i]
	}
	adx := sumDX / float64(period)

	// Wilder smoothing for remaining DX values (newer = toward end of slice)
	for i := period; i < len(dxValues); i++ {
		adx = (adx*float64(period-1) + dxValues[i]) / float64(period)
	}
	return adx
}

// Stochastic returns %K and %D values.
// MT4/MT5: %K is smoothed by `slowing` periods. %D is SMA of %K over dPeriod.
func Stochastic(src BarSource, kPeriod, dPeriod, slowing, shift int) (k, d decimal.Decimal) {
	n := src.Len()
	if n < kPeriod+slowing+shift {
		return decimal.NewFromInt(50), decimal.NewFromInt(50)
	}

	// rawK computes the raw %K for a given shift
	rawK := func(s int) float64 {
		if s+kPeriod > n {
			return 50
		}
		hh := highestHigh(src, kPeriod, s)
		ll := lowestLow(src, kPeriod, s)
		if hh == ll {
			return 50
		}
		cc, _ := src.Close(s).Float64()
		return (cc - ll) / (hh - ll) * 100
	}

	// %K = average of last `slowing` raw K values
	var kSum float64
	for i := 0; i < slowing; i++ {
		kSum += rawK(shift + i)
	}
	kVal := kSum / float64(slowing)

	// %D = SMA of %K over dPeriod
	var dSum float64
	for i := 0; i < dPeriod; i++ {
		// Recompute smoothed K for each shift in the D period
		var sk float64
		for j := 0; j < slowing; j++ {
			sk += rawK(shift + i + j)
		}
		dSum += sk / float64(slowing)
	}
	dVal := dSum / float64(dPeriod)

	return decimal.NewFromFloat(kVal), decimal.NewFromFloat(dVal)
}

// MFI returns the Money Flow Index.
// MT4/MT5: prevTP uses the previous bar's own H/L/C, not current bar's H/L with prev C.
func MFI(src BarSource, period, shift int) decimal.Decimal {
	n := src.Len()
	if n < period+shift+1 {
		return decimal.Zero
	}

	var posFlow, negFlow float64
	for i := shift; i < shift+period; i++ {
		if i+1 >= n {
			break
		}
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		c, _ := src.Close(i).Float64()
		tp := (h + l + c) / 3

		// Previous bar's typical price uses previous bar's H/L/C
		prevH, _ := src.High(i + 1).Float64()
		prevL, _ := src.Low(i + 1).Float64()
		prevC, _ := src.Close(i + 1).Float64()
		prevTP := (prevH + prevL + prevC) / 3

		flow := tp * float64(src.Volume(i))
		if tp > prevTP {
			posFlow += flow
		} else if tp < prevTP {
			negFlow += flow
		}
	}

	if negFlow == 0 {
		return decimal.NewFromInt(100)
	}
	mfr := posFlow / negFlow
	return decimal.NewFromFloat(100 - 100/(1+mfr))
}

// WPR returns Williams Percent Range.
// MT4/MT5: %R = (HH - Close) / (HH - LL) * -100
func WPR(src BarSource, period, shift int) decimal.Decimal {
	if src.Len() < period+shift {
		return decimal.Zero
	}
	hh := highestHigh(src, period, shift)
	ll := lowestLow(src, period, shift)
	cc, _ := src.Close(shift).Float64()
	if hh == ll {
		return decimal.NewFromInt(-50)
	}
	return decimal.NewFromFloat((hh - cc) / (hh - ll) * -100)
}

// Momentum returns Close[shift] - Close[shift+period].
// appliedPrice: 1=close, 2=open, 3=high, 4=low, 5=median, 6=typical, 7=weighted.
func Momentum(src BarSource, period, shift int, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	if src.Len() < period+shift {
		return decimal.Zero
	}
	curr, _ := src.Close(shift).Float64()
	prev, _ := src.Close(shift + period).Float64()
	return decimal.NewFromFloat(curr - prev)
}

// CCI returns the Commodity Channel Index.
// MT4/MT5: CCI = (TP - meanTP) / (0.015 * meanDeviation)
// appliedPrice: 1=close, 2=open, 3=high, 4=low, 5=median, 6=typical, 7=weighted.
func CCI(src BarSource, period, shift int, appliedPrice int) decimal.Decimal {
	if src.Len() < period+shift {
		return decimal.Zero
	}
	tp := func(i int) float64 { return selectPrice(src, appliedPrice, i) }
	var sumTP float64
	for i := shift; i < shift+period; i++ {
		sumTP += tp(i)
	}
	meanTP := sumTP / float64(period)
	var meanDev float64
	for i := shift; i < shift+period; i++ {
		dev := tp(i) - meanTP
		if dev < 0 {
			dev = -dev
		}
		meanDev += dev
	}
	meanDev /= float64(period)
	if meanDev == 0 {
		return decimal.Zero
	}
	currentTP := tp(shift)
	return decimal.NewFromFloat((currentTP - meanTP) / (0.015 * meanDev))
}
