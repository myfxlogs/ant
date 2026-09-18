// vm_local_array_test.go — MQL-COMPILER-LOCAL-ARRAYS (2026-09-19).
//
// MQL function-local array declarations (`double p[]` / `int a[3]`) used to
// be a compile-time hard failure. They now compile to OP_NEW_ARRAY with the
// declaration size (0 = dynamic), and ArrayResize on a plain variable compiles
// to OP_ARRAY_RESIZE which writes the resized value back to its slot —
// fixing bug A (global ArrayResize silently dropped its new length) and
// enabling locals end-to-end. Unsupported forms (initializer, non-constant
// size, multi-dimensional) remain compile errors.
//
// Adversarial proofs (4): delete OP_NEW_ARRAY handling → T1/T2 RED; restore
// the negative-path "not supported" rejection → T2/T6 RED; delete the
// OP_ARRAY_RESIZE slot write-back → T3 RED (bug A revives); delete the
// init_declarator pre-scan → T5 initializer sub-case RED (silent
// miscompilation revives).
package mql2go

import (
	"context"
	"strings"
	"testing"
)

// newLocalArrayVM compiles MQL source with a minimal context and runs OnInit.
func newLocalArrayVM(t *testing.T, src string) *VMRunner {
	t.Helper()
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	vmRunner.vm.SetContext(&accountStatusTestContext{})
	return vmRunner
}

// TestLocalDynamicArrayWithResize — T1, the original symptom end-to-end:
// dynamic local array + ArrayResize + element write/read inside a user
// function.
func TestLocalDynamicArrayWithResize(t *testing.T) {
	vmRunner := newLocalArrayVM(t,
		"int r; void f(){ double p[]; ArrayResize(p,3); p[1]=2.5; r=p[1]; } "+
			"int OnInit(){ return 0; } void OnBar(){ f(); }")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	r, ok := vmRunner.GetGlobal("r")
	if !ok {
		t.Fatal("global r not found")
	}
	if got := r.ToDecimal().String(); got != "2.5" {
		t.Fatalf("r = %s, want 2.5 (dynamic local array + resize + subscript)", got)
	}
}

// TestLocalSizedArrayInFunction — T2: a sized local array starts zeroed and
// survives element write/readback inside a user function (frame locals).
func TestLocalSizedArrayInFunction(t *testing.T) {
	vmRunner := newLocalArrayVM(t,
		"int a[3]; int zero; int r; void f(){ zero=a[2]; a[2]=7; r=a[2]; } "+
			"int OnInit(){ return 0; } void OnBar(){ f(); }")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	if got := getGlobalInt(t, vmRunner, "zero"); got != 0 {
		t.Fatalf("a[2] initial = %d, want 0 (zero-filled elements)", got)
	}
	if got := getGlobalInt(t, vmRunner, "r"); got != 7 {
		t.Fatalf("r = %d, want 7 (element write/read in frame locals)", got)
	}
}

// TestGlobalArrayResizeSlotWriteBack — T3, bug A regression: ArrayResize on
// a global must lengthen the slot's array (previously the re-sliced header
// never propagated and ArraySize stayed at the declared length).
//
// Adversarial (M3): delete the OP_ARRAY_RESIZE slot write-back → len stays 2
// → RED.
func TestGlobalArrayResizeSlotWriteBack(t *testing.T) {
	vmRunner := newLocalArrayVM(t,
		"double g[2]; double x; int OnInit(){ ArrayResize(g,5); g[4]=1.5; x=g[4]; return 0; } void OnBar(){}")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	x, ok := vmRunner.GetGlobal("x")
	if !ok {
		t.Fatal("global x not found")
	}
	if got := x.ToDecimal().String(); got != "1.5" {
		t.Fatalf("g[4] after resize = %s, want 1.5 (ArraySize stayed at the old length — bug A revived)", got)
	}
}

// TestGlobalArrayResizeShrink — T4: shrinking truncates the array.
func TestGlobalArrayResizeShrink(t *testing.T) {
	vmRunner := newLocalArrayVM(t,
		"int g[3]; int s; int OnInit(){ g[0]=1; g[1]=2; g[2]=3; ArrayResize(g,1); s=ArraySize(g); return 0; } void OnBar(){}")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	if got := getGlobalInt(t, vmRunner, "s"); got != 1 {
		t.Fatalf("ArraySize after shrink = %d, want 1", got)
	}
}

// TestLocalArrayCompileRejected — T5: unsupported forms stay compile errors.
// The initializer rejection also closes bug B (initializer silently
// compiled the array as a scalar).
//
// Adversarial (M4): delete the init_declarator pre-scan → the initializer
// sub-cases compile silently → RED.
func TestLocalArrayCompileRejected(t *testing.T) {
	rejectCases := []struct {
		name    string
		src     string
		wantMsg string
	}{
		{"local initializer list", "void f(){ int a[2]={1,2}; } int OnInit(){ return 0; }", "local array initializer not supported"},
		{"non-constant size", "void f(int n){ int a[n]; } int OnInit(){ return 0; }", "array size must be a constant"},
		{"multi-dimensional", "void f(){ int a[2][3]; } int OnInit(){ return 0; }", "multi-dimensional arrays not supported"},
		{"empty initializer", "void f(){ double p[]={}; } int OnInit(){ return 0; }", "local array initializer not supported"},
	}
	for _, tc := range rejectCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CompileMQL(tc.src)
			if err == nil {
				t.Fatalf("%s: CompileMQL err = nil, want error containing %q", tc.name, tc.wantMsg)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("%s: err = %v, want it to contain %q", tc.name, err, tc.wantMsg)
			}
		})
	}
}

// TestLocalArrayOOBFailClosed — T6: local array element access keeps the
// fail-closed OOB contract (read and write).
func TestLocalArrayOOBFailClosed(t *testing.T) {
	vmRunner := newLocalArrayVM(t,
		"int OnInit(){ return 0; } void OnBar(){ int a[3]; int x=a[5]; }")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	err := vmRunner.vm.RunOnBar(context.Background())
	// Index-bounds messages are shape-identical to the global path (dispatch
	// spec); the local-vs-global distinction lives in slot resolution and the
	// not-an-array/slot-range messages.
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("read OOB err = %v, want it to contain 'out of range'", err)
	}

	vmRunner = newLocalArrayVM(t,
		"int OnInit(){ return 0; } void OnBar(){ int a[3]; a[-1]=1; }")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	err = vmRunner.vm.RunOnBar(context.Background())
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("negative-index write err = %v, want it to contain 'out of range'", err)
	}
}

// TestLocalArraySizeAfterResize — T7: ArraySize reflects the resized local.
func TestLocalArraySizeAfterResize(t *testing.T) {
	vmRunner := newLocalArrayVM(t,
		"int s; void f(){ double p[]; ArrayResize(p,3); s=ArraySize(p); } "+
			"int OnInit(){ return 0; } void OnBar(){ f(); }")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	if got := getGlobalInt(t, vmRunner, "s"); got != 3 {
		t.Fatalf("ArraySize(p) = %d, want 3", got)
	}
}
