package mql2go

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"alphaforge/tools/mql2go/interp"
)

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
