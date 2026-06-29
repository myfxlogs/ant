package interp

import (
	"github.com/shopspring/decimal"
	"anttrader/strategy/sdk"
)

// callTrade implements MQL4 trading functions: OrderSend, OrderClose,
// OrderModify, OrderDelete, OrderSelect, OrdersTotal, and all Order* property functions.
func (it *Interpreter) callTrade(name string, args []Expr) (Value, bool) {
	if it.ctx == nil {
		return NoneVal(), false
	}
	broker := it.ctx.Broker()

	switch name {
	// ── Order placement ──────────────────────────────────────────────
	case "OrderSend":
		if broker == nil {
			return IntVal(-1), true
		}
		return it.execOrderSend(args), true

	// ── Order close/modify/delete ────────────────────────────────────
	case "OrderClose":
		if broker == nil {
			return BoolVal(false), true
		}
		return it.execOrderClose(args), true
	case "OrderCloseBy":
		if broker == nil {
			return BoolVal(false), true
		}
		return it.execOrderCloseBy(args), true
	case "OrderModify":
		if broker == nil {
			return BoolVal(false), true
		}
		return it.execOrderModify(args), true
	case "OrderDelete":
		if broker == nil {
			return BoolVal(false), true
		}
		return it.execOrderDelete(args), true

	// ── Order selection ──────────────────────────────────────────────
	case "OrdersTotal":
		return IntVal(int32(it.orderPool.Total())), true
	case "OrdersHistoryTotal":
		return IntVal(int32(it.orderPool.HistoryTotal())), true
	case "OrderSelect":
		return it.execOrderSelect(args), true

	// ── Order property functions (MQL4) ──────────────────────────────
	case "OrderTicket":
		return IntVal(int32(it.orderPool.Ticket())), true
	case "OrderSymbol":
		return StringVal(it.orderPool.Symbol()), true
	case "OrderType":
		return IntVal(it.orderPool.Type()), true
	case "OrderLots":
		return it.orderPool.Lots(), true
	case "OrderOpenPrice":
		return it.orderPool.OpenPrice(), true
	case "OrderStopLoss":
		return it.orderPool.StopLoss(), true
	case "OrderTakeProfit":
		return it.orderPool.TakeProfit(), true
	case "OrderProfit":
		return it.orderPool.Profit(), true
	case "OrderCommission":
		return it.orderPool.Commission(), true
	case "OrderSwap":
		return it.orderPool.Swap(), true
	case "OrderMagicNumber":
		return IntVal(it.orderPool.MagicNumber()), true
	case "OrderComment":
		return StringVal(it.orderPool.Comment()), true
	case "OrderOpenTime":
		return Value{Kind: ValDatetime, Datetime: it.orderPool.OpenTime()}, true
	case "OrderCloseTime":
		return Value{Kind: ValDatetime, Datetime: it.orderPool.CloseTime()}, true
	case "OrderClosePrice":
		return it.orderPool.ClosePrice(), true

	// ── MQL5 position functions ──────────────────────────────────────
	case "PositionsTotal":
		return IntVal(int32(it.posPool.Total())), true
	case "PositionGetTicket":
		if len(args) >= 1 {
			idx := int(it.evalExpr(&args[0]).ToInt())
			return IntVal(int32(it.posPool.GetTicket(idx))), true
		}
		return IntVal(0), true
	case "PositionSelectByTicket":
		if len(args) >= 1 {
			ticket := it.evalExpr(&args[0]).ToInt()
			return BoolVal(it.posPool.SelectByTicket(int64(ticket))), true
		}
		return BoolVal(false), true
	case "PositionGetSymbol":
		return StringVal(it.posPool.Symbol()), true
	case "PositionGetDouble":
		if len(args) >= 1 {
			prop := it.evalExpr(&args[0]).ToInt()
			return it.posPool.GetDouble(prop), true
		}
		return NoneVal(), true
	case "PositionGetInteger":
		if len(args) >= 1 {
			prop := it.evalExpr(&args[0]).ToInt()
			return Value{Kind: ValInt, Int: int32(it.posPool.GetInteger(prop))}, true
		}
		return IntVal(0), true
	case "PositionGetString":
		if len(args) >= 1 {
			prop := it.evalExpr(&args[0]).ToInt()
			return StringVal(it.posPool.GetString(prop)), true
		}
		return StringVal(""), true

	// ── Account functions ────────────────────────────────────────────
	case "AccountBalance":
		return DecimalVal(it.ctx.Account().Balance), true
	case "AccountEquity":
		return DecimalVal(it.ctx.Account().Equity), true
	case "AccountFreeMargin":
		return DecimalVal(it.ctx.Account().FreeMargin), true
	case "AccountMargin":
		return DecimalVal(it.ctx.Account().Margin), true
	case "AccountLeverage":
		return IntVal(it.ctx.Account().Leverage), true
	case "AccountNumber":
		return IntVal(0), true // backtest has no real account number
	case "AccountStopoutLevel":
		return IntVal(0), true // backtest has no stopout level
	case "AccountCurrency":
		return StringVal("USD"), true
	case "AccountName":
		return StringVal("Backtest"), true
	case "AccountCompany":
		return StringVal("SimBroker"), true
	}

	return NoneVal(), false
}

// execOrderSend implements MQL4 OrderSend(symbol, cmd, volume, price, slippage, sl, tp, comment, magic, expiration, color)
func (it *Interpreter) execOrderSend(args []Expr) Value {
	if len(args) < 4 {
		return IntVal(-1)
	}
	symbol := it.evalExpr(&args[0]).ToString()
	cmd := it.evalExpr(&args[1]).ToInt()
	volume := it.evalExpr(&args[2]).ToDecimal()
	price := it.evalExpr(&args[3]).ToDecimal()

	var sl, tp decimal.Decimal
	var deviation int32
	var magic int32
	var comment string

	if len(args) >= 5 {
		deviation = it.evalExpr(&args[4]).ToInt()
	}
	if len(args) >= 6 {
		sl = it.evalExpr(&args[5]).ToDecimal()
	}
	if len(args) >= 7 {
		tp = it.evalExpr(&args[6]).ToDecimal()
	}
	if len(args) >= 8 {
		comment = it.evalExpr(&args[7]).ToString()
	}
	if len(args) >= 9 {
		magic = it.evalExpr(&args[8]).ToInt()
	}

	req := sdk.OrderRequest{
		Symbol:    symbol,
		Volume:    volume,
		Price:     price,
		StopLoss:  sl,
		TakeProfit: tp,
		Deviation: deviation,
		Magic:     magic,
		Comment:   comment,
	}

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

	result, err := it.ctx.Broker().OrderSend(req)
	if err != nil {
		return IntVal(-1)
	}
	return IntVal(int32(result.Ticket))
}

// execOrderClose implements MQL4 OrderClose(ticket, lots, price, slippage)
func (it *Interpreter) execOrderClose(args []Expr) Value {
	if len(args) < 3 {
		return BoolVal(false)
	}
	ticket := int64(it.evalExpr(&args[0]).ToInt())
	lots := it.evalExpr(&args[1]).ToDecimal()

	_, err := it.ctx.Broker().PositionClose(ticket, lots)
	return BoolVal(err == nil)
}

// execOrderModify implements MQL4 OrderModify(ticket, price, sl, tp, expiration, color)
func (it *Interpreter) execOrderModify(args []Expr) Value {
	if len(args) < 4 {
		return BoolVal(false)
	}
	ticket := int64(it.evalExpr(&args[0]).ToInt())
	sl := it.evalExpr(&args[2]).ToDecimal()
	tp := it.evalExpr(&args[3]).ToDecimal()

	_, err := it.ctx.Broker().PositionModify(ticket, sl, tp)
	return BoolVal(err == nil)
}

// execOrderDelete implements MQL4 OrderDelete(ticket)
func (it *Interpreter) execOrderDelete(args []Expr) Value {
	if len(args) < 1 {
		return BoolVal(false)
	}
	ticket := int64(it.evalExpr(&args[0]).ToInt())

	_, err := it.ctx.Broker().OrderDelete(ticket)
	return BoolVal(err == nil)
}

// execOrderCloseBy implements MQL4 OrderCloseBy(ticket1, ticket2, color)
func (it *Interpreter) execOrderCloseBy(args []Expr) Value {
	if len(args) < 2 {
		return BoolVal(false)
	}
	ticket1 := int64(it.evalExpr(&args[0]).ToInt())
	ticket2 := int64(it.evalExpr(&args[1]).ToInt())

	_, err := it.ctx.Broker().PositionCloseBy(ticket1, ticket2)
	return BoolVal(err == nil)
}

// execOrderSelect implements OrderSelect(index, select, pool)
func (it *Interpreter) execOrderSelect(args []Expr) Value {
	if len(args) < 2 {
		return BoolVal(false)
	}
	index := int(it.evalExpr(&args[0]).ToInt())
	selectMode := it.evalExpr(&args[1]).ToInt()

	// Third argument: pool mode (MODE_TRADES=0, MODE_HISTORY=1)
	poolMode := int32(0) // default MODE_TRADES
	if len(args) >= 3 {
		poolMode = it.evalExpr(&args[2]).ToInt()
	}

	switch selectMode {
	case 0: // SELECT_BY_POS
		if poolMode == 1 { // MODE_HISTORY
			return BoolVal(it.orderPool.SelectHistory(index))
		}
		return BoolVal(it.orderPool.Select(index))
	case 1: // SELECT_BY_TICKET
		return BoolVal(it.orderPool.SelectByTicket(int64(index)))
	}
	return BoolVal(false)
}
