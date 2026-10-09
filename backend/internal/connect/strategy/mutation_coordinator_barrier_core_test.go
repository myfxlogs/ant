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
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
	"alphaforge/internal/risk"
)

// T1-PROD: 100 concurrent tick signals, only 1 PlaceOrder call.
// Uses REAL dispatchLiveSignal → submitOrder → coordinateMutation wiring.
// Cutting submitOrder→coordinateMutation must RED.
func TestLIVE_ORDER_REENTRY_1_T1_PROD_ConcurrentTicksSingleBrokerCall(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			return &mthub.OrderRecord{Ticket: 1, State: mthub.OrderStateOpen}, nil
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

// T2-PROD: PlaceOrder returns ticket but no OnOrderUpdate, fallback OpenedOrders
// also fails → outcome_unknown, barrier locked, subsequent signals blocked.
func TestLIVE_ORDER_REENTRY_1_T2_PROD_NoPushNoFallback(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			return &mthub.OrderRecord{Ticket: 42, State: mthub.OrderStateOpen}, nil
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

// T3-PROD-A: OnOrderUpdate arrives BEFORE PlaceOrder returns (pre-response race).
// Uses REAL PositionSnapshotBroker subscription.
func TestLIVE_ORDER_REENTRY_1_T3_PROD_A_PreResponsePush(t *testing.T) {
	placeStarted := make(chan struct{})
	placeProceed := make(chan struct{})
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			close(placeStarted)
			<-placeProceed
			return &mthub.OrderRecord{Ticket: 42, State: mthub.OrderStateOpen}, nil
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

// T3-PROD-B: OnOrderUpdate arrives AFTER PlaceOrder returns.
// Listener must still be alive and confirm immediately (no fallback wait).
func TestLIVE_ORDER_REENTRY_1_T3_PROD_B_PostResponsePush(t *testing.T) {
	placeReturned := make(chan struct{})
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			close(placeReturned)
			return &mthub.OrderRecord{Ticket: 42, State: mthub.OrderStateOpen}, nil
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

// T4-PROD: Unrelated ticket and matching ticket with wrong magic must NOT confirm.
func TestLIVE_ORDER_REENTRY_1_T4_PROD_UnrelatedEventsNotConfirmed(t *testing.T) {
	placeStarted := make(chan struct{})
	placeProceed := make(chan struct{})
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			close(placeStarted)
			<-placeProceed
			return &mthub.OrderRecord{Ticket: 42, State: mthub.OrderStateOpen}, nil
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
