package mql2go

import (
	"testing"

	"anttrader/tools/mql2go/interp"
)

func TestCompileToIR_CTradeInstance(t *testing.T) {
	source := `
#include <Trade/Trade.mqh>
input int MagicNumber = 12345;

CTrade trade;

int OnInit()
{
    trade.SetExpertMagicNumber(MagicNumber);
    return 0;
}

void OnBar()
{
    double ma = iMA(14, 0, MODE_SMA);
    if (Close[1] > ma)
    {
        trade.Buy(0.1, _Symbol, 0, 0, 0, "buy");
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

	// Check that "trade" is registered as a global with type "CTrade"
	foundTrade := false
	for _, g := range ir.Globals {
		if g.Name == "trade" && g.Type == "CTrade" {
			foundTrade = true
			break
		}
	}
	if !foundTrade {
		t.Error("CTrade 'trade' global not found")
	}

	// OnInit should have statements (SetExpertMagicNumber call)
	if len(ir.OnInit) == 0 {
		t.Error("OnInit should have statements")
	}
}

func TestCompileToIR_UserDefinedStruct(t *testing.T) {
	source := `
struct MyConfig {
    int magic;
    double lotSize;
    string symbol;
};

MyConfig cfg;

int OnInit()
{
    cfg.magic = 12345;
    cfg.lotSize = 0.1;
    cfg.symbol = "EURUSD";
    return 0;
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	// Check that "cfg" is registered as a global with type "MyConfig"
	foundCfg := false
	for _, g := range ir.Globals {
		if g.Name == "cfg" && g.Type == "MyConfig" {
			foundCfg = true
			break
		}
	}
	if !foundCfg {
		t.Error("MyConfig 'cfg' global not found")
	}

	// OnInit should have field assignment statements
	if len(ir.OnInit) == 0 {
		t.Error("OnInit should have statements")
	}
}

func TestPreprocessMQL_Define(t *testing.T) {
	source := `#define MAGIC 12345
#define LOT_SIZE 0.1

int OnInit()
{
    int magic = MAGIC;
    double lot = LOT_SIZE;
    return 0;
}
`
	processed := PreprocessMQL(source)
	ir, err := CompileToIR(processed)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	// The #define should have substituted MAGIC→12345 and LOT_SIZE→0.1
	// Check that OnInit has assignment statements
	if len(ir.OnInit) == 0 {
		t.Error("OnInit should have statements")
	}

	// Verify assignments exist
	foundAssign := false
	for _, s := range ir.OnInit {
		if s.Kind == interp.StmtExpr && s.Expr != nil && (s.Expr.Kind == interp.ExprAssignment || s.Expr.Kind == interp.ExprDecl) {
			foundAssign = true
			break
		}
	}
	if !foundAssign {
		t.Error("should have assignment statements")
	}
}

func TestPreprocessMQL_PropertyStripped(t *testing.T) {
	source := `#property strict
#property copyright "Test"
#property version "1.00"

int OnInit()
{
    return 0;
}
`
	processed := PreprocessMQL(source)
	// #property lines should be replaced with empty lines
	if containsProperty(processed) {
		t.Error("#property lines should be stripped")
	}
}

func containsProperty(s string) bool {
	for _, line := range splitLines(s) {
		if len(line) > 9 && line[:9] == "#property" {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
