package interp

import (
	"github.com/shopspring/decimal"

	"anttrader/strategy/indicators"
)

// arrayBarSource adapts a user-provided decimal array to the indicators.BarSource interface.
// This enables *OnArray indicator variants (iMAOnArray, iRSIOnArray, etc.) that compute
// indicators on arbitrary arrays instead of bar data.
// All OHLC fields map to the same array value; Volume is zero.
type arrayBarSource struct {
	data []decimal.Decimal
}

func (a *arrayBarSource) Open(i int) decimal.Decimal {
	if i < 0 || i >= len(a.data) {
		return decimal.Zero
	}
	return a.data[i]
}

func (a *arrayBarSource) High(i int) decimal.Decimal {
	return a.Open(i)
}

func (a *arrayBarSource) Low(i int) decimal.Decimal {
	return a.Open(i)
}

func (a *arrayBarSource) Close(i int) decimal.Decimal {
	return a.Open(i)
}

func (a *arrayBarSource) Volume(i int) int64 {
	return 0
}

func (a *arrayBarSource) Len() int {
	return len(a.data)
}

// valueArrayToDecimal converts a Value array (ValArray) to []decimal.Decimal.
func valueArrayToDecimal(arr []Value) []decimal.Decimal {
	out := make([]decimal.Decimal, len(arr))
	for i, v := range arr {
		out[i] = v.ToDecimal()
	}
	return out
}

// isOnArrayFunc returns true if the function name is a *OnArray variant.
func isOnArrayFunc(name string) bool {
	switch name {
	case "iMAOnArray", "iRSIOnArray", "iATROnArray", "iBandsOnArray",
		"iStdDevOnArray", "iMomentumOnArray", "iCCIOnArray", "iMACDOnArray":
		return true
	}
	return false
}

// callIndicatorOnArray dispatches *OnArray indicator functions.
// These compute indicators on user-provided arrays, not bar data.
// MQL4 signature: iXOnArray(array, total, period, shift, ...)
// total=0 means use entire array.
func (it *Interpreter) callIndicatorOnArray(name string, args []Expr) (Value, bool) {
	// Only handle *OnArray functions
	if !isOnArrayFunc(name) {
		return NoneVal(), false
	}

	vals := make([]Value, len(args))
	for i := range args {
		vals[i] = it.evalExpr(&args[i])
	}

	if len(vals) < 1 || vals[0].Kind != ValArray {
		return NoneVal(), false
	}

	data := valueArrayToDecimal(vals[0].Array)
	if len(data) == 0 {
		return DecimalVal(decimal.Zero), true
	}
	src := &arrayBarSource{data: data}

	switch name {
	case "iMAOnArray":
		// iMAOnArray(array, total, period, ma_shift, ma_method, shift)
		if len(vals) >= 6 {
			period := int(vals[2].ToInt())
			maShift := int(vals[3].ToInt())
			method := maMethodName(vals[4].ToInt())
			shift := int(vals[5].ToInt())
			return DecimalVal(maOnArray(src, period, maShift+shift, method)), true
		}
	case "iRSIOnArray":
		// iRSIOnArray(array, total, period, shift)
		if len(vals) >= 4 {
			period := int(vals[2].ToInt())
			shift := int(vals[3].ToInt())
			return DecimalVal(rsiOnArray(src, period, shift)), true
		}
	case "iATROnArray":
		// iATROnArray(array, total, period, shift) — ATR on array doesn't make full sense
		// (ATR needs High/Low), but with arrayBarSource all OHLC = same value,
		// so TR = 0, ATR = 0. Return 0 for compatibility.
		return DecimalVal(decimal.Zero), true
	case "iBandsOnArray":
		// iBandsOnArray(array, total, deviation, bands_shift, mode, shift)
		if len(vals) >= 6 {
			period := 20 // default
			if len(vals) >= 7 {
				period = int(vals[6].ToInt())
			}
			dev := vals[2].ToDecimal()
			shift := int(vals[5].ToInt())
			upper, lower := indicators.Envelopes(src, period, dev, "sma", 0, shift)
			mid := upper.Add(lower).Div(decimal.NewFromInt(2))
			mode := vals[4].ToInt()
			switch mode {
			case 0:
				return DecimalVal(upper), true
			case 1:
				return DecimalVal(lower), true
			default:
				return DecimalVal(mid), true
			}
		}
	case "iStdDevOnArray":
		// iStdDevOnArray(array, total, ma_period, ma_shift, ma_method, shift)
		if len(vals) >= 6 {
			period := int(vals[2].ToInt())
			shift := int(vals[5].ToInt())
			return DecimalVal(stdDevOnArray(src, period, shift)), true
		}
	case "iMomentumOnArray":
		// iMomentumOnArray(array, total, period, shift)
		if len(vals) >= 4 {
			period := int(vals[2].ToInt())
			shift := int(vals[3].ToInt())
			return DecimalVal(momentumOnArray(src, period, shift)), true
		}
	case "iCCIOnArray":
		// iCCIOnArray(array, total, period, shift)
		if len(vals) >= 4 {
			period := int(vals[2].ToInt())
			shift := int(vals[3].ToInt())
			return DecimalVal(cciOnArray(src, period, shift)), true
		}
	case "iMACDOnArray":
		// iMACDOnArray(array, total, fast_period, slow_period, signal_period, shift)
		if len(vals) >= 6 {
			fastP := int(vals[2].ToInt())
			slowP := int(vals[3].ToInt())
			signalP := int(vals[4].ToInt())
			shift := int(vals[5].ToInt())
			return DecimalVal(macdOnArray(src, fastP, slowP, signalP, shift)), true
		}
	}
	return NoneVal(), false
}

// ── OnArray indicator implementations ────────────────────────────────
// These delegate to the shared indicators package via arrayBarSource.

func maOnArray(src indicators.BarSource, period, shift int, method string) decimal.Decimal {
	switch method {
	case "sma":
		return decimal.NewFromFloat(indicators.SMAFloat(src, period, shift))
	case "ema":
		return decimal.NewFromFloat(indicators.EMAFloat(src, period, shift))
	case "smma":
		return decimal.NewFromFloat(indicators.SMMAFloat(src, period, shift))
	default:
		return decimal.NewFromFloat(indicators.SMAFloat(src, period, shift))
	}
}

func rsiOnArray(src indicators.BarSource, period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(indicators.RSIFloat(src, period, shift))
}

func bandsOnArray(src indicators.BarSource, period int, dev decimal.Decimal, shift int) (upper, lower decimal.Decimal) {
	return indicators.Envelopes(src, period, dev, "sma", 0, shift)
}

func stdDevOnArray(src indicators.BarSource, period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(indicators.StdDevFloat(src, period, shift))
}

func momentumOnArray(src indicators.BarSource, period, shift int) decimal.Decimal {
	return indicators.Momentum(src, period, shift)
}

func cciOnArray(src indicators.BarSource, period, shift int) decimal.Decimal {
	return indicators.CCI(src, period, shift)
}

func macdOnArray(src indicators.BarSource, fastP, slowP, signalP, shift int) decimal.Decimal {
	return indicators.MACD(src, fastP, slowP, shift)
}
