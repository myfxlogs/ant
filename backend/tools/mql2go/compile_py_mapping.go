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

func mapPythonMethod(method, fullPath string) string {
	// ── ctx.broker.* → CTrade method builtins ──
	if strings.Contains(fullPath, "broker.") {
		switch method {
		case "buy":
			return "CTrade.Buy"
		case "sell":
			return "CTrade.Sell"
		case "buy_limit":
			return "CTrade.BuyLimit"
		case "sell_limit":
			return "CTrade.SellLimit"
		case "buy_stop":
			return "CTrade.BuyStop"
		case "sell_stop":
			return "CTrade.SellStop"
		case "close":
			return "CTrade.PositionClose"
		case "close_partial":
			return "CTrade.PositionClosePartial"
		case "close_by":
			return "CTrade.PositionCloseBy"
		case "modify":
			return "CTrade.PositionModify"
		case "delete":
			return "CTrade.OrderDelete"
		case "order_send":
			return "OrderSend"
		case "order_close":
			return "OrderClose"
		case "order_modify":
			return "OrderModify"
		case "order_delete":
			return "OrderDelete"
		}
	}
	// ── ctx.positions.* → position builtins ──
	if strings.Contains(fullPath, "positions.") {
		switch method {
		case "count":
			return "PositionsTotal"
		case "total":
			return "PositionsTotal"
		}
	}
	// ── Direct ctx.* method calls ──
	if strings.HasPrefix(fullPath, "ctx.") && strings.Count(fullPath, ".") == 1 {
		switch method {
		case "ask":
			return "Ask"
		case "bid":
			return "Bid"
		case "symbol":
			return "Symbol"
		case "point":
			return "Point"
		case "digits":
			return "Digits"
		case "spread":
			return "Spread"
		case "period":
			return "Period"
		}
	}
	// ── Legacy snake_case trade function names ──
	switch method {
	case "order_send":
		return "OrderSend"
	case "order_close":
		return "OrderClose"
	case "order_modify":
		return "OrderModify"
	case "order_delete":
		return "OrderDelete"
	case "position_close":
		return "PositionClose"
	case "position_modify":
		return "PositionModify"
	case "positions_total":
		return "PositionsTotal"
	case "orders_total":
		return "OrdersTotal"
	}
	if strings.Contains(fullPath, "indicators.") {
		switch method {
		case "ima":
			return "iMA"
		case "irsi":
			return "iRSI"
		case "iatr":
			return "iATR"
		case "ibands":
			return "iBands"
		case "imacd":
			return "iMACD"
		case "istochastic":
			return "iStochastic"
		case "icci":
			return "iCCI"
		case "iadx":
			return "iADX"
		case "imomentum":
			return "iMomentum"
		case "iwpr":
			return "iWPR"
		case "imfi":
			return "iMFI"
		case "iobv":
			return "iOBV"
		case "isar":
			return "iSAR"
		case "istddev":
			return "iStdDev"
		}
		return lookupBuiltinCaseInsensitive(method)
	}
	if strings.Contains(fullPath, "bars.") || strings.Contains(fullPath, "bars()") {
		switch method {
		case "close":
			return "Close"
		case "open":
			return "Open"
		case "high":
			return "High"
		case "low":
			return "Low"
		case "volume":
			return "Volume"
		case "time":
			return "Time"
		}
	}
	// Higher timeframe bar access: ctx.bars_tf("H4").close(0)
	// Maps to iClose("", 240, shift) — symbol="" means primary symbol, 240=H4.
	// The compiler injects symbol="" + converts the TF string to MQL period int.
	if strings.Contains(fullPath, "bars_tf.") {
		switch method {
		case "close":
			return "iClose"
		case "open":
			return "iOpen"
		case "high":
			return "iHigh"
		case "low":
			return "iLow"
		case "volume":
			return "iVolume"
		case "time":
			return "iTime"
		}
	}
	// Multi-symbol bar access: ctx.bars_for_symbol("EURUSD").close(0)
	// Maps to iClose(symbol, 0, shift) — timeframe=0 means PERIOD_CURRENT.
	// The compiler prepends the inner call's symbol arg + injects timeframe=0.
	if strings.Contains(fullPath, "bars_for_symbol.") {
		switch method {
		case "close":
			return "iClose"
		case "open":
			return "iOpen"
		case "high":
			return "iHigh"
		case "low":
			return "iLow"
		case "volume":
			return "iVolume"
		case "time":
			return "iTime"
		}
	}
	if strings.Contains(fullPath, "account.") {
		switch method {
		case "balance":
			return "AccountBalance"
		case "equity":
			return "AccountEquity"
		case "margin":
			return "AccountMargin"
		case "free_margin":
			return "AccountFreeMargin"
		case "profit":
			return "AccountProfit"
		case "leverage":
			return "AccountLeverage"
		}
	}
	if strings.Contains(fullPath, "symbol_info.") || strings.Contains(fullPath, "market.") ||
		strings.Contains(fullPath, "ctx.symbol_info.") || strings.Contains(fullPath, "ctx.market.") {
		switch method {
		case "bid":
			return "Bid"
		case "ask":
			return "Ask"
		case "point":
			return "Point"
		case "digits":
			return "Digits"
		case "spread":
			return "Spread"
		case "symbol":
			return "Symbol"
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
			return []string{"volume", "symbol", "price", "sl", "tp", "comment"}
		case strings.HasSuffix(fullPath, "buy_limit") || strings.HasSuffix(fullPath, "sell_limit") ||
			strings.HasSuffix(fullPath, "buy_stop") || strings.HasSuffix(fullPath, "sell_stop"):
			// CTrade.BuyLimit/SellLimit/BuyStop/SellStop(volume, price, sl, tp, comment)
			return []string{"volume", "price", "sl", "tp", "comment"}
		case strings.HasSuffix(fullPath, "modify"):
			// CTrade.PositionModify(ticket, sl, tp)
			return []string{"ticket", "sl", "tp"}
		case strings.HasSuffix(fullPath, "close"):
			// CTrade.PositionClose(ticket)
			return []string{"ticket"}
		case strings.HasSuffix(fullPath, "close_partial"):
			// CTrade.PositionClosePartial(ticket, volume)
			return []string{"ticket", "volume"}
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
	"lot": "volume",
}

// resolveParamName converts a keyword argument name to its canonical name.
func resolveParamName(name string) string {
	if alias, ok := pythonParamAliases[name]; ok {
		return alias
	}
	return name
}
