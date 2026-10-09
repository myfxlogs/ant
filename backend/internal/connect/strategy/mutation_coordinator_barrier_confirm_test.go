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
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

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
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			return nil, fmt.Errorf("%w: mt4 OrderSend: code=130 msg=Invalid S/L or T/P", mthub.ErrBrokerRejected)
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

// T9: Position-only update does NOT refresh financials; financial-only does NOT refresh positions.
func TestLIVE_ORDER_REENTRY_1_T9_ProvenanceSeparation(t *testing.T) {
	pc := NewPositionCache(zap.NewNop())
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Initial: both fresh.
	snap := &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(12000),
		FinancialsAuthoritative: true, FinancialsSource: "account_summary",
		CapturedAt:             baseTime,
		PositionsAuthoritative: true,
		PositionsCapturedAt:    baseTime,
		PositionsSource:        "order_stream",
		Positions:              []mthub.PositionSnapshotItem{{Ticket: 1}},
	}
	pc.PutSnapshot(snap, baseTime)

	// Financial-only refresh at baseTime+10s.
	finTime := baseTime.Add(10 * time.Second)
	finRefresh := &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(11000), Equity: decimal.NewFromInt(13000),
		FinancialsAuthoritative: true, FinancialsSource: "account_summary",
		CapturedAt: finTime,
		// PositionsAuthoritative=false → financial-only.
	}
	pc.PutSnapshot(finRefresh, finTime)

	// Financials should be fresh at finTime+1s.
	now := finTime.Add(1 * time.Second)
	finSnap, ok := pc.GetFreshFinancialSnapshot("acct-1", now)
	if !ok || !finSnap.Equity.Equal(decimal.NewFromInt(13000)) {
		t.Fatalf("T9: financial refresh should update financials: ok=%v equity=%s", ok, finSnap.Equity)
	}
	// Positions should still be stale (captured at baseTime, now is finTime+1s > 90s from baseTime? No, 11s).
	// Actually baseTime+11s is within 90s, so positions are still fresh from the initial snapshot.
	// Let's test with a time past max age for positions but fresh for financials.
	staleNow := baseTime.Add(AccountSnapshotMaxAge + time.Second)
	_, posOk := pc.GetFreshPositionSnapshot("acct-1", staleNow)
	if posOk {
		t.Fatal("T9: financial-only refresh must NOT make old positions fresh at staleNow")
	}
	// Financials should still be fresh at staleNow (finTime is baseTime+10s, staleNow is baseTime+91s → 81s < 90s).
	_, finOk := pc.GetFreshFinancialSnapshot("acct-1", staleNow)
	if !finOk {
		t.Fatal("T9: financial refresh should keep financials fresh at staleNow")
	}
}

// T10-REQUEST: dispatch constructs OrderRequest.Magic = StrategyMagic(scheduleID).

// T10-REQUEST: dispatch constructs OrderRequest.Magic = StrategyMagic(scheduleID).
func TestLIVE_ORDER_REENTRY_1_T10_REQUEST_MagicInOrderRequest(t *testing.T) {
	var capturedMagic int32
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			capturedMagic = req.Magic
			return &mthub.OrderRecord{Ticket: 1, State: mthub.OrderStateOpen}, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{Ticket: 1, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	expected := strategyMagic(cfg.ScheduleID)
	if capturedMagic != expected {
		t.Fatalf("T10-REQUEST: OrderRequest.Magic=%d, want StrategyMagic=%d", capturedMagic, expected)
	}
}

// MUTATION-CLOSE: CloseOrder timeout → outcome_unknown, barrier locked.

// R3-ADVERSARIAL: Incompatible updateType must NOT confirm a mutation.
// A "modify" event with matching ticket+magic must NOT confirm a "close" action.
func TestLIVE_ORDER_REENTRY_1_R3_IncompatibleUpdateTypeNotConfirmed(t *testing.T) {
	placeStarted := make(chan struct{})
	placeProceed := make(chan struct{})
	exec := &prodMockExecutor{
		closeFn: func(ctx context.Context, ticket int64, lots decimal.Decimal) error {
			close(placeStarted)
			<-placeProceed
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// Ticket 42 still present → close NOT confirmed by read-after-write.
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
		sig := &antv1.StrategySignal{SignalType: "close", Volume: "0", ExecutedTicket: 42}
		srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
	}()

	<-placeStarted
	// Push a "modify" event with matching ticket+magic — must NOT confirm close.
	publishOrderUpdate(broker, cfg.AccountID, 42, magic, "modify")
	close(placeProceed)
	<-done

	// Barrier should be outcome_unknown (close not confirmed by incompatible event
	// or by read-after-write which shows ticket still present).
	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("R3: barrier state=%s, want outcome_unknown (incompatible updateType must not confirm)", state)
	}
}

// R5-ADVERSARIAL: Modify read-after-write must verify SL/TP actually changed.
// If the order's SL/TP don't match the requested values, it's NOT confirmed.

// TestQS16_ReadAfterWriteConfirmTransitionsBarrier verifies that when
// read-after-write verifies the mutation but the synthetic
// NotifyConfirmationEvent cannot migrate the barrier (cancel action: "cancel"
// is not in its own updateType compatibility set — real broker events are
// "close"/"pending_close"), waitForConfirmation still drives the barrier to
// barrierConfirmed via ConfirmByAuthoritativeRead instead of returning a
// state the barrier never reached (QS-1.6).
func TestQS16_ReadAfterWriteConfirmTransitionsBarrier(t *testing.T) {
	exec := &prodMockExecutor{
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// Ticket 123 absent → cancel verified by authoritative read.
			return nil, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	barrier := NewTradeBarrier(zap.NewNop())
	magic := strategyMagic(cfg.ScheduleID)

	// Drive the barrier to acceptedUnconfirmed for a cancel mutation — the
	// state waitForConfirmation sees after NotifyBrokerAccepted.
	if !barrier.Acquire("cancel_123", magic, string(actionCancel)) {
		t.Fatal("Acquire failed")
	}
	barrier.NotifyBrokerAccepted(123)
	if state := barrier.State(); state != barrierAcceptedUnconfirmed {
		t.Fatalf("setup: state=%s, want accepted_unconfirmed", state)
	}

	conf := confirmationConfig{
		pushWait:              50 * time.Millisecond,
		readAfterWriteTimeout: 2 * time.Second,
		mutationRPCTimeout:    5 * time.Second,
		recoveryDelay:         50 * time.Millisecond,
	}
	got := srv.waitForConfirmation(context.Background(), cfg, barrier, 123, magic,
		actionCancel, verifyTicketAbsent(123), conf)
	if got != barrierConfirmed {
		t.Fatalf("waitForConfirmation returned %s, want confirmed", got)
	}
	// QS-1.6 core assertion: the barrier itself must BE confirmed (state
	// machine migrated), not merely reported as confirmed by the coordinator.
	if state := barrier.State(); state != barrierConfirmed {
		t.Fatalf("barrier state=%s after waitForConfirmation, want confirmed — coordinator must not report a state the barrier never reached", state)
	}
}

// TestQS16_ReadAfterWriteConfirmTransitionsBarrier_ZeroTicket covers the
// second root-cause branch: NotifyConfirmationEvent early-returns on
// ticket==0 (trade_barrier.go:219). An open mutation whose broker RPC
// returned ticket=0 can never migrate via the push-event path — the
// authoritative read must transition the barrier via the state machine.

// TestQS16_ReadAfterWriteConfirmTransitionsBarrier_ZeroTicket covers the
// second root-cause branch: NotifyConfirmationEvent early-returns on
// ticket==0 (trade_barrier.go:219). An open mutation whose broker RPC
// returned ticket=0 can never migrate via the push-event path — the
// authoritative read must transition the barrier via the state machine.
func TestQS16_ReadAfterWriteConfirmTransitionsBarrier_ZeroTicket(t *testing.T) {
	exec := &prodMockExecutor{
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// An order record with Ticket=0 → verifyTicketPresent(0) = true.
			return []*mthub.OrderRecord{{Ticket: 0, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	barrier := NewTradeBarrier(zap.NewNop())
	magic := strategyMagic(cfg.ScheduleID)

	if !barrier.Acquire("open_0", magic, string(actionOpen)) {
		t.Fatal("Acquire failed")
	}
	barrier.NotifyBrokerAccepted(0)
	if state := barrier.State(); state != barrierAcceptedUnconfirmed {
		t.Fatalf("setup: state=%s, want accepted_unconfirmed", state)
	}

	conf := confirmationConfig{
		pushWait:              50 * time.Millisecond,
		readAfterWriteTimeout: 2 * time.Second,
		mutationRPCTimeout:    5 * time.Second,
		recoveryDelay:         50 * time.Millisecond,
	}
	got := srv.waitForConfirmation(context.Background(), cfg, barrier, 0, magic,
		actionOpen, verifyTicketPresent(0), conf)
	if got != barrierConfirmed {
		t.Fatalf("waitForConfirmation returned %s, want confirmed", got)
	}
	if state := barrier.State(); state != barrierConfirmed {
		t.Fatalf("barrier state=%s after waitForConfirmation, want confirmed (ticket==0 early-return bypassed)", state)
	}
}
