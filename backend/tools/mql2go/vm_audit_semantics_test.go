package mql2go

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// ── VM-COMPILER-SEMANTICS-2 behavior tests ───────────────────────────

// TestVM_Audit_CompoundAssignField verifies that compound assignment on a
// struct field (obj.field += value) preserves compound semantics.
// VM-COMPILER-SEMANTICS-2: previously the compound operator was silently
// dropped, making `obj.field += v` equivalent to `obj.field = v`.
func TestVM_Audit_CompoundAssignField(t *testing.T) {
	const source = `
struct MyStruct { int x; };
MyStruct g_s;
int g_after = -1;
int OnInit() { return 0; }
void OnTick() {
    g_s.x = 10;
    g_s.x += 5;
    g_after = g_s.x;
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	after, _ := runner.GetGlobal("g_after")
	// 10 + 5 = 15 (compound += preserved)
	if after.ToInt() != 15 {
		t.Errorf("g_after = %d, want 15 (compound += on field must preserve semantics)", after.ToInt())
	}
}

// TestVM_Audit_CompoundAssignArray verifies that compound assignment on an
// array element (arr[i] += value) preserves compound semantics.
func TestVM_Audit_CompoundAssignArray(t *testing.T) {
	const source = `
int g_arr[3];
int g_after = -1;
int OnInit() { return 0; }
void OnTick() {
    g_arr[0] = 10;
    g_arr[0] += 5;
    g_after = g_arr[0];
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	after, _ := runner.GetGlobal("g_after")
	if after.ToInt() != 15 {
		t.Errorf("g_after = %d, want 15 (compound += on array must preserve semantics)", after.ToInt())
	}
}

// TestVM_Audit_CastExpressionInit verifies that findInitValue recognizes
// cast_expression nodes in declarations. VM-COMPILER-SEMANTICS-2: previously
// cast expressions in initializers were not found, leaving variables uninitialized.
func TestVM_Audit_CastExpressionInit(t *testing.T) {
	const source = `
int g_result = -1;
int OnInit() { return 0; }
void OnTick() {
    double d = 3.7;
    int x = (int)d;
    g_result = x;
}`
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
	val, _ := runner.GetGlobal("g_result")
	if val.ToInt() != 3 {
		t.Fatalf("g_result = %d, want 3 (cast expression init must be found by findInitValue)", val.ToInt())
	}
}

// TestVM_Audit_UnknownRootNodeRejected verifies that unrecognized top-level
// nodes produce a compile error instead of being silently skipped.
// VM-COMPILER-SEMANTICS-2: previously unknown root nodes were silently ignored.
func TestVM_Audit_UnknownRootNodeRejected(t *testing.T) {
	// "typedef" at root level is not handled by the compiler — it should
	// produce an error, not silently skip.
	const source = `
typedef int MyInt;
int OnInit() { return 0; }
void OnTick() { int x = 1; }
`
	_, err := CompileMQL(source)
	if err == nil {
		t.Fatal("CompileMQL should reject unrecognized top-level 'typedef' node, got nil error")
	}
}

// ── VM-TIMESERIES-SEMANTICS-2 behavior tests ─────────────────────────

// TestVM_Audit_IllegalTimeframeRecordsBlindSpot verifies that iClose with
// an illegal timeframe period fails closed (returns an error) instead of
// silently falling back to the primary timeframe.
// VM-TIMESERIES-SEMANTICS-3: illegal timeframe is now a fatal error.
func TestVM_Audit_IllegalTimeframeRecordsBlindSpot(t *testing.T) {
	const source = `
double g_result = -1;
int OnInit() { return 0; }
void OnTick() {
    // Period 999 is not a valid MQL timeframe — must fail-closed, not silently fall back.
    g_result = iClose("EURUSD", 999, 0);
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err == nil {
		t.Fatal("engine.Run should fail-closed for illegal timeframe period 999, got nil error")
	}
	if !strings.Contains(err.Error(), "illegal timeframe") {
		t.Fatalf("error should mention illegal timeframe, got: %v", err)
	}
}

// TestVM_Audit_PeriodCurrentZeroIsValid verifies that PERIOD_CURRENT (0)
// is NOT treated as illegal — it should resolve to the primary timeframe.
func TestVM_Audit_PeriodCurrentZeroIsValid(t *testing.T) {
	tf, ok := intToTF(0)
	if !ok {
		t.Fatal("intToTF(0) should return ok=true (PERIOD_CURRENT is valid)")
	}
	if tf != "" {
		t.Errorf("intToTF(0)=%q, want empty string (PERIOD_CURRENT sentinel)", tf)
	}
}

// TestVM_Audit_IllegalPeriodReturnsFalse verifies that intToTF returns
// false for unknown period values.
func TestVM_Audit_IllegalPeriodReturnsFalse(t *testing.T) {
	_, ok := intToTF(999)
	if ok {
		t.Fatal("intToTF(999) should return ok=false (illegal period)")
	}
	_, ok = intToTF(-1)
	if ok {
		t.Fatal("intToTF(-1) should return ok=false (illegal period)")
	}
}

// TestVM_Audit_PeriodSecondsCurrentTimeframe verifies that PeriodSeconds(0)
// (PERIOD_CURRENT) resolves to the context timeframe, not an empty string
// that defaults to 3600. VM-TIMESERIES-SEMANTICS-3.
func TestVM_Audit_PeriodSecondsCurrentTimeframe(t *testing.T) {
	const source = `
int g_result = -1;
int OnInit() { return 0; }
void OnTick() {
    g_result = PeriodSeconds(0);
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M5", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	val, _ := runner.GetGlobal("g_result")
	// M5 = 300 seconds. If PERIOD_CURRENT resolved to empty string,
	// tfDurationSeconds("") would return 0, not 300.
	if val.ToInt() != 300 {
		t.Fatalf("PeriodSeconds(0) = %d, want 300 (M5 context timeframe)", val.ToInt())
	}
}

// TestVM_Audit_PeriodSecondsIllegalFailsClosed verifies that PeriodSeconds
// with an illegal period fails closed (returns error). VM-TIMESERIES-SEMANTICS-3.
func TestVM_Audit_PeriodSecondsIllegalFailsClosed(t *testing.T) {
	const source = `
int g_result = -1;
int OnInit() { return 0; }
void OnTick() {
    g_result = PeriodSeconds(999);
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	engine := backtest.New(backtest.Config{
		Symbol: "EURUSD", Timeframe: "M1", InitialCapital: decimal.NewFromInt(100000), Leverage: 100,
	}, runner, auditBars(3))
	_, err = engine.Run(context.Background())
	if err == nil {
		t.Fatal("PeriodSeconds(999) should fail-closed, got nil error")
	}
}

// TestVM_Audit_TimeSecondsConstant verifies that TIME_SECONDS is 4 (MQL
// bit flag), not 3. VM-TIMESERIES-SEMANTICS-3.
func TestVM_Audit_TimeSecondsConstant(t *testing.T) {
	v, ok := interp.MQLConstants["TIME_SECONDS"]
	if !ok {
		t.Fatal("TIME_SECONDS constant not found")
	}
	if v.ToInt() != 4 {
		t.Fatalf("TIME_SECONDS = %d, want 4 (MQL bit flag: DATE=1, MINUTES=2, SECONDS=4)", v.ToInt())
	}
}

// ── VM-API-TRUTH-2 behavior tests ────────────────────────────────────

// TestVM_Audit_IsConnectedReturnsTrue verifies that IsConnected() returns
// constant true in both backtest and live contexts. VM-API-TRUTH-2: the VM
// is always connected to its host process; this is not a network probe.
func TestVM_Audit_IsConnectedReturnsTrue(t *testing.T) {
	vm := newSignalTestVM()
	val, err := builtinIsConnected(vm, nil)
	if err != nil {
		t.Fatalf("IsConnected: %v", err)
	}
	if !val.Bool {
		t.Error("IsConnected() = false, want true (VM is always connected to host)")
	}
}

// TestVM_Audit_IsDemoReturnsTrue verifies that IsDemo() returns true.
// VM-API-TRUTH-2: in backtest/live paper mode, the account is always demo.
func TestVM_Audit_IsDemoReturnsTrue(t *testing.T) {
	vm := newSignalTestVM()
	val, err := builtinIsDemo(vm, nil)
	if err != nil {
		t.Fatalf("IsDemo: %v", err)
	}
	if !val.Bool {
		t.Error("IsDemo() = false, want true (paper/demo mode)")
	}
}

// TestVM_Audit_IsDemoRealAccountReturnsFalse verifies that IsDemo() returns
// false when the account is a real account (IsDemo=false in context).
// VM-API-TRUTH-3: previously IsDemo was hardcoded true, even for real accounts.
// Adversarial: revert builtinIsDemo to `return BoolVal(true), nil` → RED.
func TestVM_Audit_IsDemoRealAccountReturnsFalse(t *testing.T) {
	vm := newSignalTestVM()
	vm.ctx = &testContext{
		broker:  &testBroker{},
		account: sdk.AccountInfo{IsDemo: false, IsConnected: true, IsTradeAllowed: true},
	}
	val, err := builtinIsDemo(vm, nil)
	if err != nil {
		t.Fatalf("IsDemo: %v", err)
	}
	if val.Bool {
		t.Error("IsDemo() = true, want false (real account: IsDemo must reflect context, not hardcoded)")
	}
}

// TestVM_Audit_IsTradeAllowedReturnsTrue verifies that IsTradeAllowed()
// returns constant true. VM-API-TRUTH-2: the VM host always allows trading;
// broker-side trade permission is checked at order submission, not here.
func TestVM_Audit_IsTradeAllowedReturnsTrue(t *testing.T) {
	vm := newSignalTestVM()
	val, err := builtinIsTradeAllowed(vm, nil)
	if err != nil {
		t.Fatalf("IsTradeAllowed: %v", err)
	}
	if !val.Bool {
		t.Error("IsTradeAllowed() = false, want true (host always allows trading)")
	}
}

// TestVM_Audit_AccountNumberFromContext verifies that AccountNumber() reads
// from the context's AccountInfo.Login, not a hardcoded value.
// VM-API-TRUTH-2: AccountNumber must reflect the actual account login.
func TestVM_Audit_AccountNumberFromContext(t *testing.T) {
	vm := newSignalTestVM()
	vm.ctx = &accountTestContext{login: 12345}
	val, err := builtinAccountNumber(vm, nil)
	if err != nil {
		t.Fatalf("AccountNumber: %v", err)
	}
	if val.ToInt() != 12345 {
		t.Errorf("AccountNumber() = %d, want 12345 (should read from context Login)", val.ToInt())
	}
}
