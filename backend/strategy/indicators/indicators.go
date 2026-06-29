package indicators

import (
	"math"

	"github.com/shopspring/decimal"
)

// BarSource is the minimal interface needed for indicator calculations.
// Index 0 = most recent bar, increasing index = older bars.
type BarSource interface {
	Open(i int) decimal.Decimal
	High(i int) decimal.Decimal
	Low(i int) decimal.Decimal
	Close(i int) decimal.Decimal
	Volume(i int) int64
	Len() int
}

// ── Helper functions ────────────────────────────────────────────────

func sma(src BarSource, period, shift int) float64 {
	if src.Len() < period+shift {
		return 0
	}
	var sum float64
	for i := shift; i < shift+period; i++ {
		c, _ := src.Close(i).Float64()
		sum += c
	}
	return sum / float64(period)
}

func ema(src BarSource, period, shift int) float64 {
	n := src.Len()
	if n < period+shift {
		return 0
	}
	alpha := 2.0 / float64(period+1)
	var e float64
	for i := period + shift - 1; i >= shift; i-- {
		p, _ := src.Close(i).Float64()
		if i == period+shift-1 {
			e = p
		} else {
			e = p*alpha + e*(1-alpha)
		}
	}
	return e
}

func smma(src BarSource, period, shift int) float64 {
	n := src.Len()
	if n < period+shift {
		return 0
	}
	var sum float64
	for i := shift; i < shift+period; i++ {
		c, _ := src.Close(i).Float64()
		sum += c
	}
	prev := sum / float64(period)
	for i := shift + period; i < n && i < shift+period*2; i++ {
		c, _ := src.Close(i).Float64()
		prev = (prev*float64(period-1) + c) / float64(period)
	}
	return prev
}

func highestHigh(src BarSource, period, shift int) float64 {
	if src.Len() < period+shift {
		return 0
	}
	hh, _ := src.High(shift).Float64()
	for i := shift + 1; i < shift+period; i++ {
		h, _ := src.High(i).Float64()
		if h > hh {
			hh = h
		}
	}
	return hh
}

func lowestLow(src BarSource, period, shift int) float64 {
	if src.Len() < period+shift {
		return 0
	}
	ll, _ := src.Low(shift).Float64()
	for i := shift + 1; i < shift+period; i++ {
		l, _ := src.Low(i).Float64()
		if l < ll {
			ll = l
		}
	}
	return ll
}

// ── Shared MQL4/MQL5 indicators ─────────────────────────────────────

// Alligator returns jaw, teeth, lips values.
// SMMA with offsets: jaw(13,8), teeth(8,5), lips(5,3) are typical defaults.
func Alligator(src BarSource, jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, appliedPrice int, shift int) (jaw, teeth, lips decimal.Decimal) {
	jawVal := smma(src, jawPeriod, shift+jawShift)
	teethVal := smma(src, teethPeriod, shift+teethShift)
	lipsVal := smma(src, lipsPeriod, shift+lipsShift)
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
	if src.Len() < period+shift {
		return
	}
	mid := sma(src, period, shift)
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
	macdVal := ema(src, fastP, shift) - ema(src, slowP, shift)
	// Signal = SMA of MACD over signalP bars
	var sigSum float64
	for i := shift; i < shift+signalP; i++ {
		m := ema(src, fastP, i) - ema(src, slowP, i)
		sigSum += m
	}
	signal := sigSum / float64(signalP)
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

// Force returns Force Index = (Close[shift] - Close[shift+1]) * Volume[shift], smoothed by MA.
func Force(src BarSource, period int, method string, appliedPrice int, shift int) decimal.Decimal {
	if src.Len() < period+shift+1 {
		return decimal.Zero
	}
	var sum float64
	for i := shift; i < shift+period; i++ {
		if i+1 >= src.Len() {
			break
		}
		c, _ := src.Close(i).Float64()
		prevC, _ := src.Close(i + 1).Float64()
		vol := float64(src.Volume(i))
		sum += (c - prevC) * vol
	}
	return decimal.NewFromFloat(sum / float64(period))
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
	return decimal.NewFromFloat(jawF - teethF), decimal.NewFromFloat(teethF - lipsF)
}

// AC returns Accelerator Oscillator = AO - SMA(AO, 5).
func AC(src BarSource, shift int) decimal.Decimal {
	aoVal := AO(src, shift)
	aoF, _ := aoVal.Float64()
	// SMA of AO over 5 bars
	var sum float64
	for i := shift; i < shift+5; i++ {
		aoI, _ := AO(src, i).Float64()
		sum += aoI
	}
	smaAO := sum / 5.0
	return decimal.NewFromFloat(aoF - smaAO)
}

// AD returns Accumulation/Distribution line.
func AD(src BarSource, shift int) decimal.Decimal {
	n := src.Len()
	if n < shift+2 {
		return decimal.Zero
	}
	var ad float64
	for i := n - 1; i > shift; i-- {
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
	if src.Len() < period+shift {
		return decimal.Zero
	}
	low, _ := src.Low(shift).Float64()
	emaVal := ema(src, period, shift)
	return decimal.NewFromFloat(low - emaVal)
}

// BullsPower returns High - EMA(period).
func BullsPower(src BarSource, period, appliedPrice, shift int) decimal.Decimal {
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

// ── MQL5-only indicators ────────────────────────────────────────────

// AMA returns Adaptive Moving Average.
func AMA(src BarSource, period, fastP, slowP, shift int) decimal.Decimal {
	n := src.Len()
	if n < period+shift {
		return decimal.Zero
	}
	fastSC := 2.0 / float64(fastP+1)
	slowSC := 2.0 / float64(slowP+1)
	// Start with SMA
	amaVal := sma(src, period, shift+period-1)
	for i := shift + period - 1; i >= shift; i-- {
		if i+period >= n {
			continue
		}
		price, _ := src.Close(i).Float64()
		// Efficiency Ratio
		change := math.Abs(price - float64(src.Close(i+period).InexactFloat64()))
		var volatility float64
		for j := i; j < i+period; j++ {
			c1, _ := src.Close(j).Float64()
			c2, _ := src.Close(j + 1).Float64()
			volatility += math.Abs(c1 - c2)
		}
		er := 1.0
		if volatility > 0 {
			er = change / volatility
		}
		sc := math.Pow(er*(fastSC-slowSC)+slowSC, 2)
		amaVal = price + sc*(price-amaVal)
	}
	return decimal.NewFromFloat(amaVal)
}

// DEMA returns Double Exponential Moving Average = 2*EMA - EMA(EMA).
func DEMA(src BarSource, period, shift int) decimal.Decimal {
	if src.Len() < period*2+shift {
		return decimal.Zero
	}
	e1 := ema(src, period, shift)
	e2 := ema(src, period, shift+period)
	return decimal.NewFromFloat(2*e1 - e2)
}

// TEMA returns Triple Exponential Moving Average = 3*EMA - 3*EMA(EMA) + EMA(EMA(EMA)).
func TEMA(src BarSource, period, shift int) decimal.Decimal {
	if src.Len() < period*3+shift {
		return decimal.Zero
	}
	e1 := ema(src, period, shift)
	e2 := ema(src, period, shift+period)
	e3 := ema(src, period, shift+period*2)
	return decimal.NewFromFloat(3*e1 - 3*e2 + e3)
}

// FrAMA returns Fractal Adaptive Moving Average.
func FrAMA(src BarSource, period, shift int) decimal.Decimal {
	n := src.Len()
	if n < period+shift+2 {
		return decimal.Zero
	}
	// Simplified FrAMA: use EMA with adaptive alpha based on fractal dimension
	price, _ := src.Close(shift).Float64()
	prevPrice, _ := src.Close(shift + 1).Float64()

	// Compute fractal dimension approximation
	var range_ float64
	for i := shift; i < shift+period; i++ {
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		range_ += h - l
	}
	avgRange := range_ / float64(period)
	if avgRange == 0 {
		return decimal.NewFromFloat(prevPrice)
	}
	// Adaptive alpha: higher when range is small (trending), lower when large (ranging)
	alpha := math.Pow(float64(period), -0.5)
	if math.Abs(price-prevPrice) > avgRange {
		alpha = 2.0 / float64(period+1)
	}
	amaVal := prevPrice + alpha*(price-prevPrice)
	return decimal.NewFromFloat(amaVal)
}

// VIDyA returns Variable Index Dynamic Average.
func VIDyA(src BarSource, cmoP, cmoShift, maP, maShift, shift int) decimal.Decimal {
	n := src.Len()
	if n < cmoP+maP+shift {
		return decimal.Zero
	}
	// CMO (Chande Momentum Oscillator)
	var sumUp, sumDown float64
	for i := shift; i < shift+cmoP; i++ {
		if i+1 >= n {
			break
		}
		c, _ := src.Close(i).Float64()
		prevC, _ := src.Close(i + 1).Float64()
		diff := c - prevC
		if diff > 0 {
			sumUp += diff
		} else {
			sumDown -= diff
		}
	}
	cmo := 0.0
	if sumUp+sumDown > 0 {
		cmo = (sumUp - sumDown) / (sumUp + sumDown)
	}
	absCMO := math.Abs(cmo)
	// VIDyA = price * SC + prevVIDyA * (1 - SC), SC = alpha * |CMO|
	alpha := 2.0 / float64(maP+1)
	sc := alpha * absCMO
	price, _ := src.Close(shift).Float64()
	prevVIDyA := ema(src, maP, shift+1)
	return decimal.NewFromFloat(price*sc + prevVIDyA*(1-sc))
}

// TriX returns the Triple Exponential Average percentage change.
func TriX(src BarSource, period, shift int) decimal.Decimal {
	if src.Len() < period*3+shift+1 {
		return decimal.Zero
	}
	e3 := ema(src, period, shift+period*2)
	prevE3 := ema(src, period, shift+1+period*2)
	if prevE3 == 0 {
		return decimal.Zero
	}
	return decimal.NewFromFloat((e3 - prevE3) / prevE3 * 100)
}

// ADXWilder returns ADX with Wilder's smoothing.
func ADXWilder(src BarSource, period, shift int) decimal.Decimal {
	n := src.Len()
	if n < period*2+shift {
		return decimal.Zero
	}
	var prevPlusDM, prevMinusDM, prevTR float64
	// Initialize with simple sums
	for i := shift + period; i > shift; i-- {
		if i >= n {
			continue
		}
		high1, _ := src.High(i).Float64()
		high2, _ := src.High(i + 1).Float64()
		low1, _ := src.Low(i).Float64()
		low2, _ := src.Low(i + 1).Float64()
		prevClose, _ := src.Close(i + 1).Float64()

		plusDM := high1 - high2
		minusDM := low2 - low1
		if plusDM < 0 {
			plusDM = 0
		}
		if minusDM < 0 {
			minusDM = 0
		}
		if plusDM > minusDM {
			minusDM = 0
		} else {
			plusDM = 0
		}
		tr := high1 - low1
		if d := high1 - prevClose; d > tr {
			tr = d
		}
		if d := prevClose - low1; d > tr {
			tr = d
		}
		prevPlusDM += plusDM
		prevMinusDM += minusDM
		prevTR += tr
	}
	// Wilder smoothing
	for i := shift; i > 0; i-- {
		if i >= n || i+1 >= n {
			continue
		}
		high1, _ := src.High(i).Float64()
		high2, _ := src.High(i + 1).Float64()
		low1, _ := src.Low(i).Float64()
		low2, _ := src.Low(i + 1).Float64()
		prevClose, _ := src.Close(i + 1).Float64()

		plusDM := high1 - high2
		minusDM := low2 - low1
		if plusDM < 0 {
			plusDM = 0
		}
		if minusDM < 0 {
			minusDM = 0
		}
		if plusDM > minusDM {
			minusDM = 0
		} else {
			plusDM = 0
		}
		tr := high1 - low1
		if d := high1 - prevClose; d > tr {
			tr = d
		}
		if d := prevClose - low1; d > tr {
			tr = d
		}
		prevTR = prevTR - prevTR/float64(period) + tr
		prevPlusDM = prevPlusDM - prevPlusDM/float64(period) + plusDM
		prevMinusDM = prevMinusDM - prevMinusDM/float64(period) + minusDM
	}
	if prevTR == 0 {
		return decimal.Zero
	}
	plusDI := 100 * prevPlusDM / prevTR
	minusDI := 100 * prevMinusDM / prevTR
	dx := math.Abs(plusDI - minusDI)
	adx := dx // simplified: single smoothing pass
	return decimal.NewFromFloat(adx)
}

// Chaikin returns Chaikin Oscillator = EMA(MFV, fast) - EMA(MFV, slow).
func Chaikin(src BarSource, fastP, slowP, shift int) decimal.Decimal {
	n := src.Len()
	if n < slowP+shift+1 {
		return decimal.Zero
	}
	// MFV = ((Close-Low)-(High-Close))/(High-Low) * Volume
	mfv := func(i int) float64 {
		if i >= n {
			return 0
		}
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		c, _ := src.Close(i).Float64()
		vol := float64(src.Volume(i))
		hl := h - l
		if hl == 0 {
			return 0
		}
		return ((c - l) - (h - c)) / hl * vol
	}
	// EMA of MFV
	emaMFV := func(period, sh int) float64 {
		alpha := 2.0 / float64(period+1)
		var e float64
		for i := sh + period - 1; i >= sh; i-- {
			v := mfv(i)
			if i == sh+period-1 {
				e = v
			} else {
				e = v*alpha + e*(1-alpha)
			}
		}
		return e
	}
	fast := emaMFV(fastP, shift)
	slow := emaMFV(slowP, shift)
	return decimal.NewFromFloat(fast - slow)
}

// Volumes returns the volume at the given shift.
func Volumes(src BarSource, shift int) decimal.Decimal {
	if src.Len() < shift+1 {
		return decimal.Zero
	}
	return decimal.NewFromInt(src.Volume(shift))
}

// ── Exported helper functions for OnArray support ────────────────────
// These provide the same algorithms as the private helpers but are exported
// so the interpreter's *OnArray dispatch can call them with an arrayBarSource.

// SMAFloat returns the simple moving average as float64.
func SMAFloat(src BarSource, period, shift int) float64 {
	return sma(src, period, shift)
}

// EMAFloat returns the exponential moving average as float64.
func EMAFloat(src BarSource, period, shift int) float64 {
	return ema(src, period, shift)
}

// SMMAFloat returns the smoothed moving average as float64.
func SMMAFloat(src BarSource, period, shift int) float64 {
	return smma(src, period, shift)
}

// RSI computes RSI on a BarSource and returns float64.
func RSIFloat(src BarSource, period, shift int) float64 {
	if src.Len() < period+shift+1 {
		return 0
	}
	var avgGain, avgLoss float64
	for j := shift + 1; j <= shift+period; j++ {
		curr, _ := src.Close(j - 1).Float64()
		prev, _ := src.Close(j).Float64()
		diff := curr - prev
		if diff > 0 {
			avgGain += diff
		} else {
			avgLoss -= diff
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// StdDevFloat computes standard deviation on a BarSource and returns float64.
func StdDevFloat(src BarSource, period, shift int) float64 {
	if src.Len() < period+shift {
		return 0
	}
	var sum float64
	for j := shift; j < shift+period; j++ {
		c, _ := src.Close(j).Float64()
		sum += c
	}
	mean := sum / float64(period)
	var variance float64
	for j := shift; j < shift+period; j++ {
		c, _ := src.Close(j).Float64()
		d := c - mean
		variance += d * d
	}
	if period > 1 {
		return math.Sqrt(variance / float64(period - 1))
	}
	return 0
}

// Momentum computes momentum (Close[shift] - Close[shift+period]) on a BarSource.
func Momentum(src BarSource, period, shift int) decimal.Decimal {
	if src.Len() < period+shift {
		return decimal.Zero
	}
	curr, _ := src.Close(shift).Float64()
	prev, _ := src.Close(shift + period).Float64()
	return decimal.NewFromFloat(curr - prev)
}

// CCI computes Commodity Channel Index on a BarSource.
func CCI(src BarSource, period, shift int) decimal.Decimal {
	n := src.Len()
	if n < period+shift {
		return decimal.Zero
	}
	idx := n - 1 - shift
	var sumTP float64
	for j := idx; j > idx-period && j >= 0; j-- {
		h, _ := src.High(j).Float64()
		l, _ := src.Low(j).Float64()
		c, _ := src.Close(j).Float64()
		sumTP += (h + l + c) / 3
	}
	meanTP := sumTP / float64(period)
	var meanDev float64
	for j := idx; j > idx-period && j >= 0; j-- {
		h, _ := src.High(j).Float64()
		l, _ := src.Low(j).Float64()
		c, _ := src.Close(j).Float64()
		tp := (h + l + c) / 3
		dev := tp - meanTP
		if dev < 0 {
			dev = -dev
		}
		meanDev += dev
	}
	meanDev /= float64(period)
	if meanDev == 0 {
		return decimal.Zero
	}
	idx = n - 1 - shift
	h, _ := src.High(idx).Float64()
	l, _ := src.Low(idx).Float64()
	c, _ := src.Close(idx).Float64()
	tp := (h + l + c) / 3
	return decimal.NewFromFloat((tp - meanTP) / (0.015 * meanDev))
}

// MACD computes MACD line = EMA(fast) - EMA(slow) on a BarSource.
func MACD(src BarSource, fast, slow, shift int) decimal.Decimal {
	return decimal.NewFromFloat(ema(src, fast, shift) - ema(src, slow, shift))
}
