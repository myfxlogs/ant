package mql2go

import (
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
