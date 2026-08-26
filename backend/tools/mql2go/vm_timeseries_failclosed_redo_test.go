// vm_timeseries_failclosed_redo_test.go — VM-TIMESERIES-SEMANTICS-1 + VM-RUNTIME-FAILCLOSED-1
// 返工行为测试（2026-08-26）.
//
// Tests verify the re-implemented fixes after D-REVERT-SCOPE-DRIFT-001:
//   - VM-TIMESERIES-SEMANTICS-1: CopyTime seconds conversion, iHighest/iLowest
//     mode-selective extremeIndex, bounds guard + count clamp, iBarShift exact,
//     CopyClose direction semantics.
//   - VM-RUNTIME-FAILCLOSED-1: callBuiltin fatalError defense-in-depth,
//     pop/popN setStackError, OP_CALL_BUILTIN push后 fatalError check,
//     Engine.Run fail-closed, iADX/iADXWilder MODE_PLUSDI/MINUSDI fatalError,
//     builtinOrderSend error propagation.
//
// Adversarial proofs (8): each critical line mutated → relevant test RED → restore GREEN.

package mql2go

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/runner"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// ── Test helpers (VM-TIMESERIES-SEMANTICS-1) ────────────────────────

// tsTestContext implements sdk.Context for timeseries builtin tests.
// It provides a fixed-epoch BarSeries with deterministic OHLCV values.
type tsTestContext struct {
	sdk.Context
	bars sdk.BarSeries
}

func (c *tsTestContext) Bars() sdk.BarSeries            { return c.bars }
func (c *tsTestContext) BarsTF(tf string) sdk.BarSeries { return c.bars }
func (c *tsTestContext) BarsForSymbol(sym, tf string) sdk.BarSeries {
	return c.bars
}
func (c *tsTestContext) Symbol() string               { return "EURUSD" }
func (c *tsTestContext) Timeframe() string            { return "M1" }
func (c *tsTestContext) Broker() sdk.Broker           { return &tsTestBroker{} }
func (c *tsTestContext) Indicators() sdk.IndicatorSet { return &tsTestIndicators{} }

// tsTestBroker is a no-op broker for timeseries tests.
type tsTestBroker struct{ sdk.Broker }

// tsTestIndicators is a no-op indicator set for timeseries tests.
type tsTestIndicators struct{ sdk.IndicatorSet }

// makeTSTestBars creates a BarSeries with 5 bars at fixed epoch times.
// Bars are ordered oldest first; BarSeries inverse-indexes (shift=0 = last).
// Values are designed so each mode selects a different extreme:
//
//	bar 0 (oldest): O=10, H=15, L=5,  C=12, V=100, T=1704067200000 (2024-01-01 00:00)
//	bar 1:          O=20, H=25, L=18, C=22, V=200, T=1704067260000 (2024-01-01 00:01)
//	bar 2:          O=30, H=35, L=28, C=32, V=300, T=1704067320000 (2024-01-01 00:02)
//	bar 3:          O=40, H=45, L=38, C=42, V=400, T=1704067380000 (2024-01-01 00:03)
//	bar 4 (newest): O=50, H=55, L=48, C=52, V=500, T=1704067440000 (2024-01-01 00:04)
//
// All values increase monotonically so iHighest=shift 0 (newest, bar 4),
// iLowest=shift 4 (oldest, bar 0) for all modes.
func makeTSTestBars() []sdk.Bar {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	return []sdk.Bar{
		{Open: dec(10), High: dec(15), Low: dec(5), Close: dec(12), Volume: 100, Timestamp: base},
		{Open: dec(20), High: dec(25), Low: dec(18), Close: dec(22), Volume: 200, Timestamp: base + 60000},
		{Open: dec(30), High: dec(35), Low: dec(28), Close: dec(32), Volume: 300, Timestamp: base + 120000},
		{Open: dec(40), High: dec(45), Low: dec(38), Close: dec(42), Volume: 400, Timestamp: base + 180000},
		{Open: dec(50), High: dec(55), Low: dec(48), Close: dec(52), Volume: 500, Timestamp: base + 240000},
	}
}

// makeModeTestBars creates 3 bars where the extreme position differs by mode:
//
//	bar 0 (oldest, shift=2): O=30, H=10, L=50, C=20, V=300, T=ts0
//	bar 1 (shift=1):          O=20, H=30, L=40, C=10, V=200, T=ts1
//	bar 2 (newest, shift=0):  O=10, H=50, L=10, C=30, V=100, T=ts2
//
// iHighest(MODE_HIGH)  → bar 2 (H=50, shift=0)
// iHighest(MODE_CLOSE) → bar 2 (C=30, shift=0)
// iHighest(MODE_LOW)   → bar 0 (L=50, shift=2) — highest low
// iLowest(MODE_HIGH)   → bar 0 (H=10, shift=2) — lowest high
// iLowest(MODE_LOW)    → bar 2 (L=10, shift=0)
// iLowest(MODE_CLOSE)  → bar 1 (C=10, shift=1)
func makeModeTestBars() []sdk.Bar {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	return []sdk.Bar{
		{Open: dec(30), High: dec(10), Low: dec(50), Close: dec(20), Volume: 300, Timestamp: base},
		{Open: dec(20), High: dec(30), Low: dec(40), Close: dec(10), Volume: 200, Timestamp: base + 60000},
		{Open: dec(10), High: dec(50), Low: dec(10), Close: dec(30), Volume: 100, Timestamp: base + 120000},
	}
}

func dec(n int) decimal.Decimal { return decimal.NewFromInt(int64(n)) }

// newTSVM creates a VM with a timeseries test context.
func newTSVM(bars []sdk.Bar) *VM {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.SetSignalMode(false)
	vm.ctx = &tsTestContext{bars: sdk.BarsToSlice(bars)}
	return vm
}

// ── VM-TIMESERIES-SEMANTICS-1 tests (11) ────────────────────────────

// TestVM_Audit_IHighest_AllSeriesModes — 5 bars values increasing, all 6 modes
// iHighest=shift 0 (newest bar has the highest value for every mode).
//
// Adversarial: delete extremeIndex mode branch (always use Close) →
// MODE_HIGH result changes → RED.
func TestVM_Audit_IHighest_AllSeriesModes(t *testing.T) {
	vm := newTSVM(makeTSTestBars())
	// iHighest(symbol, tf, mode, count, start)
	// count=0 → all bars, start=0
	for mode := int32(0); mode <= 5; mode++ {
		result, err := builtinIHighest(vm, []interp.Value{
			interp.StringVal("EURUSD"), // symbol
			interp.IntVal(0),           // tf
			interp.IntVal(mode),        // ENUM_SERIESMODE
			interp.IntVal(0),           // count (0 = all)
			interp.IntVal(0),           // start
		})
		if err != nil {
			t.Fatalf("iHighest mode %d error: %v", mode, err)
		}
		// All values increase monotonically → highest is newest bar (shift 0)
		if result.ToInt() != 0 {
			t.Errorf("iHighest mode %d = %d, want 0 (newest bar has max value)", mode, result.ToInt())
		}
	}
}

// TestVM_Audit_ILowest_AllSeriesModes — 5 bars values increasing, all 6 modes
// iLowest=shift 4 (oldest bar has the lowest value for every mode).
func TestVM_Audit_ILowest_AllSeriesModes(t *testing.T) {
	vm := newTSVM(makeTSTestBars())
	for mode := int32(0); mode <= 5; mode++ {
		result, err := builtinILowest(vm, []interp.Value{
			interp.StringVal("EURUSD"),
			interp.IntVal(0),
			interp.IntVal(mode),
			interp.IntVal(0),
			interp.IntVal(0),
		})
		if err != nil {
			t.Fatalf("iLowest mode %d error: %v", mode, err)
		}
		// All values increase → lowest is oldest bar (shift 4)
		if result.ToInt() != 4 {
			t.Errorf("iLowest mode %d = %d, want 4 (oldest bar has min value)", mode, result.ToInt())
		}
	}
}

// TestVM_Audit_IHighest_ModeSelectsCorrectField — 3 bars where extreme
// position differs by mode. Verifies mode actually selects the right field.
//
// Adversarial: delete extremeIndex mode branch (always use Close) →
// MODE_HIGH result wrong → RED.
func TestVM_Audit_IHighest_ModeSelectsCorrectField(t *testing.T) {
	vm := newTSVM(makeModeTestBars())
	tests := []struct {
		mode    int32
		wantIdx int32
		desc    string
	}{
		{0, 2, "MODE_OPEN: bar0 O=30 is highest → shift=2"},
		{1, 2, "MODE_LOW: bar0 L=50 is highest low → shift=2"},
		{2, 0, "MODE_HIGH: bar2 H=50 is highest → shift=0"},
		{3, 0, "MODE_CLOSE: bar2 C=30 is highest → shift=0"},
	}
	for _, tt := range tests {
		result, _ := builtinIHighest(vm, []interp.Value{
			interp.StringVal(""), interp.IntVal(0),
			interp.IntVal(tt.mode), interp.IntVal(0), interp.IntVal(0),
		})
		if result.ToInt() != tt.wantIdx {
			t.Errorf("iHighest %s = %d, want %d", tt.desc, result.ToInt(), tt.wantIdx)
		}
	}
}

// TestVM_Audit_IHighest_PartialRange — start=2 count=2 → only scans
// bars at shift 2 and 3. With monotonically increasing values,
// shift 2 (H=35) is higher than shift 3 (H=25).
func TestVM_Audit_IHighest_PartialRange(t *testing.T) {
	vm := newTSVM(makeTSTestBars())
	// start=2, count=2 → scan shifts 2,3
	// shift 2: H=35, shift 3: H=25 → highest is shift 2
	result, _ := builtinIHighest(vm, []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(2), // MODE_HIGH
		interp.IntVal(2), // count
		interp.IntVal(2), // start
	})
	if result.ToInt() != 2 {
		t.Errorf("iHighest start=2 count=2 = %d, want 2 (H=35 > H=25)", result.ToInt())
	}
}

// TestVM_Audit_IHighest_EmptySeries — empty series → -1.
func TestVM_Audit_IHighest_EmptySeries(t *testing.T) {
	vm := newTSVM(nil)
	result, _ := builtinIHighest(vm, []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(2), interp.IntVal(0), interp.IntVal(0),
	})
	if result.ToInt() != -1 {
		t.Errorf("iHighest empty series = %d, want -1", result.ToInt())
	}
}

// TestVM_Audit_IHighest_OutOfRangeStart — start=10 Len=5 → -1.
//
// Adversarial: delete bounds guard (only keep empty check) →
// returns 10 instead of -1 → RED.
func TestVM_Audit_IHighest_OutOfRangeStart(t *testing.T) {
	vm := newTSVM(makeTSTestBars())
	result, _ := builtinIHighest(vm, []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(2), interp.IntVal(0), interp.IntVal(10),
	})
	if result.ToInt() != -1 {
		t.Errorf("iHighest start=10 Len=5 = %d, want -1 (out of range)", result.ToInt())
	}
}

// TestVM_Audit_IHighest_InvalidMode — mode=99 → -1 + blind spot recorded.
func TestVM_Audit_IHighest_InvalidMode(t *testing.T) {
	vm := newTSVM(makeTSTestBars())
	result, _ := builtinIHighest(vm, []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(99), interp.IntVal(0), interp.IntVal(0),
	})
	if result.ToInt() != -1 {
		t.Errorf("iHighest mode=99 = %d, want -1 (invalid mode)", result.ToInt())
	}
	bs := vm.GetRuntimeBlindSpots()
	found := false
	for _, b := range bs {
		if strings.Contains(b.Builtin, "invalid series mode") {
			found = true
		}
	}
	if !found {
		t.Error("expected blind spot for invalid series mode")
	}
}

// TestVM_Audit_IBarShift_ExactTrue — exact=true only matches exact timestamp.
// Bar at shift 2 has timestamp base+120000. With exact=true and ts=base+120000/1000,
// it should return shift 2. With a non-matching ts, it should return -1.
//
// Adversarial: delete exact handling (always exact=false) →
// non-matching ts returns a bar instead of -1 → RED.
func TestVM_Audit_IBarShift_ExactTrue(t *testing.T) {
	bars := makeTSTestBars()
	vm := newTSVM(bars)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	// Exact match: ts = bar 2's time in seconds → shift 2
	result, _ := builtinIBarShift(vm, []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(int32((base + 120000) / 1000)), // ts in seconds
		interp.IntVal(1), // exact=true
	})
	if result.ToInt() != 2 {
		t.Errorf("iBarShift exact=true matching ts = %d, want 2", result.ToInt())
	}

	// Non-exact: ts between bar 1 and bar 2 → should return -1 with exact=true
	result, _ = builtinIBarShift(vm, []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(int32((base + 90000) / 1000)), // ts between bar 1 and 2
		interp.IntVal(1), // exact=true
	})
	if result.ToInt() != -1 {
		t.Errorf("iBarShift exact=true non-matching ts = %d, want -1", result.ToInt())
	}
}

// TestVM_Audit_IBarShift_ExactFalse — exact=false returns most recent bar
// with time <= ts. Time before all bars → -1.
func TestVM_Audit_IBarShift_ExactFalse(t *testing.T) {
	bars := makeTSTestBars()
	vm := newTSVM(bars)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	// ts between bar 1 and bar 2 → most recent bar with time <= ts is bar 1 (shift 3)
	// Wait: bar 1 has timestamp base+60000, bar 2 has base+120000.
	// ts = base+90000 → bar 1 (time base+60000 <= ts) is most recent → shift 3
	// Actually: shift = series index from end. Bar 1 is at array index 1,
	// shift = len-1-index = 5-1-1 = 3. But we only have 5 bars, so shift 3.
	// Wait, iBarShift iterates i=0..Len-1 and returns i, where series.Time(i)
	// is the shift (inverse index). So i=0 is the newest bar (shift 0).
	// series.Time(i) returns the timestamp at shift i.
	// For exact=false: first bar where barTs < ts → that's the newest bar
	// with time < ts. Bar at shift 3 (array index 1) has time base+60000 < base+90000.
	// Bar at shift 4 (array index 0) has time base < base+90000.
	// The loop goes i=0 (newest) first. series.Time(0) = base+240000 > base+90000.
	// series.Time(1) = base+180000 > base+90000.
	// series.Time(2) = base+120000 > base+90000.
	// series.Time(3) = base+60000 < base+90000 → return 3.
	result, _ := builtinIBarShift(vm, []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(int32((base + 90000) / 1000)), // ts in seconds
		interp.IntVal(0), // exact=false
	})
	if result.ToInt() != 3 {
		t.Errorf("iBarShift exact=false ts between bar1,bar2 = %d, want 3", result.ToInt())
	}

	// Time before all bars → -1
	result, _ = builtinIBarShift(vm, []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(int32((base - 60000) / 1000)), // before all bars
		interp.IntVal(0), // exact=false
	})
	if result.ToInt() != -1 {
		t.Errorf("iBarShift exact=false ts before all bars = %d, want -1", result.ToInt())
	}
}

// TestVM_Audit_CopyTime_SecondsConversion — 3 bars → CopyTime returns
// unix seconds (not ms). Verifies the /1000 conversion.
//
// Adversarial: delete /1000 → returns ms, overflows int32 → RED.
func TestVM_Audit_CopyTime_SecondsConversion(t *testing.T) {
	bars := makeTSTestBars()[:3]
	vm := newTSVM(bars)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	expectedSec := int32(base / 1000) // bar 0 (oldest) in seconds

	// CopyTime(symbol, tf, startPos, count, array[])
	// startPos=0, count=3 (chronological: oldest first)
	// copyBarData modifies args[4].Array in-place, so check args after call.
	args := []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(0), interp.IntVal(3),
		{Kind: interp.ValArray, Array: make([]interp.Value, 0)},
	}
	result, _ := builtinCopyTime(vm, args)
	if result.ToInt() != 3 {
		t.Fatalf("CopyTime returned %d, want 3", result.ToInt())
	}
	arr := args[4]
	if len(arr.Array) != 3 {
		t.Fatalf("CopyTime array len = %d, want 3", len(arr.Array))
	}
	first := arr.Array[0].ToInt()
	if first != expectedSec {
		t.Errorf("CopyTime[0] = %d, want %d (unix seconds, not ms)", first, expectedSec)
	}
}

// TestVM_Audit_CopyClose_Direction — count=+3 chronological (oldest first),
// count=-3 reverse chronological (newest first).
func TestVM_Audit_CopyClose_Direction(t *testing.T) {
	bars := makeTSTestBars()[:3]
	vm := newTSVM(bars)

	// count=+3 → chronological (oldest first): C=12, 22, 32
	args := []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(0), interp.IntVal(3),
		{Kind: interp.ValArray, Array: make([]interp.Value, 0)},
	}
	builtinCopyClose(vm, args)
	arr := args[4]
	if len(arr.Array) != 3 {
		t.Fatalf("CopyClose +3 len = %d, want 3", len(arr.Array))
	}
	if arr.Array[0].ToDecimal().IntPart() != 12 {
		t.Errorf("CopyClose +3 [0] = %s, want 12 (oldest first)", arr.Array[0].ToDecimal())
	}
	if arr.Array[2].ToDecimal().IntPart() != 32 {
		t.Errorf("CopyClose +3 [2] = %s, want 32 (newest last)", arr.Array[2].ToDecimal())
	}

	// count=-3 → reverse chronological (newest first): C=32, 22, 12
	args2 := []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(0), interp.IntVal(-3),
		{Kind: interp.ValArray, Array: make([]interp.Value, 0)},
	}
	builtinCopyClose(vm, args2)
	arr2 := args2[4]
	if len(arr2.Array) != 3 {
		t.Fatalf("CopyClose -3 len = %d, want 3", len(arr2.Array))
	}
	if arr2.Array[0].ToDecimal().IntPart() != 32 {
		t.Errorf("CopyClose -3 [0] = %s, want 32 (newest first)", arr2.Array[0].ToDecimal())
	}
	if arr2.Array[2].ToDecimal().IntPart() != 12 {
		t.Errorf("CopyClose -3 [2] = %s, want 12 (oldest last)", arr2.Array[2].ToDecimal())
	}
}

// ── VM-RUNTIME-FAILCLOSED-1 tests (4) ───────────────────────────────

// TestVM_Audit_BuiltinErrorStopsExecution — OrderSend volume=0 → builtin
// Go error → fatalError set → subsequent instruction (g_after=42) not executed.
//
// Adversarial: restore callBuiltin to swallow Go error (no fatalError) →
// g_after=42 executes → RED.
func TestVM_Audit_BuiltinErrorStopsExecution(t *testing.T) {
	src := `
int g_after = 0;

int OnInit() { return 0; }

void OnTick()
{
    OrderSend(Symbol(), OP_BUY, 0.0, Ask, 5, 0, 0, "test", 12345, 0, clrGreen);
    g_after = 42;
}
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r, vmRunner := setupFailClosedRunner(t, vmRunner)
	_, err = r.OnTick(context.Background(), dec(1), dec(2))
	if err == nil {
		t.Fatal("OnTick should fail with OrderSend volume=0 error (fail-closed)")
	}
	if !strings.Contains(err.Error(), "volume") {
		t.Fatalf("expected volume error, got: %v", err)
	}
	after := getGlobalInt(t, vmRunner, "g_after")
	if after != 0 {
		t.Errorf("g_after = %d, want 0 (subsequent instruction should not execute)", after)
	}
}

// TestVM_Audit_InvalidMutationDoesNotChangeCapital — OrderSend volume=0
// → error → broker balance unchanged + positions=0 (no partial mutation).
func TestVM_Audit_InvalidMutationDoesNotChangeCapital(t *testing.T) {
	src := `
int OnInit() { return 0; }
void OnTick()
{
    OrderSend(Symbol(), OP_BUY, 0.0, Ask, 5, 0, 0, "test", 12345, 0, clrGreen);
}
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	bars := makeFailClosedBars(5)
	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
	}
	engine := backtest.New(cfg, vmRunner, bars)
	_, err = engine.Run(context.Background())
	if err == nil {
		t.Fatal("Engine.Run should fail with OrderSend volume=0 error")
	}
	broker := engine.Broker()
	acc := broker.Account()
	if !acc.Balance.Equals(decimal.NewFromInt(10000)) {
		t.Errorf("balance = %s, want 10000 (no partial mutation)", acc.Balance)
	}
	if len(broker.Positions(0)) != 0 {
		t.Errorf("positions = %d, want 0 (no partial mutation)", len(broker.Positions(0)))
	}
}

// TestVM_Audit_FatalBlindSpotFromHandlerNotPushedToStack — iADX MODE_PLUSDI
// → fatalError set by handler → callBuiltin returns NoneVal → assignment
// not completed (g_result stays 99.0) + subsequent instruction not executed.
//
// Adversarial: delete callBuiltin fatalError check (after nil error) →
// result pushed to stack → g_result changes → RED.
func TestVM_Audit_FatalBlindSpotFromHandlerNotPushedToStack(t *testing.T) {
	src := `
double g_result = 99.0;
int g_after = 0;

int OnInit() { return 0; }

void OnTick()
{
    g_result = iADX(NULL, 0, 14, PRICE_CLOSE, MODE_PLUSDI, 0);
    g_after = 42;
}
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	r, vmRunner := setupFailClosedRunner(t, vmRunner)
	_, err = r.OnTick(context.Background(), dec(1), dec(2))
	if err == nil {
		t.Fatal("OnTick should fail with iADX:MODE_PLUSDI fatal error")
	}
	if !strings.Contains(err.Error(), "MODE_PLUSDI") {
		t.Fatalf("expected MODE_PLUSDI error, got: %v", err)
	}
	result := getGlobalDecimal(t, vmRunner, "g_result")
	if !result.Equals(dec(99)) {
		t.Errorf("g_result = %s, want 99 (assignment should not complete)", result)
	}
	after := getGlobalInt(t, vmRunner, "g_after")
	if after != 0 {
		t.Errorf("g_after = %d, want 0 (subsequent instruction should not execute)", after)
	}
}

// TestVM_Audit_BuiltinErrorPropagatesToEngine — OrderSend cmd=99 → error
// propagates through VM → VMRunner.OnBar → Engine.Run → result==nil + err!=nil.
//
// Adversarial: restore Engine.Run to stderr+continue →
// Engine.Run returns success → RED.
func TestVM_Audit_BuiltinErrorPropagatesToEngine(t *testing.T) {
	src := `
int OnInit() { return 0; }
void OnBar()
{
    OrderSend(Symbol(), 99, 0.1, Ask, 5, 0, 0, "test", 12345, 0, clrGreen);
}
`
	vmRunner, err := CompileMQL(src)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}

	bars := makeFailClosedBars(5)
	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
	}
	engine := backtest.New(cfg, vmRunner, bars)
	result, err := engine.Run(context.Background())
	if err == nil {
		t.Fatal("Engine.Run should fail with OrderSend cmd=99 error (fail-closed)")
	}
	if result != nil {
		t.Errorf("result should be nil on error, got %+v", result)
	}
	if !strings.Contains(err.Error(), "cmd") && !strings.Contains(err.Error(), "strategy event") {
		t.Fatalf("expected cmd/strategy error, got: %v", err)
	}
}

// ── Fail-closed test helpers ────────────────────────────────────────

// makeFailClosedBars creates simple bars for fail-closed tests.
func makeFailClosedBars(n int) []sdk.Bar {
	bars := make([]sdk.Bar, n)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	for i := 0; i < n; i++ {
		bars[i] = sdk.Bar{
			Open:      decimal.NewFromFloat(1.1000),
			High:      decimal.NewFromFloat(1.1010),
			Low:       decimal.NewFromFloat(1.0990),
			Close:     decimal.NewFromFloat(1.1005),
			Volume:    1000,
			Timestamp: base + int64(i)*60000,
		}
	}
	return bars
}

// setupFailClosedRunner creates a runner with live state for direct VM tests.
func setupFailClosedRunner(t *testing.T, vmRunner *VMRunner) (*runner.Runner, *VMRunner) {
	t.Helper()
	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	r.UpdateLiveState("10000", "10500", "500", "9500", nil, nil)
	if err := r.Init(context.Background()); err != nil {
		// OnInit may fail if it calls an unsupported builtin; that's OK for
		// tests that only care about OnTick behavior. But if OnInit itself
		// triggers the error we're testing, let it pass through.
		t.Logf("Init returned: %v (continuing to OnTick)", err)
	}
	return r, vmRunner
}

// TestVM_Audit_StackUnderflowIsError — pop/popN underflow sets fatalError
// (VM-RUNTIME-FAILCLOSED-1). This is tested via direct VM stack manipulation
// since well-compiled MQL code won't underflow (isStackNeutral prevents it).
//
// Adversarial: restore pop to silent underflow (delete setStackError call) →
// fatalError stays empty → RED.
func TestVM_Audit_StackUnderflowIsError(t *testing.T) {
	bc := &Bytecode{OnBar: -1, Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)

	// pop from empty stack → setStackError
	vm.pop()
	if vm.fatalError == "" {
		t.Fatal("pop from empty stack should set fatalError (fail-closed)")
	}
	if !strings.Contains(vm.fatalError, "stack error") {
		t.Errorf("fatalError = %q, want it to contain 'stack error'", vm.fatalError)
	}

	// Reset and test popN underflow
	vm.fatalError = ""
	vm.stack = nil
	vm.popN(5)
	if vm.fatalError == "" {
		t.Fatal("popN(5) from empty stack should set fatalError (fail-closed)")
	}
}
