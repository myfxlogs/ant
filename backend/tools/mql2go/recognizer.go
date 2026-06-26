package mql2go

import "strings"

// Recognizer extracts trading intent from a parsed MQL AST.
type Recognizer struct {
	globalVars map[string]bool // names of global variables
	externs    map[string]bool
}

// NewRecognizer creates a Recognizer from a SourceFile.
func NewRecognizer(ast *SourceFile) *Recognizer {
	r := &Recognizer{
		globalVars: make(map[string]bool),
		externs:    make(map[string]bool),
	}
	if ast != nil {
		for _, decl := range ast.Declarations {
			if vd, ok := decl.(*VarDecl); ok {
				if vd.IsExtern || vd.IsInput {
					r.externs[vd.Name] = true
				} else {
					r.globalVars[vd.Name] = true
				}
			}
		}
	}
	return r
}

// ── Parameters ─────────────────────────────────────────────────────

func (r *Recognizer) ExtractParams(ast *SourceFile) []ParamSpec {
	var params []ParamSpec
	if ast == nil {
		return params
	}
	for _, decl := range ast.Declarations {
		vd, ok := decl.(*VarDecl)
		if !ok || (!vd.IsExtern && !vd.IsInput) {
			continue
		}
		if isNoiseParam(vd.Name, vd.Value) {
			continue
		}
		params = append(params, ParamSpec{
			Name:    vd.Name,
			Label:   vd.Name,
			Type:    paramType(vd.VarType),
			Default: extractDefault(vd.Value),
			Group:   guessGroup(vd.Name),
		})
	}
	return params
}

func (r *Recognizer) ExtractState(ast *SourceFile) []StateVar {
	var state []StateVar
	if ast == nil {
		return state
	}
	for _, decl := range ast.Declarations {
		vd, ok := decl.(*VarDecl)
		if !ok || vd.IsExtern || vd.IsInput {
			continue
		}
		state = append(state, StateVar{
			Name:    vd.Name,
			GoType:  vd.VarType,
			Initial: extractDefault(vd.Value),
		})
	}
	return state
}

// ── Entry recognition ──────────────────────────────────────────────

func (r *Recognizer) ExtractEntries(ast *SourceFile) []EntryRule {
	var entries []EntryRule
	r.walkFunctions(ast, func(fn *FuncDef) {
		if fn.Body == nil {
			return
		}
		// Extract local variables
		locals := extractLocalVars(fn.Body)
		_ = locals
		// Scan for if→OrderSend patterns
		r.scanForEntries(fn.Body, &entries)
	})
	return entries
}

func (r *Recognizer) scanForEntries(body Node, entries *[]EntryRule) {
	cs, ok := body.(*CompoundStmt)
	if !ok {
		return
	}
	for _, stmt := range cs.Statements {
		if ifStmt, ok := stmt.(*IfStmt); ok {
			r.matchIfEntry(ifStmt, entries)
		}
		if cs2, ok := stmt.(*CompoundStmt); ok {
			r.scanForEntries(cs2, entries)
		}
	}
}

func (r *Recognizer) matchIfEntry(ifStmt *IfStmt, entries *[]EntryRule) {
	os := findOrderSend(ifStmt.ThenBranch)
	if os == nil {
		return
	}
	action := orderSendAction(os)
	if action == "" {
		return
	}
	entry := EntryRule{
		Action: action,
		Volume: orderSendArg(os, 2),
		Price:  orderSendArg(os, 3),
		Magic:  orderSendArg(os, 8),
	}
	if c := orderSendArg(os, 5); c != "" && c != "0" {
		entry.StopLoss = c
	}
	if c := orderSendArg(os, 6); c != "" && c != "0" {
		entry.TakeProfit = c
	}
	if c := orderSendArg(os, 7); c != "" {
		entry.Comment = c
	}
	*entries = append(*entries, entry)
}

// ── Exit recognition ───────────────────────────────────────────────

func (r *Recognizer) ExtractExits(ast *SourceFile) []ExitRule {
	var exits []ExitRule
	r.walkFunctions(ast, func(fn *FuncDef) {
		if fn.Body == nil {
			return
		}
		if hasCall(fn.Body, "OrderClose") {
			exits = append(exits, ExitRule{
				Trigger:  TriggerMagic,
				Action:   "position_close",
				MagicVal: "s.magic",
			})
		}
		if hasCall(fn.Body, "OrderDelete") {
			exits = append(exits, ExitRule{
				Trigger:  TriggerDelete,
				Action:   "order_delete",
				MagicVal: "s.magic",
			})
		}
	})
	return exits
}

// ── Execution model ────────────────────────────────────────────────

func (r *Recognizer) DetectExecution(ast *SourceFile) ExecutionModel {
	hasGrid := false
	r.walkFunctions(ast, func(fn *FuncDef) {
		if fn.Body != nil && hasCall(fn.Body, "OrderSend") {
			// Check for gridPlaced flag pattern
			if fn.Body != nil {
				r.walkNodes(fn.Body, func(n Node) bool {
					if id, ok := n.(*Identifier); ok {
						if id.Name == "gridPlaced" {
							hasGrid = true
							return false
						}
					}
					return true
				})
			}
		}
	})
	if hasGrid {
		return ExecutionModel{Kind: ExecOnInitGrid}
	}
	return ExecutionModel{Kind: ExecOnBar}
}

// ── Sizing ─────────────────────────────────────────────────────────

func (r *Recognizer) DetectSizing(intent *StrategyIntent) *SizingRule {
	// Check for martingale pattern
	for _, entry := range intent.Entry {
		if strings.Contains(entry.Volume, "MathPow") || strings.Contains(entry.Volume, "*") {
			return &SizingRule{
				Kind:       SizingMartingale,
				Expression: "s.baseLot",
			}
		}
	}
	return &SizingRule{
		Kind:       SizingFixed,
		Expression: "s.lotSize",
	}
}

// ── Timer ──────────────────────────────────────────────────────────

func (r *Recognizer) DetectTimer(ast *SourceFile) *TimerRule {
	var timer *TimerRule
	r.walkFunctions(ast, func(fn *FuncDef) {
		if fn.Body != nil && hasCall(fn.Body, "EventSetTimer") {
			// Extract timer interval
			r.walkNodes(fn.Body, func(n Node) bool {
				if call, ok := n.(*CallExpr); ok && call.Name == "EventSetTimer" {
					if len(call.Args) > 0 {
						if nl, ok := call.Args[0].(*NumberLiteral); ok {
							timer = &TimerRule{IntervalSeconds: parseInt(nl.Value)}
						}
					}
					return false
				}
				return true
			})
		}
	})
	if timer == nil {
		// Check for EventSetMillisecondTimer
		r.walkFunctions(ast, func(fn *FuncDef) {
			if fn.Body != nil && hasCall(fn.Body, "EventSetMillisecondTimer") {
				timer = &TimerRule{IntervalSeconds: 5} // default
			}
		})
	}
	return timer
}

// ── AST traversal helpers ──────────────────────────────────────────

func (r *Recognizer) walkFunctions(ast *SourceFile, fn func(*FuncDef)) {
	if ast == nil {
		return
	}
	for _, decl := range ast.Declarations {
		if fd, ok := decl.(*FuncDef); ok {
			fn(fd)
		}
	}
}

func (r *Recognizer) walkNodes(node Node, visit func(Node) bool) {
	if node == nil || !visit(node) {
		return
	}
	switch n := node.(type) {
	case *CompoundStmt:
		for _, s := range n.Statements {
			r.walkNodes(s, visit)
		}
	case *IfStmt:
		r.walkNodes(n.Condition, visit)
		r.walkNodes(n.ThenBranch, visit)
		r.walkNodes(n.ElseBranch, visit)
	case *ForStmt:
		r.walkNodes(n.Body, visit)
	case *ExpressionStmt:
		r.walkNodes(n.Expr, visit)
	case *BinaryOp:
		r.walkNodes(n.Left, visit)
		r.walkNodes(n.Right, visit)
	case *CallExpr:
		for _, a := range n.Args {
			r.walkNodes(a, visit)
		}
	}
}

func hasCall(node Node, name string) bool {
	found := false
	r := &Recognizer{}
	r.walkNodes(node, func(n Node) bool {
		if call, ok := n.(*CallExpr); ok && call.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

func findOrderSend(node Node) *CallExpr {
	if node == nil {
		return nil
	}
	if es, ok := node.(*ExpressionStmt); ok {
		if call, ok := es.Expr.(*CallExpr); ok && call.Name == "OrderSend" {
			return call
		}
	}
	if cs, ok := node.(*CompoundStmt); ok {
		for _, s := range cs.Statements {
			if result := findOrderSend(s); result != nil {
				return result
			}
		}
	}
	if ifStmt, ok := node.(*IfStmt); ok {
		// Check then-branch
		if result := findOrderSend(ifStmt.ThenBranch); result != nil {
			return result
		}
		// Check else-branch
		if result := findOrderSend(ifStmt.ElseBranch); result != nil {
			return result
		}
	}
	return nil
}

func orderSendAction(call *CallExpr) OrderAction {
	if len(call.Args) < 2 {
		return ""
	}
	id, ok := call.Args[1].(*Identifier)
	if !ok {
		return ""
	}
	switch id.Name {
	case "OP_BUY":
		return ActionMarketBuy
	case "OP_SELL":
		return ActionMarketSell
	case "OP_BUYLIMIT":
		return ActionBuyLimit
	case "OP_SELLLIMIT":
		return ActionSellLimit
	case "OP_BUYSTOP":
		return ActionBuyStop
	case "OP_SELLSTOP":
		return ActionSellStop
	}
	return ""
}

func orderSendArg(call *CallExpr, idx int) string {
	if idx >= len(call.Args) {
		return ""
	}
	return nodeToString(call.Args[idx])
}

func nodeToString(n Node) string {
	switch v := n.(type) {
	case *Identifier:
		return v.Name
	case *NumberLiteral:
		return v.Value
	case *StringLiteral:
		return `"` + v.Value + `"`
	case *CallExpr:
		return v.Name + "(...)"
	case *BinaryOp:
		return nodeToString(v.Left) + " " + v.Op + " " + nodeToString(v.Right)
	default:
		return ""
	}
}

func extractLocalVars(body *CompoundStmt) map[string]bool {
	locals := make(map[string]bool)
	r := &Recognizer{}
	r.walkNodes(body, func(n Node) bool {
		if vd, ok := n.(*VarDecl); ok {
			locals[vd.Name] = true
		}
		return true
	})
	return locals
}

func extractDefault(value Node) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case *NumberLiteral:
		return v.Value
	case *StringLiteral:
		return v.Value
	case *Identifier:
		switch v.Name {
		case "true":
			return "true"
		case "false":
			return "false"
		default:
			return v.Name
		}
	}
	return ""
}

func isNoiseParam(name string, value Node) bool {
	if strings.Contains(name, "说明") || strings.Contains(name, "选择") ||
		strings.Contains(name, "提示") {
		return true
	}
	if sv, ok := value.(*StringLiteral); ok {
		val := sv.Value
		if strings.Contains(val, "====") {
			return true
		}
		if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
			return true
		}
		if len(val) > 20 && containsNonASCII(val) {
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

func paramType(mqlType string) ParamType {
	switch mqlType {
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

func guessGroup(name string) ParamGroup {
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
	if strings.Contains(lower, "risk") || strings.Contains(lower, "margin") {
		return GroupRisk
	}
	return GroupEntry
}

func containsNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}
