package mql2go

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
)

// TestParamPipeline_FloatDefaultParam verifies the full parameter chain:
// MQL source (extern double Lots=0.1) → CompileMQL → cfg.Params (user value 0.5)
// → engine.Run → result.Trades volume == 0.5 (user value, not default 0.1, not 0).
//
// 0.5 is used instead of 0.42 because the MACD Sample EA calls NormalizeDouble(lot, 1)
// in LotsOptimized(), which rounds to 1 decimal place. 0.5 survives this rounding
// and is clearly distinguishable from the default 0.1.
func TestParamPipeline_FloatDefaultParam(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "macd_sample.mq4"))
	if err != nil {
		t.Fatalf("failed to read macd_sample.mq4: %v", err)
	}

	runner, err := CompileMQL(string(source))
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		Params: map[string]string{
			"Lots":           "0.5",
			"MACDOpenLevel":  "3",
			"MACDCloseLevel": "2",
			"MATrendPeriod":  "26",
		},
	}

	engine := backtest.New(cfg, runner, makeE2EBars(80))
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	if len(result.Trades) == 0 {
		t.Skip("MACD Sample produced 0 trades on this data — cannot verify param injection")
	}

	wantVol := decimal.NewFromFloat(0.5)
	for i, tr := range result.Trades {
		if !tr.Volume.Equal(wantVol) {
			t.Errorf("trade[%d] volume=%s, want 0.5 (param pipeline broken: user value not injected)",
				i, tr.Volume)
		}
	}
}

// TestParamPipeline_DefaultValueWhenParamOmitted verifies that when Params
// does NOT include Lots, the default value from MQL source (input double Lots=0.1)
// is used — not 0, not garbage.
//
// Uses a synthetic EA that directly uses Lots in OrderSend (no money management),
// because the MACD Sample's LotsOptimized() calculates lot size dynamically.
func TestParamPipeline_DefaultValueWhenParamOmitted(t *testing.T) {
	// Synthetic EA: directly uses Lots in OrderSend (no money management).
	// input double Lots=0.1 with FLOAT default — exercises findIdent/findType/findInitValue.
	source := `
input double Lots=0.1;
input int MAPeriod=14;
int OnInit(){return 0;}
void OnBar()
{
    double ma = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    if (Close[1] > ma)
        OrderSend(Symbol(), OP_BUY, Lots, Ask, 3, 0, 0, "test", 0, 0, Green);
}
`
	runner, err := CompileMQL(string(source))
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	// Params deliberately omits Lots — default 0.1 from source should be used.
	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		Params: map[string]string{
			"MAPeriod": "14",
		},
	}

	engine := backtest.New(cfg, runner, makeE2EBars(80))
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	if len(result.Trades) == 0 {
		t.Skip("EA produced 0 trades on this data — cannot verify default param")
	}

	// Default value 0.1 should be injected (findInitValue extracts 0.1, not "double").
	for i, tr := range result.Trades {
		if tr.Volume.IsZero() {
			t.Errorf("trade[%d] volume=0, want 0.1 (default not injected — findInitValue bug)", i)
		}
		if !tr.Volume.Equal(decimal.NewFromFloat(0.1)) {
			t.Errorf("trade[%d] volume=%s, want 0.1 (default from source not applied)", i, tr.Volume)
		}
	}
}
