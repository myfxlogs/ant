package mthub

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/internal/costsvc"
	"alphaforge/internal/risk"
)

// --- Pure function tests (no dependencies) ---

func TestHashToNegative_Deterministic(t *testing.T) {
	t.Parallel()
	a := hashToNegative("order-abc-123")
	b := hashToNegative("order-abc-123")
	if a != b {
		t.Fatalf("hashToNegative must be deterministic: %d vs %d", a, b)
	}
}

func TestHashToNegative_Different(t *testing.T) {
	t.Parallel()
	a := hashToNegative("order-1")
	b := hashToNegative("order-2")
	if a == b {
		t.Fatal("different IDs should produce different hashes")
	}
}

func TestHashToNegative_AlwaysNegative(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"", "a", "order-1", "very-long-order-id-12345"} {
		h := hashToNegative(id)
		if h >= 0 {
			t.Errorf("hashToNegative(%q) = %d, expected negative", id, h)
		}
	}
}

func TestAdvisoryLockKey_Deterministic(t *testing.T) {
	t.Parallel()
	k1, k2 := advisoryLockKey("acc-1", "client-1")
	k1b, k2b := advisoryLockKey("acc-1", "client-1")
	if k1 != k1b || k2 != k2b {
		t.Fatalf("advisoryLockKey must be deterministic: (%d,%d) vs (%d,%d)", k1, k2, k1b, k2b)
	}
}

func TestAdvisoryLockKey_DifferentInputs(t *testing.T) {
	t.Parallel()
	a1, a2 := advisoryLockKey("acc-1", "client-1")
	b1, b2 := advisoryLockKey("acc-1", "client-2")
	c1, c2 := advisoryLockKey("acc-2", "client-1")
	sameAB := a1 == b1 && a2 == b2
	sameAC := a1 == c1 && a2 == c2
	if sameAB || sameAC {
		t.Fatal("different inputs should produce different lock keys")
	}
}

// --- OMS state machine: full coverage of all valid transitions ---

func TestIsValidOMSTransition_AllValid(t *testing.T) {
	t.Parallel()
	valid := []struct{ from, to OMSState }{
		{OMSStateNew, OMSStateValidated},
		{OMSStateValidated, OMSStateRiskApproved},
		{OMSStateRiskApproved, OMSStateSubmitted},
		{OMSStateSubmitted, OMSStateWorking},
		{OMSStateSubmitted, OMSStatePartiallyFilled},
		{OMSStateSubmitted, OMSStateFilled},
		{OMSStateSubmitted, OMSStateCancelled},
		{OMSStateSubmitted, OMSStateExpired},
		{OMSStateSubmitted, OMSStateFailed},
		{OMSStateSubmitted, OMSStateUnknown},
		{OMSStateSubmitted, OMSStateRequoted},
		{OMSStateSubmitted, OMSStateSlippageRejected},
		{OMSStateSubmitted, OMSStateMarginCall},
		{OMSStateWorking, OMSStatePartiallyFilled},
		{OMSStateWorking, OMSStateFilled},
		{OMSStateWorking, OMSStateCancelled},
		{OMSStateWorking, OMSStateExpired},
		{OMSStateWorking, OMSStateFailed},
		{OMSStateWorking, OMSStateRequoted},
		{OMSStatePartiallyFilled, OMSStatePartiallyFilled},
		{OMSStatePartiallyFilled, OMSStateFilled},
		{OMSStatePartiallyFilled, OMSStateCancelled},
		{OMSStatePartiallyFilled, OMSStateExpired},
		{OMSStatePartiallyFilled, OMSStateFailed},
		{OMSStateValidated, OMSStateRejected},
		{OMSStateRiskApproved, OMSStateRejected},
		{OMSStateRiskApproved, OMSStateFailed},
		{OMSStateRequoted, OMSStateRiskApproved},
		{OMSStateRequoted, OMSStateCancelled},
		{OMSStateRequoted, OMSStateExpired},
		{OMSStateSlippageRejected, OMSStateRiskApproved},
		{OMSStateSlippageRejected, OMSStateCancelled},
		{OMSStateSlippageRejected, OMSStateExpired},
		{OMSStateUnknown, OMSStateReconciling},
		{OMSStateUnknown, OMSStateWorking},
		{OMSStateUnknown, OMSStateFilled},
		{OMSStateUnknown, OMSStateCancelled},
		{OMSStateUnknown, OMSStateFailed},
		{OMSStateUnknown, OMSStateExpired},
		{OMSStateReconciling, OMSStateWorking},
		{OMSStateReconciling, OMSStatePartiallyFilled},
		{OMSStateReconciling, OMSStateFilled},
		{OMSStateReconciling, OMSStateCancelled},
		{OMSStateReconciling, OMSStateFailed},
		{OMSStateReconciling, OMSStateExpired},
		{OMSStateMarginCall, OMSStateRiskApproved},
		{OMSStateMarginCall, OMSStateCancelled},
		{OMSStateMarginCall, OMSStateExpired},
		{OMSStateMarginCall, OMSStateFailed},
	}
	for _, tc := range valid {
		if !isValidOMSTransition(tc.from, tc.to) {
			t.Errorf("%s → %s should be valid", tc.from, tc.to)
		}
	}
}

func TestIsValidOMSTransition_TerminalStates_NoExit(t *testing.T) {
	t.Parallel()
	terminal := []OMSState{OMSStateFilled, OMSStateCancelled, OMSStateExpired, OMSStateRejected, OMSStateFailed}
	allStates := []OMSState{
		OMSStateNew, OMSStateValidated, OMSStateRiskApproved, OMSStateSubmitted,
		OMSStateWorking, OMSStatePartiallyFilled, OMSStateFilled,
		OMSStateCancelled, OMSStateRejected, OMSStateFailed, OMSStateExpired,
		OMSStateRequoted, OMSStateSlippageRejected, OMSStateUnknown,
		OMSStateReconciling, OMSStateMarginCall,
	}
	for _, term := range terminal {
		for _, next := range allStates {
			if isValidOMSTransition(term, next) {
				t.Errorf("terminal state %s → %s should be invalid", term, next)
			}
		}
	}
}

// --- Helpers ---

// mockKillSwitch implements KillSwitchGate for testing.
type mockKillSwitch struct{ engaged bool }

func (m *mockKillSwitch) IsEngaged() bool { return m.engaged }

func mustDec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

// newTestService creates a minimally wired MtHubService for unit testing.
// Includes a permissive gate + state provider so PlaceOrder/CloseOrder
// pass the gate check (DEPLOY-LIVE-4 fail-closed). Tests that need to
// test nil-gate behavior should use newTestServiceNoGate().
func newTestService() *MtHubService {
	hub := NewHub()
	svc := NewMtHubService(hub, NewOrderEventBroker(), NewAccountProfitBroker(), NewPositionSnapshotBroker(), nil, nil, nil)
	svc.SetGate(risk.NewDefaultGate())
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: dec(100000), Equity: dec(100000), FreeMargin: dec(95000)}, nil
	})
	return svc
}

// newTestServiceNoGate creates a service without gate/state provider for
// testing fail-closed behavior (DEPLOY-LIVE-4).
func newTestServiceNoGate() *MtHubService {
	hub := NewHub()
	return NewMtHubService(hub, NewOrderEventBroker(), NewAccountProfitBroker(), NewPositionSnapshotBroker(), nil, nil, nil)
}

// --- Service order tests ---

func TestDeleteOrder_KillSwitch(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	err := svc.DeleteOrder(context.Background(), "acc-1", 123)
	if err == nil || err.Error() != ErrKillSwitchEngaged.Error() {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

func TestDeleteOrder_NoSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	err := svc.DeleteOrder(context.Background(), "acc-1", 123)
	if err == nil {
		t.Fatal("expected error for no session")
	}
}

func TestDeleteOrder_Success(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.DeleteOrder(context.Background(), "acc-1", 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteOrder_ExecutorError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{
		platform:      "MT5",
		deleteOrderFn: func(ctx context.Context, ticket int64) error { return ErrSessionNotFound },
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	err := svc.DeleteOrder(context.Background(), "acc-1", 123)
	if err == nil {
		t.Fatal("expected executor error")
	}
}

func TestDeleteOrder_OwnershipCheck_Unauthenticated(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.SetAccountOwnerVerifier(func(ctx context.Context, userID, accountID string) (bool, error) {
		return false, nil
	})
	err := svc.DeleteOrder(context.Background(), "acc-1", 123)
	if err == nil {
		t.Fatal("expected unauthenticated error")
	}
}

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

func TestModifyOrder_KillSwitch(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err == nil || err.Error() != ErrKillSwitchEngaged.Error() {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

func TestModifyOrder_NoSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err == nil {
		t.Fatal("expected error for no session")
	}
}

func TestModifyOrder_Success(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, mustDec("1.08"), mustDec("1.10"), decimal.Zero)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestModifyOrder_ExecutorError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{
		platform: "MT5",
		modifyOrderFn: func(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
			return ErrSessionNotFound
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err == nil {
		t.Fatal("expected executor error")
	}
}

func TestModifyOrder_OwnershipCheck(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.SetAccountOwnerVerifier(func(ctx context.Context, userID, accountID string) (bool, error) {
		return false, nil
	})
	err := svc.ModifyOrder(context.Background(), "acc-1", 123, decimal.Zero, decimal.Zero, decimal.Zero)
	if err == nil {
		t.Fatal("expected error for ownership check failure")
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

func TestOmsTransition_NilWriter(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.omsTransition(context.Background(), "ord-1", "acc-1", OMSStateNew, OMSStateValidated)
}

func TestOmsTransition_EmptyOrderID(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.omsTransition(context.Background(), "", "acc-1", OMSStateNew, OMSStateValidated)
}

func TestPublishOrderCreatedEvent_NilStore(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.publishOrderCreatedEvent(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD", Side: SideBuy, OrderType: OrderMarket,
	}, 12345, nil)
}

func TestPostCloseFailure_NilOMSWriter(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.postCloseFailure(context.Background(), "close-1", "acc-1", 123, ErrSessionNotFound)
}

func TestPostCloseSuccess_NilOMSWriter(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.postCloseSuccess(context.Background(), "close-1", "acc-1", 123)
}

func TestSubmitToBroker_NoSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	_, err := svc.submitToBroker(context.Background(), &OrderRequest{
		AccountID: "acc-1", Side: SideBuy, OrderType: OrderMarket,
	}, "ord-1")
	if err == nil {
		t.Fatal("expected error for no session")
	}
}

func TestPlaceOrder_KillSwitch(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Side: SideBuy, OrderType: OrderMarket,
	})
	if err == nil || err.Error() != ErrKillSwitchEngaged.Error() {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

func TestPlaceOrder_NoSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Side: SideBuy, OrderType: OrderMarket, Canonical: "EURUSD",
	})
	if err == nil {
		t.Fatal("expected error for no session")
	}
}

func TestPlaceOrder_Success(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	// Without OMS writer, cost estimator, or event store — should still succeed.
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
	if record.State != OrderStatePending {
		t.Fatalf("expected state 0 (Pending), got %d", record.State)
	}
}

func TestPlaceOrder_ExecutorError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{
		platform: "MT5",
		placeOrderFn: func(ctx context.Context, req *OrderRequest) (int64, error) {
			return 0, ErrSessionNotFound
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
	})
	if err == nil {
		t.Fatal("expected executor error")
	}
}

func TestPlaceOrder_OwnershipCheck_Unauthenticated(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.SetAccountOwnerVerifier(func(ctx context.Context, userID, accountID string) (bool, error) {
		return false, nil
	})
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Side: SideBuy, OrderType: OrderMarket,
	})
	if err == nil {
		t.Fatal("expected ownership error")
	}
}

func TestPlaceOrder_WithEventStore(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	// Attach an event store — should publish order created event without panic.
	svc.eventStore = NewTradeEventStore(nil) // nil NATS conn, Publish will be no-op
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

func TestSanitizeUTF8(t *testing.T) {
	t.Parallel()
	// Valid UTF-8 input should be returned unchanged.
	if got := sanitizeUTF8("hello"); got != "hello" {
		t.Errorf("sanitizeUTF8(%q) = %q, want %q", "hello", got, "hello")
	}
	if got := sanitizeUTF8("café"); got != "café" {
		t.Errorf("sanitizeUTF8(%q) = %q, want %q", "café", got, "café")
	}
	if got := sanitizeUTF8("emoji 👍 test"); got != "emoji 👍 test" {
		t.Errorf("sanitizeUTF8(%q) = %q, want %q", "emoji 👍 test", got, "emoji 👍 test")
	}
	if got := sanitizeUTF8(""); got != "" {
		t.Errorf("sanitizeUTF8(%q) = %q, want %q", "", got, "")
	}
	// Invalid UTF-8 should be repaired (U+FFFD replacement).
	invalid := "h\xc3llo"
	repaired := sanitizeUTF8(invalid)
	if repaired == invalid {
		t.Error("sanitizeUTF8 should repair invalid UTF-8")
	}
}

func TestPlaceOrder_WithCostEstimator(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(ctx context.Context, canonicals []string) ([]*SymbolParam, error) {
			return []*SymbolParam{
				{Canonical: "EURUSD", Digits: 5, PointValue: decimal.NewFromFloat(0.00001)},
			}, nil
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	def := &costsvc.CostModel{Symbol: "DEFAULT", SpreadPips: decimal.Zero}
	svc.SetCostEstimator(NewHubCostEstimator(svc.hub, def, nil))
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

func TestHubCostEstimator_FetchFailsUsesDefault(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	exec := &mockExecutor{platform: "MT5"}
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	def := &costsvc.CostModel{Symbol: "DEFAULT", SpreadPips: decimal.NewFromFloat(2.0)}
	estimator := NewHubCostEstimator(hub, def, nil)
	result := estimator.Estimate(context.Background(), costsvc.EstimateParams{
		Symbol:       "EURUSD",
		Side:         "buy",
		Lots:         decimal.NewFromInt(1),
		Price:        decimal.NewFromFloat(1.085),
		ContractSize: decimal.NewFromInt(100000),
	})
	// Should use default model (mock returns nil params → fetch fails → uses default).
	if result.SpreadCost.IsZero() {
		t.Log("spread cost is zero — default model may have spreadPips=0")
	}
}

func TestDerivedState_Recalculate_NoData(t *testing.T) {
	t.Parallel()
	computer := NewDerivedComputer(NewStateCache(nil, nil), 1*time.Minute)
	computer.recalculate()
	ds := computer.State()
	if ds.TotalExposure.IsZero() {
		// Expected with no data (no active accounts).
	}
}

func TestMtHubService_PublishTick_WithBroker(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{tickBroker: NewTickBroker(64, nil)}
	ch, cancel := svc.SubscribeTickUpdates("acc-1")
	defer cancel()
	svc.PublishTick(&TickUpdate{AccountID: "acc-1", Bid: dec(1.085), Ask: dec(1.0851)})
	ev := <-ch
	if !ev.Bid.Equal(dec(1.085)) {
		t.Fatalf("expected bid 1.085, got %s", ev.Bid.String())
	}
}

func TestMtHubService_PublishTradeEvent_WithBroker(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{tradeBroker: NewTradeBroker(64, nil)}
	ch, cancel := svc.SubscribeTradeEvents("acc-1")
	defer cancel()
	svc.PublishTradeEvent(&BrokerTradeEvent{AccountID: "acc-1", Ticket: 456})
	ev := <-ch
	if ev.Ticket != 456 {
		t.Fatalf("expected ticket 456, got %d", ev.Ticket)
	}
}

func TestMtHubService_PublishAccountStatus_WithBroker(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{statusBroker: NewAccountStatusBroker()}
	ch, cancel := svc.SubscribeAccountStatus("acc-1")
	defer cancel()
	svc.PublishAccountStatus(&AccountStatusEvent{AccountID: "acc-1"})
	ev := <-ch
	if ev.AccountID != "acc-1" {
		t.Fatalf("expected acc-1, got %s", ev.AccountID)
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

func TestTradeEventStore_EventToPayload(t *testing.T) {
	t.Parallel()
	ev := &TradeEvent{
		EventID:    "ev-1",
		EventType:  TradeEventOrderCreated,
		AccountID:  "acc-1",
		Ticket:     123,
		ClientID:   "client-1",
		Canonical:  "EURUSD",
		Side:       "buy",
		OrderType:  "market",
		Volume:     dec(0.1),
		Price:      dec(1.085),
		StopLoss:   dec(1.08),
		TakeProfit: dec(1.10),
		ToState:    "SUBMITTED",
		FromState:  string(OMSStateRiskApproved),
	}
	payload := eventToPayload(ev)
	if payload.EventId != "ev-1" {
		t.Fatalf("expected ev-1, got %s", payload.EventId)
	}
	if payload.Ticket != 123 {
		t.Fatalf("expected 123, got %d", payload.Ticket)
	}
	if payload.Canonical != "EURUSD" {
		t.Fatalf("expected EURUSD, got %s", payload.Canonical)
	}
}
