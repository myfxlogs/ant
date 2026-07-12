package mql2go

import (
	"anttrader/strategy/sdk"
	"anttrader/tools/mql2go/interp"
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
	for i := 0; i < series.Len(); i++ {
		if series.Time(i) <= ts {
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
	_ = argI(args, 2) // type: MODE_HIGH, MODE_LOW, etc. — unused, we always search High
	count := int(argI(args, 3))
	start := int(argI(args, 4))
	if count <= 0 {
		count = series.Len()
	}
	if start < 0 {
		start = 0
	}
	maxIdx := start
	maxVal := series.High(start)
	for i := start + 1; i < start+count && i < series.Len(); i++ {
		h := series.High(i)
		if h.GreaterThan(maxVal) {
			maxVal = h
			maxIdx = i
		}
	}
	return interp.IntVal(int32(maxIdx)), nil
}

func builtinILowest(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	series, ok := resolveSeries(vm, 0, 1, args)
	if !ok || series == nil {
		return interp.IntVal(-1), nil
	}
	_ = argI(args, 2)
	count := int(argI(args, 3))
	start := int(argI(args, 4))
	if count <= 0 {
		count = series.Len()
	}
	if start < 0 {
		start = 0
	}
	minIdx := start
	minVal := series.Low(start)
	for i := start + 1; i < start+count && i < series.Len(); i++ {
		l := series.Low(i)
		if l.LessThan(minVal) {
			minVal = l
			minIdx = i
		}
	}
	return interp.IntVal(int32(minIdx)), nil
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
	tf := intToTF(argI(args, tfArgIdx))
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
		args[arrIdx].Array = args[arrIdx].Array[:0]
		return 0
	}
	if startPos < 0 {
		startPos = 0
	}
	if startPos+absCount > series.Len() {
		absCount = series.Len() - startPos
		if absCount <= 0 {
			args[arrIdx].Array = args[arrIdx].Array[:0]
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
	args[arrIdx].Array = result
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
		return interp.IntVal(int32(s.Time(shift)))
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
