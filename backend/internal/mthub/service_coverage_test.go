package mthub

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/costsvc"
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

// --- preTradeChecks coverage ---

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
	if err == nil || err != ErrRateLimited {
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
	if err != ErrRateLimited {
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
	if err != ErrRateLimited {
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
	if err != ErrRateLimited {
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

func TestOmsTransition_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.omsTransition(context.Background(), "ord-1", "acc-1", OMSStateNew, OMSStateValidated)
}

// --- DerivedState recalculate with data ---

func TestDerivedState_Recalculate_WithData(t *testing.T) {
	t.Parallel()
	cache := NewStateCache(nil, testLogger())
	cache.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "BUY", Volume: dec(0.1), Price: dec(1.085),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	computer := NewDerivedComputer(cache, 1*time.Minute)
	computer.recalculate()
	ds := computer.State()
	acc := ds.GetAccount("acc-1")
	if acc == nil {
		t.Fatal("expected acc-1 derived state")
	}
	if !acc.Exposure.GreaterThan(decimal.Zero) {
		t.Fatal("expected positive exposure")
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

// --- Hub method coverage ---

// --- Pure function coverage ---

func TestIsTerminalOrderState(t *testing.T) {
	t.Parallel()
	terminals := []string{"CLOSED", "FILLED", "CANCELLED", "EXPIRED", "FAILED", "REJECTED"}
	for _, s := range terminals {
		if !isTerminalOrderState(s) {
			t.Errorf("%s should be terminal", s)
		}
	}
	nonTerminals := []string{"WORKING", "SUBMITTED", "NEW", "VALIDATED", "RISK_APPROVED"}
	for _, s := range nonTerminals {
		if isTerminalOrderState(s) {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestCostToProto(t *testing.T) {
	t.Parallel()
	est := costsvc.CostBreakdown{
		SpreadCost:   dec(1.5),
		Commission:   dec(7),
		SlippageCost: dec(0.5),
		SwapCost:     dec(0),
		TotalCost:    dec(9),
	}
	p := costToProto(&est)
	if p.SpreadCost != "1.5" {
		t.Fatalf("expected 1.5, got %s", p.SpreadCost)
	}
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
	if err != ErrKillSwitchEngaged {
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

// --- HubCostEstimator Refresh ---

func TestHubCostEstimator_RefreshCache(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	est := NewHubCostEstimator(hub, &costsvc.CostModel{Symbol: "DEFAULT"}, nil)
	est.Refresh("EURUSD")
}

// --- HubCostEstimator with active account but nil executor ---

func TestHubCostEstimator_FetchSymbolModel_NilExecutor(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, nil)
	est := NewHubCostEstimator(hub, &costsvc.CostModel{Symbol: "DEFAULT", SpreadPips: dec(2)}, nil)
	result := est.Estimate(context.Background(), costsvc.EstimateParams{
		Symbol: "EURUSD", Side: "buy", Lots: dec(1), Price: dec(1.085),
		ContractSize: dec(100000),
	})
	_ = result
}

// --- estimateOrderCost ---

func TestEstimateOrderCost(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetCostEstimator(NewHubCostEstimator(NewHub(), &costsvc.CostModel{
		Symbol: "DEFAULT", SpreadPips: dec(1.5), PipSize: dec(0.0001), PipValue: dec(10),
	}, nil))
	ce := svc.estimateOrderCost(context.Background(), &OrderRequest{
		Canonical: "EURUSD", Side: SideBuy, Volume: dec(0.1), Price: dec(1.085),
	})
	if ce == nil {
		t.Fatal("expected non-nil cost estimate")
	}
}

// --- SubscribeAccountStatus without broker ---

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

func TestHubCostEstimator_CacheHit(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(_ context.Context, _ []string) ([]*SymbolParam, error) {
			return []*SymbolParam{
				{Canonical: "EURUSD", Digits: 5, PointValue: dec(1)},
			}, nil
		},
	}
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	est := NewHubCostEstimator(hub, &costsvc.CostModel{Symbol: "DEFAULT"}, nil)
	params := costsvc.EstimateParams{
		Symbol: "EURUSD", Side: "buy", Lots: dec(1), Price: dec(1.085),
		ContractSize: dec(100000),
	}
	est.Estimate(context.Background(), params)
	est.Estimate(context.Background(), params)
}

// --- HubCostEstimator fetch error path ---

func TestHubCostEstimator_FetchError(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(_ context.Context, _ []string) ([]*SymbolParam, error) {
			return nil, ErrSessionNotFound
		},
	}
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	est := NewHubCostEstimator(hub, &costsvc.CostModel{Symbol: "DEFAULT", SpreadPips: dec(2)}, nil)
	result := est.Estimate(context.Background(), costsvc.EstimateParams{
		Symbol: "EURUSD", Side: "buy", Lots: dec(1), Price: dec(1.085),
		ContractSize: dec(100000),
	})
	_ = result
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
	if err != ErrRateLimited {
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
	if err != ErrRateLimited {
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
	if err != ErrRateLimited {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

// --- StateCache: position increase weighted average ---

func TestStateCache_PositionIncrease_WeightedAvg(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())

	// Buy 0.1 at 1.0850
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "BUY", Volume: dec(0.1), Price: dec(1.0850),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	pos := c.GetPosition("acc-1", "EURUSD")
	if !pos.AvgPrice.Equal(dec(1.0850)) {
		t.Fatalf("expected avg price 1.0850, got %s", pos.AvgPrice)
	}

	// Buy 0.05 more at 1.0900 → position increases, weighted avg
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 2,
		Canonical: "EURUSD", Side: "BUY", Volume: dec(0.05), Price: dec(1.0900),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	pos = c.GetPosition("acc-1", "EURUSD")
	// Weighted: (1.0850*0.1 + 1.0900*0.05) / 0.15 = (0.10850 + 0.05450) / 0.15 = 0.16300 / 0.15 = 1.08667...
	expected := dec(1.0850).Mul(dec(0.1)).Add(dec(1.0900).Mul(dec(0.05))).Div(dec(0.15))
	if !pos.AvgPrice.Equal(expected) {
		t.Fatalf("expected weighted avg %s, got %s", expected, pos.AvgPrice)
	}
}

// --- StateCache: short position increase weighted average ---

func TestStateCache_ShortPositionIncrease_WeightedAvg(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())

	// Sell 0.1 at 1.0850 → short
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "SELL", Volume: dec(0.1), Price: dec(1.0850),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	pos := c.GetPosition("acc-1", "EURUSD")
	if !pos.NetVolume.Equal(dec(-0.1)) {
		t.Fatalf("expected -0.1, got %s", pos.NetVolume)
	}

	// Sell 0.05 more at 1.0900 → short increases
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 2,
		Canonical: "EURUSD", Side: "SELL", Volume: dec(0.05), Price: dec(1.0900),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	pos = c.GetPosition("acc-1", "EURUSD")
	if !pos.NetVolume.Equal(dec(-0.15)) {
		t.Fatalf("expected -0.15, got %s", pos.NetVolume)
	}
	expected := dec(1.0850).Mul(dec(0.1)).Add(dec(1.0900).Mul(dec(0.05))).Div(dec(0.15))
	if !pos.AvgPrice.Equal(expected) {
		t.Fatalf("expected weighted avg %s, got %s", expected, pos.AvgPrice)
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

// --- Pure function: isValidOMSTransition ---

func TestIsValidOMSTransition(t *testing.T) {
	t.Parallel()
	valid := []struct{ from, to OMSState }{
		{OMSStateNew, OMSStateValidated},
		{OMSStateValidated, OMSStateRiskApproved},
		{OMSStateValidated, OMSStateRejected},
		{OMSStateRiskApproved, OMSStateSubmitted},
		{OMSStateRiskApproved, OMSStateFailed},
		{OMSStateSubmitted, OMSStateWorking},
		{OMSStateSubmitted, OMSStateFilled},
		{OMSStateSubmitted, OMSStateCancelled},
		{OMSStateWorking, OMSStatePartiallyFilled},
		{OMSStateWorking, OMSStateFilled},
		{OMSStateUnknown, OMSStateReconciling},
		{OMSStateReconciling, OMSStateWorking},
		{OMSStateRequoted, OMSStateRiskApproved},
		{OMSStateMarginCall, OMSStateCancelled},
	}
	for _, tr := range valid {
		if !isValidOMSTransition(tr.from, tr.to) {
			t.Errorf("%s → %s should be valid", tr.from, tr.to)
		}
	}
	invalid := []struct{ from, to OMSState }{
		{OMSStateNew, OMSStateSubmitted},
		{OMSStateFilled, OMSStateWorking},
		{OMSStateCancelled, OMSStateWorking},
		{OMSStateRejected, OMSStateNew},
		{OMSStateExpired, OMSStateWorking},
	}
	for _, tr := range invalid {
		if isValidOMSTransition(tr.from, tr.to) {
			t.Errorf("%s → %s should be invalid", tr.from, tr.to)
		}
	}
}

// --- Pure function: hashToNegative ---

func TestHashToNegative_Coverage(t *testing.T) {
	t.Parallel()
	v := hashToNegative("test-order-id")
	if v >= 0 {
		t.Fatalf("expected negative value, got %d", v)
	}
	v2 := hashToNegative("test-order-id")
	if v != v2 {
		t.Fatal("hashToNegative should be deterministic")
	}
	v3 := hashToNegative("different-id")
	if v == v3 {
		t.Fatal("different inputs should produce different hashes")
	}
}

// --- Pure function: advisoryLockKey ---

func TestAdvisoryLockKey_Coverage(t *testing.T) {
	t.Parallel()
	k1, k2 := advisoryLockKey("acc-1", "client-1")
	_ = k1
	_ = k2
	k3, k4 := advisoryLockKey("acc-1", "client-1")
	if k1 != k3 || k2 != k4 {
		t.Fatal("advisoryLockKey should be deterministic")
	}
	k5, k6 := advisoryLockKey("acc-2", "client-1")
	if k1 == k5 && k2 == k6 {
		t.Fatal("different accountID should produce different keys")
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
	if err != ErrKillSwitchEngaged {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

// --- CloseOrder with kill switch ---

func TestCloseOrder_KillSwitchEngaged(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	err := svc.CloseOrder(context.Background(), "acc-1", 123, decimal.NewFromInt(1))
	if err != ErrKillSwitchEngaged {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

// --- ModifyOrder with kill switch ---

func TestModifyOrder_KillSwitchEngaged(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err != ErrKillSwitchEngaged {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

// --- DeleteOrder with kill switch ---

func TestDeleteOrder_KillSwitchEngaged(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	err := svc.DeleteOrder(context.Background(), "acc-1", 123)
	if err != ErrKillSwitchEngaged {
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

func TestHubCostEstimator_DoubleCheckedLockingCacheHit(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	fetchCh := make(chan struct{})
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(_ context.Context, _ []string) ([]*SymbolParam, error) {
			<-fetchCh // block until signaled
			return []*SymbolParam{
				{Canonical: "EURUSD", Digits: 5, PointValue: dec(1)},
			}, nil
		},
	}
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	est := NewHubCostEstimator(hub, &costsvc.CostModel{Symbol: "DEFAULT"}, nil)
	params := costsvc.EstimateParams{
		Symbol: "EURUSD", Side: "buy", Lots: dec(1), Price: dec(1.085),
		ContractSize: dec(100000),
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// G1: will fetch (slow), then cache the result.
	go func() {
		defer wg.Done()
		est.Estimate(context.Background(), params)
	}()

	// Give G1 time to acquire the write lock and start fetching.
	time.Sleep(10 * time.Millisecond)

	// G2: will miss RLock cache, wait for write lock. When G1 finishes and unlocks,
	// G2 acquires lock and hits the double-checked locking cache hit (line 62).
	go func() {
		defer wg.Done()
		est.Estimate(context.Background(), params)
	}()

	// Give G2 time to hit the RLock cache miss and wait for the write lock.
	time.Sleep(10 * time.Millisecond)

	// Release G1's fetch — G1 caches and unlocks, G2 acquires lock and hits cache.
	close(fetchCh)
	wg.Wait()
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
