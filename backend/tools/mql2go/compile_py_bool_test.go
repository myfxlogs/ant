package mql2go

import (
	"context"
	"testing"

	"alphaforge/tools/mql2go/interp"
)

// QS-1.4: Python bool(x) must follow Python truthiness (Value.IsTrue), not x != 0.
// Under the old compile form `x != 0`, Value.Equal returns false for mixed
// None/Int so bool(None) evaluated to true — the opposite of Python semantics.
func TestCompilePython_BoolSemantics(t *testing.T) {
	source := `from decimal import Decimal

class BoolSem:
    def on_bar(self) -> None:
        self.r_none = bool(None)
        self.r_zero = bool(0)
        self.r_dec = bool(0.0)
        self.r_empty = bool("")
        self.r_false = bool(False)
        self.r_one = bool(1)
        self.r_str = bool("a")
        self.r_dec_pos = bool(0.5)
        self.r_noarg = bool()
        return
`
	vmRunner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if err := vmRunner.vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}

	cases := []struct {
		name string
		want bool
	}{
		{"r_none", false},
		{"r_zero", false},
		{"r_dec", false},
		{"r_empty", false},
		{"r_false", false},
		{"r_one", true},
		{"r_str", true},
		{"r_dec_pos", true},
		{"r_noarg", false},
	}
	for _, tc := range cases {
		v, ok := vmRunner.GetGlobal(tc.name)
		if !ok {
			t.Fatalf("global %q not found", tc.name)
		}
		if v.Kind != interp.ValBool {
			t.Errorf("%s: Kind = %v, want ValBool (bool() must produce a real bool)", tc.name, v.Kind)
			continue
		}
		if v.Bool != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, v.Bool, tc.want)
		}
	}
}

// QS-1.4: bool(x) compiles to double logical NOT (ExprUnary "!" wrapping
// ExprUnary "!"), which lowers to two OP_NOT instructions reusing IsTrue().
func TestCompilePython_BoolCompilesToDoubleNot(t *testing.T) {
	source := `class S:
    def on_bar(self) -> None:
        self.b = bool(None)
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) == 0 || ir.OnBar[0].Expr == nil {
		t.Fatal("expected assignment statement in OnBar")
	}
	assign := ir.OnBar[0].Expr
	if assign.Kind != interp.ExprAssignment || len(assign.Args) != 1 {
		t.Fatalf("expected ExprAssignment with one arg, got kind=%v", assign.Kind)
	}
	outer := assign.Args[0]
	if outer.Kind != interp.ExprUnary || outer.Op != "!" || len(outer.Args) != 1 {
		t.Fatalf("expected outer ExprUnary '!' for bool(), got kind=%v op=%q", outer.Kind, outer.Op)
	}
	inner := outer.Args[0]
	if inner.Kind != interp.ExprUnary || inner.Op != "!" || len(inner.Args) != 1 {
		t.Fatalf("expected inner ExprUnary '!' for bool(), got kind=%v op=%q", inner.Kind, inner.Op)
	}
}
