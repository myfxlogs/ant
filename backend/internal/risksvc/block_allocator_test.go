package risksvc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
)

func decF(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

func decClose(a, b decimal.Decimal) bool {
	return a.Sub(b).Abs().LessThan(decF(0.01))
}

func TestProRataAllocator_EqualEquity(t *testing.T) {
	t.Parallel()
	a := &ProRataAllocator{}
	accounts := []AllocAccount{
		{AccountID: "a1", Equity: decF(50_000), FreeMargin: decF(50_000)},
		{AccountID: "a2", Equity: decF(50_000), FreeMargin: decF(50_000)},
	}
	result := a.Allocate(context.Background(), decF(1.0), accounts)
	if len(result) != 2 {
		t.Fatalf("want 2 allocations, got %d", len(result))
	}
	if !decClose(result["a1"], decF(0.5)) {
		t.Fatalf("a1 should get ~0.5, got %s", result["a1"].String())
	}
	if !decClose(result["a2"], decF(0.5)) {
		t.Fatalf("a2 should get ~0.5, got %s", result["a2"].String())
	}
}

func TestProRataAllocator_Proportional(t *testing.T) {
	t.Parallel()
	a := &ProRataAllocator{}
	accounts := []AllocAccount{
		{AccountID: "big", Equity: decF(80_000), FreeMargin: decF(80_000)},
		{AccountID: "small", Equity: decF(20_000), FreeMargin: decF(20_000)},
	}
	result := a.Allocate(context.Background(), decF(1.0), accounts)
	if result["big"].LessThan(decF(0.7)) || result["big"].GreaterThan(decF(0.9)) {
		t.Fatalf("big account should get ~0.8, got %s", result["big"].String())
	}
	if result["small"].LessThan(decF(0.1)) || result["small"].GreaterThan(decF(0.3)) {
		t.Fatalf("small account should get ~0.2, got %s", result["small"].String())
	}
}

func TestProRataAllocator_ZeroVolume(t *testing.T) {
	t.Parallel()
	a := &ProRataAllocator{}
	accounts := []AllocAccount{
		{AccountID: "a1", Equity: decF(50_000)},
	}
	result := a.Allocate(context.Background(), decimal.Zero, accounts)
	if len(result) != 0 {
		t.Fatalf("zero volume should give empty result, got %d", len(result))
	}
}

func TestProRataAllocator_ZeroEquity(t *testing.T) {
	t.Parallel()
	a := &ProRataAllocator{}
	accounts := []AllocAccount{
		{AccountID: "a1", Equity: decimal.Zero},
		{AccountID: "a2", Equity: decimal.Zero},
	}
	result := a.Allocate(context.Background(), decF(1.0), accounts)
	if len(result) != 0 {
		t.Fatalf("all zero equity should give empty result")
	}
}

func TestFIFOAllocator_PriorityOrder(t *testing.T) {
	t.Parallel()
	a := &FIFOAllocator{}
	accounts := []AllocAccount{
		{AccountID: "third", Priority: 3, FreeMargin: decF(1.0)},
		{AccountID: "first", Priority: 1, FreeMargin: decF(1.0)},
		{AccountID: "second", Priority: 2, FreeMargin: decF(1.0)},
	}
	result := a.Allocate(context.Background(), decF(0.6), accounts)
	firstShare := result["first"]
	if firstShare.LessThanOrEqual(decF(0.5)) {
		t.Fatalf("first priority account should get most allocation, got %s", firstShare.String())
	}
	// FIFO fills first account completely; second/third get nothing if first has capacity
	if result["third"].GreaterThan(decimal.Zero) {
		t.Fatalf("third priority should have no allocation when first exhausts volume, got %s", result["third"].String())
	}
}

func TestFIFOAllocator_ExhaustsInOrder(t *testing.T) {
	t.Parallel()
	a := &FIFOAllocator{}
	accounts := []AllocAccount{
		{AccountID: "a1", Priority: 1, FreeMargin: decF(0.3)},
		{AccountID: "a2", Priority: 2, FreeMargin: decF(1.0)},
	}
	result := a.Allocate(context.Background(), decF(1.0), accounts)
	if result["a1"].GreaterThan(decF(0.3 + 1e-9)) {
		t.Fatalf("a1 should be capped at free margin 0.3, got %s", result["a1"].String())
	}
	if result["a2"].LessThan(decF(0.6)) {
		t.Fatalf("a2 should get remainder, got %s", result["a2"].String())
	}
}

func TestVWAPAllocator_CapacityWeighted(t *testing.T) {
	t.Parallel()
	a := &VWAPAllocator{}
	accounts := []AllocAccount{
		{AccountID: "high", FreeMargin: decF(80_000)},
		{AccountID: "low", FreeMargin: decF(20_000)},
	}
	result := a.Allocate(context.Background(), decF(1.0), accounts)
	if result["high"].LessThan(decF(0.7)) || result["high"].GreaterThan(decF(0.9)) {
		t.Fatalf("high capacity should get ~0.8, got %s", result["high"].String())
	}
}

func TestVWAPAllocator_ZeroCapacity(t *testing.T) {
	t.Parallel()
	a := &VWAPAllocator{}
	accounts := []AllocAccount{
		{AccountID: "a1", FreeMargin: decimal.Zero},
	}
	result := a.Allocate(context.Background(), decF(1.0), accounts)
	if len(result) != 0 {
		t.Fatalf("zero capacity should give empty result")
	}
}

func TestAllocator_Names(t *testing.T) {
	t.Parallel()
	if n := (&ProRataAllocator{}).Name(); n != "pro_rata" {
		t.Fatalf("want pro_rata, got %s", n)
	}
	if n := (&FIFOAllocator{}).Name(); n != "fifo" {
		t.Fatalf("want fifo, got %s", n)
	}
	if n := (&VWAPAllocator{}).Name(); n != "vwap" {
		t.Fatalf("want vwap, got %s", n)
	}
}

func TestAllocator_SumsToVolume(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	accounts := []AllocAccount{
		{AccountID: "a1", Equity: decF(100_000), FreeMargin: decF(50_000), Priority: 1},
		{AccountID: "a2", Equity: decF(50_000), FreeMargin: decF(30_000), Priority: 2},
		{AccountID: "a3", Equity: decF(25_000), FreeMargin: decF(40_000), Priority: 3},
	}
	for _, alloc := range []BlockAllocator{&ProRataAllocator{}, &FIFOAllocator{}, &VWAPAllocator{}} {
		result := alloc.Allocate(ctx, decF(0.5), accounts)
		sum := decimal.Zero
		for _, v := range result {
			sum = sum.Add(v)
		}
		if sum.LessThanOrEqual(decimal.Zero) || sum.GreaterThan(decF(0.5+1e-9)) {
			t.Fatalf("%s: allocation sum %s should be ≤ 0.5", alloc.Name(), sum.String())
		}
	}
}
