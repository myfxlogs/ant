package mthub

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

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

func TestModifyOrder_Success_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, dec(1.08), dec(1.10), decimal.Zero)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestModifyOrder_NoSession_LoggerPath(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err == nil {
		t.Fatal("expected session not found error")
	}
}

func TestModifyOrder_ExecutorError_LoggerPath(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockExecutor{
		platform: "MT5",
		modifyOrderFn: func(_ context.Context, _ int64, _, _, _ decimal.Decimal) error {
			return ErrSessionNotFound
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err == nil {
		t.Fatal("expected executor error")
	}
}

// --- CloseOrder executor error with logger and event store ---
