package mql2go

import (
	"fmt"

	"alphaforge/tools/mql2go/interp"
)

// This file contains compiler helper methods (emit, scope, variable
// resolution) and function/statement compilation extracted from
// compile.go for file-size compliance.

// emit appends an instruction and returns its index.
func (c *astCompiler) emit(op Opcode, a, b int32, line uint32) int32 {
	idx := int32(len(c.bc.Code))
	c.bc.Code = append(c.bc.Code, Instruction{Op: op, A: a, B: b, Line: line})
	return idx
}

// emitJump emits a jump instruction with a placeholder target.
// The target is patched later via patchJump.
func (c *astCompiler) emitJump(op Opcode, line uint32) int32 {
	return c.emit(op, -1, 0, line) // -1 = placeholder
}

// patchJump sets the jump target of the instruction at idx to the current code position.
func (c *astCompiler) patchJump(idx int32) {
	c.bc.Code[idx].A = int32(len(c.bc.Code))
}

// patchJumps resolves all remaining forward jumps.
func (c *astCompiler) patchJumps() {
	// All jumps should have been patched by patchJump calls during compilation.
	// This is a safety net — any remaining -1 targets are bugs.
	for i, ins := range c.bc.Code {
		if (ins.Op == OP_JMP || ins.Op == OP_JMP_IF_FALSE || ins.Op == OP_JMP_IF_TRUE) && ins.A == -1 {
			panic(fmt.Sprintf("unpatched jump at instruction %d", i))
		}
	}
}

func (c *astCompiler) patchUserCalls() {
	for _, patch := range c.userCallPatches {
		fn, ok := c.bc.Funcs[patch.callee]
		if !ok || fn.EntryPC < 0 {
			if c.err == nil {
				c.err = fmt.Errorf("unresolved user function: %s", patch.callee)
			}
			continue
		}
		c.bc.Code[patch.instruction].A = fn.EntryPC
	}
}

// addConst adds a constant to the pool and returns its ID.
func (c *astCompiler) addConst(v interp.Value) ConstID {
	id := ConstID(len(c.bc.Consts))
	c.bc.Consts = append(c.bc.Consts, constFromValue(v))
	return id
}

// pushScope enters a new local variable scope.
func (c *astCompiler) pushScope() {
	c.localScopes = append(c.localScopes, make(map[string]VarID))
}

// popScope exits the current local variable scope.
func (c *astCompiler) popScope() {
	if len(c.localScopes) > 0 {
		c.localScopes = c.localScopes[:len(c.localScopes)-1]
	}
}

// resolveVar resolves a variable name to (slotID, isGlobal).
func (c *astCompiler) resolveVar(name string) (VarID, bool) {
	// Check local scopes (innermost first)
	for i := len(c.localScopes) - 1; i >= 0; i-- {
		if id, ok := c.localScopes[i][name]; ok {
			return id, false
		}
	}
	// Check globals
	if id, ok := c.bc.GlobalSlots[name]; ok {
		return id, true
	}
	// Unknown variable — check if it's a known MQL constant, enum, or series name
	// that should have been resolved earlier. If not, it's likely a typo.
	if interp.IsMQLConstant(name) || isSeriesName(name) {
		// These should have been resolved in compileExpr, but if we get here
		// (e.g. used as a variable target), register as global to avoid crash.
		id := VarID(len(c.bc.GlobalSlots))
		c.bc.GlobalSlots[name] = id
		return id, true
	}
	if _, ok := c.bc.Enums[name]; ok {
		id := VarID(len(c.bc.GlobalSlots))
		c.bc.GlobalSlots[name] = id
		return id, true
	}
	// MQL4 and Python allow implicit variable declaration (assign without declaring).
	// Record as warning + blind spot, but still register to avoid crash.
	if c.bc.Version == "mql4" || c.bc.Version == "python" {
		c.bc.Coverage.AddBlindSpot("implicit variable: " + name)
		id := VarID(len(c.bc.GlobalSlots))
		c.bc.GlobalSlots[name] = id
		return id, true
	}
	// MQL5 requires explicit declaration — this is likely a typo.
	if c.err == nil {
		c.err = fmt.Errorf("unknown variable: %s (not declared, not a constant, not an enum)", name)
	}
	id := VarID(len(c.bc.GlobalSlots))
	c.bc.GlobalSlots[name] = id
	return id, true
}

func isEventFunction(name string) bool {
	switch name {
	case "OnInit", "OnTick", "OnBar", "OnTimer", "OnTrade", "OnTradeTransaction", "OnBookEvent", "OnDeinit", "start":
		return true
	}
	return false
}

// compileEventBody compiles an event handler body, tracking local variable slots.
// Event handlers don't go through OP_CALL_USER, so they need their own local space.
// The entry PC is the instruction before the first body instruction (the ENTER_* marker).
func (c *astCompiler) compileEventBody(body []interp.Statement) {
	entryPC := int32(len(c.bc.Code)) - 1 // the ENTER_* marker just emitted
	c.nextLocalSlot = 0
	c.pushScope()
	c.compileStmts(body)
	c.popScope()
	c.bc.EventLocals[entryPC] = c.nextLocalSlot
	c.emit(OP_RETURN, 0, 0, 0)
}

// ── Function compilation ─────────────────────────────────────────────

func (c *astCompiler) compileUserFuncBody(name string, fn *interp.FuncDef) {
	// Update EntryPC to the actual function body start.
	// Pass 1 emitted OP_ENTER_FUNC markers at the original EntryPC positions,
	// but the bodies are compiled after ALL markers (Pass 2). So entryPC+1
	// (used by executeCallUser) would land on the next function's marker,
	// not this function's body. Fix: point EntryPC at the real body start
	// and change executeCallUser to jump to entryPC (no +1).
	bodyStart := int32(len(c.bc.Code))
	entry := c.bc.Funcs[name]
	entry.EntryPC = bodyStart
	c.currentFunc = &FuncEntry{
		Name:      name,
		EntryPC:   bodyStart,
		NumParams: len(fn.Params),
		NumLocals: len(fn.Params),
	}
	for _, p := range fn.Params {
		c.currentFunc.ParamName = append(c.currentFunc.ParamName, p.Name)
	}

	c.nextLocalSlot = len(fn.Params)
	c.pushScope()
	for i, p := range fn.Params {
		c.localScopes[len(c.localScopes)-1][p.Name] = VarID(i)
	}
	c.compileStmts(fn.Body)
	c.popScope()

	// Update NumLocals with the actual count (params + locals declared in body)
	c.currentFunc.NumLocals = c.nextLocalSlot
	c.bc.Funcs[name] = *c.currentFunc

	// Ensure function ends with RETURN: push None first so caller gets a value
	c.emit(OP_PUSH_CONST, int32(c.addConst(interp.NoneVal())), 0, 0)
	c.emit(OP_RETURN, 0, 0, 0)
	c.currentFunc = nil
}

// ── Statement compilation ────────────────────────────────────────────

func (c *astCompiler) compileStmts(stmts []interp.Statement) {
	for _, s := range stmts {
		c.compileStmt(&s)
	}
}

func (c *astCompiler) compileStmt(s *interp.Statement) {
	if s == nil {
		return
	}
	switch s.Kind {
	case interp.StmtExpr:
		c.compileExpr(s.Expr)
		// Only pop if the expression leaves a value on the stack.
		// Stack-neutral expressions (assignment, declaration, update) don't.
		if !isStackNeutral(s.Expr) {
			c.emit(OP_POP, 0, 0, 0)
		}

	case interp.StmtIf:
		c.compileIf(s)

	case interp.StmtFor:
		c.compileFor(s)

	case interp.StmtWhile:
		c.compileWhile(s)

	case interp.StmtDoWhile:
		c.compileDoWhile(s)

	case interp.StmtReturn:
		if s.Expr != nil {
			c.compileExpr(s.Expr)
		} else {
			c.emit(OP_PUSH_CONST, int32(c.addConst(interp.NoneVal())), 0, 0)
		}
		c.emit(OP_RETURN, 0, 0, 0)

	case interp.StmtBlock:
		c.pushScope()
		c.compileStmts(s.Body)
		c.popScope()

	case interp.StmtSwitch:
		c.compileSwitch(s)

	case interp.StmtBreak:
		if len(c.loopStack) > 0 {
			jmp := c.emitJump(OP_JMP, 0)
			c.loopStack[len(c.loopStack)-1].breakJumps = append(c.loopStack[len(c.loopStack)-1].breakJumps, jmp)
		}

	case interp.StmtContinue:
		if len(c.loopStack) > 0 {
			jmp := c.emitJump(OP_JMP, 0)
			c.loopStack[len(c.loopStack)-1].continueJumps = append(c.loopStack[len(c.loopStack)-1].continueJumps, jmp)
		}
	}
}

func (c *astCompiler) compileIf(s *interp.Statement) {
	// Compile condition
	c.compileExpr(s.Cond)
	jmpFalse := c.emitJump(OP_JMP_IF_FALSE, 0)

	// Then body
	c.compileStmts(s.Body)

	if len(s.ElseBody) > 0 {
		jmpEnd := c.emitJump(OP_JMP, 0)
		c.patchJump(jmpFalse)
		c.compileStmts(s.ElseBody)
		c.patchJump(jmpEnd)
	} else {
		c.patchJump(jmpFalse)
	}
}
