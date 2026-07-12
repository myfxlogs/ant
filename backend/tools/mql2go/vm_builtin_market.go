package mql2go

import (
	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
	"anttrader/tools/mql2go/interp"
)

// intToTF converts an MQL period int to a timeframe string.
// period=0 means PERIOD_CURRENT (primary timeframe) → returns "".
// Covers both MQL4 and MQL5 period constants.
func intToTF(period int32) string {
	if period == 0 {
		return ""
	}
	switch period {
	case 1:
		return "M1"
	case 2:
		return "M2"
	case 3:
		return "M3"
	case 4:
		return "M4"
	case 5:
		return "M5"
	case 6:
		return "M6"
	case 10:
		return "M10"
	case 12:
		return "M12"
	case 15:
		return "M15"
	case 20:
		return "M20"
	case 30:
		return "M30"
	case 60:
		return "H1"
	case 120:
		return "H2"
	case 180:
		return "H3"
	case 240:
		return "H4"
	case 360:
		return "H6"
	case 480:
		return "H8"
	case 720:
		return "H12"
	case 1440:
		return "D1"
	case 10080:
		return "W1"
	case 43200:
		return "MN1"
	default:
		return ""
	}
}

// ── Cross-timeframe / cross-symbol market data builtins ──────────────
// iClose(symbol, timeframe, shift) → decimal
// These now support multi-symbol: when symbol != primary, delegates to BarsForSymbol.

func builtinIClose(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	sym := argS(args, 0)
	tf := intToTF(argI(args, 1))
	shift := int(argI(args, 2))
	bars := resolveBarSeries(vm, sym, tf)
	if bars == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(bars.Close(shift)), nil
}

func builtinIOpen(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	sym := argS(args, 0)
	tf := intToTF(argI(args, 1))
	shift := int(argI(args, 2))
	bars := resolveBarSeries(vm, sym, tf)
	if bars == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(bars.Open(shift)), nil
}

func builtinIHigh(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	sym := argS(args, 0)
	tf := intToTF(argI(args, 1))
	shift := int(argI(args, 2))
	bars := resolveBarSeries(vm, sym, tf)
	if bars == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(bars.High(shift)), nil
}

func builtinILow(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	sym := argS(args, 0)
	tf := intToTF(argI(args, 1))
	shift := int(argI(args, 2))
	bars := resolveBarSeries(vm, sym, tf)
	if bars == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(bars.Low(shift)), nil
}

func builtinITime(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	sym := argS(args, 0)
	tf := intToTF(argI(args, 1))
	shift := int(argI(args, 2))
	bars := resolveBarSeries(vm, sym, tf)
	if bars == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(bars.Time(shift) / 1000)), nil
}

func builtinIVolume(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	sym := argS(args, 0)
	tf := intToTF(argI(args, 1))
	shift := int(argI(args, 2))
	bars := resolveBarSeries(vm, sym, tf)
	if bars == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(bars.Volume(shift))), nil
}

// resolveBarSeries returns the BarSeries for the given symbol and timeframe.
// sym="" or sym==primary → use Bars()/BarsTF() for the primary symbol.
// sym != primary → use BarsForSymbol(sym, tf) for multi-symbol access.
func resolveBarSeries(vm *VM, sym, tf string) sdk.BarSeries {
	if sym == "" || sym == vm.ctx.Symbol() {
		if tf == "" || tf == vm.ctx.Timeframe() {
			return vm.ctx.Bars()
		}
		return vm.ctx.BarsTF(tf)
	}
	return vm.ctx.BarsForSymbol(sym, tf)
}
