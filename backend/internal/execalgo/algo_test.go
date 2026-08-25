package execalgo

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// ---- TWAP ----

func TestTwap_EqualSlices(t *testing.T) {
	t.Parallel()
	algo := NewTwap(5 * time.Minute)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(1.0),
		StartTime: refTime(), EndTime: refTime().Add(20 * time.Minute),
	}
	sched, err := algo.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(sched.Slices) != 4 {
		t.Fatalf("expected 4 slices, got %d", len(sched.Slices))
	}
	// All slices should have equal volume (1.0/4 = 0.25)
	for i, c := range sched.Slices {
		if !closeEnoughAlgo(c.Volume, decFromFloat(0.25)) {
			t.Errorf("slice %d volume = %s, want 0.25", i, c.Volume.String())
		}
	}
	// Check spacing
	for i := 0; i < len(sched.Slices)-1; i++ {
		gap := sched.Slices[i+1].TargetTime.Sub(sched.Slices[i].TargetTime)
		if gap != 5*time.Minute {
			t.Errorf("gap between slice %d and %d = %v, want 5m", i, i+1, gap)
		}
	}
	// First slice after start
	if !sched.Slices[0].TargetTime.After(parent.StartTime) {
		t.Error("first slice should be after start time")
	}
	if err := sched.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestTwap_SingleSliceIfDurationEqualsInterval(t *testing.T) {
	t.Parallel()
	algo := NewTwap(time.Minute)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "sell", TotalVolume: decFromFloat(0.5),
		StartTime: refTime(), EndTime: refTime().Add(time.Minute),
	}
	sched, err := algo.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(sched.Slices) != 1 {
		t.Fatalf("expected 1 slice, got %d", len(sched.Slices))
	}
	if !closeEnoughAlgo(sched.Slices[0].Volume, decFromFloat(0.5)) {
		t.Errorf("volume = %s, want 0.5", sched.Slices[0].Volume.String())
	}
}

func TestTwap_TotalVolumeMatches(t *testing.T) {
	t.Parallel()
	algo := NewTwap(2 * time.Minute)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(3.0),
		StartTime: refTime(), EndTime: refTime().Add(10 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	if !closeEnoughAlgo(sched.TotalScheduledVolume(), parent.TotalVolume) {
		t.Errorf("total scheduled = %s, want %s", sched.TotalScheduledVolume().String(), parent.TotalVolume.String())
	}
}

func TestTwap_ZeroVolume(t *testing.T) {
	t.Parallel()
	algo := NewTwap(time.Minute)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decimal.Zero,
		StartTime: refTime(), EndTime: refTime().Add(10 * time.Minute),
	}
	_, err := algo.Schedule(parent)
	if err == nil {
		t.Fatal("expected error for zero volume")
	}
}

func TestTwap_ZeroDuration(t *testing.T) {
	t.Parallel()
	algo := NewTwap(time.Minute)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(1.0),
		StartTime: refTime(), EndTime: refTime(),
	}
	_, err := algo.Schedule(parent)
	if err == nil {
		t.Fatal("expected error for zero duration")
	}
}

func TestTwap_DefaultInterval(t *testing.T) {
	t.Parallel()
	algo := NewTwap(0)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(1.0),
		StartTime: refTime(), EndTime: refTime().Add(5 * time.Minute),
	}
	sched, err := algo.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(sched.Slices) == 0 {
		t.Fatal("expected at least 1 slice")
	}
}

func TestTwap_Name(t *testing.T) {
	t.Parallel()
	algo := NewTwap(time.Minute)
	if algo.Name() != "TWAP" {
		t.Errorf("Name = %s, want TWAP", algo.Name())
	}
}

// ---- VWAP ----

func TestVwap_FlatProfileEqualsTwap(t *testing.T) {
	t.Parallel()
	vwap := NewVwap(FlatVolumeProfile{}, 6)
	twap := NewTwap(10 * time.Minute)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(1.0),
		StartTime: refTime(), EndTime: refTime().Add(60 * time.Minute),
	}
	vwapSched, _ := vwap.Schedule(parent)
	twapSched, _ := twap.Schedule(parent)

	if len(vwapSched.Slices) != 6 {
		t.Fatalf("vwap slices = %d, want 6", len(vwapSched.Slices))
	}
	// With flat profile, each bucket gets 1/6 of volume
	expectedVol := decFromFloat(1.0 / 6.0)
	for i, c := range vwapSched.Slices {
		if !closeEnoughAlgo(c.Volume, expectedVol) {
			t.Errorf("vwap slice %d volume = %s, want %s", i, c.Volume.String(), expectedVol.String())
		}
	}
	_ = twapSched
}

type customProfile struct {
	fractions map[int]float64 // hour → fraction
}

func (p customProfile) Fraction(_ string, bucketStart time.Time) float64 {
	if f, ok := p.fractions[bucketStart.Hour()]; ok {
		return f
	}
	return 0.1
}

func TestVwap_CustomProfile(t *testing.T) {
	t.Parallel()
	// Peak hours 9-11 get more volume
	profile := customProfile{
		fractions: map[int]float64{
			10: 0.4, // 40% of daily volume
			11: 0.3, // 30%
			12: 0.2, // 20%
			13: 0.1, // 10%
		},
	}
	vwap := NewVwap(profile, 4)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(1.0),
		StartTime: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
	}
	sched, err := vwap.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(sched.Slices) != 4 {
		t.Fatalf("slices = %d, want 4", len(sched.Slices))
	}
	// First slice (10:00) should be largest (0.4 weight), last (13:00) smallest (0.1)
	if !sched.Slices[0].Volume.GreaterThan(sched.Slices[3].Volume) {
		t.Errorf("first slice (%s) should be larger than last (%s)",
			sched.Slices[0].Volume.String(), sched.Slices[3].Volume.String())
	}
}

func TestVwap_NilProfileDefaultsToFlat(t *testing.T) {
	t.Parallel()
	vwap := NewVwap(nil, 5)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(1.0),
		StartTime: refTime(), EndTime: refTime().Add(60 * time.Minute),
	}
	sched, err := vwap.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(sched.Slices) != 5 {
		t.Fatalf("slices = %d, want 5", len(sched.Slices))
	}
}

func TestVwap_ZeroVolume(t *testing.T) {
	t.Parallel()
	vwap := NewVwap(FlatVolumeProfile{}, 5)
	_, err := vwap.Schedule(ParentOrder{Side: "buy", TotalVolume: decimal.Zero, StartTime: refTime(), EndTime: refTime().Add(time.Hour)})
	if err == nil {
		t.Fatal("expected error for zero volume")
	}
}

func TestVwap_NegativeProfileFraction(t *testing.T) {
	t.Parallel()
	profile := customProfile{fractions: map[int]float64{10: -0.5}}
	vwap := NewVwap(profile, 3)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(1.0),
		StartTime: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2025, 6, 15, 13, 0, 0, 0, time.UTC),
	}
	sched, err := vwap.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	// Negative fractions are clamped to 0; remaining buckets get proportional volume
	total := sched.TotalScheduledVolume()
	if !closeEnoughAlgo(total, decFromFloat(1.0)) {
		t.Errorf("total volume = %s, want 1.0", total.String())
	}
}

// ---- POV ----

func TestPov_RespectsParticipationRate(t *testing.T) {
	t.Parallel()
	algo := NewPov(0.1, time.Minute, 1.0) // 10% of 1.0 lot per minute
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(0.3),
		StartTime: refTime(), EndTime: refTime().Add(5 * time.Minute),
	}
	sched, err := algo.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	// Each slice should be at most 0.1 * 1.0 = 0.1
	cap := decFromFloat(0.1 + 0.0001)
	for i, c := range sched.Slices {
		if c.Volume.GreaterThan(cap) {
			t.Errorf("slice %d volume = %s exceeds rate cap 0.1", i, c.Volume.String())
		}
	}
	// Total should equal parent
	if !closeEnoughAlgo(sched.TotalScheduledVolume(), parent.TotalVolume) {
		t.Errorf("total = %s, want %s", sched.TotalScheduledVolume().String(), parent.TotalVolume.String())
	}
}

func TestPov_StopsWhenVolumeExhausted(t *testing.T) {
	t.Parallel()
	algo := NewPov(0.1, time.Minute, 1.0)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "sell", TotalVolume: decFromFloat(0.15),
		StartTime: refTime(), EndTime: refTime().Add(10 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	// Should stop early after volume exhausted (~2 slices: 0.1 + 0.05)
	if len(sched.Slices) > 2 {
		t.Errorf("expected at most 2 slices, got %d", len(sched.Slices))
	}
}

func TestPov_LargeParentSmallRate(t *testing.T) {
	t.Parallel()
	algo := NewPov(0.01, 30*time.Second, 2.0) // 1% of 2 lots per 30s
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(10.0),
		StartTime: refTime(), EndTime: refTime().Add(10 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	// Many small slices
	if len(sched.Slices) < 10 {
		t.Errorf("expected many slices, got %d", len(sched.Slices))
	}
}

func TestPov_ZeroVolume(t *testing.T) {
	t.Parallel()
	algo := NewPov(0.1, time.Minute, 1.0)
	_, err := algo.Schedule(ParentOrder{Side: "buy", TotalVolume: decimal.Zero, StartTime: refTime(), EndTime: refTime().Add(time.Minute)})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPov_ZeroDuration(t *testing.T) {
	t.Parallel()
	algo := NewPov(0.1, time.Minute, 1.0)
	_, err := algo.Schedule(ParentOrder{Side: "buy", TotalVolume: decFromFloat(1.0), StartTime: refTime(), EndTime: refTime()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPov_DefaultRate(t *testing.T) {
	t.Parallel()
	algo := NewPov(0, time.Minute, 1.0) // should default to 0.05
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(0.5),
		StartTime: refTime(), EndTime: refTime().Add(5 * time.Minute),
	}
	sched, err := algo.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(sched.Slices) == 0 {
		t.Fatal("expected slices with default rate")
	}
}

// ---- Implementation Shortfall ----
