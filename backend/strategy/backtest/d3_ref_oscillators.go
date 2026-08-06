package backtest

// ── RSI (Wilder's) ───────────────────────────────────────────────────

// refRSI computes Relative Strength Index at the given shift.
func refRSI(data []float64, period, shift int) float64 {
	n := len(data)
	if n < period+1 {
		return 0
	}
	avgGain := 0.0
	avgLoss := 0.0
	for i := 1; i <= period; i++ {
		diff := data[i] - data[i-1]
		if diff > 0 {
			avgGain += diff
		} else {
			avgLoss -= diff
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	for i := period + 1; i < n-shift; i++ {
		diff := data[i] - data[i-1]
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
	return 100 - 100/(1+rs)
}

// ── ATR (Wilder's) ───────────────────────────────────────────────────

// refATR computes Average True Range at the given shift.
func refATR(bars []refBar, period, shift int) float64 {
	n := len(bars)
	if n < period+1 {
		return 0
	}
	tr := func(i int) float64 {
		hl := bars[i].High - bars[i].Low
		hc := abs(bars[i].High - bars[i-1].Close)
		lc := abs(bars[i].Low - bars[i-1].Close)
		return max3(hl, hc, lc)
	}
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += tr(i)
	}
	atr := sum / float64(period)
	for i := period + 1; i < n-shift; i++ {
		atr = (atr*float64(period-1) + tr(i)) / float64(period)
	}
	return atr
}

// ── ADX (Wilder's) ───────────────────────────────────────────────────

// refADX computes Average Directional Index at the given shift.
func refADX(bars []refBar, period, shift int) float64 {
	n := len(bars)
	if n < 2*period+1 {
		return 0
	}
	plusDM := make([]float64, n)
	minusDM := make([]float64, n)
	tr := make([]float64, n)
	for i := 1; i < n; i++ {
		up := bars[i].High - bars[i-1].High
		down := bars[i-1].Low - bars[i].Low
		if up > down && up > 0 {
			plusDM[i] = up
		}
		if down > up && down > 0 {
			minusDM[i] = down
		}
		hl := bars[i].High - bars[i].Low
		hc := abs(bars[i].High - bars[i-1].Close)
		lc := abs(bars[i].Low - bars[i-1].Close)
		tr[i] = max3(hl, hc, lc)
	}
	smoothPlusDM := wilderSmooth(plusDM, period)
	smoothMinusDM := wilderSmooth(minusDM, period)
	smoothTR := wilderSmooth(tr, period)
	dxAbs := make([]float64, n)
	for i := period; i < n; i++ {
		if smoothTR[i] == 0 {
			continue
		}
		plusDI := 100 * smoothPlusDM[i] / smoothTR[i]
		minusDI := 100 * smoothMinusDM[i] / smoothTR[i]
		diSum := plusDI + minusDI
		if diSum == 0 {
			continue
		}
		dxAbs[i] = 100 * abs(plusDI-minusDI) / diSum
	}
	dxSeed := 0.0
	count := 0
	for i := period; i < 2*period; i++ {
		if dxAbs[i] > 0 {
			dxSeed += dxAbs[i]
			count++
		}
	}
	if count == 0 {
		return 0
	}
	dxSeed /= float64(count)
	adx := dxSeed
	for i := 2 * period; i < n-shift; i++ {
		if dxAbs[i] > 0 {
			adx = (adx*float64(period-1) + dxAbs[i]) / float64(period)
		}
	}
	return adx
}

// wilderSmooth applies Wilder's smoothing to a slice starting from index 1.
func wilderSmooth(data []float64, period int) []float64 {
	n := len(data)
	out := make([]float64, n)
	if n < period+1 {
		return out
	}
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += data[i]
	}
	out[period] = sum
	for i := period + 1; i < n; i++ {
		out[i] = out[i-1] - out[i-1]/float64(period) + data[i]
	}
	return out
}

// ── MACD ─────────────────────────────────────────────────────────────

// refMACDLine computes MACD line = EMA(fast) - EMA(slow) at the given shift.
func refMACDLine(data []float64, fast, slow, shift int) float64 {
	return refEMA(data, fast, shift) - refEMA(data, slow, shift)
}

// refMACDSignal computes MACD signal line = EMA(signal) of MACD line values.
func refMACDSignal(data []float64, fast, slow, signal, shift int) float64 {
	n := len(data)
	if n < slow+signal {
		return 0
	}
	macdLine := make([]float64, n)
	for s := 0; s < n; s++ {
		macdLine[n-1-s] = refMACDLine(data, fast, slow, s)
	}
	validStart := slow - 1
	if validStart+signal >= n {
		return 0
	}
	seed := 0.0
	for i := validStart; i < validStart+signal; i++ {
		seed += macdLine[i]
	}
	seed /= float64(signal)
	k := 2.0 / float64(signal+1)
	ema := seed
	for i := validStart + signal; i < n-shift; i++ {
		ema = macdLine[i]*k + ema*(1-k)
	}
	return ema
}

// ── Stochastic ───────────────────────────────────────────────────────

// refStochastic computes %K and %D at the given shift.
// Returns (k, d).
func refStochastic(bars []refBar, kPeriod, dPeriod, slowing, shift int) (float64, float64) {
	n := len(bars)
	idx := n - 1 - shift
	if idx < kPeriod+slowing-2 {
		return 0, 0
	}
	kSum := 0.0
	for s := 0; s < slowing; s++ {
		i := idx - s
		highest := bars[i-kPeriod+1].High
		lowest := bars[i-kPeriod+1].Low
		for j := i - kPeriod + 2; j <= i; j++ {
			if bars[j].High > highest {
				highest = bars[j].High
			}
			if bars[j].Low < lowest {
				lowest = bars[j].Low
			}
		}
		rng := highest - lowest
		if rng == 0 {
			kSum += 0
		} else {
			kSum += 100 * (bars[i].Close - lowest) / rng
		}
	}
	k := kSum / float64(slowing)
	dSum := 0.0
	for s := 0; s < dPeriod; s++ {
		if s == 0 {
			dSum += k
		} else {
			i := idx - s
			if i < kPeriod+slowing-2 {
				dSum += k
				continue
			}
			ks := 0.0
			for sl := 0; sl < slowing; sl++ {
				j := i - sl
				highest := bars[j-kPeriod+1].High
				lowest := bars[j-kPeriod+1].Low
				for jj := j - kPeriod + 2; jj <= j; jj++ {
					if bars[jj].High > highest {
						highest = bars[jj].High
					}
					if bars[jj].Low < lowest {
						lowest = bars[jj].Low
					}
				}
				rng := highest - lowest
				if rng == 0 {
					ks += 0
				} else {
					ks += 100 * (bars[j].Close - lowest) / rng
				}
			}
			dSum += ks / float64(slowing)
		}
	}
	d := dSum / float64(dPeriod)
	return k, d
}

// ── Bollinger Bands ──────────────────────────────────────────────────

// refBollinger computes (upper, middle, lower) bands at the given shift.
func refBollinger(data []float64, period int, deviation float64, shift int) (float64, float64, float64) {
	idx := len(data) - 1 - shift
	if idx < period-1 {
		return 0, 0, 0
	}
	sum := 0.0
	for i := idx - period + 1; i <= idx; i++ {
		sum += data[i]
	}
	mean := sum / float64(period)
	sqSum := 0.0
	for i := idx - period + 1; i <= idx; i++ {
		diff := data[i] - mean
		sqSum += diff * diff
	}
	std := sqrt(sqSum / float64(period))
	upper := mean + deviation*std
	lower := mean - deviation*std
	return upper, mean, lower
}

// ── CCI ──────────────────────────────────────────────────────────────

// refCCI computes Commodity Channel Index at the given shift.
// appliedPrice: 1=close, 2=open, 3=high, 4=low, 5=median, 6=typical, 7=weighted
func refCCI(bars []refBar, period, shift, appliedPrice int) float64 {
	n := len(bars)
	idx := n - 1 - shift
	if idx < period-1 {
		return 0
	}
	selectPrice := func(i int) float64 {
		b := bars[i]
		switch appliedPrice {
		case 1:
			return b.Close
		case 2:
			return b.Open
		case 3:
			return b.High
		case 4:
			return b.Low
		case 5:
			return (b.High + b.Low) / 2
		case 6:
			return (b.High + b.Low + b.Close) / 3
		case 7:
			return (b.High + b.Low + b.Close*2) / 4
		default:
			return (b.High + b.Low + b.Close) / 3
		}
	}
	prices := make([]float64, period)
	for i := 0; i < period; i++ {
		prices[i] = selectPrice(idx - period + 1 + i)
	}
	sum := 0.0
	for _, v := range prices {
		sum += v
	}
	mean := sum / float64(period)
	mad := 0.0
	for _, v := range prices {
		mad += abs(v - mean)
	}
	mad /= float64(period)
	if mad == 0 {
		return 0
	}
	currentPrice := prices[period-1]
	return (currentPrice - mean) / (0.015 * mad)
}

// ── Ichimoku ─────────────────────────────────────────────────────────

// refIchimoku computes (tenkan, kijun, senkouA, senkouB) at the given shift.
func refIchimoku(bars []refBar, tenkanPeriod, kijunPeriod, senkouPeriod, shift int) (float64, float64, float64, float64) {
	n := len(bars)
	idx := n - 1 - shift
	tenkan := midpoint(bars, idx, tenkanPeriod)
	kijun := midpoint(bars, idx, kijunPeriod)
	senkouAIdx := idx - kijunPeriod
	var senkouA float64
	if senkouAIdx >= tenkanPeriod-1 && senkouAIdx >= kijunPeriod-1 {
		t := midpoint(bars, senkouAIdx, tenkanPeriod)
		k := midpoint(bars, senkouAIdx, kijunPeriod)
		senkouA = (t + k) / 2
	}
	senkouBIdx := idx - kijunPeriod
	var senkouB float64
	if senkouBIdx >= senkouPeriod-1 {
		senkouB = midpoint(bars, senkouBIdx, senkouPeriod)
	}
	return tenkan, kijun, senkouA, senkouB
}

func midpoint(bars []refBar, idx, period int) float64 {
	if idx < period-1 {
		return 0
	}
	highest := bars[idx-period+1].High
	lowest := bars[idx-period+1].Low
	for i := idx - period + 2; i <= idx; i++ {
		if bars[i].High > highest {
			highest = bars[i].High
		}
		if bars[i].Low < lowest {
			lowest = bars[i].Low
		}
	}
	return (highest + lowest) / 2
}

// ── Parabolic SAR ────────────────────────────────────────────────────

// refSAR computes Parabolic SAR at the given shift.
func refSAR(bars []refBar, step, maxAF float64, shift int) float64 {
	n := len(bars)
	if n < 2 {
		return 0
	}
	sar := make([]float64, n)
	af := step
	isLong := bars[1].Close > bars[0].Close
	if isLong {
		sar[0] = bars[0].Low
		ep := bars[0].High
		for i := 1; i < n; i++ {
			sar[i] = sar[i-1] + af*(ep-sar[i-1])
			if bars[i].Low < sar[i] {
				sar[i] = ep
				isLong = false
				af = step
				ep = bars[i].Low
			} else {
				if bars[i].High > ep {
					ep = bars[i].High
					if af < maxAF {
						af += step
					}
				}
			}
		}
	} else {
		sar[0] = bars[0].High
		ep := bars[0].Low
		for i := 1; i < n; i++ {
			sar[i] = sar[i-1] + af*(ep-sar[i-1])
			if bars[i].High > sar[i] {
				sar[i] = ep
				isLong = true
				af = step
				ep = bars[i].High
			} else {
				if bars[i].Low < ep {
					ep = bars[i].Low
					if af < maxAF {
						af += step
					}
				}
			}
		}
	}
	return sar[n-1-shift]
}

// ── Math helpers ─────────────────────────────────────────────────────

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func max3(a, b, c float64) float64 {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	g := x / 2
	for i := 0; i < 20; i++ {
		g = (g + x/g) / 2
	}
	return g
}
