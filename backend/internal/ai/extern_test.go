package ai

import (
"testing"
)

func TestExtractParams_MQLExtern(t *testing.T) {
code := `extern int CCIPeriod = 14;
extern double Lots = 0.1;
extern int MagicNumber = 987;
extern string Label = "test";
extern bool UseTrailing = true;

int OnInit() { return INIT_SUCCEEDED; }
void OnTick() { }`

params := ExtractParams(code)
if len(params) != 4 {
t.Fatalf("expected 4 numeric params (string filtered), got %d: %+v", len(params), params)
}

// Check CCIPeriod
if params[0].Name != "CCIPeriod" || params[0].Default != 14 {
t.Errorf("CCIPeriod: expected default=14, got %+v", params[0])
}
if params[0].Type != "int" {
t.Errorf("CCIPeriod: expected type=int, got %s", params[0].Type)
}

// Check Lots
if params[1].Name != "Lots" || params[1].Default != 0.1 {
t.Errorf("Lots: expected default=0.1, got %+v", params[1])
}
if params[1].Type != "float" {
t.Errorf("Lots: expected type=float, got %s", params[1].Type)
}

// Check bool
found := false
for _, p := range params {
if p.Name == "UseTrailing" {
found = true
if p.Default != 1 {
t.Errorf("UseTrailing: expected default=1 (true), got %f", p.Default)
}
}
}
if !found {
t.Error("UseTrailing bool param not found")
}
}

func TestExtractParams_MQL5Input(t *testing.T) {
code := `input int FastPeriod = 12;
input double Threshold = 0.5;`

params := ExtractParams(code)
if len(params) != 2 {
t.Fatalf("expected 2 params, got %d: %+v", len(params), params)
}
if params[0].Name != "FastPeriod" || params[0].Default != 12 {
t.Errorf("FastPeriod: expected default=12, got %+v", params[0])
}
}

func TestExtractParamsWithAnnotations_MQLExternWithParamAnnotation(t *testing.T) {
code := `// @param CCIPeriod 14 range=5:30:5
extern int CCIPeriod = 14;
extern double Lots = 0.1;`

params := ExtractParamsWithAnnotations(code)
if len(params) != 2 {
t.Fatalf("expected 2 params, got %d: %+v", len(params), params)
}

// CCIPeriod should have range from @param annotation
if params[0].Name != "CCIPeriod" {
t.Fatalf("first param should be CCIPeriod, got %s", params[0].Name)
}
if params[0].Min != 5 || params[0].Max != 30 || params[0].Step != 5 {
t.Errorf("CCIPeriod range: expected 5:30:5, got min=%f max=%f step=%f", params[0].Min, params[0].Max, params[0].Step)
}

// Lots should have no range (only default from extern)
if params[1].Name != "Lots" {
t.Fatalf("second param should be Lots, got %s", params[1].Name)
}
if params[1].Min != 0.1 || params[1].Max != 0.1 {
t.Errorf("Lots: expected min=max=0.1, got min=%f max=%f", params[1].Min, params[1].Max)
}
}
