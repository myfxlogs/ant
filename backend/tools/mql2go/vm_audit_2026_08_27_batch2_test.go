package mql2go

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"alphaforge/tools/mql2go/interp"
)

// vm_audit_2026_08_27_batch2_test.go — VM-AUDIT-2026-08-27 批次 2 对抗测试.
//
// Tests verify the P2/P3 defense-in-depth fixes:
//   - VM-AUDIT-2026-08-27-3 (BUG-3): executeCallUser inline loop checks
//     MaxStackDepth so a user-function loop can't grow the stack to MaxTicks.
//   - VM-AUDIT-2026-08-27-4 (BUG-4): OP_CALL_BUILTIN early-returns after popN
//     stack underflow so callBuiltin isn't called with partial args.
//
// These are defense-in-depth: the MQL compiler generates balanced push/pop
// bytecode, so the tests construct bytecode directly to trigger the paths.
//
// Adversarial proofs: each critical line mutated → relevant test RED → restore GREEN.

// --- T1: VM-AUDIT-2026-08-27-3 executeCallUser MaxStackDepth (BUG-3) ---

// TestVM_CallUserStackDepthLimit verifies BUG-3 fix: executeCallUser's inline
// loop checks MaxStackDepth so a user function that pushes without popping
// can't grow the stack to MaxTicks (~80-160MB) before the instruction limit
// fires. The check returns "strategy exceeded max stack depth" instead of
// "strategy exceeded instruction limit".
//
// Bytecode layout (constructed directly, not via the MQL compiler which
// balances push/pop):
//
//	0: OP_ENTER_ONBAR        (event marker, EntryPC for OnBar = 0)
//	1: OP_CALL_USER fn=4, nArgs=0   (call user function at pc=4)
//	2: OP_RETURN             (end of OnBar)
//	3: OP_HALT
//	4: OP_PUSH_CONST 0       (user function body: push one value, no pop, no return)
//	5: OP_JMP target=4       (infinite loop pushing onto the stack)
//
// The loop pushes one value per iteration. Without the MaxStackDepth check
// in executeCallUser, the stack grows until MaxTicks (10M) → instruction
// limit error. With the check, it stops at MaxStackDepth+1 (4097) → stack
// depth error.
//
// Adversarial proof: delete the `if len(vm.stack) > MaxStackDepth` check in
// executeCallUser → the loop runs until MaxTicks → error message is
// "strategy exceeded instruction limit" (not "max stack depth") → RED.
// Restore → error message is "strategy exceeded max stack depth" → GREEN.
func TestVM_CallUserStackDepthLimit(t *testing.T) {
	// Build bytecode directly.
	bc := &Bytecode{
		Consts: []ConstValue{{Kind: interp.ValInt, Int: 1}}, // const 0 = int 1
		Code: []Instruction{
			{Op: OP_ENTER_ONBAR, A: 0, B: 0}, // 0: event marker
			{Op: OP_CALL_USER, A: 4, B: 0},   // 1: call user fn at pc=4, 0 args
			{Op: OP_RETURN, A: 0, B: 0},      // 2: end OnBar
			{Op: OP_HALT, A: 0, B: 0},        // 3: halt
			{Op: OP_PUSH_CONST, A: 0, B: 0},  // 4: push const 0 (int 1)
			{Op: OP_JMP, A: 4, B: 0},         // 5: jmp back to 4 (infinite push loop)
		},
		GlobalSlots: map[string]VarID{},
		Funcs: map[string]FuncEntry{
			"pusher": {Name: "pusher", EntryPC: 4, NumParams: 0, NumLocals: 0},
		},
		Builtins:    map[string]BuiltinID{},
		EventLocals: map[int32]int{0: 0},
		OnBar:       0,
		OnInit:      -1, OnTick: -1, OnTrade: -1, OnTimer: -1,
		OnDeinit: -1, OnTradeTransaction: -1, OnBookEvent: -1,
	}

	vm := NewVM(bc)
	vm.SetSignalMode(false)

	err := vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("expected error from stack-overflowing user function, got nil")
	}
	if !strings.Contains(err.Error(), "max stack depth") {
		t.Fatalf("expected 'max stack depth' error, got: %v (BUG-3: without the check, this would be 'instruction limit')", err)
	}
}

// --- T2: VM-AUDIT-2026-08-27-4 popN underflow stops builtin (BUG-4) ---

// TestVM_PopNStackUnderflowStopsBuiltin verifies BUG-4 fix: when popN
// encounters stack underflow (nArgs > len(stack)), it sets fatalError, and
// OP_CALL_BUILTIN's new early return prevents callBuiltin from executing
// with partial arguments.
//
// Bytecode layout (constructed directly):
//
//	0: OP_PUSH_CONST 0       (push 1 value onto stack)
//	1: OP_CALL_BUILTIN id=mockBuiltinID, nArgs=3  (needs 3 args, only 1 on stack)
//	2: OP_HALT
//
// A mock builtin handler increments a counter when called. With the fix,
// the handler is NOT called (early return after popN underflow). Without
// the fix, the handler IS called with 1 arg (counter increments).
//
// Adversarial proof: delete the `if vm.fatalError != ""` early return after
// popN → callBuiltin is called with partial args → counter increments → RED.
// Restore → counter stays 0 → GREEN.
func TestVM_PopNStackUnderflowStopsBuiltin(t *testing.T) {
	// Register a mock builtin handler that increments a counter when called.
	// We use a builtin slot that has no production handler (nil fn) and
	// temporarily set a handler. "GlobalVariableGet" is a non-critical builtin
	// with nil fn by default — safe to borrow for the test.
	mockName := "GlobalVariableGet"
	mockID := id(mockName)
	originalFn := builtinRegistry[mockID].fn
	t.Cleanup(func() { builtinRegistry[mockID].fn = originalFn })

	var callCount int32
	builtinRegistry[mockID].fn = func(vm *VM, args []interp.Value) (interp.Value, error) {
		atomic.AddInt32(&callCount, 1)
		return interp.NoneVal(), nil
	}

	// Build bytecode: push 1 value, then call builtin needing 3 args.
	bc := &Bytecode{
		Consts: []ConstValue{{Kind: interp.ValInt, Int: 1}}, // const 0 = int 1
		Code: []Instruction{
			{Op: OP_PUSH_CONST, A: 0, B: 0},               // 0: push 1 value
			{Op: OP_CALL_BUILTIN, A: int32(mockID), B: 3}, // 1: needs 3 args, only 1 on stack
			{Op: OP_HALT, A: 0, B: 0},                     // 2: halt
		},
		GlobalSlots: map[string]VarID{},
		Funcs:       map[string]FuncEntry{},
		Builtins:    map[string]BuiltinID{},
		EventLocals: map[int32]int{0: 0},
		OnBar:       0,
		OnInit:      -1, OnTick: -1, OnTrade: -1, OnTimer: -1,
		OnDeinit: -1, OnTradeTransaction: -1, OnBookEvent: -1,
	}

	vm := NewVM(bc)
	vm.SetSignalMode(false)

	err := vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("expected error from popN stack underflow, got nil")
	}
	if !strings.Contains(err.Error(), "stack error") {
		t.Fatalf("expected error containing 'stack error', got: %v", err)
	}

	// The mock builtin handler must NOT have been called.
	if count := atomic.LoadInt32(&callCount); count != 0 {
		t.Fatalf("mock builtin was called %d times — popN underflow should have stopped callBuiltin (BUG-4)", count)
	}
}
