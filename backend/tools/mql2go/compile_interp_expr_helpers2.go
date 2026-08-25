package mql2go

import (
	"fmt"
	"strings"

	"alphaforge/tools/mql2go/interp"

	sitter "github.com/smacker/go-tree-sitter"
)

// This file contains expression compilation helpers (subscript,
// assignment, update, ternary, field) extracted from
// compile_interp_helpers.go for file-size compliance.

func (c *compiler) compileSubscript(n *sitter.Node) *interp.Expr {
	var name string
	var idx *sitter.Node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == nodeIdentifier && name == "" {
			name = c.text(child)
		} else if child.Type() != "[" && child.Type() != "]" {
			idx = child
		}
	}
	if name == "" || idx == nil {
		if c.err == nil {
			c.err = fmt.Errorf("subscript expression has invalid base or index")
		}
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
		if c.err == nil {
			c.err = fmt.Errorf("assignment expression has missing operand")
		}
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

	// Field assignment: obj.field = value → ExprField with IsAssign=true.
	// Check structured lvalues before findIdent, which intentionally descends
	// into nodes and would otherwise return the object name.
	if lhs.Type() == "field_expression" {
		fieldExpr := c.compileField(lhs)
		if fieldExpr != nil {
			if op != "=" {
				// VM-COMPILER-SEMANTICS-2: compound assign on field (obj.field += v)
				// must preserve compound semantics. Desugar to:
				//   obj.field = obj.field OP v
				// by creating a read expression and wrapping in a binary op.
				readExpr := *fieldExpr // copy the field read
				readExpr.IsAssign = false
				binaryExpr := interp.Expr{
					Kind: interp.ExprBinary,
					Op:   op[:len(op)-1], // "+=" → "+"
					Args: []interp.Expr{readExpr, c.mustExpr(rhs)},
				}
				fieldExpr.IsAssign = true
				fieldExpr.Args = append(fieldExpr.Args, binaryExpr)
				return fieldExpr
			}
			fieldExpr.IsAssign = true
			fieldExpr.Args = append(fieldExpr.Args, c.mustExpr(rhs))
			return fieldExpr
		}
	}

	// Subscript assignment: arr[i] = value
	if lhs.Type() == "subscript_expression" {
		subExpr := c.compileSubscript(lhs)
		if subExpr != nil {
			if op != "=" {
				// VM-COMPILER-SEMANTICS-2: compound assign on subscript (arr[i] += v)
				// must preserve compound semantics. Desugar to:
				//   arr[i] = arr[i] OP v
				readExpr := *subExpr // copy the subscript read
				binaryExpr := interp.Expr{
					Kind: interp.ExprBinary,
					Op:   op[:len(op)-1], // "+=" → "+"
					Args: []interp.Expr{readExpr, c.mustExpr(rhs)},
				}
				subExpr.Args = []interp.Expr{binaryExpr}
				return subExpr
			}
			subExpr.Args = []interp.Expr{c.mustExpr(rhs)}
			return subExpr
		}
	}

	// Simple variable assignment: x = value (or x += value). Only accept a
	// direct identifier; nested lvalues were handled above.
	name := c.directIdentifier(lhs)
	if name == "" {
		if c.err == nil {
			c.err = fmt.Errorf("unsupported assignment target: %s", c.text(lhs))
		}
		return nil
	}
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

func (c *compiler) directIdentifier(n *sitter.Node) string {
	if n == nil || n.Type() != nodeIdentifier {
		return ""
	}
	return c.text(n)
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
		if c.err == nil {
			c.err = fmt.Errorf("conditional expression has missing branch")
		}
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
		case nodeIdentifier:
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
		if c.err == nil {
			c.err = fmt.Errorf("field expression has invalid object or field")
		}
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
