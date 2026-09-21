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
	// VM-BLOCK-SCOPE-1: body gets its own scope inside the for scope — body
	// declarations shadow for-init names instead of rebinding them (for-init
	// and update keep resolving in the outer for scope).
	c.pushScope()
	c.compileStmts(s.Body)
	c.popScope()
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
	// VM-BLOCK-SCOPE-1: loop body scope — body declarations die with the body.
	c.pushScope()
	c.compileStmts(s.Body)
	c.popScope()
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
	// VM-BLOCK-SCOPE-1: loop body scope — body declarations die with the body.
	c.pushScope()
	c.compileStmts(s.Body)
	c.popScope()
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

	lc := &loopContext{} // switch uses loopContext for break only
	c.loopStack = append(c.loopStack, lc)
	// VM-BLOCK-SCOPE-1: the whole switch body shares ONE scope — case labels
	// don't create scopes (C/MQL semantics: a declaration in one case is
	// reachable in later cases via fallthrough), and nothing leaks past the
	// switch. Patching below compiles no statements, so the scope closes here.
	c.pushScope()
	endJumps := []int32{}

	// VM-COMPILER-SEMANTICS-3 S1: preserve original case order (default stays
	// in place). Default does not emit a comparison but is a fallthrough target.
	var caseBodyStarts []int32     // body start for each case (fallthrough target)
	var regularCaseStarts []int32  // comparison start for regular cases only
	var regularJmpFalse []int32    // JMP_IF_FALSE for regular cases
	var regularJmpIdx []int        // original case index for each regular jmp
	var fallthroughJmps []int32    // JMP from fallthrough case body to next case body
	var fallthroughTargets []int32 // target case body index for each fallthrough jmp
	var defaultBodyStart int32
	hasDefault := false

	// If default is the first case and there are regular cases, emit a skip JMP
	// to the first regular case comparison so default body is not executed
	// unconditionally (C semantics: default runs only when no case matches).
	var defaultSkipJmp int32 = -1
	if len(s.Cases) > 0 && s.Cases[0].Expr == nil {
		hasRegular := false
		for _, sc := range s.Cases {
			if sc.Expr != nil {
				hasRegular = true
				break
			}
		}
		if hasRegular {
			defaultSkipJmp = c.emitJump(OP_JMP, 0)
		}
	}

	for i, sc := range s.Cases {
		if sc.Expr == nil {
			// Default: no comparison, just body. Fallthrough target.
			bodyStart := int32(len(c.bc.Code))
			caseBodyStarts = append(caseBodyStarts, bodyStart)
			if !hasDefault {
				hasDefault = true
				defaultBodyStart = bodyStart
			}
			c.compileStmts(sc.Body)
			if sc.HasBreak {
				endJumps = append(endJumps, c.emitJump(OP_JMP, 0))
			} else if i+1 < len(s.Cases) {
				fj := c.emitJump(OP_JMP, 0)
				fallthroughJmps = append(fallthroughJmps, fj)
				fallthroughTargets = append(fallthroughTargets, int32(i+1))
			}
		} else {
			regularCaseStarts = append(regularCaseStarts, int32(len(c.bc.Code)))
			c.emit(OP_DUP, 0, 0, 0)
			c.compileExpr(sc.Expr)
			c.emit(OP_EQ, 0, 0, 0)
			jmpNext := c.emitJump(OP_JMP_IF_FALSE, 0)
			regularJmpFalse = append(regularJmpFalse, jmpNext)
			regularJmpIdx = append(regularJmpIdx, i)
			bodyStart := int32(len(c.bc.Code))
			caseBodyStarts = append(caseBodyStarts, bodyStart)
			c.compileStmts(sc.Body)
			if sc.HasBreak {
				endJumps = append(endJumps, c.emitJump(OP_JMP, 0))
			} else if i+1 < len(s.Cases) {
				fj := c.emitJump(OP_JMP, 0)
				fallthroughJmps = append(fallthroughJmps, fj)
				fallthroughTargets = append(fallthroughTargets, int32(i+1))
			}
		}
	}

	// Patch default skip JMP (if default was first) to the first regular case.
	c.popScope() // VM-BLOCK-SCOPE-1: close the shared switch body scope
	if defaultSkipJmp >= 0 && len(regularCaseStarts) > 0 {
		c.bc.Code[defaultSkipJmp].A = regularCaseStarts[0]
	}

	// Patch each regular case's JMP_IF_FALSE.
	// VM-COMPILER-SEMANTICS-3 S1: default stays in original order.
	// - Fallthrough case (no break): JMP_IF_FALSE targets next case BODY
	//   (could be default body or regular case body).
	// - Normal case (has break): JMP_IF_FALSE targets next REGULAR case's
	//   comparison (skip default — default has no comparison). If no next
	//   regular case, target default body (if any) or POP.
	for ri, jf := range regularJmpFalse {
		caseIdx := regularJmpIdx[ri]
		sc := s.Cases[caseIdx]
		if !sc.HasBreak {
			c.bc.Code[jf].A = caseBodyStarts[caseIdx+1]
		} else {
			if ri+1 < len(regularCaseStarts) {
				c.bc.Code[jf].A = regularCaseStarts[ri+1]
			} else if hasDefault {
				c.bc.Code[jf].A = defaultBodyStart
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

	// Pop the switch expression.
	// VM-COMPILER-SEMANTICS-3 S2: record popPC so break paths execute OP_POP
	// before continuing past the switch. Previously break JMPs skipped OP_POP,
	// leaving the switch value on the stack and polluting subsequent statements.
	popPC := int32(len(c.bc.Code))
	c.emit(OP_POP, 0, 0, 0)
	for _, ej := range endJumps {
		c.bc.Code[ej].A = popPC
	}
	for _, bj := range lc.breakJumps {
		c.bc.Code[bj].A = popPC
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
	case interp.ExprField, interp.ExprSubscript:
		// Field/subscript assignment (obj.field = val, arr[i] = val) is stack-neutral:
		// OP_SET_FIELD / OP_STORE_ARRAY pop 2 and push nothing.
		return e.IsAssign
	case interp.ExprSeq:
		// ExprSeq is stack-neutral if its last child is stack-neutral
		// (only the last child leaves a value, if any).
		if len(e.Args) == 0 {
			return true
		}
		return isStackNeutral(&e.Args[len(e.Args)-1])
	}
	return false
}
