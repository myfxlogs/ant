package mql2go

import (
	sitter "github.com/smacker/go-tree-sitter"
	"alphaforge/tools/mql2go/interp"
)

func (c *pyCompiler) compilePyAssignment(n *sitter.Node) *interp.Expr {
	children := getNamedChildren(n)
	if len(children) < 2 {
		return nil
	}
	lhs := children[0]
	// For typed assignment (c1: float = expr), children = [identifier, type, expr]
	// Skip the type node to find the actual RHS
	rhs := children[1]
	if rhs.Type() == "type" {
		if len(children) < 3 {
			return nil
		}
		rhs = children[2]
	}
	// self.field = value → variable assignment (check before findIdentPy,
	// which returns "self" for attribute nodes)
	if sfn := c.selfFieldName(lhs); sfn != "" {
		return &interp.Expr{
			Kind: interp.ExprAssignment,
			Name: sfn,
			Args: []interp.Expr{*c.mustPyExpr(rhs)},
		}
	}
	name := c.findIdentPy(lhs)
	if name != "" {
		return &interp.Expr{
			Kind: interp.ExprAssignment,
			Name: name,
			Args: []interp.Expr{*c.mustPyExpr(rhs)},
		}
	}
	if lhs.Type() == "subscript" {
		subExpr := c.compilePySubscript(lhs)
		if subExpr != nil {
			subExpr.Args = []interp.Expr{*c.mustPyExpr(rhs)}
			return subExpr
		}
	}
	if lhs.Type() == nodeAttribute {
		// Check if this is a position field write (pos.sl = x) — not allowed
		objNode := lhs.NamedChild(0)
		if objNode != nil && objNode.Type() == nodeIdentifier {
			objName := c.text(objNode)
			if c.posLoopVars != nil && c.posLoopVars[objName] {
				fieldName := ""
				if fieldNode := lhs.NamedChild(1); fieldNode != nil && fieldNode.Type() == nodeIdentifier {
					fieldName = c.text(fieldNode)
				}
				c.errorf(lhs, "cannot assign to position field '%s.%s' — use ctx.broker.modify() instead",
					objName, fieldName)
				return nil
			}
		}
		attrExpr := c.compilePyAttribute(lhs)
		if attrExpr != nil {
			attrExpr.IsAssign = true
			attrExpr.Args = append(attrExpr.Args, *c.mustPyExpr(rhs))
			return attrExpr
		}
	}
	return nil
}

func (c *pyCompiler) compilePyAugmentedAssign(n *sitter.Node) *interp.Expr {
	children := getNamedChildren(n)
	if len(children) < 2 {
		return nil
	}
	lhs := children[0]
	rhs := children[1]
	op := "+="
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		ct := child.Type()
		if ct == "+=" || ct == "-=" || ct == "*=" || ct == "/=" || ct == "%=" ||
			ct == "//=" || ct == "**=" || ct == "&=" || ct == "|=" || ct == "^=" {
			op = ct
			break
		}
	}
	switch op {
	case "**=":
		// **= has no opcode — desugar to x = MathPow(x, rhs) like ** → MathPow
		if sfn := c.selfFieldName(lhs); sfn != "" {
			return &interp.Expr{
				Kind: interp.ExprAssignment,
				Name: sfn,
				Args: []interp.Expr{{
					Kind: interp.ExprCall,
					Name: "MathPow",
					Args: []interp.Expr{
						{Kind: interp.ExprVar, Name: sfn},
						*c.mustPyExpr(rhs),
					},
				}},
			}
		}
		name := c.findIdentPy(lhs)
		if name != "" {
			return &interp.Expr{
				Kind: interp.ExprAssignment,
				Name: name,
				Args: []interp.Expr{{
					Kind: interp.ExprCall,
					Name: "MathPow",
					Args: []interp.Expr{
						{Kind: interp.ExprVar, Name: name},
						*c.mustPyExpr(rhs),
					},
				}},
			}
		}
		return nil
	}
	// self.field += value → field += value (check before findIdentPy)
	if sfn := c.selfFieldName(lhs); sfn != "" {
		return &interp.Expr{
			Kind: interp.ExprCompoundAssign,
			Name: sfn,
			Op:   op,
			Args: []interp.Expr{*c.mustPyExpr(rhs)},
		}
	}
	name := c.findIdentPy(lhs)
	if name != "" {
		return &interp.Expr{
			Kind: interp.ExprCompoundAssign,
			Name: name,
			Op:   op,
			Args: []interp.Expr{*c.mustPyExpr(rhs)},
		}
	}
	if lhs.Type() == nodeAttribute {
		objNode := lhs.NamedChild(0)
		if objNode != nil && objNode.Type() == nodeIdentifier {
			objName := c.text(objNode)
			if c.posLoopVars != nil && c.posLoopVars[objName] {
				fieldName := ""
				if fieldNode := lhs.NamedChild(1); fieldNode != nil && fieldNode.Type() == nodeIdentifier {
					fieldName = c.text(fieldNode)
				}
				c.errorf(lhs, "cannot assign to position field '%s.%s' — use ctx.broker.modify() instead",
					objName, fieldName)
				return nil
			}
		}
	}
	return nil
}

func (c *pyCompiler) compilePyAttribute(n *sitter.Node) *interp.Expr {
	// self.field → instance variable (mapped to global)
	if name := c.selfFieldName(n); name != "" {
		return &interp.Expr{Kind: interp.ExprVar, Name: name}
	}
	var obj *sitter.Node
	var fieldName string
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == nodeIdentifier {
			if obj == nil {
				obj = child
			} else {
				fieldName = c.text(child)
			}
		}
	}
	if obj == nil || fieldName == "" {
		return nil
	}
	// pos.field → PositionGetXxx(prop) for position loop variables
	objName := c.text(obj)
	if c.posLoopVars != nil && c.posLoopVars[objName] {
		if mapping, ok := positionFieldMap[fieldName]; ok {
			return &interp.Expr{
				Kind: interp.ExprCall,
				Name: mapping.builtin,
				Args: []interp.Expr{{Kind: interp.ExprLiteral, Val: interp.IntVal(mapping.prop)}},
			}
		}
	}
	return &interp.Expr{
		Kind: interp.ExprField,
		Name: fieldName,
		Args: []interp.Expr{*c.mustPyExpr(obj)},
	}
}

func (c *pyCompiler) compilePySubscript(n *sitter.Node) *interp.Expr {
	var obj *sitter.Node
	var idx *sitter.Node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case nodeIdentifier, nodeAttribute, nodeCall:
			if obj == nil {
				obj = child
			} else {
				idx = child
			}
		case "integer", "binary_operator", nodeParenExpr:
			idx = child
		}
	}
	if obj == nil || idx == nil {
		return nil
	}
	objName := c.findIdentPy(obj)
	// self.field[idx] → field[idx]
	if objName == "self" {
		if sfn := c.selfFieldName(obj); sfn != "" {
			objName = sfn
		}
	}
	if objName == "" {
		if obj.Type() == nodeAttribute {
			return &interp.Expr{
				Kind:  interp.ExprSubscript,
				Name:  c.text(obj),
				Index: c.compileExpr(idx),
			}
		}
		return nil
	}
	return &interp.Expr{
		Kind:  interp.ExprSubscript,
		Name:  objName,
		Index: c.compileExpr(idx),
	}
}

func (c *pyCompiler) findIdentPy(n *sitter.Node) string {
	if n == nil {
		return ""
	}
	if n.Type() == nodeIdentifier {
		return c.text(n)
	}
	// Don't recurse into attribute nodes — their first identifier is the object,
	// not the variable being assigned to. e.g. pos.sl should NOT return "pos".
	if n.Type() == nodeAttribute {
		return ""
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == nodeIdentifier {
			return c.text(child)
		}
	}
	return ""
}

func (c *pyCompiler) mustPyExpr(n *sitter.Node) *interp.Expr {
	e := c.compileExpr(n)
	if e == nil {
		e = &interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(0)}
	}
	return e
}

// selfFieldName returns the field name if n is a `self.field` attribute, else "".
// Also records the field in selfVars for global allocation.
func (c *pyCompiler) selfFieldName(n *sitter.Node) string {
	if n == nil || n.Type() != nodeAttribute {
		return ""
	}
	obj := n.NamedChild(0)
	if obj == nil || obj.Type() != nodeIdentifier || c.text(obj) != "self" {
		return ""
	}
	field := n.NamedChild(1)
	if field == nil || field.Type() != nodeIdentifier {
		return ""
	}
	name := c.text(field)
	if c.selfVars == nil {
		c.selfVars = make(map[string]bool)
	}
	c.selfVars[name] = true
	return name
}
