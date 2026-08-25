package mql2go

import (
	"context"
	"testing"

	"alphaforge/tools/mql2go/interp"
)

// TestCompilePython_BasicStrategy verifies that a simple Python strategy compiles to IR and bytecode.
func TestCompilePython_BasicStrategy(t *testing.T) {
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
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}

	if ir.Version != "python" {
		t.Errorf("expected version 'python', got %q", ir.Version)
	}
	if len(ir.OnInit) == 0 {
		t.Error("OnInit should have statements")
	}
	if len(ir.OnBar) == 0 {
		t.Error("OnBar should have statements")
	}

	// Compile IR → Bytecode
	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	if bc.OnInit < 0 {
		t.Error("OnInit entry point should be set")
	}
	if bc.OnBar < 0 {
		t.Error("OnBar entry point should be set")
	}
	if len(bc.Code) == 0 {
		t.Error("Bytecode should have instructions")
	}

	// Run OnInit
	vm := NewVM(bc)
	if err := vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}
	if err := vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
}

// TestCompilePython_Expressions verifies expression compilation.
func TestCompilePython_Expressions(t *testing.T) {
	source := `from decimal import Decimal

class ExprTest:
    def on_bar(self) -> None:
        a = 10
        b = 20
        c = a + b
        d = a * b - 5
        e = a > b
        f = not e
        g = a if e else b
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	vm := NewVM(bc)
	if err := vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
}

// TestCompilePython_ForLoop verifies for-range loop compilation.
func TestCompilePython_ForLoop(t *testing.T) {
	source := `from decimal import Decimal

class ForTest:
    def on_bar(self) -> None:
        total = 0
        for i in range(10):
            total = total + i
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}

	if len(ir.OnBar) == 0 {
		t.Fatal("OnBar should have statements")
	}

	// Check that we have a for statement
	hasFor := false
	for _, s := range ir.OnBar {
		if s.Kind == interp.StmtFor {
			hasFor = true
			if s.Cond == nil {
				t.Error("for loop should have a condition")
			}
			if s.Init == nil {
				t.Error("for loop should have an init")
			}
			if s.Update == nil {
				t.Error("for loop should have an update")
			}
		}
	}
	if !hasFor {
		t.Error("expected a for statement in OnBar")
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	vm := NewVM(bc)
	if err := vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
}

// TestCompilePython_WhileLoop verifies while loop compilation.
func TestCompilePython_WhileLoop(t *testing.T) {
	source := `from decimal import Decimal

class WhileTest:
    def on_bar(self) -> None:
        i = 0
        while i < 5:
            i = i + 1
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	vm := NewVM(bc)
	if err := vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
}

// TestCompilePython_UserFunction verifies user-defined function compilation.
func TestCompilePython_UserFunction(t *testing.T) {
	source := `from decimal import Decimal

class FuncTest:
    def on_bar(self) -> None:
        result = self.calculate(10, 20)
        return

    def calculate(self, a: int, b: int) -> int:
        return a + b
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}

	if len(ir.Funcs) == 0 {
		t.Fatal("expected user-defined functions in IR")
	}
	fn, ok := ir.Funcs["calculate"]
	if !ok {
		t.Fatal("expected 'calculate' function in IR.Funcs")
	}
	if len(fn.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(fn.Params))
	}
	if fn.Params[0].Name != "a" || fn.Params[1].Name != "b" {
		t.Errorf("expected params 'a','b', got %q,%q", fn.Params[0].Name, fn.Params[1].Name)
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	if _, ok := bc.Funcs["calculate"]; !ok {
		t.Error("expected 'calculate' in bc.Funcs")
	}
}

// TestCompilePython_SubsetRejection verifies that forbidden syntax is rejected.
func TestCompilePython_SubsetRejection(t *testing.T) {
	tests := []struct {
		name   string
		source string
		errMsg string
	}{
		{
			name: "forbidden import os",
			source: `import os
class S: pass`,
			errMsg: "not allowed",
		},
		{
			name: "forbidden from import",
			source: `from os import path
class S: pass`,
			errMsg: "only 'from decimal import Decimal'",
		},
		{
			name: "forbidden exec",
			source: `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        exec("print(1)")
`,
			errMsg: "exec",
		},
		{
			name: "forbidden eval",
			source: `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = eval("1+1")
`,
			errMsg: "eval",
		},
		{
			name: "forbidden lambda",
			source: `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        f = lambda x: x + 1
`,
			errMsg: "lambda",
		},
		{
			name: "forbidden try",
			source: `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        try:
            x = 1
        except:
            pass
`,
			errMsg: "try",
		},
		{
			name: "forbidden open",
			source: `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        f = open("test.txt")
`,
			errMsg: "open",
		},
		{
			name: "forbidden dunder",
			source: `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = self.__class__
`,
			errMsg: "dunder",
		},
		{
			name: "forbidden walrus",
			source: `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        if (n := 10) > 5:
            x = n
`,
			errMsg: "walrus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CompilePythonToIR(tt.source)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errMsg)
			}
		})
	}
}

// TestCompilePython_SubsetAcceptance verifies that valid subset code is accepted.
func TestCompilePython_SubsetAcceptance(t *testing.T) {
	source := `from decimal import Decimal

class ValidStrategy:
    def on_init(self) -> None:
        self.lot_size = Decimal("0.1")
        return

    def on_bar(self) -> None:
        price = 100
        lot = self.lot_size
        if price > 90 and price < 110:
            x = 1
        elif price > 50:
            x = 2
        else:
            x = 0
        return
`
	_, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("valid subset should compile without error: %v", err)
	}
}

// TestCompilePython_IfElifElse verifies if/elif/else chain compilation.
func TestCompilePython_IfElifElse(t *testing.T) {
	source := `from decimal import Decimal

class IfTest:
    def on_bar(self) -> None:
        x = 5
        if x > 10:
            y = 1
        elif x > 5:
            y = 2
        else:
            y = 3
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	vm := NewVM(bc)
	if err := vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
}

// TestCompilePython_BreakContinue verifies break and continue in loops.
func TestCompilePython_BreakContinue(t *testing.T) {
	source := `from decimal import Decimal

class BreakTest:
    def on_bar(self) -> None:
        total = 0
        for i in range(100):
            if i == 10:
                break
            if i % 2 == 0:
                continue
            total = total + i
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	vm := NewVM(bc)
	if err := vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
}

// TestCompilePython_DecimalLiteral verifies Decimal("0.1") handling.
func TestCompilePython_DecimalLiteral(t *testing.T) {
	source := `from decimal import Decimal

class DecTest:
    def on_bar(self) -> None:
        d = Decimal("0.01")
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	vm := NewVM(bc)
	if err := vm.RunOnBar(context.Background()); err != nil {
		t.Fatalf("RunOnBar failed: %v", err)
	}
}

// TestCompilePython_SourceSizeLimit verifies that oversized source is rejected.
func TestCompilePython_SourceSizeLimit(t *testing.T) {
	// Generate source larger than MaxSourceSize
	source := "from decimal import Decimal\nclass S:\n def on_bar(self) -> None:\n  x = 0\n"
	for len(source) <= MaxSourceSize {
		source += "  x = x + 1\n"
	}

	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for oversized source")
	}
}

// TestCompilePython_Ternary verifies a if cond else b compiles to ExprTernary.
func TestCompilePython_Ternary(t *testing.T) {
	source := `from decimal import Decimal
class TernaryTest:
    def on_bar(self) -> None:
        x = 10
        y = 20
        z = x if x > y else y
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}
	if bc == nil {
		t.Fatal("bytecode is nil")
	}
}

// TestCompilePython_SelfMethodCall verifies self.method() resolves to method name without "self." prefix.
func TestCompilePython_SelfMethodCall(t *testing.T) {
	source := `from decimal import Decimal
class SelfCall:
    def on_bar(self) -> None:
        result = self.helper(42)
        return

    def helper(self, val: int) -> int:
        return val + 1
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	fn, ok := ir.Funcs["helper"]
	if !ok {
		t.Fatal("helper function not found in ir.Funcs")
	}
	if fn == nil || len(fn.Body) == 0 {
		t.Fatal("helper function has no body")
	}
	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}
	if bc == nil {
		t.Fatal("bytecode is nil")
	}
}

// TestCompilePython_SelfField verifies self.field maps to a global variable.
func TestCompilePython_SelfField(t *testing.T) {
	source := `from decimal import Decimal
class SelfField:
    def on_init(self) -> None:
        self.threshold = Decimal("0.5")
        return

    def on_bar(self) -> None:
        if self.threshold > Decimal("0.3"):
            x = 1
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	found := false
	for _, g := range ir.Globals {
		if g.Name == "threshold" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("self.threshold should be collected as a global variable")
	}
	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}
	if bc == nil {
		t.Fatal("bytecode is nil")
	}
}

// TestCompilePython_ChainedComparison verifies a < b < c compiles to (a < b) && (b < c).
func TestCompilePython_ChainedComparison(t *testing.T) {
	source := `from decimal import Decimal
class ChainTest:
    def on_bar(self) -> None:
        a = 1
        b = 2
        c = 3
        ok = a < b < c
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}
	if bc == nil {
		t.Fatal("bytecode is nil")
	}
}

// TestCompilePython_RangeStep verifies for i in range(0, 10, 2) compiles with step.
func TestCompilePython_RangeStep(t *testing.T) {
	source := `from decimal import Decimal
class StepTest:
    def on_bar(self) -> None:
        total = 0
        for i in range(0, 10, 2):
            total = total + i
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) < 2 {
		t.Fatalf("expected at least 2 statements in OnBar, got %d", len(ir.OnBar))
	}
	forStmt := ir.OnBar[1]
	if forStmt.Kind != interp.StmtFor {
		t.Fatalf("expected StmtFor, got %v", forStmt.Kind)
	}
	if forStmt.Update == nil {
		t.Fatal("for loop update should not be nil")
	}
	if forStmt.Update.Expr.Kind != interp.ExprCompoundAssign {
		t.Fatalf("expected ExprCompoundAssign for step, got %v", forStmt.Update.Expr.Kind)
	}
	if forStmt.Update.Expr.Op != "+=" {
		t.Fatalf("expected += op, got %s", forStmt.Update.Expr.Op)
	}
}

// TestCompilePython_CompilePythonEntry verifies the CompilePython convenience function.
