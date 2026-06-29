package interp

import (
	"fmt"
	"math"
	"strings"

	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
)

// callBuiltin dispatches a builtin function call.
// Unknown functions log an error and return NoneVal — never silently skip.

var builtinTable = map[string]func(*Interpreter, []Value) (Value, error){
	// Layer 3 — Math functions (decimal.Decimal, no float64)
	"MathAbs":   mathAbs,
	"MathMax":   mathMax,
	"MathMin":   mathMin,
	"MathSqrt":  mathSqrt,
	"MathPow":   mathPow,
	"MathLog":   mathLog,
	"MathRound": mathRound,

	// Layer 4 — Platform functions
	"Print":  builtinPrint,
	"Alert":  builtinPrint, // alias
}

func (it *Interpreter) callBuiltin(name string, args []Expr) (valOut Value) {
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
		if v, ok := it.callIndicatorOnArray(name, args); ok {
			return v
		}
		if v, ok := it.callTrade(name, args); ok {
			return v
		}
		// Unimplemented — record runtime blind spot
		it.recordBlindSpot(name)
		return NoneVal()
	}

	// Evaluate arguments
	vals := make([]Value, len(args))
	for i := range args {
		vals[i] = it.evalExpr(&args[i])
	}

	result, err := fn(it, vals)
	if err != nil {
		it.recordBlindSpot(name)
		return NoneVal()
	}
	return result
}

// recordBlindSpot logs an unimplemented function call at runtime.
// Fatal blind spots abort the current execution via panic(errFatalBlindSpot),
// which is recovered in execBlock to return an error to the caller.
func (it *Interpreter) recordBlindSpot(name string) {
	if it.runtimeBlindSpots == nil {
		it.runtimeBlindSpots = make(map[string]int)
	}
	it.runtimeBlindSpots[name]++
	if it.ctx != nil {
		it.ctx.Log(fmt.Sprintf("MQL interpreter: unimplemented function %q (hit #%d)", name, it.runtimeBlindSpots[name]))
	}
	// Fatal blind spots abort execution — returning NoneVal for trade/indicator
	// functions would silently corrupt EA logic (e.g. OrderSend returns 0 → no order)
	if classifyRuntimeSeverity(name) == "致命" {
		panic(errFatalBlindSpot)
	}
}

// dispatchClassMethod handles MQL5 class method calls (CTrade.Buy, etc.).
func (it *Interpreter) dispatchClassMethod(cls *ClassInstance, method string, args []Expr) Value {
	if cls == nil {
		return NoneVal()
	}
	switch cls.Name {
	case "CTrade":
		return it.execCTrade(cls, method, args)
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
	return DecimalVal(decimal.NewFromFloat(math.Sqrt(f))), nil
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
	return DecimalVal(decimal.NewFromFloat(math.Pow(base, exp))), nil
}

func mathLog(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return NoneVal(), fmt.Errorf("MathLog: not enough args")
	}
	f, _ := args[0].ToDecimal().Float64()
	if f <= 0 {
		return DecimalVal(decimal.Zero), nil
	}
	return DecimalVal(decimal.NewFromFloat(math.Log(f))), nil
}

func mathRound(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return NoneVal(), fmt.Errorf("MathRound: not enough args")
	}
	f, _ := args[0].ToDecimal().Float64()
	return DecimalVal(decimal.NewFromFloat(math.Round(f))), nil
}

// ── Layer 4: Platform functions ─────────────────────────────────────

func builtinPrint(it *Interpreter, args []Value) (Value, error) {
	if it.ctx != nil {
		var parts []string
		for _, a := range args {
			parts = append(parts, a.ToString())
		}
		it.ctx.Log(strings.Join(parts, " "))
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

	// MQL5 SymbolInfoDouble(symbol, property) → double
	case "SymbolInfoDouble":
		if len(args) >= 2 && it.ctx.Broker() != nil {
			sym := it.evalExpr(&args[0]).ToString()
			prop := it.evalExpr(&args[1]).ToInt()
			info, err := it.ctx.Broker().SymbolInfo(sym)
			if err != nil {
				return DecimalVal(decimal.Zero), true
			}
			return DecimalVal(symbolInfoDouble(info, prop)), true
		}
		return DecimalVal(decimal.Zero), true

	// MQL5 SymbolInfoInteger(symbol, property) → long
	case "SymbolInfoInteger":
		if len(args) >= 2 && it.ctx.Broker() != nil {
			sym := it.evalExpr(&args[0]).ToString()
			prop := it.evalExpr(&args[1]).ToInt()
			info, err := it.ctx.Broker().SymbolInfo(sym)
			if err != nil {
				return IntVal(0), true
			}
			return IntVal(symbolInfoInteger(info, prop)), true
		}
		return IntVal(0), true

	// MQL5 SymbolInfoString(symbol, property) → string
	case "SymbolInfoString":
		if len(args) >= 2 && it.ctx.Broker() != nil {
			sym := it.evalExpr(&args[0]).ToString()
			prop := it.evalExpr(&args[1]).ToInt()
			info, err := it.ctx.Broker().SymbolInfo(sym)
			if err != nil {
				return StringVal(""), true
			}
			return StringVal(symbolInfoString(info, prop)), true
		}
		return StringVal(""), true

	// MQL4 MarketInfo(symbol, mode) → double
	case "MarketInfo":
		if len(args) >= 2 && it.ctx.Broker() != nil {
			sym := it.evalExpr(&args[0]).ToString()
			mode := it.evalExpr(&args[1]).ToInt()
			info, err := it.ctx.Broker().SymbolInfo(sym)
			if err != nil {
				return DecimalVal(decimal.Zero), true
			}
			return DecimalVal(marketInfo(info, mode)), true
		}
		return DecimalVal(decimal.Zero), true

	// Cross-timeframe market data: iHigh/iLow/iOpen/iClose/iTime
	// MQL4/MQL5: iHigh(symbol, timeframe, shift) → double
	case "iHigh":
		if len(args) >= 3 {
			tf := periodToTimeframe(it.evalExpr(&args[1]).ToInt())
			shift := int(it.evalExpr(&args[2]).ToInt())
			return DecimalVal(it.ctx.BarsTF(tf).High(shift)), true
		}
		return DecimalVal(decimal.Zero), true
	case "iLow":
		if len(args) >= 3 {
			tf := periodToTimeframe(it.evalExpr(&args[1]).ToInt())
			shift := int(it.evalExpr(&args[2]).ToInt())
			return DecimalVal(it.ctx.BarsTF(tf).Low(shift)), true
		}
		return DecimalVal(decimal.Zero), true
	case "iOpen":
		if len(args) >= 3 {
			tf := periodToTimeframe(it.evalExpr(&args[1]).ToInt())
			shift := int(it.evalExpr(&args[2]).ToInt())
			return DecimalVal(it.ctx.BarsTF(tf).Open(shift)), true
		}
		return DecimalVal(decimal.Zero), true
	case "iClose":
		if len(args) >= 3 {
			tf := periodToTimeframe(it.evalExpr(&args[1]).ToInt())
			shift := int(it.evalExpr(&args[2]).ToInt())
			return DecimalVal(it.ctx.BarsTF(tf).Close(shift)), true
		}
		return DecimalVal(decimal.Zero), true
	case "iTime":
		if len(args) >= 3 {
			tf := periodToTimeframe(it.evalExpr(&args[1]).ToInt())
			shift := int(it.evalExpr(&args[2]).ToInt())
			return Value{Kind: ValDatetime, Datetime: it.ctx.BarsTF(tf).Time(shift)}, true
		}
		return Value{Kind: ValDatetime, Datetime: 0}, true
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

// periodToTimeframe maps MQL period int values to SDK timeframe strings.
func periodToTimeframe(period int32) string {
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
		return ""
	}
}

// callIndicator is implemented in builtin_indicators.go.
// callTrade is implemented in builtin_trade.go.

// ── SymbolInfo / MarketInfo property mappers ───────────────────────

// MQL5 SYMBOL_* double property constants.
const (
	symBid         = 1
	symAsk         = 2
	symLast        = 3
	symPoint       = 11
	symTickSize    = 15
	symTickValue   = 16
	symSwapLong    = 18
	symSwapShort   = 19
	symVolumeMin   = 23
	symVolumeMax   = 24
	symVolumeStep  = 25
	symTradeAccVal = 30
)

// MQL5 SYMBOL_* integer property constants.
const (
	symDigits     = 12
	symSpread     = 13
	symStopsLevel = 14
)

// MQL5 SYMBOL_* string property constants.
const (
	symCurrencyBase   = 1
	symCurrencyProfit = 2
	symCurrencyMargin = 3
)

// MQL4 MarketInfo mode constants.
const (
	miBid       = 1
	miAsk       = 2
	miPoint     = 3
	miDigits    = 4
	miSpread    = 5
	miStopLevel = 6
	miMinLot    = 17
	miMaxLot    = 18
	miLotStep   = 19
	miTickValue = 16
	miTickSize  = 20
)

func symbolInfoDouble(info sdk.SymbolInfo, prop int32) decimal.Decimal {
	switch prop {
	case symBid:
		return decimal.Zero // no live bid in backtest
	case symAsk:
		return decimal.Zero
	case symPoint:
		return info.Point
	case symTickSize:
		return info.TickSize
	case symTickValue:
		return info.TickValue
	case symSwapLong:
		return info.SwapLong
	case symSwapShort:
		return info.SwapShort
	case symVolumeMin:
		return info.VolumeMin
	case symVolumeMax:
		return info.VolumeMax
	case symVolumeStep:
		return info.VolumeStep
	}
	return decimal.Zero
}

func symbolInfoInteger(info sdk.SymbolInfo, prop int32) int32 {
	switch prop {
	case symDigits:
		return info.Digits
	case symSpread:
		return info.Spread
	case symStopsLevel:
		return info.StopsLevel
	}
	return 0
}

func symbolInfoString(info sdk.SymbolInfo, prop int32) string {
	switch prop {
	case symCurrencyBase, symCurrencyProfit, symCurrencyMargin:
		return info.Name
	}
	return ""
}

func marketInfo(info sdk.SymbolInfo, mode int32) decimal.Decimal {
	switch mode {
	case miBid, miAsk:
		return decimal.Zero
	case miPoint:
		return info.Point
	case miDigits:
		return decimal.NewFromInt(int64(info.Digits))
	case miSpread:
		return decimal.NewFromInt(int64(info.Spread))
	case miStopLevel:
		return decimal.NewFromInt(int64(info.StopsLevel))
	case miMinLot:
		return info.VolumeMin
	case miMaxLot:
		return info.VolumeMax
	case miLotStep:
		return info.VolumeStep
	case miTickValue:
		return info.TickValue
	case miTickSize:
		return info.TickSize
	}
	return decimal.Zero
}
