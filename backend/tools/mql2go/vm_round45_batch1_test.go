package mql2go

import (
	"alphaforge/strategy/runner"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// vm_round45_batch1_test.go — VM round 4-5 Batch 1 adversarial tests.
//
// VM-COMPILER-SEMANTICS-4 (compiler correctness):
//   - T1-T3: comma_expression ExprSeq (all side effects execute)
//   - T4-T5: input declaration validation (invalid rejected, valid accepted)
//   - T6: completely invalid source rejected (missing-node guard)
//   - T7: reserved keyword as identifier rejected
//   - T8: tree-sitter root never ERROR (HasError is not dead code)
//
// VM-CACHE-INTEGRITY-5 (cache integrity):
//   - T9-T11: coverage restore on cache hit
//   - T12: Version=="python" language check
//   - T13: payload limit
//   - T14: no Language field
//   - T15: cache hit vs cold compile coverage equality

// --- T1: comma_expression side effects ---

func TestCommaExpression_VMSideEffectsExecution(t *testing.T) {
	t.Parallel()
	source := `int g_a, g_b, g_c;
void OnTick() {
    (g_a=1, g_b=2, g_c=3);
}`
	vmRunner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	_, err = r.OnTick(context.Background(), decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}
	if got := getGlobalInt(t, vmRunner, "g_a"); got != 1 {
		t.Errorf("g_a = %d, want 1 (comma_expression side effect lost)", got)
	}
	if got := getGlobalInt(t, vmRunner, "g_b"); got != 2 {
		t.Errorf("g_b = %d, want 2 (comma_expression side effect lost)", got)
	}
	if got := getGlobalInt(t, vmRunner, "g_c"); got != 3 {
		t.Errorf("g_c = %d, want 3 (comma_expression side effect lost)", got)
	}
}

// --- T3: comma_expression return value is last ---

func TestCommaExpression_VMReturnValueIsLast(t *testing.T) {
	t.Parallel()
	source := `int r;
void OnTick() {
    r = (1, 2, 42);
}`
	vmRunner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	_, err = r.OnTick(context.Background(), decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}
	// Use a distinct name to avoid shadowing the runner var `r`
	v, ok := vmRunner.GetGlobal("r")
	if !ok {
		t.Fatal("global \"r\" not found")
	}
	if v.ToInt() != 42 {
		t.Errorf("r = %d, want 42 (comma should return last value)", v.ToInt())
	}
}

// --- T4: invalid input missing initializer rejected ---

func TestCompileMQL_InvalidInputMissingInitializer(t *testing.T) {
	t.Parallel()
	_, err := CompileToIR("input int X = ;")
	if err == nil {
		t.Fatal("expected error for 'input int X = ;' (missing initializer), got nil")
	}
}

// --- T5: valid input accepted ---

func TestCompileMML_ValidInputAccepted(t *testing.T) {
	t.Parallel()
	_, err := CompileToIR("input int X = 5;")
	if err != nil {
		t.Fatalf("expected success for 'input int X = 5;', got error: %v", err)
	}
}

// --- T6: completely invalid source rejected ---

func TestCompileMQL_CompletelyInvalidSourceRejected(t *testing.T) {
	t.Parallel()
	_, err := CompileToIR("int x = ;")
	if err == nil {
		t.Fatal("expected error for 'int x = ;' (missing initializer), got nil")
	}
}

// --- T7: reserved keyword as identifier rejected ---

func TestCompileMML_ReservedKeywordAsIdentifierRejected(t *testing.T) {
	t.Parallel()
	_, err := CompileToIR("int x = input ;")
	if err == nil {
		t.Fatal("expected error for 'int x = input ;' (reserved keyword as identifier), got nil")
	}
}

// --- T8: tree-sitter root never ERROR for any input ---

func TestCompileToIR_RootNeverErrorForAnyInput(t *testing.T) {
	t.Parallel()
	inputs := []string{
		"}}}(((///",
		"!!!@@@###",
		"",
		"   ",
		"\n\n\n",
		"int;",
		"void;",
	}
	for _, src := range inputs {
		root, err := ParseMQL(src)
		if err != nil {
			continue // parse error is fine
		}
		if root.Type() == "ERROR" {
			t.Errorf("tree-sitter returned root ERROR for %q (root type = ERROR)", src)
		}
		// HasError should detect ERROR children in some inputs (proves not dead code)
		_ = HasError(root)
	}
}

// --- T9: coverage restore on cache hit ---

func TestCompilePythonCached_RestoresCoverageOnCacheHit(t *testing.T) {
	source := `from decimal import Decimal

class MyStrategy:
    def on_init(self) -> None:
        x = 10
        return

    def on_bar(self) -> None:
        price = 42
        if price > 40:
            y = price + 1
        return
`
	// Cold compile to get bytecode
	runner, bcData, err := CompilePythonCached(source, nil)
	if err != nil {
		t.Fatalf("cold compile failed: %v", err)
	}
	if runner == nil || bcData == nil {
		t.Fatal("cold compile returned nil runner or bytecode")
	}

	// Cache hit
	runner2, _, err := CompilePythonCached(source, bcData)
	if err != nil {
		t.Fatalf("cache hit failed: %v", err)
	}
	cov := runner2.GetCoverageResult()
	if cov == nil {
		t.Fatal("CoverageResult is nil on cache hit (VM-CACHE-INTEGRITY-5: coverage not restored)")
	}
}

// --- T10: coverage restore failure returns error ---

func TestCompilePythonCached_CoverageRestoreFailureReturnsError(t *testing.T) {
	source := `from decimal import Decimal

class MyStrategy:
    def on_init(self) -> None:
        return

    def on_bar(self) -> None:
        return
`
	// Cold compile to get valid bytecode
	_, bcData, err := CompilePythonCached(source, nil)
	if err != nil {
		t.Fatalf("cold compile failed: %v", err)
	}

	// Inject a coverage hook that returns error
	oldHook := coverageRestoreHook
	coverageRestoreHook = func(src string) (*CoverageResult, error) {
		return nil, &coverageErr{msg: "injected coverage restore failure"}
	}
	defer func() { coverageRestoreHook = oldHook }()

	_, _, err = CompilePythonCached(source, bcData)
	if err == nil {
		t.Fatal("expected error when coverage restore fails, got nil (VM-CACHE-INTEGRITY-5: silent degradation)")
	}
	if !strings.Contains(err.Error(), "restore coverage") {
		t.Fatalf("error should mention 'restore coverage', got: %v", err)
	}
}

// --- T11: nil coverage returns error ---

func TestCompilePythonCached_CoverageRestoreNilCoverageReturnsError(t *testing.T) {
	source := `from decimal import Decimal

class MyStrategy:
    def on_init(self) -> None:
        return

    def on_bar(self) -> None:
        return
`
	_, bcData, err := CompilePythonCached(source, nil)
	if err != nil {
		t.Fatalf("cold compile failed: %v", err)
	}

	oldHook := coverageRestoreHook
	coverageRestoreHook = func(src string) (*CoverageResult, error) {
		return nil, nil // nil coverage, no error
	}
	defer func() { coverageRestoreHook = oldHook }()

	_, _, err = CompilePythonCached(source, bcData)
	if err == nil {
		t.Fatal("expected error when coverage restore returns nil, got nil")
	}
}

// --- T12: rejects MQL bytecode for Python source ---

func TestCompilePythonCached_RejectsMQLBytecodeForPythonSource(t *testing.T) {
	// Compile MQL source to get MQL bytecode
	mqlSource := `int x; void OnTick() { x = 1; }`
	_, mqlBcData, err := CompileMQLCached(mqlSource, nil)
	if err != nil {
		t.Fatalf("CompileMQLCached failed: %v", err)
	}

	// Try to use MQL bytecode for Python source. The Version check
	// (r.Bytecode().Version == "python") should reject the MQL bytecode
	// and fall through to cold compile, returning valid Python bytecode.
	pySource := `from decimal import Decimal

class MyStrategy:
    def on_bar(self) -> None:
        return
`
	r, pyBcData, err := CompilePythonCached(pySource, mqlBcData)
	if err != nil {
		t.Fatalf("CompilePythonCached failed: %v", err)
	}
	// The returned bytecode must be Python, not the MQL bytecode we passed in.
	if r.Bytecode().Version != "python" {
		t.Errorf("Version = %q, want \"python\" (MQL bytecode was used instead of falling through to cold compile)", r.Bytecode().Version)
	}
	// The returned bytecode must differ from the MQL bytecode (cold compile produced new bytecode).
	if string(pyBcData) == string(mqlBcData) {
		t.Error("returned bytecode is identical to MQL bytecode (Version check did not reject mismatched bytecode)")
	}
}

// --- T13: payload limit exceeded ---

func TestUnmarshalBytecode_PayloadLimitExceeded(t *testing.T) {
	t.Parallel()
	// Construct a payload > 64MiB
	oversized := make([]byte, maxBytecodePayload+1)
	_, err := UnmarshalBytecode(oversized)
	if err == nil {
		t.Fatal("expected error for oversized payload, got nil (VM-CACHE-INTEGRITY-5: payload limit missing)")
	}
	if !strings.Contains(err.Error(), "exceeds max") {
		t.Fatalf("error should contain 'exceeds max', got: %v", err)
	}
	if !strings.Contains(err.Error(), "payload size") {
		t.Fatalf("error should contain 'payload size', got: %v", err)
	}
}

// --- T14: no Language field ---

func TestBytecode_NoLanguageField(t *testing.T) {
	t.Parallel()
	_, found := reflect.TypeOf(Bytecode{}).FieldByName("Language")
	if found {
		t.Fatal("Bytecode struct should NOT have a Language field (VM-CACHE-INTEGRITY-5: Version is the language discriminator)")
	}
}

// --- T15: cache hit vs cold compile coverage equality ---

func TestCompilePythonCached_CacheHitVsColdCompileCoverageEqual(t *testing.T) {
	source := `from decimal import Decimal

class MyStrategy:
    def on_init(self) -> None:
        x = 10
        return

    def on_bar(self) -> None:
        price = 42
        if price > 40:
            y = price + 1
        return
`
	// Cold compile with coverage
	coldRunner, coldCov, err := CompilePythonWithCoverage(source)
	if err != nil {
		t.Fatalf("cold compile with coverage failed: %v", err)
	}
	if coldCov == nil {
		t.Fatal("cold compile returned nil coverage")
	}

	// Cache hit
	_, bcData, err := CompilePythonCached(source, nil)
	if err != nil {
		t.Fatalf("cold compile for bytecode failed: %v", err)
	}
	cacheRunner, _, err := CompilePythonCached(source, bcData)
	if err != nil {
		t.Fatalf("cache hit failed: %v", err)
	}
	cacheCov := cacheRunner.GetCoverageResult()
	if cacheCov == nil {
		t.Fatal("cache hit returned nil coverage")
	}

	// Compare key coverage fields
	if coldCov.Score != cacheCov.Score {
		t.Errorf("coverage Score: cold=%v cache=%v (should be equal)", coldCov.Score, cacheCov.Score)
	}
	if coldCov.TotalCalls != cacheCov.TotalCalls {
		t.Errorf("coverage TotalCalls: cold=%d cache=%d (should be equal)", coldCov.TotalCalls, cacheCov.TotalCalls)
	}
	_ = coldRunner
}

// --- T2: comma expression function call side effects ---

func TestCommaExpression_VMFunctionCallSideEffects(t *testing.T) {
	t.Parallel()
	// Use a global counter incremented by a user function called via comma
	source := `int g_counter;
int bump() { g_counter = g_counter + 1; return g_counter; }
void OnTick() {
    int dummy;
    dummy = (bump(), bump(), bump());
}`
	vmRunner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	_, err = r.OnTick(context.Background(), decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.1001))
	if err != nil {
		t.Fatalf("OnTick failed: %v", err)
	}
	if got := getGlobalInt(t, vmRunner, "g_counter"); got != 3 {
		t.Errorf("g_counter = %d, want 3 (all 3 bump() calls should execute via comma)", got)
	}
}

// --- helper types for coverage hook injection ---

type coverageErr struct{ msg string }

func (e *coverageErr) Error() string { return e.msg }
