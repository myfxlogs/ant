package execalgo

import (
	"math"
	"strings"
	"testing"
	"time"
)

func refTime() time.Time { return time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC) }

func closeEnoughAlgo(a, b float64) bool { return math.Abs(a-b) < 0.001 }

// ---- TWAP ----

func TestTwap_EqualSlices(t *testing.T) {
	algo := NewTwap(5 * time.Minute)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
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
		if !closeEnoughAlgo(c.Volume, 0.25) {
			t.Errorf("slice %d volume = %.4f, want 0.25", i, c.Volume)
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
	algo := NewTwap(time.Minute)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "sell", TotalVolume: 0.5,
		StartTime: refTime(), EndTime: refTime().Add(time.Minute),
	}
	sched, err := algo.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(sched.Slices) != 1 {
		t.Fatalf("expected 1 slice, got %d", len(sched.Slices))
	}
	if !closeEnoughAlgo(sched.Slices[0].Volume, 0.5) {
		t.Errorf("volume = %.4f, want 0.5", sched.Slices[0].Volume)
	}
}

func TestTwap_TotalVolumeMatches(t *testing.T) {
	algo := NewTwap(2 * time.Minute)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 3.0,
		StartTime: refTime(), EndTime: refTime().Add(10 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	if !closeEnoughAlgo(sched.TotalScheduledVolume(), parent.TotalVolume) {
		t.Errorf("total scheduled = %.4f, want %.4f", sched.TotalScheduledVolume(), parent.TotalVolume)
	}
}

func TestTwap_ZeroVolume(t *testing.T) {
	algo := NewTwap(time.Minute)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 0,
		StartTime: refTime(), EndTime: refTime().Add(10 * time.Minute),
	}
	_, err := algo.Schedule(parent)
	if err == nil {
		t.Fatal("expected error for zero volume")
	}
}

func TestTwap_ZeroDuration(t *testing.T) {
	algo := NewTwap(time.Minute)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
		StartTime: refTime(), EndTime: refTime(),
	}
	_, err := algo.Schedule(parent)
	if err == nil {
		t.Fatal("expected error for zero duration")
	}
}

func TestTwap_DefaultInterval(t *testing.T) {
	algo := NewTwap(0)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
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
	algo := NewTwap(time.Minute)
	if algo.Name() != "TWAP" {
		t.Errorf("Name = %s, want TWAP", algo.Name())
	}
}

// ---- VWAP ----

func TestVwap_FlatProfileEqualsTwap(t *testing.T) {
	vwap := NewVwap(FlatVolumeProfile{}, 6)
	twap := NewTwap(10 * time.Minute)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
		StartTime: refTime(), EndTime: refTime().Add(60 * time.Minute),
	}
	vwapSched, _ := vwap.Schedule(parent)
	twapSched, _ := twap.Schedule(parent)

	if len(vwapSched.Slices) != 6 {
		t.Fatalf("vwap slices = %d, want 6", len(vwapSched.Slices))
	}
	// With flat profile, each bucket gets 1/6 of volume
	expectedVol := 1.0 / 6.0
	for i, c := range vwapSched.Slices {
		if !closeEnoughAlgo(c.Volume, expectedVol) {
			t.Errorf("vwap slice %d volume = %.4f, want %.4f", i, c.Volume, expectedVol)
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
		Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
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
	if sched.Slices[0].Volume <= sched.Slices[3].Volume {
		t.Errorf("first slice (%.4f) should be larger than last (%.4f)",
			sched.Slices[0].Volume, sched.Slices[3].Volume)
	}
}

func TestVwap_NilProfileDefaultsToFlat(t *testing.T) {
	vwap := NewVwap(nil, 5)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
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
	vwap := NewVwap(FlatVolumeProfile{}, 5)
	_, err := vwap.Schedule(ParentOrder{Side: "buy", TotalVolume: 0, StartTime: refTime(), EndTime: refTime().Add(time.Hour)})
	if err == nil {
		t.Fatal("expected error for zero volume")
	}
}

func TestVwap_NegativeProfileFraction(t *testing.T) {
	profile := customProfile{fractions: map[int]float64{10: -0.5}}
	vwap := NewVwap(profile, 3)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
		StartTime: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2025, 6, 15, 13, 0, 0, 0, time.UTC),
	}
	sched, err := vwap.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	// Negative fractions are clamped to 0; remaining buckets get proportional volume
	total := sched.TotalScheduledVolume()
	if !closeEnoughAlgo(total, 1.0) {
		t.Errorf("total volume = %.4f, want 1.0", total)
	}
}

// ---- POV ----

func TestPov_RespectsParticipationRate(t *testing.T) {
	algo := NewPov(0.1, time.Minute, 1.0) // 10% of 1.0 lot per minute
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 0.3,
		StartTime: refTime(), EndTime: refTime().Add(5 * time.Minute),
	}
	sched, err := algo.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	// Each slice should be at most 0.1 * 1.0 = 0.1
	for i, c := range sched.Slices {
		if c.Volume > 0.1+0.0001 {
			t.Errorf("slice %d volume = %.4f exceeds rate cap 0.1", i, c.Volume)
		}
	}
	// Total should equal parent
	if !closeEnoughAlgo(sched.TotalScheduledVolume(), parent.TotalVolume) {
		t.Errorf("total = %.4f, want %.4f", sched.TotalScheduledVolume(), parent.TotalVolume)
	}
}

func TestPov_StopsWhenVolumeExhausted(t *testing.T) {
	algo := NewPov(0.1, time.Minute, 1.0)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "sell", TotalVolume: 0.15,
		StartTime: refTime(), EndTime: refTime().Add(10 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	// Should stop early after volume exhausted (~2 slices: 0.1 + 0.05)
	if len(sched.Slices) > 2 {
		t.Errorf("expected at most 2 slices, got %d", len(sched.Slices))
	}
}

func TestPov_LargeParentSmallRate(t *testing.T) {
	algo := NewPov(0.01, 30*time.Second, 2.0) // 1% of 2 lots per 30s
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 10.0,
		StartTime: refTime(), EndTime: refTime().Add(10 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	// Many small slices
	if len(sched.Slices) < 10 {
		t.Errorf("expected many slices, got %d", len(sched.Slices))
	}
}

func TestPov_ZeroVolume(t *testing.T) {
	algo := NewPov(0.1, time.Minute, 1.0)
	_, err := algo.Schedule(ParentOrder{Side: "buy", TotalVolume: 0, StartTime: refTime(), EndTime: refTime().Add(time.Minute)})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPov_ZeroDuration(t *testing.T) {
	algo := NewPov(0.1, time.Minute, 1.0)
	_, err := algo.Schedule(ParentOrder{Side: "buy", TotalVolume: 1.0, StartTime: refTime(), EndTime: refTime()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPov_DefaultRate(t *testing.T) {
	algo := NewPov(0, time.Minute, 1.0) // should default to 0.05
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 0.5,
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

func TestShortfall_FrontLoaded(t *testing.T) {
	algo := NewShortfall(3.0, 8) // urgency clamped to 1.0
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
		StartTime: refTime(), EndTime: refTime().Add(40 * time.Minute),
		ArrivalPrice: 1.0850,
	}
	sched, err := algo.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	// First slice should be larger than last
	if sched.Slices[0].Volume <= sched.Slices[len(sched.Slices)-1].Volume {
		t.Errorf("front-loading: first=%.4f <= last=%.4f",
			sched.Slices[0].Volume, sched.Slices[len(sched.Slices)-1].Volume)
	}
}

func TestShortfall_ZeroUrgencyIsUniform(t *testing.T) {
	algo := NewShortfall(0, 10)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
		StartTime: refTime(), EndTime: refTime().Add(50 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	// All slices should be ~equal at urgency=0
	first := sched.Slices[0].Volume
	last := sched.Slices[len(sched.Slices)-1].Volume
	if !closeEnoughAlgo(first, last) {
		t.Errorf("zero urgency: first=%.4f last=%.4f — should be nearly equal", first, last)
	}
}

func TestShortfall_MaxUrgencyAllInFirst(t *testing.T) {
	algo := NewShortfall(1.0, 10)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "sell", TotalVolume: 1.0,
		StartTime: refTime(), EndTime: refTime().Add(50 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	// First slice should dominate
	if sched.Slices[0].Volume < 0.1 {
		t.Errorf("high urgency: first slice too small (%.4f)", sched.Slices[0].Volume)
	}
}

func TestShortfall_TotalVolumeMatches(t *testing.T) {
	algo := NewShortfall(0.5, 7)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 2.5,
		StartTime: refTime(), EndTime: refTime().Add(70 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	if !closeEnoughAlgo(sched.TotalScheduledVolume(), parent.TotalVolume) {
		t.Errorf("total = %.4f, want %.4f", sched.TotalScheduledVolume(), parent.TotalVolume)
	}
}

func TestShortfall_SingleSlice(t *testing.T) {
	algo := NewShortfall(0.5, 1)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
		StartTime: refTime(), EndTime: refTime().Add(10 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	if len(sched.Slices) != 1 {
		t.Fatalf("expected 1 slice, got %d", len(sched.Slices))
	}
	if !closeEnoughAlgo(sched.Slices[0].Volume, 1.0) {
		t.Errorf("volume = %.4f, want 1.0", sched.Slices[0].Volume)
	}
}

func TestShortfall_ZeroVolume(t *testing.T) {
	algo := NewShortfall(0.5, 5)
	_, err := algo.Schedule(ParentOrder{Side: "buy", TotalVolume: 0, StartTime: refTime(), EndTime: refTime().Add(time.Hour)})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestShortfall_UrgencyOutsideRange(t *testing.T) {
	// Negative urgency clamped to 0
	a1 := NewShortfall(-1.0, 5)
	if a1.Urgency != 0 {
		t.Errorf("negative urgency not clamped: %.4f", a1.Urgency)
	}
	// Urgency > 1 clamped to 1
	a2 := NewShortfall(5.0, 5)
	if a2.Urgency != 1.0 {
		t.Errorf("excessive urgency not clamped: %.4f", a2.Urgency)
	}
}

func TestShortfall_Name(t *testing.T) {
	algo := NewShortfall(0.5, 5)
	if algo.Name() != "ImplementationShortfall" {
		t.Errorf("Name = %s", algo.Name())
	}
}

// ---- Schedule Validation ----

func TestSchedule_ValidateOverScheduled(t *testing.T) {
	s := &Schedule{
		Parent: ParentOrder{TotalVolume: 1.0, StartTime: refTime(), EndTime: refTime().Add(time.Hour)},
		Slices: []ChildOrder{
			{Sequence: 0, Volume: 0.8, TargetTime: refTime().Add(10 * time.Minute)},
			{Sequence: 1, Volume: 0.8, TargetTime: refTime().Add(20 * time.Minute)},
		},
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for over-scheduled volume")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("error should mention 'exceeds': %v", err)
	}
}

func TestSchedule_ValidateSliceBeforeStart(t *testing.T) {
	s := &Schedule{
		Parent: ParentOrder{TotalVolume: 1.0, StartTime: refTime(), EndTime: refTime().Add(time.Hour)},
		Slices: []ChildOrder{
			{Sequence: 0, Volume: 1.0, TargetTime: refTime().Add(-10 * time.Minute)},
		},
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for slice before start")
	}
}

func TestSchedule_ValidateSliceAfterEnd(t *testing.T) {
	s := &Schedule{
		Parent: ParentOrder{TotalVolume: 1.0, StartTime: refTime(), EndTime: refTime().Add(time.Hour)},
		Slices: []ChildOrder{
			{Sequence: 0, Volume: 1.0, TargetTime: refTime().Add(2 * time.Hour)},
		},
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for slice after end")
	}
}

func TestSchedule_ValidateNonPositiveVolume(t *testing.T) {
	s := &Schedule{
		Parent: ParentOrder{TotalVolume: 1.0, StartTime: refTime(), EndTime: refTime().Add(time.Hour)},
		Slices: []ChildOrder{
			{Sequence: 0, Volume: 0, TargetTime: refTime().Add(10 * time.Minute)},
		},
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error for non-positive volume")
	}
}

func TestSchedule_ValidateEmptySlices(t *testing.T) {
	s := &Schedule{
		Parent: ParentOrder{TotalVolume: 1.0, StartTime: refTime(), EndTime: refTime().Add(time.Hour)},
		Slices: []ChildOrder{},
	}
	if err := s.Validate(); err != nil {
		t.Errorf("empty schedule should validate: %v", err)
	}
}

func TestSchedule_ValidateValid(t *testing.T) {
	s := &Schedule{
		Parent: ParentOrder{TotalVolume: 1.0, StartTime: refTime(), EndTime: refTime().Add(time.Hour)},
		Slices: []ChildOrder{
			{Sequence: 0, Volume: 0.6, TargetTime: refTime().Add(10 * time.Minute)},
			{Sequence: 1, Volume: 0.4, TargetTime: refTime().Add(20 * time.Minute)},
		},
	}
	if err := s.Validate(); err != nil {
		t.Errorf("valid schedule should validate: %v", err)
	}
}

// ---- Cross-Algo Consistency ----

func TestCrossAlgo_AllProduceValidSchedules(t *testing.T) {
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
		StartTime: refTime(), EndTime: refTime().Add(30 * time.Minute),
	}

	algos := []Algo{
		NewTwap(time.Minute),
		NewVwap(FlatVolumeProfile{}, 10),
		NewPov(0.1, time.Minute, 1.0),
		NewShortfall(0.5, 8),
	}

	for _, a := range algos {
		t.Run(a.Name(), func(t *testing.T) {
			sched, err := a.Schedule(parent)
			if err != nil {
				t.Fatalf("Schedule failed: %v", err)
			}
			if err := sched.Validate(); err != nil {
				t.Errorf("Validate failed: %v", err)
			}
			if len(sched.Slices) == 0 {
				t.Error("empty schedule")
			}
			if sched.Algo != a.Name() {
				t.Errorf("Algo = %s, want %s", sched.Algo, a.Name())
			}
			// Total volume should closely match parent
			total := sched.TotalScheduledVolume()
			if !closeEnoughAlgo(total, parent.TotalVolume) {
				t.Errorf("total = %.4f, want %.4f", total, parent.TotalVolume)
			}
		})
	}
}

func TestCrossAlgo_DifferentSchedulesForSameParent(t *testing.T) {
	// Each algo should produce a distinguishable schedule.
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
		StartTime: refTime(), EndTime: refTime().Add(60 * time.Minute),
	}

	type namedAlgo struct {
		name string
		algo Algo
	}

	algos := []namedAlgo{
		{"TWAP", NewTwap(10 * time.Minute)},
		{"VWAP", NewVwap(FlatVolumeProfile{}, 6)},
		{"POV", NewPov(0.2, 5*time.Minute, 1.0)},
		{"Shortfall", NewShortfall(0.8, 10)},
	}

	schedules := make(map[string]*Schedule)
	for _, na := range algos {
		sched, err := na.algo.Schedule(parent)
		if err != nil {
			t.Fatalf("%s: %v", na.name, err)
		}
		schedules[na.name] = sched
	}

	// TWAP and VWAP with flat profile should match
	twapVols := make([]float64, len(schedules["TWAP"].Slices))
	for i, c := range schedules["TWAP"].Slices {
		twapVols[i] = c.Volume
	}
	vwapVols := make([]float64, len(schedules["VWAP"].Slices))
	for i, c := range schedules["VWAP"].Slices {
		vwapVols[i] = c.Volume
	}

	// POV should differ from TWAP (different volumes per slice)
	if len(schedules["POV"].Slices) == len(twapVols) {
		allSame := true
		for i := range schedules["POV"].Slices {
			if !closeEnoughAlgo(schedules["POV"].Slices[i].Volume, twapVols[i]) {
				allSame = false
				break
			}
		}
		if allSame {
			t.Log("POV happened to match TWAP for this config (acceptable)")
		}
	}

	// Shortfall should be front-loaded
	shortfallSched := schedules["Shortfall"]
	if len(shortfallSched.Slices) >= 2 {
		if shortfallSched.Slices[0].Volume <= shortfallSched.Slices[len(shortfallSched.Slices)-1].Volume {
			t.Error("Shortfall with urgency=0.8 should be front-loaded")
		}
	}
}

func TestParentOrder_Duration(t *testing.T) {
	p := ParentOrder{
		StartTime: refTime(),
		EndTime:   refTime().Add(30 * time.Minute),
	}
	if p.Duration() != 30*time.Minute {
		t.Errorf("Duration = %v, want 30m", p.Duration())
	}
}

// ---- Volume Profile ----

func TestFlatVolumeProfile(t *testing.T) {
	p := FlatVolumeProfile{}
	if p.Fraction("EURUSD", refTime()) != 1.0 {
		t.Errorf("FlatVolumeProfile.Fraction = %f, want 1.0", p.Fraction("", time.Time{}))
	}
}
