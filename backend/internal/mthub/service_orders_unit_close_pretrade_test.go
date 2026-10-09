package mthub

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestPreTradeChecks_KillSwitch(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	err := svc.preTradeChecks(context.Background(), &OrderRequest{AccountID: "acc-1"})
	if err == nil || err.Error() != ErrKillSwitchEngaged.Error() {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

func TestPreTradeChecks_NilDeps(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	err := svc.preTradeChecks(context.Background(), &OrderRequest{AccountID: "acc-1", Side: SideBuy})
	if err != nil {
		t.Fatalf("unexpected error with nil deps: %v", err)
	}
}

func TestPreCloseChecks_KillSwitch(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	err := svc.preCloseChecks(context.Background(), "acc-1")
	if err == nil || err.Error() != ErrKillSwitchEngaged.Error() {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

func TestPreCloseChecks_NilDeps(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	err := svc.preCloseChecks(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error with nil deps: %v", err)
	}
}

func TestPreCloseChecks_NoUserID(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetAccountOwnerVerifier(func(ctx context.Context, userID, accountID string) (bool, error) {
		return true, nil
	})
	err := svc.preCloseChecks(context.Background(), "acc-1")
	if err == nil {
		t.Fatal("expected unauthenticated error")
	}
}

func TestCloseOrder_KillSwitch(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil || err.Error() != ErrKillSwitchEngaged.Error() {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

func TestCloseOrder_NoSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected error for no session")
	}
}

func TestCloseOrder_Success(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCloseOrder_ExecutorError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{
		platform:     "MT5",
		closeOrderFn: func(ctx context.Context, ticket int64, lots decimal.Decimal) error { return ErrSessionNotFound },
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected executor error")
	}
}

func TestCloseOrder_OwnershipCheck(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.SetAccountOwnerVerifier(func(ctx context.Context, userID, accountID string) (bool, error) {
		return false, nil
	})
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected error for ownership check failure")
	}
}

func TestPreTradeChecks_WithRateLimit(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	// Note: UserLimiter requires a real usermgr context with user ID.
	// Without it, the rate limit path is skipped (uid == "").
	// This test verifies the nil-path is safe.
	svc.SetUserLimiter(nil) // nil limiter → no rate limiting
	err := svc.preTradeChecks(context.Background(), &OrderRequest{AccountID: "acc-1", Side: SideBuy})
	if err != nil {
		t.Fatalf("unexpected error with nil deps: %v", err)
	}
}
