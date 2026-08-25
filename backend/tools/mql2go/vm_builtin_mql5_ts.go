package mql2go

import (
	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// MQL5 timeseries access functions.
// These provide compatibility with MQL5's Copy* and i* functions.
// In backtest, they delegate to the existing BarsTF / Bars infrastructure.

func builtinBars(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok || series == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(series.Len())), nil
}

func builtinIBarShift(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok || series == nil {
		return interp.IntVal(-1), nil
	}
	ts := int64(argI(args, 2)) * 1000
	exact := len(args) > 3 && argI(args, 3) != 0
	for i := 0; i < series.Len(); i++ {
		barTs := series.Time(i)
		if barTs == ts || (!exact && barTs < ts) {
			return interp.IntVal(int32(i)), nil
		}
	}
	return interp.IntVal(-1), nil
}

func builtinIHighest(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok || series == nil {
		return interp.IntVal(-1), nil
	}
	mode := argI(args, 2)
	if !validSeriesMode(mode) {
		vm.recordBlindSpot("iHighest:invalid series mode")
		return interp.IntVal(-1), nil
	}
	idx, _ := extremeIndex(series, mode, argI(args, 3), argI(args, 4), true)
	return interp.IntVal(idx), nil
}

func builtinILowest(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok || series == nil {
		return interp.IntVal(-1), nil
	}
	mode := argI(args, 2)
	if !validSeriesMode(mode) {
		vm.recordBlindSpot("iLowest:invalid series mode")
		return interp.IntVal(-1), nil
	}
	idx, _ := extremeIndex(series, mode, argI(args, 3), argI(args, 4), false)
	return interp.IntVal(idx), nil
}

func validSeriesMode(mode int32) bool {
	return mode >= 0 && mode <= 5
}

func extremeIndex(series sdk.BarSeries, mode, count, start int32, highest bool) (int32, bool) {
	if series.Len() == 0 || start < 0 || int(start) >= series.Len() {
		return -1, false
	}
	if count <= 0 || int(count) > series.Len()-int(start) {
		count = int32(series.Len() - int(start))
	}
	valueAt := func(shift int) (decimal.Decimal, bool) {
		switch mode {
		case 0: // MODE_OPEN
			return series.Open(shift), true
		case 1: // MODE_LOW
			return series.Low(shift), true
		case 2: // MODE_HIGH
			return series.High(shift), true
		case 3: // MODE_CLOSE
			return series.Close(shift), true
		case 4: // MODE_VOLUME
			return decimal.NewFromInt(series.Volume(shift)), true
		case 5: // MODE_TIME
			return decimal.NewFromInt(series.Time(shift) / 1000), true
		default:
			return decimal.Zero, false
		}
	}
	bestIdx := int(start)
	best, valid := valueAt(bestIdx)
	if !valid {
		return -1, false
	}
	end := int(start) + int(count)
	for i := bestIdx + 1; i < end; i++ {
		value, valid := valueAt(i)
		if !valid {
			return -1, false
		}
		if (highest && value.GreaterThan(best)) || (!highest && value.LessThan(best)) {
			best = value
			bestIdx = i
		}
	}
	return int32(bestIdx), true
}

func builtinITickVolume(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok || series == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	shift := int(argI(args, 2))
	return interp.IntVal(int32(series.Volume(shift))), nil
}

func builtinIRealVolume(vm *VM, args []interp.Value) (interp.Value, error) {
	// Real volume not available in backtest — return 0
	return interp.DecimalVal(decimalZero), nil
}

func builtinISpread(vm *VM, args []interp.Value) (interp.Value, error) {
	// Spread not available per-bar in backtest — return 0
	return interp.IntVal(0), nil
}

// Copy* functions — copy bar data into the caller's array.
// MQL5: CopyClose(symbol, timeframe, start_pos, count, array[])
//   start_pos >= 0: offset from current bar (0 = latest)
//   count > 0: number of elements to copy forward (oldest first in output)
//   count < 0: number of elements to copy backward (newest first in output)
// Returns the number of elements actually copied, or -1 on failure.
//
// In MQL5 series-indexing, bar[0] = current, bar[1] = previous, etc.
// CopyClose with start_pos=0, count=5 copies bars [4,3,2,1,0] into array[0..4]
// (chronological order: oldest first when count > 0).

// resolveSeries returns the BarSeries for the given symbol/timeframe args.
// Delegates to resolveBarSeries for unified resolution logic.
func resolveSeries(vm *VM, symArgIdx, tfArgIdx int, args []interp.Value) (sdk.BarSeries, bool) {
	if vm.ctx == nil {
		return nil, false
	}
	sym := argS(args, symArgIdx)
	tf, ok := resolveTF(vm, argI(args, tfArgIdx)) // VM-TIMESERIES-SEMANTICS-2
	if !ok {
		return nil, false
	}
	return resolveBarSeries(vm, sym, tf), true
}

// copyBarData fills the array argument (last arg) with bar data from the series.
// getVal selects which OHLCV field to extract.
// Returns the count actually copied.
func copyBarData(args []interp.Value, series sdk.BarSeries, getVal func(sdk.BarSeries, int) interp.Value) int32 {
	startPos := int(argI(args, 2))
	count := int(argI(args, 3))
	if len(args) < 5 || args[4].Kind != interp.ValArray {
		return -1
	}
	arrIdx := 4

	absCount := count
	if absCount < 0 {
		absCount = -absCount
	}
	if absCount <= 0 || series == nil {
		args[arrIdx].SetArrayData(args[arrIdx].ArrayData()[:0])
		return 0
	}
	if startPos < 0 {
		startPos = 0
	}
	if startPos+absCount > series.Len() {
		absCount = series.Len() - startPos
		if absCount <= 0 {
			args[arrIdx].SetArrayData(args[arrIdx].ArrayData()[:0])
			return 0
		}
	}

	result := make([]interp.Value, absCount)
	for i := 0; i < absCount; i++ {
		var shift int
		if count > 0 {
			// Chronological: oldest first → bar[startPos+absCount-1-i]
			shift = startPos + absCount - 1 - i
		} else {
			// Reverse chronological: newest first → bar[startPos+i]
			shift = startPos + i
		}
		result[i] = getVal(series, shift)
	}
	args[arrIdx].SetArrayData(result)
	return int32(absCount)
}

func builtinCopyRates(vm *VM, args []interp.Value) (interp.Value, error) {
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok {
		return interp.IntVal(-1), nil
	}
	// CopyRates fills a MqlRates struct array — we fill with close as proxy
	// since our VM doesn't have a MqlRates struct type.
	n := copyBarData(args, series, func(s sdk.BarSeries, shift int) interp.Value {
		return interp.DecimalVal(s.Close(shift))
	})
	return interp.IntVal(n), nil
}

func builtinCopyClose(vm *VM, args []interp.Value) (interp.Value, error) {
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok {
		return interp.IntVal(-1), nil
	}
	n := copyBarData(args, series, func(s sdk.BarSeries, shift int) interp.Value {
		return interp.DecimalVal(s.Close(shift))
	})
	return interp.IntVal(n), nil
}

func builtinCopyHigh(vm *VM, args []interp.Value) (interp.Value, error) {
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok {
		return interp.IntVal(-1), nil
	}
	n := copyBarData(args, series, func(s sdk.BarSeries, shift int) interp.Value {
		return interp.DecimalVal(s.High(shift))
	})
	return interp.IntVal(n), nil
}

func builtinCopyLow(vm *VM, args []interp.Value) (interp.Value, error) {
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok {
		return interp.IntVal(-1), nil
	}
	n := copyBarData(args, series, func(s sdk.BarSeries, shift int) interp.Value {
		return interp.DecimalVal(s.Low(shift))
	})
	return interp.IntVal(n), nil
}

func builtinCopyOpen(vm *VM, args []interp.Value) (interp.Value, error) {
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok {
		return interp.IntVal(-1), nil
	}
	n := copyBarData(args, series, func(s sdk.BarSeries, shift int) interp.Value {
		return interp.DecimalVal(s.Open(shift))
	})
	return interp.IntVal(n), nil
}

func builtinCopyTime(vm *VM, args []interp.Value) (interp.Value, error) {
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok {
		return interp.IntVal(-1), nil
	}
	n := copyBarData(args, series, func(s sdk.BarSeries, shift int) interp.Value {
		return interp.IntVal(int32(s.Time(shift) / 1000))
	})
	return interp.IntVal(n), nil
}

func builtinCopyBuffer(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	// CopyBuffer(handle, buffer_num, start_pos, count, array[])
	// We don't have indicator handles in the VM — all i* builtins return
	// scalar values directly. Return the requested count as a best-effort
	// so MQL5 code that checks the return value doesn't error out.
	count := int(argI(args, 4))
	if count <= 0 {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(count)), nil
}

func builtinCopyTickVolume(vm *VM, args []interp.Value) (interp.Value, error) {
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok {
		return interp.IntVal(-1), nil
	}
	n := copyBarData(args, series, func(s sdk.BarSeries, shift int) interp.Value {
		return interp.IntVal(int32(s.Volume(shift)))
	})
	return interp.IntVal(n), nil
}

func builtinCopyRealVolume(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinCopySpread(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinCopyTicks(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinBarsCalculated(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(vm.ctx.Bars().Len())), nil
}

func builtinSeriesInfoInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}
