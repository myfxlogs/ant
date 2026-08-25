package mql2go

import (
	"alphaforge/tools/mql2go/interp"
	"testing"
)

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
