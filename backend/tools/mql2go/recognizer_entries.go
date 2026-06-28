package mql2go

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// ── Entry recognition ───────────────────────────────────────────────

func extractEntriesCST(root *sitter.Node, version string) []EntryRule {
	var entries []EntryRule
	for _, fn := range findFunctions(root) {
		body := funcBody(fn)
		if body == nil {
			continue
		}
		scanForEntriesCST(body, &entries, version)
	}
	return entries
}

func scanForEntriesCST(body *sitter.Node, entries *[]EntryRule, version string) {
	for i := 0; i < int(body.ChildCount()); i++ {
		c := body.Child(i)
		switch c.Type() {
		case "if_statement":
			os := findOrderSendCST(c, version)
			if os != nil {
				e := entryFromOrderSend(os)
				e.Conditions = append(e.Conditions, extractIfCondition(c))
				*entries = append(*entries, e)
			}
		case "expression_statement":
			os := findOrderSendCST(c, version)
			if os != nil {
				*entries = append(*entries, entryFromOrderSend(os))
			}
		}
	}
}

func extractIfCondition(ifNode *sitter.Node) string {
	paren := childByType("", ifNode, "parenthesized_expression")
	if paren == nil {
		return ""
	}
	for i := 0; i < int(paren.NamedChildCount()); i++ {
		c := paren.NamedChild(i)
		return nodeText("", c)
	}
	return ""
}

func entryFromOrderSend(os *sitter.Node) EntryRule {
	e := EntryRule{}
	name := callFuncName(os)

	// CTrade method call (MQL5): trade.Buy(volume, symbol, price, sl, tp, comment)
	if isCTradeMethod(name) {
		return entryFromCTrade(os, name)
	}

	// MQL4 OrderSend: arg[1] is order type
	typeArg := callArgID(os, 1)
	switch typeArg {
	case "OP_BUY":
		e.Action = ActionMarketBuy
	case "OP_SELL":
		e.Action = ActionMarketSell
	case "OP_BUYLIMIT":
		e.Action = ActionBuyLimit
	case "OP_SELLLIMIT":
		e.Action = ActionSellLimit
	case "OP_BUYSTOP":
		e.Action = ActionBuyStop
	case "OP_SELLSTOP":
		e.Action = ActionSellStop
	}
	// Extract other args
	args := childByType("", os, "argument_list")
	if args != nil {
		named := getNamedChildren(args)
		if len(named) > 2 {
			e.Volume = nodeText("", named[2])
		}
		if len(named) > 3 {
			e.Price = nodeText("", named[3])
		}
		if len(named) > 5 {
			e.StopLoss = nodeText("", named[5])
		}
		if len(named) > 6 {
			e.TakeProfit = nodeText("", named[6])
		}
		if len(named) > 7 {
			e.Comment = nodeText("", named[7])
		}
		if len(named) > 8 {
			e.Magic = nodeText("", named[8])
		}
	}
	return e
}

// entryFromCTrade extracts an EntryRule from a CTrade method call.
// MQL5 CTrade signatures:
//   Buy(volume, symbol, price, sl, tp, comment)        → market buy
//   Sell(volume, symbol, price, sl, tp, comment)       → market sell
//   BuyLimit(volume, price, symbol, sl, tp, comment)   → buy limit
//   SellLimit(volume, price, symbol, sl, tp, comment)  → sell limit
//   BuyStop(volume, price, symbol, sl, tp, comment)    → buy stop
//   SellStop(volume, price, symbol, sl, tp, comment)   → sell stop
func entryFromCTrade(call *sitter.Node, methodName string) EntryRule {
	e := EntryRule{}
	switch methodName {
	case "Buy":
		e.Action = ActionMarketBuy
	case "Sell":
		e.Action = ActionMarketSell
	case "BuyLimit":
		e.Action = ActionBuyLimit
	case "SellLimit":
		e.Action = ActionSellLimit
	case "BuyStop":
		e.Action = ActionBuyStop
	case "SellStop":
		e.Action = ActionSellStop
	}

	args := childByType("", call, "argument_list")
	if args == nil {
		return e
	}
	named := getNamedChildren(args)

	// Market orders: Buy/Sell(volume, symbol, price, sl, tp, comment)
	// Pending orders: BuyLimit/SellLimit/BuyStop/SellStop(volume, price, symbol, sl, tp, comment)
	if methodName == "Buy" || methodName == "Sell" {
		if len(named) > 0 {
			e.Volume = nodeText("", named[0])
		}
		if len(named) > 2 {
			e.Price = nodeText("", named[2])
		}
		if len(named) > 3 {
			e.StopLoss = nodeText("", named[3])
		}
		if len(named) > 4 {
			e.TakeProfit = nodeText("", named[4])
		}
		if len(named) > 5 {
			e.Comment = nodeText("", named[5])
		}
	} else {
		// Pending orders: arg order is (volume, price, symbol, sl, tp, comment)
		if len(named) > 0 {
			e.Volume = nodeText("", named[0])
		}
		if len(named) > 1 {
			e.Price = nodeText("", named[1])
		}
		if len(named) > 3 {
			e.StopLoss = nodeText("", named[3])
		}
		if len(named) > 4 {
			e.TakeProfit = nodeText("", named[4])
		}
		if len(named) > 5 {
			e.Comment = nodeText("", named[5])
		}
	}
	return e
}

// ── Exit recognition ────────────────────────────────────────────────

func extractExitsCST(root *sitter.Node, version string) []ExitRule {
	var exits []ExitRule
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() == "call_expression" {
			name := callFuncName(n)
			switch name {
			case "OrderClose":
				if version != "mql4" {
					return true
				}
				exit := ExitRule{Trigger: TriggerMagic, Action: "position_close", MagicVal: "s.magic"}
				if cond := findEnclosingIfCondition(root, n); cond != "" {
					exit.Action = "position_close:" + cond
				}
				exits = append(exits, exit)
			case "OrderDelete":
				exit := ExitRule{Trigger: TriggerDelete, Action: "order_delete", MagicVal: "s.magic"}
				if cond := findEnclosingIfCondition(root, n); cond != "" {
					exit.Action = "order_delete:" + cond
				}
				exits = append(exits, exit)
			case "OrderCloseBy":
				if version != "mql4" {
					return true
				}
				exit := ExitRule{Trigger: TriggerAll, Action: "position_close_by", MagicVal: "s.magic"}
				if cond := findEnclosingIfCondition(root, n); cond != "" {
					exit.Action = "position_close_by:" + cond
				}
				exits = append(exits, exit)
			case "PositionClose":
				if version != "mql5" {
					return true
				}
				exit := ExitRule{Trigger: TriggerMagic, Action: "position_close", MagicVal: "s.magic"}
				if cond := findEnclosingIfCondition(root, n); cond != "" {
					exit.Action = "position_close:" + cond
				}
				exits = append(exits, exit)
			case "PositionClosePartial":
				if version != "mql5" {
					return true
				}
				exit := ExitRule{Trigger: TriggerMagic, Action: "position_close_partial", MagicVal: "s.magic"}
				if cond := findEnclosingIfCondition(root, n); cond != "" {
					exit.Action = "position_close_partial:" + cond
				}
				exits = append(exits, exit)
			case "PositionCloseBy":
				if version != "mql5" {
					return true
				}
				exit := ExitRule{Trigger: TriggerAll, Action: "position_close_by", MagicVal: "s.magic"}
				if cond := findEnclosingIfCondition(root, n); cond != "" {
					exit.Action = "position_close_by:" + cond
				}
				exits = append(exits, exit)
		}
		}
		return true
	})
	return exits
}

// findEnclosingIfCondition walks the CST to find the if statement that encloses
// the given call node, and returns its condition text.
func findEnclosingIfCondition(root *sitter.Node, target *sitter.Node) string {
	targetStart := target.StartByte()
	var result string
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "if_statement" {
			return true
		}
		// Check if target is within this if's body
		if targetStart >= n.StartByte() && targetStart < n.EndByte() {
			if cond := extractIfCondition(n); cond != "" {
				result = cond
				// Don't return — keep walking to find a deeper (innermost) if
			}
		}
		return true
	})
	return result
}

// ── Modify recognition (OrderModify / CTrade.PositionModify) ────────

func extractModifiesCST(root *sitter.Node, version string) []ModifyRule {
	var modifies []ModifyRule
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "call_expression" {
			return true
		}
		name := callFuncName(n)
		switch name {
		case "OrderModify":
			if version != "mql4" {
				return true
			}
			mr := ModifyRule{MagicVal: "s.magic", Kind: "manual_modify"}
			args := childByType("", n, "argument_list")
			if args != nil {
				named := getNamedChildren(args)
				if len(named) > 2 {
					mr.StopLoss = nodeText("", named[2])
				}
				if len(named) > 3 {
					mr.TakeProfit = nodeText("", named[3])
				}
			}
			if cond := findEnclosingIfCondition(root, n); cond != "" {
				mr.Condition = cond
				if strings.Contains(cond, "OrderStopLoss") || strings.Contains(cond, "TrailingStop") {
					mr.Kind = "trailing_stop"
				}
			}
			modifies = append(modifies, mr)
		case "PositionModify":
			if version != "mql5" {
				return true
			}
			mr := ModifyRule{MagicVal: "s.magic"}
			args := childByType("", n, "argument_list")
			if args != nil {
				named := getNamedChildren(args)
				if len(named) > 1 {
					mr.StopLoss = nodeText("", named[1])
				}
				if len(named) > 2 {
					mr.TakeProfit = nodeText("", named[2])
				}
			}
			if cond := findEnclosingIfCondition(root, n); cond != "" {
				mr.Condition = cond
			}
			modifies = append(modifies, mr)
		}
		return true
	})
	return modifies
}
