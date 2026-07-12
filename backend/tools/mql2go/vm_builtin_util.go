package mql2go

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// VM builtin helper functions and utility builtins.
// Extracted from vm_builtin_impls.go to keep file sizes under the 450-line limit.

// argD returns arg i as decimal, defaulting to zero if out of range.
func argD(args []interp.Value, i int) decimal.Decimal {
	if i >= len(args) {
		return decimal.Zero
	}
	return args[i].ToDecimal()
}

// argI returns arg i as int32, defaulting to 0 if out of range.
func argI(args []interp.Value, i int) int32 {
	if i >= len(args) {
		return 0
	}
	return args[i].ToInt()
}

// argS returns arg i as string, defaulting to "" if out of range.
func argS(args []interp.Value, i int) string {
	if i >= len(args) {
		return ""
	}
	return args[i].ToString()
}

// safeDecimalFromFloat converts a float64 to decimal.Decimal,
// returning decimal.Zero for ±Inf or NaN inputs to prevent panics.
func safeDecimalFromFloat(f float64) decimal.Decimal {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return decimal.Zero
	}
	return decimal.NewFromFloat(f)
}

// ── Utility builtins ─────────────────────────────────────────────────

func builtinNormalizeDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	value := argD(args, 0)
	digits := int(argI(args, 1))
	return interp.DecimalVal(value.Round(int32(digits))), nil
}

func builtinDoubleToString(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(argD(args, 0).String()), nil
}

func builtinIntegerToString(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(fmt.Sprintf("%d", argI(args, 0))), nil
}

func builtinStringToDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	d, err := decimal.NewFromString(argS(args, 0))
	if err != nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(d), nil
}

func builtinStringToInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	var n int32
	fmt.Sscanf(argS(args, 0), "%d", &n)
	return interp.IntVal(n), nil
}

func builtinTimeCurrent(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(vm.ctx.ServerTime() / 1000)), nil
}

func builtinNoop(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.NoneVal(), nil
}

func builtinNoopBool(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinNoopInt(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinNoopDecimal(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinNoopString(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(""), nil
}

func builtinEventSetTimer(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx != nil {
		vm.ctx.SetTimer(int(argI(args, 0)))
	}
	return interp.NoneVal(), nil
}

func builtinEventKillTimer(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx != nil {
		vm.ctx.KillTimer()
	}
	return interp.NoneVal(), nil
}
