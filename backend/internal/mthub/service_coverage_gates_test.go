package mthub

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/risk"
)

func TestPreTradeChecks_GuardRejection(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetGuard(risk.NewGuard(&risk.GuardConfig{
		MaxLotSize: decimal.NewFromInt(1),
	}))
	err := svc.preTradeChecks(context.Background(), &OrderRequest{
		AccountID: "acc-1",
		Side:      SideBuy,
		Volume:    decimal.NewFromInt(10),
	})
	if err == nil {
		t.Fatal("expected guard rejection for volume > max lot size")
	}
}

func TestPreTradeChecks_ReconcileGate(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	gate := NewReconcileGate()
	gate.EnterReconciling("acc-1")
	svc.reconcileGate = gate
	err := svc.preTradeChecks(context.Background(), &OrderRequest{
		AccountID: "acc-1",
		Side:      SideBuy,
	})
	if err == nil {
		t.Fatal("expected reconcile gate rejection")
	}
}

func TestPreTradeChecks_RateLimiter(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetUserLimiter(newSaturatedLimiter())
	err := svc.preTradeChecks(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1",
		Side:      SideBuy,
	})
	if err == nil || !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestPreTradeChecks_OwnershipCheck_Error(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return false, context.DeadlineExceeded
	})
	err := svc.preTradeChecks(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1",
		Side:      SideBuy,
	})
	if err == nil {
		t.Fatal("expected ownership check error")
	}
}

func TestPreTradeChecks_OwnershipNotOwned(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetAccountOwnerVerifier(func(_ context.Context, _, _ string) (bool, error) {
		return false, nil
	})
	err := svc.preTradeChecks(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1",
		Side:      SideBuy,
	})
	if err == nil {
		t.Fatal("expected account not owned error")
	}
}

// --- evaluatePlaceGate coverage ---

func TestEvaluatePlaceGate_Rejection(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	gate := risk.NewGate()
	gate.SetKillSwitch(func() bool { return true })
	svc.SetGate(gate)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: dec(10000), Equity: dec(10000)}, nil
	})
	err := svc.evaluatePlaceGate(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1",
		Side:      SideBuy,
	}, "ord-1")
	if err == nil {
		t.Fatal("expected gate rejection from kill switch")
	}
}

func TestEvaluatePlaceGate_StateProviderError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	gate := risk.NewGate()
	gate.SetKillSwitch(func() bool { return true })
	svc.SetGate(gate)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return nil, context.DeadlineExceeded
	})
	err := svc.evaluatePlaceGate(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1",
		Side:      SideBuy,
	}, "ord-1")
	if err == nil {
		t.Fatal("expected gate rejection even with state provider error (fail-closed)")
	}
}

func TestEvaluatePlaceGate_Allow(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	gate := risk.NewGate() // no rules, no kill switch → always allows
	svc.SetGate(gate)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: dec(10000), Equity: dec(10000)}, nil
	})
	err := svc.evaluatePlaceGate(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1",
		Side:      SideBuy,
	}, "ord-1")
	if err != nil {
		t.Fatalf("expected nil error from empty gate, got %v", err)
	}
}

func TestEvaluatePlaceGate_UsesBrokerRequiredMargin(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockMarginExecutor{
		mockExecutor:   mockExecutor{platform: "MT5"},
		marginRequired: dec(40),
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	gate := risk.NewGate()
	gate.AddRule(&risk.MarginPreCheck{MaxMarginRatio: dec(0.80)})
	svc.SetGate(gate)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{
			Balance: dec(100), Equity: dec(100), FreeMargin: dec(50), UsedMargin: dec(50),
			SymbolLeverage: 100, ContractSize: dec(1),
		}, nil
	})
	err := svc.evaluatePlaceGate(ctxWithUser("user-1"), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD", Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(1), Price: dec(1.1),
	}, "ord-1")
	if err == nil {
		t.Fatal("broker required margin of 40 plus used margin 50 must exceed the 80% gate")
	}
}

// --- evaluateCloseGate coverage ---

func TestEvaluateCloseGate_Rejection(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	gate := risk.NewGate()
	gate.SetKillSwitch(func() bool { return true })
	svc.SetGate(gate)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: dec(10000), Equity: dec(10000)}, nil
	})
	err := svc.evaluateCloseGate(ctxWithUser("user-1"), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected gate rejection from kill switch on close")
	}
}

func TestEvaluateCloseGate_StateProviderError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	gate := risk.NewGate()
	gate.SetKillSwitch(func() bool { return true })
	svc.SetGate(gate)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return nil, context.DeadlineExceeded
	})
	err := svc.evaluateCloseGate(ctxWithUser("user-1"), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected gate rejection on close with state error (fail-closed)")
	}
}

func TestEvaluateCloseGate_Allow(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	gate := risk.NewGate()
	svc.SetGate(gate)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: dec(10000), Equity: dec(10000)}, nil
	})
	err := svc.evaluateCloseGate(ctxWithUser("user-1"), "acc-1", 123, decimal.NewFromInt(1))
	if err != nil {
		t.Fatalf("expected nil error from empty gate on close, got %v", err)
	}
}

// --- CloseOrder coverage ---

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

// --- Broker drop paths (fill buffer then publish to hit default case) ---

// DEPLOY-LIVE-4: gate=nil → PlaceOrder must fail-closed (return error, not pass through).
func TestEvaluatePlaceGate_NilGate_FailClosed(t *testing.T) {
	t.Parallel()
	svc := newTestServiceNoGate()
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, &mockExecutor{platform: "MT5"})
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err == nil || !strings.Contains(err.Error(), "gate not configured") {
		t.Fatalf("expected 'gate not configured' fail-closed error, got %v", err)
	}
}

// DEPLOY-LIVE-4: accountStateProvider=nil → PlaceOrder must fail-closed.

// DEPLOY-LIVE-4: accountStateProvider=nil → PlaceOrder must fail-closed.
func TestEvaluatePlaceGate_NilStateProvider_FailClosed(t *testing.T) {
	t.Parallel()
	svc := newTestServiceNoGate()
	svc.SetGate(risk.NewDefaultGate())
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, &mockExecutor{platform: "MT5"})
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err == nil {
		t.Fatal("expected fail-closed error when accountStateProvider is nil, got nil")
	}
}

// DEPLOY-LIVE-4: gate=nil → CloseOrder must fail-closed.

// DEPLOY-LIVE-4: gate=nil → CloseOrder must fail-closed.
func TestEvaluateCloseGate_NilGate_FailClosed(t *testing.T) {
	t.Parallel()
	svc := newTestServiceNoGate()
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, &mockExecutor{platform: "MT5"})
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil || !strings.Contains(err.Error(), "gate not configured") {
		t.Fatalf("expected 'gate not configured' fail-closed error, got %v", err)
	}
}

// DEPLOY-LIVE-4: accountStateProvider=nil → CloseOrder must fail-closed.

// DEPLOY-LIVE-4: accountStateProvider=nil → CloseOrder must fail-closed.
func TestEvaluateCloseGate_NilStateProvider_FailClosed(t *testing.T) {
	t.Parallel()
	svc := newTestServiceNoGate()
	svc.SetGate(risk.NewDefaultGate())
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, &mockExecutor{platform: "MT5"})
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err == nil {
		t.Fatal("expected fail-closed error when accountStateProvider is nil on close, got nil")
	}
}
