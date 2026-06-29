package backtest

import (
	"math"

	"github.com/shopspring/decimal"

	"anttrader/strategy/indicators"
	"anttrader/strategy/sdk"
)

// ── Indicators (backtest implementation) ──────────────────────────

type btIndicators struct{ bars []sdk.Bar }

func (i *btIndicators) EMA(period, shift int) decimal.Decimal {
	if len(i.bars) < period+shift {
		return decimal.Zero
	}
	alpha := 2.0 / float64(period+1)
	var ema float64
	for j := period + shift - 1; j >= shift; j-- {
		p, _ := i.bars[j].Close.Float64()
		if j == period+shift-1 {
			ema = p
		} else {
			ema = p*alpha + ema*(1-alpha)
		}
	}
	return decimal.NewFromFloat(ema)
}

func (i *btIndicators) MA(period, shift int, method string) decimal.Decimal {
	return i.EMA(period, shift)
}

func (i *btIndicators) RSI(period, shift int) decimal.Decimal {
	if len(i.bars) < period+shift+1 {
		return decimal.Zero
	}
	var avgGain, avgLoss float64
	for j := shift + 1; j <= shift+period; j++ {
		curr, _ := i.bars[j-1].Close.Float64()
		prev, _ := i.bars[j].Close.Float64()
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
		return decimal.NewFromInt(100)
	}
	rs := avgGain / avgLoss
	return decimal.NewFromFloat(100 - (100 / (1 + rs)))
}

func (i *btIndicators) MACD(fast, slow, signal, shift int) decimal.Decimal {
	return i.EMA(fast, shift).Sub(i.EMA(slow, shift))
}

func (i *btIndicators) MACDSignal(fast, slow, signal, shift int) decimal.Decimal {
	return i.MACD(fast, slow, signal, shift)
}

func (i *btIndicators) ATR(period, shift int) decimal.Decimal {
	if len(i.bars) < period+shift+1 {
		return decimal.Zero
	}
	var sumTR float64
	for j := shift; j < shift+period; j++ {
		high, _ := i.bars[j].High.Float64()
		low, _ := i.bars[j].Low.Float64()
		prevClose, _ := i.bars[j+1].Close.Float64()
		tr := high - low
		if d := high - prevClose; d > tr {
			tr = d
		} else if -d > tr {
			tr = -d
		}
		if d := prevClose - low; d > tr {
			tr = d
		} else if -d > tr {
			tr = -d
		}
		sumTR += tr
	}
	return decimal.NewFromFloat(sumTR / float64(period))
}

func (i *btIndicators) Bollinger(period int, deviation decimal.Decimal, shift int) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	if len(i.bars) < period+shift {
		return decimal.Zero, decimal.Zero, decimal.Zero
	}
	var sum float64
	for j := shift; j < shift+period; j++ {
		c, _ := i.bars[j].Close.Float64()
		sum += c
	}
	middle := sum / float64(period)
	var variance float64
	for j := shift; j < shift+period; j++ {
		c, _ := i.bars[j].Close.Float64()
		d := c - middle
		variance += d * d
	}
	std := 0.0
	if period > 1 {
		std = math.Sqrt(variance / float64(period-1))
	}
	dev, _ := deviation.Float64()
	mid := decimal.NewFromFloat(middle)
	sd := decimal.NewFromFloat(std * dev)
	return mid.Add(sd), mid, mid.Sub(sd)
}

func (i *btIndicators) Momentum(period, shift int) decimal.Decimal {
	if len(i.bars) < period+shift {
		return decimal.Zero
	}
	curr, _ := i.bars[shift].Close.Float64()
	prev, _ := i.bars[shift+period].Close.Float64()
	return decimal.NewFromFloat(curr - prev)
}

func (i *btIndicators) StdDev(period, shift int) decimal.Decimal {
	if len(i.bars) < period+shift {
		return decimal.Zero
	}
	var sum float64
	for j := shift; j < shift+period; j++ {
		c, _ := i.bars[j].Close.Float64()
		sum += c
	}
	mean := sum / float64(period)
	var variance float64
	for j := shift; j < shift+period; j++ {
		c, _ := i.bars[j].Close.Float64()
		d := c - mean
		variance += d * d
	}
	if period > 1 {
		return decimal.NewFromFloat(math.Sqrt(variance / float64(period-1)))
	}
	return decimal.Zero
}

func (i *btIndicators) Stochastic(kPeriod, dPeriod, slowing, shift int) (decimal.Decimal, decimal.Decimal) {
	n := len(i.bars)
	if n < kPeriod+shift {
		return decimal.NewFromInt(50), decimal.NewFromInt(50)
	}
	idx := n - 1 - shift
	var highestHigh, lowestLow float64
	for j := idx; j > idx-kPeriod && j >= 0; j-- {
		h, _ := i.bars[j].High.Float64()
		l, _ := i.bars[j].Low.Float64()
		if j == idx {
			highestHigh = h
			lowestLow = l
		} else {
			if h > highestHigh {
				highestHigh = h
			}
			if l < lowestLow {
				lowestLow = l
			}
		}
	}
	currentClose, _ := i.bars[idx].Close.Float64()
	kVal := 50.0
	if highestHigh != lowestLow {
		kVal = (currentClose - lowestLow) / (highestHigh - lowestLow) * 100
	}
	var kSum float64
	for d := 0; d < dPeriod; d++ {
		di := idx - d
		if di < 0 || di-kPeriod+1 < 0 {
			break
		}
		var hh, ll float64
		for j := di; j > di-kPeriod && j >= 0; j-- {
			h, _ := i.bars[j].High.Float64()
			l, _ := i.bars[j].Low.Float64()
			if j == di {
				hh = h
				ll = l
			} else {
				if h > hh {
					hh = h
				}
				if l < ll {
					ll = l
				}
			}
		}
		cc, _ := i.bars[di].Close.Float64()
		if hh == ll {
			kSum += 50
		} else {
			kSum += (cc - ll) / (hh - ll) * 100
		}
	}
	return decimal.NewFromFloat(kVal), decimal.NewFromFloat(kSum / float64(dPeriod))
}

func (i *btIndicators) CCI(period, shift int) decimal.Decimal {
	n := len(i.bars)
	if n < period+shift {
		return decimal.Zero
	}
	idx := n - 1 - shift
	var sumTP float64
	for j := idx; j > idx-period && j >= 0; j-- {
		h, _ := i.bars[j].High.Float64()
		l, _ := i.bars[j].Low.Float64()
		c, _ := i.bars[j].Close.Float64()
		sumTP += (h + l + c) / 3
	}
	meanTP := sumTP / float64(period)
	var meanDev float64
	for j := idx; j > idx-period && j >= 0; j-- {
		h, _ := i.bars[j].High.Float64()
		l, _ := i.bars[j].Low.Float64()
		c, _ := i.bars[j].Close.Float64()
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
	h, _ := i.bars[idx].High.Float64()
	l, _ := i.bars[idx].Low.Float64()
	c, _ := i.bars[idx].Close.Float64()
	currentTP := (h + l + c) / 3
	return decimal.NewFromFloat((currentTP - meanTP) / (0.015 * meanDev))
}

func (i *btIndicators) ADX(period, shift int) decimal.Decimal {
	n := len(i.bars)
	if n < period*2+shift {
		return decimal.Zero
	}
	idx := n - 1 - shift
	var sumDX float64
	for j := idx; j > idx-period && j >= 1; j-- {
		high1, _ := i.bars[j].High.Float64()
		high2, _ := i.bars[j-1].High.Float64()
		low1, _ := i.bars[j].Low.Float64()
		low2, _ := i.bars[j-1].Low.Float64()
		prevClose, _ := i.bars[j-1].Close.Float64()
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
		if tr == 0 {
			continue
		}
		sumDX += (plusDM - minusDM) / tr * 100
	}
	if sumDX < 0 {
		sumDX = -sumDX
	}
	return decimal.NewFromFloat(sumDX / float64(period))
}

func (i *btIndicators) MFI(period, shift int) decimal.Decimal {
	n := len(i.bars)
	if n < period+shift+1 {
		return decimal.Zero
	}
	idx := n - 1 - shift
	var posFlow, negFlow float64
	for j := idx; j > idx-period && j >= 1; j-- {
		h, _ := i.bars[j].High.Float64()
		l, _ := i.bars[j].Low.Float64()
		c, _ := i.bars[j].Close.Float64()
		prevC, _ := i.bars[j-1].Close.Float64()
		tp := (h + l + c) / 3
		prevTP := (h + l + prevC) / 3
		flow := tp * float64(i.bars[j].Volume)
		if tp > prevTP {
			posFlow += flow
		} else if tp < prevTP {
			negFlow += flow
		}
	}
	if negFlow == 0 {
		return decimal.NewFromInt(100)
	}
	return decimal.NewFromFloat(100 - 100/(1+posFlow/negFlow))
}

func (i *btIndicators) OBV(shift int) decimal.Decimal {
	n := len(i.bars)
	if n < shift+2 {
		return decimal.Zero
	}
	var obv float64
	for j := n - 1; j > shift; j-- {
		curr, _ := i.bars[j-1].Close.Float64()
		prev, _ := i.bars[j].Close.Float64()
		vol := float64(i.bars[j-1].Volume)
		if curr > prev {
			obv += vol
		} else if curr < prev {
			obv -= vol
		}
	}
	return decimal.NewFromFloat(obv)
}

func (i *btIndicators) SAR(step, maximum decimal.Decimal, shift int) decimal.Decimal {
	n := len(i.bars)
	if n < shift+2 {
		return decimal.Zero
	}
	idx := n - 1 - shift
	high, _ := i.bars[idx].High.Float64()
	low, _ := i.bars[idx].Low.Float64()
	prevHigh, _ := i.bars[idx-1].High.Float64()
	prevLow, _ := i.bars[idx-1].Low.Float64()
	ep := prevHigh
	sar := prevLow
	if sar > low {
		sar = low
	}
	accel, _ := step.Float64()
	maxVal, _ := maximum.Float64()
	if accel > maxVal {
		accel = maxVal
	}
	sar = sar + accel*(ep-sar)
	if sar > low {
		sar = low
	}
	_ = high
	return decimal.NewFromFloat(sar)
}

func (i *btIndicators) WPR(period, shift int) decimal.Decimal {
	n := len(i.bars)
	if n < period+shift {
		return decimal.Zero
	}
	idx := n - 1 - shift
	var highestHigh, lowestLow float64
	for j := idx; j > idx-period && j >= 0; j-- {
		h, _ := i.bars[j].High.Float64()
		l, _ := i.bars[j].Low.Float64()
		if j == idx {
			highestHigh = h
			lowestLow = l
		} else {
			if h > highestHigh {
				highestHigh = h
			}
			if l < lowestLow {
				lowestLow = l
			}
		}
	}
	currentClose, _ := i.bars[idx].Close.Float64()
	if highestHigh == lowestLow {
		return decimal.NewFromInt(-50)
	}
	return decimal.NewFromFloat((highestHigh - currentClose) / (highestHigh - lowestLow) * -100)
}

func (i *btIndicators) ICustom(name string, params []decimal.Decimal, buffer, shift int) decimal.Decimal {
	return decimal.Zero
}

// ── Shared MQL4/MQL5 indicators ──

func (i *btIndicators) Alligator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, appliedPrice int, shift int) (jaw, teeth, lips decimal.Decimal) {
	return indicators.Alligator(i.barSource(), jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, appliedPrice, shift)
}
func (i *btIndicators) Ichimoku(tenkan, kijun, senkou int, shift int) (tenkanVal, kijunVal, senkouA, senkouB decimal.Decimal) {
	return indicators.Ichimoku(i.barSource(), tenkan, kijun, senkou, shift)
}
func (i *btIndicators) Envelopes(period int, deviation decimal.Decimal, method string, appliedPrice int, shift int) (upper, lower decimal.Decimal) {
	return indicators.Envelopes(i.barSource(), period, deviation, method, appliedPrice, shift)
}
func (i *btIndicators) DeMarker(period, shift int) decimal.Decimal {
	return indicators.DeMarker(i.barSource(), period, shift)
}
func (i *btIndicators) OsMA(fastPeriod, slowPeriod, signalPeriod, appliedPrice, shift int) decimal.Decimal {
	return indicators.OsMA(i.barSource(), fastPeriod, slowPeriod, signalPeriod, appliedPrice, shift)
}
func (i *btIndicators) RVI(period, shift int) decimal.Decimal {
	return indicators.RVI(i.barSource(), period, shift)
}
func (i *btIndicators) Force(period int, method string, appliedPrice int, shift int) decimal.Decimal {
	return indicators.Force(i.barSource(), period, method, appliedPrice, shift)
}
func (i *btIndicators) Fractals(shift int) (upper, lower decimal.Decimal) {
	return indicators.Fractals(i.barSource(), shift)
}
func (i *btIndicators) Gator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, appliedPrice int, shift int) (upper, lower decimal.Decimal) {
	return indicators.Gator(i.barSource(), jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, appliedPrice, shift)
}
func (i *btIndicators) AC(shift int) decimal.Decimal {
	return indicators.AC(i.barSource(), shift)
}
func (i *btIndicators) AD(shift int) decimal.Decimal {
	return indicators.AD(i.barSource(), shift)
}
func (i *btIndicators) AO(shift int) decimal.Decimal {
	return indicators.AO(i.barSource(), shift)
}
func (i *btIndicators) BearsPower(period int, appliedPrice int, shift int) decimal.Decimal {
	return indicators.BearsPower(i.barSource(), period, appliedPrice, shift)
}
func (i *btIndicators) BullsPower(period int, appliedPrice int, shift int) decimal.Decimal {
	return indicators.BullsPower(i.barSource(), period, appliedPrice, shift)
}
func (i *btIndicators) BWMFI(shift int) decimal.Decimal {
	return indicators.BWMFI(i.barSource(), shift)
}

// ── MQL5-only indicators ──

func (i *btIndicators) AMA(period, fastPeriod, slowPeriod, shift int) decimal.Decimal {
	return indicators.AMA(i.barSource(), period, fastPeriod, slowPeriod, shift)
}
func (i *btIndicators) DEMA(period, shift int) decimal.Decimal {
	return indicators.DEMA(i.barSource(), period, shift)
}
func (i *btIndicators) TEMA(period, shift int) decimal.Decimal {
	return indicators.TEMA(i.barSource(), period, shift)
}
func (i *btIndicators) FrAMA(period, shift int) decimal.Decimal {
	return indicators.FrAMA(i.barSource(), period, shift)
}
func (i *btIndicators) VIDyA(cmoPeriod, cmoShift, maPeriod, maShift, shift int) decimal.Decimal {
	return indicators.VIDyA(i.barSource(), cmoPeriod, cmoShift, maPeriod, maShift, shift)
}
func (i *btIndicators) TriX(period, shift int) decimal.Decimal {
	return indicators.TriX(i.barSource(), period, shift)
}
func (i *btIndicators) ADXWilder(period, shift int) decimal.Decimal {
	return indicators.ADXWilder(i.barSource(), period, shift)
}
func (i *btIndicators) Chaikin(fastPeriod, slowPeriod, shift int) decimal.Decimal {
	return indicators.Chaikin(i.barSource(), fastPeriod, slowPeriod, shift)
}
func (i *btIndicators) Volumes(shift int) decimal.Decimal {
	return indicators.Volumes(i.barSource(), shift)
}
