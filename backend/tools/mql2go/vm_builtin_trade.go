package mql2go

import (
	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// ── MQL4 trade builtins ──────────────────────────────────────────────

func builtinOrderSend(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.IntVal(-1), nil
	}
	// OrderSend(symbol, cmd, volume, price, slippage, sl, tp, comment, magic, expiration, color)
	symbol := argS(args, 0)
	cmd := argI(args, 1)
	volume := argD(args, 2)
	price := argD(args, 3)
	deviation := argI(args, 4)
	sl := argD(args, 5)
	tp := argD(args, 6)
	comment := argS(args, 7)
	magic := argI(args, 8)

	req := sdk.OrderRequest{
		Symbol:     symbol,
		Volume:     volume,
		Price:      price,
		StopLoss:   sl,
		TakeProfit: tp,
		Deviation:  deviation,
		Magic:      magic,
		Comment:    comment,
	}
	mapOrderCmd(cmd, &req)

	result, err := vm.ctx.Broker().OrderSend(req)
	if err != nil {
		return interp.IntVal(-1), nil
	}
	return interp.IntVal(int32(result.Ticket)), nil
}

func mapOrderCmd(cmd int32, req *sdk.OrderRequest) {
	switch cmd {
	case 0: // OP_BUY
		req.Type = sdk.OrderMarket
		req.Side = sdk.SideBuy
	case 1: // OP_SELL
		req.Type = sdk.OrderMarket
		req.Side = sdk.SideSell
	case 2: // OP_BUYLIMIT
		req.Type = sdk.OrderLimit
		req.Side = sdk.SideBuy
	case 3: // OP_SELLLIMIT
		req.Type = sdk.OrderLimit
		req.Side = sdk.SideSell
	case 4: // OP_BUYSTOP
		req.Type = sdk.OrderStop
		req.Side = sdk.SideBuy
	case 5: // OP_SELLSTOP
		req.Type = sdk.OrderStop
		req.Side = sdk.SideSell
	}
}

func builtinOrderClose(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), nil
	}
	// OrderClose(ticket, lots, price, slippage)
	ticket := int64(argI(args, 0))
	volume := argD(args, 1)
	_, err := vm.ctx.Broker().PositionClose(ticket, volume)
	return interp.BoolVal(err == nil), nil
}

func builtinOrderCloseBy(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), nil
	}
	// OrderCloseBy(ticket1, ticket2)
	ticket1 := int64(argI(args, 0))
	ticket2 := int64(argI(args, 1))
	_, err := vm.ctx.Broker().PositionCloseBy(ticket1, ticket2)
	return interp.BoolVal(err == nil), nil
}

func builtinOrderModify(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), nil
	}
	// OrderModify(ticket, price, sl, tp, expiration, color)
	ticket := int64(argI(args, 0))
	sl := argD(args, 2)
	tp := argD(args, 3)
	_, err := vm.ctx.Broker().PositionModify(ticket, sl, tp)
	return interp.BoolVal(err == nil), nil
}

func builtinOrderDelete(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), nil
	}
	ticket := int64(argI(args, 0))
	_, err := vm.ctx.Broker().OrderDelete(ticket)
	return interp.BoolVal(err == nil), nil
}

func builtinOrdersTotal(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.IntVal(0), nil
	}
	// Use cached positions if already loaded this event; otherwise load now.
	if vm.cachedPositions == nil {
		vm.cachedPositions = vm.ctx.Broker().Positions(0)
	}
	return interp.IntVal(int32(len(vm.cachedPositions))), nil
}

// builtinOrderSelect implements MQL4 OrderSelect(index, SELECT_BY_POS, MODE_TRADES).
// It sets currentPos to the i-th position from the cached list.
func builtinOrderSelect(vm *VM, args []interp.Value) (interp.Value, error) {
	index := int(argI(args, 0))
	// SELECT_BY_POS = 0, SELECT_BY_TICKET = 1
	selectBy := argI(args, 1)

	if selectBy == 0 {
		// SELECT_BY_POS — index into cached positions
		if vm.cachedPositions == nil && vm.ctx != nil && vm.ctx.Broker() != nil {
			vm.cachedPositions = vm.ctx.Broker().Positions(0)
		}
		if index >= 0 && index < len(vm.cachedPositions) {
			vm.currentPos = &vm.cachedPositions[index]
			return interp.BoolVal(true), nil
		}
		return interp.BoolVal(false), nil
	}

	// SELECT_BY_TICKET — find by ticket number
	ticket := int64(argI(args, 0))
	if vm.cachedPositions == nil && vm.ctx != nil && vm.ctx.Broker() != nil {
		vm.cachedPositions = vm.ctx.Broker().Positions(0)
	}
	for i := range vm.cachedPositions {
		if vm.cachedPositions[i].Ticket == ticket {
			vm.currentPos = &vm.cachedPositions[i]
			return interp.BoolVal(true), nil
		}
	}
	return interp.BoolVal(false), nil
}

// Order property functions — read from the VM's current position context.
// The VM stores the "current position" being iterated in currentPos.

func builtinOrderStopLoss(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.StopLoss), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderTakeProfit(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.TakeProfit), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderTicket(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.IntVal(int32(vm.currentPos.Ticket)), nil
	}
	return interp.IntVal(0), nil
}

func builtinOrderType(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.IntVal(int32(vm.currentPos.Side)), nil
	}
	return interp.IntVal(0), nil
}

func builtinOrderLots(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.Volume), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderSymbol(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.StringVal(vm.currentPos.Symbol), nil
	}
	return interp.StringVal(""), nil
}

func builtinOrderOpenPrice(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.OpenPrice), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderClosePrice(vm *VM, args []interp.Value) (interp.Value, error) {
	// MQL4: OrderClosePrice returns 0 for open orders, close price for closed orders.
	// Since our Position struct only tracks open positions, return 0.
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderProfit(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.Profit), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderMagicNumber(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.IntVal(vm.currentPos.Magic), nil
	}
	return interp.IntVal(0), nil
}

func builtinOrderComment(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.StringVal(vm.currentPos.Comment), nil
	}
	return interp.StringVal(""), nil
}

func builtinOrderOpenTime(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.IntVal(int32(vm.currentPos.OpenTime.Unix())), nil
	}
	return interp.IntVal(0), nil
}

func builtinOrderCloseTime(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

// ── MQL5 position builtins ───────────────────────────────────────────

func builtinPositionsTotal(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.IntVal(0), nil
	}
	if vm.cachedPositions == nil {
		vm.cachedPositions = vm.ctx.Broker().Positions(0)
	}
	return interp.IntVal(int32(len(vm.cachedPositions))), nil
}

func builtinPositionGetTicket(vm *VM, args []interp.Value) (interp.Value, error) {
	index := int(argI(args, 0))
	if vm.cachedPositions == nil && vm.ctx != nil && vm.ctx.Broker() != nil {
		vm.cachedPositions = vm.ctx.Broker().Positions(0)
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
		return interp.IntVal(int32(vm.currentPos.Side)), nil
	case 3: // POSITION_TIME
		return interp.DatetimeVal(vm.currentPos.OpenTime.UnixMilli()), nil
	default:
		return interp.IntVal(vm.currentPos.Magic), nil
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
	ticket := int64(argI(args, 0))
	if vm.cachedPositions == nil && vm.ctx != nil && vm.ctx.Broker() != nil {
		vm.cachedPositions = vm.ctx.Broker().Positions(0)
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
		return interp.BoolVal(false), nil
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
		Comment:    comment,
	}
	_, err := vm.ctx.Broker().OrderSend(req)
	if err != nil {
		return interp.BoolVal(false), nil
	}
	return interp.BoolVal(true), nil
}

func builtinCTradePositionClose(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), nil
	}
	ticket := int64(argI(args, 0))
	_, err := vm.ctx.Broker().PositionClose(ticket, decimal.Zero)
	return interp.BoolVal(err == nil), nil
}

func builtinCTradePositionClosePartial(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), nil
	}
	ticket := int64(argI(args, 0))
	volume := argD(args, 1)
	_, err := vm.ctx.Broker().PositionClose(ticket, volume)
	return interp.BoolVal(err == nil), nil
}

func builtinCTradePositionCloseBy(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), nil
	}
	t1 := int64(argI(args, 0))
	t2 := int64(argI(args, 1))
	_, err := vm.ctx.Broker().PositionCloseBy(t1, t2)
	return interp.BoolVal(err == nil), nil
}

func builtinCTradePositionModify(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), nil
	}
	ticket := int64(argI(args, 0))
	sl := argD(args, 1)
	tp := argD(args, 2)
	_, err := vm.ctx.Broker().PositionModify(ticket, sl, tp)
	return interp.BoolVal(err == nil), nil
}

func builtinCTradeOrderDelete(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.BoolVal(false), nil
	}
	ticket := int64(argI(args, 0))
	_, err := vm.ctx.Broker().OrderDelete(ticket)
	return interp.BoolVal(err == nil), nil
}
