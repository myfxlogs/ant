package mthub

import (
	"context"
	"errors"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/costsvc"
	"alphaforge/internal/risk"
)

// --- CloseOrder with logger + executor error ---

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

func TestPlaceOrder_WithCostEstimatorAndEventStore(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.eventStore = NewTradeEventStore(nil)
	record, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.Ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", record.Ticket)
	}
}

// --- Logger paths in DeleteOrder ---

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

func TestPlaceOrder_WithHubCostEstimator(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(_ context.Context, _ []string) ([]*SymbolParam, error) {
			return []*SymbolParam{
				{Canonical: "EURUSD", Digits: 5, PointValue: dec(1)},
			}, nil
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetCostEstimator(NewHubCostEstimator(svc.hub, &costsvc.CostModel{
		Symbol: "DEFAULT", SpreadPips: dec(2),
	}, nil))
	svc.eventStore = NewTradeEventStore(nil)
	record, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.Ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", record.Ticket)
	}
}

// --- PlaceOrder with guard (covers preTradeChecks guard path) ---

func TestPlaceOrder_WithGuard(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetGuard(risk.NewGuard(&risk.GuardConfig{MaxLotSize: dec(100)}))
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlaceOrder_GuardRejected(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetGuard(risk.NewGuard(&risk.GuardConfig{MaxLotSize: dec(0.05)}))
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err == nil {
		t.Fatal("expected guard rejection")
	}
}

// --- PlaceOrder with reconcileGate ---

func TestPlaceOrder_ReconcileGateBlocked(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.reconcileGate = NewReconcileGate()
	svc.reconcileGate.EnterReconciling("acc-1")
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err == nil {
		t.Fatal("expected reconcile gate error")
	}
}

// --- CloseOrder with reconcileGate blocked ---

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

func TestPlaceOrder_RateLimited(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetUserLimiter(newSaturatedLimiter())
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	_, err := svc.PlaceOrder(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

// --- PlaceOrder with guard and sell side ---

func TestPlaceOrder_WithGuard_SellSide(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetGuard(risk.NewGuard(&risk.GuardConfig{MaxLotSize: dec(100)}))
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideSell, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ModifyOrder with logger (covers logger paths) ---

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

func TestPlaceOrder_WithCostEstimate_EventStore(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(_ context.Context, _ []string) ([]*SymbolParam, error) {
			return []*SymbolParam{
				{Canonical: "EURUSD", Digits: 5, PointValue: dec(1)},
			}, nil
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetCostEstimator(NewHubCostEstimator(svc.hub, &costsvc.CostModel{
		Symbol: "DEFAULT", SpreadPips: dec(2), PipSize: dec(0.0001), PipValue: dec(10),
	}, nil))
	svc.eventStore = NewTradeEventStore(nil)
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- PlaceOrder with kill switch ---

func TestPlaceOrder_KillSwitchEngaged(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if !errors.Is(err, ErrKillSwitchEngaged) {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

// --- CloseOrder with kill switch ---

func TestCloseOrder_KillSwitchEngaged(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if !errors.Is(err, ErrKillSwitchEngaged) {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

// --- ModifyOrder with kill switch ---

func TestModifyOrder_KillSwitchEngaged(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if !errors.Is(err, ErrKillSwitchEngaged) {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

// --- DeleteOrder with kill switch ---

func TestDeleteOrder_KillSwitchEngaged(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	err := svc.DeleteOrder(context.Background(), "acc-1", 123)
	if !errors.Is(err, ErrKillSwitchEngaged) {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

// --- PlaceOrder with account owner verifier ---

func TestPlaceOrder_OwnershipDenied(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return false, nil
	})
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	_, err := svc.PlaceOrder(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err == nil {
		t.Fatal("expected ownership error")
	}
}

func TestPlaceOrder_OwnershipCheckError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return false, ErrSessionNotFound
	})
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	_, err := svc.PlaceOrder(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err == nil {
		t.Fatal("expected ownership check error")
	}
}

func TestPlaceOrder_OwnershipUnauthenticated(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return true, nil
	})
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err == nil {
		t.Fatal("expected unauthenticated error")
	}
}

// --- Idempotency guard error paths (using failing Redis client) ---

func newFailingRedisClient() *goredis.Client {
	return goredis.NewClient(&goredis.Options{
		Addr:        "localhost:0", // invalid port — connection refused
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
}

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
