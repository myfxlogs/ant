package interp

import "testing"

func TestValidateDefenseA_NoEntry_Point(t *testing.T) {
	t.Parallel()
	ir := &IR{
		Version: "mql4",
		Funcs:   map[string]*FuncDef{},
	}
	violations := ValidateDefenseA(ir)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Rule != "no_entry_point" {
		t.Fatalf("rule: want no_entry_point, got %s", violations[0].Rule)
	}
	if violations[0].Severity != SeverityFatal {
		t.Fatalf("severity: want %s, got %s", SeverityFatal, violations[0].Severity)
	}
}

func TestValidateDefenseA_HasOnTick(t *testing.T) {
	t.Parallel()
	ir := &IR{
		Version: "mql4",
		OnTick:  []Statement{{Kind: StmtExpr}},
		Funcs:   map[string]*FuncDef{},
	}
	violations := ValidateDefenseA(ir)
	for _, v := range violations {
		if v.Rule == "no_entry_point" {
			t.Fatal("should not report no_entry_point when OnTick exists")
		}
	}
}

func TestValidateDefenseA_HasOnBar(t *testing.T) {
	t.Parallel()
	ir := &IR{
		Version: "mql4",
		OnBar:   []Statement{{Kind: StmtExpr}},
		Funcs:   map[string]*FuncDef{},
	}
	violations := ValidateDefenseA(ir)
	for _, v := range violations {
		if v.Rule == "no_entry_point" {
			t.Fatal("should not report no_entry_point when OnBar exists")
		}
	}
}

func TestValidateDefenseA_HasStartFunc(t *testing.T) {
	t.Parallel()
	ir := &IR{
		Version: "mql4",
		Funcs:   map[string]*FuncDef{"start": {}},
	}
	violations := ValidateDefenseA(ir)
	for _, v := range violations {
		if v.Rule == "no_entry_point" {
			t.Fatal("should not report no_entry_point when start function exists")
		}
	}
}

func TestValidateDefenseA_ParamNameKeyword(t *testing.T) {
	t.Parallel()
	ir := &IR{
		Version: "mql4",
		OnTick:  []Statement{{Kind: StmtExpr}},
		Funcs:   map[string]*FuncDef{},
		Params: []ParamDecl{
			{Name: "double", Type: "int"},
		},
	}
	violations := ValidateDefenseA(ir)
	found := false
	for _, v := range violations {
		if v.Rule == "param_name_keyword" && v.Identifier == "double" {
			found = true
		}
	}
	if !found {
		t.Fatal("should report param_name_keyword for 'double'")
	}
}

func TestValidateDefenseA_ParamNameBuiltinVar(t *testing.T) {
	t.Parallel()
	ir := &IR{
		Version: "mql4",
		OnTick:  []Statement{{Kind: StmtExpr}},
		Funcs:   map[string]*FuncDef{},
		Params: []ParamDecl{
			{Name: "Close", Type: "double"},
		},
	}
	violations := ValidateDefenseA(ir)
	found := false
	for _, v := range violations {
		if v.Rule == "param_name_keyword" && v.Identifier == "Close" {
			found = true
		}
	}
	if !found {
		t.Fatal("should report param_name_keyword for 'Close' (built-in series name)")
	}
}

func TestValidateDefenseA_ParamNameDuplicate(t *testing.T) {
	t.Parallel()
	ir := &IR{
		Version: "mql4",
		OnTick:  []Statement{{Kind: StmtExpr}},
		Funcs:   map[string]*FuncDef{},
		Params: []ParamDecl{
			{Name: "Lots", Type: "double"},
			{Name: "Lots", Type: "int"},
		},
	}
	violations := ValidateDefenseA(ir)
	found := false
	for _, v := range violations {
		if v.Rule == "param_name_duplicate" && v.Identifier == "Lots" {
			found = true
		}
	}
	if !found {
		t.Fatal("should report param_name_duplicate for 'Lots'")
	}
}

func TestValidateDefenseA_ValidParams(t *testing.T) {
	t.Parallel()
	ir := &IR{
		Version: "mql4",
		OnTick:  []Statement{{Kind: StmtExpr}},
		Funcs:   map[string]*FuncDef{},
		Params: []ParamDecl{
			{Name: "Lots", Type: "double"},
			{Name: "TakeProfit", Type: "int"},
			{Name: "StopLoss", Type: "int"},
		},
	}
	violations := ValidateDefenseA(ir)
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations for valid IR, got %d: %v", len(violations), violations)
	}
}

func TestValidateDefenseA_MultipleViolations(t *testing.T) {
	t.Parallel()
	ir := &IR{
		Version: "mql4",
		Funcs:   map[string]*FuncDef{},
		Params: []ParamDecl{
			{Name: "int", Type: "double"},
			{Name: "int", Type: "string"},
		},
	}
	violations := ValidateDefenseA(ir)
	if len(violations) < 3 {
		t.Fatalf("expected at least 3 violations (keyword + duplicate + no_entry), got %d", len(violations))
	}
}

func TestFormatDefenseAViolations_Empty(t *testing.T) {
	t.Parallel()
	if s := FormatDefenseAViolations(nil); s != "" {
		t.Fatalf("expected empty string, got %q", s)
	}
}

func TestFormatDefenseAViolations_NonEmpty(t *testing.T) {
	t.Parallel()
	violations := []DefenseAViolation{
		{Rule: "no_entry_point", Severity: SeverityFatal, Message: "no entry point"},
	}
	s := FormatDefenseAViolations(violations)
	if s == "" {
		t.Fatal("expected non-empty string")
	}
}
