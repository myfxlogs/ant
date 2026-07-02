package mql2go

import (
	"math"

	"github.com/shopspring/decimal"

	"anttrader/tools/mql2go/interp"
)

// ── *OnArray indicator variants ──────────────────────────────────────
// These compute indicators on user-provided arrays instead of symbol/timeframe data.
// MQL signature: iXOnArray(array[], total, ...params, shift)
// total=0 means use the entire array. Array is indexed from oldest (0) to newest (len-1).
// shift=0 means the most recent element.

// arrayToDecimals converts a VM array Value to a slice of decimal.Decimal.
func arrayToDecimals(arr interp.Value) []decimal.Decimal {
	if arr.Kind != interp.ValArray {
		return nil
	}
	out := make([]decimal.Decimal, len(arr.Array))
	for i, v := range arr.Array {
		out[i] = v.ToDecimal()
	}
	return out
}

// sliceForShift returns the data slice ending at (len-shift) from the end.
// MQL arrays are indexed oldest→newest; shift=0 is the latest bar.
func sliceForShift(data []decimal.Decimal, total, shift int) []decimal.Decimal {
	n := len(data)
	if total > 0 && total < n {
		n = total
	}
	end := n - shift
	if end <= 0 {
		return nil
	}
	return data[:end]
}

// sma computes Simple Moving Average over the last `period` elements.
func sma(data []decimal.Decimal, period int) decimal.Decimal {
	if len(data) < period || period <= 0 {
		return decimal.Zero
	}
	sum := decimal.Zero
	for i := len(data) - period; i < len(data); i++ {
		sum = sum.Add(data[i])
	}
	return sum.Div(decimal.NewFromInt(int64(period)))
}

// ema computes Exponential Moving Average over the data slice.
func ema(data []decimal.Decimal, period int) decimal.Decimal {
	if len(data) < period || period <= 0 {
		return decimal.Zero
	}
	mult := decimal.NewFromFloat(2.0 / float64(period+1))
	result := sma(data[:period], period)
	for i := period; i < len(data); i++ {
		result = data[i].Mul(mult).Add(result.Mul(decimal.NewFromInt(1).Sub(mult)))
	}
	return result
}

// smma computes Smoothed Moving Average.
func smma(data []decimal.Decimal, period int) decimal.Decimal {
	if len(data) < period || period <= 0 {
		return decimal.Zero
	}
	prev := sma(data[:period], period)
	for i := period; i < len(data); i++ {
		prev = prev.Mul(decimal.NewFromInt(int64(period-1))).Add(data[i]).Div(decimal.NewFromInt(int64(period)))
	}
	return prev
}

// lwma computes Linearly Weighted Moving Average.
func lwma(data []decimal.Decimal, period int) decimal.Decimal {
	if len(data) < period || period <= 0 {
		return decimal.Zero
	}
	sum := decimal.Zero
	weightSum := int64(0)
	for i := 0; i < period; i++ {
		w := int64(period - i)
		weightSum += w
		sum = sum.Add(data[len(data)-period+i].Mul(decimal.NewFromInt(w)))
	}
	return sum.Div(decimal.NewFromInt(weightSum))
}

// maOnArray computes a moving average of the given method on data.
func maOnArray(data []decimal.Decimal, period int, method string) decimal.Decimal {
	switch method {
	case "ema":
		return ema(data, period)
	case "smma":
		return smma(data, period)
	case "lwma":
		return lwma(data, period)
	default:
		return sma(data, period)
	}
}

// builtinIMAOnArray: iMAOnArray(array[], total, ma_period, ma_shift, ma_method, shift)
func builtinIMAOnArray(vm *VM, args []interp.Value) (interp.Value, error) {
	data := arrayToDecimals(args[0])
	if len(data) == 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	total := int(argI(args, 1))
	maPeriod := int(argI(args, 2))
	maMethod := maMethodName(argI(args, 4))
	shift := int(argI(args, 5))
	slice := sliceForShift(data, total, shift)
	return interp.DecimalVal(maOnArray(slice, maPeriod, maMethod)), nil
}

// builtinIRSIOnArray: iRSIOnArray(array[], total, rsi_period, shift)
func builtinIRSIOnArray(vm *VM, args []interp.Value) (interp.Value, error) {
	data := arrayToDecimals(args[0])
	if len(data) < 2 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	total := int(argI(args, 1))
	period := int(argI(args, 2))
	shift := int(argI(args, 3))
	slice := sliceForShift(data, total, shift)
	if len(slice) < period+1 || period <= 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}

	avgGain := decimal.Zero
	avgLoss := decimal.Zero
	for i := 1; i <= period; i++ {
		diff := slice[i].Sub(slice[i-1])
		if diff.GreaterThan(decimal.Zero) {
			avgGain = avgGain.Add(diff)
		} else {
			avgLoss = avgLoss.Sub(diff)
		}
	}
	avgGain = avgGain.Div(decimal.NewFromInt(int64(period)))
	avgLoss = avgLoss.Div(decimal.NewFromInt(int64(period)))

	for i := period + 1; i < len(slice); i++ {
		diff := slice[i].Sub(slice[i-1])
		gain := decimal.Zero
		loss := decimal.Zero
		if diff.GreaterThan(decimal.Zero) {
			gain = diff
		} else {
			loss = diff.Neg()
		}
		avgGain = avgGain.Mul(decimal.NewFromInt(int64(period-1))).Add(gain).Div(decimal.NewFromInt(int64(period)))
		avgLoss = avgLoss.Mul(decimal.NewFromInt(int64(period-1))).Add(loss).Div(decimal.NewFromInt(int64(period)))
	}

	if avgLoss.IsZero() {
		return interp.DecimalVal(decimal.NewFromInt(100)), nil
	}
	rs := avgGain.Div(avgLoss)
	rsi := decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))
	return interp.DecimalVal(rsi), nil
}

// builtinIATROnArray: iATROnArray(array[], total, atr_period, shift)
func builtinIATROnArray(vm *VM, args []interp.Value) (interp.Value, error) {
	data := arrayToDecimals(args[0])
	if len(data) < 2 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	total := int(argI(args, 1))
	period := int(argI(args, 2))
	shift := int(argI(args, 3))
	slice := sliceForShift(data, total, shift)
	if len(slice) < period+1 || period <= 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}

	trs := make([]decimal.Decimal, 0, len(slice)-1)
	for i := 1; i < len(slice); i++ {
		tr := slice[i].Sub(slice[i-1]).Abs()
		trs = append(trs, tr)
	}
	if len(trs) < period {
		return interp.DecimalVal(decimal.Zero), nil
	}

	atr := decimal.Zero
	for i := 0; i < period; i++ {
		atr = atr.Add(trs[i])
	}
	atr = atr.Div(decimal.NewFromInt(int64(period)))

	for i := period; i < len(trs); i++ {
		atr = atr.Mul(decimal.NewFromInt(int64(period-1))).Add(trs[i]).Div(decimal.NewFromInt(int64(period)))
	}
	return interp.DecimalVal(atr), nil
}

// builtinIBandsOnArray: iBandsOnArray(array[], total, bands_period, deviation, bands_shift, mode, shift)
func builtinIBandsOnArray(vm *VM, args []interp.Value) (interp.Value, error) {
	data := arrayToDecimals(args[0])
	if len(data) == 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	total := int(argI(args, 1))
	period := int(argI(args, 2))
	deviation := argD(args, 3)
	mode := argI(args, 5)
	shift := int(argI(args, 6))
	slice := sliceForShift(data, total, shift)
	if len(slice) < period || period <= 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}

	mid := sma(slice, period)
	sumSq := decimal.Zero
	for i := len(slice) - period; i < len(slice); i++ {
		diff := slice[i].Sub(mid)
		sumSq = sumSq.Add(diff.Mul(diff))
	}
	variance := sumSq.Div(decimal.NewFromInt(int64(period)))
	stdDev := decimal.NewFromFloat(math.Sqrt(variance.InexactFloat64()))
	upper := mid.Add(stdDev.Mul(deviation))
	lower := mid.Sub(stdDev.Mul(deviation))

	switch mode {
	case 1:
		return interp.DecimalVal(upper), nil
	case 2:
		return interp.DecimalVal(lower), nil
	default:
		return interp.DecimalVal(mid), nil
	}
}

// builtinIStdDevOnArray: iStdDevOnArray(array[], total, ma_period, ma_shift, ma_method, shift)
func builtinIStdDevOnArray(vm *VM, args []interp.Value) (interp.Value, error) {
	data := arrayToDecimals(args[0])
	if len(data) == 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	total := int(argI(args, 1))
	period := int(argI(args, 2))
	maMethod := maMethodName(argI(args, 4))
	shift := int(argI(args, 5))
	slice := sliceForShift(data, total, shift)
	if len(slice) < period || period <= 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}

	mid := maOnArray(slice, period, maMethod)
	sumSq := decimal.Zero
	for i := len(slice) - period; i < len(slice); i++ {
		diff := slice[i].Sub(mid)
		sumSq = sumSq.Add(diff.Mul(diff))
	}
	variance := sumSq.Div(decimal.NewFromInt(int64(period)))
	stdDev := decimal.NewFromFloat(math.Sqrt(variance.InexactFloat64()))
	return interp.DecimalVal(stdDev), nil
}

// builtinIMomentumOnArray: iMomentumOnArray(array[], total, mom_period, shift)
func builtinIMomentumOnArray(vm *VM, args []interp.Value) (interp.Value, error) {
	data := arrayToDecimals(args[0])
	if len(data) < 2 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	total := int(argI(args, 1))
	period := int(argI(args, 2))
	shift := int(argI(args, 3))
	slice := sliceForShift(data, total, shift)
	if len(slice) < period+1 || period <= 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	cur := slice[len(slice)-1]
	old := slice[len(slice)-1-period]
	if old.IsZero() {
		return interp.DecimalVal(decimal.NewFromInt(100)), nil
	}
	mom := cur.Div(old).Mul(decimal.NewFromInt(100))
	return interp.DecimalVal(mom), nil
}

// builtinICCIOnArray: iCCIOnArray(array[], total, cci_period, shift)
func builtinICCIOnArray(vm *VM, args []interp.Value) (interp.Value, error) {
	data := arrayToDecimals(args[0])
	if len(data) < 2 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	total := int(argI(args, 1))
	period := int(argI(args, 2))
	shift := int(argI(args, 3))
	slice := sliceForShift(data, total, shift)
	if len(slice) < period || period <= 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}

	typical := slice[len(slice)-1]
	smaVal := sma(slice, period)
	meanDev := decimal.Zero
	for i := len(slice) - period; i < len(slice); i++ {
		meanDev = meanDev.Add(slice[i].Sub(smaVal).Abs())
	}
	meanDev = meanDev.Div(decimal.NewFromInt(int64(period)))
	if meanDev.IsZero() {
		return interp.DecimalVal(decimal.Zero), nil
	}
	cci := typical.Sub(smaVal).Div(meanDev).Div(decimal.NewFromFloat(0.015))
	return interp.DecimalVal(cci), nil
}

// builtinIMACDOnArray: iMACDOnArray(array[], total, fast_ema, slow_ema, signal, mode, shift)
func builtinIMACDOnArray(vm *VM, args []interp.Value) (interp.Value, error) {
	data := arrayToDecimals(args[0])
	if len(data) < 2 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	total := int(argI(args, 1))
	fast := int(argI(args, 2))
	slow := int(argI(args, 3))
	signal := int(argI(args, 4))
	mode := argI(args, 5)
	shift := int(argI(args, 6))
	slice := sliceForShift(data, total, shift)
	if len(slice) < slow || slow <= 0 || fast <= 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}

	macdLine := ema(slice, fast).Sub(ema(slice, slow))

	if mode == 1 {
		// Signal line = EMA of MACD values. We approximate by computing EMA of the
		// difference series over the available data.
		if len(slice) < slow+signal {
			return interp.DecimalVal(decimal.Zero), nil
		}
		macdSeries := make([]decimal.Decimal, 0, len(slice)-slow+1)
		for i := slow; i <= len(slice); i++ {
			macdSeries = append(macdSeries, ema(slice[:i], fast).Sub(ema(slice[:i], slow)))
		}
		return interp.DecimalVal(ema(macdSeries, signal)), nil
	}
	return interp.DecimalVal(macdLine), nil
}
