package mql2go

import (
	"fmt"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// ── Signal-mode trade builtins (LIVE-HARNESS-PARITY Task 3) ───────────
//
// These builtins are moved from vm_builtin_trade.go to stay under the 758-line
// redline. In signalMode (live/paper), close/modify/delete/cancel emit an
// sdk.Signal instead of calling the broker directly. The server-side dispatch
// (dispatchLiveSignal) then routes the signal to the OMS or paper engine.

func builtinOrderClose(vm *VM, args []interp.Value) (interp.Value, error) {
	ticket := int64(argI(args, 0))
	volume := argD(args, 1)
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:      sdk.ActionClose,
			OrderTicket: ticket,
			Volume:      volume,
		}
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	}
	// ORDERSEND-NILBROKER-FAILCLOSED-1: no broker is an environment defect, not a rejection — fail closed.
	if vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("OrderClose: no broker in the VM")
	}
	res, err := vm.ctx.Broker().PositionClose(ticket, volume)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("OrderClose broker error: %w", err)
	}
	switch res.RetCode {
	case sdk.RetDone, sdk.RetDonePartial:
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	case "":
		return interp.BoolVal(false), fmt.Errorf("OrderClose: broker returned empty RetCode")
	default:
		vm.lastError = mqlErrFromRetCode(res.RetCode)
		return interp.BoolVal(false), nil
	}
}

func builtinOrderCloseBy(vm *VM, args []interp.Value) (interp.Value, error) {
	ticket1 := int64(argI(args, 0))
	ticket2 := int64(argI(args, 1))
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:         sdk.ActionClose,
			OrderTicket:    ticket1,
			OppositeTicket: ticket2, // VM-TRADE-CONTEXT-2
		}
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	}
	// ORDERSEND-NILBROKER-FAILCLOSED-1: no broker is an environment defect, not a rejection — fail closed.
	if vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("OrderCloseBy: no broker in the VM")
	}
	res, err := vm.ctx.Broker().PositionCloseBy(ticket1, ticket2)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("OrderCloseBy broker error: %w", err)
	}
	switch res.RetCode {
	case sdk.RetDone, sdk.RetDonePartial:
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	case "":
		return interp.BoolVal(false), fmt.Errorf("OrderCloseBy: broker returned empty RetCode")
	default:
		vm.lastError = mqlErrFromRetCode(res.RetCode)
		return interp.BoolVal(false), nil
	}
}

func builtinOrderModify(vm *VM, args []interp.Value) (interp.Value, error) {
	ticket := int64(argI(args, 0))
	price := argD(args, 1)
	sl := argD(args, 2)
	tp := argD(args, 3)
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:      sdk.ActionModify,
			OrderTicket: ticket,
			Price:       price,
			StopLoss:    sl,
			TakeProfit:  tp,
		}
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	}
	// ORDERSEND-NILBROKER-FAILCLOSED-1: no broker is an environment defect, not a rejection — fail closed.
	if vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("OrderModify: no broker in the VM")
	}
	res, err := vm.ctx.Broker().PositionModify(ticket, sl, tp)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("OrderModify broker error: %w", err)
	}
	switch res.RetCode {
	case sdk.RetDone, sdk.RetDonePartial:
	case "":
		return interp.BoolVal(false), fmt.Errorf("OrderModify: broker returned empty RetCode")
	default:
		vm.lastError = mqlErrFromRetCode(res.RetCode)
		return interp.BoolVal(false), nil
	}
	if !price.IsZero() {
		if pm, ok := vm.ctx.Broker().(pendingPriceModifier); ok {
			res, err := pm.PositionModifyPrice(ticket, price)
			if err != nil {
				return interp.BoolVal(false), fmt.Errorf("OrderModify broker error: %w", err)
			}
			switch res.RetCode {
			case sdk.RetDone, sdk.RetDonePartial:
			case "":
				return interp.BoolVal(false), fmt.Errorf("OrderModify: broker returned empty RetCode")
			default:
				vm.lastError = mqlErrFromRetCode(res.RetCode)
				return interp.BoolVal(false), nil
			}
		}
	}
	vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
	return interp.BoolVal(true), nil
}

func builtinOrderDelete(vm *VM, args []interp.Value) (interp.Value, error) {
	ticket := int64(argI(args, 0))
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:      sdk.ActionCancel,
			OrderTicket: ticket,
		}
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	}
	// ORDERSEND-NILBROKER-FAILCLOSED-1: no broker is an environment defect, not a rejection — fail closed.
	if vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("OrderDelete: no broker in the VM")
	}
	res, err := vm.ctx.Broker().OrderDelete(ticket)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("OrderDelete broker error: %w", err)
	}
	switch res.RetCode {
	case sdk.RetDone, sdk.RetDonePartial:
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	case "":
		return interp.BoolVal(false), fmt.Errorf("OrderDelete: broker returned empty RetCode")
	default:
		vm.lastError = mqlErrFromRetCode(res.RetCode)
		return interp.BoolVal(false), nil
	}
}

func builtinCTradePositionClose(vm *VM, args []interp.Value) (interp.Value, error) {
	ticket := int64(argI(args, 0))
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:      sdk.ActionClose,
			OrderTicket: ticket,
			Volume:      decimal.Zero,
		}
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	}
	// ORDERSEND-NILBROKER-FAILCLOSED-1: no broker is an environment defect, not a rejection — fail closed.
	if vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("CTrade.PositionClose: no broker in the VM")
	}
	res, err := vm.ctx.Broker().PositionClose(ticket, decimal.Zero)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("CTrade.PositionClose broker error: %w", err)
	}
	switch res.RetCode {
	case sdk.RetDone, sdk.RetDonePartial:
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	case "":
		return interp.BoolVal(false), fmt.Errorf("CTrade.PositionClose: broker returned empty RetCode")
	default:
		vm.lastError = mqlErrFromRetCode(res.RetCode)
		return interp.BoolVal(false), nil
	}
}

func builtinCTradePositionClosePartial(vm *VM, args []interp.Value) (interp.Value, error) {
	ticket := int64(argI(args, 0))
	volume := argD(args, 1)
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:      sdk.ActionClose,
			OrderTicket: ticket,
			Volume:      volume,
		}
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	}
	// ORDERSEND-NILBROKER-FAILCLOSED-1: no broker is an environment defect, not a rejection — fail closed.
	if vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("CTrade.PositionClosePartial: no broker in the VM")
	}
	res, err := vm.ctx.Broker().PositionClose(ticket, volume)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("CTrade.PositionClosePartial broker error: %w", err)
	}
	switch res.RetCode {
	case sdk.RetDone, sdk.RetDonePartial:
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	case "":
		return interp.BoolVal(false), fmt.Errorf("CTrade.PositionClosePartial: broker returned empty RetCode")
	default:
		vm.lastError = mqlErrFromRetCode(res.RetCode)
		return interp.BoolVal(false), nil
	}
}

func builtinCTradePositionCloseBy(vm *VM, args []interp.Value) (interp.Value, error) {
	t1 := int64(argI(args, 0))
	t2 := int64(argI(args, 1))
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:         sdk.ActionClose,
			OrderTicket:    t1,
			OppositeTicket: t2, // VM-TRADE-CONTEXT-2
		}
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	}
	// ORDERSEND-NILBROKER-FAILCLOSED-1: no broker is an environment defect, not a rejection — fail closed.
	if vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("CTrade.PositionCloseBy: no broker in the VM")
	}
	res, err := vm.ctx.Broker().PositionCloseBy(t1, t2)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("CTrade.PositionCloseBy broker error: %w", err)
	}
	switch res.RetCode {
	case sdk.RetDone, sdk.RetDonePartial:
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	case "":
		return interp.BoolVal(false), fmt.Errorf("CTrade.PositionCloseBy: broker returned empty RetCode")
	default:
		vm.lastError = mqlErrFromRetCode(res.RetCode)
		return interp.BoolVal(false), nil
	}
}

func builtinCTradePositionModify(vm *VM, args []interp.Value) (interp.Value, error) {
	ticket := int64(argI(args, 0))
	sl := argD(args, 1)
	tp := argD(args, 2)
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:      sdk.ActionModify,
			OrderTicket: ticket,
			StopLoss:    sl,
			TakeProfit:  tp,
		}
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	}
	// ORDERSEND-NILBROKER-FAILCLOSED-1: no broker is an environment defect, not a rejection — fail closed.
	if vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("CTrade.PositionModify: no broker in the VM")
	}
	res, err := vm.ctx.Broker().PositionModify(ticket, sl, tp)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("CTrade.PositionModify broker error: %w", err)
	}
	switch res.RetCode {
	case sdk.RetDone, sdk.RetDonePartial:
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	case "":
		return interp.BoolVal(false), fmt.Errorf("CTrade.PositionModify: broker returned empty RetCode")
	default:
		vm.lastError = mqlErrFromRetCode(res.RetCode)
		return interp.BoolVal(false), nil
	}
}

func builtinCTradeOrderDelete(vm *VM, args []interp.Value) (interp.Value, error) {
	ticket := int64(argI(args, 0))
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:      sdk.ActionCancel,
			OrderTicket: ticket,
		}
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	}
	// ORDERSEND-NILBROKER-FAILCLOSED-1: no broker is an environment defect, not a rejection — fail closed.
	if vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("CTrade.OrderDelete: no broker in the VM")
	}
	res, err := vm.ctx.Broker().OrderDelete(ticket)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("CTrade.OrderDelete broker error: %w", err)
	}
	switch res.RetCode {
	case sdk.RetDone, sdk.RetDonePartial:
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	case "":
		return interp.BoolVal(false), fmt.Errorf("CTrade.OrderDelete: broker returned empty RetCode")
	default:
		vm.lastError = mqlErrFromRetCode(res.RetCode)
		return interp.BoolVal(false), nil
	}
}

func builtinCloseAll(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action: sdk.ActionCloseAll,
		}
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	}
	// ORDERSEND-NILBROKER-FAILCLOSED-1: no broker is an environment defect, not a rejection — fail closed.
	if vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("CloseAll: no broker in the VM")
	}
	positions := vm.ctx.Broker().Positions(0)
	allOK := true
	for _, pos := range positions {
		res, err := vm.ctx.Broker().PositionClose(pos.Ticket, decimal.Zero)
		if err != nil {
			return interp.BoolVal(false), fmt.Errorf("CloseAll broker error: %w", err)
		}
		switch res.RetCode {
		case sdk.RetDone, sdk.RetDonePartial:
		case "":
			return interp.BoolVal(false), fmt.Errorf("CloseAll: broker returned empty RetCode")
		default:
			// Aggregation semantics kept: a partial failure = overall false,
			// recorded in _LastError instead of being silently swallowed.
			allOK = false
			vm.lastError = mqlErrFromRetCode(res.RetCode)
		}
	}
	vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
	return interp.BoolVal(allOK), nil
}
