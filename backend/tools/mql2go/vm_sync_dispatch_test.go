package mql2go

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// ── VM-LIVE-SYNC-DISPATCH-1 (R1): synchronous signal dispatch ──────────
//
// In signalMode with a sync dispatcher (live sessions), trade builtins must
// return the broker's REAL outcome — OrderSend returns the real ticket,
// close/modify/delete return the real bool. Without a dispatcher (paper),
// the sentinel behaviour is preserved. Removing the syncDispatch call from
// emitSignal must turn T1/T4 RED (fabricated ticket 1 / dropped orders).

// T1: sync dispatcher returns real broker ticket — OrderSend returns it
// verbatim and backfills it into the emitted signal for the audit path.
func TestSyncDispatch_OrderSend_RealTicket(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	vm.SetSyncDispatcher(func(sig *sdk.Signal) (int64, error) {
		return 5678, nil
	})
	res, err := builtinOrderSend(vm, []interp.Value{
		interp.StringVal("EURUSD"),                   // symbol
		interp.IntVal(0),                             // OP_BUY
		interp.DecimalVal(decimal.NewFromFloat(0.1)), // volume
		interp.DecimalVal(decimal.Zero),              // price
		interp.IntVal(3),                             // slippage
		interp.DecimalVal(decimal.Zero),              // sl
		interp.DecimalVal(decimal.Zero),              // tp
		interp.StringVal(""),                         // comment
		interp.IntVal(0),                             // magic
		interp.IntVal(0),                             // expiration
		interp.IntVal(0),                             // color
	})
	if err != nil {
		t.Fatalf("OrderSend returned Go error: %v", err)
	}
	if int64(res.Int) != 5678 {
		t.Fatalf("OrderSend=%v, want real broker ticket 5678 (sentinel 1 is the R1 defect)", res)
	}
	if vm.signal == nil {
		t.Fatal("signal not emitted — audit path broken")
	}
	if vm.signal.OrderTicket != 5678 {
		t.Fatalf("signal.OrderTicket=%d, want 5678 — real ticket must ride the audit signal", vm.signal.OrderTicket)
	}
}

// T2: dispatcher failure → truthful failure (-1 + _LastError), not fake success.
func TestSyncDispatch_OrderSend_Rejected(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	vm.SetSyncDispatcher(func(sig *sdk.Signal) (int64, error) {
		return 0, errors.New("broker rejected: invalid volume")
	})
	res, err := builtinOrderSend(vm, []interp.Value{
		interp.StringVal("EURUSD"),
		interp.IntVal(0),
		interp.DecimalVal(decimal.NewFromFloat(0.1)),
		interp.DecimalVal(decimal.Zero),
		interp.IntVal(3),
		interp.DecimalVal(decimal.Zero),
		interp.DecimalVal(decimal.Zero),
		interp.StringVal(""),
		interp.IntVal(0),
		interp.IntVal(0),
		interp.IntVal(0),
	})
	if err != nil {
		t.Fatalf("OrderSend returned Go error (should be business -1): %v", err)
	}
	if int64(res.Int) != -1 {
		t.Fatalf("OrderSend=%v, want -1 on dispatch failure", res)
	}
	if vm.lastError != 146 {
		t.Fatalf("vm.lastError=%d, want 146 (ERR_TRADE_CONTEXT_BUSY)", vm.lastError)
	}
	// Signal is still emitted for the audit path even when rejected.
	if vm.signal == nil {
		t.Fatal("rejected signal must still be emitted for audit")
	}
}

// T2b: OrderClose via sync dispatcher — false on rejection, true on confirm.
func TestSyncDispatch_OrderClose_RealOutcome(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	vm.SetSyncDispatcher(func(sig *sdk.Signal) (int64, error) {
		return 42, nil
	})
	res, _ := builtinOrderClose(vm, []interp.Value{
		interp.IntVal(42),
		interp.DecimalVal(decimal.NewFromFloat(0.1)),
	})
	if res.Bool != true {
		t.Fatalf("OrderClose=%v, want true on confirmed close", res)
	}

	vm2 := newSignalTestVM()
	vm2.signal = nil
	vm2.SetSyncDispatcher(func(sig *sdk.Signal) (int64, error) {
		return 0, errors.New("close outcome unknown")
	})
	res2, _ := builtinOrderClose(vm2, []interp.Value{
		interp.IntVal(42),
		interp.DecimalVal(decimal.NewFromFloat(0.1)),
	})
	if res2.Bool != false {
		t.Fatalf("OrderClose=%v, want false on dispatch failure (optimistic true is the defect)", res2)
	}
	if vm2.lastError != 146 {
		t.Fatalf("vm.lastError=%d, want 146", vm2.lastError)
	}
}

// T3: no dispatcher (paper mode equivalent) — sentinel preserved.
func TestSyncDispatch_NilDispatcher_Sentinel(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	res, err := builtinOrderSend(vm, []interp.Value{
		interp.StringVal("EURUSD"),
		interp.IntVal(0),
		interp.DecimalVal(decimal.NewFromFloat(0.1)),
		interp.DecimalVal(decimal.Zero),
		interp.IntVal(3),
		interp.DecimalVal(decimal.Zero),
		interp.DecimalVal(decimal.Zero),
		interp.StringVal(""),
		interp.IntVal(0),
		interp.IntVal(0),
		interp.IntVal(0),
	})
	if err != nil {
		t.Fatalf("OrderSend error: %v", err)
	}
	if int64(res.Int) != 1 {
		t.Fatalf("OrderSend=%v, want sentinel 1 without dispatcher (paper residual)", res)
	}
}

// T4: two OrderSend calls in one event each reach the dispatcher —
// the old single-slot signal silently dropped the first order.
func TestSyncDispatch_TwoOrders_NoDrop(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	calls := 0
	tickets := []int64{111, 222}
	vm.SetSyncDispatcher(func(sig *sdk.Signal) (int64, error) {
		calls++
		return tickets[calls-1], nil
	})
	args := func() []interp.Value {
		return []interp.Value{
			interp.StringVal("EURUSD"),
			interp.IntVal(0),
			interp.DecimalVal(decimal.NewFromFloat(0.1)),
			interp.DecimalVal(decimal.Zero),
			interp.IntVal(3),
			interp.DecimalVal(decimal.Zero),
			interp.DecimalVal(decimal.Zero),
			interp.StringVal(""),
			interp.IntVal(0),
			interp.IntVal(0),
			interp.IntVal(0),
		}
	}
	r1, _ := builtinOrderSend(vm, args())
	r2, _ := builtinOrderSend(vm, args())
	if calls != 2 {
		t.Fatalf("dispatcher called %d times, want 2 — second order must not be dropped", calls)
	}
	if int64(r1.Int) != 111 || int64(r2.Int) != 222 {
		t.Fatalf("tickets=%v,%v want 111,222", r1, r2)
	}
}

// T5: dispatch failure still emits the signal for the response/audit path.
func TestSyncDispatch_FailureEmitsSignal(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	vm.SetSyncDispatcher(func(sig *sdk.Signal) (int64, error) {
		return 0, errors.New("rejected")
	})
	_, _ = builtinOrderClose(vm, []interp.Value{
		interp.IntVal(7),
		interp.DecimalVal(decimal.NewFromFloat(0.1)),
	})
	if vm.signal == nil || vm.signal.OrderTicket != 7 {
		t.Fatal("failed dispatch must still emit signal for audit")
	}
}
