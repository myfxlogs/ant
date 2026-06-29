package mql2go

import (
	"fmt"

	"anttrader/tools/mql2go/interp"
)

// ── IR Expression → Go source emission ─────────────────────────────

func (g *irGenerator) expr(e *interp.Expr) string {
	if e == nil {
		return ""
	}
	switch e.Kind {
	case interp.ExprLiteral:
		return g.emitLiteral(e.Val)

	case interp.ExprVar:
		return g.emitVar(e.Name)

	case interp.ExprConst:
		return g.emitConst(e.Name)

	case interp.ExprBinary:
		return g.emitBinary(e)

	case interp.ExprUnary:
		return g.emitUnary(e)

	case interp.ExprCall:
		return g.emitCall(e)

	case interp.ExprSubscript:
		return g.emitSubscript(e)

	case interp.ExprField:
		return g.emitField(e)

	case interp.ExprTernary:
		cond := g.expr(e.Cond)
		thenE := g.expr(e.ThenExpr)
		elseE := g.expr(e.ElseExpr)
		retType := "decimal.Decimal"
		if g.isIntExpr(e.ThenExpr) && g.isIntExpr(e.ElseExpr) {
			retType = "int32"
		}
		return "func() " + retType + " { if " + cond + " { return " + thenE + " }; return " + elseE + " }()"

	case interp.ExprAssignment:
		val := g.expr(&e.Args[0])
		return g.emitVar(e.Name) + " = " + val

	case interp.ExprDecl:
		val := g.expr(&e.Args[0])
		g.trackDecl(e.Name, &e.Args[0])
		return g.emitVar(e.Name) + " := " + val

	case interp.ExprCompoundAssign:
		val := g.expr(&e.Args[0])
		return g.emitVar(e.Name) + " " + e.Op + " " + val

	case interp.ExprUpdate:
		return g.emitVar(e.Name) + e.Op
	}
	return ""
}

func (g *irGenerator) emitLiteral(v interp.Value) string {
	switch v.Kind {
	case interp.ValInt:
		return fmt.Sprintf("%d", v.Int)
	case interp.ValDecimal:
		s := v.Decimal.String()
		return "decimal.RequireFromString(\"" + s + "\")"
	case interp.ValString:
		return fmt.Sprintf("%q", v.Str)
	case interp.ValBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case interp.ValDatetime:
		return fmt.Sprintf("%d", v.Datetime)
	}
	return "nil"
}

func (g *irGenerator) emitVar(name string) string {
	if g.param[name] {
		return "s." + name
	}
	switch name {
	case "Ask", "ask":
		return "ctx.Ask()"
	case "Bid", "bid":
		return "ctx.Bid()"
	case "Point", "point":
		return "ctx.Point()"
	case "_Point":
		return "ctx.Point()"
	case "Symbol", "symbol":
		return "ctx.Symbol()"
	case "_Symbol":
		return "ctx.Symbol()"
	case "Digits", "digits":
		return "ctx.Digits()"
	}
	if v := g.emitConst(name); v != name {
		return v
	}
	return name
}

func (g *irGenerator) emitConst(name string) string {
	switch name {
	case "OP_BUY":
		return "sdk.ActionBuy"
	case "OP_SELL":
		return "sdk.ActionSell"
	case "OP_BUYLIMIT":
		return "sdk.ActionBuyLimit"
	case "OP_SELLLIMIT":
		return "sdk.ActionSellLimit"
	case "OP_BUYSTOP":
		return "sdk.ActionBuyStop"
	case "OP_SELLSTOP":
		return "sdk.ActionSellStop"
	case "ORDER_TYPE_BUY":
		return "sdk.ActionBuy"
	case "ORDER_TYPE_SELL":
		return "sdk.ActionSell"
	case "ORDER_TYPE_BUY_LIMIT":
		return "sdk.ActionBuyLimit"
	case "ORDER_TYPE_SELL_LIMIT":
		return "sdk.ActionSellLimit"
	case "ORDER_TYPE_BUY_STOP":
		return "sdk.ActionBuyStop"
	case "ORDER_TYPE_SELL_STOP":
		return "sdk.ActionSellStop"
	case "true", "TRUE":
		return "true"
	case "false", "FALSE":
		return "false"
	case "EMPTY":
		return "-1"
	case "INVALID_HANDLE":
		return "-1"
	case "CLR_NONE":
		return "-1"
	}
	if v, ok := mqlConstInts[name]; ok {
		return fmt.Sprintf("%d", v)
	}
	if g.ir != nil && g.ir.Enums != nil {
		if v, ok := g.ir.Enums[name]; ok {
			return fmt.Sprintf("%d", v)
		}
	}
	return name
}

var mqlConstInts = map[string]int32{
	"SELECT_BY_POS":          0,
	"SELECT_BY_TICKET":       1,
	"MODE_TRADES":            0,
	"MODE_HISTORY":           1,
	"PRICE_CLOSE":            1,
	"PRICE_OPEN":             2,
	"PRICE_HIGH":             3,
	"PRICE_LOW":              4,
	"PRICE_MEDIAN":           5,
	"PRICE_TYPICAL":          6,
	"PRICE_WEIGHTED":         7,
	"MODE_SMA":               0,
	"MODE_EMA":               1,
	"MODE_SMMA":              2,
	"MODE_LWMA":              3,
	"PERIOD_M1":              1,
	"PERIOD_M5":              5,
	"PERIOD_M15":             15,
	"PERIOD_M30":             30,
	"PERIOD_H1":              60,
	"PERIOD_H4":              240,
	"PERIOD_D1":              1440,
	"PERIOD_W1":              10080,
	"PERIOD_MN1":             43200,
	"TRADE_ACTION_DEAL":      0,
	"TRADE_ACTION_PENDING":   1,
	"TRADE_ACTION_SLTP":      2,
	"TRADE_ACTION_PEND_CLOSE": 3,
}

func (g *irGenerator) emitBinary(e *interp.Expr) string {
	left := g.expr(&e.Args[0])
	right := g.expr(&e.Args[1])
	switch e.Op {
	case "<", ">", "<=", ">=":
		if g.isIntExpr(&e.Args[0]) && g.isIntExpr(&e.Args[1]) {
			return left + " " + e.Op + " " + right
		}
		if g.isDecimalExpr(&e.Args[0]) && g.isDecimalExpr(&e.Args[1]) {
			return g.emitDecimalCmp(left, e.Op, right)
		}
		return left + " " + e.Op + " " + right
	case "==", "!=":
		if g.isIntExpr(&e.Args[0]) && g.isIntExpr(&e.Args[1]) {
			return left + " " + e.Op + " " + right
		}
		if g.isDecimalExpr(&e.Args[0]) && g.isDecimalExpr(&e.Args[1]) {
			return g.emitDecimalCmp(left, e.Op, right)
		}
		return left + " " + e.Op + " " + right
	}
	return "(" + left + " " + e.Op + " " + right + ")"
}

func (g *irGenerator) emitDecimalCmp(left, op, right string) string {
	switch op {
	case ">":
		return left + ".GreaterThan(" + right + ")"
	case ">=":
		return left + ".GreaterThanOrEqual(" + right + ")"
	case "<":
		return left + ".LessThan(" + right + ")"
	case "<=":
		return left + ".LessThanOrEqual(" + right + ")"
	case "==":
		return left + ".Equal(" + right + ")"
	case "!=":
		return left + ".NotEqual(" + right + ")"
	}
	return left + " " + op + " " + right
}

func (g *irGenerator) emitUnary(e *interp.Expr) string {
	operand := g.expr(&e.Args[0])
	switch e.Op {
	case "-":
		return "(-" + operand + ")"
	case "!":
		return "(!" + operand + ")"
	case "~":
		return "(^" + operand + ")"
	}
	return operand
}

func (g *irGenerator) emitCall(e *interp.Expr) string {
	args := g.emitArgs(e.Args)
	return g.emitBuiltinCall(e.Name, args)
}

func (g *irGenerator) emitArgs(args []interp.Expr) []string {
	result := make([]string, len(args))
	for i := range args {
		result[i] = g.expr(&args[i])
	}
	return result
}

func (g *irGenerator) emitSubscript(e *interp.Expr) string {
	idx := g.expr(e.Index)
	switch e.Name {
	case "Close", "close":
		return "ctx.Bars().Close(" + idx + ")"
	case "Open", "open":
		return "ctx.Bars().Open(" + idx + ")"
	case "High", "high":
		return "ctx.Bars().High(" + idx + ")"
	case "Low", "low":
		return "ctx.Bars().Low(" + idx + ")"
	case "Volume", "volume":
		return "ctx.Bars().Volume(" + idx + ")"
	case "Time", "time":
		return "ctx.Bars().Time(" + idx + ")"
	}
	return e.Name + "[" + idx + "]"
}

// trackDecl records a variable as integer if its initializer is integer-typed.
func (g *irGenerator) trackDecl(name string, init *interp.Expr) {
	if g.isIntExpr(init) {
		g.intVars[name] = true
	}
}

// isIntVar returns true if the variable was declared with an integer initializer.
func (g *irGenerator) isIntVar(name string) bool {
	return g.intVars[name]
}

// isDecimalExpr infers whether an expression yields a decimal.Decimal value.
func (g *irGenerator) isDecimalExpr(e *interp.Expr) bool {
	if e == nil {
		return false
	}
	switch e.Kind {
	case interp.ExprLiteral:
		return e.Val.Kind == interp.ValDecimal
	case interp.ExprVar:
		if g.isIntVar(e.Name) {
			return false
		}
		if _, ok := mqlConstInts[e.Name]; ok {
			return false
		}
		return g.isDecimalVar(e.Name)
	case interp.ExprCall:
		switch e.Name {
		case "Ask", "Bid", "Point", "_Point",
			"OrderLots", "OrderOpenPrice", "OrderStopLoss", "OrderTakeProfit", "OrderProfit",
			"PositionGetDouble",
			"AccountBalance", "AccountEquity", "AccountFreeMargin", "AccountMargin",
			"MathAbs", "MathSqrt", "MathPow", "MathMax", "MathMin",
			"NormalizeDouble", "StringToDouble":
			return true
		}
		// Indicator calls return decimal.Decimal
		if isIndicatorName(e.Name) {
			return true
		}
		return false
	case interp.ExprSubscript:
		switch e.Name {
		case "Close", "Open", "High", "Low", "Volume":
			return true
		}
		return false
	case interp.ExprBinary:
		return g.isDecimalExpr(&e.Args[0]) || g.isDecimalExpr(&e.Args[1])
	case interp.ExprUnary:
		return g.isDecimalExpr(&e.Args[0])
	}
	return false
}

// isDecimalVar checks if a variable is known to be decimal (non-int param/global).
func (g *irGenerator) isDecimalVar(name string) bool {
	if g.isIntVar(name) {
		return false
	}
	_, isParamOrGlobal := g.param[name]
	return isParamOrGlobal
}

// isIndicatorName returns true for MQL indicator function names.
func isIndicatorName(name string) bool {
	switch name {
	case "iMA", "iRSI", "iATR", "iMACD", "iBands", "iStochastic",
		"iCCI", "iADX", "iMomentum", "iWPR", "iMFI", "iOBV",
		"iSAR", "iStdDev", "iAlligator", "iIchimoku", "iEnvelopes",
		"iDeMarker", "iOsMA", "iRVI", "iForce", "iFractals", "iGator",
		"iAC", "iAD", "iAO", "iBearsPower", "iBullsPower", "iBWMFI",
		"iAMA", "iDEMA", "iTEMA", "iFrAMA", "iVIDyA", "iTriX",
		"iADXWilder", "iChaikin", "iVolumes":
		return true
	}
	return false
}

// isIntExpr infers whether an expression yields an integer value.
func (g *irGenerator) isIntExpr(e *interp.Expr) bool {
	if e == nil {
		return false
	}
	switch e.Kind {
	case interp.ExprLiteral:
		return e.Val.Kind == interp.ValInt
	case interp.ExprVar:
		if g.isIntVar(e.Name) {
			return true
		}
		if _, ok := mqlConstInts[e.Name]; ok {
			return true
		}
		return false
	case interp.ExprConst:
		_, ok := mqlConstInts[e.Name]
		return ok
	case interp.ExprCall:
		switch e.Name {
		case "OrdersTotal", "PositionsTotal", "ArraySize",
			"OrderTicket", "OrderMagicNumber", "OrderType",
			"PositionGetTicket", "PositionGetInteger",
			"Digits", "Period":
			return true
		}
		return false
	case interp.ExprBinary:
		return g.isIntExpr(&e.Args[0]) && g.isIntExpr(&e.Args[1])
	case interp.ExprUnary:
		return g.isIntExpr(&e.Args[0])
	case interp.ExprSubscript:
		return false
	case interp.ExprField:
		return false
	}
	return false
}
