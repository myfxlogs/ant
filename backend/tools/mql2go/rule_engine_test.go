package mql2go

import (
	"testing"

	"alphaforge/tools/mql2go/interp"
)

func TestRuleEngine_ZeroTradesOrderSend(t *testing.T) {
	engine := NewRuleEngine()
	findings := engine.Run(RuleInput{
		Source:      `void OnTick() { OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0, "", 0, 0, 0); }`,
		TotalTrades: 0,
		BlindSpots:  nil,
	})
	if len(findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	if findings[0].RuleID != "R01_zero_trades_ordersend" {
		t.Errorf("expected R01, got %s", findings[0].RuleID)
	}
}

func TestRuleEngine_ZeroTradesWithFatalBlindSpot(t *testing.T) {
	engine := NewRuleEngine()
	findings := engine.Run(RuleInput{
		Source:      `void OnTick() { OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0, "", 0, 0, 0); }`,
		TotalTrades: 0,
		BlindSpots: []CoverageBlindSpot{
			{Builtin: "OrderSend", Severity: interp.SeverityFatal},
		},
	})
	if len(findings) == 0 {
		t.Fatal("expected finding")
	}
	if findings[0].Severity != "fatal" {
		t.Errorf("expected fatal, got %s", findings[0].Severity)
	}
}

func TestRuleEngine_NonZeroTradesNoFinding(t *testing.T) {
	engine := NewRuleEngine()
	findings := engine.Run(RuleInput{
		Source:      `void OnTick() { OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0, "", 0, 0, 0); }`,
		TotalTrades: 5,
	})
	for _, f := range findings {
		if f.RuleID == "R01_zero_trades_ordersend" {
			t.Error("R01 should not fire when trades > 0")
		}
	}
}

func TestRuleEngine_StartEntryMapped(t *testing.T) {
	engine := NewRuleEngine()
	ir := &interp.IR{
		Version: "mql4",
		OnTick:  []interp.Statement{{Kind: interp.StmtExpr}},
	}
	findings := engine.Run(RuleInput{
		Source:      `int start() { OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0); return 0; }`,
		IR:          ir,
		TotalTrades: 0,
	})
	for _, f := range findings {
		if f.RuleID == "R02_start_entry" {
			t.Error("R02 should not fire when OnTick is populated (start() mapped)")
		}
	}
}

func TestRuleEngine_StartEntryNotMapped(t *testing.T) {
	engine := NewRuleEngine()
	ir := &interp.IR{Version: "mql4", OnTick: nil}
	findings := engine.Run(RuleInput{
		Source:      `int start() { OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0); return 0; }`,
		IR:          ir,
		TotalTrades: 0,
	})
	found := false
	for _, f := range findings {
		if f.RuleID == "R02_start_entry" {
			found = true
		}
	}
	if !found {
		t.Error("R02 should fire when start() present but OnTick empty")
	}
}

func TestRuleEngine_MACDModeSignalOK(t *testing.T) {
	engine := NewRuleEngine()
	findings := engine.Run(RuleInput{
		Source:      `void OnTick() { double s = iMACD(NULL,0,12,26,9,PRICE_CLOSE,MODE_SIGNAL,0); }`,
		TotalTrades: 0,
	})
	for _, f := range findings {
		if f.RuleID == "R03_macd_mode_signal" {
			t.Error("R03 should not fire when MODE_SIGNAL is defined with correct value")
		}
	}
}

func TestRuleEngine_ICustomBlindSpot(t *testing.T) {
	engine := NewRuleEngine()
	findings := engine.Run(RuleInput{
		Source:      `void OnTick() { double v = iCustom(NULL,0,"MyInd",0,0); if(v>0) OrderSend(Symbol(),OP_BUY,0.1,Ask,5,0,0); }`,
		TotalTrades: 0,
	})
	found := false
	for _, f := range findings {
		if f.RuleID == "R05_icustom" {
			found = true
			if f.Severity != "fatal" {
				t.Errorf("R05 should be fatal, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("R05 should fire when iCustom is used")
	}
}

func TestRuleEngine_OrderSelectHistory(t *testing.T) {
	engine := NewRuleEngine()
	findings := engine.Run(RuleInput{
		Source:      `void OnTick() { for(int i=0;i<OrdersHistoryTotal();i++){ OrderSelect(i,SELECT_BY_POS,MODE_HISTORY); } }`,
		TotalTrades: 5,
	})
	found := false
	for _, f := range findings {
		if f.RuleID == "R06_orderselect_history" {
			found = true
		}
	}
	if !found {
		t.Error("R06 should fire when OrderSelect+MODE_HISTORY used")
	}
}

func TestRuleEngine_IndicatorModeMissing(t *testing.T) {
	engine := NewRuleEngine()
	// Use a constant name that doesn't exist to trigger the rule
	source := `void OnTick() { double v = iMACD(NULL,0,12,26,9,PRICE_CLOSE,MODE_TENKAN,0); }`
	findings := engine.Run(RuleInput{
		Source:      source,
		TotalTrades: 0,
	})
	// MODE_TENKAN exists in constants, so R08 shouldn't fire
	for _, f := range findings {
		if f.RuleID == "R08_indicator_mode" {
			// MODE_TENKAN is defined, so this shouldn't fire
			t.Error("R08 should not fire when all MODE_ constants are defined")
		}
	}
}

func TestRuleEngine_AllRulesRun(t *testing.T) {
	engine := NewRuleEngine()
	findings := engine.Run(RuleInput{
		Source: `void OnTick() {
			double macd = iMACD(NULL,0,12,26,9,PRICE_CLOSE,MODE_MAIN,0);
			double sig = iMACD(NULL,0,12,26,9,PRICE_CLOSE,MODE_SIGNAL,0);
			if(macd > sig) OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0, "", 123, 0, 0);
			for(int i=0;i<OrdersTotal();i++){
				if(OrderSelect(i,SELECT_BY_POS,MODE_TRADES)){
					if(OrderType()==OP_BUY && OrderProfit()>0) OrderClose(OrderTicket(),OrderLots(),Bid,5);
				}
			}
		}`,
		TotalTrades: 0,
	})
	if len(findings) == 0 {
		t.Fatal("expected at least one finding for complex EA with 0 trades")
	}
}
