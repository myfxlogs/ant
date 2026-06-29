package interp

import "testing"

func TestAnalyze_PermanentBlindSpots(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("FileOpen",
				Expr{Kind: ExprLiteral, Val: StringVal("test.csv")},
				Expr{Kind: ExprLiteral, Val: IntVal(1)},
			))},
			{Kind: StmtExpr, Expr: ptr(callExpr("ObjectCreate",
				Expr{Kind: ExprLiteral, Val: StringVal("line1")},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
			))},
		},
	}
	rep := Analyze(ir)
	if rep.Coverage != 0 {
		t.Errorf("Coverage = %.2f, want 0 (permanent blind spots not supported)", rep.Coverage)
	}
	found := false
	for _, bs := range rep.BlindSpots {
		if bs.Severity == SeverityInfo {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected at least one blind spot with severity %q, got: %v", SeverityInfo, rep.BlindSpots)
	}
}

func TestAnalyze_PermanentBlindSpot_FileOpen(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("FileOpen",
				Expr{Kind: ExprLiteral, Val: StringVal("log.txt")},
				Expr{Kind: ExprLiteral, Val: IntVal(1)},
			))},
		},
	}
	rep := Analyze(ir)
	for _, bs := range rep.BlindSpots {
		if bs.Builtin == "FileOpen" && bs.Severity != SeverityInfo {
			t.Errorf("FileOpen severity = %q, want %q", bs.Severity, SeverityInfo)
		}
	}
}

func TestAnalyze_PermanentBlindSpot_ObjectCreate(t *testing.T) {
	ir := &IR{
		Version: "mql5",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("ObjectCreate",
				Expr{Kind: ExprLiteral, Val: StringVal("trendline")},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
			))},
		},
	}
	rep := Analyze(ir)
	for _, bs := range rep.BlindSpots {
		if bs.Builtin == "ObjectCreate" && bs.Severity != SeverityInfo {
			t.Errorf("ObjectCreate severity = %q, want %q", bs.Severity, SeverityInfo)
		}
	}
}
