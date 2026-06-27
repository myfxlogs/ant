package mql2go

import (
	"strings"
	"testing"
)

// sampleMQL4EA is a representative MQL4 EA for testing the transpiler pipeline.
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

func TestAnalyze_BasicMQL4(t *testing.T) {
	intent, err := Analyze(sampleMQL4EA)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if intent.Meta.MQLVersion != "mql4" {
		t.Errorf("expected mql4, got %s", intent.Meta.MQLVersion)
	}

	if len(intent.Params) < 4 {
		t.Errorf("expected at least 4 params, got %d", len(intent.Params))
	}

	// Should detect OnTick execution
	if intent.Execution.Kind != ExecOnTick {
		t.Errorf("expected ExecOnTick, got %s", intent.Execution.Kind)
	}

	// Should detect indicators
	if len(intent.Indicators) == 0 {
		t.Error("expected at least 1 indicator")
	}

	// Should detect entries
	if len(intent.Entry) == 0 {
		t.Error("expected at least 1 entry rule")
	}

	// Should detect exits
	if len(intent.Exit) == 0 {
		t.Error("expected at least 1 exit rule")
	}
}

func TestAnalyze_MQL5Detection(t *testing.T) {
	intent, err := Analyze(sampleMQL5EA)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if intent.Meta.MQLVersion != "mql5" {
		t.Errorf("expected mql5, got %s", intent.Meta.MQLVersion)
	}

	// Should have CTrade blind spot
	foundCTrade := false
	for _, bs := range intent.BlindSpots {
		if bs.Category == "mql5_ctrade" {
			foundCTrade = true
		}
	}
	if !foundCTrade {
		t.Error("expected CTrade blind spot for MQL5 code")
	}

	// Should detect CTrade.Buy as an entry rule
	foundBuy := false
	for _, e := range intent.Entry {
		if e.Action == ActionMarketBuy {
			foundBuy = true
		}
	}
	if !foundBuy {
		t.Error("expected CTrade.Buy to produce a market_buy entry rule")
	}

	// Should detect PositionClose as an exit rule
	foundClose := false
	for _, e := range intent.Exit {
		if e.Action == "position_close" || strings.HasPrefix(e.Action, "position_close:") {
			foundClose = true
		}
	}
	if !foundClose {
		t.Error("expected PositionClose to produce a position_close exit rule")
	}
}

func TestAnalyze_BlindSpots(t *testing.T) {
	source := `
extern int Magic = 100;
void OnTick() {
    ObjectCreate("label", OBJ_LABEL, 0, 0, 0);
    OrderModify(OrderTicket(), OrderOpenPrice(), 0, 0, 0, clrWhite);
    SendMail("subject", "body");
}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	categories := map[string]bool{}
	for _, bs := range intent.BlindSpots {
		categories[bs.Category] = true
	}

	if !categories["chart_objects"] {
		t.Error("expected chart_objects blind spot")
	}
	if !categories["order_modify"] {
		t.Error("expected order_modify blind spot")
	}
	if !categories["notification"] {
		t.Error("expected notification blind spot")
	}
}

func TestAnalyze_RiskChecks(t *testing.T) {
	source := `
extern int Magic = 100;
void OnTick() {
    if (AccountFreeMargin() <= 0) return;
    OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0, "", Magic, 0, clrGreen);
}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if len(intent.Risk) == 0 {
		t.Error("expected at least 1 risk check")
	}

	found := false
	for _, r := range intent.Risk {
		if r.Kind == "margin_check" {
			found = true
		}
	}
	if !found {
		t.Error("expected margin_check risk check")
	}
}

func TestGenerate_Compiles(t *testing.T) {
	intent, err := Analyze(sampleMQL4EA)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	intent.Meta.Name = "TestStrategy"

	code := Generate(intent)
	if code == "" {
		t.Fatal("Generate returned empty code")
	}

	// Basic structural checks
	if !strings.Contains(code, "package ") {
		t.Error("generated code missing package declaration")
	}
	if !strings.Contains(code, "type TestStrategy struct") {
		t.Error("generated code missing struct declaration")
	}
	if !strings.Contains(code, "func (s *TestStrategy) OnInit") {
		t.Error("generated code missing OnInit")
	}
	if !strings.Contains(code, "func (s *TestStrategy) OnBar") {
		t.Error("generated code missing OnBar")
	}
	if !strings.Contains(code, "var _ sdk.Strategy = (*TestStrategy)(nil)") {
		t.Error("generated code missing sdk.Strategy interface assertion")
	}
}

func TestGenerate_AllIndicators(t *testing.T) {
	// Test that all indicator types generate valid code
	indicators := []IndicatorSpec{
		{SDKMethod: "ema", ResultVar: "emaVal", Params: map[string]string{"period": "14", "shift": "1"}},
		{SDKMethod: "rsi", ResultVar: "rsiVal", Params: map[string]string{"period": "14", "shift": "1"}},
		{SDKMethod: "atr", ResultVar: "atrVal", Params: map[string]string{"period": "14", "shift": "1"}},
		{SDKMethod: "macd", ResultVar: "macdVal", Params: map[string]string{"fast": "12", "slow": "26", "signal": "9", "shift": "1"}},
		{SDKMethod: "bands", ResultVar: "bandsVal", Params: map[string]string{"period": "20", "deviation": "2.0", "shift": "1"}},
		{SDKMethod: "stochastic", ResultVar: "stochVal", Params: map[string]string{"kperiod": "5", "dperiod": "3", "slowing": "3", "shift": "1"}},
		{SDKMethod: "cci", ResultVar: "cciVal", Params: map[string]string{"period": "14", "shift": "1"}},
		{SDKMethod: "adx", ResultVar: "adxVal", Params: map[string]string{"period": "14", "shift": "1"}},
		{SDKMethod: "momentum", ResultVar: "momVal", Params: map[string]string{"period": "14", "shift": "1"}},
		{SDKMethod: "wpr", ResultVar: "wprVal", Params: map[string]string{"period": "14", "shift": "1"}},
		{SDKMethod: "mfi", ResultVar: "mfiVal", Params: map[string]string{"period": "14", "shift": "1"}},
		{SDKMethod: "obv", ResultVar: "obvVal", Params: map[string]string{"shift": "1"}},
		{SDKMethod: "sar", ResultVar: "sarVal", Params: map[string]string{"step": "0.02", "maximum": "0.2", "shift": "1"}},
		{SDKMethod: "stddev", ResultVar: "stdVal", Params: map[string]string{"period": "14", "shift": "1"}},
	}

	intent := &StrategyIntent{
		Meta:       StrategyMeta{Name: "IndicatorTest", MQLVersion: "mql4"},
		Indicators: indicators,
		Execution:  ExecutionModel{Kind: ExecOnBar},
	}

	code := Generate(intent)

	for _, ind := range indicators {
		if !strings.Contains(code, ind.ResultVar) {
			t.Errorf("generated code missing result var %s for %s", ind.ResultVar, ind.SDKMethod)
		}
	}
}

func TestAnalyze_OrderModify(t *testing.T) {
	source := `
extern int Magic = 100;
extern double NewSL = 50;
extern double NewTP = 100;
void OnTick() {
    if (OrdersTotal() > 0) {
        OrderModify(OrderTicket(), OrderOpenPrice(), NewSL * Point, NewTP * Point, 0, clrWhite);
    }
}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if len(intent.Modifies) == 0 {
		t.Fatal("expected at least 1 modify rule")
	}

	mr := intent.Modifies[0]
	if mr.StopLoss == "" {
		t.Error("expected StopLoss to be extracted")
	}
	if mr.TakeProfit == "" {
		t.Error("expected TakeProfit to be extracted")
	}
	if mr.Condition == "" {
		t.Error("expected Condition to be extracted from enclosing if")
	}
}

func TestGenerate_ModifyCode(t *testing.T) {
	intent := &StrategyIntent{
		Meta:     StrategyMeta{Name: "ModifyTest", MQLVersion: "mql4"},
		Execution: ExecutionModel{Kind: ExecOnBar},
		Modifies: []ModifyRule{
			{StopLoss: "50", TakeProfit: "100", MagicVal: "s.magic"},
		},
	}

	code := Generate(intent)
	if !strings.Contains(code, "PositionModify") {
		t.Error("generated code missing PositionModify call")
	}
	if !strings.Contains(code, "decimal.NewFromFloat") {
		t.Error("generated code missing decimal conversion for SL/TP")
	}
}

func TestAnalyze_OrderCloseBy(t *testing.T) {
	source := `
extern int Magic = 100;
void OnTick() {
    if (OrdersTotal() > 1) {
        OrderCloseBy(OrderTicket(), OrderTicket() + 1);
    }
}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Should detect OrderCloseBy as an exit rule
	foundCloseBy := false
	for _, e := range intent.Exit {
		if e.Action == "position_close_by" || strings.HasPrefix(e.Action, "position_close_by:") {
			foundCloseBy = true
		}
	}
	if !foundCloseBy {
		t.Error("expected OrderCloseBy to produce a position_close_by exit rule")
	}

	// Should have OrderCloseBy blind spot
	foundBS := false
	for _, bs := range intent.BlindSpots {
		if bs.Category == "order_close_by" {
			foundBS = true
		}
	}
	if !foundBS {
		t.Error("expected order_close_by blind spot")
	}
}

func TestAnalyze_MQL5VersionFiltering(t *testing.T) {
	// MQL5 source with PositionSelect — should NOT trigger MQL4 OrderClose
	source := `
#include <Trade/Trade.mqh>
input int Magic = 100;
CTrade trade;
void OnTick() {
    if (PositionSelect(Symbol())) {
        trade.PositionClose(Symbol());
    }
}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if intent.Meta.MQLVersion != "mql5" {
		t.Fatalf("expected mql5, got %s", intent.Meta.MQLVersion)
	}

	// Should NOT have any MQL4-specific exits (OrderClose/OrderDelete/OrderCloseBy)
	for _, e := range intent.Exit {
		if strings.Contains(e.Action, "order_delete") && !strings.Contains(e.Action, "position") {
			t.Errorf("MQL5 source should not produce MQL4 order_delete exit: %s", e.Action)
		}
	}

	// Should have PositionSelect blind spot
	foundPosSelect := false
	for _, bs := range intent.BlindSpots {
		if bs.Category == "mql5_position_select" {
			foundPosSelect = true
		}
	}
	if !foundPosSelect {
		t.Error("expected mql5_position_select blind spot")
	}
}

func TestAnalyze_MQL4VersionFiltering(t *testing.T) {
	// MQL4 source — should NOT trigger MQL5 CTrade detection
	source := `
extern int Magic = 100;
void OnTick() {
    OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0, "", Magic, 0, clrGreen);
}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if intent.Meta.MQLVersion != "mql4" {
		t.Fatalf("expected mql4, got %s", intent.Meta.MQLVersion)
	}

	// Should NOT have MQL5 CTrade blind spot
	for _, bs := range intent.BlindSpots {
		if bs.Category == "mql5_ctrade" {
			t.Error("MQL4 source should not have mql5_ctrade blind spot")
		}
		if bs.Category == "mql5_position_select" {
			t.Error("MQL4 source should not have mql5_position_select blind spot")
		}
	}

	// Should have market_buy entry
	foundBuy := false
	for _, e := range intent.Entry {
		if e.Action == ActionMarketBuy {
			foundBuy = true
		}
	}
	if !foundBuy {
		t.Error("expected market_buy entry from OrderSend")
	}
}

func TestAnalyze_OrderSelectPattern(t *testing.T) {
	source := `
extern int Magic = 100;
extern int TrailingStop = 50;
void OnTick() {
    for (int i = OrdersTotal() - 1; i >= 0; i--) {
        if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
            if (OrderMagicNumber() == Magic && OrderSymbol() == Symbol()) {
                if (Bid - OrderOpenPrice() > Point * TrailingStop) {
                    if (OrderStopLoss() < Bid - Point * TrailingStop) {
                        OrderModify(OrderTicket(), OrderOpenPrice(), NormalizeDouble(Bid - Point * TrailingStop, Digits), OrderTakeProfit(), 0, Blue);
                    }
                }
            }
        }
    }
}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Should detect OrderLoopRule
	if len(intent.OrderLoops) == 0 {
		t.Fatal("expected at least one OrderLoopRule")
	}
	loop := intent.OrderLoops[0]
	if !loop.HasMagicFilter {
		t.Error("expected HasMagicFilter=true")
	}
	if !loop.HasSymbolFilter {
		t.Error("expected HasSymbolFilter=true")
	}

	// Should detect property calls
	foundTicket := false
	foundStopLoss := false
	for _, pc := range loop.PropertyCalls {
		if pc == "OrderTicket" {
			foundTicket = true
		}
		if pc == "OrderStopLoss" {
			foundStopLoss = true
		}
	}
	if !foundTicket {
		t.Error("expected OrderTicket in PropertyCalls")
	}
	if !foundStopLoss {
		t.Error("expected OrderStopLoss in PropertyCalls")
	}

	// Should detect trailing stop action
	foundTrailing := false
	for _, action := range loop.BodyActions {
		if action == "order_modify:trailing_stop" {
			foundTrailing = true
		}
	}
	if !foundTrailing {
		t.Error("expected order_modify:trailing_stop in BodyActions")
	}

	// Should have trailing stop ModifyRule
	foundTSModify := false
	for _, mr := range intent.Modifies {
		if mr.Kind == "trailing_stop" {
			foundTSModify = true
		}
	}
	if !foundTSModify {
		t.Error("expected trailing_stop ModifyRule")
	}

	// Should have OrderSelect blind spot (info level since recognized)
	foundBS := false
	for _, bs := range intent.BlindSpots {
		if bs.Category == "order_select" && bs.Severity == "信息" {
			foundBS = true
		}
	}
	if !foundBS {
		t.Error("expected order_select blind spot with info severity")
	}
}

func TestAnalyze_RemainingIndicators(t *testing.T) {
	source := `
double alligator = iAlligator(NULL, 0, 13, 8, 8, 5, 5, 3, MODE_SMMA, PRICE_MEDIAN, 1);
double ichimoku = iIchimoku(NULL, 0, 9, 26, 52, 1);
double demarker = iDeMarker(NULL, 0, 14, 1);
double osma = iOsMA(NULL, 0, 12, 26, 9, PRICE_CLOSE, 1);
double rvi = iRVI(NULL, 0, 10, 0, 1);
double force = iForce(NULL, 0, 13, MODE_SMA, PRICE_CLOSE, 1);
double ac = iAC(NULL, 0, 1);
double ao = iAO(NULL, 0, 1);
double bears = iBearsPower(NULL, 0, 13, PRICE_CLOSE, 1);
double bulls = iBullsPower(NULL, 0, 13, PRICE_CLOSE, 1);
void OnTick() {}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	expectedMethods := []string{"alligator", "ichimoku", "demarker", "osma", "rvi", "force", "ac", "ao", "bears_power", "bulls_power"}
	for _, method := range expectedMethods {
		found := false
		for _, ind := range intent.Indicators {
			if ind.SDKMethod == method {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected indicator %s in intent.Indicators", method)
		}
	}

	// Should have indicator_stub blind spots
	stubCount := 0
	for _, bs := range intent.BlindSpots {
		if bs.Category == "indicator_stub" {
			stubCount++
		}
	}
	if stubCount == 0 {
		t.Error("expected at least one indicator_stub blind spot")
	}
}

func TestAnalyze_OnTesterAndOnArray(t *testing.T) {
	source := `
double OnTester() { return 0.0; }
double customMA = iMAOnArray(myArray, 0, 14, 0, MODE_EMA, 1);
void OnTick() {}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Should have OnTester blind spot
	foundTester := false
	for _, bs := range intent.BlindSpots {
		if bs.Category == "on_tester" {
			foundTester = true
		}
	}
	if !foundTester {
		t.Error("expected on_tester blind spot")
	}

	// Should have indicator_on_array blind spot
	foundOnArray := false
	for _, bs := range intent.BlindSpots {
		if bs.Category == "indicator_on_array" {
			foundOnArray = true
		}
	}
	if !foundOnArray {
		t.Error("expected indicator_on_array blind spot")
	}
}

func TestAnalyze_MQL5PositionLoopPattern(t *testing.T) {
	source := `
#include <Trade\Trade.mqh>
CTrade trade;
input int Magic = 100;
void OnTick() {
    for (int i = PositionsTotal() - 1; i >= 0; i--) {
        ulong ticket = PositionGetTicket(i);
        if (PositionGetInteger(POSITION_MAGIC) == Magic) {
            if (PositionGetString(POSITION_SYMBOL) == Symbol()) {
                trade.PositionClose(ticket);
            }
        }
    }
}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if len(intent.PositionLoops) == 0 {
		t.Fatal("expected at least one PositionLoopRule")
	}
	loop := intent.PositionLoops[0]
	if !loop.HasMagicFilter {
		t.Error("expected HasMagicFilter=true")
	}
	if !loop.HasSymbolFilter {
		t.Error("expected HasSymbolFilter=true")
	}

	foundClose := false
	for _, action := range loop.BodyActions {
		if action == "position_close" {
			foundClose = true
		}
	}
	if !foundClose {
		t.Error("expected position_close in BodyActions")
	}

	foundProp := false
	for _, pc := range loop.PropertyCalls {
		if pc == "PositionGetTicket" {
			foundProp = true
		}
	}
	if !foundProp {
		t.Error("expected PositionGetTicket in PropertyCalls")
	}
}

func TestAnalyze_MQL5CTradeExits(t *testing.T) {
	source := `
#include <Trade\Trade.mqh>
CTrade trade;
void OnTick() {
    trade.PositionClosePartial(Symbol(), 0.5);
    trade.PositionCloseBy(ticket1, ticket2);
    trade.OrderDelete(orderTicket);
}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	foundPartial := false
	foundCloseBy := false
	foundDelete := false
	for _, exit := range intent.Exit {
		if exit.Action == "position_close_partial" {
			foundPartial = true
		}
		if exit.Action == "position_close_by" {
			foundCloseBy = true
		}
		if exit.Action == "order_delete" {
			foundDelete = true
		}
	}
	if !foundPartial {
		t.Error("expected position_close_partial exit")
	}
	if !foundCloseBy {
		t.Error("expected position_close_by exit")
	}
	if !foundDelete {
		t.Error("expected order_delete exit")
	}
}

func TestAnalyze_MQL5NewIndicators(t *testing.T) {
	source := `
int amaHandle = iAMA(NULL, 0, 14, 2, 30, 0, PRICE_CLOSE, 1);
int demaHandle = iDEMA(NULL, 0, 14, 0, PRICE_CLOSE, 1);
int temaHandle = iTEMA(NULL, 0, 14, 0, PRICE_CLOSE, 1);
int framaHandle = iFrAMA(NULL, 0, 14, 0, PRICE_CLOSE, 1);
int vidyaHandle = iVIDyA(NULL, 0, 9, 0, 14, 0, PRICE_CLOSE, 1);
int trixHandle = iTriX(NULL, 0, 14, 0, PRICE_CLOSE, 1);
int adxwHandle = iADXWilder(NULL, 0, 14, PRICE_CLOSE, 1);
int chaikinHandle = iChaikin(NULL, 0, 3, 10, MODE_SMA, PRICE_CLOSE, 1);
int volHandle = iVolumes(NULL, 0, PRICE_CLOSE, 1);
void OnTick() {}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	expectedMethods := []string{"ama", "dema", "tema", "frama", "vidya", "trix", "adx_wilder", "chaikin", "volumes"}
	for _, method := range expectedMethods {
		found := false
		for _, ind := range intent.Indicators {
			if ind.SDKMethod == method {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected indicator %s in intent.Indicators", method)
		}
	}
}

func TestAnalyze_MQL5EventsAndNativeOrderSend(t *testing.T) {
	source := `
void OnTrade() {}
void OnTradeTransaction(const MqlTradeTransaction& trans, const MqlTradeRequest& request, const MqlTradeResult& result) {}
void OnBookEvent(const string symbol) {}
void OnTesterInit() {}
void OnTesterDeinit() {}
void OnTesterPass() {}
void OnTick() {
    MqlTradeRequest request;
    MqlTradeResult result;
    OrderSend(request, result);
}
`
	intent, err := Analyze(source)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	foundOnTrade := false
	foundOnTradeTrans := false
	foundOnBookEvent := false
	foundOnTesterInit := false
	foundNativeOrderSend := false
	for _, bs := range intent.BlindSpots {
		switch bs.Category {
		case "on_trade":
			foundOnTrade = true
		case "on_trade_transaction":
			foundOnTradeTrans = true
		case "on_book_event":
			foundOnBookEvent = true
		case "on_tester_init":
			foundOnTesterInit = true
		case "native_ordersend":
			foundNativeOrderSend = true
		}
	}
	if !foundOnTrade {
		t.Error("expected on_trade blind spot")
	}
	if !foundOnTradeTrans {
		t.Error("expected on_trade_transaction blind spot")
	}
	if !foundOnBookEvent {
		t.Error("expected on_book_event blind spot")
	}
	if !foundOnTesterInit {
		t.Error("expected on_tester_init blind spot")
	}
	if !foundNativeOrderSend {
		t.Error("expected native_ordersend blind spot")
	}
}

func TestAnalyze_ThreadSafety(t *testing.T) {
	// Run Analyze concurrently to verify mutex protection
	sources := []string{sampleMQL4EA, sampleMQL5EA, `
extern int X = 1;
void OnTick() { OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0, "", X, 0, clrGreen); }
`}
	done := make(chan bool, len(sources)*3)

	for _, src := range sources {
		for i := 0; i < 3; i++ {
			go func(s string) {
				defer func() { done <- true }()
				intent, err := Analyze(s)
				if err != nil {
					t.Errorf("Analyze failed: %v", err)
					return
				}
				if intent == nil {
					t.Error("Analyze returned nil intent")
				}
			}(src)
		}
	}

	for i := 0; i < len(sources)*3; i++ {
		<-done
	}
}
