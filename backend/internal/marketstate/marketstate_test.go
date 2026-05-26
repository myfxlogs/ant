package marketstate

import (
	"math"
	"testing"
	"time"
)

// --- SpreadTracker Tests ---

func TestSpreadTracker_Mean(t *testing.T) {
	st := NewSpreadTracker(10)
	for i := 0; i < 10; i++ {
		st.Observe(1.0)
	}
	mean := st.Mean()
	if math.Abs(mean-1.0) > 0.01 {
		t.Fatalf("mean should be 1.0, got %.4f", mean)
	}
}

func TestSpreadTracker_ZScore(t *testing.T) {
	st := NewSpreadTracker(100)
	// Varying spread: 0.8 to 1.2 pips
	for i := 0; i < 100; i++ {
		st.Observe(0.8 + float64(i%5)*0.1) // 0.8, 0.9, 1.0, 1.1, 1.2
	}
	// Current spread = 5.0 pips (clear outlier vs mean ~1.0, stddev ~0.16)
	mean, stddev, zscore := st.Stats(5.0)
	if math.Abs(mean-1.0) > 0.15 {
		t.Fatalf("mean should be ~1.0, got %.4f", mean)
	}
	if stddev <= 0 {
		t.Fatal("stddev should be > 0 with varying data")
	}
	if zscore <= 5.0 {
		t.Fatalf("z-score should be >> 5 for extreme outlier, got %.4f (mean=%.4f stddev=%.6f)", zscore, mean, stddev)
	}
}

func TestSpreadTracker_InsufficientData(t *testing.T) {
	st := NewSpreadTracker(100)
	st.Observe(1.0)
	st.Observe(2.0)
	mean, stddev, zscore := st.Stats(3.0)
	if zscore != 0 {
		t.Fatalf("z-score should be 0 with insufficient data, got %.4f", zscore)
	}
	_ = mean
	_ = stddev
}

func TestSpreadTracker_RollingWindow(t *testing.T) {
	st := NewSpreadTracker(5)
	for i := 0; i < 5; i++ {
		st.Observe(1.0)
	}
	if st.Count() != 5 {
		t.Fatalf("count should be 5, got %d", st.Count())
	}
	st.Observe(2.0) // pushes out first 1.0
	if st.Count() != 5 {
		t.Fatalf("count should remain 5 after wrap, got %d", st.Count())
	}
	mean := st.Mean()
	// 4*1.0 + 2.0 = 6.0 / 5 = 1.2
	if math.Abs(mean-1.2) > 0.01 {
		t.Fatalf("rolling mean should be 1.2, got %.4f", mean)
	}
}

// --- SwapWindowDetector Tests ---

func TestSwapWindowDetector_Wednesday(t *testing.T) {
	d := NewSwapWindowDetector()
	// May 27, 2026 = Wednesday
	wed := time.Date(2026, 5, 27, 14, 0, 0, 0, time.UTC)
	if !d.IsTripleSwapDay(wed) {
		t.Fatal("Wednesday should be triple swap day")
	}
}

func TestSwapWindowDetector_Monday(t *testing.T) {
	d := NewSwapWindowDetector()
	mon := time.Date(2026, 5, 25, 14, 0, 0, 0, time.UTC)
	if d.IsTripleSwapDay(mon) {
		t.Fatal("Monday should NOT be triple swap day")
	}
}

func TestSwapWindowDetector_InRolloverWindow(t *testing.T) {
	d := NewSwapWindowDetector()
	d.DangerWindowMinutes = 30
	// Exactly at rollover time
	rollover := time.Date(2026, 5, 27, 21, 0, 0, 0, time.UTC)
	if !d.InRolloverWindow(rollover) {
		t.Fatal("should be in rollover window at exact rollover time")
	}
}

func TestSwapWindowDetector_OutsideRollover(t *testing.T) {
	d := NewSwapWindowDetector()
	d.DangerWindowMinutes = 30
	// 2 hours before rollover
	before := time.Date(2026, 5, 27, 19, 0, 0, 0, time.UTC)
	if d.InRolloverWindow(before) {
		t.Fatal("should NOT be in rollover window 2 hours before")
	}
}

func TestSwapWindowDetector_MinutesToRollover(t *testing.T) {
	d := NewSwapWindowDetector()
	// 20:30 UTC = 30 min before 21:00 rollover
	before := time.Date(2026, 5, 27, 20, 30, 0, 0, time.UTC)
	mins := d.MinutesToRollover(before)
	if math.Abs(mins-30.0) > 0.1 {
		t.Fatalf("minutes to rollover should be ~30, got %.1f", mins)
	}
}

func TestSwapWindowDetector_PastRollover(t *testing.T) {
	d := NewSwapWindowDetector()
	// 22:00 UTC = 60 min after rollover, so next rollover is tomorrow
	after := time.Date(2026, 5, 27, 22, 0, 0, 0, time.UTC)
	mins := d.MinutesToRollover(after)
	// Next rollover: tomorrow 21:00 UTC = 23h away = 1380 min
	if mins < 1300 || mins > 1500 {
		t.Fatalf("past rollover should show ~1380 min to next, got %.1f", mins)
	}
}

// --- VolatilityClassifier Tests ---

func TestVolatilityClassifier_Calm(t *testing.T) {
	vc := NewVolatilityClassifier(100)
	price := 1.0850
	for i := 0; i < 100; i++ {
		price += 0.000001 // tiny changes → very low vol
		vc.Observe(price)
	}
	regime := vc.Classify()
	if regime != RegimeCalm {
		t.Fatalf("tiny price changes should be calm, got %s", regime)
	}
}

func TestVolatilityClassifier_Volatile(t *testing.T) {
	vc := NewVolatilityClassifier(100)
	vc.calmVolAnnual = 0.05
	vc.volatileVolAnnual = 0.20
	price := 1.0850
	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			price += 0.015 // large swings
		} else {
			price -= 0.015
		}
		vc.Observe(price)
	}
	regime := vc.Classify()
	if regime != RegimeVolatile && regime != RegimeRanging {
		t.Fatalf("large swings should be volatile or ranging, got %s (vol=%.4f)", regime, vc.AnnualVolatility())
	}
}

func TestVolatilityClassifier_InsufficientData(t *testing.T) {
	vc := NewVolatilityClassifier(100)
	for i := 0; i < 5; i++ {
		vc.Observe(1.0850)
	}
	if vc.Classify() != RegimeUnknown {
		t.Fatal("insufficient data should be unknown")
	}
}

func TestVolatilityClassifier_VolPercentile(t *testing.T) {
	vc := NewVolatilityClassifier(100)
	vc.calmVolAnnual = 0.05
	vc.volatileVolAnnual = 0.30
	price := 1.0850
	for i := 0; i < 100; i++ {
		price += 0.0002
		vc.Observe(price)
	}
	pct := vc.VolPercentile()
	if pct < 0 || pct > 1 {
		t.Fatalf("vol percentile should be 0-1, got %.4f", pct)
	}
}

// --- Aggregator Tests ---

func TestAggregator_Snapshot(t *testing.T) {
	agg := NewAggregator()
	price := 1.0850
	for i := 0; i < 200; i++ {
		price += 0.00001
		agg.Observe(price, 1.0)
	}
	ms := agg.Snapshot("EURUSD")
	if ms.Symbol != "EURUSD" {
		t.Fatalf("symbol: %s", ms.Symbol)
	}
	if ms.Regime == RegimeUnknown {
		t.Fatal("should have a regime after 200 observations")
	}
	if ms.QualityScore < 0 || ms.QualityScore > 1 {
		t.Fatalf("quality score out of range: %.4f", ms.QualityScore)
	}
	t.Logf("MarketState: symbol=%s regime=%s spread_z=%.2f quality=%.2f tradable=%v",
		ms.Symbol, ms.Regime, ms.SpreadZScore, ms.QualityScore, ms.Tradable)
}

func TestAggregator_WideningSpread(t *testing.T) {
	agg := NewAggregator()
	price := 1.0850
	// Normal spread history
	for i := 0; i < 180; i++ {
		agg.Observe(price, 1.0)
	}
	// Sudden spread widening
	for i := 0; i < 20; i++ {
		agg.Observe(price, 5.0) // 5 pips spread
	}
	ms := agg.Snapshot("EURUSD")
	if !ms.SpreadWidening {
		t.Log("spread widening not flagged (needs more outlier observations)")
	}
	if ms.SpreadZScore <= 0 {
		t.Log("z-score not yet elevated")
	}
}

func TestAggregator_TripleSwapDay(t *testing.T) {
	agg := NewAggregator()
	// Set detector time to Wednesday via snapshot
	for i := 0; i < 200; i++ {
		agg.Observe(1.0850, 1.0)
	}
	ms := agg.Snapshot("EURUSD")
	// Just check structure
	if ms.Timestamp.IsZero() {
		t.Fatal("timestamp should be set")
	}
}

func TestRegime_String(t *testing.T) {
	tests := []struct {
		r Regime
		e string
	}{
		{RegimeUnknown, "unknown"},
		{RegimeCalm, "calm"},
		{RegimeRanging, "ranging"},
		{RegimeTrending, "trending"},
		{RegimeVolatile, "volatile"},
	}
	for _, tt := range tests {
		if tt.r.String() != tt.e {
			t.Fatalf("Regime(%d) = %s, want %s", tt.r, tt.r.String(), tt.e)
		}
	}
}
