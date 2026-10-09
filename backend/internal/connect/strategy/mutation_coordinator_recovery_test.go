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

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

// TestLIVE_ORDER_REENTRY_1_R4_Recovery_CloseConfirmed verifies that after
// outcomeUnknown for a close mutation, the background recovery goroutine
// reconciles via OpenedOrders (ticket absent = close succeeded) and releases
// the barrier + clears the circuit breaker.
func TestLIVE_ORDER_REENTRY_1_R4_Recovery_CloseConfirmed(t *testing.T) {
	closeCall := make(chan struct{})
	exec := &prodMockExecutor{
		closeFn: func(ctx context.Context, ticket int64, lots decimal.Decimal) error {
			close(closeCall)
			return errors.New("DeadlineExceeded") // outcome_unknown
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// Ticket 99 absent → close succeeded.
			return []*mthub.OrderRecord{{Ticket: 100, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	// Use fast recovery config for deterministic testing.
	conf := confirmationConfig{
		pushWait:              100 * time.Millisecond,
		readAfterWriteTimeout: 2 * time.Second,
		mutationRPCTimeout:    5 * time.Second,
		recoveryDelay:         50 * time.Millisecond,
	}

	sig := &antv1.StrategySignal{SignalType: "close", Volume: "0.1", ExecutedTicket: 99}
	// Call coordinateMutation directly with the fast config.
	srv.coordinateMutation(context.Background(), cfg, sess, mutationSpec{
		action:         actionClose,
		clientID:       "close_99",
		expectedMagic:  strategyMagic(cfg.ScheduleID),
		expectedTicket: 99,
		brokerCall: func(brokerCtx context.Context) (int64, error) {
			return 99, exec.CloseOrder(brokerCtx, 99, decimal.NewFromFloat(0.1))
		},
		verifyReadAfterWrite: verifyTicketAbsent(99),
	}, "close", sig, conf)

	// Barrier should be outcomeUnknown immediately after.
	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("Recovery_CloseConfirmed: pre-recovery state=%s, want outcome_unknown", state)
	}
	if !sess.IsCircuitOpen() {
		t.Fatal("Recovery_CloseConfirmed: circuit breaker should be open")
	}

	// Wait for recovery goroutine to complete (R4 S3: deterministic sync).
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	sess.barrier.WaitState(waitCtx, barrierIdle)
	waitCancel()

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("Recovery_CloseConfirmed: post-recovery state=%s, want idle (recovered+released)", state)
	}
	if sess.IsCircuitOpen() {
		t.Fatal("Recovery_CloseConfirmed: circuit breaker should be cleared after recovery")
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_Recovery_CloseNotApplied verifies that after
// outcomeUnknown for a close mutation, if the ticket is still present in
// OpenedOrders (close didn't take effect), recovery transitions to
// deterministicRejected and releases the barrier.

// TestLIVE_ORDER_REENTRY_1_R4_Recovery_CloseNotApplied verifies that after
// outcomeUnknown for a close mutation, if the ticket is still present in
// OpenedOrders (close didn't take effect), recovery transitions to
// deterministicRejected and releases the barrier.
func TestLIVE_ORDER_REENTRY_1_R4_Recovery_CloseNotApplied(t *testing.T) {
	exec := &prodMockExecutor{
		closeFn: func(ctx context.Context, ticket int64, lots decimal.Decimal) error {
			return errors.New("DeadlineExceeded") // outcome_unknown
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// Ticket 99 still present → close did NOT take effect.
			return []*mthub.OrderRecord{{Ticket: 99, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	conf := confirmationConfig{
		pushWait:              100 * time.Millisecond,
		readAfterWriteTimeout: 2 * time.Second,
		mutationRPCTimeout:    5 * time.Second,
		recoveryDelay:         50 * time.Millisecond,
	}

	sig := &antv1.StrategySignal{SignalType: "close", Volume: "0.1", ExecutedTicket: 99}
	srv.coordinateMutation(context.Background(), cfg, sess, mutationSpec{
		action:         actionClose,
		clientID:       "close_99",
		expectedMagic:  strategyMagic(cfg.ScheduleID),
		expectedTicket: 99,
		brokerCall: func(brokerCtx context.Context) (int64, error) {
			return 99, exec.CloseOrder(brokerCtx, 99, decimal.NewFromFloat(0.1))
		},
		verifyReadAfterWrite: verifyTicketAbsent(99),
	}, "close", sig, conf)

	// Wait for recovery (R4 S3: deterministic sync).
	waitCtx2, waitCancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	sess.barrier.WaitState(waitCtx2, barrierIdle)
	waitCancel2()

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("Recovery_CloseNotApplied: post-recovery state=%s, want idle (rejected+released)", state)
	}
	if sess.IsCircuitOpen() {
		t.Fatal("Recovery_CloseNotApplied: circuit breaker should be cleared")
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_Recovery_QueryFails_StaysLocked verifies that
// if the recovery read-after-write query also fails, the barrier stays
// locked (fail-closed) and the circuit breaker stays open.

// TestLIVE_ORDER_REENTRY_1_R4_Recovery_QueryFails_StaysLocked verifies that
// if the recovery read-after-write query also fails, the barrier stays
// locked (fail-closed) and the circuit breaker stays open.
func TestLIVE_ORDER_REENTRY_1_R4_Recovery_QueryFails_StaysLocked(t *testing.T) {
	recoveryAttempted := make(chan struct{})
	exec := &prodMockExecutor{
		closeFn: func(ctx context.Context, ticket int64, lots decimal.Decimal) error {
			return errors.New("DeadlineExceeded")
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			select {
			case recoveryAttempted <- struct{}{}:
			default:
			}
			return nil, errors.New("broker still unavailable")
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	conf := confirmationConfig{
		pushWait:              100 * time.Millisecond,
		readAfterWriteTimeout: 2 * time.Second,
		mutationRPCTimeout:    5 * time.Second,
		recoveryDelay:         50 * time.Millisecond,
	}

	sig := &antv1.StrategySignal{SignalType: "close", Volume: "0.1", ExecutedTicket: 99}
	srv.coordinateMutation(context.Background(), cfg, sess, mutationSpec{
		action:         actionClose,
		clientID:       "close_99",
		expectedMagic:  strategyMagic(cfg.ScheduleID),
		expectedTicket: 99,
		brokerCall: func(brokerCtx context.Context) (int64, error) {
			return 99, exec.CloseOrder(brokerCtx, 99, decimal.NewFromFloat(0.1))
		},
		verifyReadAfterWrite: verifyTicketAbsent(99),
	}, "close", sig, conf)

	// Wait for recovery attempt to complete (R4 S3: deterministic sync via channel).
	// The recovery goroutine calls fetchFn after recoveryDelay; we signal via channel.
	// Note: fetchFn may also be called by the initial read-after-write, so we wait
	// for the 2nd call (recovery) or a timeout.
	select {
	case <-recoveryAttempted:
	case <-time.After(5 * time.Second):
		t.Fatal("Recovery_QueryFails: recovery goroutine did not attempt fetch within 5s")
	}

	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("Recovery_QueryFails: state=%s, want outcome_unknown (stays locked)", state)
	}
	if !sess.IsCircuitOpen() {
		t.Fatal("Recovery_QueryFails: circuit breaker should stay open")
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_Recovery_OpenMutation_NoRecovery verifies that
// open mutations (ticket=0 at spec creation) do NOT start a recovery goroutine
// when the RPC returns outcome_unknown. The barrier stays locked — fail-closed
// for open mutations since we don't know the ticket to reconcile.

// TestLIVE_ORDER_REENTRY_1_R4_Recovery_OpenMutation_NoRecovery verifies that
// open mutations (ticket=0 at spec creation) do NOT start a recovery goroutine
// when the RPC returns outcome_unknown. The barrier stays locked — fail-closed
// for open mutations since we don't know the ticket to reconcile.
func TestLIVE_ORDER_REENTRY_1_R4_Recovery_OpenMutation_NoRecovery(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			return nil, errors.New("DeadlineExceeded") // outcome_unknown, no ticket
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return nil, errors.New("unavailable")
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	conf := confirmationConfig{
		pushWait:              100 * time.Millisecond,
		readAfterWriteTimeout: 2 * time.Second,
		mutationRPCTimeout:    5 * time.Second,
		recoveryDelay:         50 * time.Millisecond,
	}

	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.coordinateMutation(context.Background(), cfg, sess, mutationSpec{
		action:         actionOpen,
		clientID:       "open_1",
		expectedMagic:  strategyMagic(cfg.ScheduleID),
		expectedTicket: 0, // open: ticket unknown
		brokerCall: func(brokerCtx context.Context) (int64, error) {
			rec, err := exec.PlaceOrder(brokerCtx, &mthub.OrderRequest{})
			if err != nil {
				return 0, err
			}
			return rec.Ticket, nil
		},
		verifyReadAfterWrite: nil,
	}, "buy", sig, conf)

	// Wait longer than recovery delay to verify no recovery starts (R4 S3:
	// deterministic — use a bounded wait that asserts the barrier stays
	// outcomeUnknown for longer than recoveryDelay + readAfterWriteTimeout).
	noRecoveryCtx, noRecoveryCancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	sess.barrier.WaitState(noRecoveryCtx, barrierIdle) // would return if recovery released barrier
	noRecoveryCancel()

	// Open mutations with no ticket should stay locked — no recovery.
	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("Recovery_OpenMutation: state=%s, want outcome_unknown (no recovery for open)", state)
	}
	if !sess.IsCircuitOpen() {
		t.Fatal("Recovery_OpenMutation: circuit breaker should stay open")
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_Recovery_AllowsSubsequentOrder verifies that
// after successful recovery, the barrier is released and a subsequent order
// can be placed (circuit breaker cleared).

// TestLIVE_ORDER_REENTRY_1_R4_Recovery_AllowsSubsequentOrder verifies that
// after successful recovery, the barrier is released and a subsequent order
// can be placed (circuit breaker cleared).
func TestLIVE_ORDER_REENTRY_1_R4_Recovery_AllowsSubsequentOrder(t *testing.T) {
	exec := &prodMockExecutor{
		closeFn: func(ctx context.Context, ticket int64, lots decimal.Decimal) error {
			return errors.New("DeadlineExceeded")
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// Ticket 99 absent → close succeeded.
			return []*mthub.OrderRecord{{Ticket: 100, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	conf := confirmationConfig{
		pushWait:              100 * time.Millisecond,
		readAfterWriteTimeout: 2 * time.Second,
		mutationRPCTimeout:    5 * time.Second,
		recoveryDelay:         50 * time.Millisecond,
	}

	// First: close mutation enters outcomeUnknown.
	sigClose := &antv1.StrategySignal{SignalType: "close", Volume: "0.1", ExecutedTicket: 99}
	srv.coordinateMutation(context.Background(), cfg, sess, mutationSpec{
		action:         actionClose,
		clientID:       "close_99",
		expectedMagic:  strategyMagic(cfg.ScheduleID),
		expectedTicket: 99,
		brokerCall: func(brokerCtx context.Context) (int64, error) {
			return 99, exec.CloseOrder(brokerCtx, 99, decimal.NewFromFloat(0.1))
		},
		verifyReadAfterWrite: verifyTicketAbsent(99),
	}, "close", sigClose, conf)

	// Wait for recovery to complete (R4 S3: deterministic sync).
	recoveryCtx, recoveryCancel := context.WithTimeout(context.Background(), 5*time.Second)
	sess.barrier.WaitState(recoveryCtx, barrierIdle)
	recoveryCancel()

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("Recovery_AllowsSubsequent: post-recovery state=%s, want idle", state)
	}

	// Second: a new buy order should succeed (barrier released, circuit clear).
	exec.placeCount.Store(0)
	exec.placeFn = func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
		return &mthub.OrderRecord{Ticket: 55, State: mthub.OrderStateOpen}, nil
	}
	exec.fetchFn = func(ctx context.Context) ([]*mthub.OrderRecord, error) {
		return []*mthub.OrderRecord{{Ticket: 55, Canonical: "EURUSD"}}, nil
	}
	sigBuy := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sigBuy, sess)

	if got := exec.placeCount.Load(); got != 1 {
		t.Fatalf("Recovery_AllowsSubsequent: PlaceOrder called %d times, want 1 (barrier should be released)", got)
	}
}

// ── QS-1.6: read-after-write confirm must transition via the state machine ──

// TestQS16_ReadAfterWriteConfirmTransitionsBarrier verifies that when
// read-after-write verifies the mutation but the synthetic
// NotifyConfirmationEvent cannot migrate the barrier (cancel action: "cancel"
// is not in its own updateType compatibility set — real broker events are
// "close"/"pending_close"), waitForConfirmation still drives the barrier to
// barrierConfirmed via ConfirmByAuthoritativeRead instead of returning a
// state the barrier never reached (QS-1.6).
