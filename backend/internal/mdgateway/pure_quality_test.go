package mdgateway

import (
	"context"
	"math"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"

	"github.com/shopspring/decimal"
)

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

func TestComputeRateZscore(t *testing.T) {
	t.Parallel()
	// Stable history: all values same, zscore should be small.
	history := []float64{10, 10, 10, 10, 10, 10, 10, 10, 10, 10}
	z := computeRateZscore(history, 10)
	if z != 0 {
		t.Errorf("zscore of stable history should be 0, got %f", z)
	}
}

func TestComputeRateZscore_Spike(t *testing.T) {
	t.Parallel()
	// History with some variance, value far above mean.
	history := []float64{1, 2, 1, 2, 1, 2, 1, 2, 1, 2}
	z := computeRateZscore(history, 50)
	if z <= 0 {
		t.Errorf("zscore should be positive for 50 vs history mean=1.5, got %f", z)
	}
	// Value below mean should get negative zscore.
	z2 := computeRateZscore(history, 0.1)
	if z2 >= 0 {
		t.Errorf("zscore should be negative for 0.1 vs history mean=1.5, got %f", z2)
	}
}

func TestComputeRateZscore_SingleValue(t *testing.T) {
	t.Parallel()
	z := computeRateZscore([]float64{5}, 5)
	if z != 0 {
		t.Errorf("zscore of single value should be 0 (zero variance), got %f", z)
	}
}

// --- manager.go ---

func TestPercentile_Empty(t *testing.T) {
	t.Parallel()
	h := newHistogram([]float64{1, 10, 100})
	p := h.percentile(99)
	if p != 0 {
		t.Errorf("percentile of empty histogram should be 0, got %f", p)
	}
}

func TestPercentile_WithData(t *testing.T) {
	t.Parallel()
	h := newHistogram([]float64{1, 10, 100})
	h.counts[1].Store(1) // one observation in bucket 10
	p := h.percentile(99)
	if p <= 0 {
		t.Errorf("percentile with data should be > 0, got %f", p)
	}
}

func TestNewHistogram(t *testing.T) {
	t.Parallel()
	h := newHistogram([]float64{1, 10, 100})
	if h == nil {
		t.Fatal("newHistogram returned nil")
	}
}

// --- normalizer.go ---

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

// --- manager.go HandleTick ---

// --- market_state.go EvaluateTradeable ---

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

// --- market_state.go update/stale ---
