package mql2go

import (
	"fmt"

	"alphaforge/tools/mql2go/interp"

	sitter "github.com/smacker/go-tree-sitter"
)

// ── Expression compilation (CST → pure Go Expr) ─────────────────────

// mustExpr wraps compileExpr to guarantee a non-nil result while preserving
// the first compilation error for the caller.
func (c *compiler) mustExpr(n *sitter.Node) interp.Expr {
	if e := c.compileExpr(n); e != nil {
		return *e
	}
	if c.err == nil {
		if n == nil {
			c.err = fmt.Errorf("missing MQL expression")
		} else {
			c.err = fmt.Errorf("unsupported MQL expression node: %s", n.Type())
		}
	}
	return interp.Expr{Kind: interp.ExprLiteral, Val: interp.NoneVal()}
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

func isMQLExpressionNodeType(nodeType string) bool {
	switch nodeType {
	case nodeNumberLiteral, "string_literal", nodeTrue, nodeFalse, nodeIdentifier,
		nodeTypeIdentifier, "null", "call_expression", "binary_expression",
		"unary_expression", "subscript_expression", "assignment_expression",
		"update_expression", "conditional_expression", nodeParenExpr,
		"field_expression", "cast_expression", "comma_expression":
		return true
	default:
		return false
	}
}

func (c *compiler) compileExpr(n *sitter.Node) *interp.Expr {
	if n == nil {
		return nil
	}
	switch n.Type() {
	case nodeNumberLiteral:
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.ParseNumberLiteral(c.text(n))}

	case "string_literal":
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.StringVal(unquote(c.text(n)))}

	case nodeTrue:
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.BoolVal(true)}
	case nodeFalse:
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.BoolVal(false)}

	case nodeIdentifier:
		name := c.text(n)
		// MQL constants like PRICE_CLOSE, OP_BUY may parse as identifier
		if interp.IsMQLConstant(name) {
			return &interp.Expr{Kind: interp.ExprConst, Name: name}
		}
		return &interp.Expr{Kind: interp.ExprVar, Name: name}

	case "null":
		return &interp.Expr{Kind: interp.ExprConst, Name: "NULL"}

	case nodeTypeIdentifier:
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

	case nodeParenExpr:
		if n.NamedChildCount() > 0 {
			return c.compileExpr(n.NamedChild(0))
		}
		if c.err == nil {
			c.err = fmt.Errorf("parenthesized expression has no operand")
		}
		return nil

	case "field_expression":
		return c.compileField(n)

	case "cast_expression":
		// (type)operand — unwrap to the operand, discarding the cast
		// (MQL is loosely typed, casts are mostly int↔double conversions)
		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			ct := child.Type()
			if ct != "primitive_type" && ct != nodeTypeIdentifier && ct != "type_descriptor" {
				return c.compileExpr(child)
			}
		}
		if c.err == nil {
			c.err = fmt.Errorf("cast expression has no operand")
		}
		return nil
	case "comma_expression":
		// VM-COMPILER-SEMANTICS-4: evaluate all children left-to-right,
		// return last value (C comma operator). Must preserve side effects
		// of all sub-expressions (assignments, function calls), not just
		// the last one. Generate ExprSeq so the compiler emits all sub-expression
		// evaluations in order.
		var args []interp.Expr
		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			if e := c.compileExpr(child); e != nil {
				args = append(args, *e)
			}
		}
		if len(args) == 0 {
			return nil
		}
		if len(args) == 1 {
			return &args[0]
		}
		return &interp.Expr{Kind: interp.ExprSeq, Args: args}

	case "argument_list":
		// Should not be compiled directly
		return nil
	default:
		if c.err == nil {
			c.err = fmt.Errorf("unsupported MQL expression node: %s", n.Type())
		}
		return nil
	}
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
	if name == "" {
		if c.err == nil {
			c.err = fmt.Errorf("call expression has no function name")
		}
		return nil
	}
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
		if c.err == nil {
			c.err = fmt.Errorf("binary expression has missing operand")
		}
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
	if n.NamedChildCount() > 0 {
		operand = n.NamedChild(0)
	}
	if operand == nil {
		if c.err == nil {
			c.err = fmt.Errorf("unary expression has missing operand")
		}
		return nil
	}
	return &interp.Expr{
		Kind: interp.ExprUnary,
		Op:   op,
		Args: []interp.Expr{c.mustExpr(operand)},
	}
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
