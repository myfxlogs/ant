package mthub

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

// Adversarial proof: PlaceOrder success → mthub_orders_placed_total{status="ok"} +1
func TestPlaceOrder_Metrics_OK(t *testing.T) {
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-metrics-ok", &Session{AccountID: "acc-metrics-ok", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)

	before := testutil.ToFloat64(OrdersPlacedTotal.WithLabelValues("MT5", orderStatusOK))
	req := &OrderRequest{
		AccountID: "acc-metrics-ok",
		Side:      SideBuy,
		OrderType: OrderMarket,
		Volume:    dec(0.1),
		Price:     dec(1.085),
	}
	_, err := svc.PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := testutil.ToFloat64(OrdersPlacedTotal.WithLabelValues("MT5", orderStatusOK))
	if after != before+1 {
		t.Errorf("expected ok counter +1: before=%v after=%v", before, after)
	}
}

// Adversarial proof: kill switch → mthub_orders_placed_total{status="rejected"} +1
func TestPlaceOrder_Metrics_Rejected(t *testing.T) {
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})

	before := testutil.ToFloat64(OrdersPlacedTotal.WithLabelValues("", orderStatusRejected))
	req := &OrderRequest{
		AccountID: "acc-metrics-rej",
		Side:      SideBuy,
		OrderType: OrderMarket,
		Volume:    dec(0.1),
		Price:     dec(1.085),
	}
	_, err := svc.PlaceOrder(context.Background(), req)
	if err == nil {
		t.Fatal("expected kill switch error")
	}
	after := testutil.ToFloat64(OrdersPlacedTotal.WithLabelValues("", orderStatusRejected))
	if after != before+1 {
		t.Errorf("expected rejected counter +1: before=%v after=%v", before, after)
	}
}

// Adversarial proof: broker error → mthub_orders_placed_total{status="err"} +1
func TestPlaceOrder_Metrics_Err(t *testing.T) {
	svc := newTestService()
	exec := &mockExecutor{
		platform: "MT4",
		placeOrderFn: func(ctx context.Context, req *OrderRequest) (int64, error) {
			return 0, ErrSessionNotFound
		},
	}
	svc.hub.Register("acc-metrics-err", &Session{AccountID: "acc-metrics-err", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)

	before := testutil.ToFloat64(OrdersPlacedTotal.WithLabelValues("MT4", orderStatusErr))
	req := &OrderRequest{
		AccountID: "acc-metrics-err",
		Side:      SideBuy,
		OrderType: OrderMarket,
		Volume:    dec(0.1),
		Price:     dec(1.085),
	}
	_, err := svc.PlaceOrder(context.Background(), req)
	if err == nil {
		t.Fatal("expected broker error")
	}
	after := testutil.ToFloat64(OrdersPlacedTotal.WithLabelValues("MT4", orderStatusErr))
	if after != before+1 {
		t.Errorf("expected err counter +1: before=%v after=%v", before, after)
	}
}

// Adversarial proof: session register → gauge=1, remove → gauge=0
func TestSessionActive_Metrics(t *testing.T) {
	hub := NewHub()
	exec := &mockExecutor{platform: "MT5"}
	hub.Register("acc-session-test", &Session{AccountID: "acc-session-test", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)

	val := testutil.ToFloat64(SessionActive.WithLabelValues("acc-session-test", "MT5"))
	if val != 1 {
		t.Errorf("expected session_active=1 after register, got %v", val)
	}

	hub.RemoveSession("acc-session-test")
	val = testutil.ToFloat64(SessionActive.WithLabelValues("acc-session-test", "MT5"))
	if val != 0 {
		t.Errorf("expected session_active=0 after remove, got %v", val)
	}
}

// Adversarial proof: event published → mthub_event_published_total{event_type="ORDER_CREATED"} +1
func TestEventPublished_Metrics(t *testing.T) {
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-event-test", &Session{AccountID: "acc-event-test", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)

	// Wire a trade event store so publishOrderCreatedEvent fires.
	svc.eventStore = NewTradeEventStore(nil)

	before := testutil.ToFloat64(EventPublishedTotal.WithLabelValues(string(TradeEventOrderCreated)))
	req := &OrderRequest{
		AccountID: "acc-event-test",
		Side:      SideBuy,
		OrderType: OrderMarket,
		Volume:    dec(0.1),
		Price:     dec(1.085),
	}
	_, err := svc.PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := testutil.ToFloat64(EventPublishedTotal.WithLabelValues(string(TradeEventOrderCreated)))
	if after != before+1 {
		t.Errorf("expected event_published +1: before=%v after=%v", before, after)
	}
}
