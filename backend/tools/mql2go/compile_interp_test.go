package mql2go

import (
	"testing"

	"anttrader/tools/mql2go/interp"
)

func TestCompileToIR_MQL4_Simple(t *testing.T) {
	source := `
extern int MagicNumber = 12345;
extern double LotSize = 0.1;

int giCounter = 0;
double gdProfit = 0.0;

int OnInit()
{
    giCounter = 0;
    return 0;
}

void OnBar()
{
    double ma = iMA(14, 0, MODE_SMA);
    if (Close[1] > ma)
    {
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 10, 0, 0, "buy", MagicNumber, 0, clrNone);
    }
    else if (Close[1] < ma)
    {
        OrderSend(Symbol(), OP_SELL, LotSize, Bid, 10, 0, 0, "sell", MagicNumber, 0, clrNone);
    }
}

void OnDeinit()
{
    Print("EA stopped");
}
`

	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	if ir.Version != "mql4" {
		t.Errorf("version = %s, want mql4", ir.Version)
	}

	// Check params
	if len(ir.Params) != 2 {
		t.Errorf("params = %d, want 2", len(ir.Params))
	} else {
		if ir.Params[0].Name != "MagicNumber" {
			t.Errorf("param[0] name = %s, want MagicNumber", ir.Params[0].Name)
		}
		if ir.Params[1].Name != "LotSize" {
			t.Errorf("param[1] name = %s, want LotSize", ir.Params[1].Name)
		}
	}

	// Check globals
	if len(ir.Globals) != 2 {
		t.Errorf("globals = %d, want 2", len(ir.Globals))
	} else {
		if ir.Globals[0].Name != "giCounter" {
			t.Errorf("global[0] name = %s, want giCounter", ir.Globals[0].Name)
		}
		if ir.Globals[1].Name != "gdProfit" {
			t.Errorf("global[1] name = %s, want gdProfit", ir.Globals[1].Name)
		}
	}

	// Check OnBar has statements
	if len(ir.OnBar) == 0 {
		t.Error("OnBar should have statements")
	}

	// Check OnInit
	if len(ir.OnInit) == 0 {
		t.Error("OnInit should have statements")
	}

	// Check OnDeinit
	if len(ir.OnDeinit) == 0 {
		t.Error("OnDeinit should have statements")
	}
}

func TestCompileToIR_MQL5_CTrade(t *testing.T) {
	source := `
#include <Trade/Trade.mqh>
input int MagicNumber = 12345;
input double LotSize = 0.1;

CTrade trade;

int OnInit()
{
    trade.SetExpertMagicNumber(MagicNumber);
    return INIT_SUCCEEDED;
}

void OnBar()
{
    double fastMA = iMA(14, 0, MODE_SMA);
    double slowMA = iMA(28, 0, MODE_SMA);

    if (fastMA > slowMA)
    {
        trade.Buy(LotSize, _Symbol, 0, 0, 0, "buy");
    }
    else if (fastMA < slowMA)
    {
        trade.Sell(LotSize, _Symbol, 0, 0, 0, "sell");
    }
}
`

	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	if ir.Version != "mql5" {
		t.Errorf("version = %s, want mql5", ir.Version)
	}

	// Check OnBar has statements
	if len(ir.OnBar) == 0 {
		t.Error("OnBar should have statements")
	}

	// Verify the if statement has condition and body
	hasIf := false
	for _, s := range ir.OnBar {
		if s.Kind == interp.StmtIf {
			hasIf = true
			if s.Cond == nil {
				t.Error("if statement should have condition")
			}
			if len(s.Body) == 0 {
				t.Error("if statement should have body")
			}
			break
		}
	}
	if !hasIf {
		t.Error("OnBar should contain an if statement")
	}
}

func TestCompileToIR_ExpressionTypes(t *testing.T) {
	source := `
void OnBar()
{
    double a = 1.5;
    double b = 2.0;
    double c = a + b;
    bool flag = a > b;
    int x = 10;
    x++;
    if (a > b && c > 3.0)
    {
        Print("condition met");
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

	// Check for assignment statements
	foundAssign := false
	foundIf := false
	foundUpdate := false
	for _, s := range ir.OnBar {
		if s.Kind == interp.StmtExpr && s.Expr != nil {
			if s.Expr.Kind == interp.ExprAssignment {
				foundAssign = true
			}
			if s.Expr.Kind == interp.ExprUpdate {
				foundUpdate = true
			}
		}
		if s.Kind == interp.StmtIf {
			foundIf = true
		}
	}
	if !foundAssign {
		t.Error("should have assignment statements")
	}
	if !foundIf {
		t.Error("should have if statement")
	}
	if !foundUpdate {
		t.Error("should have update expression (x++)")
	}
}

func TestCompileToIR_ForLoop(t *testing.T) {
	source := `
void OnBar()
{
    int total = OrdersTotal();
    for (int i = 0; i < total; i++)
    {
        if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
        {
            if (OrderMagicNumber() == 12345)
            {
                OrderClose(OrderTicket(), OrderLots(), Bid, 5);
            }
        }
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

	// Find the for loop
	foundFor := false
	for _, s := range ir.OnBar {
		if s.Kind == interp.StmtFor {
			foundFor = true
			if s.Cond == nil {
				t.Error("for loop should have condition")
			}
			if len(s.Body) == 0 {
				t.Error("for loop should have body")
			}
			break
		}
	}
	if !foundFor {
		t.Error("should have a for loop")
	}
}
