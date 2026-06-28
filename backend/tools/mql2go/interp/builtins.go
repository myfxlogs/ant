package interp

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// callBuiltin dispatches a builtin function call.
// Phase 2 will populate the full builtin table (Layer 1-4).
// Unimplemented functions return an error — never silently skip.

var builtinTable = map[string]func(*Interpreter, []Value) (Value, error){
	// Layer 3 — Math functions (decimal.Decimal, no float64)
	"MathAbs":  mathAbs,
	"MathMax":  mathMax,
	"MathMin":  mathMin,
	"MathSqrt": mathSqrt,
	"MathPow":  mathPow,

	// Layer 4 — Platform functions
	"Print":  builtinPrint,
	"Alert":  builtinPrint, // alias
}

func (it *Interpreter) callBuiltin(name string, args []Expr) Value {
	fn, ok := builtinTable[name]
	if !ok {
		// Try market data / indicator functions
		if v, ok := it.callMarketData(name, args); ok {
			return v
		}
		if v, ok := it.callIndicator(name, args); ok {
			return v
		}
		if v, ok := it.callTrade(name, args); ok {
			return v
		}
		// Unimplemented — return error value
		it.lastErr = 1
		it.errSet = true
		return NoneVal()
	}

	// Evaluate arguments
	vals := make([]Value, len(args))
	for i := range args {
		vals[i] = it.evalExpr(&args[i])
	}

	result, err := fn(it, vals)
	if err != nil {
		it.lastErr = 1
		it.errSet = true
		return NoneVal()
	}
	return result
}

// dispatchClassMethod handles MQL5 class method calls (CTrade.Buy, etc.).
// Phase 3 will populate the full CTrade method table.
func (it *Interpreter) dispatchClassMethod(cls *ClassInstance, method string, args []Expr) Value {
	if cls == nil {
		return NoneVal()
	}
	switch cls.Name {
	case "CTrade":
		return it.callCTrade(cls, method, args)
	}
	return NoneVal()
}

// ── Layer 3: Math functions (decimal.Decimal) ──────────────────────

func mathAbs(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return NoneVal(), fmt.Errorf("MathAbs: not enough args")
	}
	return DecimalVal(args[0].ToDecimal().Abs()), nil
}

func mathMax(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return NoneVal(), fmt.Errorf("MathMax: not enough args")
	}
	if args[0].ToDecimal().GreaterThan(args[1].ToDecimal()) {
		return args[0], nil
	}
	return args[1], nil
}

func mathMin(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return NoneVal(), fmt.Errorf("MathMin: not enough args")
	}
	if args[0].ToDecimal().LessThan(args[1].ToDecimal()) {
		return args[0], nil
	}
	return args[1], nil
}

func mathSqrt(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return NoneVal(), fmt.Errorf("MathSqrt: not enough args")
	}
	d := args[0].ToDecimal()
	if d.LessThan(decimal.Zero) {
		return DecimalVal(decimal.Zero), nil
	}
	f, _ := d.Float64()
	return DecimalVal(decimal.NewFromFloat(sqrt(f))), nil
}

func mathPow(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return NoneVal(), fmt.Errorf("MathPow: not enough args")
	}
	base, _ := args[0].ToDecimal().Float64()
	exp, _ := args[1].ToDecimal().Float64()
	return DecimalVal(decimal.NewFromFloat(pow(base, exp))), nil
}

// ── Layer 4: Platform functions ─────────────────────────────────────

func builtinPrint(it *Interpreter, args []Value) (Value, error) {
	if it.ctx != nil {
		var parts []string
		for _, a := range args {
			parts = append(parts, a.ToString())
		}
		it.ctx.Log(joinStrings(parts, " "))
	}
	return NoneVal(), nil
}

// ── Stubs for Phase 2 (market data, indicators, trade) ──────────────
// These will be fully implemented in Phase 2.

func (it *Interpreter) callMarketData(name string, args []Expr) (Value, bool) {
	if it.ctx == nil {
		return NoneVal(), false
	}
	switch name {
	case "Ask", "ask":
		return DecimalVal(it.ctx.Ask()), true
	case "Bid", "bid":
		return DecimalVal(it.ctx.Bid()), true
	case "Point", "point", "_Point":
		return DecimalVal(it.ctx.Point()), true
	case "Symbol", "symbol", "_Symbol":
		return StringVal(it.ctx.Symbol()), true
	case "Digits", "digits":
		return IntVal(it.ctx.Digits()), true
	case "Period":
		return IntVal(0), true // TODO: map timeframe string to int
	}
	return NoneVal(), false
}

// callIndicator is implemented in builtin_indicators.go.
// callTrade is implemented in builtin_trade.go.

func (it *Interpreter) callCTrade(cls *ClassInstance, method string, args []Expr) Value {
	if cls == nil {
		return NoneVal()
	}
	switch cls.Name {
	case "CTrade":
		return it.execCTrade(cls, method, args)
	}
	return NoneVal()
}

// ── Math helpers ────────────────────────────────────────────────────

func sqrt(f float64) float64 {
	if f <= 0 {
		return 0
	}
	x := f
	for i := 0; i < 20; i++ {
		x = (x + f/x) / 2
	}
	return x
}

func pow(base, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	if exp == 1 {
		return base
	}
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}
