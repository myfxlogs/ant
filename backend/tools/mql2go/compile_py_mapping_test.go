package mql2go

import (
	"strings"
	"testing"

	"alphaforge/tools/mql2go/interp"
)

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

func TestCompilePython_HigherTimeframeBars(t *testing.T) {
	source := `from decimal import Decimal
class S:
    def on_bar(self) -> None:
        c: float = ctx.bars_tf("H4").close(0)
        h: float = ctx.bars_tf("M15").high(1)
`
	ir, err := CompilePythonToIR(source)
	if err != nil {
		t.Fatalf("CompilePythonToIR failed: %v", err)
	}
	if len(ir.OnBar) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(ir.OnBar))
	}
	// First statement: c = iClose("", 240, 0) — symbol="" (primary), timeframe=240 (H4), shift=0
	stmt0 := ir.OnBar[0]
	if stmt0.Expr == nil || stmt0.Expr.Name != "c" {
		t.Fatalf("stmt[0]: expected name c, got %v", stmt0.Expr)
	}
	call0 := stmt0.Expr.Args[0]
	if call0.Name != "iClose" {
		t.Errorf("stmt[0]: expected iClose, got %s", call0.Name)
	}
	if len(call0.Args) != 3 {
		t.Fatalf("stmt[0]: expected 3 args, got %d", len(call0.Args))
	}
	// arg[0] = "" (primary symbol), arg[1] = 240 (H4), arg[2] = 0 (shift)
	if call0.Args[0].Kind != interp.ExprLiteral || call0.Args[0].Val.Kind != interp.ValString || call0.Args[0].Val.Str != "" {
		t.Errorf("stmt[0]: expected arg[0] = \"\", got %v", call0.Args[0])
	}
	if call0.Args[1].Kind != interp.ExprLiteral || call0.Args[1].Val.Kind != interp.ValInt || call0.Args[1].Val.ToInt() != 240 {
		t.Errorf("stmt[0]: expected arg[1] = 240 (H4), got %v", call0.Args[1])
	}
	// Second statement: h = iHigh("", 15, 1) — timeframe=15 (M15), shift=1
	stmt1 := ir.OnBar[1]
	call1 := stmt1.Expr.Args[0]
	if call1.Name != "iHigh" {
		t.Errorf("stmt[1]: expected iHigh, got %s", call1.Name)
	}
	if len(call1.Args) != 3 {
		t.Fatalf("stmt[1]: expected 3 args, got %d", len(call1.Args))
	}
	if call1.Args[1].Kind != interp.ExprLiteral || call1.Args[1].Val.Kind != interp.ValInt || call1.Args[1].Val.ToInt() != 15 {
		t.Errorf("stmt[1]: expected arg[1] = 15 (M15), got %v", call1.Args[1])
	}
}
