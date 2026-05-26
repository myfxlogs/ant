package risksvc

import (
	"context"
	"math"
	"testing"
)

func TestProRataAllocator_EqualEquity(t *testing.T) {
	a := &ProRataAllocator{}
	accounts := []AllocAccount{
		{AccountID: "a1", Equity: 50_000, FreeMargin: 50_000},
		{AccountID: "a2", Equity: 50_000, FreeMargin: 50_000},
	}
	result := a.Allocate(context.Background(), 1.0, accounts)
	if len(result) != 2 {
		t.Fatalf("want 2 allocations, got %d", len(result))
	}
	if math.Abs(result["a1"]-0.5) > 0.01 {
		t.Fatalf("a1 should get ~0.5, got %.4f", result["a1"])
	}
	if math.Abs(result["a2"]-0.5) > 0.01 {
		t.Fatalf("a2 should get ~0.5, got %.4f", result["a2"])
	}
}

func TestProRataAllocator_Proportional(t *testing.T) {
	a := &ProRataAllocator{}
	accounts := []AllocAccount{
		{AccountID: "big", Equity: 80_000, FreeMargin: 80_000},
		{AccountID: "small", Equity: 20_000, FreeMargin: 20_000},
	}
	result := a.Allocate(context.Background(), 1.0, accounts)
	if result["big"] < 0.7 || result["big"] > 0.9 {
		t.Fatalf("big account should get ~0.8, got %.4f", result["big"])
	}
	if result["small"] < 0.1 || result["small"] > 0.3 {
		t.Fatalf("small account should get ~0.2, got %.4f", result["small"])
	}
}

func TestProRataAllocator_ZeroVolume(t *testing.T) {
	a := &ProRataAllocator{}
	accounts := []AllocAccount{
		{AccountID: "a1", Equity: 50_000},
	}
	result := a.Allocate(context.Background(), 0, accounts)
	if len(result) != 0 {
		t.Fatalf("zero volume should give empty result, got %d", len(result))
	}
}

func TestProRataAllocator_ZeroEquity(t *testing.T) {
	a := &ProRataAllocator{}
	accounts := []AllocAccount{
		{AccountID: "a1", Equity: 0},
		{AccountID: "a2", Equity: 0},
	}
	result := a.Allocate(context.Background(), 1.0, accounts)
	if len(result) != 0 {
		t.Fatalf("all zero equity should give empty result")
	}
}

func TestFIFOAllocator_PriorityOrder(t *testing.T) {
	a := &FIFOAllocator{}
	accounts := []AllocAccount{
		{AccountID: "third", Priority: 3, FreeMargin: 1.0},
		{AccountID: "first", Priority: 1, FreeMargin: 1.0},
		{AccountID: "second", Priority: 2, FreeMargin: 1.0},
	}
	result := a.Allocate(context.Background(), 0.6, accounts)
	firstShare := result["first"]
	if firstShare <= 0.5 {
		t.Fatalf("first priority account should get most allocation, got %.4f", firstShare)
	}
	// FIFO fills first account completely; second/third get nothing if first has capacity
	if result["third"] > 0 {
		t.Fatalf("third priority should have no allocation when first exhausts volume, got %.4f", result["third"])
	}
}

func TestFIFOAllocator_ExhaustsInOrder(t *testing.T) {
	a := &FIFOAllocator{}
	accounts := []AllocAccount{
		{AccountID: "a1", Priority: 1, FreeMargin: 0.3},
		{AccountID: "a2", Priority: 2, FreeMargin: 1.0},
	}
	result := a.Allocate(context.Background(), 1.0, accounts)
	if result["a1"] > 0.3+1e-9 {
		t.Fatalf("a1 should be capped at free margin 0.3, got %.4f", result["a1"])
	}
	if result["a2"] < 0.6 {
		t.Fatalf("a2 should get remainder, got %.4f", result["a2"])
	}
}

func TestVWAPAllocator_CapacityWeighted(t *testing.T) {
	a := &VWAPAllocator{}
	accounts := []AllocAccount{
		{AccountID: "high", FreeMargin: 80_000},
		{AccountID: "low", FreeMargin: 20_000},
	}
	result := a.Allocate(context.Background(), 1.0, accounts)
	if result["high"] < 0.7 || result["high"] > 0.9 {
		t.Fatalf("high capacity should get ~0.8, got %.4f", result["high"])
	}
}

func TestVWAPAllocator_ZeroCapacity(t *testing.T) {
	a := &VWAPAllocator{}
	accounts := []AllocAccount{
		{AccountID: "a1", FreeMargin: 0},
	}
	result := a.Allocate(context.Background(), 1.0, accounts)
	if len(result) != 0 {
		t.Fatalf("zero capacity should give empty result")
	}
}

func TestAllocator_Names(t *testing.T) {
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
	ctx := context.Background()
	accounts := []AllocAccount{
		{AccountID: "a1", Equity: 100_000, FreeMargin: 50_000, Priority: 1},
		{AccountID: "a2", Equity: 50_000, FreeMargin: 30_000, Priority: 2},
		{AccountID: "a3", Equity: 25_000, FreeMargin: 40_000, Priority: 3},
	}
	for _, alloc := range []BlockAllocator{&ProRataAllocator{}, &FIFOAllocator{}, &VWAPAllocator{}} {
		result := alloc.Allocate(ctx, 0.5, accounts)
		sum := 0.0
		for _, v := range result {
			sum += v
		}
		if sum <= 0 || sum > 0.5+1e-9 {
			t.Fatalf("%s: allocation sum %.4f should be ≤ 0.5", alloc.Name(), sum)
		}
	}
}
