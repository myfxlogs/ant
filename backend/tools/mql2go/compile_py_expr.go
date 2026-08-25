package mql2go

import (
	"strconv"
	"strings"

	"alphaforge/tools/mql2go/interp"

	"github.com/shopspring/decimal"
	sitter "github.com/smacker/go-tree-sitter"
)

// compileExpr converts a Python CST expression node into interp.Expr.
func (c *pyCompiler) compileExpr(n *sitter.Node) *interp.Expr {
	if n == nil {
		return nil
	}
	switch n.Type() {
	case nodeIdentifier:
		return &interp.Expr{Kind: interp.ExprVar, Name: c.text(n)}

	case "integer":
		return c.compileIntegerLiteral(n)

	case nodeFloat:
		txt := c.text(n)
		txt = strings.ReplaceAll(txt, "_", "")
		d, err := decimal.NewFromString(txt)
		if err != nil {
			d = decimal.Zero
		}
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.DecimalVal(d)}

	case nodeString:
		s := unquotePython(c.text(n))
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.StringVal(s)}

	case nodeTrue:
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.BoolVal(true)}

	case nodeFalse:
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.BoolVal(false)}

	case "none":
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.NoneVal()}

	case nodeCall:
		return c.compilePyCall(n)

	case "binary_operator":
		return c.compilePyBinary(n)

	case "unary_operator":
		return c.compilePyUnary(n)

	case "not_operator":
		operand := n.NamedChild(0)
		if operand == nil {
			return nil
		}
		return &interp.Expr{
			Kind: interp.ExprUnary,
			Op:   "!",
			Args: []interp.Expr{*c.mustPyExpr(operand)},
		}

	case "comparison_operator":
		return c.compilePyComparison(n)

	case "boolean_operator":
		return c.compilePyBoolean(n)

	case nodeParenExpr:
		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			if e := c.compileExpr(child); e != nil {
				return e
			}
		}
		return nil

	case "assignment":
		return c.compilePyAssignment(n)

	case "augmented_assignment":
		return c.compilePyAugmentedAssign(n)

	case nodeAttribute:
		return c.compilePyAttribute(n)

	case "subscript":
		return c.compilePySubscript(n)

	case "conditional_expression":
		return c.compilePyTernary(n)

	case "concatenated_string":
		var sb strings.Builder
		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			if child.Type() == nodeString {
				sb.WriteString(unquotePython(c.text(child)))
			}
		}
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.StringVal(sb.String())}
	}
	return nil
}

func (c *pyCompiler) compileIntegerLiteral(n *sitter.Node) *interp.Expr {
	txt := c.text(n)
	txt = strings.ReplaceAll(txt, "_", "")
	if strings.HasPrefix(txt, "0x") || strings.HasPrefix(txt, "0X") {
		var v int64
		for _, ch := range txt[2:] {
			if ch >= '0' && ch <= '9' {
				v = v*16 + int64(ch-'0')
			} else if ch >= 'a' && ch <= 'f' {
				v = v*16 + int64(ch-'a'+10)
			} else if ch >= 'A' && ch <= 'F' {
				v = v*16 + int64(ch-'A'+10)
			}
		}
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(int32(v))}
	}
	v, err := strconv.ParseInt(txt, 10, 64)
	if err != nil {
		d, derr := decimal.NewFromString(txt)
		if derr != nil {
			d = decimal.Zero
		}
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.DecimalVal(d)}
	}
	if v > 2147483647 || v < -2147483648 {
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.DecimalVal(decimal.NewFromInt(v))}
	}
	return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(int32(v))}
}

func (c *pyCompiler) compilePyCall(n *sitter.Node) *interp.Expr {
	fnNode := n.NamedChild(0)
	if fnNode == nil {
		return nil
	}
	if fnNode.Type() == nodeAttribute {
		return c.compilePyMethodCall(n, fnNode)
	}
	name := c.text(fnNode)
	args := c.compileArgs(n)
	switch name {
	case "int":
		if len(args) > 0 {
			return &args[0]
		}
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(0)}
	case nodeFloat:
		if len(args) > 0 {
			return &args[0]
		}
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.DecimalVal(decimalZero)}
	case "str":
		if len(args) > 0 {
			return &args[0]
		}
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.StringVal("")}
	case "bool":
		if len(args) > 0 {
			return &interp.Expr{
				Kind: interp.ExprBinary,
				Op:   "!=",
				Args: []interp.Expr{args[0], {Kind: interp.ExprLiteral, Val: interp.IntVal(0)}},
			}
		}
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.BoolVal(false)}
	case "Decimal":
		if len(args) > 0 {
			return &args[0]
		}
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.DecimalVal(decimalZero)}
	case "abs":
		return &interp.Expr{Kind: interp.ExprCall, Name: "MathAbs", Args: args}
	case "max":
		return &interp.Expr{Kind: interp.ExprCall, Name: "MathMax", Args: args}
	case "min":
		return &interp.Expr{Kind: interp.ExprCall, Name: "MathMin", Args: args}
	}
	return &interp.Expr{Kind: interp.ExprCall, Name: name, Args: args}
}
