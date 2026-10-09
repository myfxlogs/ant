package execalgo

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

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
