package mthub

import (
	"testing"
	"time"
)

func TestSanitizeUTF8_Valid(t *testing.T) {
	t.Parallel()
	s := "hello world"
	if got := sanitizeUTF8(s); got != s {
		t.Fatalf("expected %s, got %s", s, got)
	}
}

func TestSanitizeUTF8_Invalid(t *testing.T) {
	t.Parallel()
	s := "hello\xff\xfeworld"
	got := sanitizeUTF8(s)
	if got == s {
		t.Fatal("expected sanitized output to differ")
	}
}

func TestEventToPayload(t *testing.T) {
	t.Parallel()
	ev := &TradeEvent{
		EventID:   "evt-1",
		EventType: TradeEventOrderCreated,
		AccountID: "acc-1",
		UserID:    "user-1",
		Broker:    "mt5",
		Ticket:    123,
		ClientID:  "client-1",
		Canonical: "EURUSD",
		Side:      "buy",
		OrderType: "market",
		Volume:    dec(0.1),
		Price:     dec(1.085),
		StopLoss:  dec(1.08),
		TakeProfit: dec(1.09),
		FromState: "NEW",
		ToState:   "SUBMITTED",
		Timestamp: time.Now(),
	}
	payload := eventToPayload(ev)
	if payload.EventId != "evt-1" {
		t.Fatalf("expected evt-1, got %s", payload.EventId)
	}
	if payload.AccountId != "acc-1" {
		t.Fatalf("expected acc-1, got %s", payload.AccountId)
	}
	if payload.Ticket != 123 {
		t.Fatalf("expected 123, got %d", payload.Ticket)
	}
	if payload.Volume != "0.1" {
		t.Fatalf("expected 0.1, got %s", payload.Volume)
	}
}

func TestOrderToCacheProto(t *testing.T) {
	t.Parallel()
	entry := &OrderStateCacheEntry{
		Ticket:    456,
		AccountID: "acc-1",
		State:     "WORKING",
		Canonical: "EURUSD",
		Side:      "buy",
		Volume:    dec(0.2),
		Price:     dec(1.085),
		UpdatedAt: time.Now(),
	}
	proto := orderToCacheProto(entry)
	if proto.Ticket != 456 {
		t.Fatalf("expected 456, got %d", proto.Ticket)
	}
	if proto.AccountId != "acc-1" {
		t.Fatalf("expected acc-1, got %s", proto.AccountId)
	}
	if proto.State != "WORKING" {
		t.Fatalf("expected WORKING, got %s", proto.State)
	}
	if proto.Volume != "0.2" {
		t.Fatalf("expected 0.2, got %s", proto.Volume)
	}
}

func TestNewIdempotencyGuard(t *testing.T) {
	t.Parallel()
	g := NewIdempotencyGuard(nil)
	if g == nil {
		t.Fatal("expected non-nil guard")
	}
}
