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

// T12 (VM-ERR-CODE-COLLAPSE-1): a broker rejection carrying a numeric code
// must surface as *sdk.BrokerRejectError with the code intact — the VM
// builtin maps it onto GetLastError instead of the blanket 146.
//
// Adversarial (M2): dropping the errors.As translation in
// dispatchSignalSync → err is a generic fmt error → assertion RED.
func TestSyncDispatch_Rejected_CarriesBrokerCode(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			return nil, &mthub.BrokerRejectError{Op: "mt4 OrderSend", Code: 136, Message: "Off quotes"}
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	_, err := srv.dispatchSignalSync(context.Background(), cfg, nil,
		&sdk.Signal{Action: sdk.ActionBuy, Symbol: "EURUSD", Volume: decimal.NewFromFloat(0.1)},
		sess, nil)
	if err == nil {
		t.Fatal("rejected dispatch must return error")
	}
	var bre *sdk.BrokerRejectError
	if !errors.As(err, &bre) || bre == nil {
		t.Fatalf("err type = %T, want *sdk.BrokerRejectError (broker code must survive the boundary)", err)
	}
	if bre.Code != 136 {
		t.Fatalf("BrokerRejectError.Code = %d, want 136 (Off quotes)", bre.Code)
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

// T11: confirmed pending open injects as a pending order with the broker's
// stated type/side — OrderType() must read OP_BUYLIMIT(2), not OP_BUY(0).
// (Live probe caught PlaceOrder replies dropping Side/OrderType → the
// injected pending read as a market position.)
func TestSyncDispatch_ConfirmedPending_InjectsAsPendingOrder(t *testing.T) {
	rec := &mthub.OrderRecord{
		Ticket: 8801, AccountID: "acct-1", Canonical: "EURUSD",
		Side: mthub.SideSell, OrderType: mthub.OrderLimit,
		Volume: decimal.NewFromFloat(0.1), OpenPrice: decimal.NewFromFloat(1.0900),
		State: mthub.OrderStatePending,
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

	go publishOrderUpdate(broker, cfg.AccountID, 8801, strategyMagic(cfg.ScheduleID), "open")

	ticket, err := srv.dispatchSignalSync(context.Background(), cfg, nil,
		&sdk.Signal{Action: sdk.ActionSellLimit, Symbol: "EURUSD",
			Volume: decimal.NewFromFloat(0.1), Price: decimal.NewFromFloat(1.09)},
		sess, r)
	if err != nil {
		t.Fatalf("dispatchSignalSync err: %v", err)
	}
	if ticket != 8801 {
		t.Fatalf("ticket=%d, want 8801", ticket)
	}
	// Pending must land in the pending pool with broker-stated type/side —
	// a zero-value record would inject OrderMarket/SideBuy (OP_BUY readback).
	if n := len(r.Broker().Positions(0)); n != 0 {
		t.Fatalf("pending injected into positions pool: %d positions, want 0", n)
	}
	orders := r.Broker().Orders(0)
	if len(orders) != 1 || orders[0].Ticket != 8801 {
		t.Fatalf("pending orders = %v, want [8801]", orders)
	}
	if orders[0].Type != sdk.OrderLimit || orders[0].Side != sdk.SideSell {
		t.Fatalf("injected pending Type=%v Side=%v, want OrderLimit/SideSell (OP_SELLLIMIT)",
			orders[0].Type, orders[0].Side)
	}
}

// T10: close_all must only CloseOrder market positions — pending orders
// require DeleteOrder (cancel_all), and CloseOrder on a pending is a
// deterministic broker rejection (wasted RPC + rejection audit noise).
func TestSyncDispatch_CloseAll_SkipsPendingOrders(t *testing.T) {
	cfg := testLiveCfg()
	market := &mthub.OrderRecord{
		Ticket: 7001, AccountID: "acct-1", Canonical: "EURUSD",
		Side: mthub.SideBuy, OrderType: mthub.OrderMarket,
		Volume: decimal.NewFromFloat(0.1), State: mthub.OrderStateOpen,
		Magic: strategyMagic(cfg.ScheduleID),
	}
	pending := &mthub.OrderRecord{
		Ticket: 7002, AccountID: "acct-1", Canonical: "EURUSD",
		Side: mthub.SideBuy, OrderType: mthub.OrderLimit,
		Volume: decimal.NewFromFloat(0.1), State: mthub.OrderStateOpen,
		Magic: strategyMagic(cfg.ScheduleID),
	}
	fetched := false
	var closedTickets []int64
	exec := &prodMockExecutor{
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			if !fetched {
				fetched = true
				return []*mthub.OrderRecord{market, pending}, nil
			}
			return []*mthub.OrderRecord{pending}, nil // market closed, pending remains
		},
		closeFn: func(ctx context.Context, ticket int64, lots decimal.Decimal) error {
			closedTickets = append(closedTickets, ticket)
			return nil
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	sess := testActiveSess()

	go publishOrderUpdate(broker, cfg.AccountID, 7001, strategyMagic(cfg.ScheduleID), "close")

	_, err := srv.dispatchSignalSync(context.Background(), cfg, nil,
		&sdk.Signal{Action: sdk.ActionCloseAll}, sess, nil)
	if err != nil {
		t.Fatalf("close_all dispatch err: %v", err)
	}
	if got := exec.closeCount.Load(); got != 1 {
		t.Fatalf("CloseOrder called %d times, want 1 — close_all must not CloseOrder pending ticket 7002", got)
	}
	for _, tk := range closedTickets {
		if tk == 7002 {
			t.Fatalf("CloseOrder called on pending order 7002 — pendings require DeleteOrder via cancel_all")
		}
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
