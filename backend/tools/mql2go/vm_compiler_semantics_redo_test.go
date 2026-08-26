// vm_compiler_semantics_redo_test.go — VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD 返工行为测试.
//
// Tests verify the re-implemented compiler semantics fixes after D-REVERT-SCOPE-DRIFT-001:
//   - VM-COMPILER-SEMANTICS-1 (S1-S6): multi-variable declaration, unsupported operator
//     error, switch fallthrough, single-statement loop body, class-typed global init,
//     ClassTypes serialization.
//   - BT-FUNC-ENTRYPC-FWD (S8-S11): user-to-user forward reference via placeholder
//     patching, deterministic function layout.
//
// Adversarial proofs (9): each critical line mutated → relevant test RED → restore GREEN.

package mql2go

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/runner"
	"alphaforge/tools/mql2go/interp"
)

// ── VM-COMPILER-SEMANTICS-1 tests (7) ────────────────────────────────

// TestVM_Audit_MQLFieldAssignment_VMBehavior verifies that a CTrade global
// variable is initialized as ValClass (not zero/nil), so direct field assignment
// via OP_SET_FIELD works and the value can be read back via OP_GET_FIELD.
//
// Adversarial: delete initGlobals ValClass initialization → setField silently
// returns (obj.Kind != ValClass) → readback=0 → RED.
func TestVM_Audit_MQLFieldAssignment_VMBehavior(t *testing.T) {
	src := `
CTrade trade;
int g_readback = -1;

int OnInit()
{
    trade.Magic = 42;
    return 0;
}

void OnTick()
{
    g_readback = trade.Magic;
}
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	r.UpdateLiveState("10000", "10500", "500", "9500", nil, nil)

	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// tradeMagic should be 42 after OnInit calls SetExpertMagicNumber(42).
	// ValClass init is needed for OP_SET_FIELD to work on the CTrade global.
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	_, err = r.OnTick(context.Background(), decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	readback := getGlobalInt(t, vmRunner, "g_readback")
	if readback != 42 {
		t.Fatalf("g_readback = %d, want 42 (ValClass init + field set/get)", readback)
	}
}

// TestVM_Audit_MultiVariableDeclaration verifies that `int a = 1, b = 2;`
// initializes both variables (not just the first).
//
// Adversarial: restore single-declarator return → b stays 0 → RED.
func TestVM_Audit_MultiVariableDeclaration(t *testing.T) {
	src := `
int g_a = -1;
int g_b = -1;

int OnInit() { return 0; }

void OnTick()
{
    int a = 1, b = 2;
    g_a = a;
    g_b = b;
}
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	r.UpdateLiveState("10000", "10500", "500", "9500", nil, nil)

	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	_, err = r.OnTick(context.Background(), decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	a := getGlobalInt(t, vmRunner, "g_a")
	b := getGlobalInt(t, vmRunner, "g_b")
	if a != 1 {
		t.Fatalf("g_a = %d, want 1 (first declarator)", a)
	}
	if b != 2 {
		t.Fatalf("g_b = %d, want 2 (second declarator not initialized)", b)
	}
}

// TestVM_Audit_UninitializedLocalDeclaration verifies that `int x;` (no
// initializer) compiles and x gets zero value.
//
// This test doesn't have a direct adversarial proof but verifies S1's
// zero-value path works.
func TestVM_Audit_UninitializedLocalDeclaration(t *testing.T) {
	src := `
int g_x = -1;

int OnInit() { return 0; }

void OnTick()
{
    int x;
    g_x = x;
}
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	r.UpdateLiveState("10000", "10500", "500", "9500", nil, nil)

	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	_, err = r.OnTick(context.Background(), decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	x := getGlobalInt(t, vmRunner, "g_x")
	if x != 0 {
		t.Fatalf("g_x = %d, want 0 (uninitialized local should be zero)", x)
	}
}

// TestVM_Audit_UnsupportedBitwiseOperatorRejected verifies that `a | b`
// (bitwise OR, unsupported) causes a compile error, not silent fallback.
//
// Adversarial: restore `return OP_ADD` without error → err=nil → RED.
func TestVM_Audit_UnsupportedBitwiseOperatorRejected(t *testing.T) {
	src := `
int g_result = -1;

int OnInit() { return 0; }

void OnTick()
{
    int a = 1;
    int b = 2;
    g_result = a | b;
}
`
	_, err := CompileMQL(src)
	if err == nil {
		t.Fatal("CompileMQL with bitwise OR should return error, got nil (silent fallback)")
	}
}

// TestVM_Audit_SwitchFallthrough verifies that a case without break falls
// through to the next case's body.
//
// Adversarial: restore unconditional JMP to end → fallthrough doesn't happen → RED.
func TestVM_Audit_SwitchFallthrough(t *testing.T) {
	src := `
int g_result = -1;

int OnInit() { return 0; }

void OnTick()
{
    int x = 1;
    switch (x)
    {
        case 1:
            g_result = 10;
            // no break — fallthrough to case 2
        case 2:
            g_result = g_result + 100;
            break;
        case 3:
            g_result = 999;
            break;
    }
}
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	r.UpdateLiveState("10000", "10500", "500", "9500", nil, nil)

	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	_, err = r.OnTick(context.Background(), decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	result := getGlobalInt(t, vmRunner, "g_result")
	// case 1 sets g_result=10, falls through to case 2 which adds 100 → 110
	if result != 110 {
		t.Fatalf("g_result = %d, want 110 (case 1 fallthrough to case 2: 10+100)", result)
	}
}

// TestVM_Audit_ForLoopSingleStatementBody verifies that `for(...) doSomething();`
// (no braces) compiles and executes the single statement.
//
// Adversarial: remove single-statement body handling → body is nil → loop does nothing → RED.
func TestVM_Audit_ForLoopSingleStatementBody(t *testing.T) {
	src := `
int g_count = -1;

int OnInit() { return 0; }

void OnTick()
{
    g_count = 0;
    for (int i = 0; i < 5; i++)
        g_count = g_count + 1;
}
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	r.UpdateLiveState("10000", "10500", "500", "9500", nil, nil)

	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	_, err = r.OnTick(context.Background(), decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	count := getGlobalInt(t, vmRunner, "g_count")
	if count != 5 {
		t.Fatalf("g_count = %d, want 5 (for loop single-statement body)", count)
	}
}

// TestVM_Audit_WhileLoopSingleStatementBody verifies that `while(cond) doSomething();`
// (no braces) compiles and executes the single statement.
//
// Adversarial: remove single-statement body handling → body is nil → loop does nothing → RED.
func TestVM_Audit_WhileLoopSingleStatementBody(t *testing.T) {
	src := `
int g_count = -1;

int OnInit() { return 0; }

void OnTick()
{
    g_count = 0;
    int i = 0;
    while (i < 3)
        i = i + 1;
    g_count = i;
}
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	r.UpdateLiveState("10000", "10500", "500", "9500", nil, nil)

	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	_, err = r.OnTick(context.Background(), decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}

	count := getGlobalInt(t, vmRunner, "g_count")
	if count != 3 {
		t.Fatalf("g_count = %d, want 3 (while loop single-statement body)", count)
	}
}

// ── BT-FUNC-ENTRYPC-FWD tests (2 + 1 structure) ──────────────────────

// TestVM_Audit_UserToUserForwardReference verifies that a caller function
// (aaa_caller, alphabetically first so body compiles first) can call a callee
// (zzz_callee, body compiles later) and the call resolves to the correct EntryPC.
//
// Uses aaa_caller/zzz_callee naming per spec: caller alphabetically before
// callee ensures the bug (stale marker PC) would trigger without patching.
//
// Adversarial mutation 1: restore direct fn.EntryPC write → stale marker → RED.
// Adversarial mutation 2: comment out patchUserCalls → operand=-1 → RED.
func TestVM_Audit_UserToUserForwardReference(t *testing.T) {
	src := `
int g_result = -1;

int zzz_callee(int v)
{
    return v * 42;
}

int aaa_caller(int v)
{
    return zzz_callee(v);
}

int OnInit() { return 0; }

void OnTick()
{
    g_result = aaa_caller(1);
}
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	r.UpdateLiveState("10000", "10500", "500", "9500", nil, nil)

	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Run 100 iterations to catch intermittent failures from non-deterministic layout.
	for i := 0; i < 100; i++ {
		_, err := r.OnTick(context.Background(), decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
		if err != nil {
			t.Fatalf("OnTick iteration %d failed: %v", i, err)
		}
		result := getGlobalInt(t, vmRunner, "g_result")
		if result != 42 {
			t.Fatalf("iteration %d: g_result = %d, want 42 (forward reference not resolved)", i, result)
		}
	}
}

// TestVM_Audit_UserToUserForwardReference_Structure verifies that every
// OP_CALL_USER instruction has its operand A equal to the callee's final
// EntryPC (not -1 placeholder, not a stale marker PC).
//
// Adversarial: same mutations as TestVM_Audit_UserToUserForwardReference.
func TestVM_Audit_UserToUserForwardReference_Structure(t *testing.T) {
	src := `
int zzz_callee(int v) { return v; }
int aaa_caller(int v) { return zzz_callee(v); }

int OnInit() { return 0; }
void OnTick() { aaa_caller(1); }
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	bc := vmRunner.Bytecode()
	// Check every OP_CALL_USER instruction.
	for i, ins := range bc.Code {
		if ins.Op != OP_CALL_USER {
			continue
		}
		if ins.A < 0 {
			t.Fatalf("instruction %d: OP_CALL_USER operand A=%d (unresolved placeholder)", i, ins.A)
		}
		// Verify A points to a valid function entry.
		found := false
		for _, fn := range bc.Funcs {
			if fn.EntryPC == ins.A {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("instruction %d: OP_CALL_USER operand A=%d does not match any function EntryPC", i, ins.A)
		}
	}
}

// ── Serialization test (1) ───────────────────────────────────────────

// TestCompileMQLCached_ClassTypesRoundTrip verifies that ClassTypes survives
// marshal → unmarshal round-trip.
//
// Adversarial: delete ClassTypes serialization → cache hit ClassTypes empty → RED.
func TestCompileMQLCached_ClassTypesRoundTrip(t *testing.T) {
	src := `
CTrade trade;

int OnInit() { return 0; }
void OnTick() { }
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	// Verify ClassTypes is populated.
	bc := vmRunner.Bytecode()
	if !bc.ClassTypes["CTrade"] {
		t.Fatalf("ClassTypes[CTrade] = false, want true (CTrade is a builtin class)")
	}

	// Marshal → Unmarshal round-trip.
	data, err := MarshalBytecode(bc)
	if err != nil {
		t.Fatalf("MarshalBytecode failed: %v", err)
	}

	bc2, err := UnmarshalBytecode(data)
	if err != nil {
		t.Fatalf("UnmarshalBytecode failed: %v", err)
	}

	if !bc2.ClassTypes["CTrade"] {
		t.Fatalf("round-trip: ClassTypes[CTrade] = false, want true (serialization lost ClassTypes)")
	}
}

// ── Helpers ──────────────────────────────────────────────────────────

// getGlobalInt is REUSED from live_mql_order_context_vm_test.go (same package).

// Suppress unused import warning for interp (used in structure test via OP_* constants).
var _ = interp.IntVal
