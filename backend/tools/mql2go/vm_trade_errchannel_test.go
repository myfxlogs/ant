// vm_trade_errchannel_test.go — TRADE-BUILTIN-ERR-SWALLOW-1 (2026-09-18).
//
// sdk.Broker write methods carry two channels: error = infrastructure
// failure (VM must fail closed/fatal), RetCode = business result
// (rejection → false/-1 + mapped MQL _LastError). The builtins used to
// swallow broker errors as false,nil and ignore RetCode entirely — rejections
// disguised as success, infra failures disguised as rejections.
//
// Table-driven scenarios (a)-(g) + GetLastError read-then-clear.
//
// Adversarial proofs (4): see the dispatch mutation list — restore bare
// err→false (fake success revives), restore err→false,nil (infra silently
// swallowed), delete vm.lastError write, delete the empty-RetCode case.
package mql2go

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// errChannelBroker is a programmable mock: each op returns a canned
// (OrderResult, error) pair. closeSeq (when non-empty) is consumed per call
// so CloseAll can see per-position outcomes.
type errChannelBroker struct {
	sdk.Broker
	orderSendRes   sdk.OrderResult
	orderSendErr   error
	closeRes       sdk.OrderResult
	closeErr       error
	closeSeq       []sdk.OrderResult
	closeSeqIdx    int
	closeByRes     sdk.OrderResult
	closeByErr     error
	modifyRes      sdk.OrderResult
	modifyErr      error
	modifyPriceRes sdk.OrderResult
	modifyPriceErr error
	deleteRes      sdk.OrderResult
	deleteErr      error
	positions      []sdk.Position
}

func (b *errChannelBroker) OrderSend(sdk.OrderRequest) (sdk.OrderResult, error) {
	return b.orderSendRes, b.orderSendErr
}
func (b *errChannelBroker) PositionClose(int64, decimal.Decimal) (sdk.OrderResult, error) {
	if b.closeSeqIdx < len(b.closeSeq) {
		res := b.closeSeq[b.closeSeqIdx]
		b.closeSeqIdx++
		return res, nil
	}
	return b.closeRes, b.closeErr
}
func (b *errChannelBroker) PositionCloseBy(int64, int64) (sdk.OrderResult, error) {
	return b.closeByRes, b.closeByErr
}
func (b *errChannelBroker) PositionModify(int64, decimal.Decimal, decimal.Decimal) (sdk.OrderResult, error) {
	return b.modifyRes, b.modifyErr
}
func (b *errChannelBroker) PositionModifyPrice(int64, decimal.Decimal) (sdk.OrderResult, error) {
	return b.modifyPriceRes, b.modifyPriceErr
}
func (b *errChannelBroker) OrderDelete(int64) (sdk.OrderResult, error) {
	return b.deleteRes, b.deleteErr
}
func (b *errChannelBroker) Positions(int32) []sdk.Position { return b.positions }

func newErrChannelVM(t *testing.T, b *errChannelBroker) *VM {
	t.Helper()
	vm := NewVM(&Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)})
	vm.ctx = &testContext{broker: b}
	return vm
}

// TestTradeErrChannel_RejectionBecomesLastError — (a)/(b): business
// rejections return false/-1 with NO error (not a fatal) and set _LastError
// to the mapped MQL4 code.
//
// Adversarial: restore the bare err→false form (delete the RetCode switch) →
// the builtin "succeeds" on a rejection → RED; delete the vm.lastError write
// → the code assertion REDs.
func TestTradeErrChannel_RejectionBecomesLastError(t *testing.T) {
	t.Run("(a) OrderDelete RetRejected", func(t *testing.T) {
		vm := newErrChannelVM(t, &errChannelBroker{deleteRes: sdk.OrderResult{RetCode: sdk.RetRejected}})
		v, err := builtinOrderDelete(vm, []interp.Value{interp.IntVal(1)})
		if err != nil {
			t.Fatalf("err = %v, want nil (business rejection is not infra)", err)
		}
		if v.Bool {
			t.Fatal("OrderDelete = true, want false (rejection must not fake success)")
		}
		if vm.lastError != 146 {
			t.Fatalf("lastError = %d, want 146 (ERR_TRADE_CONTEXT_BUSY approximation for RetRejected)", vm.lastError)
		}
	})

	t.Run("(b) OrderSend RetNoMoney", func(t *testing.T) {
		vm := newErrChannelVM(t, &errChannelBroker{orderSendRes: sdk.OrderResult{RetCode: sdk.RetNoMoney}})
		v, err := builtinOrderSend(vm, []interp.Value{
			interp.StringVal("EURUSD"), interp.IntVal(0), interp.IntVal(1),
		})
		if err != nil {
			t.Fatalf("err = %v, want nil (margin rejection must NOT kill the backtest as fatal)", err)
		}
		if v.Int != -1 {
			t.Fatalf("OrderSend = %d, want -1 (real MQL returns -1 + _LastError=134)", v.Int)
		}
		if vm.lastError != 134 {
			t.Fatalf("lastError = %d, want 134 (ERR_NOT_ENOUGH_MONEY)", vm.lastError)
		}
	})
}

// TestTradeErrChannel_InfraStaysFatal — (c): a plain broker error is
// infrastructure → fatal (propagated), not a silent false.
//
// Adversarial: restore err→false,nil → err == nil → RED.
func TestTradeErrChannel_InfraStaysFatal(t *testing.T) {
	vm := newErrChannelVM(t, &errChannelBroker{closeErr: fmt.Errorf("no executor configured")})
	v, err := builtinOrderClose(vm, []interp.Value{interp.IntVal(1), interp.IntVal(1)})
	if err == nil {
		t.Fatal("err = nil, want fatal (infra failure must not disguise as rejection)")
	}
	if !strings.Contains(err.Error(), "broker error") {
		t.Fatalf("err = %v, want it to contain 'broker error'", err)
	}
	if v.Bool {
		t.Fatal("OrderClose = true, want false alongside the fatal error")
	}
}

// TestTradeErrChannel_EmptyRetCodeFatal — (d): an empty RetCode is a contract
// violation → fail closed, NOT silently treated as a rejection or success.
//
// Adversarial: delete the "" case → empty code falls to default (rejection
// semantics) → the err assertion REDs.
func TestTradeErrChannel_EmptyRetCodeFatal(t *testing.T) {
	vm := newErrChannelVM(t, &errChannelBroker{closeRes: sdk.OrderResult{}})
	_, err := builtinOrderClose(vm, []interp.Value{interp.IntVal(1), interp.IntVal(1)})
	if err == nil {
		t.Fatal("err = nil, want fatal (empty RetCode = contract violation)")
	}
	if !strings.Contains(err.Error(), "empty RetCode") {
		t.Fatalf("err = %v, want it to contain 'empty RetCode'", err)
	}
}

// TestTradeErrChannel_DoneStillSucceeds — (e): RetDone/RetDonePartial keep
// the success paths intact (regression protection).
func TestTradeErrChannel_DoneStillSucceeds(t *testing.T) {
	t.Run("(e) OrderSend RetDone returns ticket", func(t *testing.T) {
		vm := newErrChannelVM(t, &errChannelBroker{orderSendRes: sdk.OrderResult{RetCode: sdk.RetDone, Ticket: 77}})
		v, err := builtinOrderSend(vm, []interp.Value{
			interp.StringVal("EURUSD"), interp.IntVal(0), interp.IntVal(1),
		})
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if v.Int != 77 {
			t.Fatalf("ticket = %d, want 77", v.Int)
		}
	})

	t.Run("(e) OrderSend RetDonePartial counts as done", func(t *testing.T) {
		vm := newErrChannelVM(t, &errChannelBroker{orderSendRes: sdk.OrderResult{RetCode: sdk.RetDonePartial, Ticket: 5}})
		v, err := builtinOrderSend(vm, []interp.Value{
			interp.StringVal("EURUSD"), interp.IntVal(0), interp.IntVal(1),
		})
		if err != nil || v.Int != 5 {
			t.Fatalf("ticket = %d err = %v, want 5/nil (partial fill is still a fill)", v.Int, err)
		}
	})

	doneSites := []struct {
		name string
		fn   func(vm *VM, args []interp.Value) (interp.Value, error)
		args []interp.Value
	}{
		{"OrderClose", builtinOrderClose, []interp.Value{interp.IntVal(1), interp.IntVal(1)}},
		{"OrderCloseBy", builtinOrderCloseBy, []interp.Value{interp.IntVal(1), interp.IntVal(2)}},
		{"OrderModify", builtinOrderModify, []interp.Value{interp.IntVal(1), interp.IntVal(0), interp.IntVal(0), interp.IntVal(0)}},
		{"OrderDelete", builtinOrderDelete, []interp.Value{interp.IntVal(1)}},
		{"CTrade.PositionClose", builtinCTradePositionClose, []interp.Value{interp.IntVal(1)}},
		{"CTrade.OrderDelete", builtinCTradeOrderDelete, []interp.Value{interp.IntVal(1)}},
		{"CTrade.PositionCloseBy", builtinCTradePositionCloseBy, []interp.Value{interp.IntVal(1), interp.IntVal(2)}},
		{"CTrade.PositionModify", builtinCTradePositionModify, []interp.Value{interp.IntVal(1), interp.IntVal(0), interp.IntVal(0)}},
		{"CTrade.PositionClosePartial", builtinCTradePositionClosePartial, []interp.Value{interp.IntVal(1), interp.IntVal(1)}},
		{"CTrade.Buy", builtinCTradeBuy, []interp.Value{interp.IntVal(1), interp.StringVal("EURUSD")}},
	}
	done := sdk.OrderResult{RetCode: sdk.RetDone, Ticket: 1}
	for _, site := range doneSites {
		t.Run("(e) "+site.name, func(t *testing.T) {
			vm := newErrChannelVM(t, &errChannelBroker{
				orderSendRes: done, closeRes: done, closeByRes: done,
				modifyRes: done, deleteRes: done,
			})
			v, err := site.fn(vm, site.args)
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if !v.Bool {
				t.Fatal("value = false, want true (RetDone must stay a success)")
			}
		})
	}
}

// TestTradeErrChannel_CloseAllAggregates — (f): one rejected position makes
// CloseAll false and records _LastError, without a fatal (aggregation
// semantics preserved, but the failure is no longer silently swallowed).
func TestTradeErrChannel_CloseAllAggregates(t *testing.T) {
	vm := newErrChannelVM(t, &errChannelBroker{
		positions: []sdk.Position{{Ticket: 1}, {Ticket: 2}},
		closeSeq: []sdk.OrderResult{
			{RetCode: sdk.RetDone, Ticket: 1},
			{RetCode: sdk.RetRejected, Ticket: 2},
		},
	})
	v, err := builtinCloseAll(vm, nil)
	if err != nil {
		t.Fatalf("err = %v, want nil (per-position rejection is business, not infra)", err)
	}
	if v.Bool {
		t.Fatal("CloseAll = true, want false (one rejected position = overall false)")
	}
	if vm.lastError != 146 {
		t.Fatalf("lastError = %d, want 146 (rejection recorded, was silently swallowed)", vm.lastError)
	}
}

// TestTradeErrChannel_ModifyPriceBranch — (g): the pendingPriceModifier
// branch follows the same three-state split.
func TestTradeErrChannel_ModifyPriceBranch(t *testing.T) {
	vm := newErrChannelVM(t, &errChannelBroker{
		modifyRes:      sdk.OrderResult{RetCode: sdk.RetDone, Ticket: 1},
		modifyPriceRes: sdk.OrderResult{RetCode: sdk.RetRejected, Ticket: 1},
	})
	v, err := builtinOrderModify(vm, []interp.Value{
		interp.IntVal(1), interp.IntVal(1), interp.IntVal(0), interp.IntVal(0),
	})
	if err != nil {
		t.Fatalf("err = %v, want nil (price rejection is business, not infra)", err)
	}
	if v.Bool {
		t.Fatal("OrderModify = true, want false (price branch rejected)")
	}
	if vm.lastError != 146 {
		t.Fatalf("lastError = %d, want 146", vm.lastError)
	}
}

// TestTradeErrChannel_GetLastErrorReadThenClear — the mapped code is visible
// through the real MQL GetLastError() builtin, which reads-then-clears:
// first read = mapped code, second read = 0.
func TestTradeErrChannel_GetLastErrorReadThenClear(t *testing.T) {
	vm := newErrChannelVM(t, &errChannelBroker{deleteRes: sdk.OrderResult{RetCode: sdk.RetRejected}})
	if _, err := builtinOrderDelete(vm, []interp.Value{interp.IntVal(1)}); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	gl, _ := builtinGetLastError(vm, nil)
	if got := gl.Int; got != 146 {
		t.Fatalf("GetLastError() = %d, want 146 (first read)", got)
	}
	gl, _ = builtinGetLastError(vm, nil)
	if got := gl.Int; got != 0 {
		t.Fatalf("GetLastError() = %d, want 0 (second read clears)", got)
	}
}
