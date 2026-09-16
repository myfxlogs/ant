package mql2go

// vm_audit_test.go — VM-RUNTIME-FAILCLOSED-2 behavior tests.
//
// Tests verify that silent arithmetic/stack/slot failures now set fatalError
// via setStackError, which runLoop's top-of-loop check catches to stop
// execution (fail-closed). All tests assert BOTH:
//   - the error is returned (not nil)
//   - subsequent code does NOT execute (g_after stays 0 for MQL tests)
//
// Adversarial proofs (S6): 4 mutations, each RED→restore→GREEN.

import (
	"context"
	"strings"
	"testing"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// ── MQL-level tests (compile + run through VM) ─────────────────────

// TestVM_Audit_DivisionByZeroStopsExecution: int division by zero sets
// fatalError → runLoop stops → g_after assignment never runs.
func TestVM_Audit_DivisionByZeroStopsExecution(t *testing.T) {
	src := `
int g_after = 0;

int OnInit() { return 0; }

void OnBar()
{
    int x = 10 / 0;
    g_after = 42;
}`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	vm := vmRunner.vm
	vm.SetContext(&tsTestContext{bars: sdk.BarsToSlice(makeFailClosedBars(3))})
	if err := vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	err = vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("OnBar should fail with division by zero error, got nil")
	}
	if !strings.Contains(err.Error(), "division by zero") {
		t.Fatalf("error should contain 'division by zero', got: %v", err)
	}
	after := getGlobalInt(t, vmRunner, "g_after")
	if after != 0 {
		t.Fatalf("g_after = %d after OnBar, want 0 (assignment after div-by-zero must not execute)", after)
	}
}

// TestVM_Audit_DecimalDivisionByZeroStopsExecution: double division by zero
// sets fatalError → runLoop stops → g_after stays 0.
func TestVM_Audit_DecimalDivisionByZeroStopsExecution(t *testing.T) {
	src := `
int g_after = 0;

int OnInit() { return 0; }

void OnBar()
{
    double x = 10.0 / 0.0;
    g_after = 42;
}`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	vm := vmRunner.vm
	vm.SetContext(&tsTestContext{bars: sdk.BarsToSlice(makeFailClosedBars(3))})
	if err := vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	err = vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("OnBar should fail with decimal division by zero error, got nil")
	}
	if !strings.Contains(err.Error(), "division by zero") {
		t.Fatalf("error should contain 'division by zero', got: %v", err)
	}
	after := getGlobalInt(t, vmRunner, "g_after")
	if after != 0 {
		t.Fatalf("g_after = %d after OnBar, want 0 (assignment after div-by-zero must not execute)", after)
	}
}

// TestVM_Audit_ModuloByZeroStopsExecution: int modulo by zero sets
// fatalError → runLoop stops → g_after stays 0.
func TestVM_Audit_ModuloByZeroStopsExecution(t *testing.T) {
	src := `
int g_after = 0;

int OnInit() { return 0; }

void OnBar()
{
    int x = 10 % 0;
    g_after = 42;
}`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	vm := vmRunner.vm
	vm.SetContext(&tsTestContext{bars: sdk.BarsToSlice(makeFailClosedBars(3))})
	if err := vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	err = vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("OnBar should fail with modulo by zero error, got nil")
	}
	if !strings.Contains(err.Error(), "modulo") {
		t.Fatalf("error should contain 'modulo', got: %v", err)
	}
	after := getGlobalInt(t, vmRunner, "g_after")
	if after != 0 {
		t.Fatalf("g_after = %d after OnBar, want 0 (assignment after mod-by-zero must not execute)", after)
	}
}

// ── Direct bytecode tests (construct VM manually) ──────────────────

// TestVM_Audit_OpDupUnderflowStopsExecution: OP_DUP on empty stack sets
// fatalError → runLoop top check stops execution.
func TestVM_Audit_OpDupUnderflowStopsExecution(t *testing.T) {
	vm := NewVM(&Bytecode{
		Code: []Instruction{
			{Op: OP_DUP},
			{Op: OP_HALT},
		},
	})
	err := vm.runLoop(context.Background())
	if err == nil {
		t.Fatal("runLoop should fail with OP_DUP underflow error, got nil")
	}
	if !strings.Contains(err.Error(), "OP_DUP underflow") {
		t.Fatalf("error should contain 'OP_DUP underflow', got: %v", err)
	}
}

// TestVM_Audit_OpSwapUnderflowStopsExecution: OP_SWAP with <2 elements sets
// fatalError → runLoop top check stops execution.
func TestVM_Audit_OpSwapUnderflowStopsExecution(t *testing.T) {
	vm := NewVM(&Bytecode{
		Code: []Instruction{
			{Op: OP_SWAP},
			{Op: OP_HALT},
		},
	})
	vm.push(interp.IntVal(1)) // only 1 element — SWAP needs 2
	err := vm.runLoop(context.Background())
	if err == nil {
		t.Fatal("runLoop should fail with OP_SWAP underflow error, got nil")
	}
	if !strings.Contains(err.Error(), "OP_SWAP underflow") {
		t.Fatalf("error should contain 'OP_SWAP underflow', got: %v", err)
	}
}

// TestVM_Audit_PushVarOutOfRangeStopsExecution: OP_PUSH_VAR with slot 99
// when locals=0 sets fatalError → runLoop stops.
func TestVM_Audit_PushVarOutOfRangeStopsExecution(t *testing.T) {
	vm := NewVM(&Bytecode{
		Code: []Instruction{
			{Op: OP_PUSH_VAR, A: 99},
			{Op: OP_HALT},
		},
	})
	// vm.locals is nil (0 slots) — NewVM doesn't allocate locals
	err := vm.runLoop(context.Background())
	if err == nil {
		t.Fatal("runLoop should fail with OP_PUSH_VAR out of range error, got nil")
	}
	if !strings.Contains(err.Error(), "OP_PUSH_VAR slot 99 out of range") {
		t.Fatalf("error should contain 'OP_PUSH_VAR slot 99 out of range', got: %v", err)
	}
}

// TestVM_Audit_PushGlobalOutOfRangeStopsExecution: OP_PUSH_GLOBAL with slot 99
// when globals=0 sets fatalError → runLoop stops.
func TestVM_Audit_PushGlobalOutOfRangeStopsExecution(t *testing.T) {
	vm := NewVM(&Bytecode{
		Code: []Instruction{
			{Op: OP_PUSH_GLOBAL, A: 99},
			{Op: OP_HALT},
		},
	})
	// vm.globals is nil (0 slots) — NewVM doesn't call initGlobals
	err := vm.runLoop(context.Background())
	if err == nil {
		t.Fatal("runLoop should fail with OP_PUSH_GLOBAL out of range error, got nil")
	}
	if !strings.Contains(err.Error(), "OP_PUSH_GLOBAL slot 99 out of range") {
		t.Fatalf("error should contain 'OP_PUSH_GLOBAL slot 99 out of range', got: %v", err)
	}
}
