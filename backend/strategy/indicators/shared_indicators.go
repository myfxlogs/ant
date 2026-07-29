package indicators

import (
	"github.com/shopspring/decimal"
)

// ── Shared MQL4/MQL5 indicators ─────────────────────────────────────

// Alligator returns jaw, teeth, lips values.
// SMMA with offsets: jaw(13,8), teeth(8,5), lips(5,3) are typical defaults.
func Alligator(src BarSource, jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, appliedPrice int, shift int) (jaw, teeth, lips decimal.Decimal) {
	src = withAppliedPrice(src, appliedPrice)
	var maFunc func(BarSource, int, int) float64
	switch method {
	case "SMA", "sma":
		maFunc = sma
	case maEMA, maEma:
		maFunc = ema
	case maSMMA, maSmma, "":
		maFunc = smma
	case maLWMA, maLwma:
		maFunc = lwma
	default:
		maFunc = smma
	}
	jawVal := maFunc(src, jawPeriod, shift+jawShift)
	teethVal := maFunc(src, teethPeriod, shift+teethShift)
	lipsVal := maFunc(src, lipsPeriod, shift+lipsShift)
	return decimal.NewFromFloat(jawVal), decimal.NewFromFloat(teethVal), decimal.NewFromFloat(lipsVal)
}

// Ichimoku returns tenkan, kijun, senkouA, senkouB.
func Ichimoku(src BarSource, tenkanP, kijunP, senkouP, shift int) (tenkan, kijun, senkouA, senkouB decimal.Decimal) {
	if src.Len() < senkouP+shift {
		return
	}
	tHH := highestHigh(src, tenkanP, shift)
	tLL := lowestLow(src, tenkanP, shift)
	tenkanVal := (tHH + tLL) / 2

	kHH := highestHigh(src, kijunP, shift)
	kLL := lowestLow(src, kijunP, shift)
	kijunVal := (kHH + kLL) / 2

	sA := (tenkanVal + kijunVal) / 2

	sHH := highestHigh(src, senkouP, shift)
	sLL := lowestLow(src, senkouP, shift)
	sB := (sHH + sLL) / 2

	return decimal.NewFromFloat(tenkanVal), decimal.NewFromFloat(kijunVal),
		decimal.NewFromFloat(sA), decimal.NewFromFloat(sB)
}

// Envelopes returns upper, lower bands around a moving average.
func Envelopes(src BarSource, period int, deviation decimal.Decimal, method string, appliedPrice int, shift int) (upper, lower decimal.Decimal) {
	src = withAppliedPrice(src, appliedPrice)
	if src.Len() < period+shift {
		return
	}
	var mid float64
	switch method {
	case maEMA, maEma:
		mid = ema(src, period, shift)
	case maSMMA, maSmma:
		mid = smma(src, period, shift)
	case maLWMA, maLwma:
		mid = lwma(src, period, shift)
	default:
		mid = sma(src, period, shift)
	}
	dev, _ := deviation.Float64()
	band := mid * dev / 100.0
	return decimal.NewFromFloat(mid + band), decimal.NewFromFloat(mid - band)
}

// DeMarker returns the DeMarker oscillator value.
func DeMarker(src BarSource, period, shift int) decimal.Decimal {
	if src.Len() < period+shift+1 {
		return decimal.Zero
	}
	var maxNum, minNum float64
	for i := shift; i < shift+period; i++ {
		if i+1 >= src.Len() {
			break
		}
		high, _ := src.High(i).Float64()
		prevHigh, _ := src.High(i + 1).Float64()
		low, _ := src.Low(i).Float64()
		prevLow, _ := src.Low(i + 1).Float64()

		dmMax := high - prevHigh
		if dmMax < 0 {
			dmMax = 0
		}
		dmMin := prevLow - low
		if dmMin < 0 {
			dmMin = 0
		}
		maxNum += dmMax
		minNum += dmMin
	}
	if maxNum+minNum == 0 {
		return decimal.NewFromFloat(0.5)
	}
	return decimal.NewFromFloat(maxNum / (maxNum + minNum))
}

// OsMA returns MACD histogram = MACD line - Signal line.
func OsMA(src BarSource, fastP, slowP, signalP, appliedPrice, shift int) decimal.Decimal {
	if src.Len() < slowP+shift {
		return decimal.Zero
	}
	pricedSrc := withAppliedPrice(src, appliedPrice)
	macdVal := ema(pricedSrc, fastP, shift) - ema(pricedSrc, slowP, shift)
	signal, _ := MACDSignal(src, fastP, slowP, signalP, shift, appliedPrice).Float64()
	return decimal.NewFromFloat(macdVal - signal)
}

// RVI returns Relative Vigor Index.
func RVI(src BarSource, period, shift int) decimal.Decimal {
	if src.Len() < period+shift+1 {
		return decimal.Zero
	}
	var num, den float64
	for i := shift; i < shift+period; i++ {
		if i+1 >= src.Len() {
			break
		}
		c, _ := src.Close(i).Float64()
		o, _ := src.Open(i).Float64()
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		num += (c - o)
		den += (h - l)
	}
	if den == 0 {
		return decimal.Zero
	}
	return decimal.NewFromFloat(num / den)
}

// Force returns Force Index = MA of (Close[shift] - Close[shift+1]) * Volume[shift].
// MT4/MT5: smoothing method is selectable (SMA/EMA/SMMA/LWMA).
func Force(src BarSource, period int, method string, appliedPrice int, shift int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	n := src.Len()
	if n < period+shift+1 {
		return decimal.Zero
	}
	// Build a temporary price source of raw force values for MA smoothing.
	forceVals := make([]float64, n)
	for i := n - 2; i >= 0; i-- {
		c, _ := src.Close(i).Float64()
		prevC, _ := src.Close(i + 1).Float64()
		vol := float64(src.Volume(i))
		forceVals[i] = (c - prevC) * vol
	}
	tmpSrc := &sliceBarSource{closes: forceVals}
	switch method {
	case maEMA, maEma:
		return decimal.NewFromFloat(ema(tmpSrc, period, shift))
	case maSMMA, maSmma:
		return decimal.NewFromFloat(smma(tmpSrc, period, shift))
	case maLWMA, maLwma:
		return decimal.NewFromFloat(lwma(tmpSrc, period, shift))
	default:
		return decimal.NewFromFloat(sma(tmpSrc, period, shift))
	}
}

// Fractals returns upper, lower fractal values.
// A 5-bar fractal: middle bar has highest high (upper) or lowest low (lower).
func Fractals(src BarSource, shift int) (upper, lower decimal.Decimal) {
	n := src.Len()
	if n < shift+3 {
		return
	}
	idx := shift + 2 // middle of 5-bar window
	if idx >= n {
		return
	}
	midHigh, _ := src.High(idx).Float64()
	midLow, _ := src.Low(idx).Float64()

	isUpper := true
	isLower := true
	for i := idx - 2; i <= idx+2; i++ {
		if i < 0 || i >= n || i == idx {
			continue
		}
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		if h >= midHigh {
			isUpper = false
		}
		if l <= midLow {
			isLower = false
		}
	}
	if isUpper {
		upper = decimal.NewFromFloat(midHigh)
	}
	if isLower {
		lower = decimal.NewFromFloat(midLow)
	}
	return
}

// Gator returns the Gator oscillator: |jaw-teeth| (upper) and |teeth-lips| (lower).
func Gator(src BarSource, jawP, jawS, teethP, teethS, lipsP, lipsS int, method string, appliedPrice int, shift int) (upper, lower decimal.Decimal) {
	jaw, teeth, lips := Alligator(src, jawP, jawS, teethP, teethS, lipsP, lipsS, method, appliedPrice, shift)
	jawF, _ := jaw.Float64()
	teethF, _ := teeth.Float64()
	lipsF, _ := lips.Float64()
	u := jawF - teethF
	if u < 0 {
		u = -u
	}
	l := teethF - lipsF
	if l < 0 {
		l = -l
	}
	return decimal.NewFromFloat(u), decimal.NewFromFloat(l)
}

// AC returns Accelerator Oscillator = AO - SMA(AO, 5).
// AO = SMA(median, 5) - SMA(median, 34), median = (H+L)/2.
func AC(src BarSource, shift int) decimal.Decimal {
	n := src.Len()
	if n < 34+shift+5 {
		return decimal.Zero
	}
	// median[i] = (High[i] + Low[i]) / 2
	median := func(i int) float64 {
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		return (h + l) / 2
	}
	// Compute AO for shifts [shift, shift+5) in one pass
	var aoVals [5]float64
	for k := 0; k < 5; k++ {
		s := shift + k
		var sum5, sum34 float64
		for i := s; i < s+5; i++ {
			sum5 += median(i)
		}
		for i := s; i < s+34; i++ {
			sum34 += median(i)
		}
		aoVals[k] = sum5/5.0 - sum34/34.0
	}
	smaAO := (aoVals[0] + aoVals[1] + aoVals[2] + aoVals[3] + aoVals[4]) / 5.0
	return decimal.NewFromFloat(aoVals[0] - smaAO)
}

// AD returns Accumulation/Distribution line.
func AD(src BarSource, shift int) decimal.Decimal {
	n := src.Len()
	if n < shift+1 {
		return decimal.Zero
	}
	var ad float64
	for i := n - 1; i >= shift; i-- {
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		c, _ := src.Close(i).Float64()
		vol := float64(src.Volume(i))
		hl := h - l
		if hl == 0 {
			continue
		}
		mfv := ((c - l) - (h - c)) / hl * vol
		ad += mfv
	}
	return decimal.NewFromFloat(ad)
}

// AO returns Awesome Oscillator = SMA(median, 5) - SMA(median, 34).
func AO(src BarSource, shift int) decimal.Decimal {
	n := src.Len()
	if n < 34+shift {
		return decimal.Zero
	}
	var sum5, sum34 float64
	for i := shift; i < shift+5; i++ {
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		sum5 += (h + l) / 2
	}
	for i := shift; i < shift+34; i++ {
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		sum34 += (h + l) / 2
	}
	return decimal.NewFromFloat(sum5/5.0 - sum34/34.0)
}

// BearsPower returns Low - EMA(period).
func BearsPower(src BarSource, period, appliedPrice, shift int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	if src.Len() < period+shift {
		return decimal.Zero
	}
	low, _ := src.Low(shift).Float64()
	emaVal := ema(src, period, shift)
	return decimal.NewFromFloat(low - emaVal)
}

// BullsPower returns High - EMA(period).
func BullsPower(src BarSource, period, appliedPrice, shift int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	if src.Len() < period+shift {
		return decimal.Zero
	}
	high, _ := src.High(shift).Float64()
	emaVal := ema(src, period, shift)
	return decimal.NewFromFloat(high - emaVal)
}

// BWMFI returns Bill Williams Market Facilitation Index = (High - Low) / Volume.
func BWMFI(src BarSource, shift int) decimal.Decimal {
	if src.Len() < shift+1 {
		return decimal.Zero
	}
	h, _ := src.High(shift).Float64()
	l, _ := src.Low(shift).Float64()
	vol := float64(src.Volume(shift))
	if vol == 0 {
		return decimal.Zero
	}
	return decimal.NewFromFloat((h - l) / vol)
}
