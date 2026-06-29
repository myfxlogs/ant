package interp

import (
	"testing"

	"github.com/shopspring/decimal"
)

// helper: build a simple ExprCall
func callExpr(name string, args ...Expr) Expr {
	return Expr{Kind: ExprCall, Name: name, Args: args}
}

// helper: build an ExprField method call (obj.method(args...))
func methodExpr(objName, method string, args ...Expr) Expr {
	obj := Expr{Kind: ExprVar, Name: objName}
	all := append([]Expr{obj}, args...)
	return Expr{Kind: ExprField, Name: method, Args: all}
}

func TestAnalyze_MQL4_FullSupport(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("iMA",
				Expr{Kind: ExprLiteral, Val: IntVal(14)},
				Expr{Kind: ExprLiteral, Val: IntVal(1)},
				Expr{Kind: ExprLiteral, Val: IntVal(1)},
			))},
			{Kind: StmtExpr, Expr: ptr(callExpr("OrderSend",
				Expr{Kind: ExprVar, Name: "Symbol"},
				Expr{Kind: ExprConst, Name: "OP_BUY"},
				Expr{Kind: ExprLiteral, Val: DecimalVal0(0.1)},
				Expr{Kind: ExprVar, Name: "Ask"},
			))},
		},
	}
	rep := Analyze(ir)
	if rep.Coverage != 1.0 {
		t.Errorf("Coverage = %.2f, want 1.0", rep.Coverage)
	}
	if len(rep.BlindSpots) != 0 {
		t.Errorf("BlindSpots = %v, want empty", rep.BlindSpots)
	}
	if rep.ExecKind != "on_bar" {
		t.Errorf("ExecKind = %q, want on_bar", rep.ExecKind)
	}
	if len(rep.Indicators) != 1 || rep.Indicators[0] != "iMA" {
		t.Errorf("Indicators = %v, want [iMA]", rep.Indicators)
	}
}

func TestAnalyze_MQL5_CTradeAndPosition(t *testing.T) {
	ir := &IR{
		Version: "mql5",
		Globals: []GlobalVar{
			{Name: "trade", Type: "CTrade"},
		},
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(methodExpr("trade", "Buy",
				Expr{Kind: ExprLiteral, Val: DecimalVal0(0.1)},
			))},
			{Kind: StmtExpr, Expr: ptr(callExpr("PositionGetDouble",
				Expr{Kind: ExprConst, Name: "POSITION_VOLUME"},
			))},
		},
	}
	rep := Analyze(ir)
	if rep.Coverage != 1.0 {
		t.Errorf("Coverage = %.2f, want 1.0 (CTrade.Buy + PositionGetDouble should be supported)", rep.Coverage)
	}
	if len(rep.BlindSpots) != 0 {
		t.Errorf("BlindSpots = %v, want empty", rep.BlindSpots)
	}
}

func TestAnalyze_MQL5_UnknownCTradeMethod(t *testing.T) {
	ir := &IR{
		Version: "mql5",
		Globals: []GlobalVar{
			{Name: "trade", Type: "CTrade"},
		},
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(methodExpr("trade", "XxxUnknown",
				Expr{Kind: ExprLiteral, Val: IntVal(1)},
			))},
		},
	}
	rep := Analyze(ir)
	if rep.Coverage != 0.0 {
		t.Errorf("Coverage = %.2f, want 0.0", rep.Coverage)
	}
	if len(rep.BlindSpots) != 1 {
		t.Fatalf("BlindSpots = %v, want 1 entry", rep.BlindSpots)
	}
	bs := rep.BlindSpots[0]
	if bs.Builtin != "XxxUnknown" {
		t.Errorf("BlindSpot Builtin = %q, want XxxUnknown", bs.Builtin)
	}
	if bs.Severity != "致命" {
		t.Errorf("BlindSpot Severity = %q, want 致命", bs.Severity)
	}
}

func TestAnalyze_MissingBuiltin_iCustom(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("iCustom",
				Expr{Kind: ExprVar, Name: "Symbol"},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
				Expr{Kind: ExprLiteral, Val: IntVal(14)},
			))},
		},
	}
	rep := Analyze(ir)
	if rep.Coverage != 0.0 {
		t.Errorf("Coverage = %.2f, want 0.0", rep.Coverage)
	}
	if len(rep.BlindSpots) != 1 {
		t.Fatalf("BlindSpots = %v, want 1 entry", rep.BlindSpots)
	}
	if rep.BlindSpots[0].Builtin != "iCustom" {
		t.Errorf("BlindSpot = %q, want iCustom", rep.BlindSpots[0].Builtin)
	}
	if rep.BlindSpots[0].Severity != "致命" {
		t.Errorf("Severity = %q, want 致命", rep.BlindSpots[0].Severity)
	}
}

func TestAnalyze_PreviouslyStubIndicator_NowImplemented(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("iAlligator",
				Expr{Kind: ExprLiteral, Val: IntVal(13)},
				Expr{Kind: ExprLiteral, Val: IntVal(8)},
				Expr{Kind: ExprLiteral, Val: IntVal(8)},
				Expr{Kind: ExprLiteral, Val: IntVal(5)},
				Expr{Kind: ExprLiteral, Val: IntVal(5)},
				Expr{Kind: ExprLiteral, Val: IntVal(3)},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
			))},
		},
	}
	rep := Analyze(ir)
	// iAlligator is now fully implemented — Coverage should be 1.0
	if rep.Coverage != 1.0 {
		t.Errorf("Coverage = %.2f, want 1.0 (iAlligator is implemented)", rep.Coverage)
	}
	if len(rep.BlindSpots) != 0 {
		t.Errorf("BlindSpots = %v, want empty (iAlligator is implemented)", rep.BlindSpots)
	}
	// iAlligator is no longer a stub
	if IsStubIndicator("iAlligator") {
		t.Error("IsStubIndicator(iAlligator) = true, want false (now fully implemented)")
	}
}

func TestAnalyze_UserFuncNotBlindSpot(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		Funcs: map[string]*FuncDef{
			"MyHelper": {
				Name: "MyHelper",
				Params: []ParamDecl{{Name: "x", Type: "double"}},
				Body: []Statement{
					{Kind: StmtReturn, Expr: ptr(Expr{Kind: ExprVar, Name: "x"})},
				},
			},
		},
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("MyHelper",
				Expr{Kind: ExprLiteral, Val: IntVal(42)},
			))},
		},
	}
	rep := Analyze(ir)
	if rep.Coverage != 1.0 {
		t.Errorf("Coverage = %.2f, want 1.0 (user func not a blind spot)", rep.Coverage)
	}
	if len(rep.BlindSpots) != 0 {
		t.Errorf("BlindSpots = %v, want empty", rep.BlindSpots)
	}
}

func TestAnalyze_MixedSupportedAndBlind(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("iMA",
				Expr{Kind: ExprLiteral, Val: IntVal(14)},
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
				Expr{Kind: ExprLiteral, Val: IntVal(1)},
			))},
			{Kind: StmtExpr, Expr: ptr(callExpr("iCustom",
				Expr{Kind: ExprLiteral, Val: IntVal(0)},
			))},
			{Kind: StmtExpr, Expr: ptr(callExpr("OrderSend",
				Expr{Kind: ExprVar, Name: "Symbol"},
				Expr{Kind: ExprConst, Name: "OP_BUY"},
				Expr{Kind: ExprLiteral, Val: DecimalVal0(0.1)},
				Expr{Kind: ExprVar, Name: "Ask"},
			))},
		},
	}
	rep := Analyze(ir)
	if rep.TotalCalls != 3 {
		t.Errorf("TotalCalls = %d, want 3", rep.TotalCalls)
	}
	if rep.SupportedCalls != 2 {
		t.Errorf("SupportedCalls = %d, want 2", rep.SupportedCalls)
	}
	want := 2.0 / 3.0
	if rep.Coverage < want-0.01 || rep.Coverage > want+0.01 {
		t.Errorf("Coverage = %.4f, want ~%.4f", rep.Coverage, want)
	}
	if len(rep.BlindSpots) != 1 || rep.BlindSpots[0].Builtin != "iCustom" {
		t.Errorf("BlindSpots = %v, want [iCustom]", rep.BlindSpots)
	}
}

func TestAnalyze_ExecKind_OnTick(t *testing.T) {
	ir := &IR{
		Version: "mql5",
		OnTick: []Statement{
			{Kind: StmtExpr, Expr: ptr(callExpr("AccountBalance"))},
		},
	}
	rep := Analyze(ir)
	if rep.ExecKind != "on_tick" {
		t.Errorf("ExecKind = %q, want on_tick", rep.ExecKind)
	}
}

// ── helpers ─────────────────────────────────────────────────────────

func ptr(e Expr) *Expr { return &e }

// DecimalVal0 creates a decimal from a float for test convenience.
func DecimalVal0(f float64) Value {
	return DecimalVal(decimal.NewFromFloat(f))
}
