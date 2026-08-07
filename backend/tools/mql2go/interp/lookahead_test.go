package interp

import (
	"testing"
)

func TestDetectLookahead_CleanSeries(t *testing.T) {
	t.Parallel()
	ir := &IR{
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprCall,
				Name: "OrderSend",
				Args: []Expr{
					{Kind: ExprCall, Name: "iMA", Args: []Expr{
						litStr("EURUSD"), litInt(0), litInt(14), litInt(0), litInt(0), litInt(0), litInt(1),
					}},
				},
			}},
		},
	}
	violations := DetectLookahead(ir)
	if len(violations) != 0 {
		t.Fatalf("clean shift=1 should have zero violations, got: %+v", violations)
	}
}

func TestDetectLookahead_NegativeSeriesShift(t *testing.T) {
	t.Parallel()
	ir := &IR{
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprSubscript,
				Name: "Close",
				Index: &Expr{Kind: ExprUnary, Op: "-", Args: []Expr{litInt(1)}},
			}},
		},
	}
	violations := DetectLookahead(ir)
	if len(violations) != 1 {
		t.Fatalf("Close[-1] should produce 1 violation, got %d: %+v", len(violations), violations)
	}
	if violations[0].Severity != SeverityFatal {
		t.Fatalf("expected fatal severity, got %s", violations[0].Severity)
	}
	if violations[0].ShiftVal != -1 {
		t.Fatalf("expected shift=-1, got %d", violations[0].ShiftVal)
	}
}

func TestDetectLookahead_NegativeIndicatorShift(t *testing.T) {
	t.Parallel()
	ir := &IR{
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprCall,
				Name: "iMA",
				Args: []Expr{
					litStr("EURUSD"), litInt(0), litInt(14), litInt(0), litInt(0), litInt(0),
					{Kind: ExprUnary, Op: "-", Args: []Expr{litInt(2)}},
				},
			}},
		},
	}
	violations := DetectLookahead(ir)
	if len(violations) != 1 {
		t.Fatalf("iMA(..., -2) should produce 1 violation, got %d: %+v", len(violations), violations)
	}
	if violations[0].Severity != SeverityFatal {
		t.Fatalf("expected fatal severity, got %s", violations[0].Severity)
	}
	if violations[0].ShiftVal != -2 {
		t.Fatalf("expected shift=-2, got %d", violations[0].ShiftVal)
	}
}

func TestDetectLookahead_ZeroShiftOK(t *testing.T) {
	t.Parallel()
	ir := &IR{
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprSubscript,
				Name: "Close",
				Index: &Expr{Kind: ExprLiteral, Val: intVal(0)},
			}},
		},
	}
	violations := DetectLookahead(ir)
	if len(violations) != 0 {
		t.Fatalf("Close[0] should have zero violations, got: %+v", violations)
	}
}

func TestDetectLookahead_PositiveShiftOK(t *testing.T) {
	t.Parallel()
	ir := &IR{
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprCall,
				Name: "iRSI",
				Args: []Expr{
					litStr("EURUSD"), litInt(0), litInt(14), litInt(0), litInt(3),
				},
			}},
		},
	}
	violations := DetectLookahead(ir)
	if len(violations) != 0 {
		t.Fatalf("iRSI(..., 3) should have zero violations, got: %+v", violations)
	}
}

func TestDetectLookahead_VariableShiftWarning(t *testing.T) {
	t.Parallel()
	ir := &IR{
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprCall,
				Name: "iMA",
				Args: []Expr{
					litStr("EURUSD"), litInt(0), litInt(14), litInt(0), litInt(0), litInt(0),
					{Kind: ExprVar, Name: "myShift"},
				},
			}},
		},
	}
	violations := DetectLookahead(ir)
	if len(violations) != 1 {
		t.Fatalf("variable shift should produce 1 violation, got %d: %+v", len(violations), violations)
	}
	if violations[0].Severity != SeverityWarning {
		t.Fatalf("expected warning severity for variable shift, got %s", violations[0].Severity)
	}
	if violations[0].IsLiteral {
		t.Fatal("variable shift should not be literal")
	}
}

func TestDetectLookahead_SubtractionShiftWarning(t *testing.T) {
	t.Parallel()
	ir := &IR{
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprSubscript,
				Name: "High",
				Index: &Expr{
					Kind: ExprBinary, Op: "-",
					Args: []Expr{{Kind: ExprVar, Name: "x"}, litInt(2)},
				},
			}},
		},
	}
	violations := DetectLookahead(ir)
	if len(violations) != 1 {
		t.Fatalf("x-2 shift should produce 1 violation, got %d: %+v", len(violations), violations)
	}
	if violations[0].Severity != SeverityWarning {
		t.Fatalf("expected warning severity for non-constant subtraction, got %s", violations[0].Severity)
	}
}

func TestDetectLookahead_ConstantSubtractionEvaluated(t *testing.T) {
	t.Parallel()
	ir := &IR{
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprSubscript,
				Name: "Low",
				Index: &Expr{
					Kind: ExprBinary, Op: "-",
					Args: []Expr{litInt(1), litInt(3)},
				},
			}},
		},
	}
	violations := DetectLookahead(ir)
	if len(violations) != 1 {
		t.Fatalf("1-3=-2 shift should produce 1 violation, got %d: %+v", len(violations), violations)
	}
	if violations[0].Severity != SeverityFatal {
		t.Fatalf("expected fatal for constant -2, got %s", violations[0].Severity)
	}
	if violations[0].ShiftVal != -2 {
		t.Fatalf("expected shift=-2, got %d", violations[0].ShiftVal)
	}
}

func TestDetectLookahead_Deduplicate(t *testing.T) {
	t.Parallel()
	ir := &IR{
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprSubscript,
				Name: "Close",
				Index: &Expr{Kind: ExprUnary, Op: "-", Args: []Expr{litInt(1)}},
			}},
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprSubscript,
				Name: "Close",
				Index: &Expr{Kind: ExprUnary, Op: "-", Args: []Expr{litInt(1)}},
			}},
		},
	}
	violations := DetectLookahead(ir)
	if len(violations) != 1 {
		t.Fatalf("duplicate Close[-1] should deduplicate to 1, got %d", len(violations))
	}
}

func TestDetectLookahead_MultipleViolations(t *testing.T) {
	t.Parallel()
	ir := &IR{
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprSubscript,
				Name: "Close",
				Index: &Expr{Kind: ExprUnary, Op: "-", Args: []Expr{litInt(1)}},
			}},
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprCall,
				Name: "iRSI",
				Args: []Expr{
					litStr("EURUSD"), litInt(0), litInt(14), litInt(0),
					{Kind: ExprUnary, Op: "-", Args: []Expr{litInt(3)}},
				},
			}},
		},
	}
	violations := DetectLookahead(ir)
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %d: %+v", len(violations), violations)
	}
}

func TestDetectLookahead_NonSeriesSubscriptIgnored(t *testing.T) {
	t.Parallel()
	ir := &IR{
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{
				Kind: ExprSubscript,
				Name: "myArray",
				Index: &Expr{Kind: ExprUnary, Op: "-", Args: []Expr{litInt(1)}},
			}},
		},
	}
	violations := DetectLookahead(ir)
	if len(violations) != 0 {
		t.Fatalf("non-series array subscript should be ignored, got: %+v", violations)
	}
}

func TestDetectLookahead_InUserFunction(t *testing.T) {
	t.Parallel()
	ir := &IR{
		Funcs: map[string]*FuncDef{
			"CheckForOpen": {
				Name: "CheckForOpen",
				Body: []Statement{
					{Kind: StmtExpr, Expr: &Expr{
						Kind: ExprSubscript,
						Name: "Close",
						Index: &Expr{Kind: ExprUnary, Op: "-", Args: []Expr{litInt(1)}},
					}},
				},
			},
		},
	}
	violations := DetectLookahead(ir)
	if len(violations) != 1 {
		t.Fatalf("lookahead in user function should be detected, got %d: %+v", len(violations), violations)
	}
}

// ── helpers ──────────────────────────────────────────────────────────

func litInt(v int32) Expr {
	return Expr{Kind: ExprLiteral, Val: intVal(v)}
}

func litStr(s string) Expr {
	return Expr{Kind: ExprLiteral, Val: strVal(s)}
}

func intVal(v int32) Value {
	return Value{Kind: ValInt, Int: v}
}

func strVal(s string) Value {
	return Value{Kind: ValString, Str: s}
}
