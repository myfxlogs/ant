package runner

import (
	"math"

	"github.com/shopspring/decimal"

	"anttrader/strategy/indicators"
	"anttrader/strategy/sdk"
)

// indicatorSet implements sdk.IndicatorSet backed by the runner's bar data.
type indicatorSet struct {
	runner *Runner
}

func (is *indicatorSet) bars() sdk.BarSeries {
	is.runner.ctx.mu.RLock()
	defer is.runner.ctx.mu.RUnlock()
	return is.runner.ctx.bars
}

func (is *indicatorSet) MA(period, shift int, method string) decimal.Decimal {
	return is.EMA(period, shift) // default to EMA
}

func (is *indicatorSet) EMA(period, shift int) decimal.Decimal {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return decimal.Zero
	}
	alpha := 2.0 / float64(period+1)
	var ema float64
	for i := period + shift - 1; i >= shift; i-- {
		price, _ := bars.Close(i).Float64()
		if i == period+shift-1 {
			ema = price
		} else {
			ema = price*alpha + ema*(1-alpha)
		}
	}
	return decimal.NewFromFloat(ema)
}

func (is *indicatorSet) RSI(period, shift int) decimal.Decimal {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift+1 {
		return decimal.Zero
	}
	var avgGain, avgLoss float64
	for i := shift + 1; i <= shift+period; i++ {
		curr, _ := bars.Close(i - 1).Float64()
		prev, _ := bars.Close(i).Float64()
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

func (is *indicatorSet) MACD(fastPeriod, slowPeriod, signalPeriod, shift int) decimal.Decimal {
	fastEMA := is.EMA(fastPeriod, shift)
	slowEMA := is.EMA(slowPeriod, shift)
	return fastEMA.Sub(slowEMA)
}

func (is *indicatorSet) MACDSignal(fastPeriod, slowPeriod, signalPeriod, shift int) decimal.Decimal {
	return is.MACD(fastPeriod, slowPeriod, signalPeriod, shift)
}

func (is *indicatorSet) ATR(period, shift int) decimal.Decimal {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return decimal.Zero
	}
	var sumTR float64
	for i := shift; i < shift+period; i++ {
		high, _ := bars.High(i).Float64()
		low, _ := bars.Low(i).Float64()
		prevClose, _ := bars.Close(i + 1).Float64()
		tr1 := high - low
		tr2 := high - prevClose
		tr3 := prevClose - low
		tr := tr1
		if tr2 < 0 {
			tr2 = -tr2
		}
		if tr3 < 0 {
			tr3 = -tr3
		}
		if tr2 > tr {
			tr = tr2
		}
		if tr3 > tr {
			tr = tr3
		}
		sumTR += tr
	}
	return decimal.NewFromFloat(sumTR / float64(period))
}

func (is *indicatorSet) Bollinger(period int, deviation decimal.Decimal, shift int) (upper, middle, lower decimal.Decimal) {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return decimal.Zero, decimal.Zero, decimal.Zero
	}
	var sum float64
	for i := shift; i < shift+period; i++ {
		c, _ := bars.Close(i).Float64()
		sum += c
	}
	mean := sum / float64(period)
	var variance float64
	for i := shift; i < shift+period; i++ {
		c, _ := bars.Close(i).Float64()
		diff := c - mean
		variance += diff * diff
	}
	stdDev := 0.0
	if period > 1 {
		stdDev = math.Sqrt(variance / float64(period-1))
	}
	dev, _ := deviation.Float64()
	mid := decimal.NewFromFloat(mean)
	sd := decimal.NewFromFloat(stdDev * dev)
	return mid.Add(sd), mid, mid.Sub(sd)
}

func (is *indicatorSet) Stochastic(kPeriod, dPeriod, slowing, shift int) (k, d decimal.Decimal) {
	bars := is.bars()
	if bars == nil || bars.Len() < kPeriod+shift {
		return decimal.NewFromInt(50), decimal.NewFromInt(50)
	}
	var highestHigh, lowestLow float64
	for i := shift; i < shift+kPeriod; i++ {
		h, _ := bars.High(i).Float64()
		l, _ := bars.Low(i).Float64()
		if i == shift {
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
	currentClose, _ := bars.Close(shift).Float64()
	var kVal float64
	if highestHigh == lowestLow {
		kVal = 50
	} else {
		kVal = (currentClose - lowestLow) / (highestHigh - lowestLow) * 100
	}
	var kSum float64
	for i := shift; i < shift+dPeriod && i < bars.Len(); i++ {
		var hh, ll float64
		for j := i; j < i+kPeriod && j < bars.Len(); j++ {
			h, _ := bars.High(j).Float64()
			l, _ := bars.Low(j).Float64()
			if j == i {
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
		cc, _ := bars.Close(i).Float64()
		if hh == ll {
			kSum += 50
		} else {
			kSum += (cc - ll) / (hh - ll) * 100
		}
	}
	return decimal.NewFromFloat(kVal), decimal.NewFromFloat(kSum / float64(dPeriod))
}

func (is *indicatorSet) CCI(period, shift int) decimal.Decimal {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return decimal.Zero
	}
	var sumTP float64
	for i := shift; i < shift+period; i++ {
		h, _ := bars.High(i).Float64()
		l, _ := bars.Low(i).Float64()
		c, _ := bars.Close(i).Float64()
		sumTP += (h + l + c) / 3
	}
	meanTP := sumTP / float64(period)
	var meanDev float64
	for i := shift; i < shift+period; i++ {
		h, _ := bars.High(i).Float64()
		l, _ := bars.Low(i).Float64()
		c, _ := bars.Close(i).Float64()
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
	currentH, _ := bars.High(shift).Float64()
	currentL, _ := bars.Low(shift).Float64()
	currentC, _ := bars.Close(shift).Float64()
	currentTP := (currentH + currentL + currentC) / 3
	return decimal.NewFromFloat((currentTP - meanTP) / (0.015 * meanDev))
}

func (is *indicatorSet) ADX(period, shift int) decimal.Decimal {
	bars := is.bars()
	if bars == nil || bars.Len() < period*2+shift {
		return decimal.Zero
	}
	var sumDX float64
	for i := shift; i < shift+period; i++ {
		high1, _ := bars.High(i).Float64()
		high2, _ := bars.High(i + 1).Float64()
		low1, _ := bars.Low(i).Float64()
		low2, _ := bars.Low(i + 1).Float64()
		prevClose, _ := bars.Close(i + 1).Float64()
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

func (is *indicatorSet) MFI(period, shift int) decimal.Decimal {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift+1 {
		return decimal.Zero
	}
	var posFlow, negFlow float64
	for i := shift; i < shift+period; i++ {
		h, _ := bars.High(i).Float64()
		l, _ := bars.Low(i).Float64()
		c, _ := bars.Close(i).Float64()
		prevC, _ := bars.Close(i + 1).Float64()
		typicalPrice := (h + l + c) / 3
		prevTypical := (h + l + prevC) / 3
		flow := typicalPrice * float64(bars.Volume(i))
		if typicalPrice > prevTypical {
			posFlow += flow
		} else if typicalPrice < prevTypical {
			negFlow += flow
		}
	}
	if negFlow == 0 {
		return decimal.NewFromInt(100)
	}
	mfr := posFlow / negFlow
	return decimal.NewFromFloat(100 - 100/(1+mfr))
}

func (is *indicatorSet) OBV(shift int) decimal.Decimal {
	bars := is.bars()
	if bars == nil || bars.Len() < shift+2 {
		return decimal.Zero
	}
	var obv float64
	for i := bars.Len() - 1; i > shift; i-- {
		curr, _ := bars.Close(i - 1).Float64()
		prev, _ := bars.Close(i).Float64()
		vol := float64(bars.Volume(i - 1))
		if curr > prev {
			obv += vol
		} else if curr < prev {
			obv -= vol
		}
	}
	return decimal.NewFromFloat(obv)
}

func (is *indicatorSet) SAR(step, maximum decimal.Decimal, shift int) decimal.Decimal {
	bars := is.bars()
	if bars == nil || bars.Len() < shift+2 {
		return decimal.Zero
	}
	high, _ := bars.High(shift).Float64()
	low, _ := bars.Low(shift).Float64()
	prevHigh, _ := bars.High(shift + 1).Float64()
	prevLow, _ := bars.Low(shift + 1).Float64()
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
	if sar < high {
		_ = high
	}
	return decimal.NewFromFloat(sar)
}

func (is *indicatorSet) StdDev(period, shift int) decimal.Decimal {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return decimal.Zero
	}
	var sum float64
	for i := shift; i < shift+period; i++ {
		c, _ := bars.Close(i).Float64()
		sum += c
	}
	mean := sum / float64(period)
	var variance float64
	for i := shift; i < shift+period; i++ {
		c, _ := bars.Close(i).Float64()
		d := c - mean
		variance += d * d
	}
	if period > 1 {
		return decimal.NewFromFloat(math.Sqrt(variance / float64(period-1)))
	}
	return decimal.Zero
}

func (is *indicatorSet) WPR(period, shift int) decimal.Decimal {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return decimal.Zero
	}
	var highestHigh, lowestLow float64
	for i := shift; i < shift+period; i++ {
		h, _ := bars.High(i).Float64()
		l, _ := bars.Low(i).Float64()
		if i == shift {
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
	currentClose, _ := bars.Close(shift).Float64()
	if highestHigh == lowestLow {
		return decimal.NewFromInt(-50)
	}
	return decimal.NewFromFloat((highestHigh - currentClose) / (highestHigh - lowestLow) * -100)
}

func (is *indicatorSet) Momentum(period, shift int) decimal.Decimal {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return decimal.Zero
	}
	curr, _ := bars.Close(shift).Float64()
	prev, _ := bars.Close(shift + period).Float64()
	return decimal.NewFromFloat(curr - prev)
}

func (is *indicatorSet) ICustom(name string, params []decimal.Decimal, buffer, shift int) decimal.Decimal {
	return decimal.Zero
}

// ── Shared MQL4/MQL5 indicators ──

func (is *indicatorSet) Alligator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, appliedPrice int, shift int) (jaw, teeth, lips decimal.Decimal) {
	return indicators.Alligator(is.barSource(), jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, appliedPrice, shift)
}

func (is *indicatorSet) Ichimoku(tenkan, kijun, senkou int, shift int) (tenkanVal, kijunVal, senkouA, senkouB decimal.Decimal) {
	return indicators.Ichimoku(is.barSource(), tenkan, kijun, senkou, shift)
}

func (is *indicatorSet) Envelopes(period int, deviation decimal.Decimal, method string, appliedPrice int, shift int) (upper, lower decimal.Decimal) {
	return indicators.Envelopes(is.barSource(), period, deviation, method, appliedPrice, shift)
}

func (is *indicatorSet) DeMarker(period, shift int) decimal.Decimal {
	return indicators.DeMarker(is.barSource(), period, shift)
}

func (is *indicatorSet) OsMA(fastPeriod, slowPeriod, signalPeriod, appliedPrice, shift int) decimal.Decimal {
	return indicators.OsMA(is.barSource(), fastPeriod, slowPeriod, signalPeriod, appliedPrice, shift)
}

func (is *indicatorSet) RVI(period, shift int) decimal.Decimal {
	return indicators.RVI(is.barSource(), period, shift)
}

func (is *indicatorSet) Force(period int, method string, appliedPrice int, shift int) decimal.Decimal {
	return indicators.Force(is.barSource(), period, method, appliedPrice, shift)
}

func (is *indicatorSet) Fractals(shift int) (upper, lower decimal.Decimal) {
	return indicators.Fractals(is.barSource(), shift)
}

func (is *indicatorSet) Gator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, appliedPrice int, shift int) (upper, lower decimal.Decimal) {
	return indicators.Gator(is.barSource(), jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, appliedPrice, shift)
}

func (is *indicatorSet) AC(shift int) decimal.Decimal {
	return indicators.AC(is.barSource(), shift)
}

func (is *indicatorSet) AD(shift int) decimal.Decimal {
	return indicators.AD(is.barSource(), shift)
}

func (is *indicatorSet) AO(shift int) decimal.Decimal {
	return indicators.AO(is.barSource(), shift)
}

func (is *indicatorSet) BearsPower(period int, appliedPrice int, shift int) decimal.Decimal {
	return indicators.BearsPower(is.barSource(), period, appliedPrice, shift)
}

func (is *indicatorSet) BullsPower(period int, appliedPrice int, shift int) decimal.Decimal {
	return indicators.BullsPower(is.barSource(), period, appliedPrice, shift)
}

func (is *indicatorSet) BWMFI(shift int) decimal.Decimal {
	return indicators.BWMFI(is.barSource(), shift)
}

// ── MQL5-only indicators ──

func (is *indicatorSet) AMA(period, fastPeriod, slowPeriod, shift int) decimal.Decimal {
	return indicators.AMA(is.barSource(), period, fastPeriod, slowPeriod, shift)
}

func (is *indicatorSet) DEMA(period, shift int) decimal.Decimal {
	return indicators.DEMA(is.barSource(), period, shift)
}

func (is *indicatorSet) TEMA(period, shift int) decimal.Decimal {
	return indicators.TEMA(is.barSource(), period, shift)
}

func (is *indicatorSet) FrAMA(period, shift int) decimal.Decimal {
	return indicators.FrAMA(is.barSource(), period, shift)
}

func (is *indicatorSet) VIDyA(cmoPeriod, cmoShift, maPeriod, maShift, shift int) decimal.Decimal {
	return indicators.VIDyA(is.barSource(), cmoPeriod, cmoShift, maPeriod, maShift, shift)
}

func (is *indicatorSet) TriX(period, shift int) decimal.Decimal {
	return indicators.TriX(is.barSource(), period, shift)
}

func (is *indicatorSet) ADXWilder(period, shift int) decimal.Decimal {
	return indicators.ADXWilder(is.barSource(), period, shift)
}

func (is *indicatorSet) Chaikin(fastPeriod, slowPeriod, shift int) decimal.Decimal {
	return indicators.Chaikin(is.barSource(), fastPeriod, slowPeriod, shift)
}

func (is *indicatorSet) Volumes(shift int) decimal.Decimal {
	return indicators.Volumes(is.barSource(), shift)
}
