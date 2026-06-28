package interp

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestSerializeDeserializeIR_RoundTrip(t *testing.T) {
	original := &IR{
		Version: "mql4",
		Globals: []GlobalVar{
			{Name: "lots", Type: "double", InitVal: &Expr{Kind: ExprLiteral, Val: DecimalVal(decimal.NewFromFloat(0.1))}},
			{Name: "maxOrders", Type: "int", InitVal: &Expr{Kind: ExprLiteral, Val: IntVal(5)}},
		},
		Params: []ParamDecl{
			{Name: "RiskPercent", Type: "double", Default: &Expr{Kind: ExprLiteral, Val: DecimalVal(decimal.NewFromFloat(2.0))}},
		},
		Funcs: map[string]*FuncDef{
			"CalcLot": {
				Name: "CalcLot",
				Params: []ParamDecl{
					{Name: "pct", Type: "double"},
					{Name: "bal", Type: "double"},
				},
				Body: []Statement{
					{
						Kind: StmtReturn,
						Expr: &Expr{
							Kind: ExprBinary, Op: "*",
							Args: []Expr{
								{Kind: ExprVar, Name: "pct"},
								{Kind: ExprVar, Name: "bal"},
							},
						},
					},
				},
			},
		},
		Enums: map[string]int32{
			"MODE_AUTO": 1,
			"MODE_MANUAL": 0,
		},
		OnBar: []Statement{
			{
				Kind: StmtFor,
				Init: &Statement{
					Kind: StmtExpr,
					Expr: &Expr{Kind: ExprAssignment, Name: "i", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(0)}}},
				},
				Cond: &Expr{Kind: ExprBinary, Op: "<", Args: []Expr{{Kind: ExprVar, Name: "i"}, {Kind: ExprLiteral, Val: IntVal(10)}}},
				Update: &Statement{Kind: StmtExpr, Expr: &Expr{Kind: ExprUpdate, Name: "i", Op: "++"}},
				Body: []Statement{
					{Kind: StmtBreak},
					{Kind: StmtContinue},
					{
						Kind: StmtIf,
						Cond: &Expr{Kind: ExprBinary, Op: "==", Args: []Expr{{Kind: ExprVar, Name: "i"}, {Kind: ExprLiteral, Val: IntVal(3)}}},
						Body: []Statement{{Kind: StmtExpr, Expr: &Expr{Kind: ExprCompoundAssign, Op: "+=", Name: "x", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(1)}}}}},
					},
				},
			},
			{
				Kind: StmtDoWhile,
				Cond: &Expr{Kind: ExprVar, Name: "flag"},
				Body: []Statement{{Kind: StmtExpr, Expr: &Expr{Kind: ExprCall, Name: "CalcLot", Args: []Expr{{Kind: ExprLiteral, Val: DecimalVal(decimal.NewFromFloat(0.02))}}}}},
			},
			{
				Kind: StmtSwitch,
				Expr: &Expr{Kind: ExprVar, Name: "mode"},
				Cases: []SwitchCase{
					{Expr: &Expr{Kind: ExprConst, Name: "MODE_AUTO"}, Body: []Statement{{Kind: StmtBreak}}},
					{Expr: nil, Body: []Statement{{Kind: StmtExpr, Expr: &Expr{Kind: ExprCall, Name: "Print"}}}},
				},
			},
		},
	}

	// Serialize
	data := SerializeIR(original)
	if len(data) == 0 {
		t.Fatal("SerializeIR returned empty data")
	}

	// Deserialize
	restored := DeserializeIR(data)
	if restored == nil {
		t.Fatal("DeserializeIR returned nil")
	}

	// Verify version
	if restored.Version != "mql4" {
		t.Errorf("Version = %s, want mql4", restored.Version)
	}

	// Verify globals
	if len(restored.Globals) != 2 {
		t.Fatalf("Globals len = %d, want 2", len(restored.Globals))
	}
	if restored.Globals[0].Name != "lots" || restored.Globals[0].Type != "double" {
		t.Errorf("Global[0] = %+v, want lots/double", restored.Globals[0])
	}
	if restored.Globals[0].InitVal == nil || restored.Globals[0].InitVal.Kind != ExprLiteral {
		t.Errorf("Global[0] InitVal should be ExprLiteral")
	}
	if !restored.Globals[0].InitVal.Val.ToDecimal().Equal(decimal.NewFromFloat(0.1)) {
		t.Errorf("Global[0] InitVal = %s, want 0.1", restored.Globals[0].InitVal.Val.ToDecimal())
	}

	// Verify params
	if len(restored.Params) != 1 {
		t.Fatalf("Params len = %d, want 1", len(restored.Params))
	}
	if restored.Params[0].Name != "RiskPercent" {
		t.Errorf("Param[0] Name = %s, want RiskPercent", restored.Params[0].Name)
	}

	// Verify funcs
	fn, ok := restored.Funcs["CalcLot"]
	if !ok {
		t.Fatal("CalcLot function not found after round-trip")
	}
	if len(fn.Params) != 2 {
		t.Errorf("CalcLot params = %d, want 2", len(fn.Params))
	}
	if fn.Params[0].Name != "pct" {
		t.Errorf("CalcLot param[0] = %s, want pct", fn.Params[0].Name)
	}
	if len(fn.Body) != 1 || fn.Body[0].Kind != StmtReturn {
		t.Errorf("CalcLot body should have 1 return statement")
	}

	// Verify enums
	if v, ok := restored.Enums["MODE_AUTO"]; !ok || v != 1 {
		t.Errorf("Enum MODE_AUTO = %d, ok=%v, want 1", v, ok)
	}
	if v, ok := restored.Enums["MODE_MANUAL"]; !ok || v != 0 {
		t.Errorf("Enum MODE_MANUAL = %d, ok=%v, want 0", v, ok)
	}

	// Verify OnBar statements
	if len(restored.OnBar) != 3 {
		t.Fatalf("OnBar len = %d, want 3", len(restored.OnBar))
	}

	// Check for loop
	forStmt := restored.OnBar[0]
	if forStmt.Kind != StmtFor {
		t.Errorf("OnBar[0] kind = %v, want StmtFor", forStmt.Kind)
	}
	if forStmt.Init == nil || forStmt.Cond == nil || forStmt.Update == nil {
		t.Error("For loop missing Init/Cond/Update")
	}
	// Body should have break, continue, and if
	if len(forStmt.Body) != 3 {
		t.Fatalf("For body len = %d, want 3", len(forStmt.Body))
	}
	if forStmt.Body[0].Kind != StmtBreak {
		t.Errorf("For body[0] = %v, want StmtBreak", forStmt.Body[0].Kind)
	}
	if forStmt.Body[1].Kind != StmtContinue {
		t.Errorf("For body[1] = %v, want StmtContinue", forStmt.Body[1].Kind)
	}
	if forStmt.Body[2].Kind != StmtIf {
		t.Errorf("For body[2] = %v, want StmtIf", forStmt.Body[2].Kind)
	}

	// Check do-while
	dwStmt := restored.OnBar[1]
	if dwStmt.Kind != StmtDoWhile {
		t.Errorf("OnBar[1] kind = %v, want StmtDoWhile", dwStmt.Kind)
	}

	// Check switch
	swStmt := restored.OnBar[2]
	if swStmt.Kind != StmtSwitch {
		t.Errorf("OnBar[2] kind = %v, want StmtSwitch", swStmt.Kind)
	}
	if len(swStmt.Cases) != 2 {
		t.Errorf("Switch cases = %d, want 2", len(swStmt.Cases))
	}
}

func TestSerializeDeserializeIR_Empty(t *testing.T) {
	// Empty data should return nil
	if ir := DeserializeIR(nil); ir != nil {
		t.Error("DeserializeIR(nil) should return nil")
	}
	if ir := DeserializeIR([]byte{}); ir != nil {
		t.Error("DeserializeIR([]byte{}) should return nil")
	}
}

func TestSerializeDeserializeIR_Minimal(t *testing.T) {
	original := &IR{
		Version: "mql5",
	}

	data := SerializeIR(original)
	if len(data) == 0 {
		t.Fatal("SerializeIR returned empty data for minimal IR")
	}

	restored := DeserializeIR(data)
	if restored == nil {
		t.Fatal("DeserializeIR returned nil for minimal IR")
	}
	if restored.Version != "mql5" {
		t.Errorf("Version = %s, want mql5", restored.Version)
	}
}

func TestSerializeDeserializeIR_ClassInstance(t *testing.T) {
	original := &IR{
		Version: "mql5",
		Globals: []GlobalVar{
			{Name: "trade", Type: "CTrade"},
		},
	}

	data := SerializeIR(original)
	restored := DeserializeIR(data)
	if restored == nil {
		t.Fatal("DeserializeIR returned nil")
	}
	if len(restored.Globals) != 1 {
		t.Fatalf("Globals len = %d, want 1", len(restored.Globals))
	}
	if restored.Globals[0].Name != "trade" || restored.Globals[0].Type != "CTrade" {
		t.Errorf("Global = %+v, want trade/CTrade", restored.Globals[0])
	}
}
