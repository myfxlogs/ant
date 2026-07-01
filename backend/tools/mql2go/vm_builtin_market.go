package mql2go

import (
	"github.com/shopspring/decimal"

	"anttrader/tools/mql2go/interp"
)

// intToTF converts an MQL period int to a timeframe string.
func intToTF(period int32) string {
	switch period {
	case 1:
		return "M1"
	case 5:
		return "M5"
	case 15:
		return "M15"
	case 30:
		return "M30"
	case 60:
		return "H1"
	case 240:
		return "H4"
	case 1440:
		return "D1"
	case 10080:
		return "W1"
	case 43200:
		return "MN1"
	default:
		return "M1"
	}
}

// ── Cross-timeframe market data builtins ─────────────────────────────

// iClose(symbol, timeframe, shift) → decimal
func builtinIClose(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	tf := intToTF(argI(args, 1))
	shift := int(argI(args, 2))
	bars := vm.ctx.BarsTF(tf)
	if bars == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(bars.Close(shift)), nil
}

// iOpen(symbol, timeframe, shift) → decimal
func builtinIOpen(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	tf := intToTF(argI(args, 1))
	shift := int(argI(args, 2))
	bars := vm.ctx.BarsTF(tf)
	if bars == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(bars.Open(shift)), nil
}

// iHigh(symbol, timeframe, shift) → decimal
func builtinIHigh(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	tf := intToTF(argI(args, 1))
	shift := int(argI(args, 2))
	bars := vm.ctx.BarsTF(tf)
	if bars == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(bars.High(shift)), nil
}

// iLow(symbol, timeframe, shift) → decimal
func builtinILow(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	tf := intToTF(argI(args, 1))
	shift := int(argI(args, 2))
	bars := vm.ctx.BarsTF(tf)
	if bars == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(bars.Low(shift)), nil
}

// iTime(symbol, timeframe, shift) → int (unix seconds)
func builtinITime(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	tf := intToTF(argI(args, 1))
	shift := int(argI(args, 2))
	bars := vm.ctx.BarsTF(tf)
	if bars == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(bars.Time(shift) / 1000)), nil
}

// iVolume(symbol, timeframe, shift) → int
func builtinIVolume(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	tf := intToTF(argI(args, 1))
	shift := int(argI(args, 2))
	bars := vm.ctx.BarsTF(tf)
	if bars == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(bars.Volume(shift))), nil
}
