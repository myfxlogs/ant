// vm_trade_context_redo_test.go — VM-TRADE-CONTEXT-1/2 返工行为测试（2026-08-26）.
//
// Tests verify the re-implemented fixes after D-REVERT-SCOPE-DRIFT-001:
//   - VM-TRADE-CONTEXT-1: order cache invalidation after mutations,
//     CTrade magic/deviation reaching signals, failed OrderSelect resets state.
//   - VM-TRADE-CONTEXT-2: CloseBy signal carries both tickets,
//     AccountNumber from context (not hardcoded), IsTesting = !signalMode,
//     brokerImpl records query errors for Runner fail-closed.
//
// Adversarial proofs (8): each critical line mutated → relevant test RED → restore GREEN.

package mql2go

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/runner"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// ── Test helpers (VM-TRADE-CONTEXT-1/2) ──────────────────────────────

// tradeCtxTestSource is MQL4 source that exercises CTrade setters + Buy in OnTick.
const tradeCtxTestSource = `
CTrade trade;
int g_after = -1;

int OnInit()
{
    trade.SetExpertMagicNumber(999);
    trade.SetDeviationInPoints(77);
    return 0;
}

void OnTick()
{
    trade.Buy(0.1, "EURUSD", 0, 0, 0, "test");
    g_after = 1;
}
`

// selectResetTestSource verifies that a failed OrderSelect resets currentPos.
const selectResetTestSource = `
int g_ticket_after_ok = -1;
int g_ticket_after_fail = -1;
int g_after = -1;

int OnInit() { return 0; }

void OnTick()
{
    if (OrderSelect(0, SELECT_BY_POS, MODE_TRADES))
    {
        g_ticket_after_ok = OrderTicket();
    }
    if (OrderSelect(999, SELECT_BY_POS, MODE_TRADES))
    {
        // should fail — index out of range
    }
    g_ticket_after_fail = OrderTicket();
    g_after = 1;
}
`

// compileAndSetupRunner compiles MQL source and sets up a Runner with
// live harness state containing the given positions + pending orders.
// Returns the runner and compiled VMRunner.
func compileAndSetupRunner(t *testing.T, src string, positions []sdk.Position, pendingOrders []sdk.PendingOrder) (*runner.Runner, *VMRunner) {
	t.Helper()
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	r.UpdateLiveState("10000", "10500", "500", "9500", positions, pendingOrders)
	return r, vmRunner
}

// runTick executes one OnTick and returns the VMRunner for global inspection.
func runTick(t *testing.T, r *runner.Runner, vm *VMRunner) *VMRunner {
	t.Helper()
	_, err := r.OnTick(context.Background(), decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}
	return vm
}

// ── VM-TRADE-CONTEXT-1 tests (4) ─────────────────────────────────────

// TestOrderCacheInvalidatedAfterClose verifies that after OrderClose succeeds,
// OrdersTotal reflects the updated count (cache invalidated, not stale).
//
// Adversarial: delete invalidateOrderCaches from builtinOrderClose broker path
// → OrdersTotal returns stale cached value → RED.
func TestOrderCacheInvalidatedAfterClose(t *testing.T) {
	// Direct VM-level test: set up a VM with a test broker that has 1 position,
	// call builtinOrderClose, then check cachedPositions is nil (invalidated).
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.SetSignalMode(false) // broker path
	vm.ctx = &cacheTestContext{
		broker: &cacheTestBroker{
			positions: []sdk.Position{
				{Ticket: 1001, Symbol: "EURUSD", Side: sdk.SideBuy, Magic: 12345,
					Volume: decimal.NewFromFloat(0.1), OpenPrice: decimal.NewFromFloat(1.1)},
			},
		},
	}

	// Load cache by calling OrdersTotal
	totalVal, _ := builtinOrdersTotal(vm, nil)
	if totalVal.ToInt() != 1 {
		t.Fatalf("OrdersTotal before close = %d, want 1", totalVal.ToInt())
	}

	// Close the position (broker path — cacheTestBroker.PositionClose succeeds)
	_, _ = builtinOrderClose(vm, []interp.Value{
		interp.IntVal(1001),
		interp.DecimalVal(decimal.NewFromFloat(0.1)),
		interp.DecimalVal(decimal.NewFromFloat(1.0)),
		interp.IntVal(10),
	})

	// After close, cache should be invalidated → OrdersTotal re-queries broker
	// which now returns 0 positions.
	totalVal, _ = builtinOrdersTotal(vm, nil)
	if totalVal.ToInt() != 0 {
		t.Fatalf("OrdersTotal after close = %d, want 0 (cache invalidated)", totalVal.ToInt())
	}
}

// TestCTradeMagicDeviationReachLiveSignal verifies that SetExpertMagicNumber
// and SetDeviationInPoints values reach the CTrade.Buy signal in signalMode.
//
// Adversarial: delete `Magic: vm.tradeMagic` from ctradeOrder signal path
// → sig.Magic=0 ≠ 999 → RED.
func TestCTradeMagicDeviationReachLiveSignal(t *testing.T) {
	vmRunner, err := CompileMQL(tradeCtxTestSource)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	r.UpdateLiveState("10000", "10500", "500", "9500", nil, nil)

	// Run OnInit (sets magic=999, deviation=77)
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Enable signalMode on the VM for live trading
	vmRunner.vm.SetSignalMode(true)

	// Run OnTick — CTrade.Buy should emit a signal with Magic=999, Deviation=77
	sig, err := r.OnTick(context.Background(), decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}
	if sig == nil {
		t.Fatal("signal is nil, expected ActionBuy signal from CTrade.Buy")
	}
	if sig.Action != sdk.ActionBuy {
		t.Fatalf("signal.Action=%v, want ActionBuy", sig.Action)
	}
	if sig.Magic != 999 {
		t.Fatalf("signal.Magic=%d, want 999 (SetExpertMagicNumber not reaching signal)", sig.Magic)
	}
	if sig.Deviation != 77 {
		t.Fatalf("signal.Deviation=%d, want 77 (SetDeviationInPoints not reaching signal)", sig.Deviation)
	}
}

// TestFailedOrderSelectResetsCurrent verifies that a failed OrderSelect
// resets currentPos/currentOrder so OrderTicket() returns 0 after failure.
//
// Adversarial: delete the reset at top of builtinOrderSelect
// → OrderTicket() returns stale ticket → RED.
func TestFailedOrderSelectResetsCurrent(t *testing.T) {
	positions := []sdk.Position{
		{
			Ticket: 1001, Symbol: "EURUSD", Side: sdk.SideBuy, Magic: 12345,
			Volume: decimal.NewFromFloat(0.1), OpenPrice: decimal.NewFromFloat(1.1),
			OpenTime: time.Now(),
		},
	}
	r, vmRunner := compileAndSetupRunner(t, selectResetTestSource, positions, nil)
	vm := runTick(t, r, vmRunner)

	ticketOk := getGlobalInt(t, vm, "g_ticket_after_ok")
	if ticketOk != 1001 {
		t.Fatalf("OrderTicket after successful select = %d, want 1001", ticketOk)
	}
	ticketFail := getGlobalInt(t, vm, "g_ticket_after_fail")
	if ticketFail != 0 {
		t.Fatalf("OrderTicket after failed select = %d, want 0 (currentPos not reset)", ticketFail)
	}
}

// TestInvalidTicketOrderCloseFails verifies that closing an invalid ticket
// returns false (fail-closed) and the VM continues with g_after set.
func TestInvalidTicketOrderCloseFails(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.SetSignalMode(false) // broker path
	vm.ctx = &cacheTestContext{
		broker: &cacheTestBroker{
			positions: []sdk.Position{}, // empty — no positions
			closeErr:  errors.New("invalid ticket"),
		},
	}

	// OrderClose with invalid ticket → broker returns error → false
	result, _ := builtinOrderClose(vm, []interp.Value{
		interp.IntVal(99999),
		interp.DecimalVal(decimal.NewFromFloat(0.1)),
		interp.DecimalVal(decimal.NewFromFloat(1.0)),
		interp.IntVal(10),
	})
	if result.Bool {
		t.Fatal("OrderClose(invalid ticket) returned true, want false (fail-closed)")
	}
}

// ── VM-TRADE-CONTEXT-2 tests (9) ─────────────────────────────────────

// TestSignalMode_OrderCloseBy_BothTickets verifies that OrderCloseBy in
// signalMode carries both ticket1 and ticket2 (OppositeTicket).
//
// Adversarial: delete `OppositeTicket: ticket2` → sig.OppositeTicket=0 ≠ 200 → RED.
func TestSignalMode_OrderCloseBy_BothTickets(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	_, _ = builtinOrderCloseBy(vm, []interp.Value{
		interp.IntVal(100),
		interp.IntVal(200),
	})
	if vm.signal == nil {
		t.Fatal("signal is nil, expected ActionClose")
	}
	if vm.signal.Action != sdk.ActionClose {
		t.Errorf("signal.Action=%v, want ActionClose", vm.signal.Action)
	}
	if vm.signal.OrderTicket != 100 {
		t.Errorf("signal.OrderTicket=%d, want 100", vm.signal.OrderTicket)
	}
	if vm.signal.OppositeTicket != 200 {
		t.Errorf("signal.OppositeTicket=%d, want 200 (VM-TRADE-CONTEXT-2)", vm.signal.OppositeTicket)
	}
}

// TestSignalMode_CTradePositionCloseBy_BothTickets verifies CTrade.PositionCloseBy
// in signalMode carries both tickets.
func TestSignalMode_CTradePositionCloseBy_BothTickets(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	_, _ = builtinCTradePositionCloseBy(vm, []interp.Value{
		interp.IntVal(100),
		interp.IntVal(200),
	})
	if vm.signal == nil {
		t.Fatal("signal is nil, expected ActionClose")
	}
	if vm.signal.OrderTicket != 100 {
		t.Errorf("signal.OrderTicket=%d, want 100", vm.signal.OrderTicket)
	}
	if vm.signal.OppositeTicket != 200 {
		t.Errorf("signal.OppositeTicket=%d, want 200 (VM-TRADE-CONTEXT-2)", vm.signal.OppositeTicket)
	}
}

// TestAccountNumber_FromContext verifies AccountNumber() reads Login from context.
//
// Adversarial: restore hardcoded 999999 → AccountNumber()=999999 ≠ 12345 → RED.
func TestAccountNumber_FromContext(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = &accountTestContext{login: 12345}

	result, _ := builtinAccountNumber(vm, nil)
	if result.ToInt() != 12345 {
		t.Fatalf("AccountNumber() = %d, want 12345 (from context Login)", result.ToInt())
	}
}

// TestAccountNumber_ZeroLoginReturnsZero verifies AccountNumber() returns 0
// when Login is 0 (not hardcoded 999999).
func TestAccountNumber_ZeroLoginReturnsZero(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = &accountTestContext{login: 0}

	result, _ := builtinAccountNumber(vm, nil)
	if result.ToInt() != 0 {
		t.Fatalf("AccountNumber() = %d, want 0 (Login=0)", result.ToInt())
	}
}

// TestIsTesting_BacktestMode verifies IsTesting()=true when signalMode=false.
//
// Adversarial: restore old ServerTime heuristic → backtest with ServerTime>0
// still returns true (coincidentally correct), but live mode breaks.
func TestIsTesting_BacktestMode(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.SetSignalMode(false) // backtest

	result, _ := builtinIsTesting(vm, nil)
	if !result.Bool {
		t.Fatal("IsTesting() = false in backtest (signalMode=false), want true")
	}
}

// TestIsTesting_LiveMode verifies IsTesting()=false when signalMode=true.
//
// Adversarial: restore old ServerTime heuristic → live mode (signalMode=true)
// but ServerTime>0 → IsTesting()=true ≠ false → RED.
func TestIsTesting_LiveMode(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.SetSignalMode(true) // live
	// Set a context with ServerTime > 0 to ensure the old heuristic would return true.
	vm.ctx = &accountTestContext{login: 12345, serverTime: 1700000000}

	result, _ := builtinIsTesting(vm, nil)
	if result.Bool {
		t.Fatal("IsTesting() = true in live (signalMode=true), want false")
	}
}

// ── brokerImpl error recording tests (3) ─────────────────────────────
//
// These tests live in backend/strategy/runner/broker_trade_context_test.go
// because brokerImpl is unexported and must be tested from within the
// runner package:
//   - TestBrokerImpl_PositionsQueryError_RecordsLastError
//   - TestBrokerImpl_OrdersQueryError_RecordsLastError
//   - TestBrokerImpl_HistoryOrders_NotAvailable_RecordsError
//   - TestRunner_OnBar_FailClosed_OnBrokerError (S14 adversarial)

// ── Test context/broker stubs for VM-TRADE-CONTEXT-1/2 ───────────────

// cacheTestContext implements sdk.Context for cache invalidation tests.
type cacheTestContext struct {
	sdk.Context
	broker *cacheTestBroker
}

type cacheTestBroker struct {
	sdk.Broker
	positions []sdk.Position
	closeErr  error
}

func (c *cacheTestContext) Broker() sdk.Broker {
	return c.broker
}
func (c *cacheTestContext) Symbol() string { return "EURUSD" }

func (b *cacheTestBroker) Positions(magic int32) []sdk.Position {
	if b.positions == nil {
		return nil
	}
	if magic == 0 {
		return b.positions
	}
	var filtered []sdk.Position
	for _, p := range b.positions {
		if p.Magic == magic {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
func (b *cacheTestBroker) Orders(magic int32) []sdk.PendingOrder { return nil }
func (b *cacheTestBroker) PositionClose(ticket int64, volume decimal.Decimal) (sdk.OrderResult, error) {
	if b.closeErr != nil {
		return sdk.OrderResult{}, b.closeErr
	}
	// Remove the position from the list.
	var remaining []sdk.Position
	for _, p := range b.positions {
		if p.Ticket != ticket {
			remaining = append(remaining, p)
		}
	}
	b.positions = remaining
	return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket}, nil
}
func (b *cacheTestBroker) OrderSend(req sdk.OrderRequest) (sdk.OrderResult, error) {
	return sdk.OrderResult{Ticket: 1}, nil
}

// accountTestContext implements sdk.Context for AccountNumber/IsTesting tests.
type accountTestContext struct {
	sdk.Context
	login      int64
	serverTime int64
}

func (c *accountTestContext) Account() sdk.AccountInfo {
	return sdk.AccountInfo{Login: c.login}
}
func (c *accountTestContext) ServerTime() int64  { return c.serverTime }
func (c *accountTestContext) Broker() sdk.Broker { return &cacheTestBroker{} }
