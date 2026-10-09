package mthub

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/risk"
)

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

func TestCloseOrder_ExecutorError_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockExecutor{
		platform:     "MT5",
		closeOrderFn: func(_ context.Context, _ int64, _ decimal.Decimal) error { return ErrSessionNotFound },
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected executor error")
	}
}

// --- DeleteOrder with kill switch + logger ---

func TestDeleteOrder_KillSwitch_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	err := svc.DeleteOrder(context.Background(), "acc-1", 123)
	if !errors.Is(err, ErrKillSwitchEngaged) {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

// --- PlaceOrder with cost estimator + event store ---

func TestDeleteOrder_Success_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.DeleteOrder(context.Background(), "acc-1", 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteOrder_ExecutorError_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockExecutor{
		platform:      "MT5",
		deleteOrderFn: func(_ context.Context, _ int64) error { return ErrSessionNotFound },
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	err := svc.DeleteOrder(context.Background(), "acc-1", 123)
	if err == nil {
		t.Fatal("expected executor error")
	}
}

func TestDeleteOrder_NoSession_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	err := svc.DeleteOrder(context.Background(), "acc-1", 123)
	if err == nil {
		t.Fatal("expected session not found error")
	}
}

// --- Logger paths in CloseOrder ---

func TestCloseOrder_Success_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCloseOrder_NoSession_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected session not found error")
	}
}

// --- PlaceOrder with cost estimator (HubCostEstimator) ---

func TestCloseOrder_ReconcileGateBlocked(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.reconcileGate = NewReconcileGate()
	svc.reconcileGate.EnterReconciling("acc-1")
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected reconcile gate error")
	}
}

// --- CloseOrder with event store (covers postCloseSuccess event path) ---

func TestCloseOrder_WithEventStore_Logger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.eventStore = NewTradeEventStore(nil)
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- CloseOrder with rate limiter ---

func TestCloseOrder_RateLimited(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetUserLimiter(newSaturatedLimiter())
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.CloseOrder(ctxWithUser("user-1"), "acc-1", 123, decimal.NewFromInt(1))
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

// --- DeleteOrder with rate limiter ---

func TestDeleteOrder_RateLimited(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetUserLimiter(newSaturatedLimiter())
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.DeleteOrder(ctxWithUser("user-1"), "acc-1", 123)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

// --- PlaceOrder with rate limiter ---

func TestCloseOrder_ExecutorError_WithEventStore(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.eventStore = NewTradeEventStore(nil)
	exec := &mockExecutor{
		platform:     "MT5",
		closeOrderFn: func(_ context.Context, _ int64, _ decimal.Decimal) error { return ErrSessionNotFound },
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected executor error")
	}
}

// --- PlaceOrder with cost estimate (covers costToProto in publishOrderCreatedEvent) ---
