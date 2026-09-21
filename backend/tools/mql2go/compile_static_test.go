// compile_static_test.go — VM-STATIC-LOCAL-1 (2026-09-21).
//
// Function-local `static` declarations must persist across calls with lazy
// (first-reach) initialization. Before the fix the storage_class_specifier
// CST node was silently dropped, so `static int done` compiled as a plain
// frame local that re-initialized on every call (live probe defect).
//
// Tests T1–T11 from docs/audits/design-vm-static-local-1.md §3.
// Red proof: stashing the S1–S3 source changes turns T1/T2/T3/T5/T6/T7/T8/T9/T11 RED
// (observed: values reset every tick); M1–M4 mutations re-red individually.
package mql2go

import (
	"context"
	"testing"

	"alphaforge/tools/mql2go/interp"
)

// compileStaticSrc compiles MQL source, failing the test on compile error.
func compileStaticSrc(t *testing.T, src string) *VMRunner {
	t.Helper()
	r, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	return r
}

// runStaticTicks executes n OnTick events on the runner's VM.
func runStaticTicks(t *testing.T, r *VMRunner, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := r.vm.RunOnTick(context.Background()); err != nil {
			t.Fatalf("OnTick %d failed: %v", i+1, err)
		}
	}
}

// staticGlobalInt reads an int-typed global slot by name after execution.
func staticGlobalInt(t *testing.T, r *VMRunner, name string) int32 {
	t.Helper()
	slot, ok := r.bc.GlobalSlots[name]
	if !ok {
		t.Fatalf("global slot %q not registered", name)
	}
	return r.vm.globals[slot].ToInt()
}

// T1: `static int d; d++;` across 3 OnTick calls → d = 1,2,3.
// Pre-fix behavior: d re-initialized per call → 1,1,1 (RED).
func TestStatic_LocalPersistsAcrossTicks(t *testing.T) {
	r := compileStaticSrc(t, `int g_out = 0;
void OnTick()
{
    static int d;
    d++;
    g_out = d;
}
`)
	runStaticTicks(t, r, 3)
	if got := staticGlobalInt(t, r, "g_out"); got != 3 {
		t.Fatalf("static local did not persist: g_out=%d, want 3 (1,2,3 across ticks)", got)
	}
	// EventLocals must not count static declarations — no frame slot consumed.
	if locals := r.bc.EventLocals[r.bc.OnTick]; locals != 0 {
		t.Fatalf("EventLocals=%d, want 0 — static must not occupy frame slots", locals)
	}
}

// T2: initializer side effects run exactly once (first reach).
// Pre-fix: InitOnce() called on every tick (RED).
func TestStatic_InitializerRunsOnce(t *testing.T) {
	r := compileStaticSrc(t, `int g_calls = 0;
int g_out = 0;
int InitOnce()
{
    g_calls++;
    return 42;
}
void OnTick()
{
    static int s = InitOnce();
    g_out = s;
}
`)
	runStaticTicks(t, r, 3)
	if got := staticGlobalInt(t, r, "g_calls"); got != 1 {
		t.Fatalf("initializer ran %d times, want exactly 1", got)
	}
	if got := staticGlobalInt(t, r, "g_out"); got != 42 {
		t.Fatalf("g_out=%d, want 42", got)
	}
}

// T3: static inside a for body — initialized on the first iteration of the
// first tick, then persists across iterations AND ticks.
// Pre-fix: k reset every iteration → g_out stuck at 101 (RED).
func TestStatic_InForBodyPersists(t *testing.T) {
	r := compileStaticSrc(t, `int g_out = 0;
void OnTick()
{
    for (int i = 0; i < 3; i++)
    {
        static int k = 100;
        k++;
        g_out = k;
    }
}
`)
	want := []int32{103, 106, 109}
	for tick, w := range want {
		runStaticTicks(t, r, 1)
		if got := staticGlobalInt(t, r, "g_out"); got != w {
			t.Fatalf("after tick %d: g_out=%d, want %d", tick+1, got, w)
		}
	}
}

// T4: same-name statics in if/else branches are independent storages
// (guards against mangling collisions). Not red pre-fix — a pin for the new
// desugaring (each declaration site gets its own __static_N slot).
func TestStatic_SameNameBranchesIndependent(t *testing.T) {
	r := compileStaticSrc(t, `int g_a = 0;
int g_b = 0;
int g_pick = 0;
void OnTick()
{
    if (g_pick == 0)
    {
        static int n = 0;
        n++;
        g_a = n;
    }
    else
    {
        static int n = 0;
        n = n + 100;
        g_b = n;
    }
    g_pick = 1;
}
`)
	runStaticTicks(t, r, 3) // tick1: if-branch; ticks 2,3: else-branch
	if got := staticGlobalInt(t, r, "g_a"); got != 1 {
		t.Fatalf("if-branch static n: g_a=%d, want 1", got)
	}
	if got := staticGlobalInt(t, r, "g_b"); got != 200 {
		t.Fatalf("else-branch static n shares storage with if-branch: g_b=%d, want 200", got)
	}
}

// T5: an inner plain local of the same name shadows the static inside its
// block; the static resolves again after the block closes (resolution order:
// localScopes before staticScopes per level).
// Pre-fix: static behaves as plain local → g_stat stuck at 6 (RED).
func TestStatic_InnerPlainLocalShadows(t *testing.T) {
	r := compileStaticSrc(t, `int g_inner = 0;
int g_stat = 0;
void OnTick()
{
    static int n = 5;
    {
        int n = 100;
        n++;
        g_inner = n;
    }
    n++;
    g_stat = n;
}
`)
	runStaticTicks(t, r, 3)
	if got := staticGlobalInt(t, r, "g_inner"); got != 101 {
		t.Fatalf("inner plain local shadow failed: g_inner=%d, want 101", got)
	}
	if got := staticGlobalInt(t, r, "g_stat"); got != 8 {
		t.Fatalf("static after block: g_stat=%d, want 8 (6,7,8 across ticks)", got)
	}
}

// T6: static inside a recursive function is shared across all invocations
// (global-storage semantics, not per-frame).
// Pre-fix: re-initialized per invocation → g_out stuck at 3 (RED).
func TestStatic_RecursiveShared(t *testing.T) {
	r := compileStaticSrc(t, `int g_out = 0;
int Recurse(int depth)
{
    static int calls = 0;
    calls++;
    if (depth > 0)
    {
        Recurse(depth - 1);
    }
    return calls;
}
void OnTick()
{
    g_out = Recurse(2);
}
`)
	runStaticTicks(t, r, 3)
	if got := staticGlobalInt(t, r, "g_out"); got != 9 {
		t.Fatalf("recursive static calls=%d, want 9 (3 per tick, shared across frames)", got)
	}
}

// T7: `static double arr[4]` — the array is allocated once (OP_NEW_ARRAY
// inside the init guard) and its contents persist across ticks.
// Pre-fix: re-allocated per tick → g_out stuck at 1 (RED).
func TestStatic_ArrayPersists(t *testing.T) {
	r := compileStaticSrc(t, `int g_out = 0;
void OnTick()
{
    static double arr[4];
    arr[0] = arr[0] + 1;
    g_out = arr[0];
}
`)
	runStaticTicks(t, r, 3)
	if got := staticGlobalInt(t, r, "g_out"); got != 3 {
		t.Fatalf("static array did not persist: g_out=%d, want 3", got)
	}
}

// T8: multi-declarator `static int a=InitA(), b=InitB();` — each declarator
// gets its own init-once guard.
// Pre-fix: both initializers run every tick (RED).
func TestStatic_MultiDeclaratorEachOnce(t *testing.T) {
	r := compileStaticSrc(t, `int g_ca = 0;
int g_cb = 0;
int g_a = 0;
int g_b = 0;
int InitA()
{
    g_ca++;
    return 10;
}
int InitB()
{
    g_cb++;
    return 20;
}
void OnTick()
{
    static int a = InitA(), b = InitB();
    g_a = a;
    g_b = b;
}
`)
	runStaticTicks(t, r, 3)
	if got := staticGlobalInt(t, r, "g_ca"); got != 1 {
		t.Fatalf("InitA ran %d times, want 1", got)
	}
	if got := staticGlobalInt(t, r, "g_cb"); got != 1 {
		t.Fatalf("InitB ran %d times, want 1", got)
	}
	if got := staticGlobalInt(t, r, "g_a"); got != 10 {
		t.Fatalf("g_a=%d, want 10", got)
	}
	if got := staticGlobalInt(t, r, "g_b"); got != 20 {
		t.Fatalf("g_b=%d, want 20", got)
	}
}

// T9: runtime-dependent initializer (`static int x = InitVal()`) is evaluated
// lazily on first reach of the declaration statement, not at program load —
// tick 1 skips the declaration branch and the initializer must not run.
// Pre-fix: plain local re-evaluates per reach (g_seen climbs) and nothing
// pins the lazy semantics (RED on the g_seen==0-after-tick-1 assertion —
// actually pre-fix g_seen is also 0 after tick 1 but 3 after tick 3).
func TestStatic_LazyFirstReachInit(t *testing.T) {
	r := compileStaticSrc(t, `int g_seen = 0;
int g_out = 0;
int InitVal()
{
    g_seen = g_seen + 1;
    return g_seen;
}
void OnTick()
{
    if (g_out == 0)
    {
        g_out = -1;
        return;
    }
    static int x = InitVal();
    g_out = x;
}
`)
	runStaticTicks(t, r, 1)
	if got := staticGlobalInt(t, r, "g_seen"); got != 0 {
		t.Fatalf("initializer ran before first reach of the declaration: g_seen=%d, want 0", got)
	}
	runStaticTicks(t, r, 2)
	if got := staticGlobalInt(t, r, "g_out"); got != 1 {
		t.Fatalf("g_out=%d, want 1 (initializer took first-reach value)", got)
	}
	if got := staticGlobalInt(t, r, "g_seen"); got != 1 {
		t.Fatalf("initializer ran %d times after 3 ticks, want 1", got)
	}
}

// T10: non-static storage classes and input/sinput in local position are
// illegal MQL — must be compile errors, never silently dropped (the original
// defect family: silent semantic fabrication).
func TestStatic_LocalIllegalStorageClassRejected(t *testing.T) {
	cases := map[string]string{
		"local extern": `void OnTick() { extern int le = 2; }`,
		"local input":  `void OnTick() { input int li = 1; }`,
		"local sinput": `void OnTick() { sinput int ls = 1; }`,
		"static on function": `static void Foo() { }
void OnTick() { Foo(); }`,
	}
	for name, src := range cases {
		if _, err := CompileMQL(src); err == nil {
			t.Errorf("%s: compiled silently — must be a compile error", name)
		}
	}
	// Top-level static/extern/input keep their existing (correct) behavior.
	ok := map[string]string{
		"top-level static": `static int gTop = 7;
void OnTick() { gTop++; }`,
		"top-level input": `input int gInp = 1;
void OnTick() { gTop2 = gInp; }
int gTop2 = 0;`,
		"top-level extern": `extern int gExt = 2;
void OnTick() { gExt++; }`,
	}
	for name, src := range ok {
		if _, err := CompileMQL(src); err != nil {
			t.Errorf("%s: regressed — must still compile, got %v", name, err)
		}
	}
}

// T11: static reads and writes bind consistently through every expression
// form: s++, s+=2, and pass-by-argument.
// Pre-fix: s resets per tick → g_out stuck at 3 (RED).
func TestStatic_ReadWriteFormsConsistent(t *testing.T) {
	r := compileStaticSrc(t, `int g_out = 0;
int g_via = 0;
int Echo(int v)
{
    return v;
}
void OnTick()
{
    static int s = 0;
    s++;
    s += 2;
    g_via = Echo(s);
    g_out = s;
}
`)
	runStaticTicks(t, r, 3)
	if got := staticGlobalInt(t, r, "g_out"); got != 9 {
		t.Fatalf("s accumulated %d, want 9 (3,6,9 across ticks)", got)
	}
	if got := staticGlobalInt(t, r, "g_via"); got != 9 {
		t.Fatalf("static passed as argument: g_via=%d, want 9", got)
	}
}

// S1 acceptance: the IR carries the static mark on the declaration.
func TestStatic_IRDeclMarked(t *testing.T) {
	ir, err := CompileToIR(`void OnTick()
{
    static int x = 1;
    int y = 2;
}
`)
	if err != nil {
		t.Fatalf("CompileToIR: %v", err)
	}
	marked := map[string]bool{}
	for _, s := range ir.OnTick {
		if s.Expr != nil && s.Expr.Kind == interp.ExprDecl {
			marked[s.Expr.Name] = s.Expr.Static
		}
	}
	if !marked["x"] {
		t.Fatalf("ExprDecl.Static not set for static x: %v", marked)
	}
	if marked["y"] {
		t.Fatalf("ExprDecl.Static set for plain local y: %v", marked)
	}
}
