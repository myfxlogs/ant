package mql2go

import (
	"context"
	"fmt"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/tools/mql2go/interp"
)

func TestVM_Audit_StackUnderflowIsError(t *testing.T) {
	vm := NewVM(&Bytecode{
		Code:        []Instruction{{Op: OP_POP}, {Op: OP_HALT}},
		GlobalSlots: map[string]VarID{},
		Funcs:       map[string]FuncEntry{},
		Builtins:    map[string]BuiltinID{},
	})
	if err := vm.runEvent(context.Background(), 0); err == nil {
		t.Fatal("stack underflow completed without an error")
	}
}

// ── VM-RUNTIME-FAILCLOSED-2 behavior tests ───────────────────────────

// TestVM_Audit_DivisionByZeroStopsExecution verifies that integer division
// by zero sets a stack error and stops execution (subsequent instruction
// g_after is not executed).
func TestVM_Audit_DivisionByZeroStopsExecution(t *testing.T) {
	const source = `
int g_result = -1;
int g_after = -1;
int OnInit() { return 0; }
void OnTick() {
    int z = 0;
    g_result = 10 / z;
    g_after = 42;
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err == nil {
		t.Fatal("division by zero should cause an error (fail-closed), got nil")
	}
	after, _ := runner.GetGlobal("g_after")
	if after.ToInt() != -1 {
		t.Fatalf("g_after = %d, want -1 (execution should stop after div-by-zero, g_after should not be set)", after.ToInt())
	}
}

// TestVM_Audit_DecimalDivisionByZeroStopsExecution verifies that decimal
// division by zero sets a stack error and stops execution.
func TestVM_Audit_DecimalDivisionByZeroStopsExecution(t *testing.T) {
	const source = `
double g_result = -1;
int g_after = -1;
int OnInit() { return 0; }
void OnTick() {
    double z = 0.0;
    g_result = 10.0 / z;
    g_after = 42;
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err == nil {
		t.Fatal("decimal division by zero should cause an error (fail-closed), got nil")
	}
	after, _ := runner.GetGlobal("g_after")
	if after.ToInt() != -1 {
		t.Fatalf("g_after = %d, want -1 (execution should stop after decimal div-by-zero)", after.ToInt())
	}
}

// TestVM_Audit_ModuloByZeroStopsExecution verifies that integer modulo by
// zero sets a stack error and stops execution.
func TestVM_Audit_ModuloByZeroStopsExecution(t *testing.T) {
	const source = `
int g_result = -1;
int g_after = -1;
int OnInit() { return 0; }
void OnTick() {
    int z = 0;
    g_result = 10 % z;
    g_after = 42;
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err == nil {
		t.Fatal("modulo by zero should cause an error (fail-closed), got nil")
	}
	after, _ := runner.GetGlobal("g_after")
	if after.ToInt() != -1 {
		t.Fatalf("g_after = %d, want -1 (execution should stop after mod-by-zero)", after.ToInt())
	}
}

// TestVM_Audit_OpDupUnderflowStopsExecution verifies that OP_DUP on an empty
// stack sets a stack error and stops execution. We use a manually crafted
// bytecode to trigger this — the compiler normally never emits OP_DUP on
// an empty stack, but corrupt bytecode could.
func TestVM_Audit_OpDupUnderflowStopsExecution(t *testing.T) {
	bc := &Bytecode{
		OnTick: 0,
		Code:   []Instruction{{Op: OP_DUP}, {Op: OP_HALT}},
		Consts: []ConstValue{},
	}
	vm := NewVM(bc)
	vm.stack = vm.stack[:0] // ensure empty stack
	err := vm.runEvent(context.Background(), 0)
	if err == nil {
		t.Fatal("OP_DUP on empty stack should cause an error, got nil")
	}
}

// TestVM_Audit_OpSwapUnderflowStopsExecution verifies that OP_SWAP with <2
// stack elements sets a stack error and stops execution.
func TestVM_Audit_OpSwapUnderflowStopsExecution(t *testing.T) {
	bc := &Bytecode{
		OnTick: 0,
		Code:   []Instruction{{Op: OP_SWAP}, {Op: OP_HALT}},
		Consts: []ConstValue{},
	}
	vm := NewVM(bc)
	vm.stack = vm.stack[:0]
	vm.push(interp.IntVal(1)) // only 1 element — SWAP needs 2
	err := vm.runEvent(context.Background(), 0)
	if err == nil {
		t.Fatal("OP_SWAP with 1 element should cause an error, got nil")
	}
}

// TestVM_Audit_PushVarOutOfRangeStopsExecution verifies that OP_PUSH_VAR with
// an out-of-range local slot sets a stack error and stops execution.
func TestVM_Audit_PushVarOutOfRangeStopsExecution(t *testing.T) {
	bc := &Bytecode{
		OnTick: 0,
		Code:   []Instruction{{Op: OP_PUSH_VAR, A: 99}, {Op: OP_HALT}},
		Consts: []ConstValue{},
	}
	vm := NewVM(bc)
	vm.locals = vm.locals[:0] // no local slots
	err := vm.runEvent(context.Background(), 0)
	if err == nil {
		t.Fatal("OP_PUSH_VAR with out-of-range slot should cause an error, got nil")
	}
}

// TestVM_Audit_PushGlobalOutOfRangeStopsExecution verifies that OP_PUSH_GLOBAL
// with an out-of-range global slot sets a stack error and stops execution.
// VM-RUNTIME-FAILCLOSED-3: bc.GlobalSlots must have ≥100 entries so the
// validator passes (it checks A < len(GlobalSlots)), but vm.globals must be
// empty so the runtime execute branch fires the setStackError path.
func TestVM_Audit_PushGlobalOutOfRangeStopsExecution(t *testing.T) {
	globalSlots := make(map[string]VarID, 100)
	for i := 0; i < 100; i++ {
		globalSlots[fmt.Sprintf("g%d", i)] = VarID(i)
	}
	bc := &Bytecode{
		OnTick:      0,
		Code:        []Instruction{{Op: OP_PUSH_GLOBAL, A: 99}, {Op: OP_HALT}},
		Consts:      []ConstValue{},
		GlobalSlots: globalSlots,
	}
	vm := NewVM(bc)
	// Set globals to a non-nil but shorter slice (50 < 99) so initGlobals
	// is NOT called (it only runs when globals==nil), and the runtime
	// OP_PUSH_GLOBAL branch fires the setStackError path.
	vm.globals = make([]interp.Value, 50)
	err := vm.runEvent(context.Background(), 0)
	if err == nil {
		t.Fatal("OP_PUSH_GLOBAL with out-of-range runtime slot should cause an error, got nil")
	}
}

// ── VM-RUNTIME-FAILCLOSED-3 behavior tests ───────────────────────────

// TestVM_Audit_FaultOnLastInstructionNotSwallowed verifies that a fault
// triggered by the LAST instruction (no OP_HALT after it) is NOT swallowed
// by the pc==len(Code) success return. VM-RUNTIME-FAILCLOSED-3: runLoop
// must check stackError/fatalError BEFORE the code-end success return.
func TestVM_Audit_FaultOnLastInstructionNotSwallowed(t *testing.T) {
	// Bytecode: OP_DUP on empty stack as the LAST instruction (no OP_HALT).
	// Before the fix, runLoop would see pc==len(Code) and return nil,
	// swallowing the stack error. After the fix, the stackError check
	// fires first and returns an error.
	bc := &Bytecode{
		OnTick: 0,
		Code:   []Instruction{{Op: OP_DUP}}, // last instruction = fault
		Consts: []ConstValue{},
	}
	vm := NewVM(bc)
	vm.stack = vm.stack[:0] // empty stack → OP_DUP underflow
	err := vm.runEvent(context.Background(), 0)
	if err == nil {
		t.Fatal("OP_DUP underflow on last instruction should cause an error, got nil (fault swallowed by code-end return)")
	}
}

// TestVM_Audit_FloorDivByZeroStopsExecution verifies that integer floor
// division (OP_FLOOR_DIV, Python's // operator) by zero sets a stack error
// and stops execution. We construct bytecode directly because MQL's /
// compiles to OP_DIV, not OP_FLOOR_DIV — using MQL source would test the
// wrong opcode and never exercise floorDiv.
func TestVM_Audit_FloorDivByZeroStopsExecution(t *testing.T) {
	bc := &Bytecode{
		OnTick: 0,
		Code: []Instruction{
			{Op: OP_PUSH_CONST, A: 0}, // push 10
			{Op: OP_PUSH_CONST, A: 1}, // push 0
			{Op: OP_FLOOR_DIV},        // 10 // 0 → setStackError
			{Op: OP_HALT},             // should never reach
		},
		Consts: []ConstValue{
			{Kind: interp.ValInt, Int: 10},
			{Kind: interp.ValInt, Int: 0},
		},
	}
	vm := NewVM(bc)
	err := vm.runEvent(context.Background(), 0)
	if err == nil {
		t.Fatal("OP_FLOOR_DIV by zero should cause an error (fail-closed), got nil")
	}
}

// TestVM_Audit_DecimalModuloByZeroStopsExecution verifies that decimal
// modulo by zero sets a stack error and stops execution.
func TestVM_Audit_DecimalModuloByZeroStopsExecution(t *testing.T) {
	const source = `
double g_result = -1;
int g_after = -1;
int OnInit() { return 0; }
void OnTick() {
    double z = 0.0;
    g_result = 10.0 % z;
    g_after = 42;
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err == nil {
		t.Fatal("decimal modulo by zero should cause an error (fail-closed), got nil")
	}
	after, _ := runner.GetGlobal("g_after")
	if after.ToInt() != -1 {
		t.Fatalf("g_after = %d, want -1 (execution should stop after decimal mod-by-zero)", after.ToInt())
	}
}

// TestVM_Audit_StoreVarOutOfRangeStopsExecution verifies that OP_STORE_VAR
// with an out-of-range local slot sets a stack error and stops execution.
func TestVM_Audit_StoreVarOutOfRangeStopsExecution(t *testing.T) {
	bc := &Bytecode{
		OnTick: 0,
		// Push a value, then try to store to out-of-range local slot 99.
		Code:   []Instruction{{Op: OP_PUSH_CONST, A: 0}, {Op: OP_STORE_VAR, A: 99}, {Op: OP_HALT}},
		Consts: []ConstValue{{Kind: interp.ValInt, Int: 42}},
	}
	vm := NewVM(bc)
	vm.locals = nil // no local slots
	err := vm.runEvent(context.Background(), 0)
	if err == nil {
		t.Fatal("OP_STORE_VAR with out-of-range slot should cause an error, got nil")
	}
}

// TestVM_Audit_StoreGlobalOutOfRangeStopsExecution verifies that
// OP_STORE_GLOBAL with an out-of-range global slot sets a stack error.
func TestVM_Audit_StoreGlobalOutOfRangeStopsExecution(t *testing.T) {
	globalSlots := make(map[string]VarID, 100)
	for i := 0; i < 100; i++ {
		globalSlots[fmt.Sprintf("g%d", i)] = VarID(i)
	}
	bc := &Bytecode{
		OnTick: 0,
		// Push a value, then try to store to out-of-range global slot 99.
		Code:        []Instruction{{Op: OP_PUSH_CONST, A: 0}, {Op: OP_STORE_GLOBAL, A: 99}, {Op: OP_HALT}},
		Consts:      []ConstValue{{Kind: interp.ValInt, Int: 42}},
		GlobalSlots: globalSlots,
	}
	vm := NewVM(bc)
	vm.globals = make([]interp.Value, 50) // non-nil but shorter than 99
	err := vm.runEvent(context.Background(), 0)
	if err == nil {
		t.Fatal("OP_STORE_GLOBAL with out-of-range runtime slot should cause an error, got nil")
	}
}
