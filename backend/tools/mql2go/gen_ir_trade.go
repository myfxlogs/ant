package mql2go

import (
	"strings"

	"anttrader/tools/mql2go/interp"
)

// ── IR field access & CTrade method mapping ────────────────────────

func (g *irGenerator) emitField(e *interp.Expr) string {
	if len(e.Args) == 0 {
		return ""
	}
	obj := g.expr(&e.Args[0])
	// Field assignment: obj.field = value
	if e.IsAssign && len(e.Args) > 1 {
		val := g.expr(&e.Args[len(e.Args)-1])
		return obj + "." + e.Name + " = " + val
	}
	// Method call (has additional args beyond object)
	if len(e.Args) > 1 {
		argStrs := make([]string, 0, len(e.Args)-1)
		for i := 1; i < len(e.Args); i++ {
			argStrs = append(argStrs, g.expr(&e.Args[i]))
		}
		switch e.Name {
		case "Buy":
			return g.emitCTradeOrder(argStrs, "sdk.OrderMarket", "sdk.SideBuy")
		case "Sell":
			return g.emitCTradeOrder(argStrs, "sdk.OrderMarket", "sdk.SideSell")
		case "BuyLimit":
			return g.emitCTradeOrder(argStrs, "sdk.OrderLimit", "sdk.SideBuy")
		case "SellLimit":
			return g.emitCTradeOrder(argStrs, "sdk.OrderLimit", "sdk.SideSell")
		case "BuyStop":
			return g.emitCTradeOrder(argStrs, "sdk.OrderStop", "sdk.SideBuy")
		case "SellStop":
			return g.emitCTradeOrder(argStrs, "sdk.OrderStop", "sdk.SideSell")
		case "PositionClose":
			if len(argStrs) >= 1 {
				return "ctx.Broker().PositionClose(int64(" + argStrs[0] + "), decimal.Zero)"
			}
		case "PositionClosePartial":
			if len(argStrs) >= 2 {
				return "ctx.Broker().PositionClose(int64(" + argStrs[0] + "), " + argStrs[1] + ")"
			}
		case "PositionModify":
			if len(argStrs) >= 3 {
				return "ctx.Broker().PositionModify(int64(" + argStrs[0] + "), " + argStrs[1] + ", " + argStrs[2] + ")"
			}
		case "OrderDelete":
			if len(argStrs) >= 1 {
				return "ctx.Broker().OrderDelete(int64(" + argStrs[0] + "))"
			}
		case "SetExpertMagicNumber":
			return "_ = " + argStrs[0] + " // SetExpertMagicNumber: no-op in preview"
		case "SetDeviationInPoints":
			return "_ = " + argStrs[0] + " // SetDeviationInPoints: no-op in preview"
		}
		return obj + "." + e.Name + "(" + strings.Join(argStrs, ", ") + ")"
	}
	return obj + "." + e.Name
}

// emitCTradeOrder emits a CTrade.Buy/Sell/etc. call as SDK OrderSend.
func (g *irGenerator) emitCTradeOrder(argStrs []string, orderType, side string) string {
	var b strings.Builder
	b.WriteString("func() bool {\n")
	b.WriteString("\t\t_, _ = ctx.Broker().OrderSend(sdk.OrderRequest{\n")
	b.WriteString("\t\t\tType: " + orderType + ",\n")
	b.WriteString("\t\t\tSide: " + side + ",\n")
	if len(argStrs) >= 1 {
		b.WriteString("\t\t\tVolume: " + argStrs[0] + ",\n")
	}
	b.WriteString("\t\t\tSymbol: ctx.Symbol(),\n")
	if len(argStrs) >= 3 {
		b.WriteString("\t\t\tPrice: " + argStrs[2] + ",\n")
	}
	if len(argStrs) >= 4 {
		b.WriteString("\t\t\tStopLoss: " + argStrs[3] + ",\n")
	}
	if len(argStrs) >= 5 {
		b.WriteString("\t\t\tTakeProfit: " + argStrs[4] + ",\n")
	}
	b.WriteString("\t\t})\n")
	b.WriteString("\t\treturn true\n")
	b.WriteString("\t}()")
	return b.String()
}
