package interp

import (
	"testing"

	"github.com/shopspring/decimal"
)

func makeArrayVal(vals ...float64) Value {
	arr := make([]Value, len(vals))
	for i, v := range vals {
		arr[i] = DecimalVal(decimal.NewFromFloat(v))
	}
	return Value{Kind: ValArray, Array: arr}
}

func TestIMAOnArray_SMA(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// iMAOnArray(array, total=0, period=3, ma_shift=0, ma_method=0(SMA), shift=0)
	arr := makeArrayVal(1.0, 2.0, 3.0, 4.0, 5.0)
	v := it.callBuiltin("iMAOnArray", []Expr{
		{Kind: ExprLiteral, Val: arr},
		{Kind: ExprLiteral, Val: IntVal(0)},  // total
		{Kind: ExprLiteral, Val: IntVal(3)},  // period
		{Kind: ExprLiteral, Val: IntVal(0)},  // ma_shift
		{Kind: ExprLiteral, Val: IntVal(0)},  // SMA
		{Kind: ExprLiteral, Val: IntVal(0)},  // shift
	})
	expected := decimal.NewFromFloat(2.0) // (1+2+3)/3
	if !v.ToDecimal().Equal(expected) {
		t.Errorf("iMAOnArray(SMA) = %s, want %s", v.ToDecimal(), expected)
	}
}

func TestIMAOnArray_EMA(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// iMAOnArray with EMA method=1
	arr := makeArrayVal(1.0, 2.0, 3.0, 4.0, 5.0)
	v := it.callBuiltin("iMAOnArray", []Expr{
		{Kind: ExprLiteral, Val: arr},
		{Kind: ExprLiteral, Val: IntVal(0)},
		{Kind: ExprLiteral, Val: IntVal(3)}, // period
		{Kind: ExprLiteral, Val: IntVal(0)},
		{Kind: ExprLiteral, Val: IntVal(1)}, // EMA
		{Kind: ExprLiteral, Val: IntVal(0)},
	})
	if v.ToDecimal().Equal(decimal.Zero) {
		t.Error("iMAOnArray(EMA) returned 0, expected non-zero")
	}
}

func TestIRSIOnArray(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// Rising prices → RSI should be high (close to 100)
	// MQL array index 0 = most recent, so highest value first
	arr := makeArrayVal(11.0, 10.0, 9.0, 8.0, 7.0, 6.0, 5.0, 4.0, 3.0, 2.0, 1.0)
	v := it.callBuiltin("iRSIOnArray", []Expr{
		{Kind: ExprLiteral, Val: arr},
		{Kind: ExprLiteral, Val: IntVal(0)},  // total
		{Kind: ExprLiteral, Val: IntVal(10)}, // period
		{Kind: ExprLiteral, Val: IntVal(0)},  // shift
	})
	rsi := v.ToDecimal()
	if rsi.LessThan(decimal.NewFromInt(90)) {
		t.Errorf("iRSIOnArray(rising prices) = %s, expected > 90", rsi)
	}
}

func TestIStdDevOnArray(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	arr := makeArrayVal(1.0, 2.0, 3.0, 4.0, 5.0)
	v := it.callBuiltin("iStdDevOnArray", []Expr{
		{Kind: ExprLiteral, Val: arr},
		{Kind: ExprLiteral, Val: IntVal(0)},  // total
		{Kind: ExprLiteral, Val: IntVal(5)},  // ma_period
		{Kind: ExprLiteral, Val: IntVal(0)},  // ma_shift
		{Kind: ExprLiteral, Val: IntVal(0)},  // SMA
		{Kind: ExprLiteral, Val: IntVal(0)},  // shift
	})
	// StdDev of [1,2,3,4,5] with sample variance = sqrt(2.5) ≈ 1.581
	sd := v.ToDecimal()
	expected := decimal.NewFromFloat(1.5811388300841898)
	if sd.Sub(expected).Abs().GreaterThan(decimal.NewFromFloat(0.001)) {
		t.Errorf("iStdDevOnArray = %s, expected ~%s", sd, expected)
	}
}

func TestIMomentumOnArray(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	arr := makeArrayVal(10.0, 20.0, 30.0, 40.0, 50.0)
	v := it.callBuiltin("iMomentumOnArray", []Expr{
		{Kind: ExprLiteral, Val: arr},
		{Kind: ExprLiteral, Val: IntVal(0)},  // total
		{Kind: ExprLiteral, Val: IntVal(2)},  // period
		{Kind: ExprLiteral, Val: IntVal(0)},  // shift
	})
	// Momentum = Close[0] - Close[2] = 10 - 30 = -20
	expected := decimal.NewFromInt(-20)
	if !v.ToDecimal().Equal(expected) {
		t.Errorf("iMomentumOnArray = %s, want %s", v.ToDecimal(), expected)
	}
}

func TestIATROnArray(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	arr := makeArrayVal(1.0, 2.0, 3.0, 4.0, 5.0)
	v := it.callBuiltin("iATROnArray", []Expr{
		{Kind: ExprLiteral, Val: arr},
		{Kind: ExprLiteral, Val: IntVal(0)},
		{Kind: ExprLiteral, Val: IntVal(3)},
		{Kind: ExprLiteral, Val: IntVal(0)},
	})
	// ATR on array where all OHLC = same → TR = 0, ATR = 0
	if !v.ToDecimal().Equal(decimal.Zero) {
		t.Errorf("iATROnArray = %s, want 0 (all OHLC same)", v.ToDecimal())
	}
}

func TestAnalyze_OnArray_NotBlindSpot(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("iMAOnArray",
				Expr{Kind: ExprLiteral, Val: makeArrayVal(1.0, 2.0, 3.0)},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
				Expr{Kind: ExprLiteral, Val: IntVal(3)},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
			))},
			{Kind: StmtExpr, Expr: ptr(callExpr("iRSIOnArray",
				Expr{Kind: ExprLiteral, Val: makeArrayVal(1.0, 2.0, 3.0)},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
				Expr{Kind: ExprLiteral, Val: IntVal(3)},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
			))},
		},
	}
	rep := Analyze(ir)
	if rep.Coverage != 1.0 {
		t.Errorf("Coverage = %.2f, want 1.0 (*OnArray are implemented)", rep.Coverage)
	}
	if len(rep.BlindSpots) != 0 {
		t.Errorf("BlindSpots = %v, want empty", rep.BlindSpots)
	}
}
