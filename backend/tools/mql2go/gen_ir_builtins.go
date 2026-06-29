package mql2go

import (
	"strings"
)

// ── MQL builtin → Go SDK call mapping ──────────────────────────────

// emitBuiltinCall maps MQL builtin functions to Go SDK calls.
func (g *irGenerator) emitBuiltinCall(name string, argStrs []string) string {
	// Indicators
	switch name {
	case "iMA":
		if len(argStrs) >= 3 {
			return g.indicatorCall("MA", argStrs, []int{0, 1})
		}
	case "iRSI":
		if len(argStrs) >= 2 {
			return g.indicatorCall("RSI", argStrs, []int{0, 1})
		}
	case "iATR":
		if len(argStrs) >= 2 {
			return g.indicatorCall("ATR", argStrs, []int{0, 1})
		}
	case "iMACD":
		if len(argStrs) >= 4 {
			return g.indicatorCall("MACD", argStrs, []int{0, 1, 2, 3})
		}
	case "iBands", "iBollinger":
		if len(argStrs) >= 3 {
			return g.indicatorCall("Bollinger", argStrs, []int{0, 1, 2})
		}
	case "iStochastic":
		if len(argStrs) >= 4 {
			return g.indicatorCall("Stochastic", argStrs, []int{0, 1, 2, 3})
		}
	case "iCCI":
		if len(argStrs) >= 2 {
			return g.indicatorCall("CCI", argStrs, []int{0, 1})
		}
	case "iADX":
		if len(argStrs) >= 2 {
			return g.indicatorCall("ADX", argStrs, []int{0, 1})
		}
	case "iMomentum":
		if len(argStrs) >= 2 {
			return g.indicatorCall("Momentum", argStrs, []int{0, 1})
		}
	case "iWPR":
		if len(argStrs) >= 2 {
			return g.indicatorCall("WPR", argStrs, []int{0, 1})
		}
	case "iMFI":
		if len(argStrs) >= 2 {
			return g.indicatorCall("MFI", argStrs, []int{0, 1})
		}
	case "iOBV":
		if len(argStrs) >= 1 {
			return g.indicatorCall("OBV", argStrs, []int{0})
		}
	case "iSAR":
		if len(argStrs) >= 3 {
			return g.indicatorCall("SAR", argStrs, []int{0, 1, 2})
		}
	case "iStdDev":
		if len(argStrs) >= 2 {
			return g.indicatorCall("StdDev", argStrs, []int{0, 1})
		}
	case "iAlligator":
		return g.indicatorCall("Alligator", argStrs, nil)
	case "iIchimoku":
		return g.indicatorCall("Ichimoku", argStrs, nil)
	case "iEnvelopes":
		return g.indicatorCall("Envelopes", argStrs, nil)
	case "iDeMarker":
		return g.indicatorCall("DeMarker", argStrs, []int{0, 1})
	case "iOsMA":
		return g.indicatorCall("OsMA", argStrs, nil)
	case "iRVI":
		return g.indicatorCall("RVI", argStrs, []int{0, 1})
	case "iForce":
		return g.indicatorCall("Force", argStrs, nil)
	case "iFractals":
		return g.indicatorCall("Fractals", argStrs, nil)
	case "iGator":
		return g.indicatorCall("Gator", argStrs, nil)
	case "iAC":
		return g.indicatorCall("AC", argStrs, nil)
	case "iAD":
		return g.indicatorCall("AD", argStrs, nil)
	case "iAO":
		return g.indicatorCall("AO", argStrs, nil)
	case "iBearsPower":
		return g.indicatorCall("BearsPower", argStrs, nil)
	case "iBullsPower":
		return g.indicatorCall("BullsPower", argStrs, nil)
	case "iBWMFI":
		return g.indicatorCall("BWMFI", argStrs, nil)
	case "iAMA":
		return g.indicatorCall("AMA", argStrs, nil)
	case "iDEMA":
		return g.indicatorCall("DEMA", argStrs, []int{0, 1})
	case "iTEMA":
		return g.indicatorCall("TEMA", argStrs, []int{0, 1})
	case "iFrAMA":
		return g.indicatorCall("FrAMA", argStrs, []int{0, 1})
	case "iVIDyA":
		return g.indicatorCall("VIDyA", argStrs, nil)
	case "iTriX":
		return g.indicatorCall("TriX", argStrs, []int{0, 1})
	case "iADXWilder":
		return g.indicatorCall("ADXWilder", argStrs, []int{0, 1})
	case "iChaikin":
		return g.indicatorCall("Chaikin", argStrs, nil)
	case "iVolumes":
		return g.indicatorCall("Volumes", argStrs, nil)
	// Math functions
	case "MathAbs":
		if len(argStrs) >= 1 {
			return argStrs[0] + ".Abs()"
		}
	case "MathMax":
		if len(argStrs) >= 2 {
			return "decimal.Max(" + argStrs[0] + ", " + argStrs[1] + ")"
		}
	case "MathMin":
		if len(argStrs) >= 2 {
			return "decimal.Min(" + argStrs[0] + ", " + argStrs[1] + ")"
		}
	case "MathSqrt":
		if len(argStrs) >= 1 {
			g.stdImports["math"] = true
			return "decimal.NewFromFloat(math.Sqrt(" + argStrs[0] + ".InexactFloat64()))"
		}
	case "MathPow":
		if len(argStrs) >= 2 {
			g.stdImports["math"] = true
			return "decimal.NewFromFloat(math.Pow(" + argStrs[0] + ".InexactFloat64(), " + argStrs[1] + ".InexactFloat64()))"
		}
	// Platform functions
	case "Print", "Comment", "Alert":
		return "ctx.Log(" + strings.Join(argStrs, " + ") + ")"
	case "NormalizeDouble":
		if len(argStrs) >= 2 {
			return argStrs[0] + ".Round(" + argStrs[1] + ")"
		}
	case "DoubleToString":
		if len(argStrs) >= 1 {
			return argStrs[0] + ".String()"
		}
	case "IntegerToString":
		if len(argStrs) >= 1 {
			g.stdImports["fmt"] = true
			return "fmt.Sprintf(\"%d\", " + argStrs[0] + ")"
		}
	case "StringToDouble":
		if len(argStrs) >= 1 {
			return "decimal.RequireFromString(" + argStrs[0] + ")"
		}
	case "StringToInteger":
		if len(argStrs) >= 1 {
			return "int32(" + argStrs[0] + ")"
		}
	case "StringLen":
		if len(argStrs) >= 1 {
			return "len(" + argStrs[0] + ")"
		}
	case "StringFind":
		if len(argStrs) >= 2 {
			g.stdImports["strings"] = true
			return "strings.Index(" + argStrs[0] + ", " + argStrs[1] + ")"
		}
	case "StringSubstr":
		if len(argStrs) >= 2 {
			return argStrs[0] + "[" + argStrs[1] + ":]"
		}
	case "StringReplace":
		if len(argStrs) >= 3 {
			g.stdImports["strings"] = true
			return "strings.ReplaceAll(" + argStrs[0] + ", " + argStrs[1] + ", " + argStrs[2] + ")"
		}
	case "StringConcatenate":
		g.stdImports["strings"] = true
		return "strings.Join([]string{" + strings.Join(argStrs, ", ") + "}, \"\")"
	case "StringTrimLeft":
		if len(argStrs) >= 1 {
			g.stdImports["strings"] = true
			return "strings.TrimLeft(" + argStrs[0] + ", \" \\t\\n\\r\")"
		}
	case "StringTrimRight":
		if len(argStrs) >= 1 {
			g.stdImports["strings"] = true
			return "strings.TrimRight(" + argStrs[0] + ", \" \\t\\n\\r\")"
		}
	case "StringSplit":
		if len(argStrs) >= 2 {
			g.stdImports["strings"] = true
			return "strings.Split(" + argStrs[1] + ", " + argStrs[0] + ")"
		}
	case "TimeToString":
		if len(argStrs) >= 1 {
			g.stdImports["time"] = true
			return "time.Unix(int64(" + argStrs[0] + "), 0).Format(\"2006.01.02 15:04\")"
		}
	case "TimeCurrent":
		return "ctx.ServerTime()"
	case "Period":
		return "int32(0) // timeframe period"
	case "Digits":
		return "ctx.Digits()"
	case "Symbol":
		return "ctx.Symbol()"
	case "Ask":
		return "ctx.Ask()"
	case "Bid":
		return "ctx.Bid()"
	case "Point":
		return "ctx.Point()"
	// Account functions
	case "AccountBalance":
		return "ctx.Account().Balance"
	case "AccountEquity":
		return "ctx.Account().Equity"
	case "AccountFreeMargin":
		return "ctx.Account().FreeMargin"
	case "AccountMargin":
		return "ctx.Account().Margin"
	case "AccountLeverage":
		return "ctx.Account().Leverage"
	// Trade functions (MQL4)
	case "OrderSend":
		return g.emitOrderSend(argStrs)
	case "OrderClose":
		if g.posLoopVar != "" {
			return "ctx.Broker().PositionClose(" + g.posLoopVar + ".Ticket, " + g.posLoopVar + ".Volume)"
		}
		if len(argStrs) >= 3 {
			return "ctx.Broker().PositionClose(int64(" + argStrs[0] + "), " + argStrs[1] + ")"
		}
	case "OrderModify":
		if len(argStrs) >= 4 {
			return "ctx.Broker().PositionModify(int64(" + argStrs[0] + "), " + argStrs[2] + ", " + argStrs[3] + ")"
		}
	case "OrderDelete":
		if len(argStrs) >= 1 {
			return "ctx.Broker().OrderDelete(int64(" + argStrs[0] + "))"
		}
	case "OrdersTotal":
		return "int32(len(ctx.Broker().Positions(0)))"
	case "OrderSelect":
		return "true"
	case "OrderTicket":
		if g.posLoopVar != "" {
			return g.posLoopVar + ".Ticket"
		}
		return "int32(0)"
	case "OrderSymbol":
		if g.posLoopVar != "" {
			return g.posLoopVar + ".Symbol"
		}
		return "\"\""
	case "OrderType":
		if g.posLoopVar != "" {
			return "int32(" + g.posLoopVar + ".Side)"
		}
		return "int32(0)"
	case "OrderLots":
		if g.posLoopVar != "" {
			return g.posLoopVar + ".Volume"
		}
		return "decimal.Zero"
	case "OrderOpenPrice":
		if g.posLoopVar != "" {
			return g.posLoopVar + ".OpenPrice"
		}
		return "decimal.Zero"
	case "OrderStopLoss":
		if g.posLoopVar != "" {
			return g.posLoopVar + ".StopLoss"
		}
		return "decimal.Zero"
	case "OrderTakeProfit":
		if g.posLoopVar != "" {
			return g.posLoopVar + ".TakeProfit"
		}
		return "decimal.Zero"
	case "OrderProfit":
		if g.posLoopVar != "" {
			return g.posLoopVar + ".Profit"
		}
		return "decimal.Zero"
	case "OrderMagicNumber":
		if g.posLoopVar != "" {
			return g.posLoopVar + ".Magic"
		}
		return "int32(0)"
	// MQL5 position functions
	case "PositionsTotal":
		return "int32(len(ctx.Broker().Positions(0)))"
	case "PositionGetTicket":
		if g.posLoopVar != "" {
			return g.posLoopVar + ".Ticket"
		}
		if len(argStrs) >= 1 {
			return "int32(0)"
		}
	case "PositionGetDouble":
		if g.posLoopVar != "" {
			if len(argStrs) >= 1 {
				switch argStrs[0] {
				case "POSITION_VOLUME":
					return g.posLoopVar + ".Volume"
				case "POSITION_PRICE_OPEN":
					return g.posLoopVar + ".OpenPrice"
				case "POSITION_SL":
					return g.posLoopVar + ".StopLoss"
				case "POSITION_TP":
					return g.posLoopVar + ".TakeProfit"
				case "POSITION_PRICE_CURRENT":
					return g.posLoopVar + ".OpenPrice"
				case "POSITION_SWAP":
					return g.posLoopVar + ".Swap"
				case "POSITION_COMMISSION":
					return g.posLoopVar + ".Commission"
				}
			}
			return g.posLoopVar + ".OpenPrice"
		}
		return "decimal.Zero"
	case "PositionGetInteger":
		if g.posLoopVar != "" {
			if len(argStrs) >= 1 {
				switch argStrs[0] {
				case "POSITION_TICKET":
					return g.posLoopVar + ".Ticket"
				case "POSITION_MAGIC":
					return g.posLoopVar + ".Magic"
				case "POSITION_TYPE":
					return "int32(" + g.posLoopVar + ".Side)"
				}
			}
			return g.posLoopVar + ".Magic"
		}
		return "int32(0)"
	case "PositionGetString":
		if g.posLoopVar != "" {
			return g.posLoopVar + ".Symbol"
		}
		return "\"\""
	case "PositionSelectByTicket":
		return "true"
	case "PositionGetSymbol":
		if g.posLoopVar != "" {
			return g.posLoopVar + ".Symbol"
		}
		return "\"\""
	// Array functions
	case "ArraySize":
		if len(argStrs) >= 1 {
			return "int32(len(" + argStrs[0] + "))"
		}
	case "ArrayResize":
		if len(argStrs) >= 2 {
			return "int32(0) // ArrayResize"
		}
	case "ArrayCopy":
		return "int32(0) // ArrayCopy"
	case "ArrayMaximum":
		if len(argStrs) >= 1 {
			return "int32(0) // ArrayMaximum"
		}
	case "ArrayMinimum":
		if len(argStrs) >= 1 {
			return "int32(0) // ArrayMinimum"
		}
	case "ArraySort":
		return "int32(0) // ArraySort"
	case "ArrayInitialize":
		return "int32(0) // ArrayInitialize"
	case "ArrayRange":
		return "int32(0) // ArrayRange"
	// EventSetTimer
	case "EventSetTimer":
		if len(argStrs) >= 1 {
			return "ctx.SetTimer(int(" + argStrs[0] + "))"
		}
	case "EventSetTimerMillisecond":
		if len(argStrs) >= 1 {
			return "ctx.SetTimer(int(" + argStrs[0] + ") / 1000)"
		}
	case "EventKillTimer":
		return "ctx.KillTimer()"
	}
	// User-defined function
	if g.ir != nil && g.ir.Funcs != nil {
		if _, ok := g.ir.Funcs[name]; ok {
			return "s." + name + "(" + strings.Join(argStrs, ", ") + ")"
		}
	}
	// Unknown — emit as-is with comment
	return "nil /* unimplemented: " + name + " */"
}

// indicatorCall emits an SDK indicator call with int-cast wrapping.
func (g *irGenerator) indicatorCall(method string, argStrs []string, intArgs []int) string {
	for _, i := range intArgs {
		if i < len(argStrs) {
			argStrs[i] = "int(" + argStrs[i] + ")"
		}
	}
	return "ctx.Indicators()." + method + "(" + strings.Join(argStrs, ", ") + ")"
}

// emitOrderSend translates MQL4 OrderSend to SDK broker call.
func (g *irGenerator) emitOrderSend(argStrs []string) string {
	if len(argStrs) < 4 {
		return "int32(-1) // OrderSend: not enough args"
	}
	symbol := argStrs[0]
	cmd := argStrs[1]
	volume := argStrs[2]
	price := argStrs[3]

	// Map MQL4 order type constant (resolved by emitConst) to SDK OrderType + Side.
	var orderType, side string
	switch cmd {
	case "sdk.ActionBuy":
		orderType, side = "sdk.OrderMarket", "sdk.SideBuy"
	case "sdk.ActionSell":
		orderType, side = "sdk.OrderMarket", "sdk.SideSell"
	case "sdk.ActionBuyLimit":
		orderType, side = "sdk.OrderLimit", "sdk.SideBuy"
	case "sdk.ActionSellLimit":
		orderType, side = "sdk.OrderLimit", "sdk.SideSell"
	case "sdk.ActionBuyStop":
		orderType, side = "sdk.OrderStop", "sdk.SideBuy"
	case "sdk.ActionSellStop":
		orderType, side = "sdk.OrderStop", "sdk.SideSell"
	default:
		orderType, side = cmd, "sdk.SideBuy"
	}

	var sl, tp, magic, comment, deviation string
	if len(argStrs) >= 6 {
		sl = argStrs[5]
	}
	if len(argStrs) >= 7 {
		tp = argStrs[6]
	}
	if len(argStrs) >= 8 {
		comment = argStrs[7]
	}
	if len(argStrs) >= 9 {
		magic = argStrs[8]
	}
	if len(argStrs) >= 5 {
		deviation = argStrs[4]
	}

	parts := []string{
		"Symbol: " + symbol,
		"Type: " + orderType,
		"Side: " + side,
		"Volume: " + volume,
		"Price: " + price,
	}
	if sl != "" {
		parts = append(parts, "StopLoss: "+sl)
	}
	if tp != "" {
		parts = append(parts, "TakeProfit: "+tp)
	}
	if deviation != "" {
		parts = append(parts, "Deviation: "+deviation)
	}
	if magic != "" {
		parts = append(parts, "Magic: "+magic)
	}
	if comment != "" {
		parts = append(parts, "Comment: "+comment)
	}

	var b strings.Builder
	b.WriteString("func() int32 {\n")
	b.WriteString("\t\tresult, _ := ctx.Broker().OrderSend(sdk.OrderRequest{\n")
	for _, p := range parts {
		b.WriteString("\t\t\t" + p + ",\n")
	}
	b.WriteString("\t\t})\n")
	b.WriteString("\t\t_ = result\n")
	b.WriteString("\t\treturn 0\n")
	b.WriteString("\t}()")
	return b.String()
}
