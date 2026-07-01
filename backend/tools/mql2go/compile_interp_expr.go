package mql2go

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"anttrader/tools/mql2go/interp"
)

// ── Expression compilation (CST → pure Go Expr) ─────────────────────

// mustExpr wraps compileExpr to guarantee a non-nil result.
// If compileExpr returns nil (unrecognized node type), returns a zero literal
// to avoid nil-pointer panics in callers that dereference immediately.
func (c *compiler) mustExpr(n *sitter.Node) interp.Expr {
	if e := c.compileExpr(n); e != nil {
		return *e
	}
	return interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(0)}
}

func (c *compiler) compileExprFromStmt(n *sitter.Node) *interp.Expr {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		// Skip semicolons and punctuation
		e := c.compileExpr(child)
		if e != nil {
			return e
		}
	}
	return nil
}

func (c *compiler) compileExpr(n *sitter.Node) *interp.Expr {
	if n == nil {
		return nil
	}
	switch n.Type() {
	case "number_literal":
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.ParseNumberLiteral(c.text(n))}

	case "string_literal":
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.StringVal(unquote(c.text(n)))}

	case "true":
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.BoolVal(true)}
	case "false":
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.BoolVal(false)}

	case "identifier":
		name := c.text(n)
		// MQL constants like PRICE_CLOSE, OP_BUY may parse as identifier
		if interp.IsMQLConstant(name) {
			return &interp.Expr{Kind: interp.ExprConst, Name: name}
		}
		return &interp.Expr{Kind: interp.ExprVar, Name: name}

	case "null":
		return &interp.Expr{Kind: interp.ExprConst, Name: "NULL"}

	case "type_identifier":
		// Predefined constant: OP_BUY, PRICE_CLOSE, etc.
		return &interp.Expr{Kind: interp.ExprConst, Name: c.text(n)}

	case "call_expression":
		return c.compileCall(n)

	case "binary_expression":
		return c.compileBinary(n)

	case "unary_expression":
		return c.compileUnary(n)

	case "subscript_expression":
		return c.compileSubscript(n)

	case "assignment_expression":
		return c.compileAssignment(n)

	case "update_expression":
		return c.compileUpdate(n)

	case "conditional_expression":
		return c.compileTernary(n)

	case "parenthesized_expression":
		for i := 0; i < int(n.NamedChildCount()); i++ {
			return c.compileExpr(n.NamedChild(i))
		}

	case "field_expression":
		return c.compileField(n)

	case "cast_expression":
		// (type)operand — unwrap to the operand, discarding the cast
		// (MQL is loosely typed, casts are mostly int↔double conversions)
		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			ct := child.Type()
			if ct != "primitive_type" && ct != "type_identifier" {
				return c.compileExpr(child)
			}
		}
		return nil

	case "comma_expression":
		// Evaluate left-to-right, return last value (C comma operator)
		var last *interp.Expr
		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			if e := c.compileExpr(child); e != nil {
				last = e
			}
		}
		return last

	case "argument_list":
		// Should not be compiled directly
		return nil
	}
	return nil
}

func (c *compiler) compileCall(n *sitter.Node) *interp.Expr {
	// Check if this is a method call: call_expression wrapping a field_expression
	var fieldNode *sitter.Node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "field_expression" {
			fieldNode = child
			break
		}
	}
	if fieldNode != nil {
		fieldExpr := c.compileField(fieldNode)
		if fieldExpr != nil {
			// Append method args from the call_expression's argument_list
			args := c.compileArgs(n)
			fieldExpr.Args = append(fieldExpr.Args, args...)
			return fieldExpr
		}
	}
	name := callFuncName(c.source, n)
	args := c.compileArgs(n)
	return &interp.Expr{Kind: interp.ExprCall, Name: name, Args: args}
}

func (c *compiler) compileBinary(n *sitter.Node) *interp.Expr {
	var op string
	var left, right *sitter.Node
	// Scan all children for the operator (operators are unnamed nodes in tree-sitter)
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		ct := child.Type()
		if ct == "operator" || isBinaryOp(ct) {
			op = ct
			if ct == "operator" {
				op = c.text(child)
			}
			continue
		}
	}
	// Scan named children for operands
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if left == nil {
			left = child
		} else {
			right = child
		}
	}
	if left == nil || right == nil {
		return nil
	}
	return &interp.Expr{
		Kind: interp.ExprBinary,
		Op:   op,
		Args: []interp.Expr{c.mustExpr(left), c.mustExpr(right)},
	}
}

func (c *compiler) compileUnary(n *sitter.Node) *interp.Expr {
	var op string
	var operand *sitter.Node
	// Scan all children for the operator
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		ct := child.Type()
		if ct == "operator" || isUnaryOp(ct) {
			op = ct
			if ct == "operator" {
				op = c.text(child)
			}
			continue
		}
	}
	// Scan named children for the operand
	for i := 0; i < int(n.NamedChildCount()); i++ {
		operand = n.NamedChild(i)
		break
	}
	if operand == nil {
		return nil
	}
	return &interp.Expr{
		Kind: interp.ExprUnary,
		Op:   op,
		Args: []interp.Expr{c.mustExpr(operand)},
	}
}

func (c *compiler) compileSubscript(n *sitter.Node) *interp.Expr {
	var name string
	var idx *sitter.Node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "identifier" && name == "" {
			name = c.text(child)
		} else if child.Type() != "[" && child.Type() != "]" {
			idx = child
		}
	}
	if name == "" || idx == nil {
		return nil
	}
	return &interp.Expr{
		Kind:  interp.ExprSubscript,
		Name:  name,
		Index: c.compileExpr(idx),
	}
}

func (c *compiler) compileAssignment(n *sitter.Node) *interp.Expr {
	children := getNamedChildren(n)
	if len(children) < 2 {
		return nil
	}
	lhs := children[0]
	rhs := children[1]

	// Detect compound assignment operator by scanning non-named children
	op := "="
	for i := 0; i < int(n.ChildCount()); i++ {
		ch := n.Child(i)
		t := ch.Type()
		if t == "+=" || t == "-=" || t == "*=" || t == "/=" || t == "%=" {
			op = t
			break
		}
	}

	// Simple variable assignment: x = value (or x += value)
	name := c.findIdent(lhs)
	if name != "" {
		if op == "=" {
			return &interp.Expr{
				Kind: interp.ExprAssignment,
				Name: name,
				Args: []interp.Expr{c.mustExpr(rhs)},
			}
		}
		return &interp.Expr{
			Kind: interp.ExprCompoundAssign,
			Name: name,
			Op:   op,
			Args: []interp.Expr{c.mustExpr(rhs)},
		}
	}

	// Field assignment: obj.field = value → ExprField with IsAssign=true
	if lhs.Type() == "field_expression" {
		fieldExpr := c.compileField(lhs)
		if fieldExpr != nil {
			fieldExpr.IsAssign = true
			fieldExpr.Args = append(fieldExpr.Args, c.mustExpr(rhs))
			return fieldExpr
		}
	}

	// Subscript assignment: arr[i] = value
	if lhs.Type() == "subscript_expression" {
		subExpr := c.compileSubscript(lhs)
		if subExpr != nil {
			subExpr.Args = []interp.Expr{c.mustExpr(rhs)}
			return subExpr
		}
	}

	return nil
}

func (c *compiler) compileUpdate(n *sitter.Node) *interp.Expr {
	text := c.text(n)
	name := strings.TrimSuffix(strings.TrimSuffix(text, "++"), "--")
	op := "++"
	if strings.HasSuffix(text, "--") {
		op = "--"
	}
	return &interp.Expr{Kind: interp.ExprUpdate, Name: name, Op: op}
}

func (c *compiler) compileTernary(n *sitter.Node) *interp.Expr {
	var cond, thenE, elseE *sitter.Node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "?" || child.Type() == ":" {
			continue
		}
		if cond == nil {
			cond = child
		} else if thenE == nil {
			thenE = child
		} else {
			elseE = child
		}
	}
	if cond == nil || thenE == nil || elseE == nil {
		return nil
	}
	return &interp.Expr{
		Kind:     interp.ExprTernary,
		Cond:     c.compileExpr(cond),
		ThenExpr: c.compileExpr(thenE),
		ElseExpr: c.compileExpr(elseE),
	}
}

func (c *compiler) compileField(n *sitter.Node) *interp.Expr {
	// field_expression: obj.field or obj.method(args)
	var obj *sitter.Node
	var fieldName string
	var args []interp.Expr
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "field_identifier":
			fieldName = c.text(child)
		case "identifier":
			if obj == nil {
				obj = child
			}
		case "call_expression":
			// method call: obj.method(args)
			fieldName = callFuncName(c.source, child)
			args = c.compileArgs(child)
		case "argument_list":
			// already handled by call_expression above
		default:
			if obj == nil {
				obj = child
			}
		}
	}
	if obj == nil || fieldName == "" {
		return nil
	}
	result := &interp.Expr{
		Kind: interp.ExprField,
		Name: fieldName,
		Args: []interp.Expr{c.mustExpr(obj)},
	}
	result.Args = append(result.Args, args...)
	return result
}

func (c *compiler) compileArgs(n *sitter.Node) []interp.Expr {
	args := childByType(n, "argument_list")
	if args == nil {
		return nil
	}
	named := getNamedChildren(args)
	var result []interp.Expr
	for _, a := range named {
		e := c.compileExpr(a)
		if e == nil {
			// Unhandled node type — push zero literal to preserve argument count.
			// Without this, a nil return silently drops the argument, causing
			// all subsequent arguments to shift positions (the NULL bug).
			e = &interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(0)}
		}
		result = append(result, *e)
	}
	return result
}

// ── Compiler helpers ────────────────────────────────────────────────

func (c *compiler) text(n *sitter.Node) string {
	return nodeText(c.source, n)
}

func (c *compiler) findIdent(n *sitter.Node) string {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "identifier" || child.Type() == "field_identifier" {
			return c.text(child)
		}
	}
	// Direct identifier
	if n.Type() == "identifier" || n.Type() == "field_identifier" {
		return c.text(n)
	}
	return ""
}

// findArraySize detects array dimension in declarations like double Gd_720[30].
// Returns (size, true) if an array dimension is found, (0, false) otherwise.
func (c *compiler) findArraySize(n *sitter.Node) (int, bool) {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "subscript_expression" {
			// The child of subscript_expression is the array name (identifier)
			// and the index is the dimension size
			for j := 0; j < int(child.NamedChildCount()); j++ {
				dim := child.NamedChild(j)
				if dim.Type() == "number_literal" {
					txt := c.text(dim)
					var size int
					fmt.Sscanf(txt, "%d", &size)
					if size > 0 {
						return size, true
					}
				}
			}
			return 0, true // array but unknown size
		}
	}
	return 0, false
}

func (c *compiler) findType(n *sitter.Node) string {
	// First check ERROR nodes for primitive types (MQL5 'input int' pattern)
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		if child.Type() == "ERROR" {
			txt := c.text(child)
			if isMQLPrimitiveType(txt) {
				return txt
			}
		}
	}
	if pt := childByType(n, "primitive_type"); pt != nil {
		return c.text(pt)
	}
	// Find type_identifier, skipping 'input'/'extern' keywords
	// (tree-sitter parses 'input BuyOrSell0 x' with 'input' as type_identifier)
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "type_identifier" {
			name := c.text(child)
			if name != "input" && name != "extern" {
				return name
			}
		}
	}
	return ""
}

func (c *compiler) findExprChild(n *sitter.Node) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "number_literal", "string_literal", "identifier",
			"call_expression", "binary_expression", "unary_expression",
			"subscript_expression", "conditional_expression",
			"parenthesized_expression", "field_expression",
			"assignment_expression", "true", "false":
			return child
		}
	}
	return nil
}

func (c *compiler) findInitValue(n *sitter.Node, declName string) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "number_literal", "string_literal",
			"call_expression", "binary_expression", "unary_expression",
			"subscript_expression", "conditional_expression",
			"parenthesized_expression", "field_expression",
			"assignment_expression", "true", "false":
			return child
		case "identifier":
			if c.text(child) != declName {
				return child
			}
		}
	}
	return nil
}

func isBinaryOp(t string) bool {
	switch t {
	case "+", "-", "*", "/", "%", "==", "!=", "<", ">", "<=", ">=",
		"&&", "||", "&", "|", "^", "<<", ">>":
		return true
	}
	return false
}

func isUnaryOp(s string) bool {
	switch s {
	case "-", "!", "~", "+":
		return true
	}
	return false
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
