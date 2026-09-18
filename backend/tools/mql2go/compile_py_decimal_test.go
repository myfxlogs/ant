// compile_py_decimal_test.go — PY-DECIMAL-CTOR-1 (2026-09-18).
//
// The Python `Decimal(...)` constructor used to pass its argument through
// verbatim: `Decimal("0.1")` produced a ValString — bool("0") was true,
// decimal arithmetic silently used 0 (a stop-loss offset became a no-op),
// and Decimal("0.0")==Decimal("0") was a string comparison. The fix
// constant-folds literal arguments into ValDecimal and rejects invalid
// literals at compile time; non-literal arguments compile to the registered
// StringToDouble builtin (runtime conversion).
//
// Adversarial proofs (3, reviewer items): restore `return &args[0]` → the
// string/bool/arithmetic/equality/int-literal tests RED; drop only the
// ValString fold → StringLiteral RED + InvalidLiteral compiles; non-literal
// → passthrough → NonLiteral RED.
package mql2go

import (
	"context"
	"strings"
	"testing"

	"alphaforge/tools/mql2go/interp"

	"github.com/shopspring/decimal"
)

// runDecimalSource compiles python source, runs OnBar, and returns global "r".
func runDecimalSource(t *testing.T, body string) interp.Value {
	t.Helper()
	source := `class S:
    def on_bar(self) -> None:
        self.r = ` + body + `
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
	return v
}

// TestCompilePython_DecimalCtorStringLiteral — Decimal("0.1") must be a
// ValDecimal with the exact value (a passthrough leaves ValString).
func TestCompilePython_DecimalCtorStringLiteral(t *testing.T) {
	v := runDecimalSource(t, `Decimal("0.1")`)
	if v.Kind != interp.ValDecimal {
		t.Fatalf("r kind = %v, want ValDecimal (passthrough produces ValString)", v.Kind)
	}
	if !v.ToDecimal().Equal(decimal.NewFromFloat(0.1)) {
		t.Fatalf("r = %s, want 0.1", v.ToDecimal())
	}
}

// TestCompilePython_DecimalCtorBool — bool(Decimal("0")) must be false
// (passthrough: bool("0") == true since the non-empty string is truthy).
func TestCompilePython_DecimalCtorBool(t *testing.T) {
	v := runDecimalSource(t, `bool(Decimal("0"))`)
	if v.Kind != interp.ValBool {
		t.Fatalf("r kind = %v, want ValBool", v.Kind)
	}
	if v.Bool {
		t.Fatal("bool(Decimal(\"0\")) = true, want false (Decimal zero is falsy)")
	}
}

// TestCompilePython_DecimalCtorArithmetic — decimal values must add exactly.
func TestCompilePython_DecimalCtorArithmetic(t *testing.T) {
	v := runDecimalSource(t, `Decimal("1.5") + Decimal("1.5")`)
	if v.Kind != interp.ValDecimal {
		t.Fatalf("r kind = %v, want ValDecimal", v.Kind)
	}
	if !v.ToDecimal().Equal(decimal.NewFromFloat(3)) {
		t.Fatalf("r = %s, want 3 (0+0 was the passthrough result)", v.ToDecimal())
	}
}

// TestCompilePython_DecimalCtorEquality — numeric equality across string
// spellings ("0.0" vs "0"), which the passthrough compared as strings.
func TestCompilePython_DecimalCtorEquality(t *testing.T) {
	v := runDecimalSource(t, `Decimal("0.0") == Decimal("0")`)
	if v.Kind != interp.ValBool {
		t.Fatalf("r kind = %v, want ValBool", v.Kind)
	}
	if !v.Bool {
		t.Fatal(`Decimal("0.0") == Decimal("0") = false, want true (string comparison leak)`)
	}
}

// TestCompilePython_DecimalCtorIntLiteral — Decimal(5) must fold to a
// ValDecimal (the Kind assertion discriminates the passthrough, which would
// leave ValInt).
func TestCompilePython_DecimalCtorIntLiteral(t *testing.T) {
	v := runDecimalSource(t, `Decimal(5)`)
	if v.Kind != interp.ValDecimal {
		t.Fatalf("r kind = %v, want ValDecimal (passthrough produces ValInt)", v.Kind)
	}
	if !v.ToDecimal().Equal(decimal.NewFromInt(5)) {
		t.Fatalf("r = %s, want 5", v.ToDecimal())
	}
}

// TestCompilePython_DecimalCtorInvalidLiteral — an unparseable string
// literal must be a compile error (fail-closed), not a runtime zero.
func TestCompilePython_DecimalCtorInvalidLiteral(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        self.r = Decimal("abc")
        return
`
	_, err := CompilePython(source)
	if err == nil {
		t.Fatal(`Decimal("abc"): CompilePython err = nil, want compile error`)
	}
	if !strings.Contains(err.Error(), "Decimal") {
		t.Fatalf("err = %v, want it to mention 'Decimal'", err)
	}
}

// TestCompilePython_DecimalCtorNone — Decimal(None) is a Python TypeError;
// fail closed at compile time.
func TestCompilePython_DecimalCtorNone(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        self.r = Decimal(None)
        return
`
	if _, err := CompilePython(source); err == nil {
		t.Fatal("Decimal(None): CompilePython err = nil, want compile error")
	}
}

// TestCompilePython_DecimalCtorNonLiteral — a non-literal argument compiles
// to the registered StringToDouble builtin (runtime conversion).
func TestCompilePython_DecimalCtorNonLiteral(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        x = "2.5"
        self.r = Decimal(x)
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
	if !v.ToDecimal().Equal(decimal.NewFromFloat(2.5)) {
		t.Fatalf("r = %s, want 2.5 (StringToDouble runtime conversion)", v.ToDecimal())
	}
}

// TestCompilePython_DecimalCtorZeroArg — Decimal() stays Decimal(0)
// (pre-existing correct semantics, regression guard).
func TestCompilePython_DecimalCtorZeroArg(t *testing.T) {
	v := runDecimalSource(t, `Decimal()`)
	if v.Kind != interp.ValDecimal {
		t.Fatalf("r kind = %v, want ValDecimal", v.Kind)
	}
	if !v.ToDecimal().IsZero() {
		t.Fatalf("r = %s, want 0 (Decimal() == Decimal('0') in Python)", v.ToDecimal())
	}
}
