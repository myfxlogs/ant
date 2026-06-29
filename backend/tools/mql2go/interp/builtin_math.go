package interp

import (
	"math"
	"math/rand"

	"github.com/shopspring/decimal"
)

// Math builtin functions — MQL5 official list + MQL4 lowercase aliases.
// All use float64 internally (decimal→float64→decimal round-trip), same as
// the existing MathSqrt/MathPow/MathLog in builtins.go.

func init() {
	builtinTable["MathCeil"] = mathCeil
	builtinTable["MathFloor"] = mathFloor
	builtinTable["MathCos"] = mathCos
	builtinTable["MathSin"] = mathSin
	builtinTable["MathTan"] = mathTan
	builtinTable["MathExp"] = mathExp
	builtinTable["MathMod"] = mathMod
	builtinTable["MathRand"] = mathRand
	builtinTable["MathSrand"] = mathSrand
	builtinTable["MathArccos"] = mathArccos
	builtinTable["MathArcsin"] = mathArcsin
	builtinTable["MathArctan"] = mathArctan
	builtinTable["MathLog10"] = mathLog10
	builtinTable["MathIsValidNumber"] = mathIsValidNumber

	// MQL4 lowercase aliases
	builtinTable["ceil"] = mathCeil
	builtinTable["floor"] = mathFloor
	builtinTable["cos"] = mathCos
	builtinTable["sin"] = mathSin
	builtinTable["tan"] = mathTan
	builtinTable["exp"] = mathExp
	builtinTable["fabs"] = mathAbs // alias for MathAbs
	builtinTable["fmax"] = mathMax // alias for MathMax
	builtinTable["fmin"] = mathMin // alias for MathMin
	builtinTable["fmod"] = mathMod
	builtinTable["log"] = mathLog // alias for MathLog
	builtinTable["log10"] = mathLog10
	builtinTable["pow"] = mathPow   // alias for MathPow
	builtinTable["round"] = mathRound // alias for MathRound
	builtinTable["rand"] = mathRand
	builtinTable["srand"] = mathSrand
	builtinTable["sqrt"] = mathSqrt // alias for MathSqrt
}

func mathCeil(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return DecimalVal(decimal.Zero), nil
	}
	f, _ := args[0].ToDecimal().Float64()
	return DecimalVal(decimal.NewFromFloat(math.Ceil(f))), nil
}

func mathFloor(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return DecimalVal(decimal.Zero), nil
	}
	f, _ := args[0].ToDecimal().Float64()
	return DecimalVal(decimal.NewFromFloat(math.Floor(f))), nil
}

func mathCos(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return DecimalVal(decimal.Zero), nil
	}
	f, _ := args[0].ToDecimal().Float64()
	return DecimalVal(decimal.NewFromFloat(math.Cos(f))), nil
}

func mathSin(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return DecimalVal(decimal.Zero), nil
	}
	f, _ := args[0].ToDecimal().Float64()
	return DecimalVal(decimal.NewFromFloat(math.Sin(f))), nil
}

func mathTan(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return DecimalVal(decimal.Zero), nil
	}
	f, _ := args[0].ToDecimal().Float64()
	return DecimalVal(decimal.NewFromFloat(math.Tan(f))), nil
}

func mathExp(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return DecimalVal(decimal.Zero), nil
	}
	f, _ := args[0].ToDecimal().Float64()
	return DecimalVal(decimal.NewFromFloat(math.Exp(f))), nil
}

func mathMod(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return DecimalVal(decimal.Zero), nil
	}
	a, _ := args[0].ToDecimal().Float64()
	b, _ := args[1].ToDecimal().Float64()
	if b == 0 {
		return DecimalVal(decimal.Zero), nil
	}
	return DecimalVal(decimal.NewFromFloat(math.Mod(a, b))), nil
}

func mathRand(it *Interpreter, args []Value) (Value, error) {
	return IntVal(int32(rand.Intn(32767))), nil
}

func mathSrand(it *Interpreter, args []Value) (Value, error) {
	if len(args) >= 1 {
		rand.Seed(int64(args[0].ToInt()))
	}
	return NoneVal(), nil
}

func mathArccos(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return DecimalVal(decimal.Zero), nil
	}
	f, _ := args[0].ToDecimal().Float64()
	return DecimalVal(decimal.NewFromFloat(math.Acos(f))), nil
}

func mathArcsin(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return DecimalVal(decimal.Zero), nil
	}
	f, _ := args[0].ToDecimal().Float64()
	return DecimalVal(decimal.NewFromFloat(math.Asin(f))), nil
}

func mathArctan(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return DecimalVal(decimal.Zero), nil
	}
	f, _ := args[0].ToDecimal().Float64()
	return DecimalVal(decimal.NewFromFloat(math.Atan(f))), nil
}

func mathLog10(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return DecimalVal(decimal.Zero), nil
	}
	f, _ := args[0].ToDecimal().Float64()
	if f <= 0 {
		return DecimalVal(decimal.Zero), nil
	}
	return DecimalVal(decimal.NewFromFloat(math.Log10(f))), nil
}

func mathIsValidNumber(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return BoolVal(false), nil
	}
	f, _ := args[0].ToDecimal().Float64()
	return BoolVal(!math.IsNaN(f) && !math.IsInf(f, 0)), nil
}
