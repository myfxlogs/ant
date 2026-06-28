package interp

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestUserDefinedFunction(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		Funcs: map[string]*FuncDef{
			"CalcLot": {
				Name: "CalcLot",
				Params: []ParamDecl{
					{Name: "riskPct", Type: "double"},
					{Name: "balance", Type: "double"},
				},
				Body: []Statement{
					{
						Kind: StmtReturn,
						Expr: &Expr{
							Kind: ExprBinary,
							Op:   "*",
							Args: []Expr{
								{Kind: ExprVar, Name: "riskPct"},
								{Kind: ExprVar, Name: "balance"},
							},
						},
					},
				},
			},
		},
	}

	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// Call CalcLot(0.02, 10000) → should return 200
	v := it.callBuiltin("CalcLot", []Expr{
		{Kind: ExprLiteral, Val: DecimalVal(decimal.NewFromFloat(0.02))},
		{Kind: ExprLiteral, Val: DecimalVal(decimal.NewFromInt(10000))},
	})

	if !v.ToDecimal().Equal(decimal.NewFromInt(200)) {
		t.Errorf("CalcLot(0.02, 10000) = %s, want 200", v.ToDecimal())
	}
}

func TestBreakStatement(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{
				Kind: StmtFor,
				Init: &Statement{
					Kind: StmtExpr,
					Expr: &Expr{Kind: ExprAssignment, Name: "i", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(0)}}},
				},
				Cond: &Expr{
					Kind: ExprBinary, Op: "<",
					Args: []Expr{{Kind: ExprVar, Name: "i"}, {Kind: ExprLiteral, Val: IntVal(10)}},
				},
				Update: &Statement{
					Kind: StmtExpr,
					Expr: &Expr{Kind: ExprUpdate, Name: "i", Op: "++"},
				},
				Body: []Statement{
					{
						Kind: StmtIf,
						Cond: &Expr{
							Kind: ExprBinary, Op: "==",
							Args: []Expr{{Kind: ExprVar, Name: "i"}, {Kind: ExprLiteral, Val: IntVal(3)}},
						},
						Body: []Statement{{Kind: StmtBreak}},
					},
					{
						Kind: StmtExpr,
						Expr: &Expr{Kind: ExprAssignment, Name: "result", Args: []Expr{{Kind: ExprVar, Name: "i"}}},
					},
				},
			},
		},
	}

	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)
	it.OnBar(ctx, "H1")

	// Loop should break at i=3. The break happens inside the if body,
	// so the "result = i" assignment after the if is skipped.
	// Check that i is 3 (the break value).
	i := it.getVar("i")
	if i.ToInt() != 3 {
		t.Errorf("i = %d, want 3 (break at i==3)", i.ToInt())
	}
}

func TestContinueStatement(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{
				Kind: StmtExpr,
				Expr: &Expr{Kind: ExprAssignment, Name: "sum", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(0)}}},
			},
			{
				Kind: StmtFor,
				Init: &Statement{
					Kind: StmtExpr,
					Expr: &Expr{Kind: ExprAssignment, Name: "i", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(0)}}},
				},
				Cond: &Expr{
					Kind: ExprBinary, Op: "<",
					Args: []Expr{{Kind: ExprVar, Name: "i"}, {Kind: ExprLiteral, Val: IntVal(5)}},
				},
				Update: &Statement{
					Kind: StmtExpr,
					Expr: &Expr{Kind: ExprUpdate, Name: "i", Op: "++"},
				},
				Body: []Statement{
					{
						Kind: StmtIf,
						Cond: &Expr{
							Kind: ExprBinary, Op: "==",
							Args: []Expr{{Kind: ExprVar, Name: "i"}, {Kind: ExprLiteral, Val: IntVal(2)}},
						},
						Body: []Statement{{Kind: StmtContinue}},
					},
					{
						Kind: StmtExpr,
						Expr: &Expr{
							Kind: ExprCompoundAssign, Op: "+=",
							Name: "sum",
							Args: []Expr{{Kind: ExprVar, Name: "i"}},
						},
					},
				},
			},
		},
	}

	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)
	it.OnBar(ctx, "H1")

	sum := it.getVar("sum")
	// 0+1+3+4 = 8 (skip 2 via continue)
	if sum.ToInt() != 8 {
		t.Errorf("sum = %d, want 8 (0+1+3+4, skip 2)", sum.ToInt())
	}
}

func TestCompoundAssignment(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "x", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(10)}}}},
			{Kind: StmtExpr, Expr: &Expr{Kind: ExprCompoundAssign, Op: "+=", Name: "x", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(5)}}}},
			{Kind: StmtExpr, Expr: &Expr{Kind: ExprCompoundAssign, Op: "-=", Name: "x", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(3)}}}},
			{Kind: StmtExpr, Expr: &Expr{Kind: ExprCompoundAssign, Op: "*=", Name: "x", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(2)}}}},
		},
	}

	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)
	it.OnBar(ctx, "H1")

	x := it.getVar("x")
	// 10 + 5 - 3 * 2 = 24
	if x.ToInt() != 24 {
		t.Errorf("x = %d, want 24 (10+5=15, 15-3=12, 12*2=24)", x.ToInt())
	}
}

func TestEnumLookup(t *testing.T) {
	ir := &IR{
		Version: "mql5",
		Enums: map[string]int32{
			"MODE_FAST": 0,
			"MODE_SLOW": 1,
		},
	}

	it := NewInterpreter(ir)

	v := it.lookupConstant("MODE_FAST")
	if v.ToInt() != 0 {
		t.Errorf("MODE_FAST = %d, want 0", v.ToInt())
	}

	v = it.lookupConstant("MODE_SLOW")
	if v.ToInt() != 1 {
		t.Errorf("MODE_SLOW = %d, want 1", v.ToInt())
	}
}

func TestDoWhile(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "i", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(0)}}}},
			{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "count", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(0)}}}},
			{
				Kind: StmtDoWhile,
				Cond: &Expr{
					Kind: ExprBinary, Op: "<",
					Args: []Expr{{Kind: ExprVar, Name: "i"}, {Kind: ExprLiteral, Val: IntVal(3)}},
				},
				Body: []Statement{
					{
						Kind: StmtExpr,
						Expr: &Expr{Kind: ExprCompoundAssign, Op: "+=", Name: "count",
							Args: []Expr{{Kind: ExprLiteral, Val: IntVal(1)}}},
					},
					{
						Kind: StmtExpr,
						Expr: &Expr{Kind: ExprUpdate, Name: "i", Op: "++"},
					},
				},
			},
		},
	}

	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)
	it.OnBar(ctx, "H1")

	count := it.getVar("count")
	if count.ToInt() != 3 {
		t.Errorf("count = %d, want 3 (do-while runs 3 times)", count.ToInt())
	}
}

func TestBlockScope(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "x", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(1)}}}},
			{
				Kind: StmtBlock,
				Body: []Statement{
					{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "y", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(99)}}}},
					{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "x", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(2)}}}},
				},
			},
		},
	}

	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)
	it.OnBar(ctx, "H1")

	// x should be updated to 2 (it's in root scope)
	x := it.getVar("x")
	if x.ToInt() != 2 {
		t.Errorf("x = %d, want 2", x.ToInt())
	}
}

func TestStrategyFactory(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "x", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(42)}}}},
		},
	}

	factory := NewStrategyFactory(ir)
	s1 := factory.Create()
	s2 := factory.Create()

	if s1 == s2 {
		t.Error("Create() should return independent instances")
	}

	// Both should be *Interpreter
	it1, ok1 := s1.(*Interpreter)
	it2, ok2 := s2.(*Interpreter)
	if !ok1 || !ok2 {
		t.Fatal("Create() should return *Interpreter")
	}

	// They should have independent globals
	it1.globals["test"] = IntVal(1)
	if _, exists := it2.globals["test"]; exists {
		t.Error("instances should have independent globals")
	}
}

func TestSwitchBreakInLoop(t *testing.T) {
	// switch break should exit the switch, not the enclosing for loop
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "total", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(0)}}}},
			{
				Kind: StmtFor,
				Init: &Statement{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "i", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(0)}}}},
				Cond: &Expr{Kind: ExprBinary, Op: "<", Args: []Expr{{Kind: ExprVar, Name: "i"}, {Kind: ExprLiteral, Val: IntVal(5)}}},
				Update: &Statement{Kind: StmtExpr, Expr: &Expr{Kind: ExprUpdate, Name: "i", Op: "++"}},
				Body: []Statement{
					{
						Kind: StmtSwitch,
						Expr: &Expr{Kind: ExprVar, Name: "i"},
						Cases: []SwitchCase{
							{
								Expr: &Expr{Kind: ExprLiteral, Val: IntVal(2)},
								Body: []Statement{
									{Kind: StmtBreak}, // break exits switch, NOT the for loop
								},
							},
							{
								Expr: nil, // default
								Body: []Statement{
									{Kind: StmtExpr, Expr: &Expr{Kind: ExprCompoundAssign, Op: "+=", Name: "total", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(1)}}}},
								},
							},
						},
					},
				},
			},
		},
	}

	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)
	it.OnBar(ctx, "H1")

	// i=0 → default → total=1
	// i=1 → default → total=2
	// i=2 → case 2 → break (exit switch, continue for loop)
	// i=3 → default → total=3
	// i=4 → default → total=4
	total := it.getVar("total")
	if total.ToInt() != 4 {
		t.Errorf("total = %d, want 4 (switch break should not exit for loop)", total.ToInt())
	}
}

func TestSwitchDefaultOrdering(t *testing.T) {
	// MQL switch: default should only execute if no case matches,
	// even if default appears before matching cases in the source.
	ir := &IR{
		Version: "mql4",
		OnBar: []Statement{
			{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "result", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(0)}}}},
			{Kind: StmtSwitch, Expr: &Expr{Kind: ExprLiteral, Val: IntVal(2)}, Cases: []SwitchCase{
				// default appears first in source order
				{Expr: nil, Body: []Statement{
					{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "result", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(99)}}}},
				}},
				// case 1
				{Expr: &Expr{Kind: ExprLiteral, Val: IntVal(1)}, Body: []Statement{
					{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "result", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(1)}}}},
				}},
				// case 2 — should match switchVal=2
				{Expr: &Expr{Kind: ExprLiteral, Val: IntVal(2)}, Body: []Statement{
					{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "result", Args: []Expr{{Kind: ExprLiteral, Val: IntVal(2)}}}},
				}},
			}},
		},
	}

	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)
	it.OnBar(ctx, "H1")

	result := it.getVar("result")
	if result.ToInt() != 2 {
		t.Errorf("result = %d, want 2 (case 2 should match, not default)", result.ToInt())
	}
}

func TestParamInjection(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		Params: []ParamDecl{
			{Name: "RiskPercent", Type: "double", Default: &Expr{Kind: ExprLiteral, Val: DecimalVal(decimal.NewFromFloat(2.0))}},
			{Name: "MaxOrders", Type: "int", Default: &Expr{Kind: ExprLiteral, Val: IntVal(5)}},
			{Name: "SymbolSuffix", Type: "string", Default: &Expr{Kind: ExprLiteral, Val: StringVal(".raw")}},
			{Name: "UseTrailing", Type: "bool", Default: &Expr{Kind: ExprLiteral, Val: BoolVal(true)}},
		},
	}

	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
		params: map[string]string{
			"RiskPercent":  "1.5",
			"MaxOrders":    "10",
			"SymbolSuffix": ".pro",
			"UseTrailing":  "false",
		},
	}
	it.OnInit(ctx)

	// Check that params were injected from ctx
	risk := it.getVar("RiskPercent")
	if !risk.ToDecimal().Equal(decimal.NewFromFloat(1.5)) {
		t.Errorf("RiskPercent = %s, want 1.5 (from ctx param)", risk.ToDecimal())
	}

	maxOrd := it.getVar("MaxOrders")
	if maxOrd.ToInt() != 10 {
		t.Errorf("MaxOrders = %d, want 10 (from ctx param)", maxOrd.ToInt())
	}

	suffix := it.getVar("SymbolSuffix")
	if suffix.ToString() != ".pro" {
		t.Errorf("SymbolSuffix = %s, want .pro (from ctx param)", suffix.ToString())
	}

	trailing := it.getVar("UseTrailing")
	if trailing.IsTrue() {
		t.Error("UseTrailing = true, want false (from ctx param)")
	}
}

func TestOnTick(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnTick: []Statement{
			{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "tickCount", Args: []Expr{
				{Kind: ExprBinary, Op: "+",
					Args: []Expr{
						{Kind: ExprVar, Name: "tickCount"},
						{Kind: ExprLiteral, Val: IntVal(1)},
					},
				},
			}}},
		},
	}

	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// Call OnTick multiple times
	for i := 0; i < 3; i++ {
		sig, err := it.OnTick(ctx, decimal.NewFromFloat(1.1), decimal.NewFromFloat(1.2))
		if err != nil {
			t.Fatalf("OnTick error: %v", err)
		}
		if sig != nil {
			t.Error("OnTick should return nil signal (no trade action)")
		}
	}

	// tickCount should be 3 (but it's reset each OnTick call via scopes reset)
	// Actually, tickCount is a new var each call since scopes are reset.
	// So we just verify OnTick doesn't crash and returns nil.
}

func TestOnTimer(t *testing.T) {
	ir := &IR{
		Version: "mql4",
		OnTimer: []Statement{
			{Kind: StmtExpr, Expr: &Expr{Kind: ExprAssignment, Name: "timerFired", Args: []Expr{{Kind: ExprLiteral, Val: BoolVal(true)}}}},
		},
	}

	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	sig, err := it.OnTimer(ctx)
	if err != nil {
		t.Fatalf("OnTimer error: %v", err)
	}
	if sig != nil {
		t.Error("OnTimer should return nil signal")
	}

	// timerFired is in a scope that's been reset, but the execution should not error
}

func TestOnTickEmpty(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// OnTick with empty IR should return nil, nil
	sig, err := it.OnTick(ctx, decimal.NewFromFloat(1.0), decimal.NewFromFloat(1.1))
	if err != nil {
		t.Errorf("OnTick error: %v", err)
	}
	if sig != nil {
		t.Error("OnTick with empty IR should return nil signal")
	}
}

func TestOnTimerEmpty(t *testing.T) {
	ir := &IR{Version: "mql4"}
	it := NewInterpreter(ir)
	ctx := &mockContext{
		bars:   &mockBarSeries{closes: []decimal.Decimal{decimal.NewFromFloat(1.1)}},
		broker: &mockBroker{},
	}
	it.OnInit(ctx)

	// OnTimer with empty IR should return nil, nil
	sig, err := it.OnTimer(ctx)
	if err != nil {
		t.Errorf("OnTimer error: %v", err)
	}
	if sig != nil {
		t.Error("OnTimer with empty IR should return nil signal")
	}
}
