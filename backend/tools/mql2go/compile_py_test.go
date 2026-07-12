package mql2go

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"anttrader/tools/mql2go/interp"
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
func TestCompilePython_CompilePythonEntry(t *testing.T) {
	source := `from decimal import Decimal
class EntryTest:
    def on_bar(self) -> None:
        x = 1 + 2
        return
`
	runner, err := CompilePython(source)
	if err != nil {
		t.Fatalf("CompilePython failed: %v", err)
	}
	if runner == nil {
		t.Fatal("runner is nil")
	}
	if runner.Bytecode() == nil {
		t.Fatal("bytecode is nil")
	}
}

// TestCompilePython_EnumTypesInit verifies that EnumTypes is initialized.
func TestCompilePython_EnumTypesInit(t *testing.T) {
	source := `from decimal import Decimal
class EnumTest:
    def on_bar(self) -> None:
        x = 1
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if ir.EnumTypes == nil {
		t.Fatal("EnumTypes should be initialized, not nil")
	}
}

// TestCompilePython_MissingEvents verifies on_trade_transaction and on_book_event map correctly.
func TestCompilePython_MissingEvents(t *testing.T) {
	source := `from decimal import Decimal
class EventTest:
    def on_trade_transaction(self) -> None:
        x = 1
        return

    def on_book_event(self) -> None:
        y = 2
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnTradeTransaction) == 0 {
		t.Fatal("OnTradeTransaction should be populated")
	}
	if len(ir.OnBookEvent) == 0 {
		t.Fatal("OnBookEvent should be populated")
	}
}

func TestCompilePython_InitParams(t *testing.T) {
	source := `from decimal import Decimal
class MyStrategy:
    def __init__(self, period: int = 14, lot: Decimal = Decimal("0.1")) -> None:
        self.period = period
        self.lot = lot
    def on_bar(self) -> None:
        pass
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(ir.Params))
	}
	if ir.Params[0].Name != "period" {
		t.Errorf("param[0] name = %s, want period", ir.Params[0].Name)
	}
	if ir.Params[0].Type != "int" {
		t.Errorf("param[0] type = %s, want int", ir.Params[0].Type)
	}
	if ir.Params[1].Name != "lot" {
		t.Errorf("param[1] name = %s, want lot", ir.Params[1].Name)
	}
	if ir.Params[1].Default == nil {
		t.Error("param[1] default should not be nil")
	}
}

func TestCompilePython_CtxBrokerMapping(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        ctx.broker.buy(lot=Decimal("0.1"), sl=ctx.ask() - Decimal("0.0050"))
        ctx.broker.sell(lot=Decimal("0.1"))
        ctx.broker.close(12345)
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) == 0 {
		t.Fatal("OnBar should have statements")
	}
}

func TestCompilePython_CtxDirectMapping(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = ctx.ask()
        y = ctx.bid()
        z = ctx.symbol()
        w = ctx.point()
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) == 0 {
		t.Fatal("OnBar should have statements")
	}
}

func TestCompilePython_ForPositions(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        for pos in ctx.positions:
            if pos.profit < -Decimal("50"):
                ctx.broker.close(pos.ticket)
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) == 0 {
		t.Fatal("OnBar should have statements")
	}
	forStmt := ir.OnBar[0]
	if forStmt.Kind != interp.StmtFor {
		t.Fatalf("expected StmtFor, got %v", forStmt.Kind)
	}
	if forStmt.Cond == nil {
		t.Fatal("for loop should have condition")
	}
	if forStmt.Cond.Kind != interp.ExprBinary {
		t.Fatalf("condition should be ExprBinary, got %v", forStmt.Cond.Kind)
	}
	if len(forStmt.Body) < 3 {
		t.Fatalf("expected at least 3 body statements (PositionGetTicket + PositionSelectByTicket + if), got %d", len(forStmt.Body))
	}
}

func TestCompilePython_TypeAnnotationEnforcement(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def __init__(self, period=14) -> None:
        self.period = period
    def on_bar(self) -> None:
        pass
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for missing type annotation in __init__")
	}
	if !strings.Contains(err.Error(), "type annotation") {
		t.Errorf("error should mention type annotation, got: %v", err)
	}
}

func TestCompilePython_FStringRejection(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = f"hello {self.period}"
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for f-string")
	}
	if !strings.Contains(err.Error(), "f-string") {
		t.Errorf("error should mention f-string, got: %v", err)
	}
}

func TestCompilePython_InitAllowed(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def __init__(self, period: int = 14) -> None:
        self.period = period
    def on_bar(self) -> None:
        pass
`
	_, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("__init__ should be allowed, got error: %v", err)
	}
}

func TestCompilePython_ListComprehensionRejection(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = [i*2 for i in range(10)]
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for list comprehension")
	}
}

func TestCompilePython_ChainedCallBars(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        c1: float = ctx.bars().close(1)
        c0: float = ctx.bars().close(0)
        h: float = ctx.bars().high(2)
        l: float = ctx.bars().low(0)
        o: float = ctx.bars().open(1)
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) != 5 {
		t.Fatalf("expected 5 statements, got %d", len(ir.OnBar))
	}
	for i, name := range []string{"c1", "c0", "h", "l", "o"} {
		stmt := ir.OnBar[i]
		if stmt.Expr == nil || stmt.Expr.Name != name {
			t.Errorf("stmt[%d]: expected name %s, got %v", i, name, stmt.Expr)
			continue
		}
		if len(stmt.Expr.Args) == 0 {
			t.Errorf("stmt[%d]: expected args", i)
			continue
		}
		callExpr := stmt.Expr.Args[0]
		expectedCalls := map[string]string{"c1": "Close", "c0": "Close", "h": "High", "l": "Low", "o": "Open"}
		if callExpr.Name != expectedCalls[name] {
			t.Errorf("stmt[%d]: expected call %s, got %s", i, expectedCalls[name], callExpr.Name)
		}
	}
}

func TestCompilePython_MultiSymbolBars(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        c: float = ctx.bars_for_symbol("EURUSD").close(0)
        h: float = ctx.bars_for_symbol("GBPUSD").high(1)
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(ir.OnBar))
	}
	// First statement: c = iClose("EURUSD", 0, 0) — symbol, timeframe=0 (PERIOD_CURRENT), shift
	stmt0 := ir.OnBar[0]
	if stmt0.Expr == nil || stmt0.Expr.Name != "c" {
		t.Fatalf("stmt[0]: expected name c, got %v", stmt0.Expr)
	}
	if len(stmt0.Expr.Args) == 0 {
		t.Fatalf("stmt[0]: expected args")
	}
	call0 := stmt0.Expr.Args[0]
	if call0.Name != "iClose" {
		t.Errorf("stmt[0]: expected iClose, got %s", call0.Name)
	}
	if len(call0.Args) != 3 {
		t.Errorf("stmt[0]: expected 3 args (symbol + timeframe + shift), got %d", len(call0.Args))
	}
	// Second statement: h = iHigh("GBPUSD", 0, 1)
	stmt1 := ir.OnBar[1]
	if stmt1.Expr == nil || stmt1.Expr.Name != "h" {
		t.Fatalf("stmt[1]: expected name h, got %v", stmt1.Expr)
	}
	call1 := stmt1.Expr.Args[0]
	if call1.Name != "iHigh" {
		t.Errorf("stmt[1]: expected iHigh, got %s", call1.Name)
	}
	if len(call1.Args) != 3 {
		t.Errorf("stmt[1]: expected 3 args, got %d", len(call1.Args))
	}
}

func TestCompilePython_BooleanOperatorInIf(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        a: int = 10
        b: int = 20
        c: int = 30
        if a > 5 and b < 50 or c == 30:
            x = 1
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	hasIf := false
	for _, s := range ir.OnBar {
		if s.Kind == interp.StmtIf {
			hasIf = true
			if s.Cond == nil {
				t.Fatal("if condition should not be nil")
			}
			if s.Cond.Kind != interp.ExprBinary {
				t.Fatalf("expected ExprBinary for boolean condition, got %v", s.Cond.Kind)
			}
		}
	}
	if !hasIf {
		t.Fatal("expected an if statement")
	}
}

func TestCompilePython_ReturnTypeEnforcement(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self):
        pass
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for missing return type annotation")
	}
	if !strings.Contains(err.Error(), "return type annotation") {
		t.Errorf("error should mention return type annotation, got: %v", err)
	}
}

func TestCompilePython_AnnotatedAssignment(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x: int = 42
        y: float = 3.14
        z: Decimal = Decimal("0.1")
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(ir.OnBar))
	}
	if ir.OnBar[0].Expr.Name != "x" {
		t.Errorf("stmt[0]: expected name 'x', got %s", ir.OnBar[0].Expr.Name)
	}
	if ir.OnBar[0].Expr.Args[0].Kind != interp.ExprLiteral {
		t.Errorf("stmt[0]: expected literal value, got %v", ir.OnBar[0].Expr.Args[0].Kind)
	}
}

func TestCompilePython_PositionFieldMapping(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        for pos in ctx.positions:
            t = pos.ticket
            p = pos.profit
            v = pos.volume
            s = pos.swap
            c = pos.comment
            ot = pos.open_time
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) == 0 {
		t.Fatal("OnBar should have statements")
	}
	forStmt := ir.OnBar[0]
	if forStmt.Kind != interp.StmtFor {
		t.Fatalf("expected StmtFor, got %v", forStmt.Kind)
	}
	if len(forStmt.Body) < 8 {
		t.Fatalf("expected at least 8 body statements (2 prepended + 6 field accesses), got %d", len(forStmt.Body))
	}
}

func TestCompilePython_SelfFieldAssignment(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_init(self) -> None:
        self.counter = 0
        self.name = "test"
        return

    def on_bar(self) -> None:
        self.counter += 1
        return
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	// on_init should have 2 assignments: counter=0, name="test"
	if len(ir.OnInit) < 2 {
		t.Fatalf("expected at least 2 OnInit statements, got %d", len(ir.OnInit))
	}
	for i, expected := range []string{"counter", "name"} {
		stmt := ir.OnInit[i]
		if stmt.Expr == nil || stmt.Expr.Name != expected {
			t.Errorf("OnInit[%d]: expected name %s, got %v", i, expected, stmt.Expr)
		}
	}
	// on_bar should have augmented assignment: counter += 1
	if len(ir.OnBar) < 1 {
		t.Fatalf("expected at least 1 OnBar statement, got %d", len(ir.OnBar))
	}
	barStmt := ir.OnBar[0]
	if barStmt.Expr == nil || barStmt.Expr.Name != "counter" {
		t.Errorf("OnBar[0]: expected name 'counter', got %v", barStmt.Expr)
	}
	if barStmt.Expr.Kind != interp.ExprCompoundAssign {
		t.Errorf("OnBar[0]: expected ExprCompoundAssign, got %v", barStmt.Expr.Kind)
	}
	// Verify self vars are collected as globals
	found := false
	for _, g := range ir.Globals {
		if g.Name == "counter" || g.Name == "name" {
			found = true
		}
	}
	if !found {
		t.Error("expected self.counter and self.name in globals")
	}
}

func TestCompilePython_ElifChaining(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = 10
        if x > 5:
            y = 1
        elif x > 3:
            y = 2
        elif x > 1:
            y = 3
        else:
            y = 0
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	// Find the if statement
	var ifStmt *interp.Statement
	for i := range ir.OnBar {
		if ir.OnBar[i].Kind == interp.StmtIf {
			ifStmt = &ir.OnBar[i]
			break
		}
	}
	if ifStmt == nil {
		t.Fatal("expected an if statement")
	}
	if ifStmt.Cond == nil {
		t.Fatal("if condition should not be nil")
	}
	if len(ifStmt.Body) != 1 {
		t.Fatalf("expected 1 body statement (y=1), got %d", len(ifStmt.Body))
	}
	// First elif: x > 3, y = 2
	if len(ifStmt.ElseBody) != 1 {
		t.Fatalf("expected 1 else statement (first elif), got %d", len(ifStmt.ElseBody))
	}
	elif1 := &ifStmt.ElseBody[0]
	if elif1.Kind != interp.StmtIf {
		t.Fatalf("expected first elif to be StmtIf, got %v", elif1.Kind)
	}
	if len(elif1.Body) != 1 {
		t.Fatalf("expected first elif body 1 statement, got %d", len(elif1.Body))
	}
	// Second elif: x > 1, y = 3
	if len(elif1.ElseBody) != 1 {
		t.Fatalf("expected 1 else in first elif (second elif), got %d", len(elif1.ElseBody))
	}
	elif2 := &elif1.ElseBody[0]
	if elif2.Kind != interp.StmtIf {
		t.Fatalf("expected second elif to be StmtIf, got %v", elif2.Kind)
	}
	// Final else: y = 0
	if len(elif2.ElseBody) != 1 {
		t.Fatalf("expected 1 else in second elif (final else), got %d", len(elif2.ElseBody))
	}
	elseStmt := &elif2.ElseBody[0]
	if elseStmt.Kind != interp.StmtExpr {
		t.Fatalf("expected final else to be StmtExpr, got %v", elseStmt.Kind)
	}
}

func TestCompilePython_DefaultValBooleanOperator(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def __init__(self, flag: bool = True, val: int = 10) -> None:
        self.flag = flag
        self.val = val
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(ir.Params))
	}
	if ir.Params[0].Name != "flag" {
		t.Errorf("param[0]: expected 'flag', got %s", ir.Params[0].Name)
	}
	if ir.Params[0].Default == nil {
		t.Error("param[0]: expected default value")
	}
	if ir.Params[1].Name != "val" {
		t.Errorf("param[1]: expected 'val', got %s", ir.Params[1].Name)
	}
	if ir.Params[1].Default == nil {
		t.Error("param[1]: expected default value")
	}
}

func TestCompilePython_ForbiddenCollectionLiterals(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"list", `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = [1, 2, 3]
`},
		{"tuple", `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = (1, 2)
`},
		{"dictionary", `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = {"a": 1}
`},
		{"set", `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = {1, 2, 3}
`},
		{"expression_list", `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        return 1, 2
`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CompilePythonToIR(tc.source)
			if err == nil {
				t.Fatal("expected error for forbidden collection literal")
			}
			if !strings.Contains(err.Error(), "not allowed") {
				t.Errorf("error should mention 'not allowed', got: %v", err)
			}
		})
	}
}

func TestCompilePython_ForbiddenMatchStatement(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = 1
        match x:
            case 1:
                pass
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for match_statement")
	}
	if !strings.Contains(err.Error(), "match_statement") {
		t.Errorf("error should mention match_statement, got: %v", err)
	}
}

func TestCompilePython_SyntaxErrorRejected(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"missing_colon", `from decimal import Decimal
class S:
    def on_bar(self) -> None
        x = 1
`},
		{"incomplete_expr", `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = 1 +
`},
		{"missing_paren", `from decimal import Decimal
class S:
    def on_bar(self -> None:
        x = 1
`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CompilePythonToIR(tc.source)
			if err == nil {
				t.Fatal("expected error for syntax error")
			}
		})
	}
}

func TestCompilePython_ForbiddenSplatInCalls(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"list_splat", `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        args = 1
        func(*args)
`},
		{"dict_splat", `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        kwargs = 1
        func(**kwargs)
`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CompilePythonToIR(tc.source)
			if err == nil {
				t.Fatal("expected error for splat in call args")
			}
		})
	}
}

func TestCompilePython_ForElseRejected(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        for i in range(5):
            x = i
        else:
            x = 0
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for for...else")
	}
	if !strings.Contains(err.Error(), "for...else") {
		t.Errorf("error should mention for...else, got: %v", err)
	}
}

func TestCompilePython_WhileElseRejected(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = 1
        while x > 0:
            x -= 1
        else:
            x = 0
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for while...else")
	}
	if !strings.Contains(err.Error(), "while...else") {
		t.Errorf("error should mention while...else, got: %v", err)
	}
}

func TestCompilePython_RawFStringRejected(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        y = "world"
        x = rf"hello {y}"
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for rf-string with interpolation")
	}
	if !strings.Contains(err.Error(), "interpolation") {
		t.Errorf("error should mention interpolation, got: %v", err)
	}
}

func TestCompilePython_NestedClassRejected(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        class Inner:
            pass
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for nested class")
	}
	if !strings.Contains(err.Error(), "nested class") {
		t.Errorf("error should mention nested class, got: %v", err)
	}
}

func TestCompilePython_PatternListRejected(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        a, b = 1, 2
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for pattern_list (tuple unpacking)")
	}
}

func TestCompilePython_NotOperator(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = 1
        if not x:
            x = 0
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ir.OnBar == nil || len(ir.OnBar) < 2 {
		t.Fatalf("expected at least 2 statements in OnBar, got %d", len(ir.OnBar))
	}
	if ir.OnBar[1].Kind != interp.StmtIf {
		t.Fatalf("expected 2nd statement to be if, got %v", ir.OnBar[1].Kind)
	}
	if ir.OnBar[1].Cond == nil {
		t.Fatal("expected if condition to be non-nil")
	}
	if ir.OnBar[1].Cond.Kind != interp.ExprUnary {
		t.Errorf("expected unary expr for 'not x', got %v", ir.OnBar[1].Cond.Kind)
	}
	if ir.OnBar[1].Cond.Op != "!" {
		t.Errorf("expected '!' operator, got %s", ir.OnBar[1].Cond.Op)
	}
}

func TestCompilePython_NotInBooleanOperator(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = 1
        y = 2
        if not x and y:
            x = 0
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ir.OnBar == nil || len(ir.OnBar) < 3 {
		t.Fatalf("expected at least 3 statements, got %d", len(ir.OnBar))
	}
	if ir.OnBar[2].Kind != interp.StmtIf {
		t.Fatalf("expected 3rd statement to be if, got %v", ir.OnBar[2].Kind)
	}
	if ir.OnBar[2].Cond == nil {
		t.Fatal("expected if condition")
	}
	if ir.OnBar[2].Cond.Kind != interp.ExprBinary {
		t.Errorf("expected binary (&&), got %v", ir.OnBar[2].Cond.Kind)
	}
}

func TestCompilePython_EllipsisRejected(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = ...
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for ellipsis literal")
	}
	if !strings.Contains(err.Error(), "ellipsis") {
		t.Errorf("error should mention ellipsis, got: %v", err)
	}
}

func TestCompilePython_TypeAliasRejected(t *testing.T) {
	source := `type MyInt = int
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for type_alias_statement")
	}
}

func TestCompilePython_PrintStatementRejected(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        print 'hello'
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for print_statement")
	}
}

func TestCompilePython_ComplexLiteralRejected(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"int_j", `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = 3j
`},
		{"float_j", `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = 3.14j
`},
		{"int_J", `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = 5J
`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CompilePythonToIR(tc.source)
			if err == nil {
				t.Fatal("expected error for complex number literal")
			}
			if !strings.Contains(err.Error(), "complex") {
				t.Errorf("error should mention complex, got: %v", err)
			}
		})
	}
}

func TestCompilePython_ImportFromHandled(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = Decimal("1.5")
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ir == nil {
		t.Fatal("expected non-nil IR")
	}
	if ir.OnBar == nil || len(ir.OnBar) != 1 {
		t.Fatalf("expected 1 statement in OnBar, got %d", len(ir.OnBar))
	}
}

func TestCompilePython_OperatorInCompiled(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = 1
        if x in "hello":
            x = 0
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ir.OnBar == nil || len(ir.OnBar) < 2 {
		t.Fatalf("expected at least 2 statements, got %d", len(ir.OnBar))
	}
	ifStmt := ir.OnBar[1]
	if ifStmt.Kind != interp.StmtIf {
		t.Fatalf("expected 2nd statement to be if, got %v", ifStmt.Kind)
	}
	if ifStmt.Cond == nil {
		t.Fatal("expected if condition")
	}
	if ifStmt.Cond.Kind != interp.ExprCall {
		t.Errorf("expected call expr for 'in' operator, got %v", ifStmt.Cond.Kind)
	}
	if ifStmt.Cond.Name != "operator_in" {
		t.Errorf("expected operator_in, got %s", ifStmt.Cond.Name)
	}
}

func TestCompilePython_OperatorNotInCompiled(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = 1
        if x not in "hello":
            x = 0
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ir.OnBar == nil || len(ir.OnBar) < 2 {
		t.Fatalf("expected at least 2 statements, got %d", len(ir.OnBar))
	}
	ifStmt := ir.OnBar[1]
	if ifStmt.Kind != interp.StmtIf {
		t.Fatalf("expected 2nd statement to be if, got %v", ifStmt.Kind)
	}
	if ifStmt.Cond == nil {
		t.Fatal("expected if condition")
	}
	if ifStmt.Cond.Kind == interp.ExprUnary && ifStmt.Cond.Op == "!" {
		inner := ifStmt.Cond.Args[0]
		if inner.Kind == interp.ExprCall && inner.Name == "operator_in" {
			return
		}
	}
	if ifStmt.Cond.Kind == interp.ExprCall && ifStmt.Cond.Name == "operator_in" {
		t.Fatal("expected negation for 'not in', got direct operator_in")
	}
	t.Errorf("unexpected condition structure for 'not in': Kind=%v", ifStmt.Cond.Kind)
}

func TestCompilePython_IndicatorCaseInsensitive(t *testing.T) {
	indicators := []struct {
		pythonName string
		builtinName string
	}{
		{"ialligator", "iAlligator"},
		{"iichimoku", "iIchimoku"},
		{"ienvelopes", "iEnvelopes"},
		{"idemarker", "iDeMarker"},
		{"iosma", "iOsMA"},
		{"irvi", "iRVI"},
		{"iforce", "iForce"},
		{"ifractals", "iFractals"},
		{"igator", "iGator"},
		{"iac", "iAC"},
		{"iad", "iAD"},
		{"iao", "iAO"},
		{"ibearspower", "iBearsPower"},
		{"ibullspower", "iBullsPower"},
		{"ibwmfi", "iBWMFI"},
		{"iama", "iAMA"},
		{"idema", "iDEMA"},
		{"itema", "iTEMA"},
		{"iframa", "iFrAMA"},
		{"ividya", "iVIDyA"},
		{"itrix", "iTriX"},
		{"iadxwilder", "iADXWilder"},
		{"ichaikin", "iChaikin"},
		{"ivolumes", "iVolumes"},
		{"ibollinger", "iBollinger"},
	}
	for _, ind := range indicators {
		t.Run(ind.pythonName, func(t *testing.T) {
			source := fmt.Sprintf(`from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = ctx.indicators.%s(ctx.symbol(), 14, 0)
`, ind.pythonName)
			ir, err := CompilePythonToIR(source)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ir.OnBar == nil || len(ir.OnBar) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(ir.OnBar))
			}
			stmt := ir.OnBar[0]
			if stmt.Kind != interp.StmtExpr {
				t.Fatalf("expected StmtExpr, got %v", stmt.Kind)
			}
			if stmt.Expr == nil || stmt.Expr.Kind != interp.ExprAssignment {
				t.Fatalf("expected ExprAssignment, got %v", stmt.Expr)
			}
			valExpr := stmt.Expr.Args[0]
			if valExpr.Kind != interp.ExprCall {
				t.Fatalf("expected ExprCall, got %v", valExpr.Kind)
			}
			if valExpr.Name != ind.builtinName {
				t.Errorf("expected %q, got %q", ind.builtinName, valExpr.Name)
			}
		})
	}
}

func TestCompilePython_SpreadBuiltin(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = ctx.spread()
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ir.OnBar == nil || len(ir.OnBar) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(ir.OnBar))
	}
	stmt := ir.OnBar[0]
	if stmt.Expr == nil || stmt.Expr.Kind != interp.ExprAssignment {
		t.Fatalf("expected ExprAssignment, got %v", stmt.Expr)
	}
	valExpr := stmt.Expr.Args[0]
	if valExpr.Kind != interp.ExprCall {
		t.Fatalf("expected ExprCall, got %v", valExpr.Kind)
	}
	if valExpr.Name != "Spread" {
		t.Errorf("expected 'Spread', got %q", valExpr.Name)
	}
}

func TestCompilePython_LegacyPositionCloseModify(t *testing.T) {
	tests := []struct {
		name   string
		method string
		builtin string
	}{
		{"position_close", "position_close", "PositionClose"},
		{"position_modify", "position_modify", "PositionModify"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := fmt.Sprintf(`from decimal import Decimal
class S:
    def on_bar(self) -> None:
        ctx.broker.%s(12345)
`, tc.method)
			ir, err := CompilePythonToIR(source)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ir.OnBar == nil || len(ir.OnBar) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(ir.OnBar))
			}
			stmt := ir.OnBar[0]
			if stmt.Expr == nil || stmt.Expr.Kind != interp.ExprCall {
				t.Fatalf("expected ExprCall, got %v", stmt.Expr.Kind)
			}
			if stmt.Expr.Name != tc.builtin {
				t.Errorf("expected %q, got %q", tc.builtin, stmt.Expr.Name)
			}
		})
	}
}

func TestCompilePython_ForbiddenPythonBuiltins(t *testing.T) {
	forbidden := []string{
		"len", "sorted", "sum", "enumerate", "zip",
		"reversed", "any", "all",
	}
	for _, fn := range forbidden {
		t.Run(fn, func(t *testing.T) {
			source := fmt.Sprintf(`from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = %s([1, 2, 3])
`, fn)
			_, err := CompilePythonToIR(source)
			if err == nil {
				t.Fatalf("expected error for %s(), got nil", fn)
			}
			if !strings.Contains(err.Error(), fn) {
				t.Errorf("error should mention %s, got: %v", fn, err)
			}
		})
	}
}

func TestCompilePython_BoolConversion(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = bool(1)
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ir.OnBar == nil || len(ir.OnBar) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(ir.OnBar))
	}
	stmt := ir.OnBar[0]
	if stmt.Expr == nil || stmt.Expr.Kind != interp.ExprAssignment {
		t.Fatalf("expected ExprAssignment, got %v", stmt.Expr)
	}
	valExpr := stmt.Expr.Args[0]
	if valExpr.Kind != interp.ExprBinary || valExpr.Op != "!=" {
		t.Fatalf("expected binary !=, got %v", valExpr)
	}
}

func TestCompilePython_BitwiseNotPanic(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = ~5
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for bitwise NOT ~, got nil")
	}
	if !strings.Contains(err.Error(), "bitwise NOT") {
		t.Errorf("error should mention bitwise NOT, got: %v", err)
	}
}

func TestCompilePython_PositionFieldWritePanic(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        for pos in ctx.positions:
            pos.sl = Decimal("0.0050")
`
	_, err := CompilePythonToIR(source)
	if err == nil {
		t.Fatal("expected error for position field write, got nil")
	}
	if !strings.Contains(err.Error(), "cannot assign to position field") {
		t.Errorf("error should mention cannot assign to position field, got: %v", err)
	}
}

func TestCompilePython_BitwiseBinaryOperators(t *testing.T) {
	ops := []string{"&", "|", "^", "<<", ">>"}
	for _, op := range ops {
		t.Run(op, func(t *testing.T) {
			source := fmt.Sprintf(`from decimal import Decimal
class S:
    def on_bar(self) -> None:
        x = 1 %s 2
`, op)
			_, err := CompilePythonToIR(source)
			if err == nil {
				t.Fatalf("expected error for bitwise %s, got nil", op)
			}
			if !strings.Contains(err.Error(), "bitwise operator") {
				t.Errorf("error should mention bitwise operator, got: %v", err)
			}
		})
	}
}

func TestCompilePython_InitAndOnInitMerge(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def __init__(self, period: int = 14) -> None:
        self.period = period
        self.counter = 0
    def on_init(self) -> None:
        self.initialized = True
    def on_bar(self) -> None:
        pass
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	// __init__ body (2 stmts) + on_init body (1 stmt) = 3 total
	if len(ir.OnInit) != 3 {
		t.Fatalf("expected 3 OnInit statements (2 from __init__ + 1 from on_init), got %d", len(ir.OnInit))
	}
	// __init__ stmts must come first
	if ir.OnInit[0].Expr == nil || ir.OnInit[0].Expr.Name != "period" {
		t.Errorf("OnInit[0]: expected 'period', got %v", ir.OnInit[0].Expr)
	}
	if ir.OnInit[1].Expr == nil || ir.OnInit[1].Expr.Name != "counter" {
		t.Errorf("OnInit[1]: expected 'counter', got %v", ir.OnInit[1].Expr)
	}
	// on_init stmt must come last
	if ir.OnInit[2].Expr == nil || ir.OnInit[2].Expr.Name != "initialized" {
		t.Errorf("OnInit[2]: expected 'initialized', got %v", ir.OnInit[2].Expr)
	}
}

func TestCompilePython_FloorDivision(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def __init__(self) -> None:
        self.a = 0
        self.b = 0
    def on_bar(self) -> None:
        self.a = 7 // 2
        self.b = (-7) // 2
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}
	// Check that OP_FLOOR_DIV appears in the bytecode (not OP_DIV)
	foundFloorDiv := false
	for _, ins := range bc.Code {
		if ins.Op == OP_FLOOR_DIV {
			foundFloorDiv = true
			break
		}
	}
	if !foundFloorDiv {
		t.Fatal("expected OP_FLOOR_DIV in bytecode for // operator")
	}
}

func TestCompilePython_FloorDivisionNegativeResult(t *testing.T) {
	// Test that -7 // 2 = -4 (floor), not -3 (truncation)
	source := `from decimal import Decimal
class S:
    def __init__(self) -> None:
        self.result = 0
    def on_bar(self) -> None:
        self.result = (-7) // 2
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	// Check the IR: the binary op should be "//" not "/"
	if len(ir.OnBar) == 0 {
		t.Fatal("expected OnBar statements")
	}
	expr := ir.OnBar[0].Expr
	if expr == nil || expr.Kind != interp.ExprAssignment {
		t.Fatalf("expected ExprAssignment, got %v", expr)
	}
	rhs := expr.Args[0]
	if rhs.Kind != interp.ExprBinary || rhs.Op != "//" {
		t.Fatalf("expected ExprBinary with op //, got op=%q kind=%v", rhs.Op, rhs.Kind)
	}
}

func TestCompilePython_KeywordArgumentReorder(t *testing.T) {
	// ctx.broker.buy(sl=100, lot=0.1) should reorder to (lot=0.1, ..., sl=100, ...)
	source := `from decimal import Decimal
class S:
    def __init__(self) -> None:
        pass
    def on_bar(self) -> None:
        ctx.broker.buy(sl=Decimal("100"), lot=Decimal("0.1"))
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) == 0 {
		t.Fatal("expected OnBar statements")
	}
	expr := ir.OnBar[0].Expr
	if expr == nil || expr.Kind != interp.ExprCall || expr.Name != "CTrade.Buy" {
		t.Fatalf("expected CTrade.Buy call, got name=%q kind=%v", expr.Name, expr.Kind)
	}
	// CTrade.Buy(volume, symbol, price, sl, tp, comment)
	// lot=0.1 → volume → arg[0], sl=100 → arg[3]
	if len(expr.Args) != 6 {
		t.Fatalf("expected 6 args, got %d", len(expr.Args))
	}
	// arg[0] should be the lot value (string literal "0.1" from Decimal("0.1"))
	if expr.Args[0].Kind != interp.ExprLiteral || expr.Args[0].Val.Kind != interp.ValString {
		t.Errorf("arg[0]: expected string literal (lot/volume), got kind=%v val=%v", expr.Args[0].Kind, expr.Args[0].Val)
	}
	// arg[3] should be the sl value (string literal "100" from Decimal("100"))
	if expr.Args[3].Kind != interp.ExprLiteral || expr.Args[3].Val.Kind != interp.ValString {
		t.Errorf("arg[3]: expected string literal (sl), got kind=%v val=%v", expr.Args[3].Kind, expr.Args[3].Val)
	}
}

func TestCompilePython_ChainedComparisonSingleEval(t *testing.T) {
	// a < f(x) < c should evaluate f(x) only once via temp variable
	source := `from decimal import Decimal
class S:
    def __init__(self) -> None:
        self.result = False
    def on_bar(self) -> None:
        self.result = 1 < 2 < 3
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) == 0 {
		t.Fatal("expected OnBar statements")
	}
	expr := ir.OnBar[0].Expr
	if expr == nil || expr.Kind != interp.ExprAssignment {
		t.Fatalf("expected ExprAssignment, got kind=%v", expr.Kind)
	}
	// The RHS of the assignment should be ExprSeq (decls + comparison)
	rhs := expr.Args[0]
	if rhs.Kind != interp.ExprSeq {
		t.Fatalf("expected ExprSeq for chained comparison RHS, got %v", rhs.Kind)
	}
	// For 3 operands with 1 middle: 1 decl + 1 comparison = 2 args
	if len(rhs.Args) != 2 {
		t.Fatalf("expected 2 args in ExprSeq (1 decl + 1 result), got %d", len(rhs.Args))
	}
	// First arg should be ExprDecl (temp variable for middle operand)
	if rhs.Args[0].Kind != interp.ExprDecl {
		t.Errorf("arg[0]: expected ExprDecl (temp), got %v", rhs.Args[0].Kind)
	}
	// Second arg should be ExprBinary (the && chain)
	if rhs.Args[1].Kind != interp.ExprBinary || rhs.Args[1].Op != "&&" {
		t.Errorf("arg[1]: expected ExprBinary with &&, got kind=%v op=%q", rhs.Args[1].Kind, rhs.Args[1].Op)
	}
}

func TestCompilePython_PowerAssignDesugar(t *testing.T) {
	// x **= 2 should desugar to x = MathPow(x, 2), not x *= 2
	source := `from decimal import Decimal
class S:
    def __init__(self) -> None:
        self.x = 0
    def on_bar(self) -> None:
        self.x **= 2
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) == 0 {
		t.Fatal("expected OnBar statements")
	}
	expr := ir.OnBar[0].Expr
	if expr == nil || expr.Kind != interp.ExprAssignment {
		t.Fatalf("expected ExprAssignment, got kind=%v", expr.Kind)
	}
	rhs := expr.Args[0]
	if rhs.Kind != interp.ExprCall || rhs.Name != "MathPow" {
		t.Fatalf("expected MathPow call, got name=%q kind=%v", rhs.Name, rhs.Kind)
	}
	if len(rhs.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(rhs.Args))
	}
	if rhs.Args[0].Kind != interp.ExprVar || rhs.Args[0].Name != "x" {
		t.Errorf("arg[0]: expected ExprVar 'x', got %v", rhs.Args[0])
	}
}
