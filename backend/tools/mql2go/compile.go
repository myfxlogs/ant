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
	if ir == nil {
		return nil, fmt.Errorf("compile AST: nil IR")
	}
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
			ClassTypes:         ir.ClassTypes,
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

	// Allocate global variable slots (globals + params).
	for _, p := range ir.Params {
		if _, exists := c.bc.GlobalSlots[p.Name]; exists {
			return nil, fmt.Errorf("duplicate global/parameter name: %s", p.Name)
		}
		c.bc.GlobalSlots[p.Name] = VarID(len(c.bc.GlobalSlots))
	}
	for _, g := range ir.Globals {
		if _, exists := c.bc.GlobalSlots[g.Name]; exists {
			return nil, fmt.Errorf("duplicate global/parameter name: %s", g.Name)
		}
		c.bc.GlobalSlots[g.Name] = VarID(len(c.bc.GlobalSlots))
	}
	c.bc.GlobalDecls = ir.Globals

	// Compile user-defined functions first (so we know their entry points).
	// Two-pass: first register all entry PCs so forward references resolve,
	// then compile bodies. Calls are patched after every body has its final
	// address because a callee's body is emitted during Pass 2.
	// Sort names so identical IR produces identical bytecode layout.
	userFuncNames := make([]string, 0, len(ir.Funcs))
	for name := range ir.Funcs {
		if !isEventFunction(name) {
			userFuncNames = append(userFuncNames, name)
		}
	}
	sort.Strings(userFuncNames)
	for _, name := range userFuncNames {
		fn := ir.Funcs[name]
		if fn == nil {
			return nil, fmt.Errorf("compile AST: nil function definition: %s", name)
		}
		entryPC := int32(len(c.bc.Code))
		c.bc.Funcs[name] = FuncEntry{
			Name:      name,
			EntryPC:   entryPC,
			NumParams: len(fn.Params),
			NumLocals: len(fn.Params),
		}
		c.emit(OP_ENTER_FUNC, int32(len(fn.Params)), 0, 0)
	}

	for _, name := range userFuncNames {
		c.compileUserFuncBody(name, ir.Funcs[name])
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

	// Halt instruction at end
	c.emit(OP_HALT, 0, 0, 0)

	// Patch user calls after all function bodies have their final addresses.
	c.patchUserCalls()

	// Patch forward jumps (placeholder targets are negative indices)
	c.patchJumps()

	// Return compile error if any (e.g. unknown constant)
	if c.err != nil {
		return nil, c.err
	}

	return c.bc, nil
}

type loopContext struct {
	breakJumps    []int32
	continueJumps []int32
}

type userCallPatch struct {
	instruction int32
	callee      string
}

type astCompiler struct {
	bc              *Bytecode
	ir              *interp.IR
	localScopes     []map[string]VarID // scope stack for local variables
	currentFunc     *FuncEntry
	nextLocalSlot   int            // next available local slot in current function
	loopStack       []*loopContext // stack of loop contexts for break/continue
	userCallPatches []userCallPatch
	err             error // first compile error encountered (e.g. unknown constant)
}
