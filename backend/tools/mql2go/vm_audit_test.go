package mql2go

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

func TestVM_Audit_UserToUserForwardReference(t *testing.T) {
	// Function names are chosen so that the caller ("aaa_caller") sorts
	// BEFORE the callee ("zzz_callee") alphabetically. The compiler sorts
	// user function names and compiles bodies in that order, so aaa_caller's
	// body is compiled before zzz_callee's body. Without relocation, the
	// OP_CALL_USER inside aaa_caller would embed zzz_callee's stale Pass-1
	// marker PC (which hasn't been updated to the body start yet).
	const source = `
int g_result = 0;
int OnInit() { return 0; }
void OnTick() { g_result = aaa_caller(); }
int aaa_caller() { return zzz_callee(); }
int zzz_callee() { return 42; }
`

	for i := 0; i < 100; i++ {
		runner, err := CompileMQL(source)
		if err != nil {
			t.Fatalf("CompileMQL iteration %d: %v", i, err)
		}
		if err := runner.vm.RunOnTick(context.Background()); err != nil {
			t.Fatalf("RunOnTick iteration %d: %v", i, err)
		}
		value, ok := runner.GetGlobal("g_result")
		if !ok || value.ToInt() != 42 {
			t.Fatalf("iteration %d: g_result = %v, want 42", i, value)
		}
	}
}

func TestVM_Audit_UserToUserForwardReference_Structure(t *testing.T) {
	// "aaa_caller" sorts before "zzz_callee", so aaa_caller's body is compiled
	// before zzz_callee's body. Without relocation, the OP_CALL_USER inside
	// aaa_caller would embed zzz_callee's stale Pass-1 marker PC. This test
	// asserts the structural invariant: every OP_CALL_USER operand equals
	// the callee's FINAL EntryPC (body start), never the OP_ENTER_FUNC marker.
	const source = `
int g_result = 0;
int OnInit() { return 0; }
void OnTick() { g_result = aaa_caller(); }
int aaa_caller() { return zzz_callee(); }
int zzz_callee() { return 42; }
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	bc := runner.Bytecode()

	// Invariant 1: every FuncEntry.EntryPC must point at the body start,
	// not at an OP_ENTER_FUNC marker.
	for name, fn := range bc.Funcs {
		if fn.EntryPC < 0 || int(fn.EntryPC) >= len(bc.Code) {
			t.Fatalf("function %s has out-of-range EntryPC %d (code len %d)", name, fn.EntryPC, len(bc.Code))
		}
		if bc.Code[fn.EntryPC].Op == OP_ENTER_FUNC {
			t.Fatalf("function %s EntryPC %d points at OP_ENTER_FUNC marker, not body start", name, fn.EntryPC)
		}
	}

	// Invariant 2: every OP_CALL_USER operand must equal a real function's
	// final EntryPC. Collect valid entry PCs.
	entryPCs := make(map[int32]string)
	for name, fn := range bc.Funcs {
		entryPCs[fn.EntryPC] = name
	}
	callCount := 0
	for i, ins := range bc.Code {
		if ins.Op != OP_CALL_USER {
			continue
		}
		callCount++
		if ins.A < 0 {
			t.Fatalf("OP_CALL_USER at instruction %d has unresolved placeholder A=%d (patchUserCalls did not run)", i, ins.A)
		}
		calleeName, ok := entryPCs[ins.A]
		if !ok {
			t.Fatalf("OP_CALL_USER at instruction %d targets PC %d which is not any function's EntryPC", i, ins.A)
		}
		// The target must not be an OP_ENTER_FUNC marker.
		if bc.Code[ins.A].Op == OP_ENTER_FUNC {
			t.Fatalf("OP_CALL_USER at instruction %d targets %s's marker PC %d, not body start", i, calleeName, ins.A)
		}
	}
	if callCount == 0 {
		t.Fatal("expected at least one OP_CALL_USER instruction in bytecode")
	}

	// Invariant 3: the aaa_caller→zzz_callee edge specifically. aaa_caller
	// calls zzz_callee, so there must be an OP_CALL_USER whose target is
	// zzz_callee's EntryPC.
	calleeEntry, ok := bc.Funcs["zzz_callee"]
	if !ok {
		t.Fatal("function 'zzz_callee' not found in bytecode")
	}
	foundCallerToCallee := false
	for _, ins := range bc.Code {
		if ins.Op == OP_CALL_USER && ins.A == calleeEntry.EntryPC {
			foundCallerToCallee = true
			break
		}
	}
	if !foundCallerToCallee {
		t.Fatalf("no OP_CALL_USER targets zzz_callee's EntryPC %d — caller→callee forward reference was not relocated", calleeEntry.EntryPC)
	}
}

func TestVM_Audit_NormalCompileCarriesCoverageResult(t *testing.T) {
	const source = `
int OnInit() { return 0; }
void OnTick() { double v = iUnknownIndicator(Symbol(), 0, 14, 0); }
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	cov := runner.GetCoverageResult()
	if cov == nil {
		t.Fatal("normal CompileMQL path lost CoverageResult")
	}
	foundFatal := false
	for _, bs := range cov.BlindSpots {
		if bs.Builtin == "iUnknownIndicator" && bs.Severity == interp.SeverityFatal {
			foundFatal = true
		}
	}
	if !foundFatal {
		t.Fatalf("unknown indicator was not classified as fatal: %+v", cov.BlindSpots)
	}
}

func TestVM_Audit_MQLFieldAssignment(t *testing.T) {
	const source = `
struct State { int value; };
State state;
int OnInit() {
    state.value = 42;
    return 0;
}
void OnTick() {}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR: %v", err)
	}
	if !containsExpr(ir.OnInit, func(e *interp.Expr) bool {
		return e.Kind == interp.ExprField && e.IsAssign && e.Name == "value"
	}) {
		t.Fatal("state.value = 42 was not preserved as a field assignment")
	}
}

func TestVM_Audit_MQLFieldAssignment_VMBehavior(t *testing.T) {
	// Verify field assignment executes correctly at the VM level:
	// state.value = 42 in OnInit, then read it back in OnTick.
	const source = `
struct State { int value; };
State state;
int g_readback = 0;
int OnInit() {
    state.value = 42;
    return 0;
}
void OnTick() { g_readback = state.value; }
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	if err := runner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit: %v", err)
	}
	if err := runner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick: %v", err)
	}
	value, ok := runner.GetGlobal("g_readback")
	if !ok || value.ToInt() != 42 {
		t.Fatalf("field assignment readback = %v, want 42", value)
	}
}

func TestVM_Audit_MQLArrayAssignment(t *testing.T) {
	const source = `
 double values[2];
 int g_value = 0;
 int OnInit() {
     values[0] = 7.5;
     return 0;
 }
 void OnTick() { g_value = values[0]; }
 `
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR: %v", err)
	}
	if !containsExpr(ir.OnInit, func(e *interp.Expr) bool {
		return e.Kind == interp.ExprSubscript && e.Name == "values" && len(e.Args) == 1
	}) {
		t.Fatal("values[0] = 7.5 was not preserved as an array assignment")
	}

	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST: %v", err)
	}
	vm := NewVM(bc)
	if err := vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit: %v", err)
	}
	if err := vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick: %v", err)
	}
	if got := vm.globals[bc.GlobalSlots["g_value"]].ToDecimal(); !got.Equal(decimal.NewFromFloat(7.5)) {
		t.Fatalf("g_value = %s, want 7.5", got)
	}
}

func TestVM_Audit_UninitializedLocalDeclaration(t *testing.T) {
	const source = `
int g_value = 0;
int OnInit() { return 0; }
void OnTick() {
    int local_value;
    local_value = 7;
    g_value = local_value;
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR: %v", err)
	}
	bc, err := CompileAST(ir)
	if err != nil {
		t.Fatalf("CompileAST: %v", err)
	}
	if _, ok := bc.GlobalSlots["local_value"]; ok {
		t.Fatal("uninitialized local declaration was incorrectly promoted to a global")
	}
	if bc.EventLocals[bc.OnTick] == 0 {
		t.Fatal("uninitialized local declaration did not allocate an event-local slot")
	}
	// Verify VM behavior: the local is zero-initialized, then assigned 7,
	// and g_value reads back 7.
	vm := NewVM(bc)
	if err := vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit: %v", err)
	}
	if err := vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick: %v", err)
	}
	if got := vm.globals[bc.GlobalSlots["g_value"]].ToInt(); got != 7 {
		t.Fatalf("uninitialized local readback = %d, want 7", got)
	}
}

func TestVM_Audit_MultiVariableDeclaration(t *testing.T) {
	// Multi-variable declaration: int x = 5, y = 6; must produce an ExprSeq
	// with two ExprDecl nodes, both accessible as locals.
	const source = `
int g_x = 0;
int g_y = 0;
int OnInit() { return 0; }
void OnTick() {
    int x = 5, y = 6;
    g_x = x;
    g_y = y;
}
`
	ir, err := CompileToIR(source)
	if err != nil {
		t.Fatalf("CompileToIR: %v", err)
	}
	// Assert the IR contains an ExprSeq with at least 2 ExprDecl children.
	foundSeq := false
	for _, stmt := range ir.OnTick {
		if stmt.Expr != nil && stmt.Expr.Kind == interp.ExprSeq {
			declCount := 0
			for _, arg := range stmt.Expr.Args {
				if arg.Kind == interp.ExprDecl {
					declCount++
				}
			}
			if declCount >= 2 {
				foundSeq = true
			}
		}
	}
	if !foundSeq {
		t.Fatal("multi-variable declaration was not preserved as ExprSeq with 2+ ExprDecl")
	}
	// Verify VM behavior: both locals are accessible and have correct values.
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	if err := runner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit: %v", err)
	}
	if err := runner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick: %v", err)
	}
	x, ok := runner.GetGlobal("g_x")
	if !ok || x.ToInt() != 5 {
		t.Fatalf("g_x = %v, want 5", x)
	}
	y, ok := runner.GetGlobal("g_y")
	if !ok || y.ToInt() != 6 {
		t.Fatalf("g_y = %v, want 6", y)
	}
}

func TestVM_Audit_LocalArrayDeclarationRejected(t *testing.T) {
	const source = `
int OnInit() { return 0; }
void OnTick() {
    double local_values[2];
    local_values[0] = 1.0;
}
`
	if _, err := CompileMQL(source); err == nil || !strings.Contains(err.Error(), "local arrays are not supported") {
		t.Fatalf("local array declaration should not be silently compiled, got %v", err)
	}
}

func TestVM_Audit_UnsupportedBitwiseOperatorRejected(t *testing.T) {
	const source = `
int OnInit() { return 0; }
void OnTick() {
    int flags = 1 & 1;
}
`
	_, err := CompileMQL(source)
	if err == nil || !strings.Contains(err.Error(), "unsupported binary operator") {
		t.Fatalf("bitwise operator should fail compilation explicitly, got %v", err)
	}
}

type auditModifySignalStrategy struct{}

func (auditModifySignalStrategy) OnInit(sdk.Context) error           { return nil }
func (auditModifySignalStrategy) OnDeinit(sdk.Context, string) error { return nil }
func (auditModifySignalStrategy) OnBar(ctx sdk.Context, _ string) (*sdk.Signal, error) {
	positions := ctx.Broker().Positions(0)
	if len(positions) == 0 {
		return &sdk.Signal{Action: sdk.ActionBuy, Symbol: "EURUSD", Volume: decimal.NewFromFloat(0.1)}, nil
	}
	return &sdk.Signal{
		Action:      sdk.ActionModify,
		OrderTicket: positions[0].Ticket,
		StopLoss:    decimal.NewFromInt(99),
		TakeProfit:  decimal.NewFromInt(101),
	}, nil
}

type auditInvalidSignalStrategy struct{}

func (auditInvalidSignalStrategy) OnInit(sdk.Context) error           { return nil }
func (auditInvalidSignalStrategy) OnDeinit(sdk.Context, string) error { return nil }
func (auditInvalidSignalStrategy) OnBar(sdk.Context, string) (*sdk.Signal, error) {
	return &sdk.Signal{Action: sdk.ActionClose, OrderTicket: 999}, nil
}

type auditContext struct {
	sdk.Context
	bars   sdk.BarSeries
	symbol string
	tf     string
	broker sdk.Broker
}

func (c *auditContext) Bars() sdk.BarSeries                        { return c.bars }
func (c *auditContext) BarsTF(string) sdk.BarSeries                { return c.bars }
func (c *auditContext) BarsForSymbol(string, string) sdk.BarSeries { return c.bars }
func (c *auditContext) Symbol() string                             { return c.symbol }
func (c *auditContext) Timeframe() string                          { return c.tf }
func (c *auditContext) Broker() sdk.Broker                         { return c.broker }

func auditBars(n int) []sdk.Bar {
	bars := make([]sdk.Bar, n)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range bars {
		price := decimal.NewFromFloat(1.1 + float64(i)*0.001)
		bars[i] = sdk.Bar{
			Open: price, High: price.Add(decimal.NewFromFloat(0.001)),
			Low: price.Sub(decimal.NewFromFloat(0.001)), Close: price,
			Volume: 100, Timestamp: base.Add(time.Duration(i) * time.Minute).UnixMilli(),
		}
	}
	return bars
}

func containsExpr(stmts []interp.Statement, match func(*interp.Expr) bool) bool {
	for i := range stmts {
		if exprContains(stmts[i].Expr, match) || exprContains(stmts[i].Cond, match) {
			return true
		}
		if containsExpr(stmts[i].Body, match) || containsExpr(stmts[i].ElseBody, match) {
			return true
		}
		if stmts[i].Init != nil && containsExpr([]interp.Statement{*stmts[i].Init}, match) {
			return true
		}
		if stmts[i].Update != nil && containsExpr([]interp.Statement{*stmts[i].Update}, match) {
			return true
		}
		for _, sc := range stmts[i].Cases {
			if exprContains(sc.Expr, match) || containsExpr(sc.Body, match) {
				return true
			}
		}
	}
	return false
}

func exprContains(e *interp.Expr, match func(*interp.Expr) bool) bool {
	if e == nil {
		return false
	}
	if match(e) {
		return true
	}
	for i := range e.Args {
		if exprContains(&e.Args[i], match) {
			return true
		}
	}
	return exprContains(e.Index, match) || exprContains(e.Cond, match) ||
		exprContains(e.ThenExpr, match) || exprContains(e.ElseExpr, match)
}
