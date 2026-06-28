package mql2go

import (
	"testing"

	"anttrader/tools/mql2go/interp"
)

func TestCompileToIR_UserFunction(t *testing.T) {
	source := `
int OnInit()
{
    return 0;
}

double CalcRisk(double pct, double balance)
{
    return pct * balance;
}

void OnBar()
{
    double risk = CalcRisk(0.02, 10000.0);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	if ir.Funcs == nil {
		t.Fatal("Funcs map should be initialized")
	}

	fn, ok := ir.Funcs["CalcRisk"]
	if !ok {
		t.Fatal("CalcRisk function not found in Funcs")
	}

	if len(fn.Params) != 2 {
		t.Errorf("CalcRisk params = %d, want 2", len(fn.Params))
	}
	if fn.Params[0].Name != "pct" {
		t.Errorf("CalcRisk param[0] = %s, want pct", fn.Params[0].Name)
	}
	if fn.Params[1].Name != "balance" {
		t.Errorf("CalcRisk param[1] = %s, want balance", fn.Params[1].Name)
	}

	// Function body should have a return statement
	if len(fn.Body) == 0 {
		t.Fatal("CalcRisk body should have statements")
	}
	if fn.Body[0].Kind != interp.StmtReturn {
		t.Errorf("CalcRisk body[0] kind = %v, want StmtReturn", fn.Body[0].Kind)
	}

	// OnBar should call CalcRisk
	if len(ir.OnBar) == 0 {
		t.Fatal("OnBar should have statements")
	}
}

func TestCompileToIR_BreakContinue(t *testing.T) {
	source := `
void OnBar()
{
    for (int i = 0; i < 10; i++)
    {
        if (i == 5)
            break;
        if (i == 2)
            continue;
    }
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	if len(ir.OnBar) == 0 {
		t.Fatal("OnBar should have statements")
	}

	// OnBar[0] should be a for loop
	forStmt := ir.OnBar[0]
	if forStmt.Kind != interp.StmtFor {
		t.Errorf("OnBar[0] kind = %v, want StmtFor", forStmt.Kind)
	}

	// Body should contain if statements with break and continue
	foundBreak := false
	foundContinue := false
	for _, s := range forStmt.Body {
		if s.Kind == interp.StmtIf {
			for _, body := range s.Body {
				if body.Kind == interp.StmtBreak {
					foundBreak = true
				}
				if body.Kind == interp.StmtContinue {
					foundContinue = true
				}
			}
		}
	}
	if !foundBreak {
		t.Error("break statement not found in compiled IR")
	}
	if !foundContinue {
		t.Error("continue statement not found in compiled IR")
	}
}

func TestCompileToIR_DoWhile(t *testing.T) {
	source := `
void OnBar()
{
    int i = 0;
    do
    {
        i++;
    }
    while (i < 5);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	if len(ir.OnBar) < 2 {
		t.Fatal("OnBar should have at least 2 statements")
	}

	// Find the do-while statement
	foundDoWhile := false
	for _, s := range ir.OnBar {
		if s.Kind == interp.StmtDoWhile {
			foundDoWhile = true
			if s.Cond == nil {
				t.Error("do-while condition should not be nil")
			}
			if len(s.Body) == 0 {
				t.Error("do-while body should not be empty")
			}
		}
	}
	if !foundDoWhile {
		t.Error("StmtDoWhile not found in compiled IR")
	}
}

func TestCompileToIR_Enum(t *testing.T) {
	source := `
enum TradeMode
{
    MODE_MANUAL,
    MODE_AUTO,
    MODE_SEMI = 10
};

int OnInit()
{
    int mode = MODE_AUTO;
    return 0;
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	if ir.Enums == nil {
		t.Fatal("Enums map should be initialized")
	}

	if v, ok := ir.Enums["MODE_MANUAL"]; !ok || v != 0 {
		t.Errorf("MODE_MANUAL = %d, ok=%v, want 0", v, ok)
	}
	if v, ok := ir.Enums["MODE_AUTO"]; !ok || v != 1 {
		t.Errorf("MODE_AUTO = %d, ok=%v, want 1", v, ok)
	}
	if v, ok := ir.Enums["MODE_SEMI"]; !ok || v != 10 {
		t.Errorf("MODE_SEMI = %d, ok=%v, want 10", v, ok)
	}
}

func TestCompileToIR_CompoundAssignment(t *testing.T) {
	source := `
void OnBar()
{
    int x = 10;
    x += 5;
    x -= 3;
    x *= 2;
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	if len(ir.OnBar) < 4 {
		t.Fatal("OnBar should have at least 4 statements")
	}

	// Check for compound assignment expressions
	foundPlusEq := false
	foundMinusEq := false
	foundMulEq := false
	for _, s := range ir.OnBar {
		if s.Expr != nil && s.Expr.Kind == interp.ExprCompoundAssign {
			switch s.Expr.Op {
			case "+=":
				foundPlusEq = true
			case "-=":
				foundMinusEq = true
			case "*=":
				foundMulEq = true
			}
		}
	}
	if !foundPlusEq {
		t.Error("+= not found in compiled IR")
	}
	if !foundMinusEq {
		t.Error("-= not found in compiled IR")
	}
	if !foundMulEq {
		t.Error("*= not found in compiled IR")
	}
}

func TestCompileToIR_IncludeStub(t *testing.T) {
	source := `
#include <Trade/Trade.mqh>
CTrade trade;

int OnInit()
{
    trade.SetExpertMagicNumber(12345);
    return 0;
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	// CTrade trade; should be registered as a global
	foundTrade := false
	for _, g := range ir.Globals {
		if g.Name == "trade" && g.Type == "CTrade" {
			foundTrade = true
			break
		}
	}
	if !foundTrade {
		t.Error("CTrade 'trade' global not found (include stub may have failed)")
	}
}
