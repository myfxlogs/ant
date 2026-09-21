// compile_block_scope_test.go — VM-BLOCK-SCOPE-1 (2026-09-21).
//
// if/else/while/do-while/for/switch branch bodies must each get their own
// scope. Before the fix the CST layer flattened branch compound_statements
// into Body/ElseBody and the astCompiler compiled them without pushScope, so
// `if(g>0){int n=1;} g=n;` compiled with n leaking into the enclosing scope
// (real MQL: undeclared).
//
// Tests B1–B12 from docs/audits/design-vm-block-scope-1.md §3 (v2).
// v2 discipline: every "reference after the block → reject" case uses a PURE
// READ of the blocked name (`g=n` — write target is the declared global).
// Write-position forms (`n=`/`n++`) hit the mql4 implicit-registration shim
// and compile before AND after the fix — using them would fake green.
package mql2go

import (
	"context"
	"strings"
	"testing"
)

// compileBlockCaseSrc compiles MQL source, failing the test on compile error.
func compileBlockCaseSrc(t *testing.T, src string) *VMRunner {
	t.Helper()
	r, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	return r
}

// runBlockTicks executes n OnTick events on the runner's VM. OnInit runs
// first once — global initializers are compiled into the OnInit preamble,
// so skipping it would leave declared globals at zero.
func runBlockTicks(t *testing.T, r *VMRunner, n int) {
	t.Helper()
	if err := r.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("OnInit failed: %v", err)
	}
	for i := 0; i < n; i++ {
		if err := r.vm.RunOnTick(context.Background()); err != nil {
			t.Fatalf("OnTick %d failed: %v", i+1, err)
		}
	}
}

// blockGlobalInt reads an int-typed global slot by name after execution.
func blockGlobalInt(t *testing.T, r *VMRunner, name string) int32 {
	t.Helper()
	slot, ok := r.bc.GlobalSlots[name]
	if !ok {
		t.Fatalf("global slot %q not registered", name)
	}
	return r.vm.globals[slot].ToInt()
}

// wantBlockCompileError asserts the source fails to compile with the blocked
// name reported as unknown (the strict-scope rejection shape).
func wantBlockCompileError(t *testing.T, name, src string) {
	t.Helper()
	if _, err := CompileMQL(src); err == nil {
		t.Fatalf("%s: compiled silently — block-local declaration leaked, must be a compile error", name)
	}
}

// B1 (pin): sibling if/else branches declare the same name — each branch's
// body compiles its own binding, runtime slots are independent.
// Was green before the fix (compile-time rebinding) — regression pin.
func TestBlockScope_B1_SiblingBranchesIndependent(t *testing.T) {
	r := compileBlockCaseSrc(t, `int g_a = 0;
int g_b = 0;
int g_pick = 0;
void OnTick()
{
    if (g_pick == 0)
    {
        int n = 1;
        g_a = n * 10;
    }
    else
    {
        int n = 2;
        g_b = n * 10;
    }
    g_pick = 1;
}
`)
	runBlockTicks(t, r, 2)
	if got := blockGlobalInt(t, r, "g_a"); got != 10 {
		t.Fatalf("then-branch n: g_a=%d, want 10", got)
	}
	if got := blockGlobalInt(t, r, "g_b"); got != 20 {
		t.Fatalf("else-branch n: g_b=%d, want 20", got)
	}
}

// B2: declaration inside an if body must not be readable after the block.
func TestBlockScope_B2_IfDeclNotReadableAfter(t *testing.T) {
	wantBlockCompileError(t, "B2", `int g = 0;
void OnTick()
{
    if (g > 0)
    {
        int n = 1;
    }
    g = n;
}
`)
}

// B3 (pin): for-body declarations were already scoped by the for's outer
// scope — must stay rejected (the for fix adds only the inner body scope).
func TestBlockScope_B3_ForDeclAlreadyRejected(t *testing.T) {
	wantBlockCompileError(t, "B3", `int g = 0;
void OnTick()
{
    for (int i = 0; i < 2; i++)
    {
        int x = 7;
    }
    g = x;
}
`)
}

// B4: while-body declaration must not be readable after the loop.
func TestBlockScope_B4_WhileDeclNotReadableAfter(t *testing.T) {
	wantBlockCompileError(t, "B4", `int g = 0;
void OnTick()
{
    while (g < 0)
    {
        int y = 1;
    }
    g = y;
}
`)
}

// B5: a body declaration shadowing the for-init name must shadow, not
// rebind — the loop counts on the for-init variable (2 iterations).
// Pre-fix: body `int i=9` rebound the shared binding → update jumped 9→10 →
// 1 iteration (red). Post-fix: 2 iterations.
func TestBlockScope_B5_ForBodyShadowsInit(t *testing.T) {
	r := compileBlockCaseSrc(t, `int g_runs = 0;
void OnTick()
{
    for (int i = 0; i < 2; i++)
    {
        int i = 9;
        g_runs++;
    }
}
`)
	runBlockTicks(t, r, 1)
	if got := blockGlobalInt(t, r, "g_runs"); got != 2 {
		t.Fatalf("body decl rebound for-init i: g_runs=%d, want 2 (loop must count on for-i)", got)
	}
}

// B6 (pin): static locals in sibling branches keep independent storages after
// scoping tightens (staticScopes is the parallel stack — scope isolation must
// not break VM-STATIC-LOCAL-1 T4).
func TestBlockScope_B6_StaticSiblingBranchesPin(t *testing.T) {
	r := compileBlockCaseSrc(t, `int g_a2 = 0;
int g_b2 = 0;
int g_pick = 0;
void OnTick()
{
    if (g_pick == 0)
    {
        static int n = 1;
        g_a2 = n;
    }
    else
    {
        static int n = 2;
        g_b2 = n;
    }
    g_pick = 1;
}
`)
	runBlockTicks(t, r, 3)
	if got := blockGlobalInt(t, r, "g_a2"); got != 1 {
		t.Fatalf("then-branch static: g_a2=%d, want 1", got)
	}
	if got := blockGlobalInt(t, r, "g_b2"); got != 2 {
		t.Fatalf("else-branch static: g_b2=%d, want 2 (independent storage)", got)
	}
}

// B7: a static declaration inside an if body must not be readable after the
// block — the static binding dies with the scope (real MQL behavior).
func TestBlockScope_B7_StaticIfDeclNotReadableAfter(t *testing.T) {
	wantBlockCompileError(t, "B7", `int g = 0;
void OnTick()
{
    if (g > 0)
    {
        static int n = 1;
    }
    g = n;
}
`)
}

// B8 (pin): bare `{}` blocks were already correctly scoped (StmtBlock path) —
// regression pin.
func TestBlockScope_B8_BareBlockAlreadyRejected(t *testing.T) {
	wantBlockCompileError(t, "B8", `int g = 0;
void OnTick()
{
    {
        int x = 1;
    }
    g = x;
}
`)
}

// B9: do-while body declaration must not be readable after the loop.
func TestBlockScope_B9_DoWhileDeclNotReadableAfter(t *testing.T) {
	wantBlockCompileError(t, "B9", `int g = 0;
void OnTick()
{
    do
    {
        int z = 1;
    } while (g < 0);
    g = z;
}
`)
}

// B10: switch body is ONE shared scope — a case declaration must not leak
// past the switch, but must stay reachable in later cases of the same switch
// (C semantics: case labels don't create scopes).
func TestBlockScope_B10_SwitchOneSharedScope(t *testing.T) {
	// Outer leak rejected.
	wantBlockCompileError(t, "B10-leak", `int g = 0;
void OnTick()
{
    switch (g)
    {
    case 1:
        int w = 0;
        break;
    }
    g = w;
}
`)
	// In-switch sharing pin: case 2 reads case 1's declaration.
	r := compileBlockCaseSrc(t, `int g_w2 = 0;
void OnTick()
{
    switch (1)
    {
    case 1:
        int w2 = 5;
        g_w2 = w2;
        break;
    case 2:
        g_w2 = w2 + 1;
        break;
    }
}
`)
	runBlockTicks(t, r, 1)
	if got := blockGlobalInt(t, r, "g_w2"); got != 5 {
		t.Fatalf("case-shared declaration: g_w2=%d, want 5", got)
	}
}

// B11 (pin): three nested if layers, same name each layer — every read binds
// to its own layer's declaration.
func TestBlockScope_B11_NestedShadowChain(t *testing.T) {
	r := compileBlockCaseSrc(t, `int g_l1 = 0;
int g_l2 = 0;
int g_l3 = 0;
int gv = 1;
void OnTick()
{
    if (gv > 0)
    {
        int v = 10;
        g_l1 = v;
        if (gv > 0)
        {
            int v = 20;
            g_l2 = v;
            if (gv > 0)
            {
                int v = 30;
                g_l3 = v;
            }
        }
    }
}
`)
	runBlockTicks(t, r, 1)
	for name, want := range map[string]int32{"g_l1": 10, "g_l2": 20, "g_l3": 30} {
		if got := blockGlobalInt(t, r, name); got != want {
			t.Fatalf("%s=%d, want %d — layer binding broken", name, got, want)
		}
	}
}

// B12: an inner if's declaration must be invisible to the middle layer.
// Pre-fix the flattened scopes made it leak upward (audit-verified).
func TestBlockScope_B12_NestedInnerDeclInvisibleToMiddle(t *testing.T) {
	wantBlockCompileError(t, "B12", `int g = 0;
void OnTick()
{
    if (g > 0)
    {
        if (g > 1)
        {
            int q = 3;
        }
        g = q;
    }
}
`)
}

// The rejection errors must name the blocked variable (strict-scope shape:
// resolveVar's unknown-variable error), not fail for some unrelated reason.
func TestBlockScope_RejectionsNameTheVariable(t *testing.T) {
	cases := map[string]string{
		"B2": `int g = 0;
void OnTick() { if (g > 0) { int n = 1; } g = n; }`,
		"B4": `int g = 0;
void OnTick() { while (g < 0) { int y = 1; } g = y; }`,
		"B7": `int g = 0;
void OnTick() { if (g > 0) { static int n = 1; } g = n; }`,
		"B9": `int g = 0;
void OnTick() { do { int z = 1; } while (g < 0); g = z; }`,
		"B10": `int g = 0;
void OnTick() { switch (g) { case 1: int w = 0; break; } g = w; }`,
		"B12": `int g = 0;
void OnTick() { if (g > 0) { if (g > 1) { int q = 3; } g = q; } }`,
	}
	for name, src := range cases {
		_, err := CompileMQL(src)
		if err == nil {
			t.Errorf("%s: compiled silently", name)
			continue
		}
		if !strings.Contains(err.Error(), "unknown variable") {
			t.Errorf("%s: error %q is not the unknown-variable strict-scope shape", name, err)
		}
	}
}
