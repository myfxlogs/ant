package mql2go

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go/interp"
)

// ── VM-TIMESERIES-SEMANTICS-1 behavior tests ─────────────────────────
//
// All tests use fixed epoch timestamps (time.Date(2024,1,1,...,time.UTC)),
// never time.Now(), per spec §10 + AGENTS.md determinism rules.

// tsBars builds N bars with 1-minute spacing starting at baseEpoch.
// OHLCV values are set so each bar is distinguishable:
//
//	bar i (chronological): Open=i+10, High=i+20, Low=i+1, Close=i+15, Volume=(i+1)*100
//
// In MQL inverse indexing: shift=0 = last chronological bar (newest).
func tsBars(n int) []sdk.Bar {
	bars := make([]sdk.Bar, n)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range bars {
		bars[i] = sdk.Bar{
			Open:      decimal.NewFromInt(int64(i + 10)),
			High:      decimal.NewFromInt(int64(i + 20)),
			Low:       decimal.NewFromInt(int64(i + 1)),
			Close:     decimal.NewFromInt(int64(i + 15)),
			Volume:    int64(i+1) * 100,
			Timestamp: base.Add(time.Duration(i) * time.Minute).UnixMilli(),
		}
	}
	return bars
}

// tsVM creates a VM with a BarSeries from tsBars(n) and no broker.
func tsVM(n int) *VM {
	bc := &Bytecode{Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = &auditContext{
		bars:   sdk.BarsToSlice(tsBars(n)),
		symbol: "EURUSD",
		tf:     "M1",
	}
	return vm
}

// MQL series mode constants.
const (
	tsModeOpen   = int32(0)
	tsModeLow    = int32(1)
	tsModeHigh   = int32(2)
	tsModeClose  = int32(3)
	tsModeVolume = int32(4)
	tsModeTime   = int32(5)
)

// TestVM_Audit_IHighest_AllSeriesModes verifies iHighest returns the correct
// bar shift for each ENUM_SERIESMODE, using fixed-epoch bars with distinct
// OHLCV values. With 5 bars (chronological 0..4, MQL shift 4..0):
//   - MODE_OPEN:   highest open=14  at chronological 4 → MQL shift 0
//   - MODE_LOW:    highest low=5    at chronological 4 → MQL shift 0
//   - MODE_HIGH:   highest high=24  at chronological 4 → MQL shift 0
//   - MODE_CLOSE:  highest close=19 at chronological 4 → MQL shift 0
//   - MODE_VOLUME: highest vol=500  at chronological 4 → MQL shift 0
//   - MODE_TIME:   highest time     at chronological 4 → MQL shift 0
//
// All modes should return shift 0 (the latest bar has the highest values
// since values increase with chronological index).
func TestVM_Audit_IHighest_AllSeriesModes(t *testing.T) {
	vm := tsVM(5)
	modes := []int32{tsModeOpen, tsModeLow, tsModeHigh, tsModeClose, tsModeVolume, tsModeTime}
	for _, mode := range modes {
		args := []interp.Value{
			interp.StringVal(""), // symbol (primary)
			interp.IntVal(0),     // timeframe (primary)
			interp.IntVal(mode),  // type
			interp.IntVal(5),     // count
			interp.IntVal(0),     // start
		}
		got, err := builtinIHighest(vm, args)
		if err != nil {
			t.Fatalf("iHighest mode=%d: %v", mode, err)
		}
		if got.ToInt() != 0 {
			t.Errorf("iHighest mode=%d: shift=%d, want 0 (latest bar has highest values)", mode, got.ToInt())
		}
	}
}

// TestVM_Audit_ILowest_AllSeriesModes verifies iLowest returns the correct
// bar shift for each ENUM_SERIESMODE. With 5 bars (chronological 0..4):
//   - MODE_OPEN:   lowest open=10  at chronological 0 → MQL shift 4
//   - MODE_LOW:    lowest low=1    at chronological 0 → MQL shift 4
//   - MODE_HIGH:   lowest high=20  at chronological 0 → MQL shift 4
//   - MODE_CLOSE:  lowest close=15 at chronological 0 → MQL shift 4
//   - MODE_VOLUME: lowest vol=100  at chronological 0 → MQL shift 4
//   - MODE_TIME:   lowest time     at chronological 0 → MQL shift 4
func TestVM_Audit_ILowest_AllSeriesModes(t *testing.T) {
	vm := tsVM(5)
	modes := []int32{tsModeOpen, tsModeLow, tsModeHigh, tsModeClose, tsModeVolume, tsModeTime}
	for _, mode := range modes {
		args := []interp.Value{
			interp.StringVal(""),
			interp.IntVal(0),
			interp.IntVal(mode),
			interp.IntVal(5),
			interp.IntVal(0),
		}
		got, err := builtinILowest(vm, args)
		if err != nil {
			t.Fatalf("iLowest mode=%d: %v", mode, err)
		}
		if got.ToInt() != 4 {
			t.Errorf("iLowest mode=%d: shift=%d, want 4 (oldest bar has lowest values)", mode, got.ToInt())
		}
	}
}

// TestVM_Audit_IHighest_ModeSelectsCorrectField verifies that iHighest
// respects the mode parameter by using bars where the highest high and
// highest close are at different positions.
//
//	bar 0 (chronological, MQL shift 2): Open=10, High=30, Low=5,  Close=12, Vol=100
//	bar 1 (chronological, MQL shift 1): Open=11, High=25, Low=1,  Close=20, Vol=200
//	bar 2 (chronological, MQL shift 0): Open=13, High=28, Low=8,  Close=15, Vol=150
//
// iHighest(MODE_HIGH, 3, 0) → highest high=30 at bar 0 → MQL shift 2
// iHighest(MODE_CLOSE, 3, 0) → highest close=20 at bar 1 → MQL shift 1
// iLowest(MODE_LOW, 3, 0) → lowest low=1 at bar 1 → MQL shift 1
func TestVM_Audit_IHighest_ModeSelectsCorrectField(t *testing.T) {
	bars := []sdk.Bar{
		{Open: decimal.NewFromInt(10), High: decimal.NewFromInt(30), Low: decimal.NewFromInt(5), Close: decimal.NewFromInt(12), Volume: 100,
			Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()},
		{Open: decimal.NewFromInt(11), High: decimal.NewFromInt(25), Low: decimal.NewFromInt(1), Close: decimal.NewFromInt(20), Volume: 200,
			Timestamp: time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC).UnixMilli()},
		{Open: decimal.NewFromInt(13), High: decimal.NewFromInt(28), Low: decimal.NewFromInt(8), Close: decimal.NewFromInt(15), Volume: 150,
			Timestamp: time.Date(2024, 1, 1, 0, 2, 0, 0, time.UTC).UnixMilli()},
	}
	bc := &Bytecode{Builtins: make(map[string]BuiltinID)}
	vm := NewVM(bc)
	vm.ctx = &auditContext{bars: sdk.BarsToSlice(bars), symbol: "EURUSD", tf: "M1"}

	// iHighest MODE_HIGH → shift 2 (bar 0 has highest high=30)
	got, _ := builtinIHighest(vm, []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(tsModeHigh), interp.IntVal(3), interp.IntVal(0),
	})
	if got.ToInt() != 2 {
		t.Fatalf("iHighest MODE_HIGH: shift=%d, want 2 (bar 0 has high=30)", got.ToInt())
	}

	// iHighest MODE_CLOSE → shift 1 (bar 1 has highest close=20)
	got, _ = builtinIHighest(vm, []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(tsModeClose), interp.IntVal(3), interp.IntVal(0),
	})
	if got.ToInt() != 1 {
		t.Fatalf("iHighest MODE_CLOSE: shift=%d, want 1 (bar 1 has close=20)", got.ToInt())
	}

	// iLowest MODE_LOW → shift 1 (bar 1 has lowest low=1)
	got, _ = builtinILowest(vm, []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(tsModeLow), interp.IntVal(3), interp.IntVal(0),
	})
	if got.ToInt() != 1 {
		t.Fatalf("iLowest MODE_LOW: shift=%d, want 1 (bar 1 has low=1)", got.ToInt())
	}
}

// TestVM_Audit_IHighest_PartialRange verifies iHighest with start>0 and
// count<Len scans only the specified sub-range. With 5 bars:
//
//	start=2, count=2 → scans MQL shifts 2,3 (chronological 2,1)
//	Highest close in {16,15} = 16 at chronological 2 → MQL shift 2
func TestVM_Audit_IHighest_PartialRange(t *testing.T) {
	vm := tsVM(5)
	args := []interp.Value{
		interp.StringVal(""),
		interp.IntVal(0),
		interp.IntVal(tsModeClose),
		interp.IntVal(2), // count
		interp.IntVal(2), // start
	}
	got, err := builtinIHighest(vm, args)
	if err != nil {
		t.Fatalf("iHighest: %v", err)
	}
	if got.ToInt() != 2 {
		t.Fatalf("iHighest close start=2 count=2: shift=%d, want 2", got.ToInt())
	}
}

// TestVM_Audit_IHighest_EmptySeries verifies iHighest returns -1 on empty series.
func TestVM_Audit_IHighest_EmptySeries(t *testing.T) {
	vm := tsVM(0)
	args := []interp.Value{
		interp.StringVal(""),
		interp.IntVal(0),
		interp.IntVal(tsModeClose),
		interp.IntVal(5),
		interp.IntVal(0),
	}
	got, err := builtinIHighest(vm, args)
	if err != nil {
		t.Fatalf("iHighest empty: %v", err)
	}
	if got.ToInt() != -1 {
		t.Fatalf("iHighest empty series: shift=%d, want -1", got.ToInt())
	}
}

// TestVM_Audit_IHighest_OutOfRangeStart verifies iHighest returns -1 when
// start >= Len() (out of range).
func TestVM_Audit_IHighest_OutOfRangeStart(t *testing.T) {
	vm := tsVM(3)
	args := []interp.Value{
		interp.StringVal(""),
		interp.IntVal(0),
		interp.IntVal(tsModeClose),
		interp.IntVal(5),
		interp.IntVal(10), // start >= Len(3)
	}
	got, err := builtinIHighest(vm, args)
	if err != nil {
		t.Fatalf("iHighest out-of-range: %v", err)
	}
	if got.ToInt() != -1 {
		t.Fatalf("iHighest start=10 Len=3: shift=%d, want -1 (out of range)", got.ToInt())
	}
}

// TestVM_Audit_IHighest_InvalidMode verifies iHighest records a blind spot
// and returns -1 for an invalid series mode (>=6).
func TestVM_Audit_IHighest_InvalidMode(t *testing.T) {
	vm := tsVM(5)
	args := []interp.Value{
		interp.StringVal(""),
		interp.IntVal(0),
		interp.IntVal(99), // invalid mode
		interp.IntVal(5),
		interp.IntVal(0),
	}
	got, err := builtinIHighest(vm, args)
	if err != nil {
		t.Fatalf("iHighest invalid mode: %v", err)
	}
	if got.ToInt() != -1 {
		t.Fatalf("iHighest invalid mode=99: shift=%d, want -1", got.ToInt())
	}
}

// TestVM_Audit_IBarShift_ExactTrue verifies iBarShift with exact=true returns
// the exact shift when the timestamp matches a bar, and -1 when it doesn't.
// Fixed epoch: bar 0 at 2024-01-01 00:00 (shift=4), bar 4 at 00:04 (shift=0).
func TestVM_Audit_IBarShift_ExactTrue(t *testing.T) {
	vm := tsVM(5)
	baseSec := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	// Exact match for bar at 00:02 (chronological 2, MQL shift 2)
	args := []interp.Value{
		interp.StringVal(""),
		interp.IntVal(0),
		interp.IntVal(int32(baseSec + 120)), // 00:02 in seconds
		interp.IntVal(1),                    // exact=true
	}
	got, err := builtinIBarShift(vm, args)
	if err != nil {
		t.Fatalf("iBarShift exact: %v", err)
	}
	if got.ToInt() != 2 {
		t.Fatalf("iBarShift exact=true time=00:02: shift=%d, want 2", got.ToInt())
	}

	// No exact match: 00:02:30 (between bars) → -1
	args[2] = interp.IntVal(int32(baseSec + 150))
	got, err = builtinIBarShift(vm, args)
	if err != nil {
		t.Fatalf("iBarShift exact no-match: %v", err)
	}
	if got.ToInt() != -1 {
		t.Fatalf("iBarShift exact=true time=00:02:30: shift=%d, want -1 (no exact match)", got.ToInt())
	}
}

// TestVM_Audit_IBarShift_ExactFalse verifies iBarShift with exact=false returns
// the nearest bar shift (bar whose time <= specified time).
func TestVM_Audit_IBarShift_ExactFalse(t *testing.T) {
	vm := tsVM(5)
	baseSec := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	// Time 00:02:30 (between bar 00:02 and 00:03) → nearest bar is 00:02 → shift 2
	args := []interp.Value{
		interp.StringVal(""),
		interp.IntVal(0),
		interp.IntVal(int32(baseSec + 150)), // 00:02:30
		interp.IntVal(0),                    // exact=false
	}
	got, err := builtinIBarShift(vm, args)
	if err != nil {
		t.Fatalf("iBarShift non-exact: %v", err)
	}
	if got.ToInt() != 2 {
		t.Fatalf("iBarShift exact=false time=00:02:30: shift=%d, want 2 (nearest bar at 00:02)", got.ToInt())
	}

	// Time before all bars (00:00 minus 60s = 2023-12-31 23:59) → -1
	args[2] = interp.IntVal(int32(baseSec - 60))
	got, err = builtinIBarShift(vm, args)
	if err != nil {
		t.Fatalf("iBarShift before all bars: %v", err)
	}
	if got.ToInt() != -1 {
		t.Fatalf("iBarShift exact=false time before all bars: shift=%d, want -1", got.ToInt())
	}
}

// TestVM_Audit_CopyTime_SecondsConversion verifies CopyTime converts bar
// timestamps from unix_ms to MQL datetime (unix seconds).
func TestVM_Audit_CopyTime_SecondsConversion(t *testing.T) {
	vm := tsVM(3)
	baseSec := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	arr := interp.Value{Kind: interp.ValArray, Array: make([]interp.Value, 0)}
	args := []interp.Value{
		interp.StringVal(""), // symbol
		interp.IntVal(0),     // timeframe
		interp.IntVal(0),     // start_pos
		interp.IntVal(3),     // count (positive = chronological order)
		arr,                  // output array
	}
	got, err := builtinCopyTime(vm, args)
	if err != nil {
		t.Fatalf("CopyTime: %v", err)
	}
	if got.ToInt() != 3 {
		t.Fatalf("CopyTime returned %d, want 3", got.ToInt())
	}
	out := args[4].ArrayData()
	if len(out) != 3 {
		t.Fatalf("CopyTime array length = %d, want 3", len(out))
	}
	// count>0 → chronological order: oldest first → [baseSec, baseSec+60, baseSec+120]
	expected := []int32{int32(baseSec), int32(baseSec + 60), int32(baseSec + 120)}
	for i, want := range expected {
		if out[i].ToInt() != want {
			t.Fatalf("CopyTime[%d] = %d, want %d (unix seconds, not ms)", i, out[i].ToInt(), want)
		}
	}
}

// TestVM_Audit_CopyClose_Direction verifies CopyClose with positive count
// returns bars in chronological order (oldest first), and with negative
// count returns reverse chronological (newest first).
func TestVM_Audit_CopyClose_Direction(t *testing.T) {
	vm := tsVM(3)

	// Positive count → chronological (oldest first): close=[15,16,17]
	arr1 := interp.Value{Kind: interp.ValArray, Array: make([]interp.Value, 0)}
	args1 := []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(0), interp.IntVal(3), arr1,
	}
	got1, _ := builtinCopyClose(vm, args1)
	if got1.ToInt() != 3 {
		t.Fatalf("CopyClose count=3: returned %d, want 3", got1.ToInt())
	}
	out1 := args1[4].ArrayData()
	// chronological: bars 0,1,2 → close 15,16,17
	if out1[0].ToDecimal().IntPart() != 15 || out1[1].ToDecimal().IntPart() != 16 || out1[2].ToDecimal().IntPart() != 17 {
		t.Fatalf("CopyClose count=+3: got [%s,%s,%s], want [15,16,17] (chronological)",
			out1[0].ToDecimal(), out1[1].ToDecimal(), out1[2].ToDecimal())
	}

	// Negative count → reverse chronological (newest first): close=[17,16,15]
	arr2 := interp.Value{Kind: interp.ValArray, Array: make([]interp.Value, 0)}
	args2 := []interp.Value{
		interp.StringVal(""), interp.IntVal(0),
		interp.IntVal(0), interp.IntVal(-3), arr2,
	}
	got2, _ := builtinCopyClose(vm, args2)
	if got2.ToInt() != 3 {
		t.Fatalf("CopyClose count=-3: returned %d, want 3", got2.ToInt())
	}
	out2 := args2[4].ArrayData()
	if out2[0].ToDecimal().IntPart() != 17 || out2[1].ToDecimal().IntPart() != 16 || out2[2].ToDecimal().IntPart() != 15 {
		t.Fatalf("CopyClose count=-3: got [%s,%s,%s], want [17,16,15] (reverse chronological)",
			out2[0].ToDecimal(), out2[1].ToDecimal(), out2[2].ToDecimal())
	}
}
