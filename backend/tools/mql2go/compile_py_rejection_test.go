package mql2go

import (
	"fmt"
	"strings"
	"testing"

	"alphaforge/tools/mql2go/interp"
)

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
		pythonName  string
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
		name    string
		method  string
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
