package interp

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestSymbolInfoDouble(t *testing.T) {
	ir := &IR{Version: "mql5"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// SymbolInfoDouble("EURUSD", SYMBOL_POINT=11)
	v := it.callBuiltin("SymbolInfoDouble", []Expr{
		{Kind: ExprLiteral, Val: StringVal("EURUSD")},
		{Kind: ExprLiteral, Val: IntVal(11)}, // SYMBOL_POINT
	})
	expected := decimal.NewFromFloat(0.00001)
	if !v.ToDecimal().Equal(expected) {
		t.Errorf("SymbolInfoDouble(SYMBOL_POINT) = %s, want %s", v.ToDecimal(), expected)
	}

	// SymbolInfoDouble("EURUSD", SYMBOL_VOLUME_MIN=23)
	v = it.callBuiltin("SymbolInfoDouble", []Expr{
		{Kind: ExprLiteral, Val: StringVal("EURUSD")},
		{Kind: ExprLiteral, Val: IntVal(23)},
	})
	expected = decimal.NewFromFloat(0.01)
	if !v.ToDecimal().Equal(expected) {
		t.Errorf("SymbolInfoDouble(SYMBOL_VOLUME_MIN) = %s, want %s", v.ToDecimal(), expected)
	}
}

func TestSymbolInfoInteger(t *testing.T) {
	ir := &IR{Version: "mql5"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// SymbolInfoInteger("EURUSD", SYMBOL_DIGITS=12)
	v := it.callBuiltin("SymbolInfoInteger", []Expr{
		{Kind: ExprLiteral, Val: StringVal("EURUSD")},
		{Kind: ExprLiteral, Val: IntVal(12)}, // SYMBOL_DIGITS
	})
	if v.ToInt() != 5 {
		t.Errorf("SymbolInfoInteger(SYMBOL_DIGITS) = %d, want 5", v.ToInt())
	}

	// SymbolInfoInteger("EURUSD", SYMBOL_SPREAD=13)
	v = it.callBuiltin("SymbolInfoInteger", []Expr{
		{Kind: ExprLiteral, Val: StringVal("EURUSD")},
		{Kind: ExprLiteral, Val: IntVal(13)},
	})
	if v.ToInt() != 2 {
		t.Errorf("SymbolInfoInteger(SYMBOL_SPREAD) = %d, want 2", v.ToInt())
	}
}

func TestSymbolInfoString(t *testing.T) {
	ir := &IR{Version: "mql5"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// SymbolInfoString("EURUSD", SYMBOL_CURRENCY_BASE=1)
	v := it.callBuiltin("SymbolInfoString", []Expr{
		{Kind: ExprLiteral, Val: StringVal("EURUSD")},
		{Kind: ExprLiteral, Val: IntVal(1)},
	})
	if v.ToString() != "EURUSD" {
		t.Errorf("SymbolInfoString(SYMBOL_CURRENCY_BASE) = %q, want EURUSD", v.ToString())
	}
}

func TestMarketInfo(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// MarketInfo("EURUSD", MODE_POINT=3)
	v := it.callBuiltin("MarketInfo", []Expr{
		{Kind: ExprLiteral, Val: StringVal("EURUSD")},
		{Kind: ExprLiteral, Val: IntVal(3)}, // MODE_POINT
	})
	expected := decimal.NewFromFloat(0.00001)
	if !v.ToDecimal().Equal(expected) {
		t.Errorf("MarketInfo(MODE_POINT) = %s, want %s", v.ToDecimal(), expected)
	}

	// MarketInfo("EURUSD", MODE_DIGITS=4)
	v = it.callBuiltin("MarketInfo", []Expr{
		{Kind: ExprLiteral, Val: StringVal("EURUSD")},
		{Kind: ExprLiteral, Val: IntVal(4)}, // MODE_DIGITS
	})
	if v.ToInt() != 5 {
		t.Errorf("MarketInfo(MODE_DIGITS) = %d, want 5", v.ToInt())
	}

	// MarketInfo("EURUSD", MODE_MINLOT=17)
	v = it.callBuiltin("MarketInfo", []Expr{
		{Kind: ExprLiteral, Val: StringVal("EURUSD")},
		{Kind: ExprLiteral, Val: IntVal(17)}, // MODE_MINLOT
	})
	expected = decimal.NewFromFloat(0.01)
	if !v.ToDecimal().Equal(expected) {
		t.Errorf("MarketInfo(MODE_MINLOT) = %s, want %s", v.ToDecimal(), expected)
	}
}

func TestAnalyze_SymbolInfo_NotBlindSpot(t *testing.T) {
	ir := &IR{
		Version: "mql5",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("SymbolInfoDouble",
				Expr{Kind: ExprLiteral, Val: StringVal("EURUSD")},
				Expr{Kind: ExprLiteral, Val: IntVal(11)},
			))},
			{Kind: StmtExpr, Expr: ptr(callExpr("MarketInfo",
				Expr{Kind: ExprLiteral, Val: StringVal("EURUSD")},
				Expr{Kind: ExprLiteral, Val: IntVal(3)},
			))},
		},
	}
	rep := Analyze(ir)
	if rep.Coverage != 1.0 {
		t.Errorf("Coverage = %.2f, want 1.0 (SymbolInfo/MarketInfo are implemented)", rep.Coverage)
	}
	if len(rep.BlindSpots) != 0 {
		t.Errorf("BlindSpots = %v, want empty", rep.BlindSpots)
	}
}
