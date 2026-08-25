package mql2go

import (
	"strings"
	"testing"
)

// ── VM-COMPILER-SEMANTICS-4 round 4 behavior tests ───────────────────
// Tests for: invalid declaration rejection, error-recovery fixtures,
// HasError guard with input/extern false-positive allowance.

// TestCompileMQL_InvalidDeclarationMissingInitializer verifies that
// "int x = ;" is rejected — the initializer is missing.
// VM-COMPILER-SEMANTICS-4 round 4: previously silently accepted because
// the compiler skipped ERROR nodes in declarations.
func TestCompileMQL_InvalidDeclarationMissingInitializer(t *testing.T) {
	_, err := CompileMQL("int x = ;")
	if err == nil {
		t.Fatal("CompileMQL should reject 'int x = ;' (missing initializer)")
	}
	if !strings.Contains(err.Error(), "syntax error") {
		t.Fatalf("error should mention syntax error, got: %s", err.Error())
	}
}

// TestCompileMQL_InvalidDeclarationMissingOperand verifies that
// "int x = 1 + ;" is rejected — the right operand is missing.
func TestCompileMQL_InvalidDeclarationMissingOperand(t *testing.T) {
	_, err := CompileMQL("int x = 1 + ;")
	if err == nil {
		t.Fatal("CompileMQL should reject 'int x = 1 + ;' (missing operand)")
	}
	if !strings.Contains(err.Error(), "syntax error") {
		t.Fatalf("error should mention syntax error, got: %s", err.Error())
	}
}

// TestCompileMQL_InvalidDeclarationMissingSemicolon verifies that
// "int x = 1" (missing semicolon) is rejected.
func TestCompileMQL_InvalidDeclarationMissingSemicolon(t *testing.T) {
	_, err := CompileMQL("int x = 1")
	if err == nil {
		t.Fatal("CompileMQL should reject 'int x = 1' (missing semicolon)")
	}
}

// TestCompileMQL_InvalidFunctionMissingBody verifies that a function
// declaration without a body is rejected.
func TestCompileMQL_InvalidFunctionMissingBody(t *testing.T) {
	_, err := CompileMQL("int OnInit() ")
	if err == nil {
		t.Fatal("CompileMQL should reject function without body")
	}
}

// TestCompileMQL_ValidInputDeclarationAccepted verifies that "input int X = 5;"
// is accepted despite tree-sitter HasError=true (known false positive).
// VM-COMPILER-SEMANTICS-4 round 4: input/extern declarations are allowed
// because the compiler handles them via collectParam fallback.
func TestCompileMQL_ValidInputDeclarationAccepted(t *testing.T) {
	_, err := CompileMQL("input int X = 5;\nint OnInit() { return 0; }")
	if err != nil {
		t.Fatalf("CompileMQL should accept valid input declaration, got: %v", err)
	}
}

// TestCompileMQL_ValidExternDeclarationAccepted verifies that
// "extern double Lots = 0.1;" is accepted despite tree-sitter HasError.
func TestCompileMQL_ValidExternDeclarationAccepted(t *testing.T) {
	_, err := CompileMQL("extern double Lots = 0.1;\nint OnInit() { return 0; }")
	if err != nil {
		t.Fatalf("CompileMQL should accept valid extern declaration, got: %v", err)
	}
}

// TestCompileMQL_ValidMQL5WithIncludeAccepted verifies that MQL5 source
// with #include <Trade/Trade.mqh> is accepted (preprocessor injects stub).
func TestCompileMQL_ValidMQL5WithIncludeAccepted(t *testing.T) {
	source := `#include <Trade/Trade.mqh>
input int MagicNumber = 12345;

CTrade trade;

int OnInit() { return 0; }`
	_, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL should accept valid MQL5 with #include, got: %v", err)
	}
}

// TestCompileMQL_CompletelyInvalidSourceRejected verifies that completely
// invalid source (no valid MQL tokens) is rejected.
func TestCompileMQL_CompletelyInvalidSourceRejected(t *testing.T) {
	sources := []string{
		"}}}(((///",
		"!!!@@@###",
		"\x00\x01\x02",
		"@@@$$$%%%",
	}
	for _, s := range sources {
		_, err := CompileMQL(s)
		if err == nil {
			t.Errorf("CompileMQL should reject invalid source %q", s)
		}
	}
}

// TestCompileMQL_EmptySourceAccepted verifies that empty source is accepted
// (no declarations = no errors).
func TestCompileMQL_EmptySourceAccepted(t *testing.T) {
	sources := []string{"", "   ", "\n\n\n"}
	for _, s := range sources {
		_, err := CompileMQL(s)
		if err != nil {
			t.Errorf("CompileMQL should accept empty source %q, got: %v", s, err)
		}
	}
}

// TestCompileMQL_ErrorRecoveryValidAfterInvalid verifies that the compiler
// does not accept source with a mix of valid and invalid declarations.
// VM-COMPILER-SEMANTICS-4 round 4: error recovery must not silently skip
// invalid declarations and accept the valid ones.
func TestCompileMQL_ErrorRecoveryValidAfterInvalid(t *testing.T) {
	source := `int x = ;
int OnInit() { return 0; }`
	_, err := CompileMQL(source)
	if err == nil {
		t.Fatal("CompileMQL should reject source with invalid declaration even if valid declarations follow")
	}
}

// TestCompileMQL_ErrorRecoveryInvalidAfterValid verifies that the compiler
// rejects source with valid declarations followed by invalid ones.
func TestCompileMQL_ErrorRecoveryInvalidAfterValid(t *testing.T) {
	source := `int OnInit() { return 0; }
int x = 1 + ;`
	_, err := CompileMQL(source)
	if err == nil {
		t.Fatal("CompileMQL should reject source with invalid declaration even if it follows valid declarations")
	}
}

// TestCompileToIR_HasErrorGuardRejectsInvalid verifies that CompileToIR
// rejects invalid source via the HasError guard.
func TestCompileToIR_HasErrorGuardRejectsInvalid(t *testing.T) {
	_, err := CompileToIR("int x = ;")
	if err == nil {
		t.Fatal("CompileToIR should reject invalid source via HasError guard")
	}
	if !strings.Contains(err.Error(), "syntax error") {
		t.Fatalf("error should mention syntax error, got: %s", err.Error())
	}
}

// TestCompileToIR_HasErrorGuardAllowsInput verifies that CompileToIR
// allows input declarations despite HasError=true (known false positive).
func TestCompileToIR_HasErrorGuardAllowsInput(t *testing.T) {
	_, err := CompileToIR("input int X = 5;\nint OnInit() { return 0; }")
	if err != nil {
		t.Fatalf("CompileToIR should allow input declaration, got: %v", err)
	}
}

// TestCompileMQL_ErrorMessageContainsNodeInfo verifies that the error
// message contains the node type and source text for debugging.
func TestCompileMQL_ErrorMessageContainsNodeInfo(t *testing.T) {
	_, err := CompileMQL("int x = ;")
	if err == nil {
		t.Fatal("should fail")
	}
	if !strings.Contains(err.Error(), "declaration") {
		t.Fatalf("error should mention node type 'declaration', got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "int x = ;") {
		t.Fatalf("error should contain source text, got: %s", err.Error())
	}
}

// ── VM-COMPILER-SEMANTICS-4 round 5: structured input/extern exception ──
// These tests verify that the strings.Contains-based exception is replaced
// with structured node-type detection. Invalid input/extern declarations
// that were previously accepted (because strings.Contains matched "input "
// or "extern " in the source text) are now rejected.

// TestCompileMQL_InvalidInputMissingInitializer verifies that
// "input int X = ;" is rejected — the initializer is missing.
// VM-COMPILER-SEMANTICS-4 round 5: strings.Contains("input int X = ;", "input ")
// returned true → old code allowed it. Structured check detects the empty
// initializer in the init_declarator and rejects it.
func TestCompileMQL_InvalidInputMissingInitializer(t *testing.T) {
	_, err := CompileMQL("input int X = ;")
	if err == nil {
		t.Fatal("CompileMQL should reject 'input int X = ;' (missing initializer)")
	}
	if !strings.Contains(err.Error(), "input declaration") {
		t.Fatalf("error should mention 'input declaration', got: %s", err.Error())
	}
}

// TestCompileMQL_InvalidInputLongNameMissingInitializer verifies that
// "input int MagicNumber = ;" is rejected (long name pattern B).
func TestCompileMQL_InvalidInputLongNameMissingInitializer(t *testing.T) {
	_, err := CompileMQL("input int MagicNumber = ;")
	if err == nil {
		t.Fatal("CompileMQL should reject 'input int MagicNumber = ;' (missing initializer)")
	}
	if !strings.Contains(err.Error(), "input declaration") {
		t.Fatalf("error should mention 'input declaration', got: %s", err.Error())
	}
}

// TestCompileMQL_InvalidExternMissingInitializer verifies that
// "extern int X = ;" is rejected — the initializer is missing.
// VM-COMPILER-SEMANTICS-4 round 5: strings.Contains("extern int X = ;", "extern ")
// returned true → old code allowed it. Structured check detects that
// extern declarations with HasError=true are real syntax errors.
func TestCompileMQL_InvalidExternMissingInitializer(t *testing.T) {
	_, err := CompileMQL("extern int X = ;")
	if err == nil {
		t.Fatal("CompileMQL should reject 'extern int X = ;' (missing initializer)")
	}
	if !strings.Contains(err.Error(), "extern declaration") {
		t.Fatalf("error should mention 'extern declaration', got: %s", err.Error())
	}
}

// TestCompileMQL_InvalidInputAsValue verifies that "int x = input ;"
// is NOT accepted as an input declaration. The old strings.Contains check
// matched "input " in the source text and allowed it. The structured check
// verifies the first named child is type_identifier "input", which is not
// the case for "int x = input ;" (first named child is primitive_type "int").
func TestCompileMQL_InvalidInputAsValue(t *testing.T) {
	_, err := CompileMQL("int x = input ;")
	if err == nil {
		t.Fatal("CompileMQL should reject 'int x = input ;' (input used as value, not a declaration)")
	}
}

// TestCompileMQL_ValidInputNoInitializer verifies that "input int X;"
// (no default value) is accepted — this is valid MQL5.
func TestCompileMQL_ValidInputNoInitializer(t *testing.T) {
	_, err := CompileMQL("input int X;\nint OnInit() { return 0; }")
	if err != nil {
		t.Fatalf("CompileMQL should accept 'input int X;' (no default value is valid MQL5), got: %v", err)
	}
}
