package mql2go

import (
	"math"
	"strings"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// Basic math and utility builtins extracted from vm_builtin_impls.go.
// VM-CODE-HYGIENE-1: split for file-lines compliance.

func builtinMathAbs(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(argD(args, 0).Abs()), nil
}

func builtinMathMax(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimal.Max(argD(args, 0), argD(args, 1))), nil
}

func builtinMathMin(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimal.Min(argD(args, 0), argD(args, 1))), nil
}

func builtinMathSqrt(vm *VM, args []interp.Value) (interp.Value, error) {
	f := argD(args, 0).InexactFloat64()
	if f < 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(safeDecimalFromFloat(math.Sqrt(f))), nil
}

func builtinMathPow(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(argD(args, 0).Pow(argD(args, 1))), nil
}

func builtinMathLog(vm *VM, args []interp.Value) (interp.Value, error) {
	f := argD(args, 0).InexactFloat64()
	if f <= 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(safeDecimalFromFloat(math.Log(f))), nil
}

func builtinMathRound(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(argD(args, 0).Round(0)), nil
}

func builtinMathFloor(vm *VM, args []interp.Value) (interp.Value, error) {
	d := argD(args, 0)
	return interp.DecimalVal(d.Sub(d.Mod(decimal.NewFromInt(1)))), nil
}

func builtinMathCeil(vm *VM, args []interp.Value) (interp.Value, error) {
	d := argD(args, 0)
	mod := d.Mod(decimal.NewFromInt(1))
	if mod.IsZero() {
		return interp.DecimalVal(d), nil
	}
	return interp.DecimalVal(d.Sub(mod).Add(decimal.NewFromInt(1))), nil
}

func builtinMathExp(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(safeDecimalFromFloat(math.Exp(argD(args, 0).InexactFloat64()))), nil
}

// ── Platform builtins ────────────────────────────────────────────────

func builtinPrint(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx != nil {
		parts := make([]string, len(args))
		for i, a := range args {
			parts[i] = a.ToString()
		}
		vm.ctx.Log(strings.Join(parts, " "))
	}
	return interp.NoneVal(), nil
}

// ── Market data series builtins (Close(), Open(), etc. without subscript) ──

func builtinSeriesClose(vm *VM, args []interp.Value) (interp.Value, error) {
	return vm.getSeriesHelper("Close", int(argI(args, 0))), nil
}

func builtinSeriesOpen(vm *VM, args []interp.Value) (interp.Value, error) {
	return vm.getSeriesHelper("Open", int(argI(args, 0))), nil
}

func builtinSeriesHigh(vm *VM, args []interp.Value) (interp.Value, error) {
	return vm.getSeriesHelper("High", int(argI(args, 0))), nil
}

func builtinSeriesLow(vm *VM, args []interp.Value) (interp.Value, error) {
	return vm.getSeriesHelper("Low", int(argI(args, 0))), nil
}

func builtinSeriesVolume(vm *VM, args []interp.Value) (interp.Value, error) {
	return vm.getSeriesHelper("Volume", int(argI(args, 0))), nil
}

func builtinSeriesTime(vm *VM, args []interp.Value) (interp.Value, error) {
	return vm.getSeriesHelper("Time", int(argI(args, 0))), nil
}

// ── Price data builtins ──────────────────────────────────────────────

func builtinBid(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(vm.ctx.Bid()), nil
}

func builtinAsk(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(vm.ctx.Ask()), nil
}

func builtinPoint(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(vm.ctx.Point()), nil
}

func builtinDigits(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(vm.ctx.Digits()), nil
}

func builtinSymbol(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.StringVal(""), nil
	}
	return interp.StringVal(vm.ctx.Symbol()), nil
}

func builtinPeriod(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	tf := vm.ctx.Timeframe()
	return interp.IntVal(tfToInt(tf)), nil
}

// builtinOperatorIn implements the Python `in` / `not in` operator.
// args[0] = left (needle), args[1] = right (haystack).
// Supports: string substring, array membership, scalar equality.
func builtinOperatorIn(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 2 {
		return interp.BoolVal(false), nil
	}
	left := args[0]
	right := args[1]
	switch right.Kind {
	case interp.ValString:
		return interp.BoolVal(strings.Contains(right.Str, left.ToString())), nil
	case interp.ValArray:
		for _, v := range right.ArrayData() {
			if v.Equal(left) {
				return interp.BoolVal(true), nil
			}
		}
		return interp.BoolVal(false), nil
	default:
		return interp.BoolVal(right.Equal(left)), nil
	}
}

// builtinSpread returns the current spread in points (Ask - Bid) / Point.
func builtinSpread(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	ask := vm.ctx.Ask()
	bid := vm.ctx.Bid()
	point := vm.ctx.Point()
	if point.IsZero() {
		return interp.IntVal(0), nil
	}
	spread := ask.Sub(bid).Div(point)
	return interp.IntVal(int32(spread.IntPart())), nil
}

func tfToInt(tf string) int32 {
	switch tf {
	case "M1":
		return 1
	case "M5":
		return 5
	case "M15":
		return 15
	case "M30":
		return 30
	case "H1":
		return 60
	case "H4":
		return 240
	case "D1":
		return 1440
	case "W1":
		return 10080
	case "MN1":
		return 43200
	default:
		return 0
	}
}
