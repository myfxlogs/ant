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

func TestVM_Audit_CopyTimeUsesSeconds(t *testing.T) {
	stamp := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := &auditContext{
		bars:   sdk.BarsToSlice([]sdk.Bar{{Timestamp: stamp.UnixMilli()}}),
		symbol: "EURUSD",
		tf:     "M1",
	}
	vm := NewVM(&Bytecode{})
	vm.ctx = ctx
	args := []interp.Value{
		interp.StringVal(""),
		interp.IntVal(0),
		interp.IntVal(0),
		interp.IntVal(1),
		{Kind: interp.ValArray, Array: make([]interp.Value, 0)},
	}
	got, err := builtinCopyTime(vm, args)
	if err != nil {
		t.Fatalf("builtinCopyTime: %v", err)
	}
	if got.ToInt() != 1 || len(args[4].Array) != 1 {
		t.Fatalf("CopyTime result = %v, array length = %d, want 1", got, len(args[4].Array))
	}
	if gotSeconds := args[4].Array[0].ToInt(); gotSeconds != int32(stamp.Unix()) {
		t.Fatalf("CopyTime value = %d, want %d seconds", gotSeconds, stamp.Unix())
	}
}

func TestVM_Audit_ExtremeAndBarShiftSemantics(t *testing.T) {
	ctx := &auditContext{
		bars: sdk.BarsToSlice([]sdk.Bar{
			{Open: decimal.NewFromInt(1), High: decimal.NewFromInt(2), Low: decimal.NewFromInt(0), Close: decimal.NewFromInt(1), Timestamp: 1000},
			{Open: decimal.NewFromInt(3), High: decimal.NewFromInt(4), Low: decimal.NewFromInt(2), Close: decimal.NewFromInt(3), Timestamp: 2000},
			{Open: decimal.NewFromInt(2), High: decimal.NewFromInt(100), Low: decimal.NewFromInt(1), Close: decimal.NewFromInt(2), Timestamp: 3000},
		}),
		symbol: "EURUSD",
		tf:     "M1",
	}
	vm := NewVM(&Bytecode{})
	vm.ctx = ctx
	args := []interp.Value{interp.StringVal(""), interp.IntVal(0), interp.IntVal(0), interp.IntVal(3), interp.IntVal(0)}
	if got, _ := builtinIHighest(vm, args); got.ToInt() != 1 {
		t.Fatalf("iHighest MODE_OPEN = %d, want shift 1", got.ToInt())
	}
	args[2] = interp.IntVal(3)
	if got, _ := builtinILowest(vm, args); got.ToInt() != 2 {
		t.Fatalf("iLowest MODE_CLOSE = %d, want shift 2", got.ToInt())
	}
	args[4] = interp.IntVal(3)
	if got, _ := builtinIHighest(vm, args); got.ToInt() != -1 {
		t.Fatalf("iHighest out-of-range start = %d, want -1", got.ToInt())
	}
	shiftArgs := []interp.Value{interp.StringVal(""), interp.IntVal(0), interp.IntVal(2), interp.IntVal(1)}
	if got, _ := builtinIBarShift(vm, shiftArgs); got.ToInt() != 1 {
		t.Fatalf("iBarShift non-exact = %d, want 1", got.ToInt())
	}
	shiftArgs[2] = interp.IntVal(2500)
	shiftArgs[3] = interp.IntVal(1)
	if got, _ := builtinIBarShift(vm, shiftArgs); got.ToInt() != -1 {
		t.Fatalf("iBarShift exact missing = %d, want -1", got.ToInt())
	}
}

func TestVM_Audit_PositionTimeUsesSeconds(t *testing.T) {
	vm := NewVM(&Bytecode{Code: []Instruction{{Op: OP_HALT}}})
	opened := time.Unix(1704164645, 0).UTC()
	vm.currentPos = &sdk.Position{OpenTime: opened}
	got, err := builtinPositionGetInteger(vm, []interp.Value{interp.IntVal(3)})
	if err != nil {
		t.Fatalf("builtinPositionGetInteger: %v", err)
	}
	if got.ToInt() != int32(opened.Unix()) {
		t.Fatalf("POSITION_TIME = %d, want %d seconds", got.ToInt(), opened.Unix())
	}
}

func TestVM_Audit_TimeFormatFlags(t *testing.T) {
	ts := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC).Unix()
	cases := []struct {
		mode int32
		want string
	}{
		{1, "2024.01.02"},
		{2, "03:04"},
		{4, "03:04:05"},
		{3, "2024.01.02 03:04"},
		{5, "2024.01.02 03:04:05"},
	}
	for _, tc := range cases {
		got, _ := builtinTimeToString(nil, []interp.Value{interp.IntVal(int32(ts)), interp.IntVal(tc.mode)})
		if got.ToString() != tc.want {
			t.Errorf("mode %d: got %q, want %q", tc.mode, got.ToString(), tc.want)
		}
	}
}

func TestVM_Audit_UnimplementedMQL5APIRejected(t *testing.T) {
	const source = `
int OnInit() { return 0; }
void OnTick() {
    CopyBuffer(1, 0, 0, 2, 0);
}
`
	if _, err := CompileMQL(source); err == nil || !strings.Contains(err.Error(), "unsupported function CopyBuffer") {
		t.Fatalf("unimplemented MQL5 API should be rejected, got %v", err)
	}
}

func TestVM_Audit_RandomIsSeededPerVM(t *testing.T) {
	first := NewVM(&Bytecode{})
	second := NewVM(&Bytecode{})
	seed := []interp.Value{interp.IntVal(123)}
	_, _ = builtinMathSrand(first, seed)
	_, _ = builtinMathSrand(second, seed)
	for i := 0; i < 3; i++ {
		left, _ := builtinMathRand(first, nil)
		right, _ := builtinMathRand(second, nil)
		if left.ToInt() != right.ToInt() {
			t.Fatalf("seeded random values differ at %d: %d != %d", i, left.ToInt(), right.ToInt())
		}
	}
}

func TestVM_Audit_RuntimeBlindSpotsSorted(t *testing.T) {
	vm := NewVM(&Bytecode{})
	vm.runtimeBlindSpots = map[string]int{
		"TimeUnknown":  2,
		"iMissing":     1,
		"OrderMissing": 3,
	}
	got := vm.GetRuntimeBlindSpots()
	if len(got) != 3 {
		t.Fatalf("blind spot count = %d, want 3", len(got))
	}
	if got[0].Builtin != "OrderMissing" || got[1].Builtin != "iMissing" || got[2].Builtin != "TimeUnknown" {
		t.Fatalf("blind spots are not deterministically severity/count sorted: %+v", got)
	}
}

func TestVM_Audit_ArrayResizePersists(t *testing.T) {
	const source = `
 double values[1];
 int g_size = 0;
 int OnInit() { return 0; }
 void OnTick() {
     ArrayResize(values, 3);
     g_size = ArraySize(values);
 }
 `
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	if err := runner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick: %v", err)
	}
	value, ok := runner.GetGlobal("g_size")
	if !ok || value.ToInt() != 3 {
		t.Fatalf("ArrayResize size = %v, want 3", value)
	}
}
