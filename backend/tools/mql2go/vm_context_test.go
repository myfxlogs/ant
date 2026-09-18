package mql2go

import (
	"strings"
	"testing"

	"alphaforge/tools/mql2go/interp"
)

// QS-2.3: vm.ctx non-nil invariant via noopContext injection.

// S5a: NewVM injects noopContext; SetContext(nil) normalizes to noop.
func TestQS23_ContextNeverNil(t *testing.T) {
	bc := &Bytecode{Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	if vm.ctx == nil {
		t.Fatal("NewVM must inject noopContext — vm.ctx is nil")
	}
	vm.SetContext(nil)
	if vm.ctx == nil {
		t.Fatal("SetContext(nil) must normalize to noopContext — vm.ctx is nil")
	}
}

// S5b: builtins on a no-SetContext VM produce the same zero values the
// former nil-ctx branches returned.
func TestQS23_NoopContextZeroValues(t *testing.T) {
	bc := &Bytecode{Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)

	if v, _ := builtinBid(vm, nil); !v.Decimal.IsZero() {
		t.Fatalf("Bid=%s, want 0", v.Decimal)
	}
	if v, _ := builtinAsk(vm, nil); !v.Decimal.IsZero() {
		t.Fatalf("Ask=%s, want 0", v.Decimal)
	}
	if v, _ := builtinPoint(vm, nil); !v.Decimal.IsZero() {
		t.Fatalf("Point=%s, want 0", v.Decimal)
	}
	if v, _ := builtinDigits(vm, nil); v.Int != 0 {
		t.Fatalf("Digits=%d, want 0", v.Int)
	}
	if v, _ := builtinSymbol(vm, nil); v.Str != "" {
		t.Fatalf("Symbol=%q, want \"\"", v.Str)
	}
	if v, _ := builtinAccountBalance(vm, nil); !v.Decimal.IsZero() {
		t.Fatalf("AccountBalance=%s, want 0", v.Decimal)
	}
	if v, _ := builtinPrint(vm, []interp.Value{interp.StringVal("x")}); v.Kind != interp.ValNone {
		t.Fatal("Print must not panic and must return NoneVal")
	}
	// ORDERSEND-NILBROKER-FAILCLOSED-1: no broker is an environment defect —
	// OrderSend fails closed with an error (signalMode exempts the signal path).
	sendArgs := []interp.Value{
		interp.StringVal("EURUSD"), interp.IntVal(0), interp.IntVal(1),
	}
	if v, err := builtinOrderSend(vm, sendArgs); err == nil || !strings.Contains(err.Error(), "no broker") {
		t.Fatalf("OrderSend(no broker)=%d err=%v, want error containing 'no broker'", v.Int, err)
	}
	// Indicator builtins route through noopIndicatorSet → 0, no panic.
	ma := []interp.Value{interp.StringVal(""), interp.IntVal(0), interp.IntVal(14),
		interp.IntVal(0), interp.IntVal(0), interp.IntVal(1), interp.IntVal(0)}
	if v, _ := builtinIMA(vm, ma); !v.Decimal.IsZero() {
		t.Fatalf("iMA=%s, want 0", v.Decimal)
	}
	rsi := []interp.Value{interp.StringVal(""), interp.IntVal(0), interp.IntVal(14),
		interp.IntVal(1), interp.IntVal(0)}
	if v, _ := builtinIRSI(vm, rsi); !v.Decimal.IsZero() {
		t.Fatalf("iRSI=%s, want 0", v.Decimal)
	}
	// Close(0) via Bars() → empty series → 0, no panic.
	cl := []interp.Value{interp.StringVal(""), interp.IntVal(0), interp.IntVal(0)}
	if v, _ := builtinIClose(vm, cl); !v.Decimal.IsZero() {
		t.Fatalf("iClose=%s, want 0", v.Decimal)
	}
	// getSeries path (Close[] without symbol) → 0.
	if v := vm.getSeries("Close", 0); !v.Decimal.IsZero() {
		t.Fatalf("getSeries(Close)=%s, want 0", v.Decimal)
	}
}

// S5c: Param* pass through the default under noop.
func TestQS23_NoopContextParamDefaults(t *testing.T) {
	bc := &Bytecode{Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	if got := vm.ctx.ParamInt("x", 42); got != 42 {
		t.Fatalf("ParamInt=%d, want 42", got)
	}
	if got := vm.ctx.ParamString("x", "dflt"); got != "dflt" {
		t.Fatalf("ParamString=%q, want \"dflt\"", got)
	}
	if got := vm.ctx.ParamBool("x", true); got != true {
		t.Fatal("ParamBool=false, want true")
	}
}

// S5d: nil broker fails closed on trade writes (no broker = environment
// defect → error, not a fake rejection) — noop.Broker() is nil.
func TestQS23_NoopBrokerNilChecks(t *testing.T) {
	bc := &Bytecode{Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	if vm.ctx.Broker() != nil {
		t.Fatal("noopContext.Broker() must be nil")
	}
	if v, err := builtinOrderClose(vm, nil); v.Bool || err == nil || !strings.Contains(err.Error(), "no broker") {
		t.Fatalf("OrderClose(no broker)=%v err=%v, want false + error containing 'no broker'", v.Bool, err)
	}
	if v, _ := builtinOrdersTotal(vm, nil); v.Int != 0 {
		t.Fatalf("OrdersTotal(no broker)=%d, want 0", v.Int)
	}
	// SetTimer/KillTimer/Log must not panic under noop.
	vm.ctx.SetTimer(1)
	vm.ctx.KillTimer()
	vm.ctx.Log("noop")
}

// S5e: Bars() is a non-nil empty series — callers must not panic on
// .Close()/.Len() access (mutation anchor for Bars()==nil regression).
func TestQS23_NoopBarsNonNilEmpty(t *testing.T) {
	bc := &Bytecode{Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	bars := vm.ctx.Bars()
	if bars == nil {
		t.Fatal("noopContext.Bars() must return a non-nil empty series")
	}
	if bars.Len() != 0 {
		t.Fatalf("Bars().Len()=%d, want 0", bars.Len())
	}
	if !bars.Close(0).IsZero() || !bars.Close(5).IsZero() {
		t.Fatal("empty Bars().Close(shift) must return 0, not panic")
	}
}
