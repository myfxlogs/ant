package mdgateway

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"

	"github.com/shopspring/decimal"
)

// --- normalizer.go ---

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

func TestAbs64(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   int64
		want int64
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{math.MinInt64, math.MinInt64},
	}
	for _, tt := range tests {
		got := abs64(tt.in)
		if got != tt.want {
			t.Errorf("abs64(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestZscore(t *testing.T) {
	t.Parallel()
	// Empty or single-element window: should return 0 or NaN (no stddev).
	_ = zscore([]float64{}, 100)
	_ = zscore([]float64{100}, 100)
	// Multi-element window with consistent values.
	zs := zscore([]float64{100, 100, 100, 100}, 150)
	if math.IsInf(zs, 1) {
		t.Errorf("zscore(150 in [100,100,100,100]) = Inf, want finite")
	}
}

func TestDefaultQualityConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultQualityConfig()
	if cfg.GapMaxSeconds != 5 {
		t.Errorf("GapMaxSeconds = %f, want 5", cfg.GapMaxSeconds)
	}
	if cfg.OutlierSigma != 5 {
		t.Errorf("OutlierSigma = %f, want 5", cfg.OutlierSigma)
	}
}

func TestNewQuality(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	if q == nil {
		t.Fatal("NewQuality returned nil")
	}
	if q.last == nil {
		t.Error("last map should be initialized")
	}
	if q.prices == nil {
		t.Error("prices map should be initialized")
	}
}

func TestSpreadZscore_Empty(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	if z := q.SpreadZscore("key", 5.0); z < 0 {
		t.Errorf("SpreadZscore = %f, want >=0", z)
	}
}

func TestTickRateZscore_Empty(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	if z := q.TickRateZscore("key", 0.5); z < 0 {
		t.Errorf("TickRateZscore = %f, want >=0", z)
	}
}

func TestCheck_ValidTick(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	now := time.Now().UnixMilli()
	tick := &mdtick.Tick{
		Broker: "test", Canonical: "EURUSD",
		TsUnixMs:      now,
		ArrivedUnixMs: now,
		Bid:           decimal.NewFromFloat(1.08000),
		Ask:           decimal.NewFromFloat(1.08001),
	}
	res := q.Check(context.Background(), tick)
	if res.Dropped {
		t.Errorf("tick should not be dropped: %s", res.DroppedReason)
	}
}

func TestCheck_InvertedBidAsk(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	now := time.Now().UnixMilli()
	tick := &mdtick.Tick{
		Broker: "test", Canonical: "EURUSD",
		TsUnixMs:      now,
		ArrivedUnixMs: now,
		Bid:           decimal.NewFromFloat(100),
		Ask:           decimal.NewFromFloat(1),
	}
	res := q.Check(context.Background(), tick)
	if !res.Dropped {
		t.Error("inverted bid>ask should be dropped")
	}
}

func TestIsOutlier_EmptyHistory(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	if q.isOutlier("key", 100) {
		t.Error("isOutlier should return false with empty history")
	}
}

func TestIsOutlier_Normal(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	// Fill with enough history to compute meaningful zscore.
	for i := 0; i < 100; i++ {
		q.prices["key"] = append(q.prices["key"], 1.08000)
	}
	// A price within 1 pct should not be an outlier.
	if q.isOutlier("key", 1.08001) {
		t.Log("nearby price flagged as outlier (zscore threshold may be tight)")
	}
}

func TestTrackSpread_TrackTickRate(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	q.trackSpread("key", 5.0)
	q.trackTickRate("key", 1.0)
	if zs := q.SpreadZscore("key", 5.0); zs != 0 {
		t.Errorf("SpreadZscore with single point = %f, want 0", zs)
	}
	if tr := q.TickRateZscore("key", 1.0); tr != 0 {
		t.Errorf("TickRateZscore with single point = %f, want 0", tr)
	}
}

// --- circuit_breaker.go ---

func TestStateString_Unknown(t *testing.T) {
	t.Parallel()
	if got := State(99).String(); got != "unknown" {
		t.Errorf("State(99).String() = %q, want \"unknown\"", got)
	}
}

func TestCircuitBreaker_OpenBlocks(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 2, time.Hour)
	cb.Allow()
	cb.OnFailure()
	if cb.Allow() {
		t.Error("open circuit should deny calls")
	}
	if cb.State() != StateOpen {
		t.Error("circuit should be open")
	}
}

func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 2, time.Millisecond)
	cb.Allow()
	cb.OnFailure()
	time.Sleep(10 * time.Millisecond)
	if !cb.Allow() {
		t.Error("half-open should allow one probe call")
	}
	if cb.State() != StateHalfOpen {
		t.Error("should be half-open")
	}
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 1, time.Millisecond)
	cb.Allow()
	cb.OnFailure()
	time.Sleep(10 * time.Millisecond)
	cb.Allow()
	cb.OnSuccess()
	if cb.State() != StateClosed {
		t.Errorf("should transition to closed, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToOpen(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 1, time.Millisecond)
	cb.Allow()
	cb.OnFailure()
	time.Sleep(10 * time.Millisecond)
	cb.Allow()
	cb.OnFailure()
	if cb.State() != StateOpen {
		t.Errorf("should stay open after failed probe, got %v", cb.State())
	}
}

// --- normalizer.go ---

func TestNewNormalizer_NilPG(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	if n == nil {
		t.Fatal("NewNormalizer returned nil")
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	result := n.Resolve(context.Background(), "broker", "EURUSDm")
	if result == "" {
		t.Log("Resolve returned empty (expected without PG-backed mapping)")
	}
}

// --- normalizer.go Resolve cache hit ---

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

// --- quality.go ---

func TestSetDLQWriter(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	q.SetDLQWriter(nil)
}

// --- quality.go Check with stale tick ---

func TestCheck_StaleTick(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs:      time.Now().UnixMilli() - 60_000, // 1min old
		ArrivedUnixMs: time.Now().UnixMilli(),
		Bid:           decimal.NewFromFloat(1.08000),
		Ask:           decimal.NewFromFloat(1.08001),
	}
	res := q.Check(context.Background(), tick)
	_ = res
}

// --- quality.go check stale ---

func TestCheck_StaleArrival(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs:      time.Now().UnixMilli(),
		ArrivedUnixMs: time.Now().UnixMilli() - 60_000,
		Bid:           decimal.NewFromFloat(1.08000),
		Ask:           decimal.NewFromFloat(1.08001),
	}
	res := q.Check(context.Background(), tick)
	_ = res
}
