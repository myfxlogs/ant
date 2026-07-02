package indicators

// ── Moving average helpers (stateless, full-history) ────────────────

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

	// Seed: SMA of the oldest `period` closes (indices n-1 down to n-period)
	var seedSum float64
	for i := n - 1; i >= n-period; i-- {
		c, _ := src.Close(i).Float64()
		seedSum += c
	}
	e := seedSum / float64(period)

	// Smooth from n-period-1 down to shift (newest = shift)
	for i := n - period - 1; i >= shift; i-- {
		c, _ := src.Close(i).Float64()
		e = c*alpha + e*(1-alpha)
	}
	return e
}

func lwma(src BarSource, period, shift int) float64 {
	n := src.Len()
	if n < period+shift {
		return 0
	}
	var sum, wsum float64
	for i := shift; i < shift+period; i++ {
		c, _ := src.Close(i).Float64()
		w := float64(period - (i - shift)) // weight: period for newest, 1 for oldest
		sum += c * w
		wsum += w
	}
	return sum / wsum
}

func smma(src BarSource, period, shift int) float64 {
	n := src.Len()
	if n < period+shift {
		return 0
	}
	// Seed: SMA of the oldest `period` closes
	var sum float64
	for i := n - 1; i >= n-period; i-- {
		c, _ := src.Close(i).Float64()
		sum += c
	}
	prev := sum / float64(period)
	// Wilder smoothing from n-period-1 down to shift
	for i := n - period - 1; i >= shift; i-- {
		c, _ := src.Close(i).Float64()
		prev = (prev*float64(period-1) + c) / float64(period)
	}
	return prev
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

// LWMAFloat returns the linear weighted moving average as float64.
func LWMAFloat(src BarSource, period, shift int) float64 {
	return lwma(src, period, shift)
}

// RSIFloat computes RSI on a BarSource and returns float64.
// Uses Wilder's smoothing method (MT4/MT5 correct).
func RSIFloat(src BarSource, period, shift int) float64 {
	return rsiWilder(src, period, shift)
}

// StdDevFloat computes standard deviation on a BarSource and returns float64.
// Uses population standard deviation (MT4/MT5 correct: divide by period, not period-1).
func StdDevFloat(src BarSource, period, shift int, method string, appliedPrice int) float64 {
	v, _ := StdDev(src, period, shift, method, appliedPrice).Float64()
	return v
}
