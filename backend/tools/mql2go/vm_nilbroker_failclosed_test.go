// vm_nilbroker_failclosed_test.go — ORDERSEND-NILBROKER-FAILCLOSED-1 (2026-09-18).
//
// All 12 trade-write builtins used to check `vm.ctx.Broker() == nil` BEFORE
// the signalMode branch, returning a fake rejection (-1/false, nil error).
// That was a double lie under signalMode: live/paper signals are dispatched
// server-side and never touch the broker, so the nil check silently dropped
// signals AND faked a rejection. The fix moved the nil check after the
// signalMode branch and made it fail closed (fmt.Errorf).
//
// Scenarios per site (table-driven):
//
//	(a) non-signal + nil broker → err != nil containing "no broker",
//	    no signal emitted (fail-closed direct evidence)
//	(b) signalMode + nil broker → err == nil, signal emitted with the right
//	    action (ordering-fix direct evidence; mutating the reorder REDs this)
//
// Adversarial proofs (3): restore broker-nil-first → (b) RED; restore
// nil→false,nil → (a) RED; remove the signalMode reorder on a sibling →
// (b) RED. See the dispatch mutation list.
package mql2go

import (
	"strings"
	"testing"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// TestTradeBuiltinsNilBrokerFailClosed covers all 12 trade-write sites × the
// two scenarios.
func TestTradeBuiltinsNilBrokerFailClosed(t *testing.T) {
	noopVM := func() *VM {
		return NewVM(&Bytecode{Builtins: make(map[string]BuiltinID)})
	}

	sites := []struct {
		name       string
		fn         func(vm *VM, args []interp.Value) (interp.Value, error)
		args       []interp.Value
		wantAction sdk.SignalAction
	}{
		{"OrderSend", builtinOrderSend, []interp.Value{
			interp.StringVal("EURUSD"), interp.IntVal(0), interp.IntVal(1),
		}, sdk.ActionBuy},
		{"CTrade.Buy", builtinCTradeBuy, []interp.Value{
			interp.IntVal(1), interp.StringVal("EURUSD"),
		}, sdk.ActionBuy},
		{"OrderClose", builtinOrderClose, []interp.Value{
			interp.IntVal(1), interp.IntVal(1),
		}, sdk.ActionClose},
		{"OrderCloseBy", builtinOrderCloseBy, []interp.Value{
			interp.IntVal(1), interp.IntVal(2),
		}, sdk.ActionClose},
		{"OrderModify", builtinOrderModify, []interp.Value{
			interp.IntVal(1), interp.IntVal(0), interp.IntVal(0), interp.IntVal(0),
		}, sdk.ActionModify},
		{"OrderDelete", builtinOrderDelete, []interp.Value{
			interp.IntVal(1),
		}, sdk.ActionCancel},
		{"CTrade.PositionClose", builtinCTradePositionClose, []interp.Value{
			interp.IntVal(1),
		}, sdk.ActionClose},
		{"CTrade.PositionClosePartial", builtinCTradePositionClosePartial, []interp.Value{
			interp.IntVal(1), interp.IntVal(1),
		}, sdk.ActionClose},
		{"CTrade.PositionCloseBy", builtinCTradePositionCloseBy, []interp.Value{
			interp.IntVal(1), interp.IntVal(2),
		}, sdk.ActionClose},
		{"CTrade.PositionModify", builtinCTradePositionModify, []interp.Value{
			interp.IntVal(1), interp.IntVal(0), interp.IntVal(0),
		}, sdk.ActionModify},
		{"CTrade.OrderDelete", builtinCTradeOrderDelete, []interp.Value{
			interp.IntVal(1),
		}, sdk.ActionCancel},
		{"CloseAll", builtinCloseAll, nil, sdk.ActionCloseAll},
	}

	for _, site := range sites {
		t.Run(site.name, func(t *testing.T) {
			// (a) non-signal + nil broker → fail closed with "no broker" error,
			// no signal emitted, fake rejection value returned.
			vm := noopVM()
			v, err := site.fn(vm, site.args)
			if err == nil {
				t.Fatal("err = nil, want 'no broker' error (fail-closed)")
			}
			if !strings.Contains(err.Error(), "no broker") {
				t.Fatalf("err = %v, want it to contain 'no broker'", err)
			}
			if vm.signal != nil {
				t.Fatalf("signal = %+v, want nil (no signal outside signalMode)", vm.signal)
			}
			switch site.name {
			case "OrderSend":
				if v.Int != -1 {
					t.Fatalf("OrderSend value = %d, want -1 alongside the error", v.Int)
				}
			default:
				if v.Bool {
					t.Fatalf("value = true, want false alongside the error")
				}
			}

			// (b) signalMode + nil broker → signal emitted (server-side OMS
			// dispatch never touches the broker), no error, success value.
			vm = noopVM()
			vm.signalMode = true
			v, err = site.fn(vm, site.args)
			if err != nil {
				t.Fatalf("err = %v, want nil (signal path must be broker-immune)", err)
			}
			if vm.signal == nil {
				t.Fatal("signal = nil, want emitted (nil-broker check must sit AFTER the signalMode branch)")
			}
			if vm.signal.Action != site.wantAction {
				t.Fatalf("signal.Action = %v, want %v", vm.signal.Action, site.wantAction)
			}
			switch site.name {
			case "OrderSend":
				if v.Int != 1 {
					t.Fatalf("OrderSend value = %d, want 1 (positive ticket in signalMode)", v.Int)
				}
			default:
				if !v.Bool {
					t.Fatal("value = false, want true (success in signalMode)")
				}
			}
		})
	}
}
