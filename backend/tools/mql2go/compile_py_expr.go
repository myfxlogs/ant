package mql2go

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
	sitter "github.com/smacker/go-tree-sitter"
	"anttrader/tools/mql2go/interp"
)

// compileExpr converts a Python CST expression node into interp.Expr.
func (c *pyCompiler) compileExpr(n *sitter.Node) *interp.Expr {
	if n == nil {
		return nil
	}
	switch n.Type() {
	case "identifier":
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

	case "float":
		txt := c.text(n)
		txt = strings.ReplaceAll(txt, "_", "")
		d, err := decimal.NewFromString(txt)
		if err != nil {
			d = decimal.Zero
		}
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.DecimalVal(d)}

	case "string":
		s := unquotePython(c.text(n))
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.StringVal(s)}

	case "true":
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.BoolVal(true)}

	case "false":
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.BoolVal(false)}

	case "none":
		return &interp.Expr{Kind: interp.ExprLiteral, Val: interp.NoneVal()}

	case "call":
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

	case "parenthesized_expression":
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

	case "attribute":
		return c.compilePyAttribute(n)

	case "subscript":
		return c.compilePySubscript(n)

	case "conditional_expression":
		return c.compilePyTernary(n)

	case "concatenated_string":
		var sb strings.Builder
		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			if child.Type() == "string" {
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
	if fnNode.Type() == "attribute" {
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
	case "float":
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
	return &interp.Expr{Kind: interp.ExprCall, Name: fullPath, Args: args}
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
		panic(fmt.Sprintf("line %d: binary operator not found in expression", n.StartPoint().Row+1))
	}
	switch op {
	case "**":
		return &interp.Expr{
			Kind: interp.ExprCall,
			Name: "MathPow",
			Args: []interp.Expr{*c.mustPyExpr(left), *c.mustPyExpr(right)},
		}
	case "&", "|", "^", "<<", ">>":
		panic(fmt.Sprintf("line %d: bitwise operator '%s' is not supported in Python subset (no VM opcode)", n.StartPoint().Row+1, op))
	}
	return &interp.Expr{
		Kind: interp.ExprBinary,
		Op:   op,
		Args: []interp.Expr{*c.mustPyExpr(left), *c.mustPyExpr(right)},
	}
}

func (c *pyCompiler) compilePyComparison(n *sitter.Node) *interp.Expr {
	var operands []*sitter.Node
	var ops []string
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		if child.IsNamed() {
			operands = append(operands, child)
		} else {
			ct := child.Type()
			switch ct {
			case "<", ">", "<=", ">=", "==", "!=", "<>", "is", "is not", "in", "not in":
				ops = append(ops, ct)
			}
		}
	}
	if len(operands) < 2 || len(ops) < 1 {
		return nil
	}
	// For simple comparison (2 operands), no temp needed
	if len(operands) == 2 {
		return c.makeComparison(operands[0], ops[0], operands[1])
	}
	// Chained comparison (3+ operands): store middle operands in temp variables
	// to avoid double evaluation. Python semantics: a < b < c means (a < b) && (b < c),
	// but b is evaluated only once.
	var seqArgs []interp.Expr
	// Declare temps for middle operands (indices 1..len-2)
	tempNames := make([]string, len(operands))
	tempNames[0] = "" // first operand: no temp needed
	tempNames[len(operands)-1] = "" // last operand: no temp needed
	for i := 1; i < len(operands)-1; i++ {
		tempName := fmt.Sprintf("__cmp_%d_%d", n.StartPoint().Row, i)
		tempNames[i] = tempName
		seqArgs = append(seqArgs, interp.Expr{
			Kind: interp.ExprDecl,
			Name: tempName,
			Args: []interp.Expr{*c.mustPyExpr(operands[i])},
		})
	}
	// Build the comparison chain using temps for middle operands
	result := c.makeComparisonWithTemp(operands[0], ops[0], tempNames[0], tempNames[1], operands[1])
	for i := 1; i < len(ops); i++ {
		leftName := tempNames[i]   // temp or "" for first
		rightName := tempNames[i+1] // temp or "" for last
		right := c.makeComparisonWithTemp(operands[i], ops[i], leftName, rightName, operands[i+1])
		result = &interp.Expr{
			Kind: interp.ExprBinary,
			Op:   "&&",
			Args: []interp.Expr{*result, *right},
		}
	}
	seqArgs = append(seqArgs, *result)
	return &interp.Expr{Kind: interp.ExprSeq, Args: seqArgs}
}

func (c *pyCompiler) makeComparison(left *sitter.Node, op string, right *sitter.Node) *interp.Expr {
	switch op {
	case "<>":
		op = "!="
	case "is":
		op = "=="
	case "is not":
		op = "!="
	case "in":
		return &interp.Expr{
			Kind: interp.ExprCall,
			Name: "operator_in",
			Args: []interp.Expr{*c.mustPyExpr(left), *c.mustPyExpr(right)},
		}
	case "not in":
		return &interp.Expr{
			Kind: interp.ExprUnary,
			Op:   "!",
			Args: []interp.Expr{{
				Kind: interp.ExprCall,
				Name: "operator_in",
				Args: []interp.Expr{*c.mustPyExpr(left), *c.mustPyExpr(right)},
			}},
		}
	}
	return &interp.Expr{
		Kind: interp.ExprBinary,
		Op:   op,
		Args: []interp.Expr{*c.mustPyExpr(left), *c.mustPyExpr(right)},
	}
}

// makeComparisonWithTemp builds a comparison where the left side may be a temp
// variable (tempLeftName != "") or a fresh compile (tempLeftName == "").
// Similarly for the right side via tempRightName + rightNode.
func (c *pyCompiler) makeComparisonWithTemp(leftNode *sitter.Node, op string, tempLeftName, tempRightName string, rightNode *sitter.Node) *interp.Expr {
	var leftExpr, rightExpr interp.Expr
	if tempLeftName != "" {
		leftExpr = interp.Expr{Kind: interp.ExprVar, Name: tempLeftName}
	} else {
		leftExpr = *c.mustPyExpr(leftNode)
	}
	if tempRightName != "" {
		rightExpr = interp.Expr{Kind: interp.ExprVar, Name: tempRightName}
	} else {
		rightExpr = *c.mustPyExpr(rightNode)
	}
	switch op {
	case "<>":
		op = "!="
	case "is":
		op = "=="
	case "is not":
		op = "!="
	case "in":
		return &interp.Expr{
			Kind: interp.ExprCall,
			Name: "operator_in",
			Args: []interp.Expr{leftExpr, rightExpr},
		}
	case "not in":
		return &interp.Expr{
			Kind: interp.ExprUnary,
			Op:   "!",
			Args: []interp.Expr{{
				Kind: interp.ExprCall,
				Name: "operator_in",
				Args: []interp.Expr{leftExpr, rightExpr},
			}},
		}
	}
	return &interp.Expr{
		Kind: interp.ExprBinary,
		Op:   op,
		Args: []interp.Expr{leftExpr, rightExpr},
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
	if op == "and" {
		op = "&&"
	} else if op == "or" {
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
		panic(fmt.Sprintf("line %d: bitwise NOT '~' is not supported in Python subset (no VM opcode)", n.StartPoint().Row+1))
	}
	return &interp.Expr{
		Kind: interp.ExprUnary,
		Op:   op,
		Args: []interp.Expr{*c.mustPyExpr(operand)},
	}
}
