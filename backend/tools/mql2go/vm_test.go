package mql2go

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// TestCompileAST_SimpleEA verifies that a simple MQL EA compiles to bytecode
// and the VM can execute the OnInit handler.
func TestCompileAST_SimpleEA(t *testing.T) {
	source := `
extern int MagicNumber = 12345;
extern double LotSize = 0.1;

int OnInit()
{
    return 0;
}

void OnTick()
{
    double price = Close[0];
    if(price > 0)
    {
        Print("price is positive");
    }
}
`
	// Compile MQL → IR (existing CST→AST converter)
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	// Compile IR → Bytecode
	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	// Verify bytecode structure
	if bc.OnInit < 0 {
		t.Error("OnInit entry point should be set")
	}
	if bc.OnTick < 0 {
		t.Error("OnTick entry point should be set")
	}
	if len(bc.Code) == 0 {
		t.Error("Bytecode should have instructions")
	}
	if len(bc.Consts) == 0 {
		t.Error("Bytecode should have constants")
	}

	// Verify globals were registered
	if _, ok := bc.GlobalSlots["MagicNumber"]; !ok {
		t.Error("MagicNumber should be in GlobalSlots")
	}
	if _, ok := bc.GlobalSlots["LotSize"]; !ok {
		t.Error("LotSize should be in GlobalSlots")
	}

	// Run OnInit
	vm := NewVM(bc)
	if err := vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}

	// Run OnTick (should not error even without SDK context)
	if err := vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick failed: %v", err)
	}
}

// TestCompileAST_Arithmetic verifies basic arithmetic compilation and execution.
func TestCompileAST_Arithmetic(t *testing.T) {
	source := `
extern int x = 10;
extern int y = 20;

void OnTick()
{
    int z = x + y;
    if(z > 25)
    {
        Print("z is large");
    }
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	vm := NewVM(bc)
	if err := vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick failed: %v", err)
	}
}

// TestCompileAST_UserFunction verifies user-defined function compilation and calling.
func TestCompileAST_UserFunction(t *testing.T) {
	source := `
int MyAdd(int a, int b)
{
    return a + b;
}

void OnTick()
{
    int result = MyAdd(3, 4);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	// Verify user function was registered
	fn, ok := bc.Funcs["MyAdd"]
	if !ok {
		t.Fatal("MyAdd function should be registered")
	}
	if fn.NumParams != 2 {
		t.Errorf("MyAdd should have 2 params, got %d", fn.NumParams)
	}

	vm := NewVM(bc)
	if err := vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick failed: %v", err)
	}
}

// TestCompileAST_CoverageReport verifies that the coverage report is populated.
func TestCompileAST_CoverageReport(t *testing.T) {
	source := `
void OnTick()
{
    UnknownFunction(42);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	if bc.Coverage == nil {
		t.Fatal("Coverage report should not be nil")
	}

	// Should have recorded a blind spot for UnknownFunction
	found := false
	for _, bs := range bc.Coverage.BlindSpots {
		if bs == "unknown function: UnknownFunction" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Coverage report should contain blind spot for UnknownFunction")
	}
}

// TestVM_ArithmeticDirect tests the VM arithmetic operations directly.
func TestVM_ArithmeticDirect(t *testing.T) {
	bc := &Bytecode{
		Consts: []ConstValue{
			{Kind: interp.ValDecimal, Dec: decimal.NewFromInt(10)},
			{Kind: interp.ValDecimal, Dec: decimal.NewFromInt(3)},
		},
		Code: []Instruction{
			{Op: OP_PUSH_CONST, A: 0},
			{Op: OP_PUSH_CONST, A: 1},
			{Op: OP_ADD},
			{Op: OP_HALT},
		},
		GlobalSlots: make(map[string]VarID),
		Funcs:       make(map[string]FuncEntry),
		Builtins:    make(map[string]BuiltinID),
	}

	vm := NewVM(bc)
	vm.globals = []interp.Value{}
	err := vm.runEvent(context.Background(), 0)
	if err != nil {
		t.Fatalf("runEvent failed: %v", err)
	}

	// Stack should have one value: 13
	if len(vm.stack) != 1 {
		t.Fatalf("expected 1 value on stack, got %d", len(vm.stack))
	}
	result := vm.stack[0]
	if result.Kind != interp.ValDecimal {
		t.Errorf("expected decimal result, got %v", result.Kind)
	}
	if !result.Decimal.Equal(decimal.NewFromInt(13)) {
		t.Errorf("expected 13, got %s", result.Decimal.String())
	}
}

// TestAnalyzeCoverage verifies the combined coverage report.
func TestAnalyzeCoverage(t *testing.T) {
	source := `
extern int Period = 14;
extern double LotSize = 0.1;

void OnTick()
{
    double rsi = iRSI(Symbol(), 0, Period, PRICE_CLOSE, 0);
    if(rsi > 70)
    {
        OrderClose(0, LotSize, Bid, 5);
    }
    UnknownFunc(42);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	result := AnalyzeCoverage(ir, bc)

	if result.Version == "" {
		t.Error("Version should be set")
	}
	if result.TotalCalls == 0 {
		t.Error("TotalCalls should be > 0")
	}
	if len(result.BlindSpots) == 0 {
		t.Error("Should have blind spots for UnknownFunc")
	}

	// Check that UnknownFunc appears in blind spots
	found := false
	for _, bs := range result.BlindSpots {
		if bs.Builtin == "UnknownFunc" {
			found = true
			break
		}
	}
	if !found {
		t.Error("UnknownFunc should be in blind spots")
	}

	// Check params were extracted
	if len(result.Indicators) == 0 {
		t.Error("Should have detected indicators")
	}
}

// TestExtractParams verifies parameter extraction from bytecode.
func TestExtractParams(t *testing.T) {
	source := `
extern int MAPeriod = 14;
extern double LotSize = 0.1;
extern string Comment = "test";

void OnTick() {}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	params := ExtractParams(bc)
	if len(params) != 3 {
		t.Fatalf("expected 3 params, got %d", len(params))
	}

	infos := ExtractParamInfos(bc)
	if len(infos) != 2 {
		t.Fatalf("expected 2 param infos (extern string Comment filtered as unreferenced), got %d", len(infos))
	}

	// Verify first param
	if infos[0].Name != "MAPeriod" {
		t.Errorf("expected MAPeriod, got %s", infos[0].Name)
	}
	if infos[0].Type != "int" {
		t.Errorf("expected int type, got %s", infos[0].Type)
	}
	if infos[0].Default != "14" {
		t.Errorf("expected default 14, got %s", infos[0].Default)
	}

	// Verify serialization
	raw := SerializeParams(bc)
	if len(raw) == 0 {
		t.Error("SerializeParams should return non-empty bytes")
	}
}

// TestExtractParamInfos_FiltersUnreferencedStringParams verifies that extern string
// parameters used as UI labels (never read in code) are filtered out,
// while extern string parameters that ARE referenced in code are kept.
func TestExtractParamInfos_FiltersUnreferencedStringParams(t *testing.T) {
	source := `
extern string LabelText = "This is a UI label, not a parameter";
extern string TradeComment = "EA";
extern int Period = 14;

void OnTick() {
    int ticket = OrderSend(Symbol(), OP_BUY, 0.1, Ask, 3, 0, 0, TradeComment, 12345, 0, clrGreen);
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR failed: %v", err)
	}
	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST failed: %v", err)
	}

	infos := ExtractParamInfos(bc)
	names := make(map[string]bool)
	for _, info := range infos {
		names[info.Name] = true
	}

	// LabelText is never referenced → filtered
	if names["LabelText"] {
		t.Error("LabelText should be filtered (unreferenced extern string)")
	}
	// TradeComment IS referenced in OrderSend → kept
	if !names["TradeComment"] {
		t.Error("TradeComment should be kept (referenced in OnTick)")
	}
	// Period is non-string → always kept
	if !names["Period"] {
		t.Error("Period should be kept (non-string param)")
	}
}

// TestVMRunner verifies the in-process execution entrypoint.
func TestVMRunner(t *testing.T) {
	source := `
extern int MagicNumber = 12345;
extern double LotSize = 0.1;

int OnInit()
{
    return 0;
}

void OnTick()
{
    double price = Close[0];
    if(price > 0)
    {
        Print("price is positive");
    }
}
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	// Verify it compiles and has bytecode
	if runner.bc == nil {
		t.Fatal("Bytecode should not be nil")
	}
	if runner.bc.OnTick < 0 {
		t.Error("OnTick should be compiled")
	}
	if runner.bc.OnInit < 0 {
		t.Error("OnInit should be compiled")
	}

	// Verify params are accessible
	params := ExtractParams(runner.bc)
	if len(params) != 2 {
		t.Errorf("expected 2 params, got %d", len(params))
	}
}

// TestCompileMQLWithCoverage verifies the combined compile + coverage path.
func TestCompileMQLWithCoverage(t *testing.T) {
	source := `
extern int Period = 14;

void OnTick()
{
    double rsi = iRSI(Symbol(), 0, Period, PRICE_CLOSE, 0);
    UnknownFunc(42);
}
`
	runner, coverage, err := CompileMQLWithCoverage(source)
	if err != nil {
		t.Fatalf("CompileMQLWithCoverage failed: %v", err)
	}

	if coverage == nil {
		t.Fatal("Coverage should not be nil")
	}
	if coverage.Version == "" {
		t.Error("Version should be set")
	}
	if len(coverage.BlindSpots) == 0 {
		t.Error("Should have blind spots")
	}

	if runner == nil {
		t.Fatal("Runner should not be nil")
	}
}
