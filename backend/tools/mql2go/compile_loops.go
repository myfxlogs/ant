package mql2go

import (
	"fmt"

	"alphaforge/tools/mql2go/interp"

	sitter "github.com/smacker/go-tree-sitter"
)

func (c *astCompiler) compileFor(s *interp.Statement) {
	c.pushScope()

	// Init
	if s.Init != nil {
		c.compileStmt(s.Init)
	}

	// Condition
	condStart := int32(len(c.bc.Code))
	var jmpEnd int32
	if s.Cond != nil {
		c.compileExpr(s.Cond)
		jmpEnd = c.emitJump(OP_JMP_IF_FALSE, 0)
	} else {
		// No condition — infinite loop (until break)
		jmpEnd = -2 // sentinel: no condition jump
	}

	// Body — push loop context so break/continue at any nesting depth are tracked
	lc := &loopContext{}
	c.loopStack = append(c.loopStack, lc)
	c.compileStmts(s.Body)
	c.loopStack = c.loopStack[:len(c.loopStack)-1]

	// Update — continue jumps here (after body, before condition check)
	updateStart := int32(len(c.bc.Code))
	for _, cj := range lc.continueJumps {
		c.bc.Code[cj].A = updateStart
	}
	if s.Update != nil {
		c.compileStmt(s.Update)
	}

	// Jump back to condition
	c.emit(OP_JMP, condStart, 0, 0)

	// Patch break jumps and condition-false jump to here
	endPC := int32(len(c.bc.Code))
	if jmpEnd >= 0 {
		c.patchJump(jmpEnd)
	}
	for _, bj := range lc.breakJumps {
		c.bc.Code[bj].A = endPC
	}

	c.popScope()
}

func (c *astCompiler) compileWhile(s *interp.Statement) {
	condStart := int32(len(c.bc.Code))

	c.compileExpr(s.Cond)
	jmpEnd := c.emitJump(OP_JMP_IF_FALSE, 0)

	lc := &loopContext{}
	c.loopStack = append(c.loopStack, lc)
	c.compileStmts(s.Body)
	c.loopStack = c.loopStack[:len(c.loopStack)-1]

	// Patch continue jumps to condition check
	for _, cj := range lc.continueJumps {
		c.bc.Code[cj].A = condStart
	}

	// Jump back to condition
	c.emit(OP_JMP, condStart, 0, 0)

	endPC := int32(len(c.bc.Code))
	c.patchJump(jmpEnd)
	for _, bj := range lc.breakJumps {
		c.bc.Code[bj].A = endPC
	}
}

func (c *astCompiler) compileDoWhile(s *interp.Statement) {
	bodyStart := int32(len(c.bc.Code))

	lc := &loopContext{}
	c.loopStack = append(c.loopStack, lc)
	c.compileStmts(s.Body)
	c.loopStack = c.loopStack[:len(c.loopStack)-1]

	// Condition check — continue jumps here
	condStart := int32(len(c.bc.Code))
	for _, cj := range lc.continueJumps {
		c.bc.Code[cj].A = condStart
	}

	c.compileExpr(s.Cond)
	c.emit(OP_JMP_IF_TRUE, bodyStart, 0, 0)

	endPC := int32(len(c.bc.Code))
	for _, bj := range lc.breakJumps {
		c.bc.Code[bj].A = endPC
	}
}

func (c *astCompiler) compileSwitch(s *interp.Statement) {
	c.compileExpr(s.Expr)

	lc := &loopContext{} // switch uses loop context for break only
	c.loopStack = append(c.loopStack, lc)

	// Build dispatch targets preserving source order (including default).
	// MQL/C allows default to appear anywhere among cases; fallthrough
	// follows source order, so we must not reorder.
	type caseTarget struct {
		isDefault bool
		body      []interp.Statement
		expr      *interp.Expr
	}
	var targets []caseTarget
	var defaultIdx int = -1
	for i, sc := range s.Cases {
		t := caseTarget{isDefault: sc.Expr == nil, body: sc.Body, expr: sc.Expr}
		if sc.Expr == nil {
			defaultIdx = i
		}
		targets = append(targets, t)
	}

	// Phase 1: Emit comparison chain for non-default cases only.
	// Each comparison: DUP, push case expr, EQ, JMP_IF_TRUE → case body.
	// The switch value stays on the stack throughout comparisons.
	// We record jump targets for all cases (including default) to patch later.
	bodyPCs := make([]int32, len(targets))
	jmpTrueIndices := make([]int32, len(targets))
	for i := range targets {
		jmpTrueIndices[i] = -1 // sentinel for default (no comparison)
	}

	// Emit comparisons for non-default cases in source order
	for i, t := range targets {
		if t.isDefault {
			continue
		}
		c.emit(OP_DUP, 0, 0, 0)
		c.compileExpr(t.expr)
		c.emit(OP_EQ, 0, 0, 0)
		jmpTrueIndices[i] = c.emitJump(OP_JMP_IF_TRUE, 0)
	}

	// No case matched — jump to default body (if present) or pop+end
	jmpDefault := c.emitJump(OP_JMP, 0)

	// Phase 2: Emit case bodies in source order (including default).
	// Fallthrough is natural: if a body doesn't end with break, execution
	// continues to the next body in source order.
	for i := range targets {
		bodyPCs[i] = int32(len(c.bc.Code))
		c.compileStmts(targets[i].body)
	}

	// Pop the switch expression — reached by fallthrough from last body
	// or by jmpDefault when no default exists.
	popPC := int32(len(c.bc.Code))
	c.emit(OP_POP, 0, 0, 0)

	// Break exit point: pop the switch value, then continue after switch.
	// Break must also consume the switch value (it's still on the stack).
	breakPopPC := popPC

	// Patch JMP_IF_TRUE to each case body (non-default only)
	for i, jt := range jmpTrueIndices {
		if jt >= 0 {
			c.bc.Code[jt].A = bodyPCs[i]
		}
	}

	// Patch jmpDefault to default body (if present) or pop+end
	if defaultIdx >= 0 {
		c.bc.Code[jmpDefault].A = bodyPCs[defaultIdx]
	} else {
		c.bc.Code[jmpDefault].A = popPC
	}

	// Patch break jumps to the pop instruction (consumes switch value),
	// then fall through to endPC. This ensures break cleans the stack.
	for _, bj := range lc.breakJumps {
		c.bc.Code[bj].A = breakPopPC
	}
	c.loopStack = c.loopStack[:len(c.loopStack)-1]
}

// isStackNeutral returns true for expression types that don't leave a value
// on the stack after compilation (they push then store/pop internally).
// These should not be followed by OP_POP in StmtExpr context.
func isStackNeutral(e *interp.Expr) bool {
	if e == nil {
		return true
	}
	switch e.Kind {
	case interp.ExprDecl,
		interp.ExprAssignment,
		interp.ExprCompoundAssign,
		interp.ExprUpdate:
		return true
	case interp.ExprField:
		// Field assignment (obj.field = v) is stack-neutral;
		// field read (obj.field) leaves a value on the stack.
		return e.IsAssign
	case interp.ExprSubscript:
		// Subscript assignment (arr[i] = v) has Args; read does not.
		return len(e.Args) > 0
	case interp.ExprSeq:
		// ExprSeq is stack-neutral if its last child is stack-neutral
		// (the last child determines whether a value remains on the stack).
		if len(e.Args) == 0 {
			return true
		}
		return isStackNeutral(&e.Args[len(e.Args)-1])
	}
	return false
}

// ── IR-level loop compilation (*compiler methods) ────────────────────

func (c *compiler) compileFor(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtFor}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "declaration":
			if s := c.compileDeclaration(child); s != nil {
				stmt.Init = s
			}
		case "expression_statement":
			// Could be init or condition
			expr := c.compileExprFromStmt(child)
			if expr != nil {
				if stmt.Init == nil {
					stmt.Init = &interp.Statement{Kind: interp.StmtExpr, Expr: expr}
				} else if stmt.Cond == nil {
					stmt.Cond = expr
				}
			}
		case "binary_expression", "call_expression", nodeIdentifier, "number_literal":
			if stmt.Cond == nil {
				stmt.Cond = c.compileExpr(child)
			}
		case "update_expression":
			if s := c.compileStmt(child); s != nil {
				stmt.Update = s
			} else {
				expr := c.compileExpr(child)
				if expr != nil {
					stmt.Update = &interp.Statement{Kind: interp.StmtExpr, Expr: expr}
				}
			}
		case nodeCompoundStatement:
			stmt.Body = c.compileBlock(child)
		default:
			if childStmt := c.compileStmt(child); childStmt != nil {
				stmt.Body = append(stmt.Body, *childStmt)
			}
		}
	}
	return stmt
}

func (c *compiler) compileWhile(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtWhile}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case nodeParenExpr:
			stmt.Cond = c.compileExpr(child)
		case nodeCompoundStatement:
			stmt.Body = c.compileBlock(child)
		default:
			if childStmt := c.compileStmt(child); childStmt != nil {
				stmt.Body = append(stmt.Body, *childStmt)
			}
		}
	}
	if stmt.Cond == nil && c.err == nil {
		c.err = fmt.Errorf("while statement has no condition")
	}
	return stmt
}

func (c *compiler) compileSwitch(n *sitter.Node) *interp.Statement {
	stmt := &interp.Statement{Kind: interp.StmtSwitch}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case nodeParenExpr:
			stmt.Expr = c.compileExpr(child)
		case nodeCompoundStatement:
			c.appendSwitchCases(stmt, child)
		case "case_statement":
			stmt.Cases = append(stmt.Cases, c.compileSwitchCase(child))
		}
	}
	if stmt.Expr == nil && c.err == nil {
		c.err = fmt.Errorf("switch statement has no condition")
	}
	return stmt
}

func (c *compiler) appendSwitchCases(stmt *interp.Statement, body *sitter.Node) {
	for i := 0; i < int(body.NamedChildCount()); i++ {
		child := body.NamedChild(i)
		if child.Type() == "case_statement" {
			stmt.Cases = append(stmt.Cases, c.compileSwitchCase(child))
		}
	}
}

func (c *compiler) compileSwitchCase(n *sitter.Node) interp.SwitchCase {
	var sc interp.SwitchCase
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "default_label":
			sc.Expr = nil
		case "case_label":
			if child.NamedChildCount() > 0 {
				sc.Expr = c.compileExpr(child.NamedChild(0))
			}
		case "break_statement":
			// Break is compiled as StmtBreak so the AST compiler can emit
			// a jump to end. Without break, fallthrough occurs (MQL/C semantics).
			sc.Body = append(sc.Body, interp.Statement{Kind: interp.StmtBreak})
		default:
			if stmt := c.compileStmt(child); stmt != nil {
				sc.Body = append(sc.Body, *stmt)
				continue
			}
			if sc.Expr == nil {
				sc.Expr = c.compileExpr(child)
			}
		}
	}
	return sc
}
