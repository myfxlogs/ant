package interp

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
)

func TestOrderCloseBy(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// OrderCloseBy(ticket1, ticket2, color)
	v := it.callBuiltin("OrderCloseBy", []Expr{
		{Kind: ExprLiteral, Val: IntVal(1001)},
		{Kind: ExprLiteral, Val: IntVal(1002)},
		{Kind: ExprLiteral, Val: IntVal(0)}, // color (ignored)
	})
	if !v.IsTrue() {
		t.Error("OrderCloseBy returned false, want true")
	}
}

func TestOrdersHistoryTotal_Empty(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	v := it.callBuiltin("OrdersHistoryTotal", nil)
	if v.ToInt() != 0 {
		t.Errorf("OrdersHistoryTotal = %d, want 0 (empty history)", v.ToInt())
	}
}

func TestOrdersHistoryTotal_WithHistory(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	broker := &mockBroker{
		history: []sdk.Position{
			{Ticket: 100, Symbol: "EURUSD", Side: sdk.SideBuy, Volume: decimal.NewFromFloat(1.0), OpenPrice: decimal.NewFromFloat(1.1), OpenTime: time.Now()},
			{Ticket: 200, Symbol: "EURUSD", Side: sdk.SideSell, Volume: decimal.NewFromFloat(0.5), OpenPrice: decimal.NewFromFloat(1.2), OpenTime: time.Now()},
		},
	}
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: broker,
	}
	it.OnInit(ctx)
	it.orderPool.Reset(ctx)

	// OrdersHistoryTotal should return 2
	v := it.callBuiltin("OrdersHistoryTotal", nil)
	if v.ToInt() != 2 {
		t.Errorf("OrdersHistoryTotal = %d, want 2", v.ToInt())
	}

	// OrderSelect(0, SELECT_BY_POS, MODE_HISTORY) should succeed
	v = it.callBuiltin("OrderSelect", []Expr{
		{Kind: ExprLiteral, Val: IntVal(0)},
		{Kind: ExprLiteral, Val: IntVal(0)}, // SELECT_BY_POS
		{Kind: ExprLiteral, Val: IntVal(1)}, // MODE_HISTORY
	})
	if !v.IsTrue() {
		t.Error("OrderSelect(0, SELECT_BY_POS, MODE_HISTORY) returned false, want true")
	}

	// OrderTicket() should return 100
	v = it.callBuiltin("OrderTicket", nil)
	if v.ToInt() != 100 {
		t.Errorf("OrderTicket() = %d, want 100", v.ToInt())
	}

	// OrderSelect(1, SELECT_BY_POS, MODE_HISTORY) should succeed
	v = it.callBuiltin("OrderSelect", []Expr{
		{Kind: ExprLiteral, Val: IntVal(1)},
		{Kind: ExprLiteral, Val: IntVal(0)}, // SELECT_BY_POS
		{Kind: ExprLiteral, Val: IntVal(1)}, // MODE_HISTORY
	})
	if !v.IsTrue() {
		t.Error("OrderSelect(1, SELECT_BY_POS, MODE_HISTORY) returned false, want true")
	}

	// OrderTicket() should return 200
	v = it.callBuiltin("OrderTicket", nil)
	if v.ToInt() != 200 {
		t.Errorf("OrderTicket() = %d, want 200", v.ToInt())
	}

	// OrderSelect(2, SELECT_BY_POS, MODE_HISTORY) should fail (out of range)
	v = it.callBuiltin("OrderSelect", []Expr{
		{Kind: ExprLiteral, Val: IntVal(2)},
		{Kind: ExprLiteral, Val: IntVal(0)}, // SELECT_BY_POS
		{Kind: ExprLiteral, Val: IntVal(1)}, // MODE_HISTORY
	})
	if v.IsTrue() {
		t.Error("OrderSelect(2, SELECT_BY_POS, MODE_HISTORY) returned true, want false (out of range)")
	}
}

func TestOrderSelect_ModeTrades_Default(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// OrderSelect without third arg should default to MODE_TRADES
	v := it.callBuiltin("OrderSelect", []Expr{
		{Kind: ExprLiteral, Val: IntVal(0)},
		{Kind: ExprLiteral, Val: IntVal(0)}, // SELECT_BY_POS
	})
	// No active orders, should return false
	if v.IsTrue() {
		t.Error("OrderSelect(0, SELECT_BY_POS) returned true, want false (no active orders)")
	}
}

func TestAnalyze_OrderCloseBy_NotBlindSpot(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("OrderCloseBy",
				Expr{Kind: ExprLiteral, Val: IntVal(1)},
				Expr{Kind: ExprLiteral, Val: IntVal(2)},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
			))},
			{Kind: StmtExpr, Expr: ptr(callExpr("OrdersHistoryTotal"))},
		},
	}
	rep := Analyze(ir)
	if rep.Coverage != 1.0 {
		t.Errorf("Coverage = %.2f, want 1.0 (OrderCloseBy/OrdersHistoryTotal are implemented)", rep.Coverage)
	}
	if len(rep.BlindSpots) != 0 {
		t.Errorf("BlindSpots = %v, want empty", rep.BlindSpots)
	}
}
