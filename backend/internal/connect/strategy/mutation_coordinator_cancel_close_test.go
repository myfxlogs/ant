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

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

// MUTATION-CLOSE: CloseOrder timeout → outcome_unknown, barrier locked.
func TestLIVE_ORDER_REENTRY_1_MUTATION_CLOSE_TimeoutStaysLocked(t *testing.T) {
	exec := &prodMockExecutor{
		closeFn: func(ctx context.Context, ticket int64, lots decimal.Decimal) error {
			return &mthub.MutationError{Phase: mthub.PhaseBroker, Cause: errors.New("context deadline exceeded")}
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{Ticket: 123, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "close", Volume: "0", ExecutedTicket: 123}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("MUTATION-CLOSE: barrier state=%s, want outcome_unknown", state)
	}
}

// MUTATION-MODIFY: Modify goes through coordinator (not just Acquire+defer Release).
// R5: read-after-write verifies SL/TP match the requested values.

// MUTATION-CANCEL: Cancel goes through coordinator (not just Acquire+defer Release).
func TestLIVE_ORDER_REENTRY_1_MUTATION_CANCEL_Wiring(t *testing.T) {
	exec := &prodMockExecutor{
		deleteFn: func(ctx context.Context, ticket int64) error {
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// Ticket 123 is absent → cancel confirmed.
			return nil, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "cancel", Volume: "0", ExecutedTicket: 123}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("MUTATION-CANCEL: barrier state=%s, want idle (confirmed+released)", state)
	}
	if got := exec.deleteCount.Load(); got != 1 {
		t.Fatalf("MUTATION-CANCEL: DeleteOrder called %d times, want 1", got)
	}
}

// R3-ADVERSARIAL: Incompatible updateType must NOT confirm a mutation.
// A "modify" event with matching ticket+magic must NOT confirm a "close" action.

// R7b: A deterministic rejection in close_all must NOT be counted as "closed".
// Uses a zaptest observer to capture the "dispatchCloseAll complete" log and
// verify the `closed` field only includes confirmed closes.
// Setup: kill switch engaged → both closes are deterministically rejected.
// R7b fix: closed=0 (only confirmed). Old code: closed=2 (rejected counted).
func TestLIVE_ORDER_REENTRY_1_R7b_CloseAllDoesNotCountRejectedAsClosed(t *testing.T) {
	cfg := testLiveCfg()
	expectedMagic := strategyMagic(cfg.ScheduleID)
	exec := &prodMockExecutor{
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{
				{Ticket: 100, Canonical: "EURUSD", Magic: expectedMagic, Volume: decimal.NewFromFloat(0.1)},
				{Ticket: 200, Canonical: "EURUSD", Magic: expectedMagic, Volume: decimal.NewFromFloat(0.1)},
			}, nil
		},
	}
	srv, svc, _ := testCoordinatorSetup(exec)
	// Kill switch engaged from the start → both CloseOrder calls rejected
	// as deterministic_rejected (pre-broker).
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})

	sess := testActiveSess()

	// Use a zaptest observer to capture the completion log.
	core, recorded := observer.New(zap.InfoLevel)
	srv.log = zap.New(core)

	srv.dispatchCloseAll(context.Background(), cfg, sess)

	// Find the "dispatchCloseAll complete" log entry.
	for _, entry := range recorded.All() {
		if entry.Message == "LiveStrategyRunner: dispatchCloseAll complete" {
			closedField := entry.ContextMap()["closed"]
			// zap.Int stores as int64 in the observer's ContextMap.
			closed, ok := toInt(closedField)
			if !ok {
				t.Fatalf("R7b: 'closed' field not int, got %T", closedField)
			}
			// Both closes were rejected (kill switch) → closed must be 0.
			// R7b: only barrierConfirmed counts. Old code counted
			// deterministicRejected too → would be 2.
			if closed != 0 {
				t.Fatalf("R7b: closed=%d, want 0 (both rejected — only confirmed counts)", closed)
			}
			return
		}
	}
	t.Fatal("R7b: dispatchCloseAll complete log not found")
}

// ── R5-⑤: verifyTicketModified distinguishes unspecified from explicit zero ──

// R5-⑤-A: Explicit SL="0" (clearing stop loss) — broker returns SL=0 → confirmed.
