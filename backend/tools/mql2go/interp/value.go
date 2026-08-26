package interp

import (
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
)

// Value is the dynamic type used during interpretation.
// All numeric values use decimal.Decimal — no float64 for prices.
type Value struct {
	Kind     ValueKind
	Int      int32
	Decimal  decimal.Decimal
	Str      string
	Bool     bool
	Array    []Value
	Datetime int64          // unix timestamp (ms)
	Class    *ClassInstance // ValClass: MQL5 class/struct instance
}

// ValueKind enumerates value types.
type ValueKind uint8

const (
	ValNone     ValueKind = iota
	ValInt
	ValDecimal
	ValBool
	ValString
	ValDatetime
	ValArray
	ValClass
)

// ClassInstance represents an MQL5 class or struct instance (CTrade, MqlTradeRequest, user-defined struct).
type ClassInstance struct {
	Name   string
	Fields map[string]Value
}

// ── Value constructors ──────────────────────────────────────────────

func IntVal(n int32) Value {
	return Value{Kind: ValInt, Int: n}
}

func DecimalVal(d decimal.Decimal) Value {
	return Value{Kind: ValDecimal, Decimal: d}
}

func BoolVal(b bool) Value {
	return Value{Kind: ValBool, Bool: b}
}

func StringVal(s string) Value {
	return Value{Kind: ValString, Str: s}
}

func NoneVal() Value {
	return Value{Kind: ValNone}
}

func DatetimeVal(ms int64) Value {
	return Value{Kind: ValDatetime, Datetime: ms}
}

// ── Value helpers ───────────────────────────────────────────────────

// IsTrue returns the truthiness of a Value following MQL semantics.
func (v Value) IsTrue() bool {
	switch v.Kind {
	case ValBool:
		return v.Bool
	case ValInt:
		return v.Int != 0
	case ValDecimal:
		return !v.Decimal.IsZero()
	case ValString:
		return v.Str != ""
	case ValNone:
		return false
	default:
		return true
	}
}

// ToDecimal converts a Value to decimal.Decimal.
func (v Value) ToDecimal() decimal.Decimal {
	switch v.Kind {
	case ValInt:
		return decimal.NewFromInt(int64(v.Int))
	case ValDecimal:
		return v.Decimal
	case ValDatetime:
		return decimal.NewFromInt(v.Datetime)
	case ValBool:
		if v.Bool {
			return decimal.NewFromInt(1)
		}
		return decimal.Zero
	case ValNone:
		return decimal.Zero
	default:
		return decimal.Zero
	}
}

// ToInt converts a Value to int32.
func (v Value) ToInt() int32 {
	switch v.Kind {
	case ValInt:
		return v.Int
	case ValDecimal:
		d := v.Decimal
		i := d.IntPart()
		return int32(i)
	case ValDatetime:
		return int32(v.Datetime)
	case ValBool:
		if v.Bool {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// ToString converts a Value to string.
func (v Value) ToString() string {
	switch v.Kind {
	case ValString:
		return v.Str
	case ValInt:
		return strconv.FormatInt(int64(v.Int), 10)
	case ValDecimal:
		return v.Decimal.String()
	case ValBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case ValNone:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Equal checks equality of two Values.
func (v Value) Equal(other Value) bool {
	if v.Kind == ValInt && other.Kind == ValInt {
		return v.Int == other.Int
	}
	if v.Kind == ValString && other.Kind == ValString {
		return v.Str == other.Str
	}
	if v.Kind == ValBool && other.Kind == ValBool {
		return v.Bool == other.Bool
	}
	// Mixed numeric: compare as decimal
	if isNumeric(v.Kind) && isNumeric(other.Kind) {
		return v.ToDecimal().Equal(other.ToDecimal())
	}
	return false
}

func isNumeric(k ValueKind) bool {
	return k == ValInt || k == ValDecimal || k == ValDatetime
}

// RuntimeBlindSpot is a single entry returned by GetRuntimeBlindSpots.
type RuntimeBlindSpot struct {
	Builtin  string
	Count    int
	Severity string
}

// ParseNumberLiteral parses a numeric literal string into a Value.
// Handles integer and decimal literals.
func ParseNumberLiteral(s string) Value {
	if i, err := strconv.ParseInt(s, 10, 32); err == nil {
		return IntVal(int32(i))
	}
	if d, err := decimal.NewFromString(s); err == nil {
		return DecimalVal(d)
	}
	return NoneVal()
}
