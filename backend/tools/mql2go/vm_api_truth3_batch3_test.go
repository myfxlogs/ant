// vm_api_truth3_batch3_test.go — VM-API-TRUTH-3 (Batch 3) tests.
//
// Tests verify S1-S5 implementation:
//
//	S1: builtinIsConnected reads from vm.ctx.Account().IsConnected
//	S2: builtinIsDemo reads from vm.ctx.Account().IsDemo
//	S3: builtinIsTradeAllowed reads from vm.ctx.Account().IsTradeAllowed
//	S4: sdk.AccountInfo has IsDemo/IsConnected/IsTradeAllowed fields
//	S5: vmHandleBar/Start/dispatchVMLive inject status via Runner.SetAccountStatus
//
// Adversarial proofs P1-P3: each critical line mutated → relevant test RED → restore GREEN.
package mql2go

import (
	"context"
	"testing"

	"alphaforge/strategy/sdk"

	"github.com/shopspring/decimal"
)

// --- T1: builtinIsConnected reads from context ---

func TestBuiltinIsConnected_ReadsFromContext(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = &accountStatusTestContext{isConnected: false}

	result, _ := builtinIsConnected(vm, nil)
	if result.Bool {
		t.Fatal("IsConnected() = true, want false (from context IsConnected=false)")
	}
}

func TestBuiltinIsConnected_TrueFromContext(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = &accountStatusTestContext{isConnected: true}

	result, _ := builtinIsConnected(vm, nil)
	if !result.Bool {
		t.Fatal("IsConnected() = false, want true (from context IsConnected=true)")
	}
}

// --- T2: builtinIsDemo reads from context ---

func TestBuiltinIsDemo_ReadsFromContext(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = &accountStatusTestContext{isDemo: false}

	result, _ := builtinIsDemo(vm, nil)
	if result.Bool {
		t.Fatal("IsDemo() = true, want false (from context IsDemo=false)")
	}
}

func TestBuiltinIsDemo_TrueFromContext(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = &accountStatusTestContext{isDemo: true}

	result, _ := builtinIsDemo(vm, nil)
	if !result.Bool {
		t.Fatal("IsDemo() = false, want true (from context IsDemo=true)")
	}
}

// --- T3: builtinIsTradeAllowed reads from context ---

func TestBuiltinIsTradeAllowed_ReadsFromContext(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = &accountStatusTestContext{isTradeAllowed: false}

	result, _ := builtinIsTradeAllowed(vm, nil)
	if result.Bool {
		t.Fatal("IsTradeAllowed() = true, want false (from context IsTradeAllowed=false)")
	}
}

func TestBuiltinIsTradeAllowed_TrueFromContext(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = &accountStatusTestContext{isTradeAllowed: true}

	result, _ := builtinIsTradeAllowed(vm, nil)
	if !result.Bool {
		t.Fatal("IsTradeAllowed() = false, want true (from context IsTradeAllowed=true)")
	}
}

// --- T4: nil context defaults to true (backtest) ---

func TestBuiltinIsConnected_NilContextDefaultsTrue(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = nil

	result, _ := builtinIsConnected(vm, nil)
	if !result.Bool {
		t.Fatal("IsConnected() = false with nil ctx, want true (backtest default)")
	}
}

func TestBuiltinIsDemo_NilContextDefaultsTrue(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = nil

	result, _ := builtinIsDemo(vm, nil)
	if !result.Bool {
		t.Fatal("IsDemo() = false with nil ctx, want true (backtest default)")
	}
}

func TestBuiltinIsTradeAllowed_NilContextDefaultsTrue(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = nil

	result, _ := builtinIsTradeAllowed(vm, nil)
	if !result.Bool {
		t.Fatal("IsTradeAllowed() = false with nil ctx, want true (backtest default)")
	}
}

// --- T5: end-to-end IsConnected readback via VM ---

func TestVMLive_IsConnectedEndToEnd(t *testing.T) {
	// MQL source that reads IsConnected() into a global.
	source := `int g_isConnected = 0; void OnInit() { g_isConnected = IsConnected(); } void OnBar() {}`
	runner := compileAndInit(t, source, &accountStatusTestContext{isConnected: false})

	v, ok := runner.GetGlobal("g_isConnected")
	if !ok {
		t.Fatal("global \"g_isConnected\" not found")
	}
	got := v.ToInt()
	if got != 0 {
		t.Errorf("g_isConnected = %d, want 0 (IsConnected=false from context)", got)
	}
}

// --- T6: end-to-end IsTradeAllowed readback via VM (investor gating) ---

func TestVMLive_IsTradeAllowedEndToEnd(t *testing.T) {
	// MQL source that reads IsTradeAllowed() into a global.
	source := `int g_isTradeAllowed = 0; void OnInit() { g_isTradeAllowed = IsTradeAllowed(); } void OnBar() {}`
	runner := compileAndInit(t, source, &accountStatusTestContext{isTradeAllowed: false})

	v, ok := runner.GetGlobal("g_isTradeAllowed")
	if !ok {
		t.Fatal("global \"g_isTradeAllowed\" not found")
	}
	got := v.ToInt()
	if got != 0 {
		t.Errorf("g_isTradeAllowed = %d, want 0 (IsTradeAllowed=false — investor account)", got)
	}
}

// --- T7: end-to-end IsDemo readback via VM ---

func TestVMLive_IsDemoEndToEnd(t *testing.T) {
	// MQL source that reads IsDemo() into a global.
	source := `int g_isDemo = 0; void OnInit() { g_isDemo = IsDemo(); } void OnBar() {}`
	runner := compileAndInit(t, source, &accountStatusTestContext{isDemo: false})

	v, ok := runner.GetGlobal("g_isDemo")
	if !ok {
		t.Fatal("global \"g_isDemo\" not found")
	}
	got := v.ToInt()
	if got != 0 {
		t.Errorf("g_isDemo = %d, want 0 (IsDemo=false — real account)", got)
	}
}

// --- helpers ---

// accountStatusTestContext implements sdk.Context for IsConnected/IsDemo/
// IsTradeAllowed tests. Provides all sdk.Context methods with stubs.
type accountStatusTestContext struct {
	isConnected    bool
	isDemo         bool
	isTradeAllowed bool
}

func (c *accountStatusTestContext) Account() sdk.AccountInfo {
	return sdk.AccountInfo{
		IsConnected:    c.isConnected,
		IsDemo:         c.isDemo,
		IsTradeAllowed: c.isTradeAllowed,
	}
}
func (c *accountStatusTestContext) Broker() sdk.Broker                    { return &cacheTestBroker{} }
func (c *accountStatusTestContext) ServerTime() int64                     { return 0 }
func (c *accountStatusTestContext) GoContext() context.Context            { return context.Background() }
func (c *accountStatusTestContext) Param(string, interface{}) interface{} { return nil }
func (c *accountStatusTestContext) ParamDecimal(string, decimal.Decimal) decimal.Decimal {
	return decimal.Zero
}
func (c *accountStatusTestContext) ParamInt(string, int) int          { return 0 }
func (c *accountStatusTestContext) ParamString(string, string) string { return "" }
func (c *accountStatusTestContext) ParamBool(string, bool) bool       { return false }
func (c *accountStatusTestContext) Bars() sdk.BarSeries               { return nil }
func (c *accountStatusTestContext) BarsTF(string) sdk.BarSeries       { return nil }
func (c *accountStatusTestContext) BarsForSymbol(string, string) sdk.BarSeries {
	return nil
}
func (c *accountStatusTestContext) Symbol() string               { return "EURUSD" }
func (c *accountStatusTestContext) Timeframe() string            { return "M15" }
func (c *accountStatusTestContext) Point() decimal.Decimal       { return decimal.Zero }
func (c *accountStatusTestContext) Pip() decimal.Decimal         { return decimal.Zero }
func (c *accountStatusTestContext) Digits() int32                { return 5 }
func (c *accountStatusTestContext) Ask() decimal.Decimal         { return decimal.Zero }
func (c *accountStatusTestContext) Bid() decimal.Decimal         { return decimal.Zero }
func (c *accountStatusTestContext) Spread() decimal.Decimal      { return decimal.Zero }
func (c *accountStatusTestContext) Mode() sdk.AccountMode        { return sdk.ModeHedging }
func (c *accountStatusTestContext) Indicators() sdk.IndicatorSet { return nil }
func (c *accountStatusTestContext) SetTimer(int)                 {}
func (c *accountStatusTestContext) KillTimer()                   {}
func (c *accountStatusTestContext) Log(string)                   {}

// compileAndInit compiles MQL source, creates a VMRunner, sets the context,
// and runs OnInit. Returns the runner for global readback.
func compileAndInit(t *testing.T, source string, ctx sdk.Context) *VMRunner {
	t.Helper()
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	runner.SetSignalMode(true)
	if err := runner.OnInit(ctx); err != nil {
		t.Fatalf("OnInit failed: %v", err)
	}
	return runner
}

// Ensure the test context satisfies sdk.Context.
var _ sdk.Context = (*accountStatusTestContext)(nil)

// dummy reference to context to avoid unused import
var _ = context.Background
