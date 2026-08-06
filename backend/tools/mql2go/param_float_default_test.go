package mql2go

import "testing"

// TestParam_FloatDefault_NameExtracted guards against a tree-sitter grammar
// quirk: an input/extern parameter whose default value is a FLOAT literal had
// its name mislabeled as the type name. Concretely,
//   input double Lots = 0.1;
// parsed with "double" (the type) as the extracted parameter name, leaving the
// real "Lots" unmapped. At backtest time the Lots slot was never injected →
// OrderSend ran with volume 0 → zero-volume trades (see the xianhua.chan
// incident: 9 consecutive backtests, 11/11 trades at volume 0, all reported
// SUCCEEDED with no diagnostic).
//
// This must never regress: every input name must resolve to its own global slot.
func TestParam_FloatDefault_NameExtracted(t *testing.T) {
	src := `
input double TakeProfit=50;
input double Lots=0.1;
input double TrailingStop=30;
input double MACDOpenLevel=3;
input double MACDCloseLevel=2;
input int MATrendPeriod=26;
int OnInit(){return 0;}
void OnTick(){}
`
	ir, err := CompileToIR(src)
	if err != nil {
		t.Fatalf("CompileToIR: %v", err)
	}
	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST: %v", err)
	}
	for _, want := range []string{"Lots", "TakeProfit", "TrailingStop",
		"MACDOpenLevel", "MACDCloseLevel", "MATrendPeriod"} {
		if _, ok := bc.GlobalSlots[want]; !ok {
			t.Errorf("input param %q missing from GlobalSlots (map=%v) — "+
				"float-default name extraction regressed", want, bc.GlobalSlots)
		}
	}
	if _, bad := bc.GlobalSlots["double"]; bad {
		t.Errorf("type name %q leaked into GlobalSlots as a param name "+
			"(map=%v) — findIdent returned the type instead of the variable", "double", bc.GlobalSlots)
	}
}
