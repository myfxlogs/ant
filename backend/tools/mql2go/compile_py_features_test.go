package mql2go

import (
	"strings"
	"testing"

	"alphaforge/tools/mql2go/interp"
)

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
