package mql2go

import (
	"fmt"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// ── MQL4 trade builtins ──────────────────────────────────────────────

func builtinOrderSend(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.IntVal(-1), fmt.Errorf("OrderSend: broker context is unavailable")
	}
	if len(args) < 7 {
		return interp.IntVal(-1), fmt.Errorf("OrderSend: expected at least 7 arguments, got %d", len(args))
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
	if !mapOrderCmd(cmd, &req) {
		return interp.IntVal(-1), fmt.Errorf("OrderSend: unsupported order command %d", cmd)
	}

	if vm.signalMode {
		action := orderCmdToSignalAction(cmd)
		if action == sdk.ActionNone {
			return interp.IntVal(-1), nil
		}
		vm.signal = &sdk.Signal{
			Action:     action,
			Symbol:     symbol,
			Volume:     volume,
			Price:      price,
			StopLoss:   sl,
			TakeProfit: tp,
			Deviation:  deviation,
			Magic:      magic,
			Comment:    comment,
		}
		// Return a positive ticket so MQL logic that checks the result works.
		return interp.IntVal(1), nil
	}

	result, err := vm.ctx.Broker().OrderSend(req)
	if err != nil {
		return interp.IntVal(-1), fmt.Errorf("OrderSend: %w", err)
	}
	vm.invalidateOrderCaches()
	return interp.IntVal(int32(result.Ticket)), nil
}

func mapOrderCmd(cmd int32, req *sdk.OrderRequest) bool {
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
	default:
		return false
	}
	return true
}

// orderCmdToSignalAction converts an MQL4 OrderSend OP_* command to an
// sdk.SignalAction. Returns ActionNone for unknown commands.
func orderCmdToSignalAction(cmd int32) sdk.SignalAction {
	switch cmd {
	case 0:
		return sdk.ActionBuy
	case 1:
		return sdk.ActionSell
	case 2:
		return sdk.ActionBuyLimit
	case 3:
		return sdk.ActionSellLimit
	case 4:
		return sdk.ActionBuyStop
	case 5:
		return sdk.ActionSellStop
	default:
		return sdk.ActionNone
	}
}

func builtinOrdersTotal(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.IntVal(0), nil
	}
	// MQL4 OrdersTotal returns both open positions and pending orders in MODE_TRADES.
	if !vm.positionsLoaded {
		vm.cachedPositions = vm.ctx.Broker().Positions(0)
		vm.positionsLoaded = true
	}
	if !vm.ordersLoaded {
		vm.cachedOrders = vm.ctx.Broker().Orders(0)
		vm.ordersLoaded = true
	}
	return interp.IntVal(int32(len(vm.cachedPositions) + len(vm.cachedOrders))), nil
}

func builtinOrdersHistoryTotal(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.IntVal(0), nil
	}
	if !vm.historyLoaded {
		vm.cachedHistory = vm.ctx.Broker().HistoryOrders(0, 0)
		vm.historyLoaded = true
	}
	return interp.IntVal(int32(len(vm.cachedHistory))), nil
}

// builtinOrderSelect implements MQL4 OrderSelect(index, select, pool).
// pool: MODE_TRADES=0 (active positions), MODE_HISTORY=1 (closed orders).
// It sets currentPos to the i-th position from the appropriate cached list.
func builtinOrderSelect(vm *VM, args []interp.Value) (interp.Value, error) {
	// A failed selection must not leave properties pointing at the previous order.
	vm.currentPos = nil
	vm.currentOrder = nil
	index := int(argI(args, 0))
	// SELECT_BY_POS = 0, SELECT_BY_TICKET = 1
	selectBy := argI(args, 1)
	// pool: MODE_TRADES = 0, MODE_HISTORY = 1 (optional, defaults to MODE_TRADES)
	pool := int32(0)
	if len(args) > 2 {
		pool = argI(args, 2)
	}

	if pool == 1 {
		// MODE_HISTORY — select from closed orders
		if !vm.historyLoaded && vm.ctx != nil && vm.ctx.Broker() != nil {
			vm.cachedHistory = vm.ctx.Broker().HistoryOrders(0, 0)
			vm.historyLoaded = true
		}
		if selectBy == 0 {
			if index >= 0 && index < len(vm.cachedHistory) {
				vm.currentPos = &vm.cachedHistory[index]
				return interp.BoolVal(true), nil
			}
			return interp.BoolVal(false), nil
		}
		// SELECT_BY_TICKET
		ticket := int64(argI(args, 0))
		for i := range vm.cachedHistory {
			if vm.cachedHistory[i].Ticket == ticket {
				vm.currentPos = &vm.cachedHistory[i]
				return interp.BoolVal(true), nil
			}
		}
		return interp.BoolVal(false), nil
	}

	// MODE_TRADES (default)
	// MQL4 MODE_TRADES includes both open positions and pending orders.
	// Indexing: [0..len(positions)-1] = positions, [len(positions)..len(positions)+len(orders)-1] = pending orders.
	if !vm.positionsLoaded && vm.ctx != nil && vm.ctx.Broker() != nil {
		vm.cachedPositions = vm.ctx.Broker().Positions(0)
		vm.positionsLoaded = true
	}
	if !vm.ordersLoaded && vm.ctx != nil && vm.ctx.Broker() != nil {
		vm.cachedOrders = vm.ctx.Broker().Orders(0)
		vm.ordersLoaded = true
	}

	if selectBy == 0 {
		// SELECT_BY_POS — index into combined positions + pending orders
		posCount := len(vm.cachedPositions)
		if index >= 0 && index < posCount {
			vm.currentPos = &vm.cachedPositions[index]
			vm.currentOrder = nil
			return interp.BoolVal(true), nil
		}
		orderIdx := index - posCount
		if orderIdx >= 0 && orderIdx < len(vm.cachedOrders) {
			vm.currentOrder = &vm.cachedOrders[orderIdx]
			vm.currentPos = nil
			return interp.BoolVal(true), nil
		}
		return interp.BoolVal(false), nil
	}

	// SELECT_BY_TICKET — find by ticket number in positions then pending orders
	ticket := int64(argI(args, 0))
	for i := range vm.cachedPositions {
		if vm.cachedPositions[i].Ticket == ticket {
			vm.currentPos = &vm.cachedPositions[i]
			vm.currentOrder = nil
			return interp.BoolVal(true), nil
		}
	}
	for i := range vm.cachedOrders {
		if vm.cachedOrders[i].Ticket == ticket {
			vm.currentOrder = &vm.cachedOrders[i]
			vm.currentPos = nil
			return interp.BoolVal(true), nil
		}
	}
	return interp.BoolVal(false), nil
}

// ── MQL5 position builtins ───────────────────────────────────────────
