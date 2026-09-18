// vm_global_array_test.go — VM-GLOBAL-ARRAY-DECL-1 behavior tests (2026-09-18).
//
// Global array declarations (`int g[2];`) were silently dropped by the
// front end (collectGlobalVar had no array_declarator case), so no global
// array ever reached initGlobals — every subscript on one silently returned
// NoneVal / dropped writes. Subscript assignments degraded to whole-slot
// scalar stores, `g[0]++` created phantom globals, and array parameters were
// silently dropped. These tests pin the end-to-end fix and the explicit
// compile-time rejections for unsupported forms.
//
// Adversarial proofs (4): each critical line mutated → relevant test RED →
// restore GREEN. See the dispatch mutation list.
package mql2go

import (
	"context"
	"strings"
	"testing"

	"alphaforge/tools/mql2go/interp"

	"github.com/shopspring/decimal"
)

// TestGlobalArrayReadWrite — a declared global array materializes as a real
// ValArray; element writes and reads work end-to-end.
//
// Adversarial: remove the array_declarator collection case in collectGlobalVar
// → g degrades to a scalar implicit global → the store errors ("is not an
// array") → RED.
func TestGlobalArrayReadWrite(t *testing.T) {
	vmRunner := newSourceVM(t,
		"int g[2]; int x; int OnInit(){ g[0]=5; g[1]=7; return 0; } void OnBar(){ x=g[0]+g[1]; }")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	gSlot, ok := vmRunner.vm.bc.GlobalSlots["g"]
	if !ok {
		t.Fatal("global g not collected into GlobalSlots")
	}
	gv := vmRunner.vm.globals[gSlot]
	if gv.Kind != interp.ValArray || len(gv.Array) != 2 {
		t.Fatalf("g kind=%v len=%d, want ValArray(2) — declaration was not collected", gv.Kind, len(gv.Array))
	}
	x, ok := vmRunner.GetGlobal("x")
	if !ok {
		t.Fatal("global x not found")
	}
	if got := x.ToInt(); got != 12 {
		t.Fatalf("x = %d, want 12 (g[0]+g[1] = 5+7)", got)
	}
}

// TestGlobalArrayReadOOB — migrated from VM-ARRAY-OOB-FAILCLOSED-1 S4-2: the
// global-declaration form is now reachable, so the OOB read errors with the
// exact fail-closed message.
func TestGlobalArrayReadOOB(t *testing.T) {
	vmRunner := newSourceVM(t,
		"int g[2]; int x; int OnInit(){ return 0; } void OnBar(){ x=g[5]; }")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	err := vmRunner.vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("x=g[5] on len-2 array: OnBar err = nil, want error (array out of range)")
	}
	if !strings.Contains(err.Error(), "index 5 out of range (len=2)") {
		t.Fatalf("err = %v, want it to contain 'index 5 out of range (len=2)'", err)
	}
}

// TestGlobalArrayWriteOOB — migrated from VM-ARRAY-OOB-FAILCLOSED-1 S4-6: OOB
// writes through OP_STORE_ARRAY fail closed with the same message family.
func TestGlobalArrayWriteOOB(t *testing.T) {
	vmRunner := newSourceVM(t,
		"int g[2]; int OnInit(){ return 0; } void OnBar(){ g[9]=1; }")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	err := vmRunner.vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("g[9]=1 on len-2 array: OnBar err = nil, want error (array out of range)")
	}
	if !strings.Contains(err.Error(), "index 9 out of range (len=2)") {
		t.Fatalf("err = %v, want it to contain 'index 9 out of range (len=2)'", err)
	}
}

// TestGlobalArrayNegativeIndex — negative subscripts are out of range.
func TestGlobalArrayNegativeIndex(t *testing.T) {
	vmRunner := newSourceVM(t,
		"int g[2]; int x; int OnInit(){ return 0; } void OnBar(){ x=g[-1]; }")
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
}

// TestGlobalArrayTypes — double/string/bool arrays round-trip, and unwritten
// elements carry the correct zero value (0 / "" / false).
func TestGlobalArrayTypes(t *testing.T) {
	vmRunner := newSourceVM(t,
		"double d[3]; string s[2]; bool b[4]; "+
			"int OnInit(){ d[1]=2.5; s[0]=\"z\"; b[2]=1; return 0; } void OnBar(){}")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	for name, wantLen := range map[string]int{"d": 3, "s": 2, "b": 4} {
		slot, ok := vmRunner.vm.bc.GlobalSlots[name]
		if !ok {
			t.Fatalf("global %s not collected", name)
		}
		gv := vmRunner.vm.globals[slot]
		if gv.Kind != interp.ValArray || len(gv.Array) != wantLen {
			t.Fatalf("%s kind=%v len=%d, want ValArray(%d)", name, gv.Kind, len(gv.Array), wantLen)
		}
	}
	d, _ := vmRunner.GetGlobal("d")
	if !d.Array[1].ToDecimal().Equal(decimal.NewFromFloat(2.5)) || !d.Array[0].ToDecimal().IsZero() {
		t.Fatalf("d[1]=%s d[0]=%s, want 2.5 / 0 (write + zero default)", d.Array[1].ToDecimal(), d.Array[0].ToDecimal())
	}
	s, _ := vmRunner.GetGlobal("s")
	if s.Array[0].ToString() != "z" || s.Array[1].ToString() != "" {
		t.Fatalf("s[0]=%q s[1]=%q, want \"z\" / \"\" (write + zero default)", s.Array[0].ToString(), s.Array[1].ToString())
	}
	b, _ := vmRunner.GetGlobal("b")
	if b.Array[2].ToInt() != 1 || b.Array[0].ToInt() != 0 {
		t.Fatalf("b[2]=%d b[0]=%d, want 1 / 0 (write + zero default)", b.Array[2].ToInt(), b.Array[0].ToInt())
	}
}

// TestGlobalArrayDeclaredAfterUse — collection is a separate pass before
// body compilation, so legal use-before-declaration source keeps working.
func TestGlobalArrayDeclaredAfterUse(t *testing.T) {
	vmRunner := newSourceVM(t,
		"int x; int OnInit(){ g[0]=3; return 0; } int g[2]; void OnBar(){ x=g[0]; }")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	x, ok := vmRunner.GetGlobal("x")
	if !ok || x.ToInt() != 3 {
		t.Fatalf("x = %v, want 3 (use before declaration)", x)
	}
}

// TestGlobalArrayStatic — `static` has no semantic difference at global
// scope; the declaration must still be collected.
func TestGlobalArrayStatic(t *testing.T) {
	vmRunner := newSourceVM(t,
		"static int g[3]; int x; int OnInit(){ g[2]=9; return 0; } void OnBar(){ x=g[2]; }")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
	x, ok := vmRunner.GetGlobal("x")
	if !ok || x.ToInt() != 9 {
		t.Fatalf("x = %v, want 9 (static global array)", x)
	}
}

// TestGlobalArrayCompileRejected — unsupported forms are explicit compile
// errors, never silent miscompiles.
//
// Adversarial: delete the S3/S4 guards (or restore findIdent-first ordering
// in compileAssignment) → the corresponding source compiles again → RED.
func TestGlobalArrayCompileRejected(t *testing.T) {
	rejectCases := []struct {
		name    string
		src     string
		wantMsg string
	}{
		{"multi-dimensional", "int g[2][3]; int OnInit(){ return 0; }", "multi-dimensional arrays not supported"},
		{"initializer list", "int g[2]={1,2}; int OnInit(){ return 0; }", "global array initializer not supported"},
		{"scalar initializer", "int g[2]=5; int OnInit(){ return 0; }", "global array initializer not supported"},
		{"array parameter", "void f(int a[]){ } int OnInit(){ return 0; }", "array parameters not supported"},
		{"compound assign", "int g[2]; int OnInit(){ g[0]+=5; return 0; }", "compound assignment on array element not supported"},
		{"element update", "int g[2]; int OnInit(){ g[0]++; return 0; }", "++/-- on array element not supported"},
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

// TestLocalArrayStillRejected — the pre-existing local-array declaration
// rejection must survive this batch untouched.
func TestLocalArrayStillRejected(t *testing.T) {
	_, err := CompileMQL("void f(){ int a[2]; } int OnInit(){ return 0; }")
	if err == nil {
		t.Fatal("local array declaration: CompileMQL err = nil, want error (local arrays not supported)")
	}
	if !strings.Contains(err.Error(), "local arrays not supported") {
		t.Fatalf("err = %v, want it to contain 'local arrays not supported'", err)
	}
}

// TestScalarGlobalNotAliased — plain scalar globals must be unaffected by
// the array collection path.
func TestScalarGlobalNotAliased(t *testing.T) {
	vmRunner := newSourceVM(t,
		"int g; int OnInit(){ g=41; g=g+1; return 0; } void OnBar(){}")
	if err := vmRunner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	g, ok := vmRunner.GetGlobal("g")
	if !ok {
		t.Fatal("global g not found")
	}
	if gv := g.Kind; gv == interp.ValArray {
		t.Fatal("scalar global g was collected as an array")
	}
	if got := g.ToInt(); got != 42 {
		t.Fatalf("g = %d, want 42 (scalar arithmetic unaffected)", got)
	}
}
