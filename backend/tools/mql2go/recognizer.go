package mql2go

import (
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
	if source == "" {
		source = parseSource
	}
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
		} else {
			// MQL5 `input` declarations: tree-sitter parses `input` as type_identifier
			// and the actual type (int/double/etc.) as an ERROR node.
			for j := 0; j < int(n.ChildCount()); j++ {
				c := n.Child(j)
				ct := c.Type()
				cText := nodeText(source, c)
				if ct == "ERROR" || (ct == "type_identifier" && cText != "input") {
					if isMQLPrimitiveType(cText) {
						vt = cText
						break
					}
				}
			}
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

func isMQLPrimitiveType(t string) bool {
	switch t {
	case "int", "long", "uint", "ulong", "double", "float", "string", "bool", "char", "short", "uchar", "ushort":
		return true
	}
	return false
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
