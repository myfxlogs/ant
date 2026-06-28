package interp

import (
	"github.com/shopspring/decimal"
	"anttrader/strategy/sdk"
)

// execCTrade dispatches MQL5 CTrade method calls.
// CTrade.Buy/Sell/BuyLimit/SellLimit/BuyStop/SellStop
// CTrade.PositionClose/PositionClosePartial/PositionCloseBy
// CTrade.PositionModify
// CTrade.OrderDelete
// CTrade.SetExpertMagicNumber
func (it *Interpreter) execCTrade(cls *ClassInstance, method string, args []Expr) Value {
	if it.ctx == nil || it.ctx.Broker() == nil {
		return NoneVal()
	}
	broker := it.ctx.Broker()

	// Get magic number from CTrade instance fields
	magic := int32(0)
	if v, ok := cls.Fields["magic"]; ok {
		magic = v.ToInt()
	}

	switch method {
	// ── Market orders ────────────────────────────────────────────────
	case "Buy":
		req := buildCTradeRequest(it, args, sdk.OrderMarket, sdk.SideBuy, magic)
		_, err := broker.OrderSend(req)
		return BoolVal(err == nil)

	case "Sell":
		req := buildCTradeRequest(it, args, sdk.OrderMarket, sdk.SideSell, magic)
		_, err := broker.OrderSend(req)
		return BoolVal(err == nil)

	// ── Pending orders ───────────────────────────────────────────────
	case "BuyLimit":
		req := buildCTradeRequest(it, args, sdk.OrderLimit, sdk.SideBuy, magic)
		_, err := broker.OrderSend(req)
		return BoolVal(err == nil)

	case "SellLimit":
		req := buildCTradeRequest(it, args, sdk.OrderLimit, sdk.SideSell, magic)
		_, err := broker.OrderSend(req)
		return BoolVal(err == nil)

	case "BuyStop":
		req := buildCTradeRequest(it, args, sdk.OrderStop, sdk.SideBuy, magic)
		_, err := broker.OrderSend(req)
		return BoolVal(err == nil)

	case "SellStop":
		req := buildCTradeRequest(it, args, sdk.OrderStop, sdk.SideSell, magic)
		_, err := broker.OrderSend(req)
		return BoolVal(err == nil)

	// ── Position close ───────────────────────────────────────────────
	case "PositionClose":
		if len(args) >= 1 {
			ticket := int64(it.evalExpr(&args[0]).ToInt())
			_, err := broker.PositionClose(ticket, decimal.Zero)
			return BoolVal(err == nil)
		}
	case "PositionClosePartial":
		if len(args) >= 2 {
			ticket := int64(it.evalExpr(&args[0]).ToInt())
			vol := it.evalExpr(&args[1]).ToDecimal()
			_, err := broker.PositionClose(ticket, vol)
			return BoolVal(err == nil)
		}
	case "PositionCloseBy":
		if len(args) >= 1 {
			ticket := int64(it.evalExpr(&args[0]).ToInt())
			_, err := broker.PositionClose(ticket, decimal.Zero)
			return BoolVal(err == nil)
		}

	// ── Position modify ──────────────────────────────────────────────
	case "PositionModify":
		if len(args) >= 3 {
			ticket := int64(it.evalExpr(&args[0]).ToInt())
			sl := it.evalExpr(&args[1]).ToDecimal()
			tp := it.evalExpr(&args[2]).ToDecimal()
			_, err := broker.PositionModify(ticket, sl, tp)
			return BoolVal(err == nil)
		}

	// ── Order delete ─────────────────────────────────────────────────
	case "OrderDelete":
		if len(args) >= 1 {
			ticket := int64(it.evalExpr(&args[0]).ToInt())
			_, err := broker.OrderDelete(ticket)
			return BoolVal(err == nil)
		}

	// ── Settings ─────────────────────────────────────────────────────
	case "SetExpertMagicNumber":
		if len(args) >= 1 {
			cls.Fields["magic"] = it.evalExpr(&args[0])
		}
		return BoolVal(true)

	case "SetDeviationInPoints":
		if len(args) >= 1 {
			cls.Fields["deviation"] = it.evalExpr(&args[0])
		}
		return BoolVal(true)
	}

	return NoneVal()
}

// buildCTradeRequest constructs an sdk.OrderRequest from CTrade method args.
// CTrade.Buy(volume, symbol, price, sl, tp, comment)
func buildCTradeRequest(it *Interpreter, args []Expr, orderType sdk.OrderType, side sdk.PositionSide, magic int32) sdk.OrderRequest {
	req := sdk.OrderRequest{Type: orderType, Side: side, Magic: magic}
	if len(args) >= 1 {
		req.Volume = it.evalExpr(&args[0]).ToDecimal()
	}
	if len(args) >= 2 {
		symVal := it.evalExpr(&args[1])
		if symVal.Kind == ValString && symVal.Str != "" {
			req.Symbol = symVal.Str
		} else if it.ctx != nil {
			req.Symbol = it.ctx.Symbol()
		}
	} else if it.ctx != nil {
		req.Symbol = it.ctx.Symbol()
	}
	if len(args) >= 3 {
		req.Price = it.evalExpr(&args[2]).ToDecimal()
	}
	if len(args) >= 4 {
		req.StopLoss = it.evalExpr(&args[3]).ToDecimal()
	}
	if len(args) >= 5 {
		req.TakeProfit = it.evalExpr(&args[4]).ToDecimal()
	}
	if len(args) >= 6 {
		req.Comment = it.evalExpr(&args[5]).ToString()
	}
	return req
}
