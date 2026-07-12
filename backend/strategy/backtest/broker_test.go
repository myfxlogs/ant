package backtest

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

func TestSimBroker_PositionClosePartial(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
	})

	// Open a position
	res, err := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderMarket,
		Volume: decimal.NewFromFloat(1.0),
		Price:  decimal.NewFromFloat(1.1000),
		Magic:  123,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.RetCode != sdk.RetDone {
		t.Fatal("OrderSend failed")
	}

	ticket := res.Ticket
	// Set close price
	for _, p := range broker.positions {
		if p.Ticket == ticket {
			p.ClosePrice = decimal.NewFromFloat(1.1200)
			break
		}
	}

	// Partial close 0.5 lots
	res, err = broker.PositionClose(ticket, decimal.NewFromFloat(0.5))
	if err != nil {
		t.Fatal(err)
	}
	if res.RetCode != sdk.RetDone {
		t.Fatal("PositionClose partial failed")
	}
	if !res.Volume.Equal(decimal.NewFromFloat(0.5)) {
		t.Errorf("closed volume = %s, want 0.5", res.Volume)
	}

	// Verify position still open with reduced volume
	positions := broker.Positions(0)
	if len(positions) != 1 {
		t.Fatalf("expected 1 open position, got %d", len(positions))
	}
	if !positions[0].Volume.Equal(decimal.NewFromFloat(0.5)) {
		t.Errorf("remaining volume = %s, want 0.5", positions[0].Volume)
	}

	// Close the rest
	_, err = broker.PositionClose(ticket, decimal.Zero)
	if err != nil {
		t.Fatal(err)
	}
	if len(broker.Positions(0)) != 0 {
		t.Error("expected 0 open positions after full close")
	}

	// Verify deals recorded (partial + full = 2 deals)
	deals := broker.Deals(0, 0, 0)
	if len(deals) != 2 {
		t.Errorf("expected 2 deals, got %d", len(deals))
	}
}

func TestSimBroker_PositionCloseBy(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
	})

	// Open buy position
	res1, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderMarket,
		Volume: decimal.NewFromFloat(1.0),
		Price:  decimal.NewFromFloat(1.1000),
		Magic:  123,
	})

	// Open sell position (opposite)
	res2, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideSell,
		Type:   sdk.OrderMarket,
		Volume: decimal.NewFromFloat(1.0),
		Price:  decimal.NewFromFloat(1.1000),
		Magic:  123,
	})

	// Set close price
	closePrice := decimal.NewFromFloat(1.1100)
	for _, p := range broker.positions {
		p.ClosePrice = closePrice
	}

	// Close by
	res, err := broker.PositionCloseBy(res1.Ticket, res2.Ticket)
	if err != nil {
		t.Fatal(err)
	}
	if res.RetCode != sdk.RetDone {
		t.Fatal("PositionCloseBy failed")
	}

	// Both positions should be closed
	if len(broker.Positions(0)) != 0 {
		t.Errorf("expected 0 open positions, got %d", len(broker.Positions(0)))
	}

	// History should have 2 closed positions
	hist := broker.HistoryOrders(0, 0)
	if len(hist) != 2 {
		t.Errorf("expected 2 history orders, got %d", len(hist))
	}

	// Deals should have 2 deals
	deals := broker.Deals(0, 0, 0)
	if len(deals) != 2 {
		t.Errorf("expected 2 deals, got %d", len(deals))
	}
}

func TestSimBroker_HistoryOrders_TimeFilter(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		Leverage:       100,
	})

	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD",
		Side:   sdk.SideBuy,
		Type:   sdk.OrderMarket,
		Volume: decimal.NewFromFloat(1.0),
		Price:  decimal.NewFromFloat(1.1000),
	})

	for _, p := range broker.positions {
		if p.Ticket == res.Ticket {
			p.ClosePrice = decimal.NewFromFloat(1.1100)
			break
		}
	}

	broker.PositionClose(res.Ticket, decimal.Zero)

	// Query with wide time range
	hist := broker.HistoryOrders(0, time.Now().UnixMilli()+1000)
	if len(hist) != 1 {
		t.Errorf("expected 1 history order, got %d", len(hist))
	}

	// Query with narrow time range (future only — should return 0)
	hist = broker.HistoryOrders(time.Now().UnixMilli()+10000, 0)
	if len(hist) != 0 {
		t.Errorf("expected 0 history orders with future filter, got %d", len(hist))
	}
}

func TestSimBroker_SymbolInfo_Defaults(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
	})

	info, err := broker.SymbolInfo("EURUSD")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "EURUSD" {
		t.Errorf("Name = %s, want EURUSD", info.Name)
	}
	if info.Digits != 5 {
		t.Errorf("Digits = %d, want 5", info.Digits)
	}
	if !info.Point.Equal(decimal.NewFromFloat(0.00001)) {
		t.Errorf("Point = %s, want 0.00001", info.Point)
	}
}

func TestSimBroker_SymbolInfo_Custom(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: decimal.NewFromInt(100000),
		SymbolDigits:   3,
		SymbolPoint:    decimal.NewFromFloat(0.001),
		VolumeMin:      decimal.NewFromFloat(0.01),
		VolumeMax:      decimal.NewFromInt(100),
		VolumeStep:     decimal.NewFromFloat(0.01),
		ContractSize:   decimal.NewFromInt(100000),
	})

	info, _ := broker.SymbolInfo("USDJPY")
	if info.Digits != 3 {
		t.Errorf("Digits = %d, want 3", info.Digits)
	}
	if !info.Point.Equal(decimal.NewFromFloat(0.001)) {
		t.Errorf("Point = %s, want 0.001", info.Point)
	}
	if !info.VolumeMin.Equal(decimal.NewFromFloat(0.01)) {
		t.Errorf("VolumeMin = %s, want 0.01", info.VolumeMin)
	}
	if !info.ContractSize.Equal(decimal.NewFromInt(100000)) {
		t.Errorf("ContractSize = %s, want 100000", info.ContractSize)
	}
}
