package mql2go

import (
	sitter "github.com/smacker/go-tree-sitter"
	"alphaforge/tools/mql2go/interp"
)

// ── Helper functions for Python CST traversal ──────────────────────

func (c *pyCompiler) findClassName(n *sitter.Node) string {
	id := findNamedChild(n, nodeIdentifier)
	if id != nil {
		return c.text(id)
	}
	return ""
}

func (c *pyCompiler) findFuncName(n *sitter.Node) string {
	id := findNamedChild(n, nodeIdentifier)
	if id != nil {
		return c.text(id)
	}
	return ""
}

func (c *pyCompiler) findBlock(n *sitter.Node) *sitter.Node {
	return findNamedChild(n, "block")
}

func (c *pyCompiler) findExprChild(n *sitter.Node) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case nodeIdentifier, nodeString, "integer", nodeFloat, "true", "false", "none",
			nodeCall, "binary_operator", "unary_operator", "not_operator", "comparison_operator",
			"boolean_operator", "conditional_expression",
			nodeParenExpr, nodeAttribute, "subscript",
			"assignment", "concatenated_string":
			return child
		}
	}
	return nil
}

func (c *pyCompiler) findDefaultVal(n *sitter.Node) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "integer", nodeFloat, nodeString, "true", "false", "none",
			nodeCall, "binary_operator", "unary_operator", "not_operator", "comparison_operator",
			"boolean_operator", "conditional_expression",
			nodeParenExpr, nodeAttribute, "subscript":
			return child
		}
	}
	return nil
}

func (c *pyCompiler) compileExprFromStmt(n *sitter.Node) *interp.Expr {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if e := c.compileExpr(child); e != nil {
			return e
		}
	}
	return nil
}

func (c *pyCompiler) callName(n *sitter.Node) string {
	if n == nil || n.Type() != nodeCall {
		return ""
	}
	fn := n.NamedChild(0)
	if fn == nil {
		return ""
	}
	if fn.Type() == nodeIdentifier {
		return c.text(fn)
	}
	if fn.Type() == nodeAttribute {
		return c.text(fn)
	}
	return ""
}

// findArgumentList locates the argument_list child node of a call expression node.
// Uses findNamedChild first (fast path), then falls back to iterating named children.
func findArgumentList(n *sitter.Node) *sitter.Node {
	args := findNamedChild(n, "argument_list")
	if args == nil {
		for i := 0; i < int(n.NamedChildCount()); i++ {
			child := n.NamedChild(i)
			if child.Type() == "argument_list" {
				args = child
				break
			}
		}
	}
	return args
}

func (c *pyCompiler) compileArgs(n *sitter.Node) []interp.Expr {
	args := findArgumentList(n)
	if args == nil {
		return nil
	}
	var result []interp.Expr
	for i := 0; i < int(args.NamedChildCount()); i++ {
		a := args.NamedChild(i)
		if a.Type() == "," {
			continue
		}
		if a.Type() == "keyword_argument" {
			for j := 0; j < int(a.NamedChildCount()); j++ {
				kw := a.NamedChild(j)
				if kw.Type() != nodeIdentifier {
					e := c.compileExpr(kw)
					if e == nil {
						e = &interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(0)}
					}
					result = append(result, *e)
					break
				}
			}
			continue
		}
		e := c.compileExpr(a)
		if e == nil {
			e = &interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(0)}
		}
		result = append(result, *e)
	}
	return result
}

// compileArgsOrdered compiles call arguments, reordering keyword arguments to
// match the canonical parameter order of the called method. This handles
// Python keyword arguments like ctx.broker.buy(sl=..., lot=...) by reordering
// them to the positional order expected by the VM builtin (lot, ..., sl, ...).
func (c *pyCompiler) compileArgsOrdered(n *sitter.Node, methodPath string) []interp.Expr {
	args := findArgumentList(n)
	if args == nil {
		return nil
	}

	type argSlot struct {
		name string // keyword name, "" = positional
		expr interp.Expr
	}

	var slots []argSlot
	for i := 0; i < int(args.NamedChildCount()); i++ {
		a := args.NamedChild(i)
		if a.Type() == "," {
			continue
		}
		if a.Type() == "keyword_argument" {
			kwName := ""
			var kwExpr *interp.Expr
			for j := 0; j < int(a.NamedChildCount()); j++ {
				kw := a.NamedChild(j)
				if kw.Type() == nodeIdentifier {
					kwName = c.text(kw)
				} else {
					e := c.compileExpr(kw)
					if e == nil {
						e = &interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(0)}
					}
					kwExpr = e
				}
			}
			if kwExpr != nil {
				slots = append(slots, argSlot{name: kwName, expr: *kwExpr})
			}
			continue
		}
		e := c.compileExpr(a)
		if e == nil {
			e = &interp.Expr{Kind: interp.ExprLiteral, Val: interp.IntVal(0)}
		}
		slots = append(slots, argSlot{expr: *e})
	}

	// If no keyword arguments, return as-is
	hasKeyword := false
	for _, s := range slots {
		if s.name != "" {
			hasKeyword = true
			break
		}
	}
	if !hasKeyword {
		result := make([]interp.Expr, len(slots))
		for i, s := range slots {
			result[i] = s.expr
		}
		return result
	}

	// Reorder based on canonical parameter order
	paramOrder := pythonMethodParamOrder(methodPath)
	if paramOrder == nil {
		// Unknown method — can't reorder, return in original order
		result := make([]interp.Expr, len(slots))
		for i, s := range slots {
			result[i] = s.expr
		}
		return result
	}

	// Build position map: param name → index
	posMap := make(map[string]int)
	for i, p := range paramOrder {
		posMap[p] = i
	}

	result := make([]interp.Expr, len(paramOrder))
	for i := range result {
		result[i] = interp.Expr{Kind: interp.ExprLiteral, Val: interp.NoneVal()}
	}

	// Place positional args first (fill from left)
	posIdx := 0
	for _, s := range slots {
		if s.name == "" {
			if posIdx < len(result) {
				result[posIdx] = s.expr
			}
			posIdx++
			continue
		}
		// Keyword arg — resolve alias then place at canonical position
		canonical := resolveParamName(s.name)
		if idx, ok := posMap[canonical]; ok {
			result[idx] = s.expr
		}
	}

	return result
}

// rangeInit returns the init expression for a range-based for-loop.
// Single-arg range(N): init i=0 (use 0, not N as the start value).
// Two-arg range(M,N) and three-arg range(M,N,S): init i=M.
func rangeInit(args []interp.Expr) interp.Expr {
	if len(args) == 1 {
		return interp.Expr{
			Kind: interp.ExprLiteral,
			Val:  interp.Value{Kind: interp.ValInt, Int: 0},
		}
	}
	return args[0]
}

// countBases returns the number of base classes in a class definition.
// Used to reject multiple inheritance (ADR-0024 D3 prohibits it).
func (c *pyCompiler) countBases(n *sitter.Node) int {
	count := 0
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		if child.Type() == "argument_list" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				gc := child.NamedChild(j)
				if gc.Type() == nodeIdentifier {
					count++
				}
			}
		}
	}
	return count
}
