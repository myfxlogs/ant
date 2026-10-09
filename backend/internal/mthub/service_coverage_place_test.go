package mthub

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/costsvc"
	"alphaforge/internal/risk"
)

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

// --- SessionState coverage ---

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

// --- HubCostEstimator Refresh ---

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

// --- StateCache: position increase weighted average ---

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

// --- Pure function: isValidOMSTransition ---
