package mql2go

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// QS-3-BASELINE S1: VM performance baseline benchmarks.
// All benchmarks run against a bare NewVM with the QS-2.3 noopContext —
// they measure VM dispatch/builtin/arith cost, not any sdk.Context
// implementation. Numbers land in docs/audits/vm-perf-baseline-2026-09.md.

// benchVM compiles MQL source and returns a bare VM (noopContext).
func benchVM(b *testing.B, src string) *VM {
	b.Helper()
	r, err := CompileMQL(src)
	if err != nil {
		b.Fatalf("compile: %v", err)
	}
	return NewVM(r.Bytecode())
}

// ── B1: dispatch loop — minimal event, measures runEvent reset + runLoop ──

func BenchmarkDispatchOnTick(b *testing.B) {
	vm := benchVM(b, "int g = 0; void OnTick() { g = 1; }")
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := vm.RunOnTick(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDispatchOnBar(b *testing.B) {
	vm := benchVM(b, "int g = 0; void OnBar() { g = 1; }")
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := vm.RunOnBar(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// ── B2: builtin call cost — dispatch+body per event ──

// Series-access builtin (Close(0) → getSeriesHelper on empty noop series).
func BenchmarkBuiltinSeriesAccess(b *testing.B) {
	vm := benchVM(b, "double g = 0; void OnTick() { g = Close(0); }")
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := vm.RunOnTick(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// Pure-compute builtin (MathAbs — no context access).
func BenchmarkBuiltinPureCompute(b *testing.B) {
	vm := benchVM(b, "double g = 0; void OnTick() { g = MathAbs(-1.5); }")
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := vm.RunOnTick(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// ── B3: executeArith per-op cost — decimal vs int ──

func benchArith(b *testing.B, op Opcode, a, c interp.Value) {
	vm := NewVM(&Bytecode{})
	ins := Instruction{Op: op}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.push(a)
		vm.push(c)
		vm.executeArith(ins)
		vm.pop()
	}
}

func BenchmarkArithDecimalAdd(b *testing.B) {
	benchArith(b, OP_ADD, interp.DecimalVal(decimal.NewFromFloat(1.5)), interp.DecimalVal(decimal.NewFromFloat(2.5)))
}

func BenchmarkArithIntAdd(b *testing.B) {
	benchArith(b, OP_ADD, interp.IntVal(1), interp.IntVal(2))
}

func BenchmarkArithDecimalMul(b *testing.B) {
	benchArith(b, OP_MUL, interp.DecimalVal(decimal.NewFromFloat(1.5)), interp.DecimalVal(decimal.NewFromFloat(2.5)))
}

func BenchmarkArithIntMul(b *testing.B) {
	benchArith(b, OP_MUL, interp.IntVal(3), interp.IntVal(7))
}

func BenchmarkArithDecimalDiv(b *testing.B) {
	benchArith(b, OP_DIV, interp.DecimalVal(decimal.NewFromFloat(7.5)), interp.DecimalVal(decimal.NewFromFloat(2.5)))
}

func BenchmarkArithIntDiv(b *testing.B) {
	benchArith(b, OP_DIV, interp.IntVal(21), interp.IntVal(7))
}

// ── B4: realistic strategy, 1000 ticks ──

// MA-crossover-shaped strategy: per tick it averages 10 + 30 Close() series
// accesses, does decimal division, comparisons and a conditional store.
const benchMACrossSrc = `
int g_sig = 0;
void OnTick() {
	double fast = 0;
	double slow = 0;
	for (int i = 0; i < 10; i++) { fast = fast + Close(i); }
	for (int i = 0; i < 30; i++) { slow = slow + Close(i); }
	fast = fast / 10;
	slow = slow / 30;
	if (fast > slow * 1.001 && Bid > 0) { g_sig = 1; }
}
`

func BenchmarkStrategy1000Ticks(b *testing.B) {
	vm := benchVM(b, benchMACrossSrc)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			if err := vm.RunOnTick(ctx); err != nil {
				b.Fatal(err)
			}
		}
	}
}
