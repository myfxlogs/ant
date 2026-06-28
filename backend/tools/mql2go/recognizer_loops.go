package mql2go

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// ── Order loop recognition (MQL4: OrdersTotal + OrderSelect) ────────

// orderPropertyMap maps MQL4 Order* property functions to SDK Position fields.
var orderPropertyMap = map[string]string{
	"OrderTicket":      "Ticket",
	"OrderSymbol":      "Symbol",
	"OrderLots":        "Volume",
	"OrderOpenPrice":   "OpenPrice",
	"OrderStopLoss":    "StopLoss",
	"OrderTakeProfit":  "TakeProfit",
	"OrderProfit":      "Profit",
	"OrderSwap":        "Swap",
	"OrderCommission":  "Commission",
	"OrderComment":     "Comment",
	"OrderMagicNumber": "Magic",
	"OrderOpenTime":    "OpenTime",
	"OrderType":        "Type",
	"OrderClosePrice":  "ClosePrice",
	"OrderCloseTime":   "CloseTime",
	"OrderExpiration":  "Expiration",
}

func extractOrderLoopsCST(root *sitter.Node, version string) []OrderLoopRule {
	if version != "mql4" {
		return nil
	}
	var loops []OrderLoopRule
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "for_statement" {
			return true
		}
		// Check if the for-loop body contains OrderSelect
		body := childByType("", n, "compound_statement")
		if body == nil {
			return true
		}
		hasOrderSelect := false
		walkCST(body, func(inner *sitter.Node) bool {
			if inner.Type() == "call_expression" && callFuncName(inner) == "OrderSelect" {
				hasOrderSelect = true
				return false
			}
			return true
		})
		if !hasOrderSelect {
			return true
		}
		loop := OrderLoopRule{}
		// Detect property calls and actions in the loop body
		walkCST(body, func(inner *sitter.Node) bool {
			if inner.Type() != "call_expression" {
				return true
			}
			name := callFuncName(inner)
			if _, ok := orderPropertyMap[name]; ok {
				loop.PropertyCalls = appendUnique(loop.PropertyCalls, name)
			}
			switch name {
			case "OrderClose":
				loop.BodyActions = appendUnique(loop.BodyActions, "position_close")
			case "OrderDelete":
				loop.BodyActions = appendUnique(loop.BodyActions, "order_delete")
			case "OrderModify":
				// Check if this is a trailing stop pattern
				if isTrailingStopPattern(body, inner) {
					loop.BodyActions = appendUnique(loop.BodyActions, "order_modify:trailing_stop")
				} else {
					loop.BodyActions = appendUnique(loop.BodyActions, "order_modify")
				}
			case "OrderCloseBy":
				loop.BodyActions = appendUnique(loop.BodyActions, "position_close_by")
			}
			if name == "OrderMagicNumber" {
				loop.HasMagicFilter = true
			}
			if name == "OrderSymbol" {
				loop.HasSymbolFilter = true
			}
			return true
		})
		loops = append(loops, loop)
		return false
	})
	return loops
}

// positionPropertyMap maps MQL5 PositionGet* functions to SDK Position fields.
var positionPropertyMap = map[string]string{
	"PositionGetDouble":  "PositionGetDouble",
	"PositionGetInteger": "PositionGetInteger",
	"PositionGetString":  "PositionGetString",
	"PositionGetSymbol":  "PositionGetSymbol",
	"PositionGetTicket":  "PositionGetTicket",
}

func extractPositionLoopsCST(root *sitter.Node, version string) []PositionLoopRule {
	if version != "mql5" {
		return nil
	}
	var loops []PositionLoopRule
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "for_statement" {
			return true
		}
		body := childByType("", n, "compound_statement")
		if body == nil {
			return true
		}
		// Check if the for-loop body contains PositionGetTicket or PositionSelect
		hasPositionIter := false
		walkCST(body, func(inner *sitter.Node) bool {
			if inner.Type() == "call_expression" {
				name := callFuncName(inner)
				if name == "PositionGetTicket" || name == "PositionSelect" || name == "PositionSelectByTicket" {
					hasPositionIter = true
					return false
				}
			}
			return true
		})
		if !hasPositionIter {
			return true
		}
		loop := PositionLoopRule{}
		walkCST(body, func(inner *sitter.Node) bool {
			if inner.Type() != "call_expression" {
				return true
			}
			name := callFuncName(inner)
			if _, ok := positionPropertyMap[name]; ok {
				loop.PropertyCalls = appendUnique(loop.PropertyCalls, name)
			}
			switch name {
			case "PositionClose":
				loop.BodyActions = appendUnique(loop.BodyActions, "position_close")
			case "PositionClosePartial":
				loop.BodyActions = appendUnique(loop.BodyActions, "position_close_partial")
			case "PositionCloseBy":
				loop.BodyActions = appendUnique(loop.BodyActions, "position_close_by")
			case "PositionModify":
				if isTrailingStopPattern(body, inner) {
					loop.BodyActions = appendUnique(loop.BodyActions, "position_modify:trailing_stop")
				} else {
					loop.BodyActions = appendUnique(loop.BodyActions, "position_modify")
				}
			case "OrderDelete":
				loop.BodyActions = appendUnique(loop.BodyActions, "order_delete")
			}
			// Check for magic/symbol filter via PositionGetInteger(POSITION_MAGIC) or PositionGetString(POSITION_SYMBOL)
			if name == "PositionGetInteger" || name == "PositionGetString" {
				args := childByType("", inner, "argument_list")
				if args != nil {
					argText := nodeText("", args)
					if strings.Contains(argText, "POSITION_MAGIC") {
						loop.HasMagicFilter = true
					}
					if strings.Contains(argText, "POSITION_SYMBOL") {
						loop.HasSymbolFilter = true
					}
				}
			}
			if name == "PositionGetSymbol" {
				loop.HasSymbolFilter = true
			}
			return true
		})
		loops = append(loops, loop)
		return false
	})
	return loops
}

// isTrailingStopPattern checks if an OrderModify call is part of a trailing stop
// pattern by looking for OrderStopLoss() comparison in the enclosing if condition.
func isTrailingStopPattern(body *sitter.Node, modifyCall *sitter.Node) bool {
	modifyStart := modifyCall.StartByte()
	found := false
	walkCST(body, func(n *sitter.Node) bool {
		if found {
			return false
		}
		if n.Type() != "if_statement" {
			return true
		}
		// Check if the OrderModify is inside this if's body
		if modifyStart >= n.StartByte() && modifyStart < n.EndByte() {
			cond := extractIfCondition(n)
			if cond != "" && strings.Contains(cond, "OrderStopLoss") {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
