package mql2go

import (
	"alphaforge/tools/mql2go/interp"
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
	endJumps := []int32{}

	// Separate default case from regular cases.
	// Default is compiled last so it only runs when no case matches.
	var defaultBody []interp.Statement
	var regularCases []interp.SwitchCase
	for _, sc := range s.Cases {
		if sc.Expr == nil {
			defaultBody = sc.Body
		} else {
			regularCases = append(regularCases, sc)
		}
	}

	// VM-COMPILER-SEMANTICS-1: compile each case with fallthrough support.
	// A case without HasBreak falls through to the next case's body
	// (skipping the next case's comparison).
	var caseStarts []int32        // comparison start for each case
	var caseBodyStarts []int32    // body start for each case (for fallthrough target)
	var jmpFalseIndices []int32   // JMP_IF_FALSE for each case
	var fallthroughJmps []int32   // JMP from fallthrough case body to next case body
	var fallthroughTargets []int32 // target case body index for each fallthrough jmp

	for i, sc := range regularCases {
		caseStarts = append(caseStarts, int32(len(c.bc.Code)))
		c.emit(OP_DUP, 0, 0, 0)
		c.compileExpr(sc.Expr)
		c.emit(OP_EQ, 0, 0, 0)
		jmpNext := c.emitJump(OP_JMP_IF_FALSE, 0)
		jmpFalseIndices = append(jmpFalseIndices, jmpNext)
		// Matched — execute case body
		bodyStart := int32(len(c.bc.Code))
		caseBodyStarts = append(caseBodyStarts, bodyStart)
		c.compileStmts(sc.Body)
		if sc.HasBreak {
			// Case ends with break — jump to end of switch.
			endJumps = append(endJumps, c.emitJump(OP_JMP, 0))
		} else if i+1 < len(regularCases) {
			// VM-COMPILER-SEMANTICS-1: fallthrough — jump to next case's BODY
			// (skip its comparison). We emit a JMP placeholder now and patch it
			// to caseBodyStarts[i+1] after all cases are compiled.
			fj := c.emitJump(OP_JMP, 0)
			fallthroughJmps = append(fallthroughJmps, fj)
			fallthroughTargets = append(fallthroughTargets, int32(i+1))
		}
	}

	// Default body (compiled after all cases so it only runs when no case matches)
	var defaultStart int32
	if defaultBody != nil {
		defaultStart = int32(len(c.bc.Code))
		c.compileStmts(defaultBody)
		endJumps = append(endJumps, c.emitJump(OP_JMP, 0))
	}

	// Patch each case's JMP_IF_FALSE.
	// VM-COMPILER-SEMANTICS-1: fallthrough cases jump to the next case's BODY
	// (not comparison), so fallthrough skips the next case's comparison.
	for i, jf := range jmpFalseIndices {
		sc := regularCases[i]
		if !sc.HasBreak && i+1 < len(regularCases) {
			// Fallthrough case: JMP_IF_FALSE targets next case's body
			c.bc.Code[jf].A = caseBodyStarts[i+1]
		} else if i+1 < len(jmpFalseIndices) {
			// Normal case: JMP_IF_FALSE targets next case's comparison
			c.bc.Code[jf].A = caseStarts[i+1]
		} else {
			// Last case — jump to default or POP
			if defaultBody != nil {
				c.bc.Code[jf].A = defaultStart
			} else {
				c.bc.Code[jf].A = int32(len(c.bc.Code))
			}
		}
	}

	// Patch fallthrough JMPs to their target case body starts.
	for idx, fj := range fallthroughJmps {
		targetIdx := fallthroughTargets[idx]
		c.bc.Code[fj].A = caseBodyStarts[targetIdx]
	}

	// Pop the switch expression
	c.emit(OP_POP, 0, 0, 0)

	endPC := int32(len(c.bc.Code))
	for _, ej := range endJumps {
		c.patchJump(ej)
	}
	for _, bj := range lc.breakJumps {
		c.bc.Code[bj].A = endPC
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
	}
	return false
}
