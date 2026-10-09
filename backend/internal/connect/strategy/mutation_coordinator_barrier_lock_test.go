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
	"sync/atomic"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

// T6-PROD: Transport timeout / unknown error → outcome_unknown, barrier locked.
func TestLIVE_ORDER_REENTRY_1_T6_PROD_TransportTimeoutStaysLocked(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			return nil, &mthub.MutationError{Phase: mthub.PhaseBroker, Cause: errors.New("context deadline exceeded")}
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

// T7-PROD: No push, single read-after-write OpenedOrders succeeds.
func TestLIVE_ORDER_REENTRY_1_T7_PROD_ReadAfterWriteSucceeds(t *testing.T) {
	var fetchCount atomic.Int64
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			return &mthub.OrderRecord{Ticket: 42, State: mthub.OrderStateOpen}, nil
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

// T7-FAIL: Read-after-write fails → outcome_unknown, barrier locked.
func TestLIVE_ORDER_REENTRY_1_T7_FAIL_ReadAfterWriteFails(t *testing.T) {
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
		t.Fatalf("T7-FAIL: barrier state=%s, want outcome_unknown", state)
	}
}

// T8-REPLAY: Retained positions with old PositionsCapturedAt must NOT be fresh,
// even if receivedAt is recent. R2: zero PositionsCapturedAt = fail-closed.
// This test reproduces the real replay scenario: an old retained snapshot
// is received NOW (new receivedAt) but has old PositionsCapturedAt.

// T8-REPLAY: Retained positions with old PositionsCapturedAt must NOT be fresh,
// even if receivedAt is recent. R2: zero PositionsCapturedAt = fail-closed.
// This test reproduces the real replay scenario: an old retained snapshot
// is received NOW (new receivedAt) but has old PositionsCapturedAt.
func TestLIVE_ORDER_REENTRY_1_T8_REPLAY_StalePositionsNotFresh(t *testing.T) {
	pc := NewPositionCache(zap.NewNop())
	oldTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	// R2 test A: old PositionsCapturedAt + new receivedAt → must fail.
	snap := &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(12000),
		FinancialsAuthoritative: true, FinancialsSource: "account_summary",
		CapturedAt:             now, // financials are fresh
		PositionsAuthoritative: true,
		PositionsCapturedAt:    oldTime, // positions are stale (old captured-at)
		PositionsSource:        "order_stream",
		Positions:              []mthub.PositionSnapshotItem{{Ticket: 1}},
	}
	pc.PutSnapshot(snap, now) // received NOW — new receivedAt

	// GetFreshTradingSnapshot must fail because PositionsCapturedAt is old.
	_, ok := pc.GetFreshTradingSnapshot("acct-1", now)
	if ok {
		t.Fatal("T8-REPLAY: GetFreshTradingSnapshot returned true for stale PositionsCapturedAt + new receivedAt — R2 violated")
	}

	// GetFreshPositionSnapshot must also fail.
	_, ok = pc.GetFreshPositionSnapshot("acct-1", now)
	if ok {
		t.Fatal("T8-REPLAY: GetFreshPositionSnapshot returned true for stale PositionsCapturedAt — R2 violated")
	}

	// R2 test B: zero PositionsCapturedAt + new receivedAt → must fail (fail-closed).
	pc2 := NewPositionCache(zap.NewNop())
	snap2 := &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(12000),
		FinancialsAuthoritative: true, FinancialsSource: "account_summary",
		CapturedAt:             now,
		PositionsAuthoritative: true,
		PositionsCapturedAt:    time.Time{}, // zero — no provenance
		PositionsSource:        "",
		Positions:              []mthub.PositionSnapshotItem{{Ticket: 1}},
	}
	pc2.PutSnapshot(snap2, now)
	_, ok = pc2.GetFreshTradingSnapshot("acct-1", now)
	if ok {
		t.Fatal("T8-REPLAY: GetFreshTradingSnapshot returned true for zero PositionsCapturedAt — R2 fail-closed violated")
	}
	_, ok = pc2.GetFreshPositionSnapshot("acct-1", now)
	if ok {
		t.Fatal("T8-REPLAY: GetFreshPositionSnapshot returned true for zero PositionsCapturedAt — R2 fail-closed violated")
	}
}

// T9: Position-only update does NOT refresh financials; financial-only does NOT refresh positions.
