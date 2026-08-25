// mutation_coordinator_test.go — Production-wiring adversarial tests for
// LIVE-ORDER-REENTRY-1 (B8). These tests exercise the REAL production call
// chain: dispatchLiveSignal → submitOrder → coordinateMutation → broker RPC
// → PositionSnapshotBroker subscription → confirmation.
//
// All tests use channel-based synchronization — NO time.Sleep for concurrency.
// Cutting the production wiring (submitOrder→coordinateMutation or
// OnOrderUpdate→barrier) must make these tests RED.

package strategy

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
	"alphaforge/internal/risk"
)

// mockKillSwitch implements mthub.KillSwitchGate for testing.
type mockKillSwitch struct{ engaged bool }

func (m *mockKillSwitch) IsEngaged() bool { return m.engaged }

// toInt converts a numeric interface{} (int, int64, etc.) from a zaptest
// observer ContextMap to an int.
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case int32:
		return int(n), true
	default:
		return 0, false
	}
}

// prodMockExecutor is a controllable mock for production-wiring tests.
// Each function is injectable per-test. Call counts are tracked atomically.
type prodMockExecutor struct {
	placeFn  func(ctx context.Context, req *mthub.OrderRequest) (int64, error)
	closeFn  func(ctx context.Context, ticket int64, lots decimal.Decimal) error
	deleteFn func(ctx context.Context, ticket int64) error
	modifyFn func(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error
	fetchFn  func(ctx context.Context) ([]*mthub.OrderRecord, error)

	placeCount  atomic.Int64
	closeCount  atomic.Int64
	deleteCount atomic.Int64
	modifyCount atomic.Int64
}

func (m *prodMockExecutor) Platform() string { return "mock" }
func (m *prodMockExecutor) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
	m.placeCount.Add(1)
	if m.placeFn != nil {
		return m.placeFn(ctx, req)
	}
	return 1, nil
}
func (m *prodMockExecutor) CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error {
	m.closeCount.Add(1)
	if m.closeFn != nil {
		return m.closeFn(ctx, ticket, lots)
	}
	return nil
}
func (m *prodMockExecutor) DeleteOrder(ctx context.Context, ticket int64) error {
	m.deleteCount.Add(1)
	if m.deleteFn != nil {
		return m.deleteFn(ctx, ticket)
	}
	return nil
}
func (m *prodMockExecutor) ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
	m.modifyCount.Add(1)
	if m.modifyFn != nil {
		return m.modifyFn(ctx, ticket, sl, tp, price)
	}
	return nil
}
func (m *prodMockExecutor) FetchOpenedOrders(ctx context.Context) ([]*mthub.OrderRecord, error) {
	if m.fetchFn != nil {
		return m.fetchFn(ctx)
	}
	return nil, nil
}
func (m *prodMockExecutor) FetchOrderHistory(_ context.Context, _, _ time.Time) ([]*mthub.OrderRecord, error) {
	return nil, nil
}
func (m *prodMockExecutor) FetchSymbolParams(_ context.Context, c []string) ([]*mthub.SymbolParam, error) {
	if len(c) == 0 {
		return nil, nil
	}
	return []*mthub.SymbolParam{{Canonical: c[0], ContractSize: decimal.NewFromInt(100000), LotSize: decimal.NewFromInt(100000)}}, nil
}
func (m *prodMockExecutor) FetchAllSymbols(_ context.Context) ([]string, error) { return nil, nil }
func (m *prodMockExecutor) FetchPriceHistory(_ context.Context, _ string, _ string, _, _ int64, _ int) ([]*mthub.Bar, error) {
	return nil, nil
}
func (m *prodMockExecutor) AddSymbols(_ context.Context, _ []string) error { return nil }
func (m *prodMockExecutor) SubscribeOrderEvents(_ context.Context, _ mthub.OrderEventHandler) error {
	return nil
}

// testCoordinatorSetup creates a full production wiring chain for testing.
// Returns the server, mtHub service, mock executor, and snapshot broker.
func testCoordinatorSetup(exec *prodMockExecutor) (*StrategyExecutionServer, *mthub.MtHubService, *mthub.PositionSnapshotBroker) {
	hub := mthub.NewHub()
	broker := mthub.NewPositionSnapshotBroker()
	svc := mthub.NewMtHubService(hub, mthub.NewOrderEventBroker(), mthub.NewAccountProfitBroker(), broker, nil, nil, nil)
	svc.SetLogger(zap.NewNop())
	svc.SetGate(risk.NewDefaultGate())
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: decimal.NewFromInt(100000), Equity: decimal.NewFromInt(100000)}, nil
	})
	hub.Register("acct-1", &mthub.Session{AccountID: "acct-1", CreatedAt: time.Now()}, exec)
	srv := &StrategyExecutionServer{log: zap.NewNop(), mtHub: svc}
	return srv, svc, broker
}

func testLiveCfg() LiveStrategyConfig {
	return LiveStrategyConfig{
		AccountID:  "acct-1",
		UserID:     "user-1",
		Symbol:     "EURUSD",
		Mode:       "live",
		RunID:      uuid.New(),
		TickSeq:    new(atomic.Int64),
		ScheduleID: uuid.New(),
	}
}

func testActiveSess() *ActiveSession {
	return &ActiveSession{barrier: NewTradeBarrier(zap.NewNop())}
}

// publishOrderUpdate publishes an OnOrderUpdate event to the snapshot broker,
// simulating a broker push. This exercises the REAL subscription wiring.
func publishOrderUpdate(broker *mthub.PositionSnapshotBroker, accountID string, ticket int64, magic int32, updateType string) {
	broker.Publish(&mthub.PositionSnapshot{
		AccountID:              accountID,
		FinancialsSource:       "order_stream",
		CapturedAt:             time.Now(),
		PositionsAuthoritative: true,
		PositionsCapturedAt:    time.Now(),
		PositionsSource:        "order_stream",
		Positions:              []mthub.PositionSnapshotItem{{Ticket: ticket, Magic: magic}},
		UpdateTicket:           ticket,
		UpdateType:             updateType,
		UpdateMagic:            magic,
	})
}

// T1-PROD: 100 concurrent tick signals, only 1 PlaceOrder call.
// Uses REAL dispatchLiveSignal → submitOrder → coordinateMutation wiring.
// Cutting submitOrder→coordinateMutation must RED.
func TestLIVE_ORDER_REENTRY_1_T1_PROD_ConcurrentTicksSingleBrokerCall(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			return 1, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{Ticket: 1, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	done := make(chan struct{}, 100)
	for i := 0; i < 100; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
			srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	if got := exec.placeCount.Load(); got != 1 {
		t.Fatalf("T1-PROD: PlaceOrder called %d times, want 1 — I1 violated", got)
	}
}

// T2-PROD: PlaceOrder returns ticket but no OnOrderUpdate, fallback OpenedOrders
// also fails → outcome_unknown, barrier locked, subsequent signals blocked.
func TestLIVE_ORDER_REENTRY_1_T2_PROD_NoPushNoFallback(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			return 42, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return nil, errors.New("broker unavailable")
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("T2-PROD: barrier state=%s, want outcome_unknown", state)
	}
	// Subsequent signal must not call PlaceOrder (barrier locked).
	sig2 := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig2, sess)
	if got := exec.placeCount.Load(); got != 1 {
		t.Fatalf("T2-PROD: PlaceOrder called %d times after outcome_unknown, want 1", got)
	}
}

// T3-PROD-A: OnOrderUpdate arrives BEFORE PlaceOrder returns (pre-response race).
// Uses REAL PositionSnapshotBroker subscription.
func TestLIVE_ORDER_REENTRY_1_T3_PROD_A_PreResponsePush(t *testing.T) {
	placeStarted := make(chan struct{})
	placeProceed := make(chan struct{})
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			close(placeStarted)
			<-placeProceed
			return 42, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{Ticket: 42, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	magic := strategyMagic(cfg.ScheduleID)
	done := make(chan struct{})
	go func() {
		defer close(done)
		sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
		srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
	}()

	// Wait for PlaceOrder to start, then publish OnOrderUpdate BEFORE it returns.
	<-placeStarted
	publishOrderUpdate(broker, cfg.AccountID, 42, magic, "open")
	// Now let PlaceOrder return.
	close(placeProceed)
	<-done

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("T3-PROD-A: barrier state=%s, want idle (confirmed+released)", state)
	}
	if got := exec.placeCount.Load(); got != 1 {
		t.Fatalf("T3-PROD-A: PlaceOrder called %d times, want 1", got)
	}
}

// T3-PROD-B: OnOrderUpdate arrives AFTER PlaceOrder returns.
// Listener must still be alive and confirm immediately (no fallback wait).
func TestLIVE_ORDER_REENTRY_1_T3_PROD_B_PostResponsePush(t *testing.T) {
	placeReturned := make(chan struct{})
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			close(placeReturned)
			return 42, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			t.Fatal("T3-PROD-B: read-after-write should not be needed when push confirms")
			return nil, nil
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()
	magic := strategyMagic(cfg.ScheduleID)

	done := make(chan struct{})
	go func() {
		defer close(done)
		sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
		srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
	}()

	// Wait for PlaceOrder to return, then publish push (deterministic, no sleep).
	<-placeReturned
	publishOrderUpdate(broker, cfg.AccountID, 42, magic, "open")
	<-done

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("T3-PROD-B: barrier state=%s, want idle (confirmed+released)", state)
	}
}

// T4-PROD: Unrelated ticket and matching ticket with wrong magic must NOT confirm.
func TestLIVE_ORDER_REENTRY_1_T4_PROD_UnrelatedEventsNotConfirmed(t *testing.T) {
	placeStarted := make(chan struct{})
	placeProceed := make(chan struct{})
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			close(placeStarted)
			<-placeProceed
			return 42, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{Ticket: 42, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()
	magic := strategyMagic(cfg.ScheduleID)

	done := make(chan struct{})
	go func() {
		defer close(done)
		sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
		srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
	}()

	<-placeStarted
	// Send unrelated ticket (99) — must not confirm.
	publishOrderUpdate(broker, cfg.AccountID, 99, magic, "open")
	// Send matching ticket (42) with wrong magic (999) — must not confirm.
	publishOrderUpdate(broker, cfg.AccountID, 42, 999, "open")
	// Now send correct ticket+magic — must confirm.
	publishOrderUpdate(broker, cfg.AccountID, 42, magic, "open")
	close(placeProceed)
	<-done

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("T4-PROD: barrier state=%s, want idle (confirmed+released)", state)
	}
}

// T5-PROD: Typed deterministic rejections (gate, kill switch, rate limit, duplicate).
// Each must release the barrier and allow the next signal.
// Uses REAL service-layer rejection paths (not executor mock).
func TestLIVE_ORDER_REENTRY_1_T5_PROD_DeterministicRejections(t *testing.T) {
	// Subtest 1: Kill switch engaged → pre-broker rejection.
	t.Run("kill_switch", func(t *testing.T) {
		exec := &prodMockExecutor{}
		srv, svc, _ := testCoordinatorSetup(exec)
		svc.SetKillSwitch(&mockKillSwitch{engaged: true})
		cfg := testLiveCfg()
		sess := testActiveSess()

		sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
		srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

		if state := sess.barrier.State(); state != barrierIdle {
			t.Fatalf("T5-PROD kill_switch: barrier state=%s, want idle (released)", state)
		}
	})

	// Subtest 2: Gate not configured → pre-broker rejection (fail-closed).
	t.Run("gate_not_configured", func(t *testing.T) {
		exec := &prodMockExecutor{}
		hub := mthub.NewHub()
		broker := mthub.NewPositionSnapshotBroker()
		svc := mthub.NewMtHubService(hub, mthub.NewOrderEventBroker(), mthub.NewAccountProfitBroker(), broker, nil, nil, nil)
		svc.SetLogger(zap.NewNop())
		// No gate set → gate not configured → pre-broker rejection.
		hub.Register("acct-1", &mthub.Session{AccountID: "acct-1", CreatedAt: time.Now()}, exec)
		srv := &StrategyExecutionServer{log: zap.NewNop(), mtHub: svc}
		cfg := testLiveCfg()
		sess := testActiveSess()

		sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
		srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

		if state := sess.barrier.State(); state != barrierIdle {
			t.Fatalf("T5-PROD gate_not_configured: barrier state=%s, want idle (released)", state)
		}
	})

	// Subtest 3: Session not found → pre-broker rejection.
	// R6: verifies the error is PhasePreBroker (not double-wrapped as PhaseBroker).
	t.Run("session_not_found", func(t *testing.T) {
		hub := mthub.NewHub()
		broker := mthub.NewPositionSnapshotBroker()
		svc := mthub.NewMtHubService(hub, mthub.NewOrderEventBroker(), mthub.NewAccountProfitBroker(), broker, nil, nil, nil)
		svc.SetLogger(zap.NewNop())
		svc.SetGate(risk.NewDefaultGate())
		svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
			return &risk.AccountState{Balance: decimal.NewFromInt(100000), Equity: decimal.NewFromInt(100000)}, nil
		})
		// No session registered → ErrSessionNotFound → pre-broker.
		srv := &StrategyExecutionServer{log: zap.NewNop(), mtHub: svc}
		cfg := testLiveCfg()
		sess := testActiveSess()

		// R6: directly call PlaceOrder to verify the error phase.
		req := &mthub.OrderRequest{
			AccountID: cfg.AccountID, Canonical: cfg.Symbol,
			Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
			Volume: decimal.NewFromFloat(0.1), Magic: strategyMagic(cfg.ScheduleID),
			ClientID: "test-r6",
		}
		_, err := svc.PlaceOrder(context.Background(), req)
		if err == nil {
			t.Fatal("T5-PROD session_not_found: PlaceOrder should return error")
		}
		// R6: verify the error is classified as deterministic_rejected (pre-broker).
		outcome := mthub.ClassifyMutationError(err)
		if outcome != "deterministic_rejected" {
			t.Fatalf("T5-PROD session_not_found: ClassifyMutationError=%s, want deterministic_rejected (R6: no double-wrapping)", outcome)
		}
		// Verify the error is a MutationError with PhasePreBroker.
		var me *mthub.MutationError
		if !errors.As(err, &me) {
			t.Fatalf("T5-PROD session_not_found: error is not MutationError, got %T", err)
		}
		if me.Phase != mthub.PhasePreBroker {
			t.Fatalf("T5-PROD session_not_found: MutationError.Phase=%v, want PhasePreBroker (R6: no double-wrapping)", me.Phase)
		}

		// Also verify the barrier is released via the full dispatch path.
		sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
		srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
		if state := sess.barrier.State(); state != barrierIdle {
			t.Fatalf("T5-PROD session_not_found: barrier state=%s, want idle (released)", state)
		}
	})
}

// T6-PROD: Transport timeout / unknown error → outcome_unknown, barrier locked.
func TestLIVE_ORDER_REENTRY_1_T6_PROD_TransportTimeoutStaysLocked(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			return 0, &mthub.MutationError{Phase: mthub.PhaseBroker, Cause: errors.New("context deadline exceeded")}
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("T6-PROD: barrier state=%s, want outcome_unknown", state)
	}
	if got := exec.placeCount.Load(); got != 1 {
		t.Fatalf("T6-PROD: PlaceOrder called %d times, want 1", got)
	}
}

// T8-BROKER-REJECT: Broker application-level rejection (e.g. MT4 code=130
// Invalid S/L or T/P) must be classified as deterministic_rejected, NOT
// outcome_unknown. The broker saw the request and definitively said no —
// no ticket was assigned, the order did not execute. The barrier must be
// released so the strategy can retry on the next tick.
//
// This test simulates the exact production scenario: account 904d14e6,
// schedule 599ddaa5, MT4 OrderSend returned code=130 "Invalid S/L or T/P".
// Before the fix, this was classified as outcome_unknown → barrier locked
// forever → strategy could never place another order.
func TestLIVE_ORDER_REENTRY_1_T8_BrokerAppRejectionReleasesBarrier(t *testing.T) {
	// Simulate MT4 adapter returning ErrBrokerRejected (wrapped by brokerError
	// in submitToBroker → MutationError{PhaseBroker, Cause: ErrBrokerRejected}).
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			return 0, fmt.Errorf("%w: mt4 OrderSend: code=130 msg=Invalid S/L or T/P", mthub.ErrBrokerRejected)
		},
	}
	srv, svc, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	// First: verify PlaceOrder classifies the error correctly via the real
	// MtHubService path (submitToBroker → brokerError → ClassifyMutationError).
	req := &mthub.OrderRequest{
		AccountID: cfg.AccountID, Canonical: cfg.Symbol,
		Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1), Magic: strategyMagic(cfg.ScheduleID),
		ClientID: "test-t8-broker-reject",
	}
	_, err := svc.PlaceOrder(context.Background(), req)
	if err == nil {
		t.Fatal("T8-BROKER-REJECT: PlaceOrder should return error")
	}
	outcome := mthub.ClassifyMutationError(err)
	if outcome != "deterministic_rejected" {
		t.Fatalf("T8-BROKER-REJECT: ClassifyMutationError=%s, want deterministic_rejected (broker app rejection is deterministic)", outcome)
	}

	// Second: verify the full dispatch path releases the barrier.
	// Reset placeCount since the direct PlaceOrder call above already counted.
	exec.placeCount.Store(0)
	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("T8-BROKER-REJECT: barrier state=%s, want idle (released after broker rejection)", state)
	}
	if got := exec.placeCount.Load(); got != 1 {
		t.Fatalf("T8-BROKER-REJECT: PlaceOrder called %d times, want 1 (via dispatch only)", got)
	}
}

// T6-CONFIRMED-RACE: Push confirms (known ticket), then RPC returns transport error.
// Must converge to confirmed, not lock. Uses close mutation (ticket known beforehand).
func TestLIVE_ORDER_REENTRY_1_T6_CONFIRMED_RACE(t *testing.T) {
	placeStarted := make(chan struct{})
	placeProceed := make(chan struct{})
	exec := &prodMockExecutor{
		closeFn: func(ctx context.Context, ticket int64, lots decimal.Decimal) error {
			close(placeStarted)
			<-placeProceed
			return &mthub.MutationError{Phase: mthub.PhaseBroker, Cause: errors.New("transport timeout")}
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{Ticket: 42, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()
	magic := strategyMagic(cfg.ScheduleID)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Close order with known ticket=42.
		sig := &antv1.StrategySignal{SignalType: "close", Volume: "0", ExecutedTicket: 42}
		srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
	}()

	<-placeStarted
	// Push confirms close of ticket 42 BEFORE RPC returns error.
	publishOrderUpdate(broker, cfg.AccountID, 42, magic, "close")
	// Now let the RPC return a transport error.
	// R1 fix: the coordinator uses WaitConfirmed (bounded pushWait) after
	// NotifyBrokerAccepted, so the listener goroutine has time to process
	// the push. No time.Sleep needed — the wait is deterministic.
	close(placeProceed)
	<-done

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("T6-CONFIRMED-RACE: barrier state=%s, want idle (converged to confirmed)", state)
	}
}

// T7-PROD: No push, single read-after-write OpenedOrders succeeds.
func TestLIVE_ORDER_REENTRY_1_T7_PROD_ReadAfterWriteSucceeds(t *testing.T) {
	var fetchCount atomic.Int64
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			return 42, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			fetchCount.Add(1)
			return []*mthub.OrderRecord{{Ticket: 42, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("T7-PROD: barrier state=%s, want idle (confirmed+released)", state)
	}
	if got := fetchCount.Load(); got != 1 {
		t.Fatalf("T7-PROD: OpenedOrders called %d times, want exactly 1 (not polling)", got)
	}
}

// T7-FAIL: Read-after-write fails → outcome_unknown, barrier locked.
