package mql2go

import (
	"testing"

	"alphaforge/tools/mql2go/interp"
)

// TestFindInitValue_FloatDefault verifies Lots default (0.1) is extracted as a number,
// not the "double" type identifier. Regression for the 3rd same-source tree-sitter
// float-default quirk (findIdent→Name, findType→Type, findInitValue→Default).
// Before fix: findInitValue returned "double" identifier → def=0 → Lots=0 → volume=0.
func TestFindInitValue_FloatDefault(t *testing.T) {
	src := `
input double Lots=0.1;
input int MATrendPeriod=26;
int OnInit(){return 0;}
void OnTick(){}
`
	runner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	for _, p := range runner.bc.Params {
		if p.Name == "Lots" {
			got := interp.EvalExprLiteral(p.Default)
			if got != "0.1" {
				t.Errorf("Lots default = %q, want %q (findInitValue float-default bug)", got, "0.1")
			}
		}
	}
}
