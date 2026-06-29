package interp

import (
	"testing"

	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
)

func TestOnTrade_Empty(t *testing.T) {
	ir := &IR{Version: "mql5"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	sig, err := it.OnTrade(ctx, sdk.TradeEvent{
		Ticket: 100,
		Symbol: "EURUSD",
	})
	if err != nil {
		t.Fatal(err)
	}
	if sig != nil {
		t.Error("expected nil signal when no OnTrade/OnTradeTransaction in IR")
	}
}

func TestOnTrade_Executed(t *testing.T) {
	ir := &IR{
		Version: "mql5",
		OnTrade: []Statement{
			{
				Kind: StmtExpr,
				Expr: ptr(callExpr("Print",
					Expr{Kind: ExprLiteral, Val: StringVal("trade event")},
				)),
			},
		},
	}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	_, err := it.OnTrade(ctx, sdk.TradeEvent{
		Ticket:    100,
		Symbol:    "EURUSD",
		EventType: sdk.TradeFilled,
		Volume:    decimal.NewFromFloat(1.0),
		Price:     decimal.NewFromFloat(1.1),
	})
	if err != nil {
		t.Fatalf("OnTrade failed: %v", err)
	}
}

func TestOnTradeTransaction_GlobalsExposed(t *testing.T) {
	ir := &IR{
		Version: "mql5",
		OnTradeTransaction: []Statement{
			{
				Kind: StmtExpr,
				Expr: ptr(callExpr("Print",
					Expr{Kind: ExprVar, Name: "_TransactionTicket"},
				)),
			},
		},
	}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	_, err := it.OnTrade(ctx, sdk.TradeEvent{
		Ticket:    42,
		Symbol:    "EURUSD",
		EventType: sdk.TradeClosed,
		Volume:    decimal.NewFromFloat(0.5),
		Price:     decimal.NewFromFloat(1.2),
		Profit:    decimal.NewFromFloat(100),
	})
	if err != nil {
		t.Fatalf("OnTrade failed: %v", err)
	}

	// Verify globals were set
	if v, ok := it.globals["_TransactionTicket"]; !ok || v.ToInt() != 42 {
		t.Errorf("_TransactionTicket = %v (ok=%v), want 42", v, ok)
	}
	if v, ok := it.globals["_TransactionType"]; !ok || v.ToInt() != 1 {
		t.Errorf("_TransactionType = %v (ok=%v), want 1 (TradeClosed)", v, ok)
	}
	if v, ok := it.globals["_TransactionSymbol"]; !ok || v.ToString() != "EURUSD" {
		t.Errorf("_TransactionSymbol = %v (ok=%v), want EURUSD", v, ok)
	}
}

func TestOnTrade_BothCallbacks(t *testing.T) {
	callCount := 0
	ir := &IR{
		Version: "mql5",
		OnTrade: []Statement{
			{
				Kind: StmtExpr,
				Expr: &Expr{Kind: ExprCall, Name: "OrdersHistoryTotal"},
			},
		},
		OnTradeTransaction: []Statement{
			{
				Kind: StmtExpr,
				Expr: &Expr{Kind: ExprCall, Name: "OrdersTotal"},
			},
		},
		Funcs: map[string]*FuncDef{},
	}
	// Wrap to count execution — we just verify no errors
	_ = callCount
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	_, err := it.OnTrade(ctx, sdk.TradeEvent{
		Ticket:    1,
		EventType: sdk.TradeFilled,
	})
	if err != nil {
		t.Fatalf("OnTrade with both callbacks failed: %v", err)
	}
}

func TestSerializeIR_OnTrade_RoundTrip(t *testing.T) {
	ir := &IR{
		Version: "mql5",
		OnTrade: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("Print",
				Expr{Kind: ExprLiteral, Val: StringVal("trade")},
			))},
		},
		OnTradeTransaction: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("Print",
				Expr{Kind: ExprLiteral, Val: StringVal("txn")},
			))},
		},
		Funcs: map[string]*FuncDef{},
	}

	data := SerializeIR(ir)
	restored := DeserializeIR(data)
	if restored == nil {
		t.Fatal("DeserializeIR returned nil")
	}
	if len(restored.OnTrade) != 1 {
		t.Errorf("OnTrade len = %d, want 1", len(restored.OnTrade))
	}
	if len(restored.OnTradeTransaction) != 1 {
		t.Errorf("OnTradeTransaction len = %d, want 1", len(restored.OnTradeTransaction))
	}
}
