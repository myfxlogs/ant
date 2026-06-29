package mql2go

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"anttrader/tools/mql2go/interp"
)

// sampleMQL4EA is a representative MQL4 EA for testing the IR pipeline.
const sampleMQL4EA = `
extern int MagicNumber = 12345;
extern double LotSize = 0.1;
extern int MAPeriod = 14;
extern double StopLoss = 50;
extern double TakeProfit = 100;

double maValue;
bool gridPlaced = false;

int OnInit() {
    return 0;
}

void OnTick() {
    maValue = iMA(NULL, 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    
    if (maValue > Close[1]) {
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, Ask - StopLoss * Point, Ask + TakeProfit * Point, "MA Buy", MagicNumber, 0, clrGreen);
    }
    
    if (maValue < Close[1]) {
        OrderSend(Symbol(), OP_SELL, LotSize, Bid, 5, Bid + StopLoss * Point, Bid - TakeProfit * Point, "MA Sell", MagicNumber, 0, clrRed);
    }
}

void OnDeinit(const int reason) {
    OrderClose(OrderTicket(), OrderLots(), Bid, 5, clrWhite);
}
`

// sampleMQL5EA tests MQL5 CTrade detection.
const sampleMQL5EA = `
#include <Trade/Trade.mqh>
input int MagicNumber = 12345;
input double LotSize = 0.1;

CTrade trade;

int OnInit() {
    trade.SetExpertMagicNumber(MagicNumber);
    return 0;
}

void OnTick() {
    if (PositionsTotal() > 0) {
        trade.PositionClose(Symbol());
    }
    trade.Buy(LotSize, Symbol(), 0, 0, 0, "Buy");
}
`

func TestCompileToIR_BasicMQL4(t *testing.T) {
	ir, err := CompileToIR(sampleMQL4EA)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	if ir.Version != "mql4" {
		t.Errorf("expected mql4, got %s", ir.Version)
	}

	if len(ir.Params) < 4 {
		t.Errorf("expected at least 4 params, got %d", len(ir.Params))
	}

	if len(ir.OnTick) == 0 {
		t.Error("expected OnTick statements")
	}

	if len(ir.OnInit) == 0 {
		t.Error("expected OnInit statements")
	}
}

func TestCompileToIR_MQL5Detection(t *testing.T) {
	ir, err := CompileToIR(sampleMQL5EA)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	if ir.Version != "mql5" {
		t.Errorf("expected mql5, got %s", ir.Version)
	}

	foundCTrade := false
	for _, g := range ir.Globals {
		if g.Type == "CTrade" {
			foundCTrade = true
		}
	}
	if !foundCTrade {
		t.Error("expected CTrade global variable")
	}
}

func TestAnalyze_MQL4FullCoverage(t *testing.T) {
	ir, err := CompileToIR(sampleMQL4EA)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	rep := interp.Analyze(ir)
	if rep.Version != "mql4" {
		t.Errorf("expected mql4, got %s", rep.Version)
	}

	if rep.TotalCalls == 0 {
		t.Error("expected some calls to be detected")
	}

	if len(rep.Indicators) == 0 {
		t.Error("expected at least 1 indicator")
	}

	if rep.ExecKind != "on_tick" {
		t.Errorf("expected on_tick, got %s", rep.ExecKind)
	}
}

func TestAnalyze_MQL5CTradeCoverage(t *testing.T) {
	ir, err := CompileToIR(sampleMQL5EA)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	rep := interp.Analyze(ir)
	if rep.Version != "mql5" {
		t.Errorf("expected mql5, got %s", rep.Version)
	}

	if rep.TotalCalls == 0 {
		t.Error("expected calls to be detected")
	}

	if rep.SupportedCalls == 0 {
		t.Error("expected some supported calls (CTrade methods are implemented)")
	}
}

func TestAnalyze_MQL5UnknownCTradeMethod_BlindSpot(t *testing.T) {
	source := `
#include <Trade/Trade.mqh>
CTrade trade;
void OnTick() {
    trade.SomeUnknownMethod(42);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	rep := interp.Analyze(ir)
	found := false
	for _, bs := range rep.BlindSpots {
		if bs.Builtin == "SomeUnknownMethod" {
			found = true
			if bs.Severity != "致命" {
				t.Errorf("expected 致命 severity, got %s", bs.Severity)
			}
		}
	}
	if !found {
		t.Error("expected SomeUnknownMethod as a blind spot")
	}
}

func TestAnalyze_iCustom_BlindSpot(t *testing.T) {
	source := `
void OnTick() {
    double v = iCustom(NULL, 0, "MyInd", 14, 0, 1);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	rep := interp.Analyze(ir)
	found := false
	for _, bs := range rep.BlindSpots {
		if bs.Builtin == "iCustom" {
			found = true
		}
	}
	if !found {
		t.Error("expected iCustom as a blind spot")
	}
}

func TestAnalyze_UserFuncNotBlindSpot(t *testing.T) {
	source := `
double MyHelper(double x) {
    return x * 2;
}
void OnTick() {
    double v = MyHelper(10);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	rep := interp.Analyze(ir)
	for _, bs := range rep.BlindSpots {
		if bs.Builtin == "MyHelper" {
			t.Error("user-defined function should NOT be a blind spot")
		}
	}
}

func TestAnalyze_MQL4VersionFiltering(t *testing.T) {
	source := `
extern int Magic = 100;
void OnTick() {
    OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0, "", Magic, 0, clrGreen);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	if ir.Version != "mql4" {
		t.Fatalf("expected mql4, got %s", ir.Version)
	}
}

func TestAnalyze_MQL5VersionFiltering(t *testing.T) {
	source := `
#include <Trade/Trade.mqh>
CTrade trade;
void OnTick() {
    trade.Buy(0.1);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	if ir.Version != "mql5" {
		t.Fatalf("expected mql5, got %s", ir.Version)
	}
}

func TestGenerateFromIR_ParsesAsValidGo(t *testing.T) {
	sources := []struct {
		name     string
		source   string
		strategy string
	}{
		{"MQL4_EA", sampleMQL4EA, "TestMQL4"},
		{"MQL5_EA", sampleMQL5EA, "TestMQL5"},
		{"MQL4_OrderSelect", `
extern int Magic = 100;
void OnTick() {
    for (int i = OrdersTotal() - 1; i >= 0; i--) {
        if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
            if (OrderMagicNumber() == Magic) {
                OrderClose(OrderTicket(), OrderLots(), Bid, 5);
            }
        }
    }
}
`, "TestOrderLoop"},
		{"MQL5_PositionLoop", `
#include <Trade\Trade.mqh>
CTrade trade;
input int Magic = 100;
void OnTick() {
    for (int i = PositionsTotal() - 1; i >= 0; i--) {
        ulong ticket = PositionGetTicket(i);
        if (PositionGetInteger(POSITION_MAGIC) == Magic) {
            trade.PositionClose(ticket);
        }
    }
}
`, "TestPosLoop"},
	}

	for _, tc := range sources {
		t.Run(tc.name, func(t *testing.T) {
			ir, err := CompileToIR(tc.source)
			if err != nil {
				t.Fatalf("CompileToIR failed: %v", err)
			}
			code := GenerateFromIR(ir, tc.strategy)

			fset := token.NewFileSet()
			_, err = parser.ParseFile(fset, "generated.go", code, parser.AllErrors)
			if err != nil {
				t.Errorf("generated code does not parse as valid Go: %v\n--- code ---\n%s", err, code)
			}
		})
	}
}

func TestGenerateFromIR_StructuralChecks(t *testing.T) {
	ir, err := CompileToIR(sampleMQL4EA)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	code := GenerateFromIR(ir, "TestStrategy")
	if code == "" {
		t.Fatal("GenerateFromIR returned empty code")
	}

	if !strings.Contains(code, "package ") {
		t.Error("generated code missing package declaration")
	}
	if !strings.Contains(code, "type TestStrategy struct") {
		t.Error("generated code missing struct declaration")
	}
	if !strings.Contains(code, "func (s *TestStrategy) OnInit") {
		t.Error("generated code missing OnInit")
	}
	if !strings.Contains(code, "var _ sdk.Strategy = (*TestStrategy)(nil)") {
		t.Error("generated code missing sdk.Strategy interface assertion")
	}
}

func TestGenerateFromIR_EndToEnd_MQL4(t *testing.T) {
	source := `
extern int MagicNumber = 12345;
extern double LotSize = 0.1;
extern int MAPeriod = 14;
extern double StopLoss = 50;
extern double TakeProfit = 100;
double maValue;
int OnInit() { return 0; }
void OnTick() {
    maValue = iMA(NULL, 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    if (maValue > Close[1]) {
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, Ask - StopLoss * Point, Ask + TakeProfit * Point, "Buy", MagicNumber, 0, clrGreen);
    }
    if (maValue < Close[1]) {
        OrderSend(Symbol(), OP_SELL, LotSize, Bid, 5, Bid + StopLoss * Point, Bid - TakeProfit * Point, "Sell", MagicNumber, 0, clrRed);
    }
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	code := GenerateFromIR(ir, "E2EMQL4")

	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, "e2e_mql4.go", code, parser.AllErrors)
	if err != nil {
		t.Errorf("generated MQL4 code does not parse: %v\n%s", err, code)
	}
}

func TestGenerateFromIR_EndToEnd_MQL5(t *testing.T) {
	source := `
#include <Trade\Trade.mqh>
CTrade trade;
input int MagicNumber = 12345;
input double LotSize = 0.1;
input int MAPeriod = 14;
input double StopLoss = 50;
input double TakeProfit = 100;
double maValue;

int OnInit() { return 0; }

void OnTick() {
    maValue = iMA(NULL, 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    if (maValue > Close[1]) {
        trade.Buy(LotSize, _Symbol, 0, Ask - StopLoss * _Point, Ask + TakeProfit * _Point, "Buy");
    }
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	code := GenerateFromIR(ir, "E2EMQL5")

	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, "e2e_mql5.go", code, parser.AllErrors)
	if err != nil {
		t.Errorf("generated MQL5 code does not parse: %v\n%s", err, code)
	}
}

func TestCompileToIR_ThreadSafety(t *testing.T) {
	sources := []string{sampleMQL4EA, sampleMQL5EA, `
extern int X = 1;
void OnTick() { OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0, "", X, 0, clrGreen); }
`}
	done := make(chan bool, len(sources)*3)

	for _, src := range sources {
		for i := 0; i < 3; i++ {
			go func(s string) {
				defer func() { done <- true }()
				ir, err := CompileToIR(s)
				if err != nil {
					t.Errorf("CompileToIR failed: %v", err)
					return
				}
				if ir == nil {
					t.Error("CompileToIR returned nil IR")
				}
			}(src)
		}
	}

	for i := 0; i < len(sources)*3; i++ {
		<-done
	}
}

func TestAnalyze_StubIndicatorIsWarning(t *testing.T) {
	source := `
void OnTick() {
    double v = iAlligator(NULL, 0, 13, 8, 8, 5, 5, 3, MODE_SMMA, PRICE_MEDIAN, 1);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	rep := interp.Analyze(ir)
	for _, bs := range rep.BlindSpots {
		if bs.Builtin == "iAlligator" {
			if bs.Severity != "警告" {
				t.Errorf("expected iAlligator (stub) to be 警告, got %s", bs.Severity)
			}
		}
	}
}

// ── Fix 18-26 regression tests ──────────────────────────────────────

func TestGenIR_DecimalRequireFromString(t *testing.T) {
	source := `
extern double LotSize = 0.1;
void OnTick() {}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	code := GenerateFromIR(ir, "DecTest")
	if !strings.Contains(code, `decimal.RequireFromString("0.1")`) {
		t.Errorf("expected decimal.RequireFromString(\"0.1\") in generated code:\n%s", code)
	}
	// Verify the generated code parses as valid Go.
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "dec_test.go", code, parser.AllErrors); err != nil {
		t.Errorf("generated code does not parse: %v\n%s", err, code)
	}
}

func TestGenIR_StringParamNoDoubleQuotes(t *testing.T) {
	source := `
extern string Comment = "test";
void OnTick() {}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	code := GenerateFromIR(ir, "StrTest")
	if !strings.Contains(code, `ctx.ParamString("Comment", "test")`) {
		t.Errorf("expected ctx.ParamString(\"Comment\", \"test\") in generated code:\n%s", code)
	}
	if strings.Contains(code, `""test""`) {
		t.Errorf("found double-quoted string default value in generated code:\n%s", code)
	}
}

func TestGenIR_StdImportsConditional(t *testing.T) {
	t.Run("with_math", func(t *testing.T) {
		source := `void OnTick() { double v = MathSqrt(2.0); }`
		ir, err := CompileToIR(source)
		if err != nil {
			t.Fatalf("CompileToIR failed: %v", err)
		}
		code := GenerateFromIR(ir, "MathImp")
		if !strings.Contains(code, `"math"`) {
			t.Errorf("expected \"math\" import when MathSqrt is used:\n%s", code)
		}
	})

	t.Run("without_stdlib", func(t *testing.T) {
		source := `
extern double X = 0.1;
void OnTick() { double v = X; }
`
		ir, err := CompileToIR(source)
		if err != nil {
			t.Fatalf("CompileToIR failed: %v", err)
		}
		code := GenerateFromIR(ir, "NoImp")
		for _, pkg := range []string{`"math"`, `"fmt"`, `"strings"`, `"time"`} {
			if strings.Contains(code, pkg) {
				t.Errorf("unexpected stdlib import %s in code without stdlib builtins:\n%s", pkg, code)
			}
		}
	})

	t.Run("with_strings", func(t *testing.T) {
		source := `void OnTick() { int n = StringFind("hello", "ell"); }`
		ir, err := CompileToIR(source)
		if err != nil {
			t.Fatalf("CompileToIR failed: %v", err)
		}
		code := GenerateFromIR(ir, "StrImp")
		if !strings.Contains(code, `"strings"`) {
			t.Errorf("expected \"strings\" import when StringFind is used:\n%s", code)
		}
	})
}

func TestGenIR_OrderSendTypeSideMapping(t *testing.T) {
	source := `
extern int Magic = 100;
void OnTick() {
    OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0, "", Magic, 0, clrGreen);
    OrderSend(Symbol(), OP_SELL, 0.1, Bid, 5, 0, 0, "", Magic, 0, clrRed);
    OrderSend(Symbol(), OP_BUYLIMIT, 0.1, 0, 5, 0, 0, "", Magic, 0, clrGreen);
    OrderSend(Symbol(), OP_SELLLIMIT, 0.1, 0, 5, 0, 0, "", Magic, 0, clrRed);
    OrderSend(Symbol(), OP_BUYSTOP, 0.1, 0, 5, 0, 0, "", Magic, 0, clrGreen);
    OrderSend(Symbol(), OP_SELLSTOP, 0.1, 0, 5, 0, 0, "", Magic, 0, clrRed);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	code := GenerateFromIR(ir, "OrdSend")

	want := []struct {
		orderType string
		side      string
	}{
		{"sdk.OrderMarket", "sdk.SideBuy"},
		{"sdk.OrderMarket", "sdk.SideSell"},
		{"sdk.OrderLimit", "sdk.SideBuy"},
		{"sdk.OrderLimit", "sdk.SideSell"},
		{"sdk.OrderStop", "sdk.SideBuy"},
		{"sdk.OrderStop", "sdk.SideSell"},
	}
	for _, tc := range want {
		if !strings.Contains(code, "Type: "+tc.orderType+",") {
			t.Errorf("expected OrderType %s in generated code", tc.orderType)
		}
		if !strings.Contains(code, "Side: "+tc.side+",") {
			t.Errorf("expected Side %s in generated code", tc.side)
		}
	}

	// Verify the generated code parses as valid Go.
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "ord_send.go", code, parser.AllErrors); err != nil {
		t.Errorf("generated code does not parse: %v\n%s", err, code)
	}
}

func TestGenIR_GlobalVarSPrefix(t *testing.T) {
	source := `
extern int Magic = 100;
double myGlobal;
void OnTick() {
    myGlobal = Magic;
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	code := GenerateFromIR(ir, "GlobTest")
	if !strings.Contains(code, "s.myGlobal = s.Magic") {
		t.Errorf("expected s.myGlobal = s.Magic in generated code:\n%s", code)
	}
}

func TestGenIR_IntTernaryReturnType(t *testing.T) {
	source := `
extern int A = 1;
extern int B = 2;
void OnTick() {
    int x = A > B ? A : B;
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	code := GenerateFromIR(ir, "TernTest")
	if !strings.Contains(code, "func() int32 {") {
		t.Errorf("expected func() int32 { for int ternary:\n%s", code)
	}
	if strings.Contains(code, "func() decimal.Decimal {") {
		t.Errorf("should not have decimal.Decimal return for int ternary:\n%s", code)
	}
}

func TestGenIR_IfTrueDeadCodeElimination(t *testing.T) {
	source := `
void OnTick() {
    if (true) {
        double x = 0.1;
    }
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	code := GenerateFromIR(ir, "DeadCode")
	if strings.Contains(code, "if true {") {
		t.Errorf("expected dead code elimination for 'if true', but found 'if true {' in:\n%s", code)
	}
	if !strings.Contains(code, `decimal.RequireFromString("0.1")`) {
		t.Errorf("expected body content after dead code elimination:\n%s", code)
	}
}

func TestGenIR_IfFalseDeadCodeElimination(t *testing.T) {
	source := `
void OnTick() {
    if (false) {
        double x = 0.1;
    } else {
        double y = 0.2;
    }
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	code := GenerateFromIR(ir, "DeadFalse")
	if strings.Contains(code, "if false {") {
		t.Errorf("expected dead code elimination for 'if false', but found 'if false {' in:\n%s", code)
	}
	if !strings.Contains(code, `decimal.RequireFromString("0.2")`) {
		t.Errorf("expected else body content after dead code elimination:\n%s", code)
	}
	if strings.Contains(code, `decimal.RequireFromString("0.1")`) {
		t.Errorf("if-false body should be eliminated, but found 0.1 literal in:\n%s", code)
	}
}

func TestGenIR_IsDecimalExpr_BuiltinCalls(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		wantMethod string
	}{
		{"MathPow", `void OnTick() { if (MathPow(2.0, 3.0) > 0.1) { double x = 0.1; } }`, ".GreaterThan("},
		{"MathMax", `void OnTick() { if (MathMax(0.1, 0.2) > 0.15) { double x = 0.1; } }`, ".GreaterThan("},
		{"MathMin", `void OnTick() { if (MathMin(0.1, 0.2) < 0.05) { double x = 0.1; } }`, ".LessThan("},
		{"NormalizeDouble", `void OnTick() { if (NormalizeDouble(0.1, 5) > 0.05) { double x = 0.1; } }`, ".GreaterThan("},
		{"StringToDouble", `void OnTick() { if (StringToDouble("0.1") > 0.05) { double x = 0.1; } }`, ".GreaterThan("},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ir, err := CompileToIR(tc.source)
			if err != nil {
				t.Fatalf("CompileToIR failed: %v", err)
			}
			code := GenerateFromIR(ir, "DecExpr"+tc.name)
			if !strings.Contains(code, tc.wantMethod) {
				t.Errorf("expected decimal comparison method %s for %s:\n%s", tc.wantMethod, tc.name, code)
			}
		})
	}
}
