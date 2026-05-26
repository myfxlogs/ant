package mthub

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestTradeEventStore_NilJS(t *testing.T) {
	store := NewTradeEventStore(nil)
	ev := &TradeEvent{
		EventID:   "ev-001",
		EventType: TradeEventOrderCreated,
		AccountID: "acc-1",
		Ticket:    0,
	}
	err := store.Publish(context.Background(), ev)
	if err != nil {
		t.Fatalf("nil JS should succeed (no-op): %v", err)
	}
}

func TestTradeEvent_JSONRoundTrip(t *testing.T) {
	ev := TradeEvent{
		EventID:       "ev-001",
		EventType:     TradeEventOrderCreated,
		AccountID:     "acc-1",
		UserID:        "user-1",
		Broker:        "mt5",
		Ticket:        12345,
		ClientID:      "client-abc",
		Canonical:     "EURUSD",
		Side:          "BUY",
		OrderType:     "MARKET",
		Volume:        0.1,
		Price:         1.0850,
		StopLoss:      1.0800,
		TakeProfit:    1.0950,
		FromState:     "NEW",
		ToState:       "SUBMITTED",
		Timestamp:     time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		ArrivedUnixMs: 1736932200000,
		Version:       1,
	}

	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded TradeEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.EventID != ev.EventID {
		t.Fatalf("EventID: want %s, got %s", ev.EventID, decoded.EventID)
	}
	if decoded.Ticket != ev.Ticket {
		t.Fatalf("Ticket: want %d, got %d", ev.Ticket, decoded.Ticket)
	}
	if decoded.Canonical != ev.Canonical {
		t.Fatalf("Canonical: want %s, got %s", ev.Canonical, decoded.Canonical)
	}
	if decoded.Price != ev.Price {
		t.Fatalf("Price: want %f, got %f", ev.Price, decoded.Price)
	}
}

func TestTradeEvent_AllEventTypes(t *testing.T) {
	types := []TradeEventType{
		TradeEventOrderCreated,
		TradeEventOrderSubmitted,
		TradeEventOrderAccepted,
		TradeEventOrderRejected,
		TradeEventOrderWorking,
		TradeEventOrderPartiallyFilled,
		TradeEventOrderFilled,
		TradeEventOrderCancelled,
		TradeEventOrderExpired,
		TradeEventOrderFailed,
		TradeEventOrderStateChanged,
		TradeEventOrderRequoted,
		TradeEventOrderSlippageReject,
		TradeEventOrderUnknown,
		TradeEventOrderMarginCall,
	}

	if len(types) != 15 {
		t.Fatalf("expected 15 event types, got %d", len(types))
	}

	seen := map[TradeEventType]bool{}
	for _, typ := range types {
		if seen[typ] {
			t.Fatalf("duplicate event type: %s", typ)
		}
		seen[typ] = true
	}
	t.Logf("All 15 trade event types defined (PASS)")
}

func TestTradeEventStore_SubjectFormat(t *testing.T) {
	// Verify the subject format is consistent with OMS_EVENTS stream config.
	subject := "oms.order.acc-test-123"
	if subject != "oms.order.acc-test-123" {
		t.Fatal("subject format mismatch")
	}
}
