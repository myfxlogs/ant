package interp

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestBuiltin_ArrayFunctions(t *testing.T) {
	it := &Interpreter{
		globals: make(map[string]Value),
		locals:  make(map[string]Value),
	}

	// Initialize an array
	it.globals["myArray"] = Value{Kind: ValArray, Array: []Value{
		DecimalVal(decimal.NewFromInt(3)),
		DecimalVal(decimal.NewFromInt(1)),
		DecimalVal(decimal.NewFromInt(2)),
	}}

	// ArraySize
	v, err := arraySize(it, []Value{StringVal("myArray")})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToInt() != 3 {
		t.Errorf("ArraySize = %d, want 3", v.ToInt())
	}

	// ArrayMaximum
	v, err = arrayMaximum(it, []Value{StringVal("myArray")})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToInt() != 0 {
		t.Errorf("ArrayMaximum = %d, want 0", v.ToInt())
	}

	// ArrayMinimum
	v, err = arrayMinimum(it, []Value{StringVal("myArray")})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToInt() != 1 {
		t.Errorf("ArrayMinimum = %d, want 1", v.ToInt())
	}

	// ArraySort
	_, err = arraySort(it, []Value{StringVal("myArray")})
	if err != nil {
		t.Fatal(err)
	}
	arr, _ := it.getArray("myArray")
	if arr[0].ToDecimal().IntPart() != 1 || arr[2].ToDecimal().IntPart() != 3 {
		t.Errorf("ArraySort result = %v, want [1,2,3]", arr)
	}

	// ArrayResize
	v, err = arrayResize(it, []Value{StringVal("myArray"), IntVal(5)})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToInt() != 5 {
		t.Errorf("ArrayResize = %d, want 5", v.ToInt())
	}
	arr, _ = it.getArray("myArray")
	if len(arr) != 5 {
		t.Errorf("Array resized len = %d, want 5", len(arr))
	}
}

func TestBuiltin_StringFunctions(t *testing.T) {
	it := &Interpreter{
		globals: make(map[string]Value),
		locals:  make(map[string]Value),
	}

	// StringConcatenate
	v, err := stringConcatenate(it, []Value{StringVal("hello"), StringVal(" "), StringVal("world")})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToString() != "hello world" {
		t.Errorf("StringConcatenate = %s, want 'hello world'", v.ToString())
	}

	// StringFind
	v, err = stringFind(it, []Value{StringVal("hello world"), StringVal("world")})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToInt() != 6 {
		t.Errorf("StringFind = %d, want 6", v.ToInt())
	}

	// StringSubstr
	v, err = stringSubstr(it, []Value{StringVal("hello world"), IntVal(6), IntVal(5)})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToString() != "world" {
		t.Errorf("StringSubstr = %s, want 'world'", v.ToString())
	}

	// StringLen
	v, err = stringLen(it, []Value{StringVal("hello")})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToInt() != 5 {
		t.Errorf("StringLen = %d, want 5", v.ToInt())
	}

	// StringTrimLeft
	v, err = stringTrimLeft(it, []Value{StringVal("  hello  ")})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToString() != "hello  " {
		t.Errorf("StringTrimLeft = %q, want 'hello  '", v.ToString())
	}
}

func TestBuiltin_ConversionFunctions(t *testing.T) {
	it := &Interpreter{
		globals: make(map[string]Value),
		locals:  make(map[string]Value),
	}

	// DoubleToString
	v, err := doubleToString(it, []Value{DecimalVal(decimal.NewFromFloat(3.14159)), IntVal(2)})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToString() != "3.14" {
		t.Errorf("DoubleToString = %s, want '3.14'", v.ToString())
	}

	// IntegerToString
	v, err = integerToString(it, []Value{IntVal(42)})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToString() != "42" {
		t.Errorf("IntegerToString = %s, want '42'", v.ToString())
	}

	// StringToDouble
	v, err = stringToDouble(it, []Value{StringVal("3.14")})
	if err != nil {
		t.Fatal(err)
	}
	if !v.ToDecimal().Equal(decimal.NewFromFloat(3.14)) {
		t.Errorf("StringToDouble = %s, want 3.14", v.ToDecimal())
	}

	// StringToInteger
	v, err = stringToInteger(it, []Value{StringVal("123")})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToInt() != 123 {
		t.Errorf("StringToInteger = %d, want 123", v.ToInt())
	}

	// NormalizeDouble
	v, err = normalizeDouble(it, []Value{DecimalVal(decimal.NewFromFloat(1.123456)), IntVal(2)})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToDecimal().String() != "1.12" {
		t.Errorf("NormalizeDouble = %s, want 1.12", v.ToDecimal().String())
	}
}

func TestBuiltin_MathFunctions(t *testing.T) {
	it := &Interpreter{
		globals: make(map[string]Value),
		locals:  make(map[string]Value),
	}

	// MathAbs
	v, err := mathAbs(it, []Value{DecimalVal(decimal.NewFromInt(-5))})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToDecimal().IntPart() != 5 {
		t.Errorf("MathAbs(-5) = %s, want 5", v.ToDecimal())
	}

	// MathMax
	v, err = mathMax(it, []Value{DecimalVal(decimal.NewFromInt(3)), DecimalVal(decimal.NewFromInt(7))})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToDecimal().IntPart() != 7 {
		t.Errorf("MathMax(3,7) = %s, want 7", v.ToDecimal())
	}

	// MathMin
	v, err = mathMin(it, []Value{DecimalVal(decimal.NewFromInt(3)), DecimalVal(decimal.NewFromInt(7))})
	if err != nil {
		t.Fatal(err)
	}
	if v.ToDecimal().IntPart() != 3 {
		t.Errorf("MathMin(3,7) = %s, want 3", v.ToDecimal())
	}
}

func TestBuiltin_UnimplementedFunction(t *testing.T) {
	it := &Interpreter{
		globals: make(map[string]Value),
		locals:  make(map[string]Value),
	}

	// Call an unimplemented function
	v := it.callBuiltin("NonExistentFunction", nil)
	if v.Kind != ValNone {
		t.Errorf("Unimplemented function should return NoneVal, got %v", v.Kind)
	}
	if !it.errSet {
		t.Error("errSet should be true after unimplemented function call")
	}
}
