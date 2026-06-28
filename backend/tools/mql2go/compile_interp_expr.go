package mql2go

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"anttrader/tools/mql2go/interp"
)

// ── Expression compilation (CST → pure Go Expr) ─────────────────────

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
		return &interp.Expr{Kind: interp.ExprVar, Name: c.text(n)}

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

	case "argument_list":
		// Should not be compiled directly
		return nil
	}
	return nil
}

func (c *compiler) compileCall(n *sitter.Node) *interp.Expr {
	name := callFuncName(n)
	args := c.compileArgs(n)
	return &interp.Expr{Kind: interp.ExprCall, Name: name, Args: args}
}

func (c *compiler) compileBinary(n *sitter.Node) *interp.Expr {
	var op string
	var left, right *sitter.Node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "operator" || isBinaryOp(child.Type()) {
			op = c.text(child)
		} else if left == nil {
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
		Args: []interp.Expr{*c.compileExpr(left), *c.compileExpr(right)},
	}
}

func (c *compiler) compileUnary(n *sitter.Node) *interp.Expr {
	var op string
	var operand *sitter.Node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "operator" || isUnaryOp(c.text(child)) {
			op = c.text(child)
		} else {
			operand = child
		}
	}
	if operand == nil {
		return nil
	}
	return &interp.Expr{
		Kind: interp.ExprUnary,
		Op:   op,
		Args: []interp.Expr{*c.compileExpr(operand)},
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

	// Simple variable assignment: x = value
	name := c.findIdent(lhs)
	if name != "" {
		return &interp.Expr{
			Kind: interp.ExprAssignment,
			Name: name,
			Args: []interp.Expr{*c.compileExpr(rhs)},
		}
	}

	// Field assignment: obj.field = value → ExprField with IsAssign=true
	if lhs.Type() == "field_expression" {
		fieldExpr := c.compileField(lhs)
		if fieldExpr != nil {
			fieldExpr.IsAssign = true
			fieldExpr.Args = append(fieldExpr.Args, *c.compileExpr(rhs))
			return fieldExpr
		}
	}

	// Subscript assignment: arr[i] = value
	if lhs.Type() == "subscript_expression" {
		subExpr := c.compileSubscript(lhs)
		if subExpr != nil {
			subExpr.Args = []interp.Expr{*c.compileExpr(rhs)}
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
		case "field_identifier", "identifier":
			fieldName = c.text(child)
		case "call_expression":
			// method call: obj.method(args)
			fieldName = callFuncName(child)
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
		Args: []interp.Expr{*c.compileExpr(obj)},
	}
	result.Args = append(result.Args, args...)
	return result
}

func (c *compiler) compileArgs(n *sitter.Node) []interp.Expr {
	args := childByType(c.source, n, "argument_list")
	if args == nil {
		return nil
	}
	named := getNamedChildren(args)
	var result []interp.Expr
	for _, a := range named {
		if e := c.compileExpr(a); e != nil {
			result = append(result, *e)
		}
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

func (c *compiler) findType(n *sitter.Node) string {
	if pt := childByType(c.source, n, "primitive_type"); pt != nil {
		return c.text(pt)
	}
	if ti := childByType(c.source, n, "type_identifier"); ti != nil {
		return c.text(ti)
	}
	// MQL5 input: type might be in ERROR node
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		if child.Type() == "ERROR" {
			return c.text(child)
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
