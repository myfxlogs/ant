package backtest

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// TestPhaseA_PendingOrderVisibleInOrdersTotal verifies that OrdersTotal includes pending orders.
func TestPhaseA_PendingOrderVisibleInOrdersTotal(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	})
	broker.SetBarPrice(decimal.NewFromFloat(1.1000))
	broker.SetBarTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	// No orders initially
	positions := broker.Positions(0)
	orders := broker.Orders(0)
	if len(positions)+len(orders) != 0 {
		t.Fatalf("expected 0 total orders, got positions=%d orders=%d", len(positions), len(orders))
	}

	// Send a buy limit pending order
	res, err := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderLimit,
		Volume: decimal.NewFromFloat(0.1),
		Price:  decimal.NewFromFloat(1.0900),
	})
	if err != nil {
		t.Fatalf("OrderSend failed: %v", err)
	}
	if res.RetCode != sdk.RetDone {
		t.Fatalf("OrderSend returned non-done: %v", res.RetCode)
	}

	// Orders should now include the pending order
	orders = broker.Orders(0)
	if len(orders) != 1 {
		t.Fatalf("expected 1 pending order, got %d", len(orders))
	}
	positions = broker.Positions(0)
	if len(positions) != 0 {
		t.Fatalf("expected 0 open positions, got %d", len(positions))
	}

	// Total = positions + pending orders = 0 + 1 = 1
	total := len(broker.Positions(0)) + len(broker.Orders(0))
	if total != 1 {
		t.Fatalf("expected OrdersTotal=1, got %d", total)
	}
}

// TestPhaseA_PendingOrderTypeCorrect verifies OrderType returns correct OP_* for pending orders.
func TestPhaseA_PendingOrderTypeCorrect(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	})
	broker.SetBarPrice(decimal.NewFromFloat(1.1000))
	broker.SetBarTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	// Buy Limit → OP_BUYLIMIT=2
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderLimit,
		Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.0900),
	})
	orders := broker.Orders(0)
	if len(orders) != 1 {
		t.Fatalf("expected 1 pending order, got %d", len(orders))
	}
	if orders[0].Type != sdk.OrderLimit {
		t.Errorf("expected OrderLimit, got %v", orders[0].Type)
	}
	// Verify the MQL4 mapping: BuyLimit + SideBuy → OP_BUYLIMIT=2
	mqlType := orderTypeToMQL4ForTest(orders[0].Type, orders[0].Side)
	if mqlType != 2 {
		t.Errorf("BuyLimit: expected OP_BUYLIMIT=2, got %d", mqlType)
	}

	// Sell Limit → OP_SELLLIMIT=3
	broker2 := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	})
	broker2.SetBarPrice(decimal.NewFromFloat(1.1000))
	broker2.SetBarTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	broker2.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideSell, Type: sdk.OrderLimit,
		Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.1100),
	})
	orders2 := broker2.Orders(0)
	if len(orders2) != 1 {
		t.Fatalf("expected 1 pending order, got %d", len(orders2))
	}
	mqlType2 := orderTypeToMQL4ForTest(orders2[0].Type, orders2[0].Side)
	if mqlType2 != 3 {
		t.Errorf("SellLimit: expected OP_SELLLIMIT=3, got %d", mqlType2)
	}

	// Buy Stop → OP_BUYSTOP=4
	broker3 := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	})
	broker3.SetBarPrice(decimal.NewFromFloat(1.1000))
	broker3.SetBarTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	broker3.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderStop,
		Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.1100),
	})
	orders3 := broker3.Orders(0)
	if len(orders3) != 1 {
		t.Fatalf("expected 1 pending order, got %d", len(orders3))
	}
	mqlType3 := orderTypeToMQL4ForTest(orders3[0].Type, orders3[0].Side)
	if mqlType3 != 4 {
		t.Errorf("BuyStop: expected OP_BUYSTOP=4, got %d", mqlType3)
	}

	// Sell Stop → OP_SELLSTOP=5
	broker4 := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	})
	broker4.SetBarPrice(decimal.NewFromFloat(1.1000))
	broker4.SetBarTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	broker4.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideSell, Type: sdk.OrderStop,
		Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.0900),
	})
	orders4 := broker4.Orders(0)
	if len(orders4) != 1 {
		t.Fatalf("expected 1 pending order, got %d", len(orders4))
	}
	mqlType4 := orderTypeToMQL4ForTest(orders4[0].Type, orders4[0].Side)
	if mqlType4 != 5 {
		t.Errorf("SellStop: expected OP_SELLSTOP=5, got %d", mqlType4)
	}
}

// orderTypeToMQL4ForTest mirrors the mql2go orderTypeToMQL4 logic for backtest package testing.
func orderTypeToMQL4ForTest(ot sdk.OrderType, side sdk.PositionSide) int32 {
	switch ot {
	case sdk.OrderLimit:
		if side == sdk.SideSell {
			return 3 // OP_SELLLIMIT
		}
		return 2 // OP_BUYLIMIT
	case sdk.OrderStop:
		if side == sdk.SideSell {
			return 5 // OP_SELLSTOP
		}
		return 4 // OP_BUYSTOP
	default:
		return 0
	}
}

// TestPhaseA_PositionModifyOnPendingOrder verifies that PositionModify can modify SL/TP on pending orders.
func TestPhaseA_PositionModifyOnPendingOrder(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	})
	broker.SetBarPrice(decimal.NewFromFloat(1.1000))
	broker.SetBarTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	res, err := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderLimit,
		Volume: decimal.NewFromFloat(0.1),
		Price:  decimal.NewFromFloat(1.0900),
	})
	if err != nil {
		t.Fatalf("OrderSend failed: %v", err)
	}

	// Modify SL/TP on the pending order
	newSL := decimal.NewFromFloat(1.0800)
	newTP := decimal.NewFromFloat(1.1200)
	_, err = broker.PositionModify(res.Ticket, newSL, newTP)
	if err != nil {
		t.Fatalf("PositionModify on pending order failed: %v", err)
	}

	// Verify the pending order was modified
	orders := broker.Orders(0)
	if len(orders) != 1 {
		t.Fatalf("expected 1 pending order, got %d", len(orders))
	}
	if !orders[0].StopLoss.Equal(newSL) {
		t.Errorf("SL: got %s, want %s", orders[0].StopLoss.String(), newSL.String())
	}
	if !orders[0].TakeProfit.Equal(newTP) {
		t.Errorf("TP: got %s, want %s", orders[0].TakeProfit.String(), newTP.String())
	}
}

// TestPhaseA_PositionModifyPriceOnPendingOrder verifies that PositionModifyPrice can change the entry price.
func TestPhaseA_PositionModifyPriceOnPendingOrder(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	})
	broker.SetBarPrice(decimal.NewFromFloat(1.1000))
	broker.SetBarTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	res, err := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderLimit,
		Volume: decimal.NewFromFloat(0.1),
		Price:  decimal.NewFromFloat(1.0900),
	})
	if err != nil {
		t.Fatalf("OrderSend failed: %v", err)
	}

	// Modify the price of the pending order
	newPrice := decimal.NewFromFloat(1.0850)
	_, err = broker.PositionModifyPrice(res.Ticket, newPrice)
	if err != nil {
		t.Fatalf("PositionModifyPrice failed: %v", err)
	}

	orders := broker.Orders(0)
	if len(orders) != 1 {
		t.Fatalf("expected 1 pending order, got %d", len(orders))
	}
	if !orders[0].Price.Equal(newPrice) {
		t.Errorf("Price: got %s, want %s", orders[0].Price.String(), newPrice.String())
	}
}

// TestPhaseA_PendingOrderFillsAndOrdersTotalDecreases verifies that after a pending order fills,
// it moves from pending to positions, and OrdersTotal reflects the change correctly.
func TestPhaseA_PendingOrderFillsAndOrdersTotalDecreases(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	})
	broker.SetBarPrice(decimal.NewFromFloat(1.1000))
	broker.SetBarTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	// Place a buy limit at 1.0900
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderLimit,
		Volume: decimal.NewFromFloat(0.1),
		Price:  decimal.NewFromFloat(1.0900),
	})

	// Before fill: 0 positions + 1 pending = 1 total
	if len(broker.Positions(0))+len(broker.Orders(0)) != 1 {
		t.Fatalf("before fill: expected total=1, got %d", len(broker.Positions(0))+len(broker.Orders(0)))
	}

	// Bar with Low <= 1.0900 triggers the fill
	bar := sdk.Bar{
		Open:      decimal.NewFromFloat(1.1000),
		High:      decimal.NewFromFloat(1.1080),
		Low:       decimal.NewFromFloat(1.0880),
		Close:     decimal.NewFromFloat(1.1050),
		Volume:    1000,
		Timestamp: time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC).UnixMilli(),
	}

	engine := &Engine{broker: broker}
	engine.checkPendingOrders(bar)

	// After fill: 1 position + 0 pending = 1 total (but now it's a position, not pending)
	positions := broker.Positions(0)
	orders := broker.Orders(0)
	if len(positions) != 1 {
		t.Fatalf("after fill: expected 1 position, got %d", len(positions))
	}
	if len(orders) != 0 {
		t.Fatalf("after fill: expected 0 pending orders, got %d", len(orders))
	}
}

// TestPhaseA_OrderDeleteOnPendingNoRegression verifies OrderDelete still works correctly.
func TestPhaseA_OrderDeleteOnPendingNoRegression(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
		ContractSize:   decimal.NewFromInt(100000),
	})
	broker.SetBarPrice(decimal.NewFromFloat(1.1000))
	broker.SetBarTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	res, err := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderLimit,
		Volume: decimal.NewFromFloat(0.1),
		Price:  decimal.NewFromFloat(1.0900),
	})
	if err != nil {
		t.Fatalf("OrderSend failed: %v", err)
	}

	// Delete the pending order
	_, err = broker.OrderDelete(res.Ticket)
	if err != nil {
		t.Fatalf("OrderDelete failed: %v", err)
	}

	// Should have 0 pending and 0 positions
	if len(broker.Orders(0)) != 0 {
		t.Errorf("expected 0 pending after delete, got %d", len(broker.Orders(0)))
	}
	if len(broker.Positions(0)) != 0 {
		t.Errorf("expected 0 positions after delete, got %d", len(broker.Positions(0)))
	}
}
