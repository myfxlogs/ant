package mql2go

import (
	"anttrader/tools/mql2go/interp"
)

// MQL5 timeseries access functions.
// These provide compatibility with MQL5's Copy* and i* functions.
// In backtest, they delegate to the existing BarsTF / Bars infrastructure.

func builtinBars(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	sym := argS(args, 0)
	tf := periodToTimeframe(argI(args, 1))
	_ = sym // current symbol only in backtest
	if tf == "" || tf == vm.ctx.Timeframe() {
		return interp.IntVal(int32(vm.ctx.Bars().Len())), nil
	}
	return interp.IntVal(int32(vm.ctx.BarsTF(tf).Len())), nil
}

func builtinIBarShift(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	sym := argS(args, 0)
	_ = sym
	tf := periodToTimeframe(argI(args, 1))
	ts := int64(argI(args,2)) * 1000
	// Find the bar whose timestamp <= ts
	if tf == "" || tf == vm.ctx.Timeframe() {
		series := vm.ctx.Bars()
		for i := 0; i < series.Len(); i++ {
			if series.Time(i) <= ts {
				return interp.IntVal(int32(i)), nil
			}
		}
	} else {
		series := vm.ctx.BarsTF(tf)
		for i := 0; i < series.Len(); i++ {
			if series.Time(i) <= ts {
				return interp.IntVal(int32(i)), nil
			}
		}
	}
	return interp.IntVal(-1), nil
}

func builtinIHighest(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	sym := argS(args, 0)
	_ = sym
	_ = argI(args, 2) // type: MODE_HIGH, MODE_LOW, etc. — unused, we always search High
	count := int(argI(args, 3))
	start := int(argI(args, 4))
	// Simplified: search in current timeframe bars
	bs := vm.ctx.Bars()
	if count <= 0 {
		count = bs.Len()
	}
	if start < 0 {
		start = 0
	}
	maxIdx := start
	maxVal := bs.High(start)
	for i := start + 1; i < start+count && i < bs.Len(); i++ {
		h := bs.High(i)
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
	sym := argS(args, 0)
	_ = sym
	_ = argI(args, 1) // timeframe — simplified to current TF
	_ = argI(args, 2)
	count := int(argI(args, 3))
	start := int(argI(args, 4))
	bs := vm.ctx.Bars()
	if count <= 0 {
		count = bs.Len()
	}
	if start < 0 {
		start = 0
	}
	minIdx := start
	minVal := bs.Low(start)
	for i := start + 1; i < start+count && i < bs.Len(); i++ {
		l := bs.Low(i)
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
	sym := argS(args, 0)
	_ = sym
	tf := periodToTimeframe(argI(args, 1))
	shift := int(argI(args, 2))
	if tf == "" || tf == vm.ctx.Timeframe() {
		return interp.IntVal(int32(vm.ctx.Bars().Volume(shift))), nil
	}
	return interp.IntVal(int32(vm.ctx.BarsTF(tf).Volume(shift))), nil
}

func builtinIRealVolume(vm *VM, args []interp.Value) (interp.Value, error) {
	// Real volume not available in backtest — return 0
	return interp.DecimalVal(decimalZero), nil
}

func builtinISpread(vm *VM, args []interp.Value) (interp.Value, error) {
	// Spread not available per-bar in backtest — return 0
	return interp.IntVal(0), nil
}

// Copy* functions — return data count or -1 on failure.
// In MQL5, CopyClose(symbol, timeframe, start_pos, count, array) copies data into the array.
// In our VM, we return the count and leave the array argument unchanged (arrays are by-value).

func builtinCopyRates(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	count := int(argI(args, 3))
	if count <= 0 {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(count)), nil
}

func builtinCopyClose(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	count := int(argI(args, 3))
	if count <= 0 {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(count)), nil
}

func builtinCopyHigh(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	count := int(argI(args, 3))
	if count <= 0 {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(count)), nil
}

func builtinCopyLow(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	count := int(argI(args, 3))
	if count <= 0 {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(count)), nil
}

func builtinCopyOpen(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	count := int(argI(args, 3))
	if count <= 0 {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(count)), nil
}

func builtinCopyTime(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	count := int(argI(args, 3))
	if count <= 0 {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(count)), nil
}

func builtinCopyBuffer(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	count := int(argI(args, 4))
	if count <= 0 {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(count)), nil
}

func builtinCopyTickVolume(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(-1), nil
	}
	count := int(argI(args, 3))
	if count <= 0 {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(count)), nil
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
