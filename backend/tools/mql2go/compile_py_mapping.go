package mql2go

import (
	"strings"
)

// lookupBuiltinCaseInsensitive does a case-insensitive lookup against builtinRegistry.
// This handles indicator names like "ialligator" → "iAlligator", "iosma" → "iOsMA".
func lookupBuiltinCaseInsensitive(name string) string {
	lower := strings.ToLower(name)
	for _, entry := range builtinRegistry {
		if strings.ToLower(entry.name) == lower {
			return entry.name
		}
	}
	return ""
}

func unquotePython(s string) string {
	s = strings.TrimSpace(s)
	for len(s) > 0 && (s[0] == 'f' || s[0] == 'F' || s[0] == 'r' || s[0] == 'R' || s[0] == 'b' || s[0] == 'B') {
		s = s[1:]
	}
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			inner := s[1 : len(s)-1]
			if len(inner) >= 6 && (strings.HasPrefix(inner, `"""`) || strings.HasPrefix(inner, `'''`)) {
				return inner[3 : len(inner)-3]
			}
			inner = strings.ReplaceAll(inner, "\\n", "\n")
			inner = strings.ReplaceAll(inner, "\\t", "\t")
			inner = strings.ReplaceAll(inner, `\"`, `"`)
			inner = strings.ReplaceAll(inner, `\'`, `'`)
			inner = strings.ReplaceAll(inner, `\\`, `\`)
			return inner
		}
	}
	return s
}

// pyMethodMap maps "domain.method" → MQL builtin name.
var pyMethodMap = map[string]string{
	// broker.* → CTrade/Order builtins
	"broker.buy": "CTrade.Buy", "broker.sell": "CTrade.Sell",
	"broker.buy_limit": "CTrade.BuyLimit", "broker.sell_limit": "CTrade.SellLimit",
	"broker.buy_stop": "CTrade.BuyStop", "broker.sell_stop": "CTrade.SellStop",
	"broker.close": "CTrade.PositionClose", "broker.close_partial": "CTrade.PositionClosePartial",
	"broker.close_by": "CTrade.PositionCloseBy", "broker.modify": "CTrade.PositionModify",
	"broker.delete":     "CTrade.OrderDelete",
	"broker.close_all":  "CloseAll",
	"broker.order_send": "OrderSend", "broker.order_close": "OrderClose",
	"broker.order_modify": "OrderModify", "broker.order_delete": "OrderDelete",
	// positions.*
	"positions.count": "PositionsTotal", "positions.total": "PositionsTotal",
	"ctx.positions": "PositionsTotal",
	// ctx.* direct
	"ctx.ask": "Ask", "ctx.bid": "Bid", "ctx.symbol": "Symbol",
	"ctx.point": "Point", "ctx.digits": "Digits", "ctx.spread": "Spread", "ctx.period": "Period",
	// account.*
	"account.balance": "AccountBalance", "account.equity": "AccountEquity",
	"account.margin": "AccountMargin", "account.free_margin": "AccountFreeMargin",
	"account.profit": "AccountProfit", "account.leverage": "AccountLeverage",
	// symbol_info.* / market.*
	"symbol_info.bid": "Bid", "symbol_info.ask": "Ask",
	"symbol_info.point": "Point", "symbol_info.digits": "Digits",
	"symbol_info.spread": "Spread", "symbol_info.symbol": "Symbol",
	"market.bid": "Bid", "market.ask": "Ask",
	"market.point": "Point", "market.digits": "Digits",
	"market.spread": "Spread", "market.symbol": "Symbol",
	"ctx.symbol_info.bid": "Bid", "ctx.symbol_info.ask": "Ask",
	"ctx.symbol_info.point": "Point", "ctx.symbol_info.digits": "Digits",
	"ctx.symbol_info.spread": "Spread", "ctx.symbol_info.symbol": "Symbol",
	"ctx.market.bid": "Bid", "ctx.market.ask": "Ask",
	"ctx.market.point": "Point", "ctx.market.digits": "Digits",
	"ctx.market.spread": "Spread", "ctx.market.symbol": "Symbol",
}

// pyBarsMethods maps bar-access method names → builtin names (shared by bars/bars_tf/bars_for_symbol).
var pyBarsMethods = map[string]string{
	"close": "Close", "open": "Open", "high": "High",
	"low": "Low", "volume": "Volume", "time": "Time",
}

// pyBarsTfMethods maps bar-access method names → cross-timeframe builtins.
var pyBarsTfMethods = map[string]string{
	"close": "iClose", "open": "iOpen", "high": "iHigh",
	"low": "iLow", "volume": "iVolume", "time": "iTime",
}

// pyIndicatorMethods maps indicator method names → MQL indicator builtins.
var pyIndicatorMethods = map[string]string{
	"ima": "iMA", "irsi": "iRSI", "iatr": "iATR", "ibands": "iBands",
	"imacd": "iMACD", "istochastic": "iStochastic", "icci": "iCCI",
	"iadx": "iADX", "imomentum": "iMomentum", "iwpr": "iWPR",
	"imfi": "iMFI", "iobv": "iOBV", "isar": "iSAR", "istddev": "iStdDev",
}

// pyLegacyTradeMethods maps legacy snake_case trade function names.
var pyLegacyTradeMethods = map[string]string{
	"order_send": "OrderSend", "order_close": "OrderClose",
	"order_modify": "OrderModify", "order_delete": "OrderDelete",
	"position_close": "PositionClose", "position_modify": "PositionModify",
	"positions_total": "PositionsTotal", "orders_total": "OrdersTotal",
}

func mapPythonMethod(method, fullPath string) string {
	// Check domain-specific maps based on path prefix
	if v, ok := pyMethodMap[fullPath+"."+method]; ok {
		return v
	}
	// broker.* also matches "ctx.broker."
	if strings.Contains(fullPath, "broker.") {
		if v, ok := pyMethodMap["broker."+method]; ok {
			return v
		}
	}
	// positions.*
	if strings.Contains(fullPath, "positions.") {
		if v, ok := pyMethodMap["positions."+method]; ok {
			return v
		}
	}
	// ctx.* direct method calls
	if strings.HasPrefix(fullPath, "ctx.") && strings.Count(fullPath, ".") == 1 {
		if v, ok := pyMethodMap["ctx."+method]; ok {
			return v
		}
	}
	// Legacy snake_case trade functions
	if v, ok := pyLegacyTradeMethods[method]; ok {
		return v
	}
	// indicators.*
	if strings.Contains(fullPath, "indicators.") {
		if v, ok := pyIndicatorMethods[method]; ok {
			return v
		}
		return lookupBuiltinCaseInsensitive(method)
	}
	// bars.* / bars().*
	if strings.Contains(fullPath, "bars.") || strings.Contains(fullPath, "bars()") {
		if v, ok := pyBarsMethods[method]; ok {
			return v
		}
	}
	// bars_tf.* → cross-timeframe
	if strings.Contains(fullPath, "bars_tf.") {
		if v, ok := pyBarsTfMethods[method]; ok {
			return v
		}
	}
	// bars_for_symbol.* → multi-symbol (same builtins as bars_tf)
	if strings.Contains(fullPath, "bars_for_symbol.") {
		if v, ok := pyBarsTfMethods[method]; ok {
			return v
		}
	}
	// account.*
	if strings.Contains(fullPath, "account.") {
		if v, ok := pyMethodMap["account."+method]; ok {
			return v
		}
	}
	// symbol_info.* / market.* / ctx.symbol_info.* / ctx.market.*
	if strings.Contains(fullPath, "symbol_info.") || strings.Contains(fullPath, "market.") ||
		strings.Contains(fullPath, "ctx.symbol_info.") || strings.Contains(fullPath, "ctx.market.") {
		domain := extractDomain(fullPath)
		if v, ok := pyMethodMap[domain+"."+method]; ok {
			return v
		}
	}
	return ""
}

func extractDomain(fullPath string) string {
	for _, prefix := range []string{"ctx.symbol_info.", "ctx.market.", "symbol_info.", "market."} {
		if strings.HasPrefix(fullPath, prefix) {
			return strings.TrimSuffix(prefix, ".")
		}
	}
	return ""
}

// tfStringToInt converts a timeframe string (e.g. "H4", "M15") to its MQL period int.
// Returns 0 (PERIOD_CURRENT) for unknown strings.
func tfStringToInt(tf string) int32 {
	switch tf {
	case "M1":
		return 1
	case "M2":
		return 2
	case "M3":
		return 3
	case "M4":
		return 4
	case "M5":
		return 5
	case "M6":
		return 6
	case "M10":
		return 10
	case "M12":
		return 12
	case "M15":
		return 15
	case "M20":
		return 20
	case "M30":
		return 30
	case "H1":
		return 60
	case "H2":
		return 120
	case "H3":
		return 180
	case "H4":
		return 240
	case "H6":
		return 360
	case "H8":
		return 480
	case "H12":
		return 720
	case "D1":
		return 1440
	case "W1":
		return 10080
	case "MN1":
		return 43200
	default:
		return 0
	}
}

// pythonMethodParamOrder returns the canonical parameter names for a Python
// SDK method, in the positional order expected by the VM builtin. This is used
// to reorder keyword arguments (e.g. ctx.broker.buy(sl=..., lot=...)) to the
// correct positional slots. Returns nil for methods that don't accept keyword args.
func pythonMethodParamOrder(fullPath string) []string {
	if strings.Contains(fullPath, "broker.") {
		switch {
		case strings.HasSuffix(fullPath, "buy") || strings.HasSuffix(fullPath, "sell"):
			// CTrade.Buy/Sell(volume, symbol, price, sl, tp, comment)
			return []string{nodeVolume, "symbol", "price", "sl", "tp", "comment"}
		case strings.HasSuffix(fullPath, "buy_limit") || strings.HasSuffix(fullPath, "sell_limit") ||
			strings.HasSuffix(fullPath, "buy_stop") || strings.HasSuffix(fullPath, "sell_stop"):
			// CTrade.BuyLimit/SellLimit/BuyStop/SellStop(volume, price, sl, tp, comment)
			return []string{nodeVolume, "price", "sl", "tp", "comment"}
		case strings.HasSuffix(fullPath, "modify"):
			// CTrade.PositionModify(ticket, sl, tp)
			return []string{"ticket", "sl", "tp"}
		case strings.HasSuffix(fullPath, "close"):
			// CTrade.PositionClose(ticket)
			return []string{"ticket"}
		case strings.HasSuffix(fullPath, "close_partial"):
			// CTrade.PositionClosePartial(ticket, volume)
			return []string{"ticket", nodeVolume}
		case strings.HasSuffix(fullPath, "close_by"):
			// CTrade.PositionCloseBy(ticket, by_ticket)
			return []string{"ticket", "by_ticket"}
		case strings.HasSuffix(fullPath, "delete"):
			// CTrade.OrderDelete(ticket)
			return []string{"ticket"}
		}
	}
	return nil
}

// pythonParamAliases maps alternative keyword names to canonical parameter names.
var pythonParamAliases = map[string]string{
	"lot": nodeVolume,
}

// resolveParamName converts a keyword argument name to its canonical name.
func resolveParamName(name string) string {
	if alias, ok := pythonParamAliases[name]; ok {
		return alias
	}
	return name
}
