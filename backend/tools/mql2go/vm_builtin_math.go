package mql2go

import (
	"math"
	"math/rand"

	"alphaforge/tools/mql2go/interp"
)

// MQL4/MQL5 Math functions — complete implementation.
// All trigonometric functions use float64 internally (math package) and
// return DecimalVal for consistency with the VM's decimal-based arithmetic.

func builtinMathCos(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(safeDecimalFromFloat(math.Cos(argD(args, 0).InexactFloat64()))), nil
}

func builtinMathSin(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(safeDecimalFromFloat(math.Sin(argD(args, 0).InexactFloat64()))), nil
}

func builtinMathTan(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(safeDecimalFromFloat(math.Tan(argD(args, 0).InexactFloat64()))), nil
}

func builtinMathArccos(vm *VM, args []interp.Value) (interp.Value, error) {
	f := argD(args, 0).InexactFloat64()
	if f < -1 || f > 1 {
		return interp.DecimalVal(decimalZero), nil
	}
	return interp.DecimalVal(safeDecimalFromFloat(math.Acos(f))), nil
}

func builtinMathArcsin(vm *VM, args []interp.Value) (interp.Value, error) {
	f := argD(args, 0).InexactFloat64()
	if f < -1 || f > 1 {
		return interp.DecimalVal(decimalZero), nil
	}
	return interp.DecimalVal(safeDecimalFromFloat(math.Asin(f))), nil
}

func builtinMathArctan(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(safeDecimalFromFloat(math.Atan(argD(args, 0).InexactFloat64()))), nil
}

func builtinMathLog10(vm *VM, args []interp.Value) (interp.Value, error) {
	f := argD(args, 0).InexactFloat64()
	if f <= 0 {
		return interp.DecimalVal(decimalZero), nil
	}
	return interp.DecimalVal(safeDecimalFromFloat(math.Log10(f))), nil
}

func builtinMathMod(vm *VM, args []interp.Value) (interp.Value, error) {
	a := argD(args, 0)
	b := argD(args, 1)
	if b.IsZero() {
		return interp.DecimalVal(decimalZero), nil
	}
	return interp.DecimalVal(a.Mod(b)), nil
}

func builtinMathRand(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(int32(rand.Intn(32767))), nil
}

func builtinMathSrand(vm *VM, args []interp.Value) (interp.Value, error) {
	seed := int64(argI(args, 0))
	rand.Seed(seed)
	return interp.NoneVal(), nil
}

func builtinMathIsValidNumber(vm *VM, args []interp.Value) (interp.Value, error) {
	f := argD(args, 0).InexactFloat64()
	return interp.BoolVal(!math.IsInf(f, 0) && !math.IsNaN(f)), nil
}

// MQL4 lowercase math aliases — dispatch to the corresponding Math* function.
func builtinAliasCeil(vm *VM, args []interp.Value) (interp.Value, error)  { return builtinMathCeil(vm, args) }
func builtinAliasFloor(vm *VM, args []interp.Value) (interp.Value, error)  { return builtinMathFloor(vm, args) }
func builtinAliasCos(vm *VM, args []interp.Value) (interp.Value, error)    { return builtinMathCos(vm, args) }
func builtinAliasSin(vm *VM, args []interp.Value) (interp.Value, error)    { return builtinMathSin(vm, args) }
func builtinAliasTan(vm *VM, args []interp.Value) (interp.Value, error)    { return builtinMathTan(vm, args) }
func builtinAliasExp(vm *VM, args []interp.Value) (interp.Value, error)    { return builtinMathExp(vm, args) }
func builtinAliasFabs(vm *VM, args []interp.Value) (interp.Value, error)   { return builtinMathAbs(vm, args) }
func builtinAliasFmax(vm *VM, args []interp.Value) (interp.Value, error)   { return builtinMathMax(vm, args) }
func builtinAliasFmin(vm *VM, args []interp.Value) (interp.Value, error)   { return builtinMathMin(vm, args) }
func builtinAliasFmod(vm *VM, args []interp.Value) (interp.Value, error)   { return builtinMathMod(vm, args) }
func builtinAliasLog(vm *VM, args []interp.Value) (interp.Value, error)    { return builtinMathLog(vm, args) }
func builtinAliasLog10(vm *VM, args []interp.Value) (interp.Value, error)  { return builtinMathLog10(vm, args) }
func builtinAliasPow(vm *VM, args []interp.Value) (interp.Value, error)    { return builtinMathPow(vm, args) }
func builtinAliasRound(vm *VM, args []interp.Value) (interp.Value, error)  { return builtinMathRound(vm, args) }
func builtinAliasRand(vm *VM, args []interp.Value) (interp.Value, error)   { return builtinMathRand(vm, args) }
func builtinAliasSrand(vm *VM, args []interp.Value) (interp.Value, error)  { return builtinMathSrand(vm, args) }
func builtinAliasSqrt(vm *VM, args []interp.Value) (interp.Value, error)   { return builtinMathSqrt(vm, args) }

var decimalZero = safeDecimalFromFloat(0)
