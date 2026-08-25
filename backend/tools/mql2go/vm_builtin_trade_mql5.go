package mql2go

import (
	"fmt"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// ── MQL5 Position* and CTrade builtins ───────────────────────────────

func builtinPositionsTotal(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.IntVal(0), nil
	}
	if !vm.positionsLoaded {
		vm.cachedPositions = vm.ctx.Broker().Positions(0)
		vm.positionsLoaded = true
	}
	return interp.IntVal(int32(len(vm.cachedPositions))), nil
}

func builtinPositionGetTicket(vm *VM, args []interp.Value) (interp.Value, error) {
	vm.currentPos = nil
	vm.currentOrder = nil
	index := int(argI(args, 0))
	if !vm.positionsLoaded && vm.ctx != nil && vm.ctx.Broker() != nil {
		vm.cachedPositions = vm.ctx.Broker().Positions(0)
		vm.positionsLoaded = true
	}
	if index >= 0 && index < len(vm.cachedPositions) {
		vm.currentPos = &vm.cachedPositions[index]
		return interp.IntVal(int32(vm.currentPos.Ticket)), nil
	}
	return interp.IntVal(0), nil
}

func builtinPositionGetDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	prop := argI(args, 0)
	switch prop {
	case 0: // POSITION_VOLUME
		return interp.DecimalVal(vm.currentPos.Volume), nil
	case 1: // POSITION_PRICE_OPEN
		return interp.DecimalVal(vm.currentPos.OpenPrice), nil
	case 2: // POSITION_SL
		return interp.DecimalVal(vm.currentPos.StopLoss), nil
	case 3: // POSITION_TP
		return interp.DecimalVal(vm.currentPos.TakeProfit), nil
	case 4: // POSITION_PRICE_CURRENT
		if vm.ctx != nil {
			if vm.currentPos.Side == sdk.SideSell {
				return interp.DecimalVal(vm.ctx.Ask()), nil
			}
			return interp.DecimalVal(vm.ctx.Bid()), nil
		}
		return interp.DecimalVal(vm.currentPos.OpenPrice), nil
	case 5: // POSITION_SWAP
		return interp.DecimalVal(vm.currentPos.Swap), nil
	case 6: // POSITION_COMMISSION
		return interp.DecimalVal(vm.currentPos.Commission), nil
	case 7: // POSITION_PROFIT
		return interp.DecimalVal(vm.currentPos.Profit), nil
	default:
		return interp.DecimalVal(decimal.Zero), nil
	}
}

func builtinPositionGetInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos == nil {
		return interp.IntVal(0), nil
	}
	prop := argI(args, 0)
	switch prop {
	case 0: // POSITION_TICKET
		return interp.IntVal(int32(vm.currentPos.Ticket)), nil
	case 1: // POSITION_MAGIC
		return interp.IntVal(vm.currentPos.Magic), nil
	case 2: // POSITION_TYPE
		return interp.IntVal(sideToOrderType(vm.currentPos.Side)), nil
	case 3: // POSITION_TIME (MQL datetime is unix seconds)
		return interp.IntVal(int32(vm.currentPos.OpenTime.Unix())), nil
	default:
		return interp.IntVal(0), nil
	}
}

func builtinPositionGetString(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		prop := argI(args, 0)
		switch prop {
		case 0: // POSITION_SYMBOL
			return interp.StringVal(vm.currentPos.Symbol), nil
		case 1: // POSITION_COMMENT
			return interp.StringVal(vm.currentPos.Comment), nil
		default:
			return interp.StringVal(vm.currentPos.Symbol), nil
		}
	}
	return interp.StringVal(""), nil
}

func builtinPositionGetSymbol(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.StringVal(vm.currentPos.Symbol), nil
	}
	return interp.StringVal(""), nil
}

func builtinPositionSelectByTicket(vm *VM, args []interp.Value) (interp.Value, error) {
	vm.currentPos = nil
	vm.currentOrder = nil
	ticket := int64(argI(args, 0))
	if !vm.positionsLoaded && vm.ctx != nil && vm.ctx.Broker() != nil {
		vm.cachedPositions = vm.ctx.Broker().Positions(0)
		vm.positionsLoaded = true
	}
	for i := range vm.cachedPositions {
		if vm.cachedPositions[i].Ticket == ticket {
			vm.currentPos = &vm.cachedPositions[i]
			return interp.BoolVal(true), nil
		}
	}
	return interp.BoolVal(false), nil
}

// ── MQL5 CTrade method builtins ──────────────────────────────────────

func builtinCTradeBuy(vm *VM, args []interp.Value) (interp.Value, error) {
	return ctradeOrder(vm, args, sdk.OrderMarket, sdk.SideBuy)
}

func builtinCTradeSell(vm *VM, args []interp.Value) (interp.Value, error) {
	return ctradeOrder(vm, args, sdk.OrderMarket, sdk.SideSell)
}

func builtinCTradeBuyLimit(vm *VM, args []interp.Value) (interp.Value, error) {
	return ctradeOrder(vm, args, sdk.OrderLimit, sdk.SideBuy)
}

func builtinCTradeSellLimit(vm *VM, args []interp.Value) (interp.Value, error) {
	return ctradeOrder(vm, args, sdk.OrderLimit, sdk.SideSell)
}

func builtinCTradeBuyStop(vm *VM, args []interp.Value) (interp.Value, error) {
	return ctradeOrder(vm, args, sdk.OrderStop, sdk.SideBuy)
}

func builtinCTradeSellStop(vm *VM, args []interp.Value) (interp.Value, error) {
	return ctradeOrder(vm, args, sdk.OrderStop, sdk.SideSell)
}

func ctradeOrder(vm *VM, args []interp.Value, orderType sdk.OrderType, side sdk.PositionSide) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), fmt.Errorf("CTrade order: broker context is unavailable")
	}
	if len(args) == 0 {
		return interp.BoolVal(false), fmt.Errorf("CTrade order: volume is required")
	}
	// CTrade.Buy(volume, symbol, price, sl, tp, comment)
	volume := argD(args, 0)
	symbol := argS(args, 1)
	if symbol == "" && vm.ctx != nil {
		symbol = vm.ctx.Symbol()
	}
	price := argD(args, 2)
	sl := argD(args, 3)
	tp := argD(args, 4)
	comment := argS(args, 5)

	req := sdk.OrderRequest{
		Symbol:     symbol,
		Type:       orderType,
		Side:       side,
		Volume:     volume,
		Price:      price,
		StopLoss:   sl,
		TakeProfit: tp,
		Deviation:  vm.tradeDeviation,
		Magic:      vm.tradeMagic,
		Comment:    comment,
	}

	if vm.signalMode {
		action := ctradeTypeToSignalAction(orderType, side)
		if action == sdk.ActionNone {
			return interp.BoolVal(false), nil
		}
		vm.signal = &sdk.Signal{
			Action:     action,
			Symbol:     symbol,
			Volume:     volume,
			Price:      price,
			StopLoss:   sl,
			TakeProfit: tp,
			Deviation:  vm.tradeDeviation,
			Magic:      vm.tradeMagic,
			Comment:    comment,
		}
		return interp.BoolVal(true), nil
	}

	_, err := vm.ctx.Broker().OrderSend(req)
	if err != nil {
		return interp.BoolVal(false), fmt.Errorf("CTrade order: %w", err)
	}
	vm.invalidateOrderCaches()
	return interp.BoolVal(true), nil
}

func builtinCTradeSetExpertMagicNumber(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 1 {
		return interp.NoneVal(), fmt.Errorf("CTrade.SetExpertMagicNumber: magic is required")
	}
	vm.tradeMagic = argI(args, 0)
	return interp.NoneVal(), nil
}

func builtinCTradeSetDeviationInPoints(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 1 {
		return interp.NoneVal(), fmt.Errorf("CTrade.SetDeviationInPoints: deviation is required")
	}
	vm.tradeDeviation = argI(args, 0)
	return interp.NoneVal(), nil
}

// ctradeTypeToSignalAction converts CTrade order type + side to sdk.SignalAction.
func ctradeTypeToSignalAction(orderType sdk.OrderType, side sdk.PositionSide) sdk.SignalAction {
	switch orderType {
	case sdk.OrderMarket:
		if side == sdk.SideSell {
			return sdk.ActionSell
		}
		return sdk.ActionBuy
	case sdk.OrderLimit:
		if side == sdk.SideSell {
			return sdk.ActionSellLimit
		}
		return sdk.ActionBuyLimit
	case sdk.OrderStop:
		if side == sdk.SideSell {
			return sdk.ActionSellStop
		}
		return sdk.ActionBuyStop
	default:
		return sdk.ActionNone
	}
}
