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

// findOrderSendCST finds an OrderSend call in a subtree, recursing into if/else/compound.
func findOrderSendCST(n *sitter.Node) *sitter.Node {
	if n == nil {
		return nil
	}
	switch n.Type() {
	case "expression_statement":
		if call := childByType("", n, "call_expression"); call != nil {
			id := childByType("", call, "identifier")
			if id == nil {
				id = childByType("", call, "field_identifier")
			}
			if id != nil && nodeText("", id) == "OrderSend" {
				return call
			}
		}
	case "compound_statement":
		for i := 0; i < int(n.ChildCount()); i++ {
			if result := findOrderSendCST(n.Child(i)); result != nil {
				return result
			}
		}
	case "if_statement":
		// Check then-branch
		if result := findOrderSendCST(findNamedChild(n, "compound_statement")); result != nil {
			return result
		}
		// Walk all children for else branch
		for i := 0; i < int(n.ChildCount()); i++ {
			c := n.Child(i)
			if c.Type() == "else_clause" {
				if result := findOrderSendCST(c); result != nil {
					return result
				}
			}
		}
	case "else_clause":
		for i := 0; i < int(n.ChildCount()); i++ {
			if result := findOrderSendCST(n.Child(i)); result != nil {
				return result
			}
		}
	case "for_statement":
		body := childByType("", n, "compound_statement")
		if result := findOrderSendCST(body); result != nil {
			return result
		}
	}
	return nil
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
	id := childByType("", n, "identifier")
	if id == nil {
		id = childByType("", n, "field_identifier")
	}
	if id == nil {
		id = childByType("", n, "statement_identifier")
	}
	return nodeText("", id)
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
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "declaration" && n.Type() != "field_declaration" {
			return true
		}
		text := nodeText(source, n)
		if !strings.Contains(text, "extern ") && !strings.Contains(text, "input ") {
			return true
		}
		// Extract type, name, value
		var vt, name string
		var value string
		for i := 0; i < int(n.ChildCount()); i++ {
			c := n.Child(i)
			switch c.Type() {
			case "primitive_type", "type_identifier", "sized_type_specifier", "macro_type_specifier":
				vt = nodeText(source, c)
			case "identifier", "field_identifier", "statement_identifier":
				if name == "" {
					name = nodeText(source, c)
				}
			case "number_literal":
				value = nodeText(source, c)
			case "string_literal", "system_lib_string":
				value = nodeText(source, c)
				if len(value) >= 2 {
					value = value[1 : len(value)-1]
				}
			}
		}
		if name == "" || isNoiseCST(name, value) {
			return true
		}
		params = append(params, ParamSpec{
			Name:    name,
			Label:   name,
			Type:    mapMQLType(vt),
			Default: value,
			Group:   guessGroupCST(name),
		})
		return true
	})
	return params
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

func extractEntriesCST(root *sitter.Node) []EntryRule {
	var entries []EntryRule
	for _, fn := range findFunctions(root) {
		body := funcBody(fn)
		if body == nil {
			continue
		}
		scanForEntriesCST(body, &entries)
	}
	return entries
}

func scanForEntriesCST(body *sitter.Node, entries *[]EntryRule) {
	for i := 0; i < int(body.ChildCount()); i++ {
		c := body.Child(i)
		switch c.Type() {
		case "if_statement":
			if os := findOrderSendCST(c); os != nil {
				entry := entryFromOrderSend(os)
				if entry.Action != "" {
					*entries = append(*entries, entry)
				}
			}
		case "compound_statement":
			scanForEntriesCST(c, entries)
		}
	}
}

func entryFromOrderSend(os *sitter.Node) EntryRule {
	e := EntryRule{}
	// arg[1] is order type
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

// ── Exit recognition ────────────────────────────────────────────────

func extractExitsCST(root *sitter.Node) []ExitRule {
	var exits []ExitRule
	hasClose := false
	hasDelete := false
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() == "call_expression" {
			name := callFuncName(n)
			if name == "OrderClose" {
				hasClose = true
			}
			if name == "OrderDelete" {
				hasDelete = true
			}
		}
		return true
	})
	if hasClose {
		exits = append(exits, ExitRule{Trigger: TriggerMagic, Action: "position_close", MagicVal: "s.magic"})
	}
	if hasDelete {
		exits = append(exits, ExitRule{Trigger: TriggerDelete, Action: "order_delete", MagicVal: "s.magic"})
	}
	return exits
}

// ── Execution model ─────────────────────────────────────────────────

func detectExecCST(root *sitter.Node) ExecutionModel {
	for _, fn := range findFunctions(root) {
		body := funcBody(fn)
		if body == nil {
			continue
		}
		if hasGridFlagCST(body) {
			return ExecutionModel{Kind: ExecOnInitGrid}
		}
	}
	// Check for market orders → on_bar
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
	walkCST(root, func(n *sitter.Node) bool {
		if n.Type() != "declaration" && n.Type() != "field_declaration" {
			return true
		}
		text := nodeText(source, n)
		if strings.Contains(text, "extern ") || strings.Contains(text, "input ") {
			return true
		}
		// Extract name and type
		var name, vt string
		for i := 0; i < int(n.ChildCount()); i++ {
			c := n.Child(i)
			switch c.Type() {
			case "primitive_type", "type_identifier", "sized_type_specifier", "macro_type_specifier":
				vt = nodeText(source, c)
			case "identifier", "field_identifier", "statement_identifier":
				if name == "" {
					name = nodeText(source, c)
				}
			case "number_literal":
				if vt == "bool" {
					vt = "bool"
				}
			}
		}
		if name == "" {
			return true
		}
		state = append(state, StateVar{
			Name:   name,
			GoType: vt,
		})
		return true
	})
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
			}
		}
		// Try to find result variable
		spec.ResultVar = findResultVar(root, name)
		specs = append(specs, spec)
		return true
	})
	return specs
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
			if call := findChild("", c, "call_expression"); call != nil {
				if callFuncName(call) == indicatorName {
					id := childByType("", c, "identifier")
					if id == nil {
						id = childByType("", c, "field_identifier")
					}
					if id == nil {
						id = childByType("", c, "statement_identifier")
					}
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
	case "iCustom":
		return "i_custom"
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
