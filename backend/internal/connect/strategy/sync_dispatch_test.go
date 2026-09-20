// sync_dispatch_test.go — VM-LIVE-SYNC-DISPATCH-1 (R1) tests.
//
// Verifies the synchronous signal-dispatch path: VM-emitted signals execute
// through coordinateMutation INSIDE the event and return real broker
// outcomes; confirmed facts are injected into the runner's live state;
// and dispatchResponse skips async re-dispatch (no double-submit).

package strategy

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
	"alphaforge/strategy/runner"
	"alphaforge/strategy/sdk"
)

// T5: confirmed open → real broker ticket returned + position injected
// into the runner's live state (same-event OrderSelect visibility).
func TestSyncDispatch_ConfirmedOpen_RealTicketAndInjection(t *testing.T) {
	rec := &mthub.OrderRecord{
		Ticket: 5678, AccountID: "acct-1", Canonical: "EURUSD",
		Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1), OpenPrice: decimal.NewFromFloat(1.0850),
		State: mthub.OrderStateOpen,
	}
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			return rec, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{rec}, nil
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()
	r := runner.New(runner.Config{Symbol: "EURUSD", Mode: "live"})

	go publishOrderUpdate(broker, cfg.AccountID, 5678, strategyMagic(cfg.ScheduleID), "open")

	ticket, err := srv.dispatchSignalSync(context.Background(), cfg, nil,
		&sdk.Signal{Action: sdk.ActionBuy, Symbol: "EURUSD", Volume: decimal.NewFromFloat(0.1)},
		sess, r)
	if err != nil {
		t.Fatalf("dispatchSignalSync err: %v", err)
	}
	if ticket != 5678 {
		t.Fatalf("ticket=%d, want 5678 (real broker ticket)", ticket)
	}
	// Same-event visibility: the confirmed position must be in live state.
	positions := r.Broker().Positions(0)
	found := false
	for _, p := range positions {
		if p.Ticket == 5678 {
			found = true
		}
	}
	if !found {
		t.Fatal("confirmed position not injected into runner live state — same-event OrderSelect would miss it")
	}
}

// T6: deterministic rejection → error returned, NO position injection.
func TestSyncDispatch_Rejected_NoInjection(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			return nil, errors.New("131 invalid trade volume") // deterministic pre-broker rejection
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()
	r := runner.New(runner.Config{Symbol: "EURUSD", Mode: "live"})

	ticket, err := srv.dispatchSignalSync(context.Background(), cfg, nil,
		&sdk.Signal{Action: sdk.ActionBuy, Symbol: "EURUSD", Volume: decimal.NewFromFloat(0.1)},
		sess, r)
	if err == nil {
		t.Fatal("rejected dispatch must return error — builtin needs it for -1")
	}
	if ticket != 0 {
		t.Fatalf("ticket=%d on rejection, want 0", ticket)
	}
	if n := len(r.Broker().Positions(0)); n != 0 {
		t.Fatalf("rejected open injected %d positions — must inject nothing", n)
	}
}

// T6b: outcome unknown (no push, read-after-write fails) → error, fail-closed.
func TestSyncDispatch_OutcomeUnknown_Error(t *testing.T) {
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

	_, err := srv.dispatchSignalSync(context.Background(), cfg, nil,
		&sdk.Signal{Action: sdk.ActionBuy, Symbol: "EURUSD", Volume: decimal.NewFromFloat(0.1)},
		sess, nil)
	if err == nil {
		t.Fatal("outcome-unknown dispatch must return error (fail-closed), not a ticket")
	}
}

// T7: cancel_all reaches dispatchCancelAll (was silently dropped into
// default no-op before R1 — latent defect B).
func TestSyncDispatch_CancelAll_Dispatched(t *testing.T) {
	cfg0 := testLiveCfg()
	pending := &mthub.OrderRecord{
		Ticket: 9001, AccountID: "acct-1", Canonical: "EURUSD",
		Side: mthub.SideBuy, OrderType: mthub.OrderLimit,
		Volume: decimal.NewFromFloat(0.1), State: mthub.OrderStateOpen,
		Magic: strategyMagic(cfg0.ScheduleID), // ARCH-4 magic filter must match
	}
	fetched := false
	exec := &prodMockExecutor{
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			if !fetched {
				fetched = true
				return []*mthub.OrderRecord{pending}, nil
			}
			return nil, nil // after delete: ticket absent
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	sess := testActiveSess()

	go publishOrderUpdate(broker, cfg0.AccountID, 9001, strategyMagic(cfg0.ScheduleID), "pending_close")

	_, err := srv.dispatchSignalSync(context.Background(), cfg0, nil,
		&sdk.Signal{Action: sdk.ActionCancelAll}, sess, nil)
	if err != nil {
		t.Fatalf("cancel_all dispatch err: %v", err)
	}
	if got := exec.deleteCount.Load(); got != 1 {
		t.Fatalf("DeleteOrder called %d times, want 1 — cancel_all must reach the broker", got)
	}
}

// T9: close_all confirmed → affected tickets removed from runner live
// state (same-event OrdersTotal must not see ghost positions). Mutating
// away affectedTickets propagation turns this RED.
func TestSyncDispatch_CloseAll_RemovesRunnerPositions(t *testing.T) {
	cfg := testLiveCfg()
	open := &mthub.OrderRecord{
		Ticket: 7001, AccountID: "acct-1", Canonical: "EURUSD",
		Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1), State: mthub.OrderStateOpen,
		Magic: strategyMagic(cfg.ScheduleID),
	}
	fetched := false
	exec := &prodMockExecutor{
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			if !fetched {
				fetched = true
				return []*mthub.OrderRecord{open}, nil
			}
			return nil, nil // after close: ticket absent
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	sess := testActiveSess()
	r := runner.New(runner.Config{Symbol: "EURUSD", Mode: "live"})
	r.ApplyConfirmedPosition(sdk.Position{Ticket: 7001, Symbol: "EURUSD", Side: sdk.SideBuy})

	go publishOrderUpdate(broker, cfg.AccountID, 7001, strategyMagic(cfg.ScheduleID), "close")

	_, err := srv.dispatchSignalSync(context.Background(), cfg, nil,
		&sdk.Signal{Action: sdk.ActionCloseAll}, sess, r)
	if err != nil {
		t.Fatalf("close_all dispatch err: %v", err)
	}
	if got := exec.closeCount.Load(); got != 1 {
		t.Fatalf("CloseOrder called %d times, want 1", got)
	}
	if n := len(r.Broker().Positions(0)); n != 0 {
		t.Fatalf("confirmed close_all left %d ghost positions in runner live state", n)
	}
}

// T8: dispatchResponse skips async dispatch for sync-dispatched sessions —
// no double-submit. Mutating the skip (alreadyDispatched ignored) turns
// this RED via placeCount=2.
func TestSyncDispatch_ResponseSkipsDoubleDispatch(t *testing.T) {
	rec := &mthub.OrderRecord{Ticket: 111, AccountID: "acct-1", Canonical: "EURUSD",
		Side: mthub.SideBuy, OrderType: mthub.OrderMarket, State: mthub.OrderStateOpen}
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			return rec, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{rec}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	resp := &antv1.ExecuteLiveResponse{Success: true}
	resp.Signals = []*antv1.StrategySignal{{SignalType: "buy", Volume: "0.1"}}
	resp.Signal = resp.Signals[0]

	// Sync-dispatched session: response signals must NOT dispatch again.
	srv.dispatchResponse(context.Background(), cfg, nil, resp, sess, true)
	if got := exec.placeCount.Load(); got != 0 {
		t.Fatalf("sync-dispatched response dispatched %d more times — double submit!", got)
	}

	// Control: alreadyDispatched=false dispatches once (async path intact).
	srv.dispatchResponse(context.Background(), cfg, nil, resp, sess, false)
	if got := exec.placeCount.Load(); got != 1 {
		t.Fatalf("non-sync response dispatched %d times, want 1", got)
	}
}
