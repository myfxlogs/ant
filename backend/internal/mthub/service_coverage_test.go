package mthub

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/interceptor"
	"alphaforge/internal/risk"
	"alphaforge/internal/usermgr"
)

// --- mockMarginExecutor implements both OrderExecutor and MarginRequirer ---

type mockMarginExecutor struct {
	mockExecutor
	marginRequired decimal.Decimal
	marginErr      error
}

func (m *mockMarginExecutor) RequiredMargin(_ context.Context, _ string, _ decimal.Decimal, _ Side, _ decimal.Decimal) (decimal.Decimal, error) {
	if m.marginErr != nil {
		return decimal.Zero, m.marginErr
	}
	return m.marginRequired, nil
}

// --- helper to create context with user ID ---

func ctxWithUser(userID string) context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey, userID)
}

// --- CloseOrder coverage ---

func TestCloseOrder_ReconcileGate(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	gate := NewReconcileGate()
	gate.EnterReconciling("acc-1")
	svc.reconcileGate = gate
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected reconcile gate rejection on close")
	}
}

func TestCloseOrder_RateLimiter(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.SetUserLimiter(newSaturatedLimiter())
	err := svc.CloseOrder(ctxWithUser("user-1"), "acc-1", 123, decimal.NewFromInt(1))
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestCloseOrder_GateRejection(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	gate := risk.NewGate()
	gate.SetKillSwitch(func() bool { return true })
	svc.SetGate(gate)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: dec(10000), Equity: dec(10000)}, nil
	})
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected gate rejection on close")
	}
}

func TestCloseOrder_WithEventStore(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.eventStore = NewTradeEventStore(nil)
	svc.SetLogger(zap.NewNop())
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCloseOrder_PreCloseChecks_OwnershipError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return false, context.DeadlineExceeded
	})
	err := svc.CloseOrder(ctxWithUser("user-1"), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected ownership check error")
	}
}

func TestCloseOrder_PreCloseChecks_NotOwned(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return false, nil
	})
	err := svc.CloseOrder(ctxWithUser("user-1"), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected account not owned error")
	}
}

// --- DeleteOrder coverage ---

func TestDeleteOrder_RateLimiter(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.SetUserLimiter(newSaturatedLimiter())
	err := svc.DeleteOrder(ctxWithUser("user-1"), "acc-1", 123)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestDeleteOrder_OwnershipError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return false, context.DeadlineExceeded
	})
	err := svc.DeleteOrder(ctxWithUser("user-1"), "acc-1", 123)
	if err == nil {
		t.Fatal("expected ownership check error")
	}
}

func TestDeleteOrder_OwnershipNotOwned(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return false, nil
	})
	err := svc.DeleteOrder(ctxWithUser("user-1"), "acc-1", 123)
	if err == nil {
		t.Fatal("expected account not owned error")
	}
}

// --- ModifyOrder coverage ---

func TestModifyOrder_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, dec(1.08), dec(1.09), decimal.Zero)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestModifyOrder_OwnershipError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return false, context.DeadlineExceeded
	})
	err := svc.ModifyOrder(ctxWithUser("user-1"), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err == nil {
		t.Fatal("expected ownership check error")
	}
}

func TestModifyOrder_OwnershipNotOwned(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return false, nil
	})
	err := svc.ModifyOrder(ctxWithUser("user-1"), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err == nil {
		t.Fatal("expected account not owned error")
	}
}

func TestModifyOrder_Unauthenticated(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return true, nil
	})
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err == nil {
		t.Fatal("expected unauthenticated error")
	}
}

func TestModifyOrder_ExecutorError_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockExecutor{
		platform:      "MT5",
		modifyOrderFn: func(_ context.Context, _ int64, _, _, _ decimal.Decimal) error { return ErrSessionNotFound },
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err == nil {
		t.Fatal("expected executor error")
	}
}

func TestModifyOrder_NoSession_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err == nil {
		t.Fatal("expected session not found error")
	}
}

// --- submitToBroker margin precheck coverage ---

func TestSubmitToBroker_MarginPrecheckPass(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockMarginExecutor{
		mockExecutor:   mockExecutor{platform: "MT5"},
		marginRequired: dec(100),
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{
			Balance:    dec(10000),
			Equity:     dec(10000),
			FreeMargin: dec(5000),
			UsedMargin: dec(100),
		}, nil
	})
	ticket, err := svc.submitToBroker(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	}, "ord-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", ticket)
	}
}

func TestSubmitToBroker_MarginPrecheckReject(t *testing.T) {
	t.Parallel()
	// D6-A: margin precheck is now handled by the Gate (MarginPreCheck rule),
	// not in submitToBroker. submitToBroker just resolves executor and places order.
	// Verify submitToBroker succeeds even with tight margin — Gate rejection is tested separately.
	svc := newTestService()
	exec := &mockMarginExecutor{
		mockExecutor:   mockExecutor{platform: "MT5"},
		marginRequired: dec(99999),
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{
			Balance:    dec(100),
			Equity:     dec(100),
			FreeMargin: dec(50),
			UsedMargin: dec(50),
		}, nil
	})
	ticket, err := svc.submitToBroker(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	}, "ord-1")
	if err != nil {
		t.Fatalf("submitToBroker should not do margin precheck (D6-A Gate handles it): %v", err)
	}
	if ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", ticket)
	}
}

func TestSubmitToBroker_MarginRPCError_Skips(t *testing.T) {
	t.Parallel()
	// D6-A: submitToBroker no longer calls RequiredMargin RPC — that was part of
	// the old risksvc.PreCheck path. submitToBroker just resolves executor and places order.
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockMarginExecutor{
		mockExecutor: mockExecutor{platform: "MT5"},
		marginErr:    context.DeadlineExceeded,
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: dec(10000), Equity: dec(10000)}, nil
	})
	ticket, err := svc.submitToBroker(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	}, "ord-1")
	if err != nil {
		t.Fatalf("expected success (no margin RPC in submitToBroker), got %v", err)
	}
	if ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", ticket)
	}
}

func TestSubmitToBroker_StateProviderError_Skips(t *testing.T) {
	t.Parallel()
	// D6-A: submitToBroker no longer fetches account state — that's done in evaluatePlaceGate.
	// submitToBroker just resolves executor and places order regardless of state provider.
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockMarginExecutor{
		mockExecutor:   mockExecutor{platform: "MT5"},
		marginRequired: dec(100),
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return nil, context.DeadlineExceeded
	})
	ticket, err := svc.submitToBroker(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	}, "ord-1")
	if err != nil {
		t.Fatalf("expected success (no state fetch in submitToBroker), got %v", err)
	}
	if ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", ticket)
	}
}

// --- PlaceOrder coverage ---

func TestPlaceOrder_GuardRejection(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetGuard(risk.NewGuard(&risk.GuardConfig{
		MaxLotSize: decimal.NewFromInt(1),
	}))
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: decimal.NewFromInt(10), Price: dec(1.085),
	})
	if err == nil {
		t.Fatal("expected guard rejection")
	}
}

func TestPlaceOrder_GateRejection(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	gate := risk.NewGate()
	gate.SetKillSwitch(func() bool { return true })
	svc.SetGate(gate)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: dec(10000), Equity: dec(10000)}, nil
	})
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err == nil {
		t.Fatal("expected gate rejection")
	}
}

func TestPlaceOrder_ReconcileGate(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	gate := NewReconcileGate()
	gate.EnterReconciling("acc-1")
	svc.reconcileGate = gate
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err == nil {
		t.Fatal("expected reconcile gate rejection")
	}
}

func TestPlaceOrder_RateLimiter(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetUserLimiter(newSaturatedLimiter())
	_, err := svc.PlaceOrder(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestPlaceOrder_OwnershipError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return false, context.DeadlineExceeded
	})
	_, err := svc.PlaceOrder(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err == nil {
		t.Fatal("expected ownership check error")
	}
}

func TestPlaceOrder_OwnershipNotOwned(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return false, nil
	})
	_, err := svc.PlaceOrder(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err == nil {
		t.Fatal("expected account not owned error")
	}
}

// --- saturatedLimiter helper: creates a UserLimiter pre-saturated for a user ---

func newSaturatedLimiter() *usermgr.UserLimiter {
	l := usermgr.NewUserLimiter(usermgr.Config{MaxEntries: 10, OrderPerUserMax: 1})
	l.AllowOrder("user-1") // exhaust the quota
	return l
}

// --- Broker drop path coverage ---

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

// --- Broker publish/subscribe coverage ---

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

func TestIdempotencyGuard_CheckAndSet_ErrorPath(t *testing.T) {
	t.Parallel()
	g := NewIdempotencyGuard(newFailingRedisClient())
	_, _, err := g.CheckAndSet(context.Background(), "acc-1", "client-1", 0)
	if err == nil {
		t.Fatal("expected error from failing Redis")
	}
}

func TestIdempotencyGuard_SetTicket_ErrorPath(t *testing.T) {
	t.Parallel()
	g := NewIdempotencyGuard(newFailingRedisClient())
	err := g.SetTicket(context.Background(), "acc-1", "client-1", 100)
	if err == nil {
		t.Fatal("expected error from failing Redis")
	}
}

func TestIdempotencyGuard_DeleteKey_ErrorPath(t *testing.T) {
	t.Parallel()
	g := NewIdempotencyGuard(newFailingRedisClient())
	// DeleteKey doesn't return an error — it should not panic.
	g.DeleteKey(context.Background(), "acc-1", "client-1")
}
