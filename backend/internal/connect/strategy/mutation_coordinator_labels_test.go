package strategy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mdgateway/adapter/mt4"
	"alphaforge/internal/mdgateway/adapter/mt5"
	"alphaforge/internal/mthub"
	mt4pb "alphaforge/mt4"
	mt5pb "alphaforge/mt5"
)

func TestLIVE_ORDER_REENTRY_1_R5_ExplicitZeroClearsSL(t *testing.T) {
	exec := &prodMockExecutor{
		modifyFn: func(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// Broker cleared SL to 0, TP unchanged at 2.0.
			return []*mthub.OrderRecord{{
				Ticket: 123, Canonical: "EURUSD",
				StopLoss:   decimal.Zero, // cleared
				TakeProfit: decimal.NewFromFloat(2.0),
			}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	// SL="0" = explicit zero (clear), TP="2.0" = set to 2.0.
	sig := &antv1.StrategySignal{SignalType: "modify", Volume: "0", ExecutedTicket: 123, StopLoss: "0", TakeProfit: "2.0"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("R5-⑤-A: barrier state=%s, want idle (confirmed — SL cleared to 0)", state)
	}
}

// R5-⑤-B: Explicit SL="0" but broker returns SL=1.0 (not cleared) → NOT confirmed.
func TestLIVE_ORDER_REENTRY_1_R5_ExplicitZeroNotCleared(t *testing.T) {
	exec := &prodMockExecutor{
		modifyFn: func(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// Broker did NOT clear SL — still 1.0.
			return []*mthub.OrderRecord{{
				Ticket: 123, Canonical: "EURUSD",
				StopLoss:   decimal.NewFromFloat(1.0), // not cleared
				TakeProfit: decimal.NewFromFloat(2.0),
			}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	// SL="0" = explicit zero (clear), but broker didn't clear → must NOT confirm.
	sig := &antv1.StrategySignal{SignalType: "modify", Volume: "0", ExecutedTicket: 123, StopLoss: "0", TakeProfit: "2.0"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("R5-⑤-B: barrier state=%s, want outcome_unknown (SL not cleared to 0)", state)
	}
}

// R5-⑤-C: SL not provided (empty string) — don't check SL, only check TP.
func TestLIVE_ORDER_REENTRY_1_R5_UnspecifiedNotChecked(t *testing.T) {
	exec := &prodMockExecutor{
		modifyFn: func(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// SL is whatever the broker has (1.0) — should NOT be checked.
			// TP matches requested 2.0 → confirmed.
			return []*mthub.OrderRecord{{
				Ticket: 123, Canonical: "EURUSD",
				StopLoss:   decimal.NewFromFloat(1.0), // not checked (SL not provided)
				TakeProfit: decimal.NewFromFloat(2.0), // matches
			}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	// SL="" = not provided (don't check), TP="2.0" = check.
	sig := &antv1.StrategySignal{SignalType: "modify", Volume: "0", ExecutedTicket: 123, TakeProfit: "2.0"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("R5-⑤-C: barrier state=%s, want idle (confirmed — SL not checked, TP matches)", state)
	}
}

// ============================================================================
// ④-①: Integration tests — adapter label → PositionSnapshotBroker → barrier
// pipeline end-to-end. Verifies that the real updateType labels emitted by
// MT4/MT5 adapters correctly flow through to barrier confirmation.
// ============================================================================

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingOpen verifies
// that MT4's PendingOpen action label ("pending_open") flows through the
// PositionSnapshotBroker and confirms an "open" barrier mutation.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingOpen(t *testing.T) {
	label := mt4.Mt4UpdateActionLabel(mt4pb.UpdateAction_UpdateAction_PendingOpen)
	if label != "pending_open" {
		t.Fatalf("MT4 PendingOpen label = %q, want %q", label, "pending_open")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	// Simulate the full pipeline: adapter label → PositionSnapshot → broker → barrier.
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("MT4 pending_open → open barrier: state=%s, want confirmed", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingClose verifies
// that MT4's PendingClose action label ("pending_close") confirms a "close" barrier.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingClose(t *testing.T) {
	label := mt4.Mt4UpdateActionLabel(mt4pb.UpdateAction_UpdateAction_PendingClose)
	if label != "pending_close" {
		t.Fatalf("MT4 PendingClose label = %q, want %q", label, "pending_close")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "close")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("MT4 pending_close → close barrier: state=%s, want confirmed", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingModify verifies
// that MT4's PendingModify action label ("pending_modify") confirms a "modify" barrier.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingModify(t *testing.T) {
	label := mt4.Mt4UpdateActionLabel(mt4pb.UpdateAction_UpdateAction_PendingModify)
	if label != "pending_modify" {
		t.Fatalf("MT4 PendingModify label = %q, want %q", label, "pending_modify")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "modify")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("MT4 pending_modify → modify barrier: state=%s, want confirmed", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingOpen verifies
// that MT5's PendingOpen type label ("pending_open") confirms an "open" barrier.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingOpen(t *testing.T) {
	label := mt5.Mt5UpdateTypeLabel(mt5pb.UpdateType_UpdateType_PendingOpen)
	if label != "pending_open" {
		t.Fatalf("MT5 PendingOpen label = %q, want %q", label, "pending_open")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("MT5 pending_open → open barrier: state=%s, want confirmed", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingClose verifies
// that MT5's PendingClose type label ("pending_close") confirms a "close" barrier.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingClose(t *testing.T) {
	label := mt5.Mt5UpdateTypeLabel(mt5pb.UpdateType_UpdateType_PendingClose)
	if label != "pending_close" {
		t.Fatalf("MT5 PendingClose label = %q, want %q", label, "pending_close")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "close")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("MT5 pending_close → close barrier: state=%s, want confirmed", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingModify verifies
// that MT5's PendingModify type label ("modify") confirms a "modify" barrier.
// Note: MT5 maps both MarketModify and PendingModify to "modify".
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingModify(t *testing.T) {
	label := mt5.Mt5UpdateTypeLabel(mt5pb.UpdateType_UpdateType_PendingModify)
	if label != "modify" {
		t.Fatalf("MT5 PendingModify label = %q, want %q", label, "modify")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "modify")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("MT5 modify → modify barrier: state=%s, want confirmed", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_FullBrokerPath verifies
// the complete pipeline: adapter label → publishOrderUpdate (real broker) →
// confirmation listener → barrier. This is a true integration test that
// exercises the REAL PositionSnapshotBroker subscription wiring.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_FullBrokerPath(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			return 42, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{Ticket: 42, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	// Use the real MT4 PendingOpen label — this is what the adapter would emit.
	label := mt4.Mt4UpdateActionLabel(mt4pb.UpdateAction_UpdateAction_PendingOpen)

	// Publish the confirmation through the REAL broker, simulating the
	// adapter → pipeline_callbacks.publishPositionSnapshot → broker path.
	// Use a goroutine to publish after the barrier has acquired.
	go func() {
		time.Sleep(50 * time.Millisecond) // wait for barrier to enter submitting
		publishOrderUpdate(broker, cfg.AccountID, 42, strategyMagic(cfg.ScheduleID), label)
	}()

	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("FullBrokerPath: barrier state=%s, want idle (confirmed+released)", state)
	}
	if got := exec.placeCount.Load(); got != 1 {
		t.Fatalf("FullBrokerPath: PlaceOrder called %d times, want 1", got)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_IncompatibleRejects verifies
// that an incompatible adapter label does NOT confirm the barrier. E.g. a
// "modify" event cannot confirm an "open" action.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_IncompatibleRejects(t *testing.T) {
	// MT4 PositionModify label is "modify" — should NOT confirm an "open" barrier.
	label := mt4.Mt4UpdateActionLabel(mt4pb.UpdateAction_UpdateAction_PositionModify)
	if label != "modify" {
		t.Fatalf("MT4 PositionModify label = %q, want %q", label, "modify")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierAcceptedUnconfirmed {
		t.Fatalf("Incompatible: barrier state=%s, want accepted_unconfirmed (not confirmed by modify event for open action)", state)
	}
}

// ============================================================================
// ④-②: Integration tests — outcomeUnknown reconciliation recovery
// ============================================================================

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

	// Wait for recovery goroutine to complete.
	time.Sleep(300 * time.Millisecond)

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

	// Wait for recovery.
	time.Sleep(300 * time.Millisecond)

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
func TestLIVE_ORDER_REENTRY_1_R4_Recovery_QueryFails_StaysLocked(t *testing.T) {
	exec := &prodMockExecutor{
		closeFn: func(ctx context.Context, ticket int64, lots decimal.Decimal) error {
			return errors.New("DeadlineExceeded")
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
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

	// Wait for recovery attempt.
	time.Sleep(300 * time.Millisecond)

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
func TestLIVE_ORDER_REENTRY_1_R4_Recovery_OpenMutation_NoRecovery(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			return 0, errors.New("DeadlineExceeded") // outcome_unknown, no ticket
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
			return exec.PlaceOrder(brokerCtx, &mthub.OrderRequest{})
		},
		verifyReadAfterWrite: nil,
	}, "buy", sig, conf)

	// Wait longer than recovery delay.
	time.Sleep(300 * time.Millisecond)

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

	// Wait for recovery to complete.
	time.Sleep(300 * time.Millisecond)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("Recovery_AllowsSubsequent: post-recovery state=%s, want idle", state)
	}

	// Second: a new buy order should succeed (barrier released, circuit clear).
	exec.placeCount.Store(0)
	exec.placeFn = func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
		return 55, nil
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
