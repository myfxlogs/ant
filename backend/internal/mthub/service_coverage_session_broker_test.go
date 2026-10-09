package mthub

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func TestSessionState_WithExecutor(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	state := svc.SessionState(context.Background(), "acc-1")
	if state != "connected" {
		t.Fatalf("expected connected, got %s", state)
	}
}

// --- Platform with session ---

func TestPlatform_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	if svc.Platform("acc-1") != "MT5" {
		t.Fatal("expected MT5")
	}
}

func TestPlatformFunc_WithSession(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	exec := &mockExecutor{platform: "MT4"}
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	if platform("acc-1", hub) != "MT4" {
		t.Fatal("expected MT4")
	}
}

// --- OpenedOrders / OrderHistory / SymbolParams / PriceHistory / SymbolList / SubscribeSymbols with session ---

func TestOpenedOrders_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	orders, err := svc.OpenedOrders(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orders != nil {
		t.Fatalf("expected nil orders from mock, got %d", len(orders))
	}
}

func TestOrderHistory_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	orders, err := svc.OrderHistory(context.Background(), "acc-1", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orders != nil {
		t.Fatalf("expected nil orders from mock, got %d", len(orders))
	}
}

func TestSymbolParams_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(_ context.Context, _ []string) ([]*SymbolParam, error) {
			return nil, nil
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	params, err := svc.SymbolParams(context.Background(), "acc-1", []string{"EURUSD"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params != nil {
		t.Fatalf("expected nil params from mock, got %d", len(params))
	}
}

func TestPriceHistory_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	bars, err := svc.PriceHistory(context.Background(), "acc-1", "EURUSD", "M1", 0, 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bars != nil {
		t.Fatalf("expected nil bars from mock, got %d", len(bars))
	}
}

func TestSymbolList_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	symbols, err := svc.SymbolList(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if symbols != nil {
		t.Fatalf("expected nil symbols from mock, got %d", len(symbols))
	}
}

func TestSubscribeSymbols_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.SubscribeSymbols(context.Background(), "acc-1", []string{"EURUSD"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- postCloseSuccess with event store ---

func TestPostCloseSuccess_WithEventStore(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.eventStore = NewTradeEventStore(nil)
	svc.postCloseSuccess(context.Background(), "close-1", "acc-1", 123)
}

// --- postCloseFailure with logger ---

func TestPostCloseFailure_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.postCloseFailure(context.Background(), "close-1", "acc-1", 123, ErrSessionNotFound)
}

// --- publishOrderCreatedEvent with event store ---

func TestPublishOrderCreatedEvent_WithEventStore(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.eventStore = NewTradeEventStore(nil)
	svc.publishOrderCreatedEvent(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	}, 12345, nil)
}

// --- publishOrderCreatedEvent with logger and event store ---

func TestPublishOrderCreatedEvent_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.eventStore = NewTradeEventStore(nil)
	svc.publishOrderCreatedEvent(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	}, 12345, &antv1.CostEstimate{})
}

// --- omsTransition with logger (nil writer still) ---

func TestTickBroker_Publish_DropPath(t *testing.T) {
	t.Parallel()
	b := NewTickBroker(1, zap.NewNop())
	ch, cancel := b.Subscribe("acc-1")
	// Fill buffer (size 1) with first publish, second publish drops.
	b.Publish(&TickUpdate{AccountID: "acc-1", Symbol: "EURUSD", Bid: dec(1.08), Ask: dec(1.09)})
	b.Publish(&TickUpdate{AccountID: "acc-1", Symbol: "EURUSD", Bid: dec(1.08), Ask: dec(1.09)})
	cancel()
	_ = ch
}

func TestTradeBroker_Publish_DropPath(t *testing.T) {
	t.Parallel()
	b := NewTradeBroker(1, zap.NewNop())
	ch, cancel := b.Subscribe("acc-1")
	// Fill buffer (size 1) with first publish, second publish drops.
	b.Publish(&BrokerTradeEvent{AccountID: "acc-1", Ticket: 123})
	b.Publish(&BrokerTradeEvent{AccountID: "acc-1", Ticket: 456})
	cancel()
	_ = ch
}

// --- Hub method coverage ---

// --- Pure function coverage ---

func TestPositionSnapshotBroker_PublishSubscribe(t *testing.T) {
	t.Parallel()
	b := NewPositionSnapshotBroker()
	ch, cancel := b.Subscribe("acc-1")
	defer cancel()
	b.Publish(&PositionSnapshot{AccountID: "acc-1"})
	select {
	case ev := <-ch:
		if ev.AccountID != "acc-1" {
			t.Fatalf("expected acc-1, got %s", ev.AccountID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for snapshot")
	}
}

func TestAccountStatusBroker_PublishSubscribe(t *testing.T) {
	t.Parallel()
	b := NewAccountStatusBroker()
	ch, cancel := b.Subscribe("acc-1")
	defer cancel()
	b.Publish(&AccountStatusEvent{AccountID: "acc-1", Status: "connected"})
	select {
	case ev := <-ch:
		if ev.Status != "connected" {
			t.Fatalf("expected connected, got %s", ev.Status)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for status event")
	}
}

func TestAccountProfitBroker_PublishSubscribe(t *testing.T) {
	t.Parallel()
	b := NewAccountProfitBroker()
	ch, cancel := b.Subscribe("acc-1")
	defer cancel()
	b.Publish(&AccountProfitEvent{AccountID: "acc-1"})
	select {
	case ev := <-ch:
		if ev.AccountID != "acc-1" {
			t.Fatalf("expected acc-1, got %s", ev.AccountID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for profit event")
	}
}

func TestOrderEventBroker_PublishSubscribe(t *testing.T) {
	t.Parallel()
	b := NewOrderEventBroker()
	ch, cancel := b.Subscribe("user-1")
	defer cancel()
	b.PublishEvent("user-1", &OrderEvent{AccountID: "acc-1"})
	select {
	case ev := <-ch:
		if ev.AccountID != "acc-1" {
			t.Fatalf("expected acc-1, got %s", ev.AccountID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for order event")
	}
}

// --- CloseOrder with logger + executor error ---

func TestSubscribeAccountStatus_NoBroker(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	ch, cancel := svc.SubscribeAccountStatus("acc-1")
	defer cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel should be closed when no broker")
		}
	default:
		t.Fatal("channel should be closed when no broker")
	}
}

// --- SubscribeUserOrderEvents ---

func TestSubscribeUserOrderEvents(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	ch, cancel := svc.SubscribeUserOrderEvents(context.Background(), "user-1")
	defer cancel()
	_ = ch
}

// --- PublishPositionSnapshot ---

func TestPublishPositionSnapshot(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.PublishPositionSnapshot(&PositionSnapshot{AccountID: "acc-1"})
}

// --- SubscribePositionSnapshots ---

func TestSubscribePositionSnapshots(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	ch, cancel := svc.SubscribePositionSnapshots(context.Background(), "acc-1")
	defer cancel()
	_ = ch
}

// --- PublishAccountProfit ---

func TestPublishAccountProfit(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.PublishAccountProfit(&AccountProfitEvent{AccountID: "acc-1"})
}

// --- SubscribeAccountProfit ---

func TestSubscribeAccountProfit(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	ch, cancel := svc.SubscribeAccountProfit(context.Background(), "acc-1")
	defer cancel()
	_ = ch
}

// --- HubCostEstimator cache hit (second call hits cache) ---

func TestPositionSnapshotBroker_Publish_DropPath(t *testing.T) {
	t.Parallel()
	b := NewPositionSnapshotBroker()
	ch, cancel := b.Subscribe("acc-1")
	// Buffer size is 8 — fill it.
	for i := 0; i < 8; i++ {
		b.Publish(&PositionSnapshot{AccountID: "acc-1"})
	}
	// This publish should hit the default (drop) case.
	b.Publish(&PositionSnapshot{AccountID: "acc-1"})
	cancel()
	_ = ch
}

func TestAccountStatusBroker_Publish_DropPath(t *testing.T) {
	t.Parallel()
	b := NewAccountStatusBroker()
	ch, cancel := b.Subscribe("acc-1")
	// Buffer size is 8 — fill it.
	for i := 0; i < 8; i++ {
		b.Publish(&AccountStatusEvent{AccountID: "acc-1"})
	}
	// This publish should hit the default (drop) case.
	b.Publish(&AccountStatusEvent{AccountID: "acc-1"})
	cancel()
	_ = ch
}

func TestOrderEventBroker_PublishEvent_DropPath(t *testing.T) {
	t.Parallel()
	b := NewOrderEventBroker()
	ch, cancel := b.Subscribe("user-1")
	// Buffer size is 64 — fill it.
	for i := 0; i < 64; i++ {
		b.PublishEvent("user-1", &OrderEvent{AccountID: "acc-1"})
	}
	// This publish should hit the default (drop) case.
	b.PublishEvent("user-1", &OrderEvent{AccountID: "acc-1"})
	cancel()
	_ = ch
}

func TestAccountProfitBroker_Publish_DropPath(t *testing.T) {
	t.Parallel()
	b := NewAccountProfitBroker()
	ch, cancel := b.Subscribe("acc-1")
	// Buffer size is 64 — fill it.
	for i := 0; i < 64; i++ {
		b.Publish(&AccountProfitEvent{AccountID: "acc-1"})
	}
	// This publish should hit the default (drop) case.
	b.Publish(&AccountProfitEvent{AccountID: "acc-1"})
	cancel()
	_ = ch
}

// --- HubCostEstimator: double-checked locking cache hit ---
