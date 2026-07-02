package indicators

import (
	"github.com/shopspring/decimal"
)

// ── Moving average and MACD indicators ──────────────────────────────

// MA returns the moving average. method: "SMA", "EMA", "SMMA", "LWMA".
// appliedPrice: 1=close, 2=open, 3=high, 4=low, 5=median, 6=typical, 7=weighted.
func MA(src BarSource, period, shift int, method string, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	switch method {
	case "SMA", "sma", "":
		return decimal.NewFromFloat(sma(src, period, shift))
	case "EMA", "ema":
		return decimal.NewFromFloat(ema(src, period, shift))
	case "SMMA", "smma":
		return decimal.NewFromFloat(smma(src, period, shift))
	case "LWMA", "lwma":
		return decimal.NewFromFloat(lwma(src, period, shift))
	default:
		return decimal.NewFromFloat(sma(src, period, shift))
	}
}

// EMA returns the exponential moving average with SMA seed.
// MT4/MT5: EMA seeds with SMA of the first `period` closes, not the first close.
func EMA(src BarSource, period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(ema(src, period, shift))
}

// MACD returns the MACD line = EMA(fast) - EMA(slow) at the given shift.
// appliedPrice: 1=close, 2=open, 3=high, 4=low, 5=median, 6=typical, 7=weighted.
func MACD(src BarSource, fast, slow, shift int, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	if src.Len() < slow+shift {
		return decimal.Zero
	}
	return decimal.NewFromFloat(ema(src, fast, shift) - ema(src, slow, shift))
}

// MACDSignal returns the signal line = EMA(signalPeriod) of MACD values.
// MT4/MT5: signal is EMA of the MACD line over signalPeriod bars, using full history.
// appliedPrice: 1=close, 2=open, 3=high, 4=low, 5=median, 6=typical, 7=weighted.
func MACDSignal(src BarSource, fast, slow, signalPeriod, shift int, appliedPrice int) decimal.Decimal {
	src = withAppliedPrice(src, appliedPrice)
	n := src.Len()
	if n < slow+signalPeriod+shift {
		return decimal.Zero
	}

	// Compute MACD for every bar from oldest (n-1) down to shift.
	// MACD at bar i = ema(fast, i) - ema(slow, i)
	// We need enough bars: slow period for EMA + signalPeriod for signal smoothing.
	// Start computing MACD from bar n-slow-1 (first bar where slow EMA is valid)
	startBar := n - slow - 1
	if startBar < shift+signalPeriod-1 {
		startBar = shift + signalPeriod - 1
	}

	// Seed: SMA of MACD over the oldest signalPeriod bars
	var macdSum float64
	for i := startBar; i >= startBar-signalPeriod+1 && i >= 0; i-- {
		macdSum += ema(src, fast, i) - ema(src, slow, i)
	}
	signal := macdSum / float64(signalPeriod)

	// EMA smoothing from startBar-signalPeriod down to shift
	alpha := 2.0 / float64(signalPeriod+1)
	for i := startBar - signalPeriod; i >= shift; i-- {
		if i < 0 {
			break
		}
		macdVal := ema(src, fast, i) - ema(src, slow, i)
		signal = macdVal*alpha + signal*(1-alpha)
	}
	return decimal.NewFromFloat(signal)
}
