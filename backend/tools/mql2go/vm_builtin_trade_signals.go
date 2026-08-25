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
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("trade operation: broker context is unavailable")
	}
	ticket := int64(argI(args, 0))
	volume := argD(args, 1)
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:      sdk.ActionClose,
			OrderTicket: ticket,
			Volume:      volume,
		}
		return interp.BoolVal(true), nil
	}
	_, err := vm.ctx.Broker().PositionClose(ticket, volume)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("PositionClose: %w", err)
	}
	vm.invalidateOrderCaches()
	return interp.BoolVal(true), nil
}

func builtinOrderCloseBy(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("trade operation: broker context is unavailable")
	}
	ticket1 := int64(argI(args, 0))
	ticket2 := int64(argI(args, 1))
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:         sdk.ActionClose,
			OrderTicket:    ticket1,
			OppositeTicket: ticket2,
		}
		return interp.BoolVal(true), nil
	}
	_, err := vm.ctx.Broker().PositionCloseBy(ticket1, ticket2)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("PositionCloseBy: %w", err)
	}
	vm.invalidateOrderCaches()
	return interp.BoolVal(true), nil
}

func builtinOrderModify(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("trade operation: broker context is unavailable")
	}
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
		return interp.BoolVal(true), nil
	}
	_, err := vm.ctx.Broker().PositionModify(ticket, sl, tp)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("PositionModify: %w", err)
	}
	if !price.IsZero() {
		pending := false
		for _, order := range vm.ctx.Broker().Orders(0) {
			if order.Ticket == ticket {
				pending = true
				break
			}
		}
		if pending {
			pm, ok := vm.ctx.Broker().(pendingPriceModifier)
			if !ok {
				return interp.BoolVal(false), fmt.Errorf("PositionModifyPrice: broker does not support pending price modification")
			}
			if _, err := pm.PositionModifyPrice(ticket, price); err != nil {
				return interp.BoolVal(false), fmt.Errorf("PositionModifyPrice: %w", err)
			}
		}
	}
	vm.invalidateOrderCaches()
	return interp.BoolVal(true), nil
}

func builtinOrderDelete(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("trade operation: broker context is unavailable")
	}
	ticket := int64(argI(args, 0))
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:      sdk.ActionCancel,
			OrderTicket: ticket,
		}
		return interp.BoolVal(true), nil
	}
	_, err := vm.ctx.Broker().OrderDelete(ticket)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("OrderDelete: %w", err)
	}
	vm.invalidateOrderCaches()
	return interp.BoolVal(true), nil
}

func builtinCTradePositionClose(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("trade operation: broker context is unavailable")
	}
	ticket := int64(argI(args, 0))
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:      sdk.ActionClose,
			OrderTicket: ticket,
			Volume:      decimal.Zero,
		}
		return interp.BoolVal(true), nil
	}
	_, err := vm.ctx.Broker().PositionClose(ticket, decimal.Zero)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("PositionClose: %w", err)
	}
	vm.invalidateOrderCaches()
	return interp.BoolVal(true), nil
}

func builtinCTradePositionClosePartial(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("trade operation: broker context is unavailable")
	}
	ticket := int64(argI(args, 0))
	volume := argD(args, 1)
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:      sdk.ActionClose,
			OrderTicket: ticket,
			Volume:      volume,
		}
		return interp.BoolVal(true), nil
	}
	_, err := vm.ctx.Broker().PositionClose(ticket, volume)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("PositionClose: %w", err)
	}
	vm.invalidateOrderCaches()
	return interp.BoolVal(true), nil
}

func builtinCTradePositionCloseBy(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("trade operation: broker context is unavailable")
	}
	t1 := int64(argI(args, 0))
	t2 := int64(argI(args, 1))
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:         sdk.ActionClose,
			OrderTicket:    t1,
			OppositeTicket: t2,
		}
		return interp.BoolVal(true), nil
	}
	_, err := vm.ctx.Broker().PositionCloseBy(t1, t2)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("PositionCloseBy: %w", err)
	}
	vm.invalidateOrderCaches()
	return interp.BoolVal(true), nil
}

func builtinCTradePositionModify(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("trade operation: broker context is unavailable")
	}
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
		return interp.BoolVal(true), nil
	}
	_, err := vm.ctx.Broker().PositionModify(ticket, sl, tp)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("PositionModify: %w", err)
	}
	vm.invalidateOrderCaches()
	return interp.BoolVal(true), nil
}

func builtinCTradeOrderDelete(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("trade operation: broker context is unavailable")
	}
	ticket := int64(argI(args, 0))
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action:      sdk.ActionCancel,
			OrderTicket: ticket,
		}
		return interp.BoolVal(true), nil
	}
	_, err := vm.ctx.Broker().OrderDelete(ticket)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("OrderDelete: %w", err)
	}
	vm.invalidateOrderCaches()
	return interp.BoolVal(true), nil
}

func builtinCloseAll(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("trade operation: broker context is unavailable")
	}
	if vm.signalMode {
		vm.signal = &sdk.Signal{
			Action: sdk.ActionCloseAll,
		}
		return interp.BoolVal(true), nil
	}
	positions := vm.ctx.Broker().Positions(0)
	for _, pos := range positions {
		if _, err := vm.ctx.Broker().PositionClose(pos.Ticket, decimal.Zero); err != nil {
			return interp.BoolVal(false), fmt.Errorf("PositionClose: %w", err)
		}
	}
	vm.invalidateOrderCaches()
	return interp.BoolVal(true), nil
}
