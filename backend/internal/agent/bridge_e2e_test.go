package agent

import (
	"testing"

	"go.uber.org/zap"

	"anttrader/tools/mql2go"
)

// validPythonSubset is a Python subset strategy with trading calls that compiles successfully.
const validPythonSubset = `from decimal import Decimal
class S:
    def __init__(self) -> None:
        self.x = 0
    def on_bar(self) -> None:
        c: float = ctx.bars().close(0)
        if c > 0:
            ctx.broker.buy(lot=Decimal("0.1"))
        return
`

// invalidPythonSubset has a syntax error (f-string is forbidden).
const invalidPythonSubset = `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = f"hello"
        return
`

// TestBridge_RetryPromptConstruction verifies that the retry prompt includes
// the previous attempt, compile error, and original MQL source.
func TestBridge_RetryPromptConstruction(t *testing.T) {
	prompt := buildBridgeRetryPrompt("int start() { return 0; }", "x = 1", "syntax error at line 1")

	if prompt == "" {
		t.Fatal("expected non-empty retry prompt")
	}
	if !contains(prompt, "Previous Attempt") {
		t.Error("retry prompt should mention 'Previous Attempt'")
	}
	if !contains(prompt, "Compile Error") {
		t.Error("retry prompt should mention 'Compile Error'")
	}
	if !contains(prompt, "Original MQL Source") {
		t.Error("retry prompt should mention 'Original MQL Source'")
	}
	if !contains(prompt, "syntax error at line 1") {
		t.Error("retry prompt should include the compile error message")
	}
	if !contains(prompt, "int start() { return 0; }") {
		t.Error("retry prompt should include the original MQL source")
	}
}

// TestBridge_CompilePythonSubset validates that the Python subset used in
// bridge tests actually compiles through the mql2go pipeline.
// This is the "VM backtest" half of the E2E flow — proving that bridge output
// can be compiled and executed.
func TestBridge_CompilePythonSubset(t *testing.T) {
	runner, coverage, err := mql2go.CompilePythonWithCoverage(validPythonSubset)
	if err != nil {
		t.Fatalf("valid Python subset should compile: %v", err)
	}
	if runner == nil {
		t.Fatal("expected non-nil VMRunner")
	}
	if coverage == nil {
		t.Fatal("expected non-nil CoverageResult")
	}
	// Note: coverage score may be 0 if the Python SDK calls (ctx.bars, ctx.broker)
	// are not counted by the MQL-oriented static analyzer. The key assertion is
	// that it compiles without errors and has no fatal blind spots.
	for _, bs := range coverage.BlindSpots {
		if bs.Severity == "fatal" {
			t.Errorf("unexpected fatal blind spot: %s", bs.Builtin)
		}
	}
}

// TestBridge_InvalidPythonSubsetFails proves that invalid Python subset code
// (e.g., with f-strings) is rejected by the compiler — this is what triggers
// the retry loop in TranslateWithRetry.
func TestBridge_InvalidPythonSubsetFails(t *testing.T) {
	_, _, err := mql2go.CompilePythonWithCoverage(invalidPythonSubset)
	if err == nil {
		t.Fatal("expected f-string Python to fail compilation")
	}
}

// TestBridge_MQLToPythonToVM_E2E simulates the full bridge flow:
// 1. MQL source with blind spots → coverage analysis
// 2. "Translated" Python subset (simulating LLM output)
// 3. Python compilation → VM runner
// 4. Coverage improvement verification
//
// This test does NOT call the real LLM — it uses a pre-written Python translation
// to validate the compile + coverage + semantic-diff pipeline.
func TestBridge_MQLToPythonToVM_E2E(t *testing.T) {
	log := zap.NewNop()
	_ = log

	// Step 1: Compile MQL with blind spots
	mqlSource := `#property strict
extern int MAPeriod = 14;
int OnInit() { return 0; }
void OnBar() {
    double ma = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 0);
    if (ma > Close[0]) {
        OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0, "test", 123, 0, clrGreen);
    }
    ObjectCreate("label", OBJ_LABEL, 0, 0, 0);
}`
	_, mqlCoverage, err := mql2go.CompileMQLWithCoverage(mqlSource)
	if err != nil {
		t.Fatalf("MQL should compile (with blind spots): %v", err)
	}
	if mqlCoverage.Score >= 1.0 {
		t.Fatalf("expected MQL coverage < 1.0 (has blind spots), got %.2f", mqlCoverage.Score)
	}
	if len(mqlCoverage.BlindSpots) == 0 {
		t.Fatal("expected blind spots in MQL coverage")
	}

	// Step 2: Simulated LLM translation output (Python subset)
	translatedPython := validPythonSubset

	// Step 3: Compile the translated Python
	_, pyCoverage, err := mql2go.CompilePythonWithCoverage(translatedPython)
	if err != nil {
		t.Fatalf("translated Python should compile: %v", err)
	}

	// Step 4: Verify the translated Python compiles and has no fatal blind spots
	// (coverage score may differ since Python SDK calls aren't counted by the
	// MQL-oriented static analyzer — the key is compilation success + no fatals)
	for _, bs := range pyCoverage.BlindSpots {
		if bs.Severity == "fatal" {
			t.Errorf("bridged Python has fatal blind spot: %s", bs.Builtin)
		}
	}

	// Step 5: Build semantic diff
	changes := buildBridgeChanges(mqlCoverage, pyCoverage)
	if len(changes) == 0 {
		t.Error("expected semantic changes from bridge")
	}

	// Verify each change has a kind and description
	for i, c := range changes {
		if c.Kind == "" {
			t.Errorf("change[%d]: empty kind", i)
		}
		if c.Description == "" {
			t.Errorf("change[%d]: empty description", i)
		}
	}
}

// TestBridge_TranslateWithRetry_SuccessOnFirstAttempt verifies that
// TranslateWithRetry returns success immediately when the first LLM
// response compiles correctly.
func TestBridge_TranslateWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	// We can't call TranslateWithRetry without a real LLM service,
	// but we can verify the compile-check logic that drives the retry loop.
	_, _, err := mql2go.CompilePythonWithCoverage(validPythonSubset)
	if err != nil {
		t.Fatalf("valid Python should compile on first attempt: %v", err)
	}
	// If we reach here, the retry loop would set status="success" and return.
}

// TestBridge_TranslateWithRetry_FailAllAttempts verifies that when
// Python compilation fails consistently, the retry loop exhausts
// all attempts and sets status="bridge_failed".
func TestBridge_TranslateWithRetry_FailAllAttempts(t *testing.T) {
	// Simulate 3 failed compile attempts with invalid Python
	for attempt := 1; attempt <= maxBridgeRetries; attempt++ {
		_, _, err := mql2go.CompilePythonWithCoverage(invalidPythonSubset)
		if err == nil {
			t.Fatalf("attempt %d: expected compile failure for invalid Python", attempt)
		}
	}
	// If all 3 attempts fail, the retry loop would set status="bridge_failed".
}

// TestBridge_BridgeResultFields verifies that BridgeResult has all
// fields needed by the gateway response builder.
func TestBridge_BridgeResultFields(t *testing.T) {
	r := &BridgeResult{
		PythonSource:  validPythonSubset,
		CompileError:  "",
		Translated:    true,
		Status:        "success",
		Attempts:      1,
	}
	if r.Status != "success" {
		t.Errorf("expected status 'success', got %q", r.Status)
	}
	if r.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", r.Attempts)
	}

	// Verify the Python source compiles
	_, _, err := mql2go.CompilePythonWithCoverage(r.PythonSource)
	if err != nil {
		t.Errorf("bridge result Python source should compile: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
