package execalgo

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestShortfall_FrontLoaded(t *testing.T) {
	t.Parallel()
	algo := NewShortfall(3.0, 8) // urgency clamped to 1.0
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(1.0),
		StartTime: refTime(), EndTime: refTime().Add(40 * time.Minute),
		ArrivalPrice: decFromFloat(1.0850),
	}
	sched, err := algo.Schedule(parent)
	if err != nil {
		t.Fatal(err)
	}
	// First slice should be larger than last
	if !sched.Slices[0].Volume.GreaterThan(sched.Slices[len(sched.Slices)-1].Volume) {
		t.Errorf("front-loading: first=%s <= last=%s",
			sched.Slices[0].Volume.String(), sched.Slices[len(sched.Slices)-1].Volume.String())
	}
}

func TestShortfall_ZeroUrgencyIsUniform(t *testing.T) {
	t.Parallel()
	algo := NewShortfall(0, 10)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(1.0),
		StartTime: refTime(), EndTime: refTime().Add(50 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	// All slices should be ~equal at urgency=0
	first := sched.Slices[0].Volume
	last := sched.Slices[len(sched.Slices)-1].Volume
	if !closeEnoughAlgo(first, last) {
		t.Errorf("zero urgency: first=%s last=%s — should be nearly equal", first.String(), last.String())
	}
}

func TestShortfall_MaxUrgencyAllInFirst(t *testing.T) {
	t.Parallel()
	algo := NewShortfall(1.0, 10)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "sell", TotalVolume: decFromFloat(1.0),
		StartTime: refTime(), EndTime: refTime().Add(50 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	// First slice should dominate
	if sched.Slices[0].Volume.LessThan(decFromFloat(0.1)) {
		t.Errorf("high urgency: first slice too small (%s)", sched.Slices[0].Volume.String())
	}
}

func TestShortfall_TotalVolumeMatches(t *testing.T) {
	t.Parallel()
	algo := NewShortfall(0.5, 7)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(2.5),
		StartTime: refTime(), EndTime: refTime().Add(70 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	if !closeEnoughAlgo(sched.TotalScheduledVolume(), parent.TotalVolume) {
		t.Errorf("total = %s, want %s", sched.TotalScheduledVolume().String(), parent.TotalVolume.String())
	}
}

func TestShortfall_SingleSlice(t *testing.T) {
	t.Parallel()
	algo := NewShortfall(0.5, 1)
	parent := ParentOrder{
		Symbol: "EURUSD", Side: "buy", TotalVolume: decFromFloat(1.0),
		StartTime: refTime(), EndTime: refTime().Add(10 * time.Minute),
	}
	sched, _ := algo.Schedule(parent)
	if len(sched.Slices) != 1 {
		t.Fatalf("expected 1 slice, got %d", len(sched.Slices))
	}
	if !closeEnoughAlgo(sched.Slices[0].Volume, decFromFloat(1.0)) {
		t.Errorf("volume = %s, want 1.0", sched.Slices[0].Volume.String())
	}
}

func TestShortfall_ZeroVolume(t *testing.T) {
	t.Parallel()
	algo := NewShortfall(0.5, 5)
	_, err := algo.Schedule(ParentOrder{Side: "buy", TotalVolume: decimal.Zero, StartTime: refTime(), EndTime: refTime().Add(time.Hour)})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestShortfall_UrgencyOutsideRange(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	algo := NewShortfall(0.5, 5)
	if algo.Name() != "ImplementationShortfall" {
		t.Errorf("Name = %s", algo.Name())
	}
}

// ---- Schedule Validation ----
