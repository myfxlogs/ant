package interp

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Layer 3+4 builtin functions: array, string, datetime, conversion, platform.

func init() {
	// Register Layer 3+4 functions into the builtin table
	builtinTable["ArrayResize"] = arrayResize
	builtinTable["ArraySize"] = arraySize
	builtinTable["ArrayCopy"] = arrayCopy
	builtinTable["ArraySetAsSeries"] = arraySetAsSeries
	builtinTable["ArrayMaximum"] = arrayMaximum
	builtinTable["ArrayMinimum"] = arrayMinimum
	builtinTable["ArraySort"] = arraySort
	builtinTable["ArrayInitialize"] = arrayInitialize

	builtinTable["StringConcatenate"] = stringConcatenate
	builtinTable["StringFind"] = stringFind
	builtinTable["StringSubstr"] = stringSubstr
	builtinTable["StringLen"] = stringLen
	builtinTable["StringReplace"] = stringReplace
	builtinTable["StringSplit"] = stringSplit
	builtinTable["StringTrimLeft"] = stringTrimLeft
	builtinTable["StringTrimRight"] = stringTrimRight

	builtinTable["DoubleToString"] = doubleToString
	builtinTable["IntegerToString"] = integerToString
	builtinTable["StringToDouble"] = stringToDouble
	builtinTable["StringToInteger"] = stringToInteger
	builtinTable["NormalizeDouble"] = normalizeDouble

	builtinTable["TimeToString"] = timeToString
	builtinTable["TimeCurrent"] = timeCurrent

	builtinTable["Comment"] = builtinPrint // alias for Print
	builtinTable["Sleep"] = builtinSleep

	// MQL4 aliases
	builtinTable["DoubleToStr"] = doubleToString // alias for DoubleToString

	// Platform no-op / stub functions
	builtinTable["EventKillTimer"]            = noopReturn0
	builtinTable["EventSetMillisecondTimer"]  = noopReturn0
	builtinTable["EventSetTimer"]             = noopReturn0
	builtinTable["ExpertRemove"]              = noopReturn0
	builtinTable["GetLastError"]              = noopReturn0
	builtinTable["IsTesting"]                 = isTesting
	builtinTable["IsOptimization"]            = noopReturn0
	builtinTable["IsVisualMode"]              = noopReturn0
	builtinTable["RefreshRates"]              = noopReturnTrue
	builtinTable["Day"]                       = dayFunc
	builtinTable["DayOfWeek"]                 = dayOfWeekFunc
	builtinTable["Hour"]                      = hourFunc
	builtinTable["Minute"]                    = minuteFunc
	builtinTable["Year"]                      = yearFunc
	builtinTable["Month"]                     = monthFunc
	builtinTable["TimeLocal"]                 = timeCurrent
	builtinTable["StrToTime"]                 = strToTime
	builtinTable["TimeToStr"]                 = timeToStr
}

// ── Array functions ─────────────────────────────────────────────────

func arrayResize(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return IntVal(-1), fmt.Errorf("ArrayResize: needs 2 args")
	}
	name := args[0].ToString()
	newSize := int(args[1].ToInt())
	arr, _ := it.getArray(name)
	if newSize < 0 {
		newSize = 0
	}
	if len(arr) < newSize {
		arr = append(arr, make([]Value, newSize-len(arr))...)
	} else {
		arr = arr[:newSize]
	}
	it.setVar(name, Value{Kind: ValArray, Array: arr})
	return IntVal(int32(newSize)), nil
}

func arraySize(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return IntVal(0), fmt.Errorf("ArraySize: needs 1 arg")
	}
	if args[0].Kind == ValArray {
		return IntVal(int32(len(args[0].Array))), nil
	}
	name := args[0].ToString()
	arr, ok := it.getArray(name)
	if !ok {
		return IntVal(0), nil
	}
	return IntVal(int32(len(arr))), nil
}

func arrayCopy(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return IntVal(-1), fmt.Errorf("ArrayCopy: needs 2 args")
	}
	dstName := args[0].ToString()
	srcName := args[1].ToString()
	src, ok := it.getArray(srcName)
	if !ok {
		return IntVal(-1), nil
	}
	dst := make([]Value, len(src))
	copy(dst, src)
	it.setVar(dstName, Value{Kind: ValArray, Array: dst})
	return IntVal(int32(len(dst))), nil
}

func arraySetAsSeries(it *Interpreter, args []Value) (Value, error) {
	// No-op in interpreter (series semantics handled by SeriesAccessor)
	return BoolVal(true), nil
}

func arrayMaximum(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return IntVal(-1), fmt.Errorf("ArrayMaximum: needs 1 arg")
	}
	arr, ok := it.getArray(args[0].ToString())
	if !ok || len(arr) == 0 {
		return IntVal(-1), nil
	}
	maxIdx := 0
	maxVal := arr[0].ToDecimal()
	for i := 1; i < len(arr); i++ {
		if arr[i].ToDecimal().GreaterThan(maxVal) {
			maxVal = arr[i].ToDecimal()
			maxIdx = i
		}
	}
	return IntVal(int32(maxIdx)), nil
}

func arrayMinimum(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return IntVal(-1), fmt.Errorf("ArrayMinimum: needs 1 arg")
	}
	arr, ok := it.getArray(args[0].ToString())
	if !ok || len(arr) == 0 {
		return IntVal(-1), nil
	}
	minIdx := 0
	minVal := arr[0].ToDecimal()
	for i := 1; i < len(arr); i++ {
		if arr[i].ToDecimal().LessThan(minVal) {
			minVal = arr[i].ToDecimal()
			minIdx = i
		}
	}
	return IntVal(int32(minIdx)), nil
}

func arraySort(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return IntVal(-1), fmt.Errorf("ArraySort: needs 1 arg")
	}
	name := args[0].ToString()
	arr, ok := it.getArray(name)
	if !ok {
		return IntVal(-1), nil
	}
	// Simple insertion sort (arrays are typically small)
	for i := 1; i < len(arr); i++ {
		for j := i; j > 0 && arr[j].ToDecimal().LessThan(arr[j-1].ToDecimal()); j-- {
			arr[j], arr[j-1] = arr[j-1], arr[j]
		}
	}
	it.setVar(name, Value{Kind: ValArray, Array: arr})
	return IntVal(int32(len(arr))), nil
}

func arrayInitialize(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return IntVal(-1), fmt.Errorf("ArrayInitialize: needs 2 args")
	}
	name := args[0].ToString()
	val := args[1]
	arr, ok := it.getArray(name)
	if !ok {
		return IntVal(-1), nil
	}
	for i := range arr {
		arr[i] = val
	}
	it.setVar(name, Value{Kind: ValArray, Array: arr})
	return IntVal(int32(len(arr))), nil
}

// ── String functions ────────────────────────────────────────────────

func stringConcatenate(it *Interpreter, args []Value) (Value, error) {
	var sb strings.Builder
	for _, a := range args {
		sb.WriteString(a.ToString())
	}
	return StringVal(sb.String()), nil
}

func stringFind(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return IntVal(-1), fmt.Errorf("StringFind: needs 2 args")
	}
	s := args[0].ToString()
	sub := args[1].ToString()
	return IntVal(int32(strings.Index(s, sub))), nil
}

func stringSubstr(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return StringVal(""), fmt.Errorf("StringSubstr: needs 2 args")
	}
	s := args[0].ToString()
	start := int(args[1].ToInt())
	if start < 0 || start >= len(s) {
		return StringVal(""), nil
	}
	length := len(s) - start
	if len(args) >= 3 {
		length = int(args[2].ToInt())
	}
	end := start + length
	if end > len(s) {
		end = len(s)
	}
	return StringVal(s[start:end]), nil
}

func stringLen(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return IntVal(0), fmt.Errorf("StringLen: needs 1 arg")
	}
	return IntVal(int32(len(args[0].ToString()))), nil
}

func stringReplace(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 3 {
		return IntVal(0), fmt.Errorf("StringReplace: needs 3 args")
	}
	s := args[0].ToString()
	old := args[1].ToString()
	newS := args[2].ToString()
	return StringVal(strings.ReplaceAll(s, old, newS)), nil
}

func stringSplit(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return IntVal(0), fmt.Errorf("StringSplit: needs 2 args")
	}
	sep := args[0].ToString()
	s := args[1].ToString()
	parts := strings.Split(s, sep)
	arr := make([]Value, len(parts))
	for i, p := range parts {
		arr[i] = StringVal(p)
	}
	return Value{Kind: ValArray, Array: arr}, nil
}

func stringTrimLeft(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), fmt.Errorf("StringTrimLeft: needs 1 arg")
	}
	return StringVal(strings.TrimLeft(args[0].ToString(), " \t\n\r")), nil
}

func stringTrimRight(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), fmt.Errorf("StringTrimRight: needs 1 arg")
	}
	return StringVal(strings.TrimRight(args[0].ToString(), " \t\n\r")), nil
}

// ── Conversion functions ────────────────────────────────────────────

func doubleToString(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), fmt.Errorf("DoubleToString: needs 1 arg")
	}
	d := args[0].ToDecimal()
	if len(args) >= 2 {
		digits := int(args[1].ToInt())
		return StringVal(d.StringFixed(int32(digits))), nil
	}
	return StringVal(d.String()), nil
}

func integerToString(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), fmt.Errorf("IntegerToString: needs 1 arg")
	}
	return StringVal(fmt.Sprintf("%d", args[0].ToInt())), nil
}

func stringToDouble(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return DecimalVal(decimal.Zero), fmt.Errorf("StringToDouble: needs 1 arg")
	}
	d, err := decimal.NewFromString(args[0].ToString())
	if err != nil {
		return DecimalVal(decimal.Zero), nil
	}
	return DecimalVal(d), nil
}

func stringToInteger(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return IntVal(0), fmt.Errorf("StringToInteger: needs 1 arg")
	}
	var n int32
	_, err := fmt.Sscanf(args[0].ToString(), "%d", &n)
	if err != nil {
		return IntVal(0), nil
	}
	return IntVal(n), nil
}

func normalizeDouble(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return DecimalVal(decimal.Zero), fmt.Errorf("NormalizeDouble: needs 2 args")
	}
	d := args[0].ToDecimal()
	digits := int32(args[1].ToInt())
	return DecimalVal(d.Round(digits)), nil
}

// ── Datetime functions ──────────────────────────────────────────────

func timeToString(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), fmt.Errorf("TimeToString: needs 1 arg")
	}
	ts := args[0].ToInt()
	// MQL format: "yyyy.mm.dd hh:mi"
	// Simplified: return unix seconds as string
	return StringVal(fmt.Sprintf("%d", ts)), nil
}

func timeCurrent(it *Interpreter, args []Value) (Value, error) {
	if it.ctx != nil {
		return Value{Kind: ValDatetime, Datetime: it.ctx.ServerTime()}, nil
	}
	return Value{Kind: ValDatetime, Datetime: 0}, nil
}

// ── Platform functions ──────────────────────────────────────────────

func builtinSleep(it *Interpreter, args []Value) (Value, error) {
	// No-op in interpreter (can't sleep in backtest)
	return NoneVal(), nil
}

func noopReturn0(it *Interpreter, args []Value) (Value, error) {
	return IntVal(0), nil
}

func noopReturnTrue(it *Interpreter, args []Value) (Value, error) {
	return BoolVal(true), nil
}

func isTesting(it *Interpreter, args []Value) (Value, error) {
	// In backtest context, IsTesting() returns true.
	// In live context, it returns false.
	// We detect by checking if ctx has ServerTime > 0 (backtest sets it).
	if it.ctx != nil && it.ctx.ServerTime() > 0 {
		return BoolVal(true), nil
	}
	return BoolVal(false), nil
}

func dayFunc(it *Interpreter, args []Value) (Value, error) {
	if it.ctx != nil {
		ts := it.ctx.ServerTime()
		if ts > 0 {
			return IntVal(int32(time.UnixMilli(ts).UTC().Day())), nil
		}
	}
	return IntVal(1), nil
}

func dayOfWeekFunc(it *Interpreter, args []Value) (Value, error) {
	if it.ctx != nil {
		ts := it.ctx.ServerTime()
		if ts > 0 {
			return IntVal(int32(time.UnixMilli(ts).UTC().Weekday())), nil
		}
	}
	return IntVal(1), nil
}

func hourFunc(it *Interpreter, args []Value) (Value, error) {
	if it.ctx != nil {
		ts := it.ctx.ServerTime()
		if ts > 0 {
			return IntVal(int32(time.UnixMilli(ts).UTC().Hour())), nil
		}
	}
	return IntVal(0), nil
}

func minuteFunc(it *Interpreter, args []Value) (Value, error) {
	if it.ctx != nil {
		ts := it.ctx.ServerTime()
		if ts > 0 {
			return IntVal(int32(time.UnixMilli(ts).UTC().Minute())), nil
		}
	}
	return IntVal(0), nil
}

func yearFunc(it *Interpreter, args []Value) (Value, error) {
	if it.ctx != nil {
		ts := it.ctx.ServerTime()
		if ts > 0 {
			return IntVal(int32(time.UnixMilli(ts).UTC().Year())), nil
		}
	}
	return IntVal(2024), nil
}

func monthFunc(it *Interpreter, args []Value) (Value, error) {
	if it.ctx != nil {
		ts := it.ctx.ServerTime()
		if ts > 0 {
			return IntVal(int32(time.UnixMilli(ts).UTC().Month())), nil
		}
	}
	return IntVal(1), nil
}

func strToTime(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return Value{Kind: ValDatetime, Datetime: 0}, fmt.Errorf("StrToTime: needs 1 arg")
	}
	s := args[0].ToString()
	// MQL format: "yyyy.mm.dd hh:mi"
	t, err := time.Parse("2006.01.02 15:04", s)
	if err != nil {
		t, err = time.Parse("2006.01.02", s)
		if err != nil {
			return Value{Kind: ValDatetime, Datetime: 0}, nil
		}
	}
	return Value{Kind: ValDatetime, Datetime: t.Unix()}, nil
}

func timeToStr(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), fmt.Errorf("TimeToStr: needs 1 arg")
	}
	ts := int64(args[0].ToInt())
	// MQL format: "yyyy.mm.dd hh:mi"
	t := time.Unix(ts, 0).UTC()
	mode := int32(0)
	if len(args) >= 2 {
		mode = args[1].ToInt()
	}
	switch mode {
	case 1: // TIME_DATE
		return StringVal(t.Format("2006.01.02")), nil
	case 2: // TIME_MINUTES
		return StringVal(t.Format("15:04")), nil
	case 4: // TIME_SECONDS
		return StringVal(t.Format("15:04:05")), nil
	default: // TIME_DATE|TIME_MINUTES
		return StringVal(t.Format("2006.01.02 15:04")), nil
	}
}
