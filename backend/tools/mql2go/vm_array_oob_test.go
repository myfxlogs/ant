// vm_array_oob_test.go — VM-ARRAY-OOB-FAILCLOSED-1 behavior tests (2026-09-18).
//
// Real MQL `array out of range` is a fatal runtime error; the VM used to
// silently return fake values (OP_PUSH_ARRAY) / drop writes (OP_STORE_ARRAY),
// and a negative-encoded local-array slot was mistaken for a global slot at
// runtime. These tests pin the fail-closed behavior per the revised dispatch:
//
//	Main path     — hand-built VM/Instructions covering every
//	                executePushArray/executeStoreArray branch + dispatch guard.
//	Source path a — builtin array stored into a global (StringSplit), then
//	                subscript OOB read ("index N out of range").
//	Source path b — subscript on a scalar global ("is not an array").
//	Source path c — local scalar subscript read (negative-encoded slot →
//	                "local array" error, same-index globals untouched).
//
// (The `int g[2]` global-declaration form is NOT reachable today — the
// declaration is dropped by the front end, tracked as VM-GLOBAL-ARRAY-DECL-1.)
//
// Adversarial proofs (4): each critical line mutated → relevant test RED →
// restore GREEN. See the dispatch acceptance criteria.
package mql2go

import (
	"context"
	"strings"
	"testing"

	"alphaforge/tools/mql2go/interp"
)

// ── Main path: hand-built VM + instructions ──────────────────────────

// newArrayOOBVM returns a VM with one global slot holding kind (defaults to
// a 2-element string array when kind is ValArray).
func newArrayOOBVM(t *testing.T, global interp.Value) *VM {
	t.Helper()
	vm := NewVM(&Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)})
	vm.globals = make([]interp.Value, 1)
	vm.globals[0] = global
	return vm
}

func strArrayVal(elems ...string) interp.Value {
	arr := make([]interp.Value, len(elems))
	for i, e := range elems {
		arr[i] = interp.StringVal(e)
	}
	return interp.Value{Kind: interp.ValArray, Array: arr}
}

// TestArraySlotBeyondGlobals — slot >= len(globals) must error on both the
// read and write path. Not reachable via source (the compiler always
// allocates slots < len(globals)), so drive execute*Array directly.
func TestArraySlotBeyondGlobals(t *testing.T) {
	vm := newArrayOOBVM(t, interp.IntVal(0))

	_ = vm.executePushArray(Instruction{Op: OP_PUSH_ARRAY, A: 5}, interp.IntVal(0))
	if vm.fatalError == "" {
		t.Fatal("OP_PUSH_ARRAY slot 5 with globals=1: fatalError empty, want error")
	}
	if !strings.Contains(vm.fatalError, "OP_PUSH_ARRAY slot 5 out of range (globals=1)") {
		t.Fatalf("fatalError = %q, want it to contain 'OP_PUSH_ARRAY slot 5 out of range (globals=1)'", vm.fatalError)
	}

	vm.fatalError = ""
	vm.executeStoreArray(Instruction{Op: OP_STORE_ARRAY, A: 5}, interp.IntVal(0), interp.IntVal(1))
	if vm.fatalError == "" {
		t.Fatal("OP_STORE_ARRAY slot 5 with globals=1: fatalError empty, want error")
	}
	if !strings.Contains(vm.fatalError, "OP_STORE_ARRAY slot 5 out of range (globals=1)") {
		t.Fatalf("fatalError = %q, want it to contain 'OP_STORE_ARRAY slot 5 out of range (globals=1)'", vm.fatalError)
	}
}

// TestNonArraySlotDirect — subscripting a non-array global must set a stack
// error on both paths. Mutation vector for executeStoreArray's silent branch.
//
// Adversarial: restore executeStoreArray's not-an-array branch to a silent
// return → the store subtest REDs (no error).
func TestNonArraySlotDirect(t *testing.T) {
	vm := newArrayOOBVM(t, interp.IntVal(42))

	_ = vm.executePushArray(Instruction{Op: OP_PUSH_ARRAY, A: 0}, interp.IntVal(0))
	if !strings.Contains(vm.fatalError, "OP_PUSH_ARRAY slot 0 is not an array") {
		t.Fatalf("push: fatalError = %q, want 'OP_PUSH_ARRAY slot 0 is not an array'", vm.fatalError)
	}

	vm.fatalError = ""
	vm.executeStoreArray(Instruction{Op: OP_STORE_ARRAY, A: 0}, interp.IntVal(0), interp.IntVal(1))
	if !strings.Contains(vm.fatalError, "OP_STORE_ARRAY slot 0 is not an array") {
		t.Fatalf("store: fatalError = %q, want 'OP_STORE_ARRAY slot 0 is not an array'", vm.fatalError)
	}
	if vm.globals[0].ToInt() != 42 {
		t.Fatalf("scalar global was modified by store: %v", vm.globals[0])
	}
}

// TestIndexOOBDirect — out-of-range (positive and negative) subscripts on a
// real array global must error with the exact message format. Mutation vector
// for executePushArray's silent index branch.
//
// Adversarial: restore executePushArray's index-OOB branch to a silent
// NoneVal return → the read subtests REDs (no error).
func TestIndexOOBDirect(t *testing.T) {
	vm := newArrayOOBVM(t, strArrayVal("a", "b"))

	_ = vm.executePushArray(Instruction{Op: OP_PUSH_ARRAY, A: 0}, interp.IntVal(9))
	if !strings.Contains(vm.fatalError, "OP_PUSH_ARRAY index 9 out of range (len=2)") {
		t.Fatalf("push idx=9: fatalError = %q, want 'index 9 out of range (len=2)'", vm.fatalError)
	}

	vm.fatalError = ""
	_ = vm.executePushArray(Instruction{Op: OP_PUSH_ARRAY, A: 0}, interp.IntVal(-1))
	if !strings.Contains(vm.fatalError, "OP_PUSH_ARRAY index -1 out of range (len=2)") {
		t.Fatalf("push idx=-1: fatalError = %q, want 'index -1 out of range (len=2)'", vm.fatalError)
	}

	vm.fatalError = ""
	vm.executeStoreArray(Instruction{Op: OP_STORE_ARRAY, A: 0}, interp.IntVal(9), interp.IntVal(1))
	if !strings.Contains(vm.fatalError, "OP_STORE_ARRAY index 9 out of range (len=2)") {
		t.Fatalf("store idx=9: fatalError = %q, want 'index 9 out of range (len=2)'", vm.fatalError)
	}
}

// TestNegativeSlotDirect — a negative-encoded slot (local array access) must
// error and never touch vm.globals. The store branch is the actual
// mis-write protection: with the old code, globals[-ins.A-1] was targeted.
func TestNegativeSlotDirect(t *testing.T) {
	vm := newArrayOOBVM(t, strArrayVal("a", "b"))

	_ = vm.executePushArray(Instruction{Op: OP_PUSH_ARRAY, A: -1}, interp.IntVal(0))
	if !strings.Contains(vm.fatalError, "OP_PUSH_ARRAY local array slot 0 not supported") {
		t.Fatalf("push: fatalError = %q, want 'local array slot 0 not supported'", vm.fatalError)
	}

	vm.fatalError = ""
	vm.executeStoreArray(Instruction{Op: OP_STORE_ARRAY, A: -1}, interp.IntVal(0), interp.StringVal("evil"))
	if !strings.Contains(vm.fatalError, "OP_STORE_ARRAY local array slot 0 not supported") {
		t.Fatalf("store: fatalError = %q, want 'local array slot 0 not supported'", vm.fatalError)
	}
	// The same-index global array must be untouched by the rejected write.
	if got := vm.globals[0].Array[0].ToString(); got != "a" {
		t.Fatalf("globals[0][0] = %q, want %q — negative-slot store touched a global array", got, "a")
	}
}

// TestInBoundsDirect — legal subscripts keep working on both paths (no
// over-blocking).
func TestInBoundsDirect(t *testing.T) {
	vm := newArrayOOBVM(t, strArrayVal("a", "b"))

	v := vm.executePushArray(Instruction{Op: OP_PUSH_ARRAY, A: 0}, interp.IntVal(0))
	if vm.fatalError != "" || v.ToString() != "a" {
		t.Fatalf("push idx=0: v=%v fatalError=%q, want %q with no error", v, vm.fatalError, "a")
	}
	v = vm.executePushArray(Instruction{Op: OP_PUSH_ARRAY, A: 0}, interp.IntVal(1))
	if vm.fatalError != "" || v.ToString() != "b" {
		t.Fatalf("push idx=1: v=%v fatalError=%q, want %q with no error", v, vm.fatalError, "b")
	}

	vm.fatalError = ""
	vm.executeStoreArray(Instruction{Op: OP_STORE_ARRAY, A: 0}, interp.IntVal(1), interp.StringVal("z"))
	if vm.fatalError != "" {
		t.Fatalf("store idx=1: fatalError=%q, want none", vm.fatalError)
	}
	if got := vm.globals[0].Array[1].ToString(); got != "z" {
		t.Fatalf("globals[0][1] = %q, want %q (in-bounds store must write)", got, "z")
	}
}

// TestPushArrayDispatchGuard — after a stack error the OP_PUSH_ARRAY dispatch
// case must not push the NoneVal fallback value.
//
// Adversarial: remove the fatalError guard (restore unconditional push) →
// the fake value lands on the stack (len=1) → RED.
func TestPushArrayDispatchGuard(t *testing.T) {
	bc := &Bytecode{
		OnBar:    0,
		Builtins: make(map[string]BuiltinID),
		Consts:   []ConstValue{constFromValue(interp.IntVal(9))},
		Code: []Instruction{
			{Op: OP_PUSH_CONST, A: 0}, // push index 9
			{Op: OP_PUSH_ARRAY, A: 0}, // globals[0] is not an array → stack error
			{Op: OP_RETURN},
			{Op: OP_HALT},
		},
	}
	vm := NewVM(bc)
	vm.globals = make([]interp.Value, 1)
	vm.globals[0] = interp.IntVal(42)

	err := vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("subscript on non-array global: OnBar err = nil, want error")
	}
	if !strings.Contains(err.Error(), "is not an array") {
		t.Fatalf("err = %v, want it to contain 'is not an array'", err)
	}
	if len(vm.stack) != 0 {
		t.Fatalf("stack len = %d, want 0 — the NoneVal fallback was pushed onto the stack after a stack error", len(vm.stack))
	}
}

// ── Source-reachable paths ───────────────────────────────────────────

// newSourceVM compiles MQL source with a minimal context attached.
func newSourceVM(t *testing.T, src string) *VMRunner {
	t.Helper()
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	vmRunner.vm.SetSignalMode(true)
	vmRunner.vm.SetContext(&accountStatusTestContext{})
	return vmRunner
}

// TestGlobalArrayFromBuiltinOOB — a builtin-produced array stored into a
// global is subscriptable; OOB reads must be fatal runtime errors (covers the
// original S4-1/2 OOB semantics on the reachable path).
//
// Adversarial: restore executePushArray's index branch to silent → OnBar
// succeeds → RED (also via TestIndexOOBDirect).
func TestGlobalArrayFromBuiltinOOB(t *testing.T) {
	t.Run("read OOB errors", func(t *testing.T) {
		vmRunner := newSourceVM(t,
			"int g; int x; int OnInit(){ g=StringSplit(\"a,b\",\",\"); return 0; } void OnBar(){ x=g[9]; }")
		if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
			t.Fatalf("RunOnInit failed: %v", err)
		}
		err := vmRunner.vm.RunOnBar(context.Background())
		if err == nil {
			t.Fatal("x=g[9] on len-2 array: OnBar err = nil, want error (array out of range)")
		}
		if !strings.Contains(err.Error(), "index 9 out of range (len=2)") {
			t.Fatalf("err = %v, want it to contain 'index 9 out of range (len=2)'", err)
		}
	})

	t.Run("negative index errors", func(t *testing.T) {
		vmRunner := newSourceVM(t,
			"int g; int x; int OnInit(){ g=StringSplit(\"a,b\",\",\"); return 0; } void OnBar(){ x=g[-1]; }")
		if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
			t.Fatalf("RunOnInit failed: %v", err)
		}
		err := vmRunner.vm.RunOnBar(context.Background())
		if err == nil {
			t.Fatal("x=g[-1]: OnBar err = nil, want error (negative index out of range)")
		}
		if !strings.Contains(err.Error(), "index -1 out of range (len=2)") {
			t.Fatalf("err = %v, want it to contain 'index -1 out of range (len=2)'", err)
		}
	})

	t.Run("in-bounds read still works", func(t *testing.T) {
		vmRunner := newSourceVM(t,
			"int g; string x; int OnInit(){ g=StringSplit(\"a,b\",\",\"); x=g[1]; return 0; } void OnBar(){}")
		if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
			t.Fatalf("in-bounds read must not error, got: %v", err)
		}
		x, ok := vmRunner.GetGlobal("x")
		if !ok {
			t.Fatal("global x not found")
		}
		if got := x.ToString(); got != "b" {
			t.Errorf("x = %q, want %q (in-bounds g[1] read)", got, "b")
		}
	})
}

// TestNonArraySubscript — subscripting a scalar global must be a runtime
// error, not a silent no-op.
func TestNonArraySubscript(t *testing.T) {
	vmRunner := newSourceVM(t, "int s; int x; int OnInit(){ return 0; } void OnBar(){ x=s[0]; }")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	err := vmRunner.vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("x=s[0] on scalar global: OnBar err = nil, want error (not an array)")
	}
	if !strings.Contains(err.Error(), "is not an array") {
		t.Fatalf("err = %v, want it to contain 'is not an array'", err)
	}
}

// TestLocalArrayAccessFails — a local (initialized declaration or function
// parameter) subscripted like an array compiles with a blind spot but the
// runtime must fail-closed with an honest error instead of silently accessing
// the same-index global slot.
//
// NOTE: the local must carry an initializer (or be a parameter) — a bare
// `int a;` declaration is dropped by the front end and the name becomes an
// implicit global (front-end quirk, adjacent to VM-GLOBAL-ARRAY-DECL-1).
//
// Adversarial: restore compileSubscript to emit int32(slot) → the local slot
// is mistaken for globals[0] (the StringSplit array) → the read silently
// succeeds → OnBar err = nil → RED.
func TestLocalArrayAccessFails(t *testing.T) {
	const src = "int g; int OnInit(){ g=StringSplit(\"a,b,c\",\",\"); return 0; } " +
		"void f(){ int a = 5; int b = a[0]; } void OnBar(){ f(); }"
	vmRunner := newSourceVM(t, src)
	vm := vmRunner.vm
	if err := vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	err := vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("local a[0] read: OnBar err = nil, want error (local array access not supported)")
	}
	if !strings.Contains(err.Error(), "local array") {
		t.Fatalf("err = %v, want it to contain 'local array'", err)
	}

	// The compile-time contract is unchanged: the blind spot is still recorded.
	found := false
	for _, bs := range vm.bc.Coverage.BlindSpots {
		if strings.Contains(bs, "local array read: a") {
			found = true
		}
	}
	if !found {
		t.Fatalf("Coverage.BlindSpots = %v, want it to contain 'local array read: a'", vm.bc.Coverage.BlindSpots)
	}

	// The same-index global array (globals[0] = g, the StringSplit result)
	// must be untouched — the negative-encoded local slot never aliases it.
	gSlot, ok := vm.bc.GlobalSlots["g"]
	if !ok {
		t.Fatal("global g not found in GlobalSlots")
	}
	gv := vm.globals[gSlot]
	if gv.Kind != interp.ValArray {
		t.Fatalf("g Kind = %v, want ValArray", gv.Kind)
	}
	if len(gv.Array) != 3 || gv.Array[0].ToString() != "a" || gv.Array[2].ToString() != "c" {
		t.Fatalf("globals[g] = %v — the rejected local-array access modified the same-index global array", gv.Array)
	}
}
