package mql2go

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
)

// ── VM-TRADE-CONTEXT-1 behavior tests ────────────────────────────────

// TestVM_Audit_OrderCacheInvalidatedAfterClose verifies that after loading
// a non-empty order cache (via OrdersTotal), then closing a position
// (via OrderClose), a subsequent OrdersTotal sees the updated count (0,
// not the stale cached count 1). This proves invalidateOrderCaches runs
// after OrderClose, not just after OrderSend.
func TestVM_Audit_OrderCacheInvalidatedAfterClose(t *testing.T) {
	const source = `
int g_phase = 0;
int g_before = -1;
int g_after = -1;
int OnInit() { return 0; }
void OnTick() {
    if (g_phase == 0) {
        // Bar 1: create a position.
        OrderSend(Symbol(), OP_BUY, 0.1, Ask, 0, 0, 0, "create", 1, 0, clrGreen);
        g_phase = 1;
    } else if (g_phase == 1) {
        // Bar 2: load non-empty cache, then close, then re-query.
        g_before = OrdersTotal();
        if (g_before > 0) {
            if (OrderSelect(0, SELECT_BY_POS, MODE_TRADES)) {
                int ticket = OrderTicket();
                OrderClose(ticket, OrderLots(), Bid, 5, clrRed);
            }
        }
        g_after = OrdersTotal();
        g_phase = 2;
    }
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	if _, err := engine.Run(context.Background()); err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}
	before, _ := runner.GetGlobal("g_before")
	after, _ := runner.GetGlobal("g_after")
	if before.ToInt() != 1 {
		t.Fatalf("g_before = %d, want 1 (position should exist before close)", before.ToInt())
	}
	if after.ToInt() != 0 {
		t.Fatalf("g_after = %d, want 0 (cache should be invalidated after OrderClose, not stale 1)", after.ToInt())
	}
}

// TestVM_Audit_CTradeMagicDeviationReachLiveSignal verifies that CTrade
// SetExpertMagicNumber and SetDeviationInPoints propagate to the sdk.Signal
// in signalMode (live/paper path), not just the broker path. This proves
// the CTrade parameter透传 covers the full chain: setter → VM state → signal.
func TestVM_Audit_CTradeMagicDeviationReachLiveSignal(t *testing.T) {
	const source = `
#include <Trade/Trade.mqh>
CTrade trade;
int OnInit() {
    trade.SetExpertMagicNumber(999);
    trade.SetDeviationInPoints(77);
    return 0;
}
void OnTick() { trade.Buy(0.1, Symbol()); }
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	// Run OnInit to set magic/deviation.
	if err := runner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit: %v", err)
	}
	if runner.vm.tradeMagic != 999 {
		t.Fatalf("tradeMagic = %d, want 999", runner.vm.tradeMagic)
	}
	if runner.vm.tradeDeviation != 77 {
		t.Fatalf("tradeDeviation = %d, want 77", runner.vm.tradeDeviation)
	}
	// Enable signal mode and run OnTick — CTrade.Buy should emit a signal
	// with the configured magic and deviation.
	runner.SetSignalMode(true)
	runner.vm.ctx = &auditContext{
		bars:   sdk.BarsToSlice(auditBars(1)),
		symbol: "EURUSD",
		tf:     "M1",
		broker: &testBroker{},
	}
	if err := runner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick: %v", err)
	}
	sig := runner.vm.Signal()
	if sig == nil {
		t.Fatal("CTrade.Buy in signalMode produced nil signal")
	}
	if sig.Magic != 999 {
		t.Fatalf("signal.Magic = %d, want 999 (CTrade magic not propagated to live signal)", sig.Magic)
	}
	if sig.Deviation != 77 {
		t.Fatalf("signal.Deviation = %d, want 77 (CTrade deviation not propagated to live signal)", sig.Deviation)
	}
	if sig.Action != sdk.ActionBuy {
		t.Fatalf("signal.Action = %v, want ActionBuy", sig.Action)
	}
}

// TestVM_Audit_FailedOrderSelectResetsCurrent verifies that a failed
// OrderSelect clears currentPos/currentOrder — so OrderTicket() returns 0
// after a failed select, not the previous order's ticket. This proves the
// selection reset at the top of builtinOrderSelect (lines 153-154) works.
func TestVM_Audit_FailedOrderSelectResetsCurrent(t *testing.T) {
	const source = `
int g_phase = 0;
int g_ticket_first = -1;
int g_ticket_after_fail = -1;
int OnInit() { return 0; }
void OnTick() {
    if (g_phase == 0) {
        OrderSend(Symbol(), OP_BUY, 0.1, Ask, 0, 0, 0, "create", 1, 0, clrGreen);
        g_phase = 1;
    } else if (g_phase == 1) {
        if (OrdersTotal() > 0) {
            if (OrderSelect(0, SELECT_BY_POS, MODE_TRADES)) {
                g_ticket_first = OrderTicket();
            }
            // Now select a non-existent index — must fail and reset current.
            if (OrderSelect(999, SELECT_BY_POS, MODE_TRADES)) {
                // Should not reach here.
            }
            g_ticket_after_fail = OrderTicket();
        }
        g_phase = 2;
    }
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	if _, err := engine.Run(context.Background()); err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}
	first, _ := runner.GetGlobal("g_ticket_first")
	afterFail, _ := runner.GetGlobal("g_ticket_after_fail")
	if first.ToInt() <= 0 {
		t.Fatalf("g_ticket_first = %d, want > 0 (first select should succeed)", first.ToInt())
	}
	if afterFail.ToInt() != 0 {
		t.Fatalf("g_ticket_after_fail = %d, want 0 (failed OrderSelect must reset currentPos so OrderTicket returns 0)", afterFail.ToInt())
	}
}

// TestVM_Audit_InvalidTicketOrderCloseFails verifies that OrderClose with
// an invalid ticket produces an error (fail-closed), not silently succeeding.
// The VM's fail-closed mechanism (VM-RUNTIME-FAILCLOSED-1) stops execution
// when the broker returns an error, so g_result stays at its initial value
// (the assignment never completes) and Engine.Run returns an error.
func TestVM_Audit_InvalidTicketOrderCloseFails(t *testing.T) {
	const source = `
int g_result = 1;
int g_after = -1;
int OnInit() { return 0; }
void OnTick() {
    // OrderClose with invalid ticket 99999 — broker returns error.
    g_result = OrderClose(99999, 0.1, Bid, 5, clrRed);
    // This line should NOT execute (fail-closed stops execution on error).
    g_after = 42;
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err == nil {
		t.Fatal("OrderClose with invalid ticket should cause an error (fail-closed), got nil")
	}
	// g_result stays at initial 1 because the assignment was interrupted by fail-closed.
	// g_after stays at -1 because execution stopped before reaching that line.
	after, _ := runner.GetGlobal("g_after")
	if after.ToInt() != -1 {
		t.Fatalf("g_after = %d, want -1 (fail-closed should stop execution after OrderClose error, g_after should not be set)", after.ToInt())
	}
}
