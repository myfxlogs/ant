package mql2go

import (
	"strings"

	"alphaforge/tools/mql2go/interp"

	sitter "github.com/smacker/go-tree-sitter"
)

// This file contains method-call handling and operator compilation
// extracted from compile_py_expr.go for file-size compliance.

func (c *pyCompiler) compilePyMethodCall(callNode, attrNode *sitter.Node) *interp.Expr {
	fullPath := c.text(attrNode)
	args := c.compileArgsOrdered(callNode, fullPath)
	parts := strings.Split(fullPath, ".")
	methodName := parts[len(parts)-1]
	if len(parts) == 2 && parts[0] == "self" {
		return &interp.Expr{Kind: interp.ExprCall, Name: methodName, Args: args}
	}
	mappedName := mapPythonMethod(methodName, fullPath)
	if mappedName != "" {
		return &interp.Expr{Kind: interp.ExprCall, Name: mappedName, Args: args}
	}
	cleanPath := c.extractAttrChain(attrNode)
	if cleanPath != fullPath && cleanPath != "" {
		mappedName = mapPythonMethod(methodName, cleanPath)
		if mappedName != "" {
			if isIBarFunc(mappedName) {
				if expr := c.buildChainedBarCall(mappedName, cleanPath, attrNode, args); expr != nil {
					return expr
				}
			}
			return &interp.Expr{Kind: interp.ExprCall, Name: mappedName, Args: args}
		}
	}
	return &interp.Expr{Kind: interp.ExprCall, Name: fullPath, Args: args}
}

func isIBarFunc(name string) bool {
	return strings.HasPrefix(name, "iClose") ||
		strings.HasPrefix(name, "iOpen") ||
		strings.HasPrefix(name, "iHigh") ||
		strings.HasPrefix(name, "iLow") ||
		strings.HasPrefix(name, "iVolume") ||
		strings.HasPrefix(name, "iTime")
}

func (c *pyCompiler) buildChainedBarCall(mappedName, cleanPath string, attrNode *sitter.Node, args []interp.Expr) *interp.Expr {
	if strings.Contains(cleanPath, "bars_for_symbol.") {
		innerArgs := c.extractInnerCallArgs(attrNode)
		if len(innerArgs) > 0 {
			combined := make([]interp.Expr, 0, len(innerArgs)+1+len(args))
			combined = append(combined, innerArgs...)
			combined = append(combined, interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(0)})
			combined = append(combined, args...)
			return &interp.Expr{Kind: interp.ExprCall, Name: mappedName, Args: combined}
		}
	}
	if strings.Contains(cleanPath, "bars_tf.") {
		innerArgs := c.extractInnerCallArgs(attrNode)
		if len(innerArgs) > 0 {
			tfInt := int32(0)
			if innerArgs[0].Kind == interp.ExprLiteral && innerArgs[0].Val.Kind == interp.ValString {
				tfInt = tfStringToInt(innerArgs[0].Val.Str)
			}
			combined := make([]interp.Expr, 0, 2+len(args))
			combined = append(combined, interp.Expr{Kind: interp.ExprLiteral, Val: interp.StringVal("")})
			combined = append(combined, interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(tfInt)})
			combined = append(combined, args...)
			return &interp.Expr{Kind: interp.ExprCall, Name: mappedName, Args: combined}
		}
	}
	return nil
}

// extractInnerCallArgs extracts the arguments from the inner call of a chained expression.
// For ctx.bars_for_symbol("EURUSD").close(0), it returns ["EURUSD"] from the inner call.
func (c *pyCompiler) extractInnerCallArgs(attrNode *sitter.Node) []interp.Expr {
	if attrNode == nil || attrNode.Type() != nodeAttribute {
		return nil
	}
	obj := attrNode.NamedChild(0)
	if obj == nil || obj.Type() != nodeCall {
		return nil
	}
	return c.compileArgs(obj)
}

// extractAttrChain builds a clean "obj.method.field" chain from an attribute node,
// stripping out call argument text. e.g. ctx.bars_for_symbol("EURUSD").close → ctx.bars_for_symbol.close
func (c *pyCompiler) extractAttrChain(n *sitter.Node) string {
	if n == nil {
		return ""
	}
	if n.Type() == nodeIdentifier {
		return c.text(n)
	}
	if n.Type() == nodeAttribute {
		obj := n.NamedChild(0)
		field := n.NamedChild(1)
		if obj == nil || field == nil {
			return c.text(n)
		}
		objPath := c.extractAttrChain(obj)
		fieldName := c.text(field)
		if objPath != "" {
			return objPath + "." + fieldName
		}
		return fieldName
	}
	if n.Type() == nodeCall {
		// Strip arguments — just return the chain of the function
		fn := n.NamedChild(0)
		if fn == nil {
			return c.text(n)
		}
		return c.extractAttrChain(fn)
	}
	return c.text(n)
}

func (c *pyCompiler) compilePyBinary(n *sitter.Node) *interp.Expr {
	var op string
	var left, right *sitter.Node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if left == nil {
			left = child
		} else {
			right = child
		}
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		ct := child.Type()
		if ct == "+" || ct == "-" || ct == "*" || ct == "/" || ct == "%" ||
			ct == "//" || ct == "**" || ct == "&" || ct == "|" || ct == "^" ||
			ct == "<<" || ct == ">>" {
			op = ct
			break
		}
	}
	if left == nil || right == nil {
		return nil
	}
	if op == "" {
		c.errorf(n, "binary operator not found in expression")
		return nil
	}
	switch op {
	case "**":
		return &interp.Expr{
			Kind: interp.ExprCall,
			Name: "MathPow",
			Args: []interp.Expr{*c.mustPyExpr(left), *c.mustPyExpr(right)},
		}
	case "&", "|", "^", "<<", ">>":
		c.errorf(n, "bitwise operator '%s' is not supported in Python subset (no VM opcode)", op)
		return nil
	}
	return &interp.Expr{
		Kind: interp.ExprBinary,
		Op:   op,
		Args: []interp.Expr{*c.mustPyExpr(left), *c.mustPyExpr(right)},
	}
}

func (c *pyCompiler) compilePyBoolean(n *sitter.Node) *interp.Expr {
	var op string
	var left, right *sitter.Node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if left == nil {
			left = child
		} else {
			right = child
		}
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		ct := child.Type()
		if ct == "and" || ct == "or" {
			op = ct
			break
		}
	}
	switch op {
	case "and":
		op = "&&"
	case "or":
		op = "||"
	}
	if op == "" || left == nil || right == nil {
		return nil
	}
	return &interp.Expr{
		Kind: interp.ExprBinary,
		Op:   op,
		Args: []interp.Expr{*c.mustPyExpr(left), *c.mustPyExpr(right)},
	}
}

func (c *pyCompiler) compilePyTernary(n *sitter.Node) *interp.Expr {
	var thenE, cond, elseE *sitter.Node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if thenE == nil {
			thenE = child
		} else if cond == nil {
			cond = child
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

func (c *pyCompiler) compilePyUnary(n *sitter.Node) *interp.Expr {
	var op string
	var operand *sitter.Node
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		ct := child.Type()
		if ct == "-" || ct == "+" || ct == "~" || ct == "not" {
			op = ct
			continue
		}
		if child.IsNamed() && operand == nil {
			operand = child
		}
	}
	if operand == nil {
		return nil
	}
	if op == "not" {
		op = "!"
	}
	if op == "~" {
		c.errorf(n, "bitwise NOT '~' is not supported in Python subset (no VM opcode)")
		return nil
	}
	return &interp.Expr{
		Kind: interp.ExprUnary,
		Op:   op,
		Args: []interp.Expr{*c.mustPyExpr(operand)},
	}
}
