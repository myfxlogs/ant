package mql2go

import "testing"

// TestFindType_FloatDefault guards the bug where findType returned "" for input
// params with FLOAT default values (e.g. "input double Lots=0.1"). The empty Type
// caused injectParams to skip Lots → OrderSend volume=0 (xianhua incident).
// findIdent was fixed earlier but findType (same tree-sitter quirk) was missed.
func TestFindType_FloatDefault(t *testing.T) {
	src := `
input double TakeProfit=50;
input double Lots=0.1;
input double TrailingStop=30;
input int MATrendPeriod=26;
int OnInit(){return 0;}
void OnTick(){}
`
	runner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	lotsFound := false
	for _, p := range runner.bc.Params {
		if p.Type == "" {
			t.Errorf("PARAM %q has empty Type — findType float-default bug regressed", p.Name)
		}
		if p.Name == "Lots" {
			lotsFound = true
			if p.Type != "double" {
				t.Errorf("Lots Type=%q, want %q (float-default findType)", p.Type, "double")
			}
		}
	}
	if !lotsFound {
		t.Error("Lots param not found in bc.Params")
	}
}
