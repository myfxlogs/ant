package mql2go

import (
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
	sitter "github.com/smacker/go-tree-sitter"
	"alphaforge/tools/mql2go/interp"
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

	case "true":
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.BoolVal(true)}

	case "false":
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

func (c *pyCompiler) compilePyMethodCall(callNode, attrNode *sitter.Node) *interp.Expr {
	fullPath := c.text(attrNode)
	args := c.compileArgsOrdered(callNode, fullPath)
	parts := strings.Split(fullPath, ".")
	methodName := parts[len(parts)-1]
	// self.method() → method()
	if len(parts) == 2 && parts[0] == "self" {
		return &interp.Expr{Kind: interp.ExprCall, Name: methodName, Args: args}
	}
	mappedName := mapPythonMethod(methodName, fullPath)
	if mappedName != "" {
		return &interp.Expr{Kind: interp.ExprCall, Name: mappedName, Args: args}
	}
	// Try clean chain path for chained calls like ctx.bars_for_symbol("X").close(0)
	cleanPath := c.extractAttrChain(attrNode)
	if cleanPath != fullPath && cleanPath != "" {
		mappedName = mapPythonMethod(methodName, cleanPath)
		if mappedName != "" {
			// For multi-symbol bar access via bars_for_symbol, prepend symbol arg
			// and inject timeframe=0 (PERIOD_CURRENT) between symbol and shift.
			// iClose(symbol, timeframe, shift) ← ctx.bars_for_symbol("EURUSD").close(0)
			if strings.HasPrefix(mappedName, "iClose") ||
				strings.HasPrefix(mappedName, "iOpen") ||
				strings.HasPrefix(mappedName, "iHigh") ||
				strings.HasPrefix(mappedName, "iLow") ||
				strings.HasPrefix(mappedName, "iVolume") ||
				strings.HasPrefix(mappedName, "iTime") {
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
				// Higher timeframe: ctx.bars_tf("H4").close(0)
				// iClose("", 240, shift) — symbol="" (primary), timeframe from TF string
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
			}
			return &interp.Expr{Kind: interp.ExprCall, Name: mappedName, Args: args}
		}
	}
	return &interp.Expr{Kind: interp.ExprCall, Name: fullPath, Args: args}
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
