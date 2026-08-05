package mql2go

import (
	"testing"

	"alphaforge/tools/mql2go/interp"
)

func TestFailureSignature_Dedup(t *testing.T) {
	source := `void OnTick() { OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0); }`
	findings := []DiagnosticFinding{
		{RuleID: "R01_zero_trades_ordersend", Severity: "warning"},
	}
	blindSpots := []CoverageBlindSpot{
		{Builtin: "iCustom", Severity: "fatal"},
	}
	runtimeBlinds := []RuntimeBlindSpot{
		{Builtin: "iADX:MODE_PLUSDI", Severity: "warning", Count: 3},
	}

	sig1 := BuildFailureSignature(source, findings, blindSpots, runtimeBlinds, 0)
	sig2 := BuildFailureSignature(source, findings, blindSpots, runtimeBlinds, 0)

	if sig1.Hash != sig2.Hash {
		t.Error("same input should produce same signature hash")
	}
}

func TestFailureSignature_DifferentSource(t *testing.T) {
	source1 := `void OnTick() { OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0); }`
	source2 := `void OnTick() { OrderSend(Symbol(), OP_SELL, 0.1, Bid, 5, 0, 0); }`
	findings := []DiagnosticFinding{
		{RuleID: "R01_zero_trades_ordersend", Severity: "warning"},
	}

	sig1 := BuildFailureSignature(source1, findings, nil, nil, 0)
	sig2 := BuildFailureSignature(source2, findings, nil, nil, 0)

	if sig1.Hash == sig2.Hash {
		t.Error("different source should produce different signature hash")
	}
}

func TestFailureSignature_DifferentRules(t *testing.T) {
	source := `void OnTick() { OrderSend(Symbol(), OP_BUY, 0.1, Ask, 5, 0, 0); }`

	sig1 := BuildFailureSignature(source, []DiagnosticFinding{{RuleID: "R01"}}, nil, nil, 0)
	sig2 := BuildFailureSignature(source, []DiagnosticFinding{{RuleID: "R05"}}, nil, nil, 0)

	if sig1.Hash == sig2.Hash {
		t.Error("different rule IDs should produce different signature hash")
	}
}

func TestBuildReproPackage(t *testing.T) {
	source := `void OnTick() { double v = iCustom(NULL,0,"MyInd",0,0); if(v>0) OrderSend(Symbol(),OP_BUY,0.1,Ask,5,0,0); }`
	findings := []DiagnosticFinding{
		{RuleID: "R05_icustom", Severity: "fatal", Title: "iCustom not supported"},
	}
	blindSpots := []CoverageBlindSpot{
		{Builtin: "iCustom", Severity: interp.SeverityFatal, Source: "static"},
	}

	pkg := BuildReproPackage(source, findings, blindSpots, nil, 0, "EURUSD", "H1")

	if pkg.Signature.Hash == "" {
		t.Error("signature hash should not be empty")
	}
	if pkg.Symbol != "EURUSD" {
		t.Errorf("expected EURUSD, got %s", pkg.Symbol)
	}
	if pkg.Timeframe != "H1" {
		t.Errorf("expected H1, got %s", pkg.Timeframe)
	}
	if len(pkg.SourcePreview) > 500 {
		t.Error("source preview should be truncated to 500 chars")
	}
	if len(pkg.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(pkg.Findings))
	}
}

func TestBuildReproPackage_LongSource(t *testing.T) {
	// Generate source > 500 chars
	source := ""
	for i := 0; i < 100; i++ {
		source += "double x = iMA(NULL,0,14,0,MODE_EMA,PRICE_CLOSE,0); "
	}

	pkg := BuildReproPackage(source, nil, nil, nil, 5, "GBPUSD", "M15")

	if len(pkg.SourcePreview) > 503 {
		t.Errorf("preview should be truncated, got %d chars", len(pkg.SourcePreview))
	}
	if !endsWith(pkg.SourcePreview, "...") {
		t.Error("truncated preview should end with ...")
	}
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
