package interp

import (
	"fmt"
	"strings"
	"time"
)

// Stub dispatch registrations for MQL5/MQL4 platform functions that are
// registered in implementedPlatform (so the analyzer doesn't report them as
// blind spots) but don't have real implementations yet. These stubs return
// safe default values so the interpreter doesn't panic.

func init() {
	// ── String functions ──────────────────────────────────────────────
	builtinTable["StringAdd"] = stringAdd
	builtinTable["StringCompare"] = stringCompare
	builtinTable["StringFormat"] = stringFormat
	builtinTable["StringGetCharacter"] = stringGetCharacter
	builtinTable["StringSetCharacter"] = stringSetCharacter
	builtinTable["StringToLower"] = stringToLower
	builtinTable["StringToUpper"] = stringToUpper
	builtinTable["StringBufferLen"] = noopReturn0
	builtinTable["StringInit"] = stringInit
	builtinTable["StringFill"] = stringFill

	// ── Array functions ───────────────────────────────────────────────
	builtinTable["ArrayBsearch"] = noopReturn0
	builtinTable["ArrayCompare"] = noopReturn0
	builtinTable["ArrayFill"] = noopReturn0
	builtinTable["ArrayFree"] = noopReturn0
	builtinTable["ArrayGetAsSeries"] = noopReturnTrue
	builtinTable["ArrayIsDynamic"] = noopReturnTrue
	builtinTable["ArrayIsSeries"] = noopReturnFalse
	builtinTable["ArrayRange"] = noopReturn0
	builtinTable["ArrayPrint"] = noopReturn0
	builtinTable["ArrayInsert"] = noopReturn0
	builtinTable["ArrayRemove"] = noopReturn0
	builtinTable["ArrayReverse"] = noopReturn0
	builtinTable["ArraySwap"] = noopReturnTrue

	// ── Conversion functions ──────────────────────────────────────────
	builtinTable["CharToString"] = charToString
	builtinTable["CharArrayToString"] = charArrayToString
	builtinTable["ShortToString"] = shortToString
	builtinTable["ShortArrayToString"] = shortArrayToString
	builtinTable["StringToColor"] = noopReturn0
	builtinTable["StringToCharArray"] = noopReturn0
	builtinTable["StringToShortArray"] = noopReturn0
	builtinTable["EnumToString"] = enumToString

	// ── Date/Time functions ───────────────────────────────────────────
	builtinTable["TimeGMT"] = timeGMT
	builtinTable["TimeGMTOffset"] = timeGMTOffset
	builtinTable["TimeDaylightSavings"] = noopReturn0
	builtinTable["TimeTradeServer"] = timeCurrent // alias
	builtinTable["TimeToStruct"] = noopReturn0
	builtinTable["StructToTime"] = noopReturn0

	// ── Checkup functions ─────────────────────────────────────────────
	builtinTable["PeriodSeconds"] = noopReturn0
	builtinTable["UninitializeReason"] = noopReturn0
	builtinTable["IsStopped"] = noopReturnFalse
	builtinTable["MQLInfoInteger"] = noopReturn0
	builtinTable["MQLInfoString"] = noopReturnEmpty
	builtinTable["TerminalInfoDouble"] = noopReturn0
	builtinTable["TerminalInfoInteger"] = noopReturn0
	builtinTable["TerminalInfoString"] = noopReturnEmpty

	// ── Common functions ──────────────────────────────────────────────
	builtinTable["GetTickCount"] = getTickCount
	builtinTable["GetTickCount64"] = getTickCount
	builtinTable["GetMicrosecondCount"] = getMicrosecondCount
	builtinTable["SetUserError"] = noopReturn0
	builtinTable["SetReturnError"] = noopReturn0

	// ── MQL4 deprecated aliases ───────────────────────────────────────
	builtinTable["CurTime"] = timeCurrent // alias for TimeCurrent
}

func noopReturnFalse(it *Interpreter, args []Value) (Value, error) {
	return BoolVal(false), nil
}

func noopReturnEmpty(it *Interpreter, args []Value) (Value, error) {
	return StringVal(""), nil
}

// ── String stub implementations ─────────────────────────────────────

func stringAdd(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return StringVal(""), nil
	}
	return StringVal(args[0].ToString() + args[1].ToString()), nil
}

func stringCompare(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return IntVal(0), nil
	}
	a, b := args[0].ToString(), args[1].ToString()
	return IntVal(int32(strings.Compare(a, b))), nil
}

func stringFormat(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), nil
	}
	format := args[0].ToString()
	rest := make([]any, len(args)-1)
	for i := 1; i < len(args); i++ {
		rest[i-1] = args[i].ToString()
	}
	return StringVal(fmt.Sprintf(format, rest...)), nil
}

func stringGetCharacter(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return IntVal(0), nil
	}
	s := args[0].ToString()
	pos := int(args[1].ToInt())
	if pos < 0 || pos >= len(s) {
		return IntVal(0), nil
	}
	return IntVal(int32(s[pos])), nil
}

func stringSetCharacter(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 3 {
		return StringVal(""), nil
	}
	s := args[0].ToString()
	pos := int(args[1].ToInt())
	ch := rune(args[2].ToInt())
	if pos < 0 || pos >= len(s) {
		return StringVal(s), nil
	}
	runes := []rune(s)
	if pos < len(runes) {
		runes[pos] = ch
	}
	return StringVal(string(runes)), nil
}

func stringToLower(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), nil
	}
	return StringVal(strings.ToLower(args[0].ToString())), nil
}

func stringToUpper(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), nil
	}
	return StringVal(strings.ToUpper(args[0].ToString())), nil
}

func stringInit(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return StringVal(""), nil
	}
	size := int(args[0].ToInt())
	ch := byte(args[1].ToInt())
	return StringVal(strings.Repeat(string(ch), size)), nil
}

func stringFill(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 2 {
		return StringVal(""), nil
	}
	s := args[0].ToString()
	ch := byte(args[1].ToInt())
	if len(s) == 0 {
		return StringVal(s), nil
	}
	b := []byte(s)
	for i := range b {
		b[i] = ch
	}
	return StringVal(string(b)), nil
}

// ── Conversion stubs ────────────────────────────────────────────────

func charToString(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), nil
	}
	return StringVal(string(rune(args[0].ToInt()))), nil
}

func charArrayToString(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), nil
	}
	if args[0].Kind == ValArray {
		var sb strings.Builder
		for _, v := range args[0].Array {
			sb.WriteRune(rune(v.ToInt()))
		}
		return StringVal(sb.String()), nil
	}
	return StringVal(args[0].ToString()), nil
}

func shortToString(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), nil
	}
	return StringVal(string(rune(args[0].ToInt()))), nil
}

func shortArrayToString(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), nil
	}
	if args[0].Kind == ValArray {
		var sb strings.Builder
		for _, v := range args[0].Array {
			sb.WriteRune(rune(v.ToInt()))
		}
		return StringVal(sb.String()), nil
	}
	return StringVal(args[0].ToString()), nil
}

func enumToString(it *Interpreter, args []Value) (Value, error) {
	if len(args) < 1 {
		return StringVal(""), nil
	}
	return StringVal(args[0].ToString()), nil
}

// ── DateTime stubs ──────────────────────────────────────────────────

func timeGMT(it *Interpreter, args []Value) (Value, error) {
	return Value{Kind: ValDatetime, Datetime: time.Now().UTC().UnixMilli()}, nil
}

func timeGMTOffset(it *Interpreter, args []Value) (Value, error) {
	_, offset := time.Now().Zone()
	return IntVal(int32(offset)), nil
}

// ── Common stubs ────────────────────────────────────────────────────

var startTime = time.Now()

func getTickCount(it *Interpreter, args []Value) (Value, error) {
	return IntVal(int32(time.Since(startTime).Milliseconds())), nil
}

func getMicrosecondCount(it *Interpreter, args []Value) (Value, error) {
	return IntVal(int32(time.Since(startTime).Microseconds())), nil
}
