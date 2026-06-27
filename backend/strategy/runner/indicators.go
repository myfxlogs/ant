package runner

import (
	"math"

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

func (is *indicatorSet) MA(period, shift int, method string) float64 {
	return is.EMA(period, shift) // default to EMA
}

func (is *indicatorSet) EMA(period, shift int) float64 {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return 0
	}
	alpha := 2.0 / float64(period+1)
	var ema float64
	// Seed with SMA for the first value
	for i := period + shift - 1; i >= shift; i-- {
		price, _ := bars.Close(i).Float64()
		if i == period+shift-1 {
			ema = price
		} else {
			ema = price*alpha + ema*(1-alpha)
		}
	}
	return ema
}

func (is *indicatorSet) RSI(period, shift int) float64 {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift+1 {
		return 0
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
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func (is *indicatorSet) MACD(fastPeriod, slowPeriod, signalPeriod, shift int) float64 {
	fastEMA := is.EMA(fastPeriod, shift)
	slowEMA := is.EMA(slowPeriod, shift)
	return fastEMA - slowEMA
}

func (is *indicatorSet) MACDSignal(fastPeriod, slowPeriod, signalPeriod, shift int) float64 {
	// Simplified: compute MACD line then EMA of it
	macdLine := is.MACD(fastPeriod, slowPeriod, signalPeriod, shift)
	// For a proper signal line we'd need a series of MACD values.
	// This is a first-order approximation.
	return macdLine
}

func (is *indicatorSet) ATR(period, shift int) float64 {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return 0
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
	return sumTR / float64(period)
}

func (is *indicatorSet) Bollinger(period int, deviation float64, shift int) (upper, middle, lower float64) {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return 0, 0, 0
	}
	// Compute SMA
	var sum float64
	for i := shift; i < shift+period; i++ {
		c, _ := bars.Close(i).Float64()
		sum += c
	}
	middle = sum / float64(period)
	// Compute standard deviation
	var variance float64
	for i := shift; i < shift+period; i++ {
		c, _ := bars.Close(i).Float64()
		diff := c - middle
		variance += diff * diff
	}
	stdDev := 0.0
	if period > 1 {
		stdDev = math.Sqrt(variance / float64(period-1))
	}
	upper = middle + deviation*stdDev
	lower = middle - deviation*stdDev
	return
}

func (is *indicatorSet) Stochastic(kPeriod, dPeriod, slowing, shift int) (k, d float64) {
	bars := is.bars()
	if bars == nil || bars.Len() < kPeriod+shift {
		return 50, 50
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
	if highestHigh == lowestLow {
		k = 50
	} else {
		k = (currentClose - lowestLow) / (highestHigh - lowestLow) * 100
	}
	// D is SMA of K over dPeriod
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
	d = kSum / float64(dPeriod)
	return
}

func (is *indicatorSet) CCI(period, shift int) float64 {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return 0
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
		return 0
	}
	currentH, _ := bars.High(shift).Float64()
	currentL, _ := bars.Low(shift).Float64()
	currentC, _ := bars.Close(shift).Float64()
	currentTP := (currentH + currentL + currentC) / 3
	return (currentTP - meanTP) / (0.015 * meanDev)
}

func (is *indicatorSet) ADX(period, shift int) float64 {
	bars := is.bars()
	if bars == nil || bars.Len() < period*2+shift {
		return 0
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
	return sumDX / float64(period)
}

func (is *indicatorSet) MFI(period, shift int) float64 {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift+1 {
		return 0
	}
	var posFlow, negFlow float64
	for i := shift; i < shift+period; i++ {
		h, _ := bars.High(i).Float64()
		l, _ := bars.Low(i).Float64()
		c, _ := bars.Close(i).Float64()
		prevC, _ := bars.Close(i + 1).Float64()
		typicalPrice := (h + l + c) / 3
		prevTypical := (h + l + prevC) / 3 // simplified
		flow := typicalPrice * float64(bars.Volume(i))
		if typicalPrice > prevTypical {
			posFlow += flow
		} else if typicalPrice < prevTypical {
			negFlow += flow
		}
	}
	if negFlow == 0 {
		return 100
	}
	mfr := posFlow / negFlow
	return 100 - 100/(1+mfr)
}

func (is *indicatorSet) OBV(shift int) float64 {
	bars := is.bars()
	if bars == nil || bars.Len() < shift+2 {
		return 0
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
	return obv
}

func (is *indicatorSet) SAR(step, maximum float64, shift int) float64 {
	bars := is.bars()
	if bars == nil || bars.Len() < shift+2 {
		return 0
	}
	// Simplified Parabolic SAR: use previous bar's extreme with acceleration
	high, _ := bars.High(shift).Float64()
	low, _ := bars.Low(shift).Float64()
	prevHigh, _ := bars.High(shift + 1).Float64()
	prevLow, _ := bars.Low(shift + 1).Float64()
	ep := prevHigh
	sar := prevLow
	if sar > low {
		sar = low
	}
	accel := step
	if accel > maximum {
		accel = maximum
	}
	sar = sar + accel*(ep-sar)
	if sar > low {
		sar = low
	}
	if sar < high {
		_ = high // SAR flips to above price in downtrend
	}
	return sar
}

func (is *indicatorSet) StdDev(period, shift int) float64 {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return 0
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
		return math.Sqrt(variance / float64(period-1))
	}
	return 0
}

func (is *indicatorSet) WPR(period, shift int) float64 {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return 0
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
		return -50
	}
	return (highestHigh - currentClose) / (highestHigh - lowestLow) * -100
}

func (is *indicatorSet) Momentum(period, shift int) float64 {
	bars := is.bars()
	if bars == nil || bars.Len() < period+shift {
		return 0
	}
	curr, _ := bars.Close(shift).Float64()
	prev, _ := bars.Close(shift + period).Float64()
	return curr - prev
}

func (is *indicatorSet) ICustom(name string, params []float64, buffer, shift int) float64 {
	return 0 // stub — custom indicators need runtime registration
}
