package mql2go

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// QS-1.3: Python function-local scope — assignment to an undeclared name inside
// a function/event must declare a function-local slot, not leak into
// GlobalSlots where it would be visible across events and functions.

// S4a: cross-function isolation. A write to `x` inside helper must not be
// observable from on_bar (and vice versa, modulo compile order — see
// TestQS13_LiteralReaderPoisonsSlot).
func TestQS13_CrossFunctionIsolation(t *testing.T) {
	source := `class S:
    def helper(self) -> None:
        x = 1
        return
    def on_bar(self) -> None:
        self.helper()
        self.r = x
        return
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	v, ok := vmRunner.GetGlobal("r")
	if !ok {
		t.Fatal("global r not found")
	}
	if v.Kind == interp.ValInt && v.Int == 1 {
		t.Errorf("r = 1: helper's local x leaked into globals — cross-function isolation broken")
	}
	if v.Kind != interp.ValNone {
		t.Errorf("r kind = %v, want ValNone (x undeclared read → implicit global None)", v.Kind)
	}
}

// S4a literal form (handoff v2): `on_bar` writes `x = 1`, `helper` reads `x`.
// helper compiles BEFORE on_bar (funcs before events), so its `return x` hits
// the unchanged resolveVar read path and implicitly registers GlobalSlots["x"]
// first. Under GlobalDecls predicate, that implicit registration is not a
// declaration — on_bar's `x = 1` still gets a local slot and helper observes
// the never-written global as None.
func TestQS13_LiteralCrossFunctionIsolation(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        x = 1
        self.r = self.helper()
        return
    def helper(self) -> int:
        return x
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	v, ok := vmRunner.GetGlobal("r")
	if !ok {
		t.Fatal("global r not found")
	}
	if v.Kind == interp.ValInt && v.Int == 1 {
		t.Errorf("r = 1: on_bar's local x leaked into globals — reader-poisoned slot wrote global")
	}
	if v.Kind != interp.ValNone {
		t.Errorf("r kind = %v, want ValNone (helper reads implicit-global x never written)", v.Kind)
	}
}

// S4b: cross-event isolation. x assigned only during on_bar#1 (inside the
// `not phase` branch, which compiles before the read below it) must not be
// visible in on_bar#2 — event locals are rebuilt per event (vm.go runEvent).
func TestQS13_CrossEventIsolation(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        if not bool(self.phase):
            x = 1
            self.phase = True
        self.r = x
        return
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar #1 failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar #2 failed: %v", err)
	}
	v, ok := vmRunner.GetGlobal("r")
	if !ok {
		t.Fatal("global r not found")
	}
	if v.Kind == interp.ValInt && v.Int == 1 {
		t.Errorf("r = 1: x leaked across events — locals must be per-event")
	}
	if v.Kind != interp.ValNone {
		t.Errorf("r kind = %v, want ValNone (fresh local slot reads as None)", v.Kind)
	}
}

// S4b sibling: assignment must not register the name in GlobalSlots at all.
func TestQS13_LocalAssignNotInGlobals(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        x = 1
        self.r = x
        return
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	v, ok := vmRunner.GetGlobal("r")
	if !ok || v.Kind != interp.ValInt || v.Int != 1 {
		t.Fatalf("r = %+v ok=%v, want IntVal(1) (same-scope local read)", v, ok)
	}
	if _, ok := vmRunner.GetGlobal("x"); ok {
		t.Errorf("x found in globals — local assignment must not register GlobalSlots")
	}
}

// S4c regression: self.count += 1 accumulates across events via GlobalSlots.
func TestQS13_SelfFieldCompoundAssignRegression(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        self.count += 1
        return
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar #1 failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar #2 failed: %v", err)
	}
	v, ok := vmRunner.GetGlobal("count")
	if !ok {
		t.Fatal("global count not found")
	}
	if !v.Decimal.Equal(decimal.NewFromInt(2)) && !(v.Kind == interp.ValInt && v.Int == 2) {
		t.Errorf("count = %+v, want 2 (self field must stay global across events)", v)
	}
}

// S4d: parameter shadowing — assignment to a param name writes the param slot,
// not a new global.
func TestQS13_ParamShadowing(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        self.r = self.f(5)
        return
    def f(self, x) -> int:
        x = x + 1
        return x
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	v, ok := vmRunner.GetGlobal("r")
	if !ok {
		t.Fatal("global r not found")
	}
	if v.Kind != interp.ValInt || v.Int != 6 {
		t.Errorf("r = %+v, want IntVal(6) (param slot assignment)", v)
	}
	if _, ok := vmRunner.GetGlobal("x"); ok {
		t.Errorf("x found in globals — param assignment must not register GlobalSlots")
	}
}

// S4e: `global`/`nonlocal` statements are fail-closed compile errors.
func TestQS13_GlobalNonlocalRejected(t *testing.T) {
	for _, kw := range []string{"global", "nonlocal"} {
		source := `class S:
    def on_bar(self) -> None:
        ` + kw + ` x
        return
`
		if _, err := CompilePython(source); err == nil {
			t.Errorf("%s: CompilePython returned nil error, want explicit rejection", kw)
		} else if !strings.Contains(err.Error(), "not allowed in Python subset") &&
			!strings.Contains(err.Error(), "global/nonlocal") {
			t.Errorf("%s: error %q is not an explicit rejection", kw, err)
		}
	}
}

// S4f: augmented assignment on a completely undeclared name is a compile error
// (Python raises NameError) — must not implicitly create a global to mutate.
func TestQS13_CompoundAssignUndeclaredRejected(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        x += 1
        return
`
	if _, err := CompilePython(source); err == nil {
		t.Fatal("CompilePython returned nil error for undeclared +=, want explicit rejection")
	} else if !strings.Contains(err.Error(), "undeclared") {
		t.Fatalf("error %q does not mention undeclared name", err)
	}
}

// S4f sibling: += on a name declared earlier in the same function still works.
func TestQS13_CompoundAssignDeclaredLocal(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        x = 1
        x += 1
        self.r = x
        return
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	v, ok := vmRunner.GetGlobal("r")
	if !ok {
		t.Fatal("global r not found")
	}
	if v.Kind != interp.ValInt || v.Int != 2 {
		t.Errorf("r = %+v, want IntVal(2) (declared local +=)", v)
	}
}

// S4g: local slot allocation lands in the function's local frame
// (EventLocals for events / NumLocals for user funcs), not GlobalSlots.
func TestQS13_LocalSlotAccounting(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        x = 1
        y = 2
        self.r = x + y
        return
    def helper(self) -> int:
        z = 3
        return z
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}
	// on_bar declares two locals (x, y).
	onBarEntry := bc.OnBar
	if got := bc.EventLocals[onBarEntry]; got < 2 {
		t.Errorf("EventLocals[on_bar] = %d, want >= 2 (x and y are locals)", got)
	}
	// helper declares one local (z).
	if fn, ok := bc.Funcs["helper"]; !ok {
		t.Fatal("helper not in bc.Funcs")
	} else if fn.NumLocals < 1 {
		t.Errorf("helper NumLocals = %d, want >= 1 (z is a local)", fn.NumLocals)
	}
}

// S4i: assignment inside a for body binds the FUNCTION scope — x survives
// the loop scope's popScope (Python has no block scope).
func TestQS13_ForBodyAssignSurvivesLoop(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        for i in range(3):
            x = i
        self.r = x
        return
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	v, ok := vmRunner.GetGlobal("r")
	if !ok {
		t.Fatal("global r not found")
	}
	if v.Kind != interp.ValInt || v.Int != 2 {
		t.Errorf("r = %+v, want IntVal(2) (x bound in function scope survives loop)", v)
	}
	if _, ok := vmRunner.GetGlobal("x"); ok {
		t.Errorf("x found in globals — for-body assignment must not register GlobalSlots")
	}
}

// S4j: the for-loop variable itself is function-scoped (ExprDecl path).
// Note: the range() desugar is i=0; i<N; i++ — i overshoots to 3 at exit
// (existing desugar semantics; Python would leave 2). Only survival is pinned.
func TestQS13_ForLoopVarSurvivesLoop(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        for i in range(3):
            pass
        self.r = i
        return
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	v, ok := vmRunner.GetGlobal("r")
	if !ok {
		t.Fatal("global r not found")
	}
	if v.Kind != interp.ValInt || v.Int != 3 {
		t.Errorf("r = %+v, want IntVal(3) (loop var survives; desugar overshoots bound)", v)
	}
	if _, ok := vmRunner.GetGlobal("i"); ok {
		t.Errorf("i found in globals — loop variable must be function-local")
	}
}

// S4k: while-body assignment survives the loop — while pushes no scope, so
// this pins symmetric behavior with for (regression guard).
func TestQS13_WhileBodyAssignSurvives(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        self.go = True
        while bool(self.go):
            x = 1
            self.go = False
        self.r = x
        return
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	v, ok := vmRunner.GetGlobal("r")
	if !ok {
		t.Fatal("global r not found")
	}
	if v.Kind != interp.ValInt || v.Int != 1 {
		t.Errorf("r = %+v, want IntVal(1) (while-body assignment survives)", v)
	}
}

// S4h: strategy parameter names live in GlobalSlots but not GlobalDecls.
// Assigning to a param name inside a function declares a local (Pythonic
// Option B semantics) — the global param value must be unchanged.
func TestQS13_ParamNameAssignStaysLocal(t *testing.T) {
	source := `class S:
    def __init__(self, threshold: float = 1.5) -> None:
        pass
    def f(self) -> None:
        threshold = 99
        return
    def on_bar(self) -> None:
        self.f()
        self.r = threshold
        return
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	v, ok := vmRunner.GetGlobal("threshold")
	if !ok {
		t.Fatal("global threshold not found")
	}
	if v.Kind == interp.ValInt && v.Int == 99 {
		t.Errorf("threshold = 99: f's assignment wrote the global param — want local (param value unchanged)")
	}
	if v.Kind == interp.ValDecimal && v.Decimal.Equal(decimal.NewFromInt(99)) {
		t.Errorf("threshold = 99: f's assignment wrote the global param — want local (param value unchanged)")
	}
}
