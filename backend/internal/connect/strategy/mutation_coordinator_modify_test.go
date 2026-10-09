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
	"testing"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

// MUTATION-MODIFY: Modify goes through coordinator (not just Acquire+defer Release).
// R5: read-after-write verifies SL/TP match the requested values.
func TestLIVE_ORDER_REENTRY_1_MUTATION_MODIFY_Wiring(t *testing.T) {
	exec := &prodMockExecutor{
		modifyFn: func(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// R5: return order with SL/TP matching the requested values.
			return []*mthub.OrderRecord{{
				Ticket: 123, Canonical: "EURUSD",
				StopLoss:   decimal.NewFromFloat(1.0),
				TakeProfit: decimal.NewFromFloat(2.0),
			}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "modify", Volume: "0", ExecutedTicket: 123, StopLoss: "1.0", TakeProfit: "2.0"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("MUTATION-MODIFY: barrier state=%s, want idle (confirmed+released)", state)
	}
	if got := exec.modifyCount.Load(); got != 1 {
		t.Fatalf("MUTATION-MODIFY: ModifyOrder called %d times, want 1", got)
	}
}

// MUTATION-CANCEL: Cancel goes through coordinator (not just Acquire+defer Release).

// R5-ADVERSARIAL: Modify read-after-write must verify SL/TP actually changed.
// If the order's SL/TP don't match the requested values, it's NOT confirmed.
func TestLIVE_ORDER_REENTRY_1_R5_ModifyVerifySLTPChanged(t *testing.T) {
	exec := &prodMockExecutor{
		modifyFn: func(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// Ticket 123 present but SL/TP are OLD values (not the requested 1.0/2.0).
			return []*mthub.OrderRecord{{
				Ticket: 123, Canonical: "EURUSD",
				StopLoss:   decimal.NewFromFloat(0.5), // old SL, not 1.0
				TakeProfit: decimal.NewFromFloat(1.5), // old TP, not 2.0
			}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "modify", Volume: "0", ExecutedTicket: 123, StopLoss: "1.0", TakeProfit: "2.0"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	// R5: modify not confirmed because SL/TP don't match → outcome_unknown.
	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("R5: barrier state=%s, want outcome_unknown (SL/TP not changed)", state)
	}
}

// R5-POSITIVE: Modify read-after-write with matching SL/TP → confirmed.

// R5-POSITIVE: Modify read-after-write with matching SL/TP → confirmed.
func TestLIVE_ORDER_REENTRY_1_R5_ModifyMatchingSLTPConfirmed(t *testing.T) {
	exec := &prodMockExecutor{
		modifyFn: func(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{
				Ticket: 123, Canonical: "EURUSD",
				StopLoss:   decimal.NewFromFloat(1.0), // matches requested
				TakeProfit: decimal.NewFromFloat(2.0), // matches requested
			}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "modify", Volume: "0", ExecutedTicket: 123, StopLoss: "1.0", TakeProfit: "2.0"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("R5-POSITIVE: barrier state=%s, want idle (confirmed+released)", state)
	}
}

// R4-BOUNDED: Event cache must be bounded — sending >maxEventCacheEntries
// unrelated events must not cause unbounded growth. R4 rework: assert
// len(eventCache) <= maxEventCacheEntries AND FIFO eviction (oldest evicted
// first, newest retained). The previous test only checked "no panic" which
// passed even with the eviction code deleted.

// R5-⑤-A: Explicit SL="0" (clearing stop loss) — broker returns SL=0 → confirmed.
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
