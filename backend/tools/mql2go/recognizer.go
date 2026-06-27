package mql2go

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// ── CST helpers ──────────────────────────────────────────────────────

func nodeType(n *sitter.Node) string {
	if n == nil {
		return ""
	}
	return n.Type()
}

func nodeText(source string, n *sitter.Node) string {
	if n == nil {
		return ""
	}
	if source == "" { source = parseSource }
	return source[n.StartByte():n.EndByte()]
}

func childByType(source string, n *sitter.Node, kind string) *sitter.Node {
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c.Type() == kind {
			return c
		}
	}
	return nil
}

func childrenByType(n *sitter.Node, kind string) []*sitter.Node {
	var out []*sitter.Node
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c.Type() == kind {
			out = append(out, c)
		}
	}
	return out
}

func findChild(source string, n *sitter.Node, kinds ...string) *sitter.Node {
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		for _, k := range kinds {
			if c.Type() == k {
				return c
			}
		}
	}
	return nil
}

func findNamedChild(n *sitter.Node, kinds ...string) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		c := n.NamedChild(i)
		for _, k := range kinds {
			if c.Type() == k {
				return c
			}
		}
	}
	return nil
}

// walkCST recursively walks CST nodes, calling visitor on every node.
func walkCST(n *sitter.Node, visitor func(*sitter.Node) bool) {
	if n == nil {
		return
	}
	if !visitor(n) {
		return
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		walkCST(n.Child(i), visitor)
	}
}

// findCall recursively searches for a call expression with the given name.
func findCall(n *sitter.Node, name string) *sitter.Node {
	var found *sitter.Node
	walkCST(n, func(n *sitter.Node) bool {
		if found != nil {
			return false
		}
		if n.Type() == "call_expression" {
			if id := childByType("", n, "identifier"); id != nil && nodeText("", id) == name {
				found = n
				return false
			}
			if id := childByType("", n, "field_identifier"); id != nil && nodeText("", id) == name {
				found = n
				return false
			}
		}
		return true
	})
	return found
}

// findOrderSendCST finds an order-placement call in a subtree.
// For MQL4: looks for OrderSend(symbol, OP_BUY, ...).
// For MQL5: looks for CTrade method calls (trade.Buy/Sell/BuyLimit/etc).
func findOrderSendCST(n *sitter.Node, version string) *sitter.Node {
	if n == nil {
		return nil
	}
	switch n.Type() {
	case "expression_statement":
		if call := childByType("", n, "call_expression"); call != nil {
			name := callFuncName(call)
			if version == "mql5" {
				if isCTradeMethod(name) {
					return call
				}
			} else {
				if name == "OrderSend" {
					return call
				}
			}
		}
	case "compound_statement":
		for i := 0; i < int(n.ChildCount()); i++ {
			if result := findOrderSendCST(n.Child(i), version); result != nil {
				return result
			}
		}
	case "if_statement":
		if result := findOrderSendCST(findNamedChild(n, "compound_statement"), version); result != nil {
			return result
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			c := n.Child(i)
			if c.Type() == "else_clause" {
				if result := findOrderSendCST(c, version); result != nil {
					return result
				}
			}
		}
	case "else_clause":
		for i := 0; i < int(n.ChildCount()); i++ {
			if result := findOrderSendCST(n.Child(i), version); result != nil {
				return result
			}
		}
	case "for_statement":
		body := childByType("", n, "compound_statement")
		if result := findOrderSendCST(body, version); result != nil {
			return result
		}
	}
	return nil
}

// isCTradeMethod returns true for CTrade Buy/Sell/BuyLimit/etc method names.
func isCTradeMethod(name string) bool {
	switch name {
	case "Buy", "Sell", "BuyLimit", "SellLimit", "BuyStop", "SellStop":
		return true
	}
	return false
}

// isCTradeExitMethod returns true for CTrade position/order close/delete methods.
func isCTradeExitMethod(name string) bool {
	switch name {
	case "PositionClose", "PositionClosePartial", "PositionCloseBy",
		"OrderDelete":
		return true
	}
	return false
}

// isCTradeModifyMethod returns true for CTrade modify methods.
func isCTradeModifyMethod(name string) bool {
	switch name {
	case "PositionModify", "OrderModify":
		return true
	}
	return false
}

func callArg(n *sitter.Node, idx int) string {
	args := childByType("", n, "argument_list")
	if args == nil {
		return ""
	}
	named := getNamedChildren(args)
	if idx < len(named) {
		c := named[idx]
		return nodeText("", c)
	}
	return ""
}

func callFuncName(n *sitter.Node) string {
	// Direct identifier (simple function call: OrderSend(...))
	if id := childByType("", n, "identifier"); id != nil {
		return nodeText("", id)
	}
	// Method call via field_expression (trade.Buy(...))
	if fe := childByType("", n, "field_expression"); fe != nil {
		if id := childByType("", fe, "field_identifier"); id != nil {
			return nodeText("", id)
		}
		if id := childByType("", fe, "identifier"); id != nil {
			return nodeText("", id)
		}
	}
	// Direct field_identifier (fallback)
	if id := childByType("", n, "field_identifier"); id != nil {
		return nodeText("", id)
	}
	if id := childByType("", n, "statement_identifier"); id != nil {
		return nodeText("", id)
	}
	return ""
}

func callArgID(n *sitter.Node, idx int) string {
	args := childByType("", n, "argument_list")
	if args == nil {
		return ""
	}
	named := getNamedChildren(args)
	if idx < len(named) {
		c := named[idx]
		// if the arg is itself an identifier, return its name
		if id := childByType("", c, "identifier"); id != nil {
			return nodeText("", id)
		}
		if id := childByType("", c, "field_identifier"); id != nil {
			return nodeText("", id)
		}
		return nodeText("", c)
	}
	return ""
}

// ── Function extraction ─────────────────────────────────────────────

func findFunctions(root *sitter.Node) []*sitter.Node {
	var fns []*sitter.Node
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() == "function_definition" {
			fns = append(fns, n)
		}
		return true
	})
	return fns
}

func funcName(n *sitter.Node) string {
	decl := childByType("", n, "function_declarator")
	if decl != nil {
		id := childByType("", decl, "identifier")
		if id == nil {
			id = childByType("", decl, "field_identifier")
		}
		if id == nil {
			id = childByType("", decl, "statement_identifier")
		}
		return nodeText("", id)
	}
	// fallback: direct identifier child
	id := childByType("", n, "identifier")
	if id == nil {
		id = childByType("", n, "field_identifier")
	}
	if id == nil {
		id = childByType("", n, "statement_identifier")
	}
	return nodeText("", id)
}

func funcBody(n *sitter.Node) *sitter.Node {
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c.Type() == "compound_statement" {
			return c
		}
	}
	return nil
}

// ── Parameter extraction ────────────────────────────────────────────

func extractParamsCST(source string, root *sitter.Node) []ParamSpec {
	var params []ParamSpec
	// Only process top-level declarations (direct children of translation_unit),
	// not declarations inside function bodies.
	for i := 0; i < int(root.ChildCount()); i++ {
		n := root.Child(i)
		if n.Type() != "declaration" && n.Type() != "field_declaration" {
			continue
		}
		text := source[n.StartByte():n.EndByte()]
		if !strings.Contains(text, "extern ") && !strings.Contains(text, "input ") {
			continue
		}
		var vt string
		if pt := childByType(source, n, "primitive_type"); pt != nil {
			vt = nodeText(source, pt)
		}
		init := childByType(source, n, "init_declarator")
		if init == nil {
			continue
		}
		name := ""
		if id := childByType(source, init, "identifier"); id != nil {
			name = nodeText(source, id)
		}
		var value string
		if nl := childByType(source, init, "number_literal"); nl != nil {
			value = nodeText(source, nl)
		} else if sl := childByType(source, init, "string_literal"); sl != nil {
			value = nodeText(source, sl)
			if len(value) >= 2 {
				value = value[1 : len(value)-1]
			}
		} else if fl := childByType(source, init, "false"); fl != nil {
			value = "false"
		} else if tr := childByType(source, init, "true"); tr != nil {
			value = "true"
		}
		if name == "" || isNoiseCST(name, value) {
			continue
		}
		params = append(params, ParamSpec{
			Name: name, Label: name, Type: mapMQLType(vt),
			Default: value, Group: guessGroupCST(name),
		})
	}
	return params
}

func isInsideFunction(n *sitter.Node) bool {
	// Walk up the parent chain to see if we're inside a function_definition
	// tree-sitter Go binding doesn't expose parent — use node position heuristic
	return false // TODO: implement when tree-sitter parent access is available
}

func isNoiseCST(name, value string) bool {
	if strings.Contains(name, "说明") || strings.Contains(name, "选择") || strings.Contains(name, "提示") {
		return true
	}
	if strings.Contains(value, "====") {
		return true
	}
	if len(value) > 20 && containsNonASCII(value) {
		return true
	}
	return false
}

func mapMQLType(t string) ParamType {
	switch t {
	case "int", "long", "uint", "ulong":
		return ParamInt
	case "double", "float":
		return ParamDouble
	case "string":
		return ParamString
	case "bool":
		return ParamBool
	}
	return ParamString
}

func guessGroupCST(name string) ParamGroup {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "lot") || strings.Contains(lower, "volume") {
		return GroupSizing
	}
	if strings.Contains(lower, "magic") || strings.Contains(lower, "comment") {
		return GroupSystem
	}
	if strings.Contains(lower, "sl") || strings.Contains(lower, "tp") ||
		strings.Contains(lower, "stop") || strings.Contains(lower, "take") {
		return GroupExit
	}
	return GroupEntry
}

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
			current := c
			for current != nil {
				if os := findOrderSendCST(current, version); os != nil {
					entry := entryFromOrderSend(os)
					if entry.Action != "" {
						if cond := extractIfCondition(current); cond != "" {
							entry.Conditions = []string{cond}
						}
						*entries = append(*entries, entry)
					}
				}
				ec := childByType("", current, "else_clause")
				if ec != nil {
					current = childByType("", ec, "if_statement")
				} else {
					current = nil
				}
			}
		case "expression_statement":
			if os := findOrderSendCST(c, version); os != nil {
				entry := entryFromOrderSend(os)
				if entry.Action != "" {
					*entries = append(*entries, entry)
				}
			}
		case "compound_statement":
			scanForEntriesCST(c, entries, version)
		}
	}
}

func extractIfCondition(ifNode *sitter.Node) string {
	paren := childByType("", ifNode, "parenthesized_expression")
	if paren == nil {
		return ""
	}
	// Get text between the parens (skip the '(' and ')')
	start := paren.StartByte() + 1
	end := paren.EndByte() - 1
	if start < end && int(end) <= len(parseSource) {
		return parseSource[start:end]
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

// ── Order loop recognition (MQL4: OrdersTotal + OrderSelect) ────────

// orderPropertyMap maps MQL4 Order* property functions to SDK Position fields.
var orderPropertyMap = map[string]string{
	"OrderTicket":     "Ticket",
	"OrderSymbol":     "Symbol",
	"OrderLots":       "Volume",
	"OrderOpenPrice":  "OpenPrice",
	"OrderStopLoss":   "StopLoss",
	"OrderTakeProfit": "TakeProfit",
	"OrderProfit":     "Profit",
	"OrderSwap":       "Swap",
	"OrderCommission": "Commission",
	"OrderComment":    "Comment",
	"OrderMagicNumber": "Magic",
	"OrderOpenTime":   "OpenTime",
	"OrderType":       "Type",
	"OrderClosePrice": "ClosePrice",
	"OrderCloseTime":  "CloseTime",
	"OrderExpiration": "Expiration",
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

// ── Execution model ─────────────────────────────────────────────────

func detectExecCST(root *sitter.Node) ExecutionModel {
	// Check for OnTick function → on_tick
	for _, fn := range findFunctions(root) {
		name := funcName(fn)
		if name == "OnTick" {
			return ExecutionModel{Kind: ExecOnTick}
		}
	}
	// Check for grid flag → on_init_grid
	for _, fn := range findFunctions(root) {
		body := funcBody(fn)
		if body == nil {
			continue
		}
		if hasGridFlagCST(body) {
			return ExecutionModel{Kind: ExecOnInitGrid}
		}
	}
	// Default: on_bar
	return ExecutionModel{Kind: ExecOnBar}
}

func hasGridFlagCST(body *sitter.Node) bool {
	found := false
	walkCST(body, func(n *sitter.Node) bool {
		if n.Type() == "identifier" && nodeText("", n) == "gridPlaced" {
			found = true
			return false
		}
		return true
	})
	return found
}

// ── State variables ─────────────────────────────────────────────────

func extractStateCST(source string, root *sitter.Node) []StateVar {
	var state []StateVar
	for i := 0; i < int(root.ChildCount()); i++ {
		n := root.Child(i)
		if n.Type() != "declaration" && n.Type() != "field_declaration" {
			continue
		}
		text := source[n.StartByte():n.EndByte()]
		if strings.Contains(text, "extern ") || strings.Contains(text, "input ") {
			continue
		}
		if childByType(source, n, "function_declarator") != nil {
			continue
		}
		var vt string
		if pt := childByType(source, n, "primitive_type"); pt != nil {
			vt = nodeText(source, pt)
		}
		init := childByType(source, n, "init_declarator")
		if init == nil {
			continue
		}
		name := ""
		if id := childByType(source, init, "identifier"); id != nil {
			name = nodeText(source, id)
		}
		if name == "" {
			continue
		}
		state = append(state, StateVar{Name: name, GoType: vt})
	}
	return state
}

// ── Sizing ──────────────────────────────────────────────────────────

func detectSizingCST(entries []EntryRule) *SizingRule {
	k := SizingFixed
	expr := "s.lotSize"
	for _, e := range entries {
		if strings.Contains(e.Volume, "MathPow") || strings.Contains(e.Volume, "*") {
			k = SizingMartingale
			expr = "s.baseLot"
			break
		}
	}
	return &SizingRule{Kind: k, Expression: expr}
}

// ── Timer ──────────────────────────────────────────────────────────

func detectTimerCST(root *sitter.Node) *TimerRule {
	var timer *TimerRule
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "call_expression" {
			return true
		}
		name := callFuncName(n)
		if name == "EventSetTimer" || name == "EventSetMillisecondTimer" {
			secs := 300
			if name == "EventSetMillisecondTimer" {
				secs = 5
			}
			// Extract interval arg
			if name == "EventSetTimer" {
				args := childByType("", n, "argument_list")
				if args != nil {
					named := getNamedChildren(args)
					if len(named) > 0 {
						if nl := childByType("", named[0], "number_literal"); nl != nil {
							secs = parseInt(nodeText("", nl))
						}
					}
				}
			}
			timer = &TimerRule{IntervalSeconds: secs}
			return false
		}
		return true
	})
	return timer
}

// ── Indicators ──────────────────────────────────────────────────────

func extractIndicatorsCST(root *sitter.Node) []IndicatorSpec {
	var specs []IndicatorSpec
	seen := make(map[string]bool)
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "call_expression" {
			return true
		}
		name := callFuncName(n)
		method := indicatorMethodCST(name)
		if method == "" {
			return true
		}
		// Deduplicate
		key := name + ":" + nodeText("", n)
		if seen[key] {
			return true
		}
		seen[key] = true

		params := make(map[string]string)
		spec := IndicatorSpec{SDKMethod: method, Params: params}
		// Extract common indicator args
		args := childByType("", n, "argument_list")
		if args != nil {
			named := getNamedChildren(args)
			switch name {
			case "iMA":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 3 {
					params["shift"] = nodeText("", named[3])
				}
			case "iRSI":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["shift"] = nodeText("", named[4])
				}
			case "iATR":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 3 {
					params["shift"] = nodeText("", named[3])
				}
			case "iMACD":
				if len(named) > 2 {
					params["fast"] = nodeText("", named[2])
				}
				if len(named) > 3 {
					params["slow"] = nodeText("", named[3])
				}
				if len(named) > 4 {
					params["signal"] = nodeText("", named[4])
				}
				if len(named) > 5 {
					params["shift"] = nodeText("", named[5])
				}
			case "iBands":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 3 {
					params["deviation"] = nodeText("", named[3])
				}
				if len(named) > 5 {
					params["shift"] = nodeText("", named[5])
				}
			case "iStochastic":
				if len(named) > 2 {
					params["kperiod"] = nodeText("", named[2])
				}
				if len(named) > 3 {
					params["dperiod"] = nodeText("", named[3])
				}
				if len(named) > 4 {
					params["slowing"] = nodeText("", named[4])
				}
				if len(named) > 6 {
					params["shift"] = nodeText("", named[6])
				}
			case "iCCI":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["shift"] = nodeText("", named[4])
				}
			case "iADX":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 3 {
					params["shift"] = nodeText("", named[3])
				}
			case "iMomentum":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["shift"] = nodeText("", named[4])
				}
			case "iWPR":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["shift"] = nodeText("", named[4])
				}
			case "iMFI":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["shift"] = nodeText("", named[4])
				}
			case "iOBV":
				if len(named) > 1 {
					params["shift"] = nodeText("", named[1])
				}
			case "iSAR":
				if len(named) > 2 {
					params["step"] = nodeText("", named[2])
				}
				if len(named) > 3 {
					params["maximum"] = nodeText("", named[3])
				}
				if len(named) > 4 {
					params["shift"] = nodeText("", named[4])
				}
			case "iStdDev":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["shift"] = nodeText("", named[4])
				}
			case "iAlligator":
				if len(named) > 2 {
					params["jaw_period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["teeth_period"] = nodeText("", named[4])
				}
				if len(named) > 6 {
					params["lips_period"] = nodeText("", named[6])
				}
				if len(named) > 10 {
					params["shift"] = nodeText("", named[10])
				}
			case "iIchimoku":
				if len(named) > 2 {
					params["tenkan"] = nodeText("", named[2])
				}
				if len(named) > 3 {
					params["kijun"] = nodeText("", named[3])
				}
				if len(named) > 4 {
					params["senkou_b"] = nodeText("", named[4])
				}
				if len(named) > 5 {
					params["shift"] = nodeText("", named[5])
				}
			case "iEnvelopes":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 5 {
					params["deviation"] = nodeText("", named[5])
				}
				if len(named) > 7 {
					params["shift"] = nodeText("", named[7])
				}
			case "iDeMarker":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 3 {
					params["shift"] = nodeText("", named[3])
				}
			case "iOsMA":
				if len(named) > 2 {
					params["fast"] = nodeText("", named[2])
				}
				if len(named) > 3 {
					params["slow"] = nodeText("", named[3])
				}
				if len(named) > 4 {
					params["signal"] = nodeText("", named[4])
				}
				if len(named) > 6 {
					params["shift"] = nodeText("", named[6])
				}
			case "iRVI":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["shift"] = nodeText("", named[4])
				}
			case "iForce":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 5 {
					params["shift"] = nodeText("", named[5])
				}
			case "iFractals":
				if len(named) > 3 {
					params["shift"] = nodeText("", named[3])
				}
			case "iGator":
				if len(named) > 2 {
					params["jaw_period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["teeth_period"] = nodeText("", named[4])
				}
				if len(named) > 6 {
					params["lips_period"] = nodeText("", named[6])
				}
				if len(named) > 10 {
					params["shift"] = nodeText("", named[10])
				}
			case "iAC":
				if len(named) > 2 {
					params["shift"] = nodeText("", named[2])
				}
			case "iAD":
				if len(named) > 2 {
					params["shift"] = nodeText("", named[2])
				}
			case "iAO":
				if len(named) > 2 {
					params["shift"] = nodeText("", named[2])
				}
			case "iBearsPower":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["shift"] = nodeText("", named[4])
				}
			case "iBullsPower":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["shift"] = nodeText("", named[4])
				}
			case "iBWMFI":
				if len(named) > 2 {
					params["shift"] = nodeText("", named[2])
				}
			case "iAMA":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 3 {
					params["fast"] = nodeText("", named[3])
				}
				if len(named) > 4 {
					params["slow"] = nodeText("", named[4])
				}
				if len(named) > 7 {
					params["shift"] = nodeText("", named[7])
				}
			case "iDEMA":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 5 {
					params["shift"] = nodeText("", named[5])
				}
			case "iTEMA":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 5 {
					params["shift"] = nodeText("", named[5])
				}
			case "iFrAMA":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 5 {
					params["shift"] = nodeText("", named[5])
				}
			case "iVIDyA":
				if len(named) > 2 {
					params["cmo_period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["period"] = nodeText("", named[4])
				}
				if len(named) > 7 {
					params["shift"] = nodeText("", named[7])
				}
			case "iTriX":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 5 {
					params["shift"] = nodeText("", named[5])
				}
			case "iADXWilder":
				if len(named) > 2 {
					params["period"] = nodeText("", named[2])
				}
				if len(named) > 4 {
					params["shift"] = nodeText("", named[4])
				}
			case "iChaikin":
				if len(named) > 2 {
					params["fast"] = nodeText("", named[2])
				}
				if len(named) > 3 {
					params["slow"] = nodeText("", named[3])
				}
				if len(named) > 6 {
					params["shift"] = nodeText("", named[6])
				}
			case "iVolumes":
				if len(named) > 3 {
					params["shift"] = nodeText("", named[3])
				}
			}
		}
		// Try to find result variable from the enclosing declaration
		spec.ResultVar = findResultVarForNode(root, n)
		specs = append(specs, spec)
		return true
	})
	return specs
}

func findResultVarForNode(root *sitter.Node, callNode *sitter.Node) string {
	callStart := callNode.StartByte()
	for _, fn := range findFunctions(root) {
		body := funcBody(fn)
		if body == nil {
			continue
		}
		for i := 0; i < int(body.ChildCount()); i++ {
			c := body.Child(i)
			// Case 1: declaration with init (double maVal = iMA(...))
			if c.Type() == "declaration" {
				init := childByType("", c, "init_declarator")
				if init == nil {
					continue
				}
				call := childByType("", init, "call_expression")
				if call == nil {
					continue
				}
				if call.StartByte() == callStart {
					id := childByType("", init, "identifier")
					if id != nil {
						return nodeText("", id)
					}
				}
			}
			// Case 2: assignment expression (maVal = iMA(...))
			if c.Type() == "expression_statement" {
				assign := childByType("", c, "assignment_expression")
				if assign == nil {
					continue
				}
				call := childByType("", assign, "call_expression")
				if call == nil {
					continue
				}
				if call.StartByte() == callStart {
					// LHS is the first named child (left = identifier)
					named := getNamedChildren(assign)
					if len(named) > 0 {
						lhs := named[0]
						if lhs.Type() == "identifier" || lhs.Type() == "field_identifier" {
							return nodeText("", lhs)
						}
						// Try child identifier (for complex LHS)
						id := childByType("", lhs, "identifier")
						if id == nil {
							id = childByType("", lhs, "field_identifier")
						}
						if id != nil {
							return nodeText("", id)
						}
					}
				}
			}
		}
	}
	return ""
}

func findResultVar(root *sitter.Node, indicatorName string) string {
	for _, fn := range findFunctions(root) {
		body := funcBody(fn)
		if body == nil {
			continue
		}
		for i := 0; i < int(body.ChildCount()); i++ {
			c := body.Child(i)
			if c.Type() != "declaration" {
				continue
			}
			// Result var is in init_declarator > identifier
			init := childByType("", c, "init_declarator")
			if init == nil {
				continue
			}
			call := childByType("", init, "call_expression")
			if call == nil {
				continue
			}
			if callFuncName(call) == indicatorName {
				id := childByType("", init, "identifier")
				if id != nil {
					return nodeText("", id)
				}
			}
		}
	}
	return ""
}

func indicatorMethodCST(name string) string {
	switch name {
	case "iMA":
		return "ema"
	case "iRSI":
		return "rsi"
	case "iATR":
		return "atr"
	case "iBands":
		return "bands"
	case "iMACD":
		return "macd"
	case "iStochastic":
		return "stochastic"
	case "iCCI":
		return "cci"
	case "iADX":
		return "adx"
	case "iMomentum":
		return "momentum"
	case "iWPR":
		return "wpr"
	case "iMFI":
		return "mfi"
	case "iOBV":
		return "obv"
	case "iSAR":
		return "sar"
	case "iStdDev":
		return "stddev"
	case "iCustom":
		return "i_custom"
	case "iAlligator":
		return "alligator"
	case "iIchimoku":
		return "ichimoku"
	case "iEnvelopes":
		return "envelopes"
	case "iDeMarker":
		return "demarker"
	case "iOsMA":
		return "osma"
	case "iRVI":
		return "rvi"
	case "iForce":
		return "force"
	case "iFractals":
		return "fractals"
	case "iGator":
		return "gator"
	case "iAC":
		return "ac"
	case "iAD":
		return "ad"
	case "iAO":
		return "ao"
	case "iBearsPower":
		return "bears_power"
	case "iBullsPower":
		return "bulls_power"
	case "iBWMFI":
		return "bwmfi"
	case "iAMA":
		return "ama"
	case "iDEMA":
		return "dema"
	case "iTEMA":
		return "tema"
	case "iFrAMA":
		return "frama"
	case "iVIDyA":
		return "vidya"
	case "iTriX":
		return "trix"
	case "iADXWilder":
		return "adx_wilder"
	case "iChaikin":
		return "chaikin"
	case "iVolumes":
		return "volumes"
	}
	return ""
}

func getNamedChildren(n *sitter.Node) []*sitter.Node {
	var out []*sitter.Node
	if n == nil {
		return out
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		out = append(out, n.NamedChild(i))
	}
	return out
}

func containsNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// ── Risk checks ────────────────────────────────────────────────────

func extractRiskChecksCST(root *sitter.Node, version string) []RiskCheck {
	var checks []RiskCheck
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "call_expression" {
			return true
		}
		name := callFuncName(n)
		switch name {
		case "AccountFreeMargin":
			checks = append(checks, RiskCheck{
				Kind:      "margin_check",
				Condition: "AccountFreeMargin() <= 0",
				Action:    "skip_order",
				Trigger:   "pre_order",
			})
		case "IsTradeAllowed":
			checks = append(checks, RiskCheck{
				Kind:      "trade_allowed",
				Condition: "!IsTradeAllowed()",
				Action:    "skip_order",
				Trigger:   "pre_order",
			})
		case "IsExpertEnabled":
			checks = append(checks, RiskCheck{
				Kind:      "expert_enabled",
				Condition: "!IsExpertEnabled()",
				Action:    "skip_order",
				Trigger:   "pre_order",
			})
		}
		return true
	})
	return checks
}

// ── Blind spots ────────────────────────────────────────────────────

func detectBlindSpots(source string, root *sitter.Node, intent *StrategyIntent) []BlindSpot {
	var spots []BlindSpot

	// Check for unsupported MQL patterns
	unsupportedCalls := map[string]BlindSpot{
		"OrderModify": {
			ID:          "BS_OrderModify",
			Category:    "order_modify",
			Severity:    "信息",
			Description: "OrderModify (修改订单 SL/TP) 已部分转译为 ctx.Broker().PositionModify()",
			Handling:    "检查生成的修改逻辑是否覆盖原始条件",
		},
		"MarketInfo": {
			ID:          "BS_MarketInfo",
			Category:    "market_data",
			Severity:    "信息",
			Description: "MarketInfo() 调用未转译，需手动替换为 ctx 方法",
			Handling:    "用 ctx.Bid()/ctx.Ask() 等替代",
		},
		"ObjectCreate": {
			ID:          "BS_ObjectCreate",
			Category:    "chart_objects",
			Severity:    "信息",
			Description: "图表对象操作 (ObjectCreate/ObjectDelete) 不支持，Go SDK 无图表 API",
			Handling:    "忽略，图表对象不影响策略逻辑",
		},
		"SendMail": {
			ID:          "BS_SendMail",
			Category:    "notification",
			Severity:    "信息",
			Description: "SendMail/SendNotification 不支持",
			Handling:    "用 ctx.Notify() 替代或忽略",
		},
		"FileOpen": {
			ID:          "BS_FileIO",
			Category:    "file_io",
			Severity:    "警告",
			Description: "文件 I/O 操作不支持，Go 策略无文件系统访问",
			Handling:    "用 ctx.ParamString() 读取配置替代",
		},
		"DLLImport": {
			ID:          "BS_DLLImport",
			Category:    "dll_import",
			Severity:    "致命",
			Description: "DLL 导入 (#import) 不支持，Go 策略无法调用外部 DLL",
			Handling:    "需手动用 Go 重新实现 DLL 功能",
		},
	}

	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "call_expression" {
			return true
		}
		name := callFuncName(n)
		if spot, ok := unsupportedCalls[name]; ok {
			spot.Location = fmt.Sprintf("line %d", n.StartPoint().Row+1)
			spot.UserActionRequired = spot.Severity == "致命"
			spots = append(spots, spot)
		}
		return true
	})

	// Check for #import directives in source text
	if strings.Contains(source, "#import") {
		spots = append(spots, unsupportedCalls["DLLImport"])
	}

	// Check for OrdersTotal() iteration pattern (not fully supported)
	hasOrdersTotal := false
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() == "call_expression" && callFuncName(n) == "OrdersTotal" {
			hasOrdersTotal = true
			return false
		}
		return true
	})
	if hasOrdersTotal {
		spots = append(spots, BlindSpot{
			ID:                 "BS_OrdersTotal",
			Category:           "order_iteration",
			Severity:           "警告",
			Description:        "OrdersTotal() 遍历模式已转译为 ctx.Broker().Positions() 遍历，但条件过滤可能不完整",
			Handling:           "检查生成的 closeAll 逻辑是否覆盖原始遍历条件",
			UserActionRequired: false,
		})
	}

	// Check for string operations that pyToGoExpr might not handle
	if strings.Contains(source, "StringConcatenate") || strings.Contains(source, "StringFormat") {
		spots = append(spots, BlindSpot{
			ID:          "BS_StringOps",
			Category:    "string_ops",
			Severity:    "信息",
			Description: "字符串操作函数未转译，需手动用 Go fmt.Sprintf 替代",
			Handling:    "用 fmt.Sprintf 替代",
		})
	}

	// Check for iCustom indicators
	for _, ind := range intent.Indicators {
		if ind.SDKMethod == "i_custom" {
			spots = append(spots, BlindSpot{
				ID:          "BS_iCustom",
				Category:    "custom_indicator",
				Severity:    "警告",
				Description: "自定义指标 iCustom() 无法自动转译，需手动实现",
				Handling:    "在 ctx.Indicators().ICustom() 中实现或用标准指标替代",
			})
			break
		}
	}

	// Check for indicators that have TODO stubs (SDK method not yet implemented)
	todoIndicators := map[string]string{
		"alligator":    "iAlligator",
		"ichimoku":     "iIchimoku",
		"envelopes":    "iEnvelopes",
		"demarker":     "iDeMarker",
		"osma":         "iOsMA",
		"rvi":          "iRVI",
		"force":        "iForce",
		"fractals":     "iFractals",
		"gator":        "iGator",
		"ac":           "iAC",
		"ad":           "iAD",
		"ao":           "iAO",
		"bears_power":  "iBearsPower",
		"bulls_power":  "iBullsPower",
		"bwmfi":        "iBWMFI",
		"ama":          "iAMA",
		"dema":         "iDEMA",
		"tema":         "iTEMA",
		"frama":        "iFrAMA",
		"vidya":        "iVIDyA",
		"trix":         "iTriX",
		"adx_wilder":   "iADXWilder",
		"chaikin":      "iChaikin",
		"volumes":      "iVolumes",
	}
	for _, ind := range intent.Indicators {
		if origName, ok := todoIndicators[ind.SDKMethod]; ok {
			spots = append(spots, BlindSpot{
				ID:          "BS_" + ind.SDKMethod,
				Category:    "indicator_stub",
				Severity:    "警告",
				Description: origName + "() 已识别但 SDK 尚未实现，生成 TODO 桩",
				Handling:    "在 IndicatorSet 接口中实现 " + origName + " 方法",
			})
		}
	}

	// Check for *OnArray indicator variants (MQL4-specific)
	onArrayIndicators := []string{
		"iMAOnArray", "iRSIOnArray", "iBandsOnArray",
		"iCCIOnArray", "iStdDevOnArray", "iMomentumOnArray",
	}
	for _, name := range onArrayIndicators {
		if strings.Contains(source, name) {
			spots = append(spots, BlindSpot{
				ID:          "BS_" + name,
				Category:    "indicator_on_array",
				Severity:    "警告",
				Description: name + "() 在自定义数组上计算指标，Go SDK 不支持",
				Handling:    "需手动实现数组指标计算逻辑",
			})
			break
		}
	}

	// Check for OnTester event (MQL4/MQL5 backtest optimization)
	for _, fn := range findFunctions(root) {
		if funcName(fn) == "OnTester" {
			spots = append(spots, BlindSpot{
				ID:          "BS_OnTester",
				Category:    "on_tester",
				Severity:    "信息",
				Description: "OnTester() 回测优化自定义指标函数不支持，Go 回测引擎无对应接口",
				Handling:    "忽略，不影响策略执行逻辑",
			})
			break
		}
	}

	// MQL5-specific event handlers
	if intent.Meta.MQLVersion == "mql5" {
		mql5Events := map[string]BlindSpot{
			"OnTrade": {
				ID:          "BS_OnTrade",
				Category:    "on_trade",
				Severity:    "信息",
				Description: "OnTrade() 交易事件回调不支持，Go SDK 无交易事件通知接口",
				Handling:    "忽略，策略逻辑应在 OnTick 中检查持仓状态",
			},
			"OnTradeTransaction": {
				ID:          "BS_OnTradeTransaction",
				Category:    "on_trade_transaction",
				Severity:    "信息",
				Description: "OnTradeTransaction() 交易事务回调不支持，Go SDK 无交易事务通知接口",
				Handling:    "忽略，策略逻辑应在 OnTick 中检查持仓状态",
			},
			"OnBookEvent": {
				ID:          "BS_OnBookEvent",
				Category:    "on_book_event",
				Severity:    "信息",
				Description: "OnBookEvent() 市场深度事件回调不支持，Go SDK 无市场深度接口",
				Handling:    "忽略，不影响策略执行逻辑",
			},
			"OnTesterInit": {
				ID:          "BS_OnTesterInit",
				Category:    "on_tester_init",
				Severity:    "信息",
				Description: "OnTesterInit() 优化开始事件不支持，Go 回测引擎无对应接口",
				Handling:    "忽略，不影响策略执行逻辑",
			},
			"OnTesterDeinit": {
				ID:          "BS_OnTesterDeinit",
				Category:    "on_tester_deinit",
				Severity:    "信息",
				Description: "OnTesterDeinit() 优化结束事件不支持，Go 回测引擎无对应接口",
				Handling:    "忽略，不影响策略执行逻辑",
			},
			"OnTesterPass": {
				ID:          "BS_OnTesterPass",
				Category:    "on_tester_pass",
				Severity:    "信息",
				Description: "OnTesterPass() 优化数据帧事件不支持，Go 回测引擎无对应接口",
				Handling:    "忽略，不影响策略执行逻辑",
			},
		}
		for _, fn := range findFunctions(root) {
			name := funcName(fn)
			if bs, ok := mql5Events[name]; ok {
				spots = append(spots, bs)
			}
		}
	}

	// MQL5: native OrderSend(MqlTradeRequest, MqlTradeResult) — struct-based, not inline params
	if intent.Meta.MQLVersion == "mql5" {
		hasNativeOrderSend := false
		walkCST(root, func(n *sitter.Node) bool {
			if n.Type() == "call_expression" && callFuncName(n) == "OrderSend" {
				// Check if it's the MQL5 struct-based OrderSend (2 args, not MQL4's 11 args)
				args := childByType("", n, "argument_list")
				if args != nil {
					named := getNamedChildren(args)
					if len(named) <= 2 {
						hasNativeOrderSend = true
						return false
					}
				}
			}
			return true
		})
		if hasNativeOrderSend {
			spots = append(spots, BlindSpot{
				ID:          "BS_NativeOrderSend",
				Category:    "native_ordersend",
				Severity:    "警告",
				Description: "MQL5 原生 OrderSend(MqlTradeRequest, MqlTradeResult) 结构体方式下单不支持自动转译",
				Handling:    "需手动转换为 ctx.Broker().OrderSend() 调用",
			})
		}
	}

	// MQL5: PositionGet* property functions used outside of recognized loop
	if intent.Meta.MQLVersion == "mql5" {
		mql5PosProps := []string{"PositionGetDouble", "PositionGetInteger", "PositionGetString", "PositionGetSymbol"}
		for _, propFunc := range mql5PosProps {
			if strings.Contains(source, propFunc) && len(intent.PositionLoops) > 0 {
				// Already recognized in PositionLoopRule — info level
				spots = append(spots, BlindSpot{
					ID:          "BS_" + propFunc,
					Category:    "position_property",
					Severity:    "信息",
					Description: propFunc + "() 持仓属性函数已识别为 PositionLoopRule 的一部分",
					Handling:    "检查生成的遍历逻辑是否覆盖原始属性访问",
				})
				break
			} else if strings.Contains(source, propFunc) {
				spots = append(spots, BlindSpot{
					ID:          "BS_" + propFunc,
					Category:    "position_property",
					Severity:    "警告",
					Description: propFunc + "() 持仓属性函数已检测到但未识别为标准遍历模式",
					Handling:    "需手动检查持仓属性访问是否正确转译",
				})
				break
			}
		}
	}

	// Check for OrderSelect iteration pattern (MQL4-specific)
	if intent.Meta.MQLVersion == "mql4" {
		hasOrderSelect := false
		walkCST(root, func(n *sitter.Node) bool {
			if n.Type() == "call_expression" && callFuncName(n) == "OrderSelect" {
				hasOrderSelect = true
				return false
			}
			return true
		})
		if hasOrderSelect && len(intent.OrderLoops) > 0 {
			// Already recognized as OrderLoopRule — downgrade to info
			spots = append(spots, BlindSpot{
				ID:          "BS_OrderSelect",
				Category:    "order_select",
				Severity:    "信息",
				Description: "OrderSelect() + Order* 属性函数遍历模式已识别为 OrderLoopRule",
				Handling:    "检查生成的遍历逻辑是否覆盖原始条件过滤",
			})
		} else if hasOrderSelect {
			spots = append(spots, BlindSpot{
				ID:          "BS_OrderSelect",
				Category:    "order_select",
				Severity:    "警告",
				Description: "OrderSelect() 调用已检测到但未识别为标准遍历模式",
				Handling:    "需手动检查订单遍历逻辑是否正确转译",
			})
		}
	}

	// MQL5 CTrade detection
	if strings.Contains(source, "CTrade") {
		spots = append(spots, BlindSpot{
			ID:                 "BS_CTrade",
			Category:           "mql5_ctrade",
			Severity:           "警告",
			Description:        "MQL5 CTrade 类用法已部分识别，但 OOP 方法调用可能不完整",
			Handling:           "检查生成的入场/出场逻辑是否覆盖所有 CTrade.Buy/Sell/PositionClose 调用",
			UserActionRequired: false,
		})
	}

	// MQL5-specific: PositionsTotal / PositionSelect / PositionGetTicket
	if intent.Meta.MQLVersion == "mql5" {
		mql5Calls := map[string]BlindSpot{
			"PositionsTotal": {
				ID:          "BS_PositionsTotal",
				Category:    "mql5_position_iter",
				Severity:    "信息",
				Description: "MQL5 PositionsTotal() 已转译为 ctx.Broker().Positions() 遍历",
				Handling:    "检查生成的遍历逻辑是否覆盖原始条件",
			},
			"PositionSelect": {
				ID:          "BS_PositionSelect",
				Category:    "mql5_position_select",
				Severity:    "警告",
				Description: "MQL5 PositionSelect() 按品种选择持仓的模式未完全转译",
				Handling:    "需手动检查生成的持仓选择逻辑",
			},
			"PositionGetTicket": {
				ID:          "BS_PositionGetTicket",
				Category:    "mql5_position_ticket",
				Severity:    "警告",
				Description: "MQL5 PositionGetTicket() 获取持仓票据的模式未转译",
				Handling:    "用 ctx.Broker().Positions() 返回的 Position.Ticket 替代",
			},
		}
		walkCST(root, func(n *sitter.Node) bool {
			if n.Type() != "call_expression" {
				return true
			}
			name := callFuncName(n)
			if spot, ok := mql5Calls[name]; ok {
				spot.Location = fmt.Sprintf("line %d", n.StartPoint().Row+1)
				spots = append(spots, spot)
			}
			return true
		})
	}

	// MQL4-specific: OrderCloseBy (close opposite positions)
	if intent.Meta.MQLVersion == "mql4" {
		hasCloseBy := false
		walkCST(root, func(n *sitter.Node) bool {
			if n.Type() == "call_expression" && callFuncName(n) == "OrderCloseBy" {
				hasCloseBy = true
				return false
			}
			return true
		})
		if hasCloseBy {
			spots = append(spots, BlindSpot{
				ID:          "BS_OrderCloseBy",
				Category:    "order_close_by",
				Severity:    "信息",
				Description: "MQL4 OrderCloseBy (对冲平仓) 已转译为 closeAll 逻辑",
				Handling:    "检查生成的平仓逻辑是否覆盖原始对冲平仓条件",
			})
		}
	}

	return spots
}
