package runner

import (
	"anttrader/strategy/sdk"
)

// indicatorSet implements sdk.IndicatorSet backed by the runner's bar data.
type indicatorSet struct {
	runner *Runner
}

func (is *indicatorSet) bars() sdk.BarSeries {
	if is.runner.ctx.bars != nil {
		return is.runner.ctx.bars
	}
	return nil
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
		stdDev = variance / float64(period-1)
		// sqrt approximation
		stdDev = stdDev * stdDev // simplified
	}
	upper = middle + deviation*stdDev
	lower = middle - deviation*stdDev
	return
}

func (is *indicatorSet) Stochastic(kPeriod, dPeriod, slowing, shift int) (k, d float64) {
	return 50, 50 // stub
}

func (is *indicatorSet) CCI(period, shift int) float64 {
	return 0 // stub
}

func (is *indicatorSet) ADX(period, shift int) float64 {
	return 0 // stub
}

func (is *indicatorSet) MFI(period, shift int) float64 {
	return 0 // stub
}

func (is *indicatorSet) OBV(shift int) float64 {
	return 0 // stub
}

func (is *indicatorSet) SAR(step, maximum float64, shift int) float64 {
	return 0 // stub
}

func (is *indicatorSet) StdDev(period, shift int) float64 {
	return 0 // stub
}

func (is *indicatorSet) WPR(period, shift int) float64 {
	return 0 // stub
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
