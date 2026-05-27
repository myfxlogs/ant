package system

import (
	"testing"
	"time"

	"anttrader/internal/mthub"
)

func TestToProtoOrderEvent_NilOrder(t *testing.T) {
	ev := &mthub.OrderEvent{
		AccountID: "acc-1",
		Ticket:    12345,
		EventType: "OPEN",
		Order:     nil, // nil Order should not panic
		Timestamp: time.Now(),
	}

	result := toProtoOrderEvent(ev)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Order == nil {
		t.Fatal("expected non-nil Order in proto (should be default-valued, not nil)")
	}
	if result.Order.Ticket != 0 {
		t.Errorf("expected Ticket=0 for nil Order, got %d", result.Order.Ticket)
	}
	if result.Ticket != 12345 {
		t.Errorf("expected Ticket=12345, got %d", result.Ticket)
	}
}

func TestToProtoOrderEvent_WithOrder(t *testing.T) {
	ev := &mthub.OrderEvent{
		AccountID: "acc-2",
		Ticket:    67890,
		EventType: "CLOSE",
		Order:     &mthub.OrderRecord{Ticket: 67890},
		Timestamp: time.Now(),
	}

	result := toProtoOrderEvent(ev)

	if result.Order == nil {
		t.Fatal("expected non-nil Order")
	}
	if result.Order.Ticket != 67890 {
		t.Errorf("expected Ticket=67890, got %d", result.Order.Ticket)
	}
}
