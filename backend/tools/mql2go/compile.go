package mql2go

import (
	"fmt"
	"sort"

	"alphaforge/tools/mql2go/interp"
)

// CompileAST takes a parsed IR (which is our AST) and compiles it to Bytecode.
// This is the one-time compilation step (~300ms for typical EA).
// The resulting Bytecode can be executed repeatedly by the VM without recompilation.
func CompileAST(ir *interp.IR) (*Bytecode, error) {
	c := &astCompiler{
		bc: &Bytecode{
			Consts:             make([]ConstValue, 0),
			Code:               make([]Instruction, 0),
			GlobalSlots:        make(map[string]VarID),
			Funcs:              make(map[string]FuncEntry),
			Builtins:           make(map[string]BuiltinID),
			Params:             ir.Params,
			Version:            ir.Version,
			Enums:              ir.Enums,
			ClassTypes:         ir.ClassTypes, // VM-COMPILER-SEMANTICS-1
			Coverage:           &CoverageReport{},
			OnInit:             -1,
			OnBar:              -1,
			OnTick:             -1,
			OnTrade:            -1,
			OnTimer:            -1,
			OnDeinit:           -1,
			OnTradeTransaction: -1,
			OnBookEvent:        -1,
			EventLocals:        make(map[int32]int),
		},
		ir:          ir,
		localScopes: []map[string]VarID{},
	}

	// Register builtin function IDs
	c.registerBuiltins()

	// Allocate global variable slots (globals + params)
	for _, p := range ir.Params {
		c.bc.GlobalSlots[p.Name] = VarID(len(c.bc.GlobalSlots))
	}
	for _, g := range ir.Globals {
		c.bc.GlobalSlots[g.Name] = VarID(len(c.bc.GlobalSlots))
	}
	c.bc.GlobalDecls = ir.Globals

	// Compile user-defined functions first (so we know their entry points).
	// Two-pass: first register all entry PCs so forward references resolve,
	// then compile bodies. Without this, non-deterministic map iteration
	// (ir.Funcs is a map) causes intermittent failures where a caller is
	// compiled before its callee is registered in bc.Funcs, making the
	// callee fall through to "unknown function" and silently return 0.
	userFuncNames := make([]string, 0, len(ir.Funcs))
	for name, fn := range ir.Funcs {
		if isEventFunction(name) {
			continue
		}
		entryPC := int32(len(c.bc.Code))
		c.bc.Funcs[name] = FuncEntry{
			Name:      name,
			EntryPC:   entryPC,
			NumParams: len(fn.Params),
			NumLocals: len(fn.Params),
		}
		c.emit(OP_ENTER_FUNC, int32(len(fn.Params)), 0, 0)
		userFuncNames = append(userFuncNames, name)
	}

	// BT-FUNC-ENTRYPC-FWD: deterministic layout (map iteration is non-deterministic).
	sort.Strings(userFuncNames)

	for _, name := range userFuncNames {
		fn := ir.Funcs[name]
		c.compileUserFuncBody(name, fn)
	}

	// Compile event handlers
	// Global initializers are prepended to OnInit (or compiled as a standalone
	// preamble if OnInit doesn't exist) so they run before any event handler.
	globalInitPC := int32(len(c.bc.Code))
	c.emit(OP_ENTER_ONINIT, 0, 0, 0)
	for _, g := range ir.Globals {
		if g.InitVal != nil {
			c.compileExpr(g.InitVal)
			slot := c.bc.GlobalSlots[g.Name]
			c.emit(OP_STORE_GLOBAL, int32(slot), 0, 0)
		}
	}
	if len(ir.OnInit) > 0 {
		c.bc.OnInit = globalInitPC
		c.compileEventBody(ir.OnInit)
	} else {
		c.bc.OnInit = globalInitPC
		c.emit(OP_RETURN, 0, 0, 0)
	}
	if len(ir.OnBar) > 0 {
		c.bc.OnBar = int32(len(c.bc.Code))
		c.emit(OP_ENTER_ONBAR, 0, 0, 0)
		c.compileEventBody(ir.OnBar)
	}
	if len(ir.OnTick) > 0 {
		c.bc.OnTick = int32(len(c.bc.Code))
		c.emit(OP_ENTER_ONTICK, 0, 0, 0)
		c.compileEventBody(ir.OnTick)
	}
	c.compileOptionalEvents(ir)

	// Halt instruction at end
	c.emit(OP_HALT, 0, 0, 0)

	// Patch forward jumps (placeholder targets are negative indices)
	c.patchJumps()

	// BT-FUNC-ENTRYPC-FWD: patch user-call placeholders after all bodies compiled.
	if err := c.patchUserCalls(); err != nil {
		if c.err == nil {
			c.err = err
		}
	}

	// Return compile error if any (e.g. unknown constant)
	if c.err != nil {
		return nil, c.err
	}

	return c.bc, nil
}

// compileOptionalEvents compiles OnTrade/OnTimer/OnDeinit/OnTradeTransaction/OnBookEvent
// event handlers. Extracted from CompileAST to reduce function length (funlen).
func (c *astCompiler) compileOptionalEvents(ir *interp.IR) {
	if len(ir.OnTrade) > 0 {
		c.bc.OnTrade = int32(len(c.bc.Code))
		c.emit(OP_ENTER_ONTRADE, 0, 0, 0)
		c.compileEventBody(ir.OnTrade)
	}
	if len(ir.OnTimer) > 0 {
		c.bc.OnTimer = int32(len(c.bc.Code))
		c.emit(OP_ENTER_ONTIMER, 0, 0, 0)
		c.compileEventBody(ir.OnTimer)
	}
	if len(ir.OnDeinit) > 0 {
		c.bc.OnDeinit = int32(len(c.bc.Code))
		c.emit(OP_ENTER_ONDEINIT, 0, 0, 0)
		c.compileEventBody(ir.OnDeinit)
	}
	if len(ir.OnTradeTransaction) > 0 {
		c.bc.OnTradeTransaction = int32(len(c.bc.Code))
		c.emit(OP_ENTER_ONTRADETRANSACTION, 0, 0, 0)
		c.compileEventBody(ir.OnTradeTransaction)
	}
	if len(ir.OnBookEvent) > 0 {
		c.bc.OnBookEvent = int32(len(c.bc.Code))
		c.emit(OP_ENTER_ONBOOKEVENT, 0, 0, 0)
		c.compileEventBody(ir.OnBookEvent)
	}
}

type loopContext struct {
	breakJumps    []int32
	continueJumps []int32
}

type astCompiler struct {
	bc              *Bytecode
	ir              *interp.IR
	localScopes     []map[string]VarID // scope stack for local variables
	staticScopes    []map[string]VarID // VM-STATIC-LOCAL-1: per-scope name → global slot for static locals (parallel to localScopes)
	staticSeq       int                // VM-STATIC-LOCAL-1: monotonic mangling sequence for __static_N/__sinit_N
	currentFunc     *FuncEntry
	nextLocalSlot   int             // next available local slot in current function
	loopStack       []*loopContext  // stack of loop contexts for break/continue
	err             error           // first compile error encountered (e.g. unknown constant)
	userCallPatches []userCallPatch // BT-FUNC-ENTRYPC-FWD: pending user-call relocations
}

// userCallPatch records a pending OP_CALL_USER instruction that needs its
// operand A patched to the callee's final EntryPC after all bodies are compiled.
// BT-FUNC-ENTRYPC-FWD: fixes forward references where caller body is compiled
// before callee body, leaving EntryPC as a stale marker PC.
type userCallPatch struct {
	instruction int32  // index of OP_CALL_USER in bc.Code
	callee      string // function name to patch
}

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

// patchUserCalls resolves all pending user-call relocations.
// BT-FUNC-ENTRYPC-FWD: called after all user function bodies are compiled,
// so all EntryPCs are final. This fixes forward references where a caller's
// body is compiled before the callee's body, leaving the OP_CALL_USER operand
// as a stale marker PC (-1 placeholder).
func (c *astCompiler) patchUserCalls() error {
	for _, p := range c.userCallPatches {
		fn, ok := c.bc.Funcs[p.callee]
		if !ok {
			return fmt.Errorf("patchUserCalls: unknown function %s", p.callee)
		}
		if fn.EntryPC < 0 {
			return fmt.Errorf("patchUserCalls: callee %s has invalid EntryPC %d", p.callee, fn.EntryPC)
		}
		c.bc.Code[p.instruction].A = fn.EntryPC
	}
	return nil
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
	// VM-STATIC-LOCAL-1: staticScopes stays index-aligned with localScopes
	// (these two helpers are the only places the stack is resized — verified
	// across all call sites), so resolveVar can probe both per scope level.
	c.staticScopes = append(c.staticScopes, make(map[string]VarID))
}

// popScope exits the current local variable scope.
func (c *astCompiler) popScope() {
	if len(c.localScopes) > 0 {
		c.localScopes = c.localScopes[:len(c.localScopes)-1]
		c.staticScopes = c.staticScopes[:len(c.staticScopes)-1]
	}
}

// resolveVar resolves a variable name in a READ position to (slotID, isGlobal).
// VM-IMPLICIT-VAR-READ-1: for MQL4/MQL5 an unknown identifier in a read
// position is a compile error — silently registering a zero-valued global
// let typos and unsupported constant aliases (e.g. SYMBOL_TICK_VALUE)
// evaluate as 0. Python keeps its QS-1.3 implicit-read semantics
// (undeclared read → never-written global None).
func (c *astCompiler) resolveVar(name string) (VarID, bool) {
	// Check local scopes (innermost first); static locals bind to their
	// mangled global slot at the same scope level (VM-STATIC-LOCAL-1) — a
	// plain local of the same name shadows the static (probed first).
	for i := len(c.localScopes) - 1; i >= 0; i-- {
		if id, ok := c.localScopes[i][name]; ok {
			return id, false
		}
		if id, ok := c.staticScopes[i][name]; ok {
			return id, true
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
	// Python: undeclared read resolves to the never-written global (None) —
	// QS-1.3 cross-function isolation semantics, covered by
	// compile_py_locals_test.go.
	if c.bc.Version == "python" {
		c.bc.Coverage.AddBlindSpot("implicit variable: " + name)
		id := VarID(len(c.bc.GlobalSlots))
		c.bc.GlobalSlots[name] = id
		return id, true
	}
	// MQL4/MQL5: read of an undeclared name is a compile error (same class as
	// the existing ExprConst unknown-constant check).
	if c.err == nil {
		c.err = fmt.Errorf("unknown variable: %s (not declared, not a constant, not an enum)", name)
	}
	id := VarID(len(c.bc.GlobalSlots))
	c.bc.GlobalSlots[name] = id
	return id, true
}

// resolveVarWrite resolves a variable name in a WRITE position
// (assignment target, declaration, array store). MQL4 and Python keep the
// implicit-declaration shim: assigning to an undeclared name registers it.
func (c *astCompiler) resolveVarWrite(name string) (VarID, bool) {
	for i := len(c.localScopes) - 1; i >= 0; i-- {
		if id, ok := c.localScopes[i][name]; ok {
			return id, false
		}
		if id, ok := c.staticScopes[i][name]; ok {
			return id, true
		}
	}
	if id, ok := c.bc.GlobalSlots[name]; ok {
		return id, true
	}
	if interp.IsMQLConstant(name) || isSeriesName(name) {
		id := VarID(len(c.bc.GlobalSlots))
		c.bc.GlobalSlots[name] = id
		return id, true
	}
	if _, ok := c.bc.Enums[name]; ok {
		id := VarID(len(c.bc.GlobalSlots))
		c.bc.GlobalSlots[name] = id
		return id, true
	}
	if c.bc.Version == "mql4" || c.bc.Version == "python" {
		c.bc.Coverage.AddBlindSpot("implicit variable: " + name)
		id := VarID(len(c.bc.GlobalSlots))
		c.bc.GlobalSlots[name] = id
		return id, true
	}
	if c.err == nil {
		c.err = fmt.Errorf("unknown variable: %s (not declared, not a constant, not an enum)", name)
	}
	id := VarID(len(c.bc.GlobalSlots))
	c.bc.GlobalSlots[name] = id
	return id, true
}

// isDeclaredGlobal reports whether name was declared at module level
// (self fields + top-level assignments collected into ir.Globals).
// Names merely registered into GlobalSlots by implicit reads do NOT count —
// they must not poison later function-scope assignments (QS-1.3).
func (c *astCompiler) isDeclaredGlobal(name string) bool {
	for _, g := range c.bc.GlobalDecls {
		if g.Name == name {
			return true
		}
	}
	return false
}

// resolveAssignTarget resolves the store slot for an assignment target.
// QS-1.3: inside a Python function/event, assignment to an undeclared name
// declares a function-local slot instead of leaking into GlobalSlots.
// Declared globals (self fields, module-level assignments) stay global.
func (c *astCompiler) resolveAssignTarget(name string) (VarID, bool) {
	if c.bc.Version == "python" && len(c.localScopes) > 0 {
		for i := len(c.localScopes) - 1; i >= 0; i-- {
			if id, ok := c.localScopes[i][name]; ok {
				return id, false
			}
		}
		if c.isDeclaredGlobal(name) {
			return c.bc.GlobalSlots[name], true
		}
		// Python has function scope, not block scope: allocate in the
		// function base scope (index 0 — the scope pushed by
		// compileUserFuncBody/compileEventBody), so names assigned inside
		// for-loop bodies survive the loop scope's popScope.
		scope := c.localScopes[0]
		scope[name] = VarID(c.nextLocalSlot)
		c.nextLocalSlot++
		return scope[name], false
	}
	return c.resolveVarWrite(name)
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

	// Then body — own scope (VM-BLOCK-SCOPE-1): the CST layer flattens the
	// branch's compound_statement into Body, so without this wrap its
	// declarations leaked into the enclosing scope and stayed readable after
	// the branch (real MQL: undeclared).
	c.pushScope()
	c.compileStmts(s.Body)
	c.popScope()

	if len(s.ElseBody) > 0 {
		jmpEnd := c.emitJump(OP_JMP, 0)
		c.patchJump(jmpFalse)
		c.pushScope()
		c.compileStmts(s.ElseBody)
		c.popScope()
		c.patchJump(jmpEnd)
	} else {
		c.patchJump(jmpFalse)
	}
}
