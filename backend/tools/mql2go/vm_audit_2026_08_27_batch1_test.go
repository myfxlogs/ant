package mql2go

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// vm_audit_2026_08_27_batch1_test.go — VM-AUDIT-2026-08-27 批次 1 对抗测试 (BUG-2).
//
// Tests verify:
//   - VM-AUDIT-2026-08-27-2 (BUG-2): runEvent resets fatalError between events
//     so a single builtin error doesn't permanently stop the strategy.
//
// Adversarial proof: delete the `vm.fatalError = ""` line in runEvent →
// second RunOnBar returns error immediately (runLoop top check) → RED.
// Restore → second RunOnBar executes normally → GREEN.

// TestVM_FatalErrorResetBetweenEvents verifies BUG-2 fix:
// runEvent resets vm.fatalError so a prior event's fatal error doesn't
// block subsequent events. The VM instance is reused in VMLiveSession
// (live mode), so without this reset, one builtin error permanently
// stops the strategy.
//
// Test design:
//  1. EA has a global counter g_called. First OnBar: g_called==1 → calls
//     iADX with MODE_PLUSDI → fatalError set → runLoop returns error.
//     g_called was incremented BEFORE the iADX call, so it persists as 1.
//  2. Second OnBar (with fatalError reset): g_called==2 → if-condition
//     false → skips iADX → g_result=42 → success. GREEN.
//  3. Mutation (delete `vm.fatalError = ""`): second OnBar → runLoop top
//     check sees fatalError still set from first event → returns error
//     immediately → g_called stays 1. RED.
//
// Note: iADX:MODE_PLUSDI is used as the fatalError trigger because it's a
// builtin Go error path (vm_builtin_indicators.go:197 sets fatalError
// directly), not a broker error. In signal mode, trade builtins don't call
// the broker, so broker errors can't trigger fatalError.
func TestVM_FatalErrorResetBetweenEvents(t *testing.T) {
	src := `
int g_called = 0;
double g_result = 0.0;

int OnInit() { return 0; }

void OnBar()
{
    g_called = g_called + 1;
    if (g_called == 1) {
        g_result = iADX(NULL, 0, 14, PRICE_CLOSE, MODE_PLUSDI, 0);
    }
    g_result = 42.0;
}
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	// Access the VM directly (same package) to call RunOnBar twice on the
	// same VM instance, simulating VMLiveSession's reuse pattern.
	vm := vmRunner.vm
	vm.SetSignalMode(true) // live mode — VM instance is reused between events
	vm.SetContext(&tsTestContext{bars: sdk.BarsToSlice(makeFailClosedBars(3))})

	// Initialize globals via OnInit.
	if err := vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit failed: %v", err)
	}

	// --- First OnBar: triggers fatalError via iADX:MODE_PLUSDI ---
	err = vm.RunOnBar(context.Background())
	if err == nil {
		t.Fatal("first OnBar should fail with iADX:MODE_PLUSDI fatal error")
	}
	if !strings.Contains(err.Error(), "MODE_PLUSDI") {
		t.Fatalf("expected MODE_PLUSDI error on first OnBar, got: %v", err)
	}

	// g_called was incremented before the iADX call, so it should be 1.
	called := getGlobalInt(t, vmRunner, "g_called")
	if called != 1 {
		t.Fatalf("g_called = %d after first OnBar, want 1 (increment before iADX)", called)
	}

	// --- Second OnBar: fatalError should be reset → normal execution ---
	err = vm.RunOnBar(context.Background())
	if err != nil {
		t.Fatalf("second OnBar should succeed (fatalError reset by runEvent), got: %v", err)
	}

	// g_called should be 2 (second event executed fully).
	called = getGlobalInt(t, vmRunner, "g_called")
	if called != 2 {
		t.Fatalf("g_called = %d after second OnBar, want 2 (second event executed)", called)
	}

	// g_result should be 42.0 (the assignment after the if-block completed).
	result := getGlobalDecimal(t, vmRunner, "g_result")
	if !result.Equals(decimal.NewFromInt(42)) {
		t.Fatalf("g_result = %s after second OnBar, want 42 (assignment completed)", result)
	}
}
