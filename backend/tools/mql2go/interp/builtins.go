package interp

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// callBuiltin dispatches a builtin function call.
// Unknown functions log an error and return NoneVal — never silently skip.

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
	// Check user-defined functions first
	if it.ir != nil && it.ir.Funcs != nil {
		if fn, ok := it.ir.Funcs[name]; ok {
			return it.callUserFunc(fn, args)
		}
	}

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
		// Unimplemented — log error and return NoneVal
		it.lastErr = 1
		it.errSet = true
		if it.ctx != nil {
			it.ctx.Log(fmt.Sprintf("MQL interpreter: unimplemented function %q", name))
		}
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

// mathSqrt computes square root.
// Known limitation: uses float64 internally (decimal→float64→decimal round-trip).
// Acceptable for math tool functions, but if an EA uses MathSqrt for position
// sizing,微小精度误差 may occur. Price calculations elsewhere use pure decimal.
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

// mathPow computes base^exp.
// Known limitation: uses float64 internally (decimal→float64→decimal round-trip).
// Same precision caveat as mathSqrt — acceptable for math tool functions.
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

// ── Market data, indicators, trade dispatch ───────────────────────

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
		return IntVal(timeframeToPeriod(it.ctx.Timeframe())), true
	}
	return NoneVal(), false
}

// timeframeToPeriod maps SDK timeframe strings to MQL period int values.
func timeframeToPeriod(tf string) int32 {
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
