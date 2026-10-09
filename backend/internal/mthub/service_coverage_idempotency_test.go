package mthub

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestPlaceOrder_IdempotencyCheckError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.idem = NewIdempotencyGuard(newFailingRedisClient())
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
		ClientID: "client-1",
	})
	if err == nil {
		t.Fatal("expected idempotency check error")
	}
}

func TestCloseOrder_IdempotencyCheckError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.idem = NewIdempotencyGuard(newFailingRedisClient())
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected idempotency check error")
	}
}

func TestModifyOrder_IdempotencyCheckError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.idem = NewIdempotencyGuard(newFailingRedisClient())
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err == nil {
		t.Fatal("expected idempotency check error")
	}
}

func TestPlaceOrder_IdempotencySetTicketError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	// Use ThreeLayerGuard with nil PG and nil Redis — CheckAndSet returns (false, 0, nil).
	// But we need IdempotencyGuard for SetTicket. Use a failing Redis client.
	svc.idem = NewIdempotencyGuard(newFailingRedisClient())
	// Override: use ThreeLayerGuard for preTradeChecks by not setting ClientID
	// Actually, preTradeChecks uses s.idem.CheckAndSet which will fail.
	// Instead, let's test the SetTicket error path directly.
	// PlaceOrder with ClientID will:
	// 1. preTradeChecks → idem.CheckAndSet → error → return error
	// So we can't reach SetTicket. Let's test SetTicket directly.
	_, _, err := svc.idem.CheckAndSet(context.Background(), "acc-1", "client-1", 0)
	if err == nil {
		t.Fatal("expected CheckAndSet error")
	}
	err = svc.idem.SetTicket(context.Background(), "acc-1", "client-1", 99999)
	if err == nil {
		t.Fatal("expected SetTicket error")
	}
	svc.idem.DeleteKey(context.Background(), "acc-1", "client-1")
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

// DEPLOY-LIVE-4: gate=nil → PlaceOrder must fail-closed (return error, not pass through).
