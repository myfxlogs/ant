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
		if bs.Severity == "永久盲区" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected at least one blind spot with severity '永久盲区', got: %v", rep.BlindSpots)
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
		if bs.Builtin == "FileOpen" && bs.Severity != "永久盲区" {
			t.Errorf("FileOpen severity = %q, want '永久盲区'", bs.Severity)
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
		if bs.Builtin == "ObjectCreate" && bs.Severity != "永久盲区" {
			t.Errorf("ObjectCreate severity = %q, want '永久盲区'", bs.Severity)
		}
	}
}
