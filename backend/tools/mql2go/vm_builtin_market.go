package mql2go

import (
	"fmt"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// intToTF converts an MQL period int to a timeframe string.
// period=0 means PERIOD_CURRENT (primary timeframe) → returns ("", true).
// VM-TIMESERIES-SEMANTICS-2: unknown/illegal periods return ("", false) so
// callers can distinguish PERIOD_CURRENT from invalid values and fail-closed
// instead of silently falling back to the primary timeframe.
func intToTF(period int32) (string, bool) {
	if period == 0 {
		return "", true // PERIOD_CURRENT
	}
	switch period {
	case 1:
		return "M1", true
	case 2:
		return "M2", true
	case 3:
		return "M3", true
	case 4:
		return "M4", true
	case 5:
		return "M5", true
	case 6:
		return "M6", true
	case 10:
		return "M10", true
	case 12:
		return "M12", true
	case 15:
		return "M15", true
	case 20:
		return "M20", true
	case 30:
		return "M30", true
	case 60:
		return "H1", true
	case 120:
		return "H2", true
	case 180:
		return "H3", true
	case 240:
		return "H4", true
	case 360:
		return "H6", true
	case 480:
		return "H8", true
	case 720:
		return "H12", true
	case 1440:
		return "D1", true
	case 10080:
		return "W1", true
	case 43200:
		return "MN1", true
	default:
		// VM-TIMESERIES-SEMANTICS-2: illegal period — return false so callers
		// can record a blind spot / fail-closed instead of silent fallback.
		return "", false
	}
}

// ── Cross-timeframe / cross-symbol market data builtins ──────────────
// iClose(symbol, timeframe, shift) → decimal
// These now support multi-symbol: when symbol != primary, delegates to BarsForSymbol.

// resolveTF validates the timeframe argument and records a blind spot for
// illegal periods. VM-TIMESERIES-SEMANTICS-2: returns false for illegal
// periods so callers can fail-closed instead of silent fallback.
func resolveTF(vm *VM, period int32) (string, bool) {
	tf, ok := intToTF(period)
	if !ok {
		// VM-TIMESERIES-SEMANTICS-3: illegal timeframe is a fatal error.
		// callBuiltin will detect fatalError and return an error.
		vm.fatalError = fmt.Sprintf("illegal timeframe period %d", period)
		return "", false
	}
	return tf, true
}

func builtinIClose(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	sym := argS(args, 0)
	tf, ok := resolveTF(vm, argI(args, 1))
	if !ok {
		return interp.DecimalVal(decimal.Zero), nil
	}
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
	tf, ok := resolveTF(vm, argI(args, 1))
	if !ok {
		return interp.DecimalVal(decimal.Zero), nil
	}
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
	tf, ok := resolveTF(vm, argI(args, 1))
	if !ok {
		return interp.DecimalVal(decimal.Zero), nil
	}
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
	tf, ok := resolveTF(vm, argI(args, 1))
	if !ok {
		return interp.DecimalVal(decimal.Zero), nil
	}
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
	tf, ok := resolveTF(vm, argI(args, 1))
	if !ok {
		return interp.IntVal(0), nil
	}
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
	tf, ok := resolveTF(vm, argI(args, 1))
	if !ok {
		return interp.IntVal(0), nil
	}
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
