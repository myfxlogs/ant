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
	broker  sdk.Broker
	account sdk.AccountInfo // VM-API-TRUTH-3: for IsDemo/IsConnected/IsTradeAllowed
}

func (c *testContext) Broker() sdk.Broker       { return c.broker }
func (c *testContext) Symbol() string           { return "EURUSD" }
func (c *testContext) Account() sdk.AccountInfo { return c.account }

func newSignalTestVM() *VM {
	bc := &Bytecode{
		OnBar:    -1,
		Builtins: make(map[string]BuiltinID),
	}
	vm := NewVM(bc)
	vm.SetSignalMode(true)
	vm.ctx = &testContext{
		broker:  &testBroker{},
		account: sdk.AccountInfo{IsDemo: true, IsConnected: true, IsTradeAllowed: true}, // backtest defaults
	}
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

// ── VM-TRADE-CONTEXT-2 behavior tests ────────────────────────────────

// TestSignalMode_OrderCloseBy_BothTickets verifies that OrderCloseBy in
// signalMode emits a signal with BOTH ticket1 and ticket2 (OppositeTicket).
// VM-TRADE-CONTEXT-2: previously only ticket1 was carried, losing ticket2.
func TestSignalMode_OrderCloseBy_BothTickets(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	_, _ = builtinOrderCloseBy(vm, []interp.Value{
		interp.IntVal(100),
		interp.IntVal(200),
	})
	if vm.signal == nil {
		t.Fatal("signalMode OrderCloseBy: signal is nil, expected ActionClose")
	}
	if vm.signal.Action != sdk.ActionClose {
		t.Errorf("signal.Action=%v, want ActionClose", vm.signal.Action)
	}
	if vm.signal.OrderTicket != 100 {
		t.Errorf("signal.OrderTicket=%d, want 100", vm.signal.OrderTicket)
	}
	if vm.signal.OppositeTicket != 200 {
		t.Errorf("signal.OppositeTicket=%d, want 200 (CloseBy must carry both tickets)", vm.signal.OppositeTicket)
	}
}

// TestSignalMode_CTradePositionCloseBy_BothTickets verifies that
// CTrade.PositionCloseBy in signalMode emits both tickets.
func TestSignalMode_CTradePositionCloseBy_BothTickets(t *testing.T) {
	vm := newSignalTestVM()
	vm.signal = nil
	_, _ = builtinCTradePositionCloseBy(vm, []interp.Value{
		interp.IntVal(300),
		interp.IntVal(400),
	})
	if vm.signal == nil {
		t.Fatal("signalMode CTrade.PositionCloseBy: signal is nil, expected ActionClose")
	}
	if vm.signal.OrderTicket != 300 {
		t.Errorf("signal.OrderTicket=%d, want 300", vm.signal.OrderTicket)
	}
	if vm.signal.OppositeTicket != 400 {
		t.Errorf("signal.OppositeTicket=%d, want 400 (CloseBy must carry both tickets)", vm.signal.OppositeTicket)
	}
}

// TestAccountNumber_FromContext verifies that AccountNumber reads from
// the context's AccountInfo.Login, not a hardcoded value.
func TestAccountNumber_FromContext(t *testing.T) {
	vm := newSignalTestVM()
	// testContext doesn't implement Account(); use a custom context.
	vm.ctx = &accountTestContext{login: 12345}
	val, err := builtinAccountNumber(vm, nil)
	if err != nil {
		t.Fatalf("AccountNumber: %v", err)
	}
	if val.ToInt() != 12345 {
		t.Errorf("AccountNumber=%d, want 12345 (from context)", val.ToInt())
	}
}

// TestAccountNumber_ZeroLoginRecordsBlindSpot verifies that AccountNumber
// records a blind spot when Login is 0 (unavailable).
func TestAccountNumber_ZeroLoginRecordsBlindSpot(t *testing.T) {
	vm := newSignalTestVM()
	vm.ctx = &accountTestContext{login: 0}
	before := len(vm.runtimeBlindSpots)
	_, err := builtinAccountNumber(vm, nil)
	if err != nil {
		t.Fatalf("AccountNumber: %v", err)
	}
	after := len(vm.runtimeBlindSpots)
	if after <= before {
		t.Error("AccountNumber with Login=0 should record a blind spot, but runtimeBlindSpots didn't grow")
	}
}

// TestAccountCompany_FromContext verifies that AccountCompany reads the
// broker company from the context's AccountInfo.Company.
// VM-API-TRUTH-2: previously AccountCompany was a noop returning "".
func TestAccountCompany_FromContext(t *testing.T) {
	vm := newSignalTestVM()
	vm.ctx = &accountTestContext{company: "ACME Broker"}
	val, err := builtinAccountCompany(vm, nil)
	if err != nil {
		t.Fatalf("AccountCompany: %v", err)
	}
	if val.ToString() != "ACME Broker" {
		t.Errorf("AccountCompany=%q, want %q (from context)", val.ToString(), "ACME Broker")
	}
}

// TestAccountCompany_EmptyContextReturnsEmpty verifies that AccountCompany
// returns empty string (not a hardcoded value) when context is nil.
func TestAccountCompany_EmptyContextReturnsEmpty(t *testing.T) {
	vm := newSignalTestVM()
	vm.ctx = nil
	val, err := builtinAccountCompany(vm, nil)
	if err != nil {
		t.Fatalf("AccountCompany: %v", err)
	}
	if val.ToString() != "" {
		t.Errorf("AccountCompany=%q, want empty string (nil context)", val.ToString())
	}
}

// TestIsTesting_BacktestMode verifies that IsTesting returns true in
// backtest mode (signalMode=false).
func TestIsTesting_BacktestMode(t *testing.T) {
	vm := newSignalTestVM()
	vm.SetSignalMode(false)
	val, _ := builtinIsTesting(vm, nil)
	if !val.Bool {
		t.Error("IsTesting in backtest mode (signalMode=false) should return true")
	}
}

// TestIsTesting_LiveMode verifies that IsTesting returns false in live mode.
func TestIsTesting_LiveMode(t *testing.T) {
	vm := newSignalTestVM()
	vm.SetSignalMode(true)
	val, _ := builtinIsTesting(vm, nil)
	if val.Bool {
		t.Error("IsTesting in live mode (signalMode=true) should return false")
	}
}

// accountTestContext implements sdk.Context with a configurable Account().
type accountTestContext struct {
	sdk.Context
	login   int64
	company string
}

func (c *accountTestContext) Account() sdk.AccountInfo {
	return sdk.AccountInfo{Login: c.login, Company: c.company}
}
