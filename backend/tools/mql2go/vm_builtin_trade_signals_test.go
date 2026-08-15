package mql2go

import (
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// ── Task 3: Signal mode for close/modify/delete/cancel ───────────────
//
// These tests verify that trade builtins moved to vm_builtin_trade_signals.go
// emit sdk.Signal in signalMode instead of calling the broker.
// Remove the signalMode branch from any builtin → vm.signal stays nil → test goes red.

type testBroker struct {
	sdk.Broker
}

func (b *testBroker) OrderSend(req sdk.OrderRequest) (sdk.OrderResult, error) {
	return sdk.OrderResult{Ticket: 1}, nil
}
func (b *testBroker) PositionClose(ticket int64, volume decimal.Decimal) (sdk.OrderResult, error) {
	return sdk.OrderResult{Ticket: ticket}, nil
}
func (b *testBroker) PositionCloseBy(t1, t2 int64) (sdk.OrderResult, error) {
	return sdk.OrderResult{Ticket: t1}, nil
}
func (b *testBroker) PositionModify(ticket int64, sl, tp decimal.Decimal) (sdk.OrderResult, error) {
	return sdk.OrderResult{Ticket: ticket}, nil
}
func (b *testBroker) OrderDelete(ticket int64) (sdk.OrderResult, error) {
	return sdk.OrderResult{Ticket: ticket}, nil
}
func (b *testBroker) Positions(magic int32) []sdk.Position  { return nil }
func (b *testBroker) Orders(magic int32) []sdk.PendingOrder { return nil }

type testContext struct {
	sdk.Context
	broker sdk.Broker
}

func (c *testContext) Broker() sdk.Broker { return c.broker }
func (c *testContext) Symbol() string     { return "EURUSD" }

func newSignalTestVM() *VM {
	bc := &Bytecode{
		OnBar:    -1,
		Builtins: make(map[string]BuiltinID),
	}
	vm := NewVM(bc)
	vm.SetSignalMode(true)
	vm.ctx = &testContext{broker: &testBroker{}}
	return vm
}

// TestSignalMode_OrderClose verifies OrderClose in signalMode emits ActionClose.
func TestSignalMode_OrderClose(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	_, _ = builtinOrderClose(vm, []interp.Value{
		interp.IntVal(42),
		interp.DecimalVal(decimal.NewFromFloat(0.1)),
		interp.DecimalVal(decimal.NewFromFloat(1.0)),
		interp.IntVal(10),
	})
	if vm.signal == nil {
		t.Fatal("signalMode OrderClose: signal is nil, expected ActionClose")
	}
	if vm.signal.Action != sdk.ActionClose {
		t.Errorf("signal.Action=%v, want ActionClose", vm.signal.Action)
	}
	if vm.signal.OrderTicket != 42 {
		t.Errorf("signal.OrderTicket=%d, want 42", vm.signal.OrderTicket)
	}
}

// TestSignalMode_OrderModify verifies OrderModify in signalMode emits ActionModify.
func TestSignalMode_OrderModify(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	_, _ = builtinOrderModify(vm, []interp.Value{
		interp.IntVal(42),
		interp.DecimalVal(decimal.NewFromFloat(1.0)),
		interp.DecimalVal(decimal.NewFromFloat(0.9)),
		interp.DecimalVal(decimal.NewFromFloat(1.1)),
		interp.IntVal(0),
		interp.IntVal(0),
	})
	if vm.signal == nil {
		t.Fatal("signalMode OrderModify: signal is nil, expected ActionModify")
	}
	if vm.signal.Action != sdk.ActionModify {
		t.Errorf("signal.Action=%v, want ActionModify", vm.signal.Action)
	}
	if vm.signal.OrderTicket != 42 {
		t.Errorf("signal.OrderTicket=%d, want 42", vm.signal.OrderTicket)
	}
}

// TestSignalMode_OrderDelete verifies OrderDelete in signalMode emits ActionCancel.
func TestSignalMode_OrderDelete(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	_, _ = builtinOrderDelete(vm, []interp.Value{interp.IntVal(99)})
	if vm.signal == nil {
		t.Fatal("signalMode OrderDelete: signal is nil, expected ActionCancel")
	}
	if vm.signal.Action != sdk.ActionCancel {
		t.Errorf("signal.Action=%v, want ActionCancel", vm.signal.Action)
	}
	if vm.signal.OrderTicket != 99 {
		t.Errorf("signal.OrderTicket=%d, want 99", vm.signal.OrderTicket)
	}
}

// TestSignalMode_CTradePositionClose verifies CTrade.PositionClose in signalMode.
func TestSignalMode_CTradePositionClose(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	_, _ = builtinCTradePositionClose(vm, []interp.Value{interp.IntVal(55)})
	if vm.signal == nil {
		t.Fatal("signalMode CTrade.PositionClose: signal is nil, expected ActionClose")
	}
	if vm.signal.Action != sdk.ActionClose {
		t.Errorf("signal.Action=%v, want ActionClose", vm.signal.Action)
	}
}

// TestSignalMode_CTradePositionModify verifies CTrade.PositionModify in signalMode.
func TestSignalMode_CTradePositionModify(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	_, _ = builtinCTradePositionModify(vm, []interp.Value{
		interp.IntVal(77),
		interp.DecimalVal(decimal.NewFromFloat(0.9)),
		interp.DecimalVal(decimal.NewFromFloat(1.1)),
	})
	if vm.signal == nil {
		t.Fatal("signalMode CTrade.PositionModify: signal is nil, expected ActionModify")
	}
	if vm.signal.Action != sdk.ActionModify {
		t.Errorf("signal.Action=%v, want ActionModify", vm.signal.Action)
	}
	if vm.signal.OrderTicket != 77 {
		t.Errorf("signal.OrderTicket=%d, want 77", vm.signal.OrderTicket)
	}
}

// TestSignalMode_CTradeOrderDelete verifies CTrade.OrderDelete in signalMode.
func TestSignalMode_CTradeOrderDelete(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	_, _ = builtinCTradeOrderDelete(vm, []interp.Value{interp.IntVal(88)})
	if vm.signal == nil {
		t.Fatal("signalMode CTrade.OrderDelete: signal is nil, expected ActionCancel")
	}
	if vm.signal.Action != sdk.ActionCancel {
		t.Errorf("signal.Action=%v, want ActionCancel", vm.signal.Action)
	}
	if vm.signal.OrderTicket != 88 {
		t.Errorf("signal.OrderTicket=%d, want 88", vm.signal.OrderTicket)
	}
}

// TestSignalMode_CloseAll verifies CloseAll in signalMode emits ActionCloseAll.
func TestSignalMode_CloseAll(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	_, _ = builtinCloseAll(vm, nil)
	if vm.signal == nil {
		t.Fatal("signalMode CloseAll: signal is nil, expected ActionCloseAll")
	}
	if vm.signal.Action != sdk.ActionCloseAll {
		t.Errorf("signal.Action=%v, want ActionCloseAll", vm.signal.Action)
	}
}

// TestNonSignalMode_OrderClose_NoSignal verifies that in non-signalMode,
// OrderClose does NOT set vm.signal (it calls the broker directly).
func TestNonSignalMode_OrderClose_NoSignal(t *testing.T) {
	vm := newSignalTestVM()
	vm.SetSignalMode(false)
	vm.signal = nil
	_, _ = builtinOrderClose(vm, []interp.Value{
		interp.IntVal(42),
		interp.DecimalVal(decimal.NewFromFloat(0.1)),
		interp.DecimalVal(decimal.NewFromFloat(1.0)),
		interp.IntVal(10),
	})
	if vm.signal != nil {
		t.Errorf("non-signalMode OrderClose: signal should be nil, got Action=%v", vm.signal.Action)
	}
}
