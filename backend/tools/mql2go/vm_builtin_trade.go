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
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.IntVal(1), nil
	}

	result, err := vm.ctx.Broker().OrderSend(req)
	if err != nil {
		return interp.IntVal(-1), nil
	}
	vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
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
	if vm.cachedPositions == nil {
		vm.cachedPositions = vm.ctx.Broker().Positions(0)
	}
	if vm.cachedOrders == nil {
		vm.cachedOrders = vm.ctx.Broker().Orders(0)
	}
	return interp.IntVal(int32(len(vm.cachedPositions) + len(vm.cachedOrders))), nil
}

func builtinOrdersHistoryTotal(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.IntVal(0), nil
	}
	if vm.cachedHistory == nil {
		vm.cachedHistory = vm.ctx.Broker().HistoryOrders(0, 0)
	}
	return interp.IntVal(int32(len(vm.cachedHistory))), nil
}

// builtinOrderSelect implements MQL4 OrderSelect(index, select, pool).
// pool: MODE_TRADES=0 (active positions), MODE_HISTORY=1 (closed orders).
// It sets currentPos to the i-th position from the appropriate cached list.
func builtinOrderSelect(vm *VM, args []interp.Value) (interp.Value, error) {
	// VM-TRADE-CONTEXT-1: reset selection state — failed select must not leave stale currentPos/currentOrder.
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
		if vm.cachedHistory == nil && vm.ctx != nil && vm.ctx.Broker() != nil {
			vm.cachedHistory = vm.ctx.Broker().HistoryOrders(0, 0)
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
	if vm.cachedPositions == nil && vm.ctx != nil && vm.ctx.Broker() != nil {
		vm.cachedPositions = vm.ctx.Broker().Positions(0)
	}
	if vm.cachedOrders == nil && vm.ctx != nil && vm.ctx.Broker() != nil {
		vm.cachedOrders = vm.ctx.Broker().Orders(0)
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

// Order property functions — read from the VM's current position context.
// The VM stores the "current position" being iterated in currentPos.

func builtinOrderStopLoss(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.DecimalVal(vm.currentOrder.StopLoss), nil
	}
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.StopLoss), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderTakeProfit(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.DecimalVal(vm.currentOrder.TakeProfit), nil
	}
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.TakeProfit), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderTicket(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.IntVal(int32(vm.currentOrder.Ticket)), nil
	}
	if vm.currentPos != nil {
		return interp.IntVal(int32(vm.currentPos.Ticket)), nil
	}
	return interp.IntVal(0), nil
}

// sideToOrderType maps sdk.PositionSide (SideBuy=1, SideSell=-1) to MQL4 OP_* constants.
// MQL4: OP_BUY=0, OP_SELL=1. MQL5 POSITION_TYPE: POSITION_TYPE_BUY=0, POSITION_TYPE_SELL=1.
func sideToOrderType(side sdk.PositionSide) int32 {
	switch side {
	case sdk.SideBuy:
		return 0 // OP_BUY / POSITION_TYPE_BUY
	case sdk.SideSell:
		return 1 // OP_SELL / POSITION_TYPE_SELL
	default:
		return -1
	}
}

// orderTypeToMQL4 maps sdk.OrderType + side to MQL4 OP_* constants.
// MQL4: OP_BUYLIMIT=2, OP_SELLLIMIT=3, OP_BUYSTOP=4, OP_SELLSTOP=5.
func orderTypeToMQL4(ot sdk.OrderType, side sdk.PositionSide) int32 {
	switch ot {
	case sdk.OrderLimit:
		if side == sdk.SideSell {
			return 3 // OP_SELLLIMIT
		}
		return 2 // OP_BUYLIMIT
	case sdk.OrderStop:
		if side == sdk.SideSell {
			return 5 // OP_SELLSTOP
		}
		return 4 // OP_BUYSTOP
	default:
		return 0
	}
}

// pendingPriceModifier allows modifying the price of a pending order.
// SimBroker implements this; live broker may not.
type pendingPriceModifier interface {
	PositionModifyPrice(ticket int64, price decimal.Decimal) (sdk.OrderResult, error)
}

func builtinOrderType(vm *VM, args []interp.Value) (interp.Value, error) {
	// Pending orders: return MQL4 OP_* constants for limit/stop types.
	if vm.currentOrder != nil {
		return interp.IntVal(orderTypeToMQL4(vm.currentOrder.Type, vm.currentOrder.Side)), nil
	}
	// Open positions: return OP_BUY/OP_SELL.
	if vm.currentPos != nil {
		return interp.IntVal(sideToOrderType(vm.currentPos.Side)), nil
	}
	return interp.IntVal(0), nil
}

func builtinOrderLots(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.DecimalVal(vm.currentOrder.Volume), nil
	}
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.Volume), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderSymbol(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.StringVal(vm.currentOrder.Symbol), nil
	}
	if vm.currentPos != nil {
		return interp.StringVal(vm.currentPos.Symbol), nil
	}
	return interp.StringVal(""), nil
}

func builtinOrderOpenPrice(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.DecimalVal(vm.currentOrder.Price), nil
	}
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.OpenPrice), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderClosePrice(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// For closed positions (from history pool), return the recorded close price.
	if vm.currentPos.ClosePrice.IsPositive() {
		return interp.DecimalVal(vm.currentPos.ClosePrice), nil
	}
	// For open positions, return current market price.
	if vm.ctx != nil {
		if vm.currentPos.Side == sdk.SideSell {
			return interp.DecimalVal(vm.ctx.Ask()), nil
		}
		return interp.DecimalVal(vm.ctx.Bid()), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderProfit(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// For closed positions, use the recorded close price.
	closePrice := vm.currentPos.ClosePrice
	// For open positions, use current market price.
	if !closePrice.IsPositive() && vm.ctx != nil {
		closePrice = vm.ctx.Bid()
		if vm.currentPos.Side == sdk.SideSell {
			closePrice = vm.ctx.Ask()
		}
	}
	if !closePrice.IsPositive() {
		return interp.DecimalVal(vm.currentPos.Profit), nil
	}
	contractSize := decimal.NewFromInt(100000)
	if vm.ctx != nil {
		if info, err := vm.ctx.Broker().SymbolInfo(vm.currentPos.Symbol); err == nil && info.ContractSize.IsPositive() {
			contractSize = info.ContractSize
		}
	}
	profit := closePrice.Sub(vm.currentPos.OpenPrice).Mul(vm.currentPos.Volume).Mul(contractSize)
	if vm.currentPos.Side == sdk.SideSell {
		profit = profit.Neg()
	}
	return interp.DecimalVal(profit), nil
}

func builtinOrderMagicNumber(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.IntVal(vm.currentOrder.Magic), nil
	}
	if vm.currentPos != nil {
		return interp.IntVal(vm.currentPos.Magic), nil
	}
	return interp.IntVal(0), nil
}

func builtinOrderComment(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.StringVal(vm.currentOrder.Comment), nil
	}
	if vm.currentPos != nil {
		return interp.StringVal(vm.currentPos.Comment), nil
	}
	return interp.StringVal(""), nil
}

func builtinOrderOpenTime(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentOrder != nil {
		return interp.IntVal(int32(vm.currentOrder.OpenTime.Unix())), nil
	}
	if vm.currentPos != nil {
		return interp.IntVal(int32(vm.currentPos.OpenTime.Unix())), nil
	}
	return interp.IntVal(0), nil
}

func builtinOrderCloseTime(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil && !vm.currentPos.CloseTime.IsZero() {
		return interp.IntVal(int32(vm.currentPos.CloseTime.Unix())), nil
	}
	return interp.IntVal(0), nil
}

func builtinOrderCommission(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.Commission), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinOrderSwap(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.currentPos != nil {
		return interp.DecimalVal(vm.currentPos.Swap), nil
	}
	return interp.DecimalVal(decimal.Zero), nil
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
	case 3: // POSITION_TIME
		return interp.DatetimeVal(vm.currentPos.OpenTime.UnixMilli()), nil
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
		Magic:      vm.tradeMagic,     // VM-TRADE-CONTEXT-1
		Deviation:  vm.tradeDeviation, // VM-TRADE-CONTEXT-1
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
			Comment:    comment,
			Magic:      vm.tradeMagic,     // VM-TRADE-CONTEXT-1
			Deviation:  vm.tradeDeviation, // VM-TRADE-CONTEXT-1
		}
		vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
		return interp.BoolVal(true), nil
	}

	_, err := vm.ctx.Broker().OrderSend(req)
	if err != nil {
		return interp.BoolVal(false), nil
	}
	vm.invalidateOrderCaches() // VM-TRADE-CONTEXT-1
	return interp.BoolVal(true), nil
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

// builtinCTradeSetExpertMagicNumber stores the CTrade magic in VM state
// (VM-TRADE-CONTEXT-1). ctradeOrder reads it for subsequent OrderSend/Signal.
func builtinCTradeSetExpertMagicNumber(vm *VM, args []interp.Value) (interp.Value, error) {
	vm.tradeMagic = argI(args, 0)
	return interp.NoneVal(), nil
}

// builtinCTradeSetDeviationInPoints stores the CTrade deviation in VM state
// (VM-TRADE-CONTEXT-1). ctradeOrder reads it for subsequent OrderSend/Signal.
func builtinCTradeSetDeviationInPoints(vm *VM, args []interp.Value) (interp.Value, error) {
	vm.tradeDeviation = argI(args, 0)
	return interp.NoneVal(), nil
}
