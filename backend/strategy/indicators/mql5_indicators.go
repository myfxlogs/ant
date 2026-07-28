package indicators

import (
	"math"

	"github.com/shopspring/decimal"
)

// ── MQL5-only indicators ────────────────────────────────────────────

// AMA returns Adaptive Moving Average.
func AMA(src BarSource, period, fastP, slowP, shift int, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
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
		tmp := er*(fastSC-slowSC) + slowSC
		sc := tmp * tmp
		amaVal = price + sc*(price-amaVal)
	}
	return decimal.NewFromFloat(amaVal)
}

// DEMA returns Double Exponential Moving Average = 2*EMA - EMA(EMA).
// Correctly computes EMA of EMA by building an EMA series then smoothing it.
func DEMA(src BarSource, period, shift int, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	n := src.Len()
	if n < period*2+shift {
		return decimal.Zero
	}
	alpha := 2.0 / float64(period+1)
	// Build EMA series over full history (oldest to newest)
	var seedSum float64
	for i := n - 1; i >= n-period; i-- {
		c, _ := src.Close(i).Float64()
		seedSum += c
	}
	ema1 := seedSum / float64(period)
	emaVals := make([]float64, 0, n)
	emaVals = append(emaVals, ema1)
	for i := n - period - 1; i >= 0; i-- {
		c, _ := src.Close(i).Float64()
		ema1 = c*alpha + ema1*(1-alpha)
		emaVals = append(emaVals, ema1)
	}
	// EMA of EMA: seed with SMA of oldest `period` EMA values
	if len(emaVals) < period+shift {
		return decimal.Zero
	}
	var e2Seed float64
	for i := 0; i < period; i++ {
		e2Seed += emaVals[i]
	}
	ema2 := e2Seed / float64(period)
	targetIdx := len(emaVals) - 1 - shift
	for i := period; i <= targetIdx; i++ {
		ema2 = emaVals[i]*alpha + ema2*(1-alpha)
	}
	return decimal.NewFromFloat(2*emaVals[targetIdx] - ema2)
}

// TEMA returns Triple Exponential Moving Average = 3*EMA - 3*EMA(EMA) + EMA(EMA(EMA)).
// Correctly computes triple nested EMA by building successive EMA series.
func TEMA(src BarSource, period, shift int, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	n := src.Len()
	if n < period*3+shift {
		return decimal.Zero
	}
	alpha := 2.0 / float64(period+1)
	// Build EMA1 series (oldest to newest)
	var seedSum float64
	for i := n - 1; i >= n-period; i-- {
		c, _ := src.Close(i).Float64()
		seedSum += c
	}
	ema1 := seedSum / float64(period)
	ema1Vals := make([]float64, 0, n)
	ema1Vals = append(ema1Vals, ema1)
	for i := n - period - 1; i >= 0; i-- {
		c, _ := src.Close(i).Float64()
		ema1 = c*alpha + ema1*(1-alpha)
		ema1Vals = append(ema1Vals, ema1)
	}
	// Build EMA2 = EMA of EMA1
	if len(ema1Vals) < period {
		return decimal.Zero
	}
	var e2Seed float64
	for i := 0; i < period; i++ {
		e2Seed += ema1Vals[i]
	}
	ema2 := e2Seed / float64(period)
	ema2Vals := make([]float64, 0, len(ema1Vals)-period+1)
	ema2Vals = append(ema2Vals, ema2)
	for i := period; i < len(ema1Vals); i++ {
		ema2 = ema1Vals[i]*alpha + ema2*(1-alpha)
		ema2Vals = append(ema2Vals, ema2)
	}
	// Build EMA3 = EMA of EMA2
	if len(ema2Vals) < period+shift {
		return decimal.Zero
	}
	var e3Seed float64
	for i := 0; i < period; i++ {
		e3Seed += ema2Vals[i]
	}
	ema3 := e3Seed / float64(period)
	targetIdx2 := len(ema2Vals) - 1 - shift
	for i := period; i <= targetIdx2; i++ {
		ema3 = ema2Vals[i]*alpha + ema3*(1-alpha)
	}
	// EMA1 at shift: index in ema1Vals
	targetIdx1 := len(ema1Vals) - 1 - shift
	// EMA2 at shift: index in ema2Vals
	return decimal.NewFromFloat(3*ema1Vals[targetIdx1] - 3*ema2Vals[targetIdx2] + ema3)
}

// FrAMA returns Fractal Adaptive Moving Average (John Ehlers).
// Uses fractal dimension D = (log(N1+N2) - log(N3)) / log(2) where
// N1, N2 are half-period ranges and N3 is full-period range.
// alpha = exp(-D), FrAMA = alpha*price + (1-alpha)*prevFrAMA (recursive).
func FrAMA(src BarSource, period, shift int, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	n := src.Len()
	if n < period+shift+1 {
		return decimal.Zero
	}
	half := period / 2
	if half < 1 {
		half = 1
	}

	// Seed: price at the oldest bar where we can compute the dimension.
	startIdx := n - period
	if startIdx < shift {
		return decimal.Zero
	}
	price, _ := src.Close(startIdx).Float64()
	frama := price

	// Iterate from startIdx-1 down to shift (oldest to newest).
	for i := startIdx - 1; i >= shift; i-- {
		hh1 := highestHigh(src, half, i)
		ll1 := lowestLow(src, half, i)
		hh2 := highestHigh(src, period-half, i+half)
		ll2 := lowestLow(src, period-half, i+half)
		hhFull := highestHigh(src, period, i)
		llFull := lowestLow(src, period, i)

		n1 := (hh1 - ll1) / float64(half)
		n2 := (hh2 - ll2) / float64(period-half)
		n3 := (hhFull - llFull) / float64(period)

		var d float64
		if n1+n2 > 0 && n3 > 0 {
			d = (math.Log(n1+n2) - math.Log(n3)) / math.Log(2)
		} else {
			d = 2
		}

		alpha := math.Exp(-d)
		if alpha < 0 {
			alpha = 0
		} else if alpha > 1 {
			alpha = 1
		}

		price, _ := src.Close(i).Float64()
		frama = alpha*price + (1-alpha)*frama
	}
	return decimal.NewFromFloat(frama)
}

// VIDyA returns Variable Index Dynamic Average.
// Recursive: VIDyA[i] = price[i]*SC(i) + VIDyA[i+1]*(1-SC(i))
// where SC = (2/(maP+1)) * |CMO(i)| and CMO is Chande Momentum Oscillator.
func VIDyA(src BarSource, cmoP, cmoShift, maP, maShift, shift int, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	n := src.Len()
	// Need enough bars: cmoP for CMO at the oldest bar + 1 for seed
	startIdx := n - cmoP - cmoShift - 1
	if startIdx < shift {
		return decimal.Zero
	}

	alpha := 2.0 / float64(maP+1)

	// Seed: price at the oldest bar
	price, _ := src.Close(startIdx).Float64()
	vidya := price

	// Iterate from startIdx-1 down to shift (oldest to newest)
	for i := startIdx - 1; i >= shift; i-- {
		// CMO at bar i+cmoShift
		cmoIdx := i + cmoShift
		if cmoIdx+cmoP >= n {
			continue
		}
		var sumUp, sumDown float64
		for j := cmoIdx; j < cmoIdx+cmoP; j++ {
			if j+1 >= n {
				break
			}
			c, _ := src.Close(j).Float64()
			prevC, _ := src.Close(j + 1).Float64()
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
		sc := alpha * math.Abs(cmo)
		price, _ := src.Close(i).Float64()
		vidya = price*sc + vidya*(1-sc)
	}
	return decimal.NewFromFloat(vidya)
}

// TriX returns the Triple Exponential Average percentage change.
// TriX = (EMA3[shift] - EMA3[shift+1]) / EMA3[shift+1] * 100
// where EMA3 = EMA(EMA(EMA(close))).
func TriX(src BarSource, period, shift int, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	n := src.Len()
	if n < period*3+shift+1 {
		return decimal.Zero
	}
	alpha := 2.0 / float64(period+1)
	// Build EMA1 series (oldest to newest)
	var seedSum float64
	for i := n - 1; i >= n-period; i-- {
		c, _ := src.Close(i).Float64()
		seedSum += c
	}
	ema1 := seedSum / float64(period)
	ema1Vals := make([]float64, 0, n)
	ema1Vals = append(ema1Vals, ema1)
	for i := n - period - 1; i >= 0; i-- {
		c, _ := src.Close(i).Float64()
		ema1 = c*alpha + ema1*(1-alpha)
		ema1Vals = append(ema1Vals, ema1)
	}
	// Build EMA2 = EMA of EMA1
	if len(ema1Vals) < period {
		return decimal.Zero
	}
	var e2Seed float64
	for i := 0; i < period; i++ {
		e2Seed += ema1Vals[i]
	}
	ema2 := e2Seed / float64(period)
	ema2Vals := make([]float64, 0, len(ema1Vals)-period+1)
	ema2Vals = append(ema2Vals, ema2)
	for i := period; i < len(ema1Vals); i++ {
		ema2 = ema1Vals[i]*alpha + ema2*(1-alpha)
		ema2Vals = append(ema2Vals, ema2)
	}
	// Build EMA3 = EMA of EMA2
	if len(ema2Vals) < period+shift+1 {
		return decimal.Zero
	}
	var e3Seed float64
	for i := 0; i < period; i++ {
		e3Seed += ema2Vals[i]
	}
	ema3 := e3Seed / float64(period)
	ema3Vals := make([]float64, 0, len(ema2Vals)-period+1)
	ema3Vals = append(ema3Vals, ema3)
	for i := period; i < len(ema2Vals); i++ {
		ema3 = ema2Vals[i]*alpha + ema3*(1-alpha)
		ema3Vals = append(ema3Vals, ema3)
	}
	idxShift := len(ema3Vals) - 1 - shift
	idxPrev := idxShift - 1
	if idxPrev < 0 || ema3Vals[idxPrev] == 0 {
		return decimal.Zero
	}
	return decimal.NewFromFloat((ema3Vals[idxShift] - ema3Vals[idxPrev]) / ema3Vals[idxPrev] * 100)
}

// ADXWilder returns ADX with Wilder's smoothing (full history).
func ADXWilder(src BarSource, period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(adxWilder(src, period, shift))
}

// Chaikin returns Chaikin Oscillator = EMA(MFV, fast) - EMA(MFV, slow).
func Chaikin(src BarSource, fastP, slowP, shift int) decimal.Decimal {
	n := src.Len()
	if n < slowP+shift+1 {
		return decimal.Zero
	}
	// MFV = ((Close-Low)-(High-Close))/(High-Low) * Volume
	mfv := func(i int) float64 {
		if i >= n || i < 0 {
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
	// Full-history EMA with SMA seed, oldest to newest
	emaMFV := func(period, sh int) float64 {
		if n < period+sh {
			return 0
		}
		alpha := 2.0 / float64(period+1)
		// Seed: SMA of oldest `period` MFV values (indices n-1 down to n-period)
		var seedSum float64
		for i := n - 1; i >= n-period; i-- {
			seedSum += mfv(i)
		}
		e := seedSum / float64(period)
		// Smooth from n-period-1 down to sh
		for i := n - period - 1; i >= sh; i-- {
			e = mfv(i)*alpha + e*(1-alpha)
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
