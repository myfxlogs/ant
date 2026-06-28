package interp

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestValueIsTrue(t *testing.T) {
	if !IntVal(1).IsTrue() {
		t.Error("IntVal(1) should be true")
	}
	if IntVal(0).IsTrue() {
		t.Error("IntVal(0) should be false")
	}
	if !DecimalVal(decimal.NewFromInt(5)).IsTrue() {
		t.Error("DecimalVal(5) should be true")
	}
	if DecimalVal(decimal.Zero).IsTrue() {
		t.Error("DecimalVal(0) should be false")
	}
	if !BoolVal(true).IsTrue() {
		t.Error("BoolVal(true) should be true")
	}
}

func TestValueToDecimal(t *testing.T) {
	if !IntVal(42).ToDecimal().Equal(decimal.NewFromInt(42)) {
		t.Error("IntVal(42).ToDecimal() should be 42")
	}
	if !DecimalVal(decimal.NewFromFloat(3.14)).ToDecimal().Equal(decimal.NewFromFloat(3.14)) {
		t.Error("DecimalVal should round-trip")
	}
	if !BoolVal(true).ToDecimal().Equal(decimal.NewFromInt(1)) {
		t.Error("BoolVal(true).ToDecimal() should be 1")
	}
}

func TestValueEqual(t *testing.T) {
	if !IntVal(5).Equal(DecimalVal(decimal.NewFromInt(5))) {
		t.Error("IntVal(5) should equal DecimalVal(5)")
	}
	if IntVal(5).Equal(IntVal(6)) {
		t.Error("IntVal(5) should not equal IntVal(6)")
	}
	if !StringVal("hello").Equal(StringVal("hello")) {
		t.Error("string equality failed")
	}
}

func TestApplyOp(t *testing.T) {
	it := &Interpreter{globals: map[string]Value{}, locals: map[string]Value{}}

	// Addition
	r := it.applyOp(IntVal(3), IntVal(4), "+")
	if !r.ToDecimal().Equal(decimal.NewFromInt(7)) {
		t.Errorf("3 + 4 = %s, want 7", r.ToDecimal())
	}

	// String concatenation
	r = it.applyOp(StringVal("hello "), StringVal("world"), "+")
	if r.ToString() != "hello world" {
		t.Errorf("string concat = %s, want 'hello world'", r.ToString())
	}

	// Comparison
	r = it.applyOp(IntVal(5), IntVal(3), ">")
	if !r.IsTrue() {
		t.Error("5 > 3 should be true")
	}

	// Logical
	r = it.applyOp(BoolVal(true), BoolVal(false), "&&")
	if r.IsTrue() {
		t.Error("true && false should be false")
	}
}

func TestApplyUnary(t *testing.T) {
	it := &Interpreter{globals: map[string]Value{}, locals: map[string]Value{}}

	r := it.applyUnary(IntVal(5), "-")
	if r.ToInt() != -5 {
		t.Errorf("-(5) = %d, want -5", r.ToInt())
	}

	r = it.applyUnary(BoolVal(false), "!")
	if !r.IsTrue() {
		t.Error("!false should be true")
	}
}

func TestParseNumberLiteral(t *testing.T) {
	v := ParseNumberLiteral("42")
	if v.Kind != ValInt || v.Int != 42 {
		t.Errorf("ParseNumberLiteral(42) = %v, want IntVal(42)", v)
	}

	v = ParseNumberLiteral("3.14")
	if v.Kind != ValDecimal {
		t.Errorf("ParseNumberLiteral(3.14) kind = %v, want ValDecimal", v.Kind)
	}
}

func TestLookupConstant(t *testing.T) {
	it := &Interpreter{globals: map[string]Value{}, locals: map[string]Value{}}

	v := it.lookupConstant("OP_BUY")
	if v.ToInt() != 0 {
		t.Errorf("OP_BUY = %d, want 0", v.ToInt())
	}

	v = it.lookupConstant("OP_SELL")
	if v.ToInt() != 1 {
		t.Errorf("OP_SELL = %d, want 1", v.ToInt())
	}

	v = it.lookupConstant("PRICE_CLOSE")
	if v.ToInt() != 1 {
		t.Errorf("PRICE_CLOSE = %d, want 1", v.ToInt())
	}
}

func TestMQL4OrderPool(t *testing.T) {
	pool := &MQL4OrderPool{}
	if pool.Total() != 0 {
		t.Errorf("empty pool Total() = %d, want 0", pool.Total())
	}
	if pool.Select(0) {
		t.Error("Select(0) on empty pool should return false")
	}
}
