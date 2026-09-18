// vm_func_fatal_delay_test.go — VM-FUNC-FATAL-DELAY-1 (2026-09-18).
//
// executeCallUser's inner fetch-execute loop lacked runLoop's top-of-loop
// fatal check: a stack/slot/arith fault inside a user function set
// fatalError but every subsequent instruction until OP_RETURN still ran
// (global writes leaked, and a faulting nested call's caller kept running).
// The fix mirrors runLoop's check at the top of the function loop, which
// also covers the entry state (popN stack-underflow fatal).
//
// Adversarial proofs (3): see the dispatch mutation list.
package mql2go

import (
	"context"
	"strings"
	"testing"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// newFatalDelayVM compiles MQL source, wires a 3-bar context, and runs
// OnInit — mirroring the vm_audit_test.go compile/run channel.
func newFatalDelayVM(t *testing.T, src string) *VMRunner {
	t.Helper()
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	vmRunner.vm.SetContext(&tsTestContext{bars: sdk.BarsToSlice(makeFailClosedBars(3))})
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	return vmRunner
}

// TestFuncInternalFaultDoesNotLeakWrites — (a): a fault inside f() must stop
// f immediately; the global write after the faulting statement never runs.
//
// Before the fix: g_after == 42 (leak). Adversarial: delete the loop-top
// fatal check → g_after == 42 AND the error only surfaces after OP_RETURN
// → RED.
func TestFuncInternalFaultDoesNotLeakWrites(t *testing.T) {
	src := "int g_after=0; void f(){ int x=10/0; g_after=42; } " +
		"int OnInit(){ return 0; } void OnBar(){ f(); }"
	vmRunner := newFatalDelayVM(t, src)
	err := vmRunner.vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("OnBar err = nil, want fatal (division by zero inside f)")
	}
	if !strings.Contains(err.Error(), "VM fatal") {
		t.Fatalf("err = %v, want it to contain 'VM fatal'", err)
	}
	if got := getGlobalInt(t, vmRunner, "g_after"); got != 0 {
		t.Fatalf("g_after = %d, want 0 — post-fault write leaked out of the function", got)
	}
}

// TestCallerDoesNotContinueAfterCalleeFault — (b): a faulting nested call
// must stop the caller too; the statement after inner() never runs.
//
// Before the fix: inner() returned cleanly past its fault, so g2 == 9 (leak).
func TestCallerDoesNotContinueAfterCalleeFault(t *testing.T) {
	src := "int g2=0; void inner(){ int x=1/0; } void outer(){ inner(); g2=9; } " +
		"int OnInit(){ return 0; } void OnBar(){ outer(); }"
	vmRunner := newFatalDelayVM(t, src)
	err := vmRunner.vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("OnBar err = nil, want fatal (inner fault must abort outer)")
	}
	if !strings.Contains(err.Error(), "VM fatal") {
		t.Fatalf("err = %v, want it to contain 'VM fatal'", err)
	}
	if got := getGlobalInt(t, vmRunner, "g2"); got != 0 {
		t.Fatalf("g2 = %d, want 0 — caller kept running after the callee fault", got)
	}
}

// TestEntryFatalStopsFunctionBody — (c): a fatal at the function entry
// (popN stack underflow) must prevent the first body instruction from
// executing.
//
// Bytecode mirrors the vm_audit_2026_08_27_batch2_test.go style:
// OP_CALL_USER with nArgs=2 on an empty stack → popN sets a fatal → the
// function body must not run.
func TestEntryFatalStopsFunctionBody(t *testing.T) {
	bc := &Bytecode{
		Consts: []ConstValue{{Kind: interp.ValInt, Int: 7}}, // const 0 = int 7
		Code: []Instruction{
			{Op: OP_ENTER_ONBAR, A: 0, B: 0}, // 0: event marker
			{Op: OP_CALL_USER, A: 4, B: 2},   // 1: call fn at pc=4 with 2 args (stack is empty)
			{Op: OP_RETURN, A: 0, B: 0},      // 2: end OnBar
			{Op: OP_HALT, A: 0, B: 0},        // 3: halt
			{Op: OP_PUSH_CONST, A: 0, B: 0},  // 4: body — push 7 (must NOT run)
			{Op: OP_RETURN, A: 0, B: 0},      // 5: end body
		},
		GlobalSlots: map[string]VarID{},
		Funcs: map[string]FuncEntry{
			"victim": {Name: "victim", EntryPC: 4, NumParams: 2, NumLocals: 2},
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
		t.Fatal("OnBar err = nil, want fatal (popN underflow at function entry)")
	}
	if !strings.Contains(err.Error(), "VM fatal") {
		t.Fatalf("err = %v, want it to contain 'VM fatal' (entry fatal must stop the body)", err)
	}
	// Discriminating assertion (review R1): the loop-top check fires BEFORE
	// the body's first fetch, so the victim body's OP_PUSH_CONST never runs
	// and the stack stays empty. Without the check the body still pushes 7
	// (stack len 1) and only runLoop's top check surfaces the same error text
	// — err text alone cannot tell the two apart.
	if len(vm.stack) != 0 {
		t.Fatalf("stack len = %d, want 0 — the function body executed despite the entry fatal", len(vm.stack))
	}
}
