package dataquality

import (
	"math"
	"testing"
	"time"
)

// --- GapDetector Tests ---

func TestGapDetector_NoGap(t *testing.T) {
	d := NewGapDetector(60)
	now := time.Now()
	d.Observe(now)
	d.Observe(now.Add(10 * time.Second)) // 10s < 60s threshold
	if d.HasGaps() {
		t.Fatal("10s interval should not be a gap")
	}
}

func TestGapDetector_GapDetected(t *testing.T) {
	d := NewGapDetector(30)
	now := time.Now()
	d.Observe(now)
	hasGap := d.Observe(now.Add(120 * time.Second)) // 120s > 30s threshold
	if !hasGap {
		t.Fatal("120s interval should be detected as gap")
	}
	if !d.HasGaps() {
		t.Fatal("should report has gaps")
	}
}

func TestGapDetector_Stats(t *testing.T) {
	d := NewGapDetector(10)
	now := time.Now()
	d.Observe(now)
	d.Observe(now.Add(30 * time.Second))
	d.Observe(now.Add(90 * time.Second))

	count, maxSec, totalSec := d.Stats()
	if count != 2 {
		t.Fatalf("gap count should be 2, got %d", count)
	}
	if maxSec < 55 || maxSec > 65 {
		t.Fatalf("max gap should be ~60s, got %.1f", maxSec)
	}
	if totalSec < 85 || totalSec > 95 {
		t.Fatalf("total gap should be ~90s, got %.1f", totalSec)
	}
}

func TestGapDetector_Reset(t *testing.T) {
	d := NewGapDetector(10)
	now := time.Now()
	d.Observe(now)
	d.Observe(now.Add(60 * time.Second))
	d.Reset()
	if d.HasGaps() {
		t.Fatal("should have no gaps after reset")
	}
}

// --- StalenessScorer Tests ---

func TestStalenessScorer_Fresh(t *testing.T) {
	s := NewStalenessScorer(300, 900)
	now := time.Now()
	s.Observe(now)
	score := s.Score(now.Add(60 * time.Second)) // 1 min later
	if score < 0.99 {
		t.Fatalf("1 min old should be fresh, score=%.4f", score)
	}
	if s.IsStale(now.Add(60 * time.Second)) {
		t.Fatal("1 min old should not be stale")
	}
}

func TestStalenessScorer_Stale(t *testing.T) {
	s := NewStalenessScorer(300, 900)
	now := time.Now()
	s.Observe(now)
	if !s.IsStale(now.Add(400 * time.Second)) {
		t.Fatal("400s should be stale (>300s threshold)")
	}
	score := s.Score(now.Add(400 * time.Second))
	if score >= 1.0 {
		t.Fatalf("stale should have score < 1.0, got %.4f", score)
	}
}

func TestStalenessScorer_Dead(t *testing.T) {
	s := NewStalenessScorer(300, 900)
	now := time.Now()
	s.Observe(now)
	if !s.IsDead(now.Add(1000 * time.Second)) {
		t.Fatal("1000s should be dead (>900s threshold)")
	}
	if s.Score(now.Add(1000*time.Second)) != 0 {
		t.Fatal("dead should have score 0")
	}
}

func TestStalenessScorer_NeverObserved(t *testing.T) {
	s := NewStalenessScorer(300, 900)
	now := time.Now()
	if s.Score(now) != 0 {
		t.Fatal("never observed should have score 0")
	}
	if !s.IsDead(now) {
		t.Fatal("never observed should be dead")
	}
	if s.SinceLastTick(now) != -1 {
		t.Fatal("never observed should return -1")
	}
}

// --- CrossSourceValidator Tests ---

func TestCrossSourceValidator_SingleSource(t *testing.T) {
	v := NewCrossSourceValidator(0.01)
	v.Observe("mt5-ic", 1.0850, 1.0851)
	valid, _, count := v.Validate()
	if !valid {
		t.Fatal("single source should always be valid")
	}
	if count != 1 {
		t.Fatalf("source count: %d", count)
	}
}

func TestCrossSourceValidator_Consistent(t *testing.T) {
	v := NewCrossSourceValidator(0.01)
	v.Observe("mt5-ic", 1.0850, 1.0851)
	v.Observe("mt4-pep", 1.0851, 1.0852)
	valid, maxDev, count := v.Validate()
	if !valid {
		t.Fatalf("consistent prices should be valid, maxDev=%.6f", maxDev)
	}
	if count != 2 {
		t.Fatalf("source count: %d", count)
	}
}

func TestCrossSourceValidator_Divergent(t *testing.T) {
	v := NewCrossSourceValidator(0.005) // 0.5% max deviation
	v.Observe("mt5-ic", 1.0850, 1.0851)  // mid ~1.08505
	v.Observe("mt4-bad", 1.1000, 1.1001) // mid ~1.10005 → 1.4% deviation
	valid, maxDev, count := v.Validate()
	if valid {
		t.Fatalf("should detect divergence, maxDev=%.6f", maxDev)
	}
	if maxDev < 0.01 {
		t.Fatalf("max deviation should be > 1%%, got %.6f", maxDev)
	}
	if count != 2 {
		t.Fatalf("source count: %d", count)
	}
}

func TestCrossSourceValidator_InvalidTick(t *testing.T) {
	v := NewCrossSourceValidator(0.01)
	v.Observe("bad", -1.0, 0.0)   // negative bid
	v.Observe("bad2", 1.10, 1.09) // bid > ask
	if v.SourceCount() != 0 {
		t.Fatalf("invalid ticks should be ignored, got %d sources", v.SourceCount())
	}
}

// --- Monitor / Report Tests ---

func TestMonitor_Healthy(t *testing.T) {
	m := NewMonitor()
	now := time.Now()
	for i := 0; i < 10; i++ {
		m.GapDetector.Observe(now.Add(time.Duration(i) * time.Second))
		m.StalenessScorer.Observe(now.Add(time.Duration(i) * time.Second))
	}
	m.CrossValidator.Observe("src1", 1.0850, 1.0851)

	report := m.Report("EURUSD")
	if report.Status != "healthy" {
		t.Fatalf("should be healthy, got %s", report.Status)
	}
	if report.QualityScore < 0.9 {
		t.Fatalf("quality score should be high, got %.2f", report.QualityScore)
	}
	if len(report.Warnings) != 0 {
		t.Fatalf("should have no warnings, got %v", report.Warnings)
	}
}

func TestMonitor_Degraded(t *testing.T) {
	m := NewMonitor()
	now := time.Now()
	// Only one tick long ago → stale
	m.GapDetector.Observe(now.Add(-400 * time.Second))
	m.StalenessScorer.Observe(now.Add(-400 * time.Second))

	report := m.Report("EURUSD")
	if report.Status == "healthy" {
		t.Fatal("should be degraded or critical")
	}
	if report.QualityScore >= 0.9 {
		t.Fatalf("stale data should reduce quality, got %.2f", report.QualityScore)
	}
}

func TestMonitor_Critical(t *testing.T) {
	m := NewMonitor()
	now := time.Now()
	m.StalenessScorer.Observe(now.Add(-1000 * time.Second)) // dead

	report := m.Report("EURUSD")
	if report.Status != "critical" {
		t.Fatalf("should be critical, got %s", report.Status)
	}
	if report.QualityScore > 0.5 {
		t.Fatalf("dead feed should have very low quality, got %.2f", report.QualityScore)
	}
}

func TestMonitor_Gaps(t *testing.T) {
	m := NewMonitor()
	m.GapDetector = NewGapDetector(10)
	now := time.Now()
	m.StalenessScorer.Observe(now)
	m.GapDetector.Observe(now)
	m.GapDetector.Observe(now.Add(60 * time.Second)) // gap!
	m.StalenessScorer.Observe(now.Add(60 * time.Second))

	report := m.Report("EURUSD")
	if !report.HasGaps {
		t.Fatal("should detect gaps")
	}
}

func TestQualityReport_Defaults(t *testing.T) {
	r := DefaultQualityReport("BTCUSD")
	if r.Symbol != "BTCUSD" {
		t.Fatalf("symbol: %s", r.Symbol)
	}
	if r.Status != "healthy" {
		t.Fatalf("default status: %s", r.Status)
	}
	if math.Abs(r.QualityScore-1.0) > 0.01 {
		t.Fatalf("default score: %.4f", r.QualityScore)
	}
}
