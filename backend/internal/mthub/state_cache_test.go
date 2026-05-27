package mthub

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	return zap.NewNop()
}

func TestStateCache_GetSetOrder(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())

	ev := &TradeEvent{
		EventType: TradeEventOrderFilled,
		AccountID: "acc-1",
		Ticket:    100,
		Canonical: "EURUSD",
		Side:      "BUY",
		Volume:    0.1,
		Price:     1.0850,
		ToState:   "FILLED",
		Timestamp: time.Now(),
	}
	c.ApplyEvent(ev)

	order := c.GetOrder(100)
	if order == nil {
		t.Fatal("order should exist")
	}
	if order.State != "FILLED" {
		t.Fatalf("want FILLED, got %s", order.State)
	}
	if order.Canonical != "EURUSD" {
		t.Fatalf("want EURUSD, got %s", order.Canonical)
	}
}

func TestStateCache_GetOrdersByAccount(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())

	c.ApplyEvent(&TradeEvent{EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1, Canonical: "EURUSD", ToState: "FILLED", Timestamp: time.Now()})
	c.ApplyEvent(&TradeEvent{EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 2, Canonical: "GBPUSD", ToState: "FILLED", Timestamp: time.Now()})
	c.ApplyEvent(&TradeEvent{EventType: TradeEventOrderFilled, AccountID: "acc-2", Ticket: 3, Canonical: "USDJPY", ToState: "FILLED", Timestamp: time.Now()})

	orders := c.GetOrdersByAccount("acc-1")
	if len(orders) != 2 {
		t.Fatalf("want 2 orders for acc-1, got %d", len(orders))
	}

	orders = c.GetOrdersByAccount("acc-2")
	if len(orders) != 1 {
		t.Fatalf("want 1 order for acc-2, got %d", len(orders))
	}
}

func TestStateCache_PositionTracking(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())

	// Buy 0.1 EURUSD.
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "BUY", Volume: 0.1, Price: 1.0850,
		ToState: "FILLED", Timestamp: time.Now(),
	})

	pos := c.GetPosition("acc-1", "EURUSD")
	if pos == nil {
		t.Fatal("position should exist")
	}
	if pos.NetVolume != 0.1 {
		t.Fatalf("want 0.1, got %f", pos.NetVolume)
	}

	// Sell 0.03 EURUSD.
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 2,
		Canonical: "EURUSD", Side: "SELL", Volume: 0.03, Price: 1.0860,
		ToState: "FILLED", Timestamp: time.Now(),
	})

	pos = c.GetPosition("acc-1", "EURUSD")
	if pos.NetVolume != 0.07 {
		t.Fatalf("want 0.07 after sell, got %f", pos.NetVolume)
	}
}

func TestStateCache_NonFillEventsDontUpdatePosition(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())

	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderCreated, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "BUY", Volume: 0.1, Price: 1.0850,
		ToState: "SUBMITTED", Timestamp: time.Now(),
	})

	pos := c.GetPosition("acc-1", "EURUSD")
	if pos != nil {
		t.Fatal("non-fill event should not create position")
	}
}

func TestStateCache_Stats(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())

	c.ApplyEvent(&TradeEvent{EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1, Canonical: "EURUSD", Side: "BUY", Volume: 0.1, ToState: "FILLED", Timestamp: time.Now()})
	c.ApplyEvent(&TradeEvent{EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 2, Canonical: "GBPUSD", Side: "SELL", Volume: 0.2, ToState: "FILLED", Timestamp: time.Now()})

	orders, positions := c.Stats()
	if orders != 2 {
		t.Fatalf("want 2 orders, got %d", orders)
	}
	if positions != 2 {
		t.Fatalf("want 2 positions, got %d", positions)
	}
}

func TestStateCache_LoadFromRedis_NoRedis(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())
	err := c.LoadFromRedis(context.Background())
	if err != nil {
		t.Fatalf("LoadFromRedis with nil redis should succeed: %v", err)
	}
}

func TestStateCache_PositionKey(t *testing.T) {
	t.Parallel()
	key := positionKey("acc-1", "EURUSD")
	if key != "acc-1:EURUSD" {
		t.Fatalf("unexpected position key: %s", key)
	}
}

func TestStateCache_GetPositionsByAccount(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())

	c.ApplyEvent(&TradeEvent{EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1, Canonical: "EURUSD", Side: "BUY", Volume: 0.1, ToState: "FILLED", Timestamp: time.Now()})
	c.ApplyEvent(&TradeEvent{EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 2, Canonical: "GBPUSD", Side: "BUY", Volume: 0.2, ToState: "FILLED", Timestamp: time.Now()})
	c.ApplyEvent(&TradeEvent{EventType: TradeEventOrderFilled, AccountID: "acc-2", Ticket: 3, Canonical: "USDJPY", Side: "SELL", Volume: 0.3, ToState: "FILLED", Timestamp: time.Now()})

	positions := c.GetPositionsByAccount("acc-1")
	if len(positions) != 2 {
		t.Fatalf("want 2 positions for acc-1, got %d", len(positions))
	}
}
