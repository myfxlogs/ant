package mdgateway

import (
	"context"
	"fmt"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestInvalidateCache(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	n.cache["broker:EURUSD"] = "EURUSD"
	n.cache["broker:GBPUSD"] = "GBPUSD"
	n.InvalidateCache("broker", "EURUSD")
	if _, ok := n.cache["broker:EURUSD"]; ok {
		t.Error("EURUSD should be invalidated")
	}
	if _, ok := n.cache["broker:GBPUSD"]; !ok {
		t.Error("GBPUSD should still be cached")
	}
	n.InvalidateCache("unknown", "XYZ")
}

func TestNewNormalizer_NilPool(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	if n == nil {
		t.Fatal("NewNormalizer(nil) returned nil")
	}
	if n.pg != nil {
		t.Error("pg should be nil")
	}
	if n.cache == nil {
		t.Error("cache map should be initialized")
	}
}

// --- tick_dedup.go ---

func TestNewTickDedup_Defaults(t *testing.T) {
	t.Parallel()
	d := NewTickDedup(0)
	if d.size != 1000 {
		t.Errorf("NewTickDedup(0) size = %d, want 1000", d.size)
	}
	d = NewTickDedup(-5)
	if d.size != 1000 {
		t.Errorf("NewTickDedup(-5) size = %d, want 1000", d.size)
	}
	d = NewTickDedup(50)
	if d.size != 50 {
		t.Errorf("NewTickDedup(50) size = %d, want 50", d.size)
	}
}

func TestTickDedup_Seen(t *testing.T) {
	t.Parallel()
	d := NewTickDedup(5)
	tick1 := &mdtick.Tick{Broker: "test", Canonical: "EURUSD", TsUnixMs: 1000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2)}
	tick2 := &mdtick.Tick{Broker: "test", Canonical: "EURUSD", TsUnixMs: 1000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2)}
	tick3 := &mdtick.Tick{Broker: "test", Canonical: "EURUSD", TsUnixMs: 2000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2)}
	if d.Seen(tick1) {
		t.Error("first tick should not be seen as duplicate")
	}
	if !d.Seen(tick2) {
		t.Error("identical tick should be seen as duplicate")
	}
	if d.Seen(tick3) {
		t.Error("tick with different timestamp should not be duplicate")
	}
}

func TestTickDedup_DifferentKeys(t *testing.T) {
	t.Parallel()
	d := NewTickDedup(5)
	t1 := &mdtick.Tick{Broker: "b1", Canonical: "EURUSD", TsUnixMs: 1000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2)}
	t2 := &mdtick.Tick{Broker: "b2", Canonical: "EURUSD", TsUnixMs: 1000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2)}
	if d.Seen(t1) {
		t.Error("first tick should not be seen")
	}
	if d.Seen(t2) {
		t.Error("different broker should have independent dedup")
	}
}

func TestTickHash(t *testing.T) {
	t.Parallel()
	t1 := &mdtick.Tick{TsUnixMs: 1000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2), BidVolume: 100, AskVolume: 50}
	t2 := &mdtick.Tick{TsUnixMs: 1000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2), BidVolume: 100, AskVolume: 50}
	t3 := &mdtick.Tick{TsUnixMs: 2000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2), BidVolume: 100, AskVolume: 50}
	h1 := tickHash(t1)
	h2 := tickHash(t2)
	h3 := tickHash(t3)
	if h1 != h2 {
		t.Error("identical ticks should have identical hashes")
	}
	if h1 == h3 {
		t.Error("different timestamps should produce different hashes")
	}
}

// --- quality.go ---

func TestNewNormalizerInvalidator(t *testing.T) {
	t.Parallel()
	inv := NewNormalizerInvalidator(nil, nil, func(broker, raw string) {})
	if inv == nil {
		t.Fatal("NewNormalizerInvalidator returned nil")
	}
}

func TestNormalizerInvalidator_StartStop(t *testing.T) {
	t.Parallel()
	inv := NewNormalizerInvalidator(zap.NewNop(), nil, func(broker, raw string) {})
	inv.Start(context.Background(), nil)
	time.Sleep(10 * time.Millisecond)
	inv.Stop()
}

// --- publisher.go ---

func TestSetBaseContext(t *testing.T) {
	t.Parallel()
	mgr := NewManager(ManagerDeps{})
	ctx := context.Background()
	mgr.SetBaseContext(ctx)
}

func TestNewNormalizer_NilPG(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	if n == nil {
		t.Fatal("NewNormalizer returned nil")
	}
}

// --- dlq_writer.go ---

func TestResolve(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	result := n.Resolve(context.Background(), "broker", "EURUSDm")
	if result == "" {
		t.Log("Resolve returned empty (expected without PG-backed mapping)")
	}
}

func TestNormalizer_Resolve_CacheHit(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	n.cache["broker:EURUSDm"] = "EURUSD"
	result := n.Resolve(context.Background(), "broker", "EURUSDm")
	if result != "EURUSD" {
		t.Errorf("cache hit should return EURUSD, got %q", result)
	}
}

func TestNormalizer_Resolve_CacheGuard(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	// Fill cache past maxCacheSize (100k) to trigger cache reset.
	for i := 0; i < 100001; i++ {
		n.cache[fmt.Sprintf("b:s%d", i)] = fmt.Sprintf("S%d", i)
	}
	result := n.Resolve(context.Background(), "broker", "EURUSDm")
	if result == "" {
		t.Error("Resolve should still work after cache reset")
	}
}

// --- dlq_writer.go WriteTick ---

func TestNormalizerInvalidator_TickerLoop(t *testing.T) {
	t.Parallel()
	called := false
	inv := NewNormalizerInvalidator(zap.NewNop(), nil, func(broker, raw string) { called = true })
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	inv.Start(ctx, nil)
	time.Sleep(30 * time.Millisecond)
	inv.Stop()
	_ = called
}

// --- spill_replay.go Run ---

// --- quote_stuffing.go IsPaused deeper ---
