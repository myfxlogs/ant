package runner

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// barCounterStrategy records OnBar calls and implements TickStrategy so
// OnTick is exercised for revision-stability checks.
type barCounterStrategy struct {
	barOnlyStrategy
	bars       int
	tickCalled bool
}

func (s *barCounterStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
	s.bars++
	return nil, nil
}

func (s *barCounterStrategy) OnTick(ctx sdk.Context, bid, ask decimal.Decimal) (*sdk.Signal, error) {
	s.tickCalled = true
	return nil, nil
}

// TestRunner_BarRevision_AdvancesOnBarOnly verifies that:
//  1. Each OnBar call advances the bar revision by exactly 1 (monotonic).
//  2. OnTick does NOT advance the revision (tick hot path must not rebuild cache).
//
// Adversarial proof: delete the r.barRev.Add(1) line in Runner.OnBar → test RED
// (revision stays 0 across OnBar calls). Add r.barRev.Add(1) in OnTick → the
// tick-no-advance assertion goes RED.
func TestRunner_BarRevision_AdvancesOnBarOnly(t *testing.T) {
	r := New(Config{Symbol: "EURUSD", Timeframe: "M1"})
	strat := &barCounterStrategy{}
	r.SetStrategy(strat)

	rev0 := r.barRevision()
	if rev0 != 0 {
		t.Fatalf("initial revision = %d, want 0", rev0)
	}

	// OnBar #1 → revision 1
	bars1 := sdk.BarsToSlice([]sdk.Bar{{Close: dec("1.0"), Timestamp: 1000}})
	_, _ = r.OnBar(context.Background(), bars1, "M1")
	rev1 := r.barRevision()
	if rev1 != 1 {
		t.Errorf("after OnBar #1: revision = %d, want 1", rev1)
	}

	// OnTick → revision must NOT advance
	_, _ = r.OnTick(context.Background(), dec("1.0001"), dec("1.0002"))
	revAfterTick := r.barRevision()
	if revAfterTick != 1 {
		t.Errorf("after OnTick: revision = %d, want 1 (tick must not advance revision)", revAfterTick)
	}
	if !strat.tickCalled {
		t.Error("OnTick was not called on strategy")
	}

	// OnBar #2 → revision 2
	bars2 := sdk.BarsToSlice([]sdk.Bar{{Close: dec("1.0"), Timestamp: 1000}, {Close: dec("1.1"), Timestamp: 2000}})
	_, _ = r.OnBar(context.Background(), bars2, "M1")
	rev2 := r.barRevision()
	if rev2 != 2 {
		t.Errorf("after OnBar #2: revision = %d, want 2", rev2)
	}

	// Multiple OnTick calls → revision stays 2
	for i := 0; i < 100; i++ {
		_, _ = r.OnTick(context.Background(), dec("1.0001"), dec("1.0002"))
	}
	revAfterTicks := r.barRevision()
	if revAfterTicks != 2 {
		t.Errorf("after 100 OnTick: revision = %d, want 2 (ticks must not advance revision)", revAfterTicks)
	}

	// OnBar #3 → revision 3
	bars3 := sdk.BarsToSlice([]sdk.Bar{{Close: dec("1.0"), Timestamp: 1000}, {Close: dec("1.1"), Timestamp: 2000}, {Close: dec("1.2"), Timestamp: 3000}})
	_, _ = r.OnBar(context.Background(), bars3, "M1")
	rev3 := r.barRevision()
	if rev3 != 3 {
		t.Errorf("after OnBar #3: revision = %d, want 3", rev3)
	}
	if strat.bars != 3 {
		t.Errorf("strategy OnBar called %d times, want 3", strat.bars)
	}
}

// TestRunnerBarSource_Revision verifies that runnerBarSource.Revision() returns
// the runner's bar revision, and returns 0 when no runner is attached (stateless path).
func TestRunnerBarSource_Revision(t *testing.T) {
	r := New(Config{Symbol: "EURUSD"})
	bars := sdk.BarsToSlice([]sdk.Bar{{Close: dec("1.0"), Timestamp: 1000}})

	// With runner — revision reflects runner state.
	r.SetStrategy(&barOnlyStrategy{}) // OnBar returns early without a strategy
	srcWithRunner := &runnerBarSource{bars: bars, runner: r}
	if srcWithRunner.Revision() != 0 {
		t.Errorf("Revision() = %d before any OnBar, want 0", srcWithRunner.Revision())
	}
	_, _ = r.OnBar(context.Background(), bars, "M1")
	if srcWithRunner.Revision() != 1 {
		t.Errorf("Revision() = %d after 1 OnBar, want 1", srcWithRunner.Revision())
	}

	// Without runner (stateless barSource path) — returns 0, never panics.
	srcNoRunner := &runnerBarSource{bars: bars}
	if srcNoRunner.Revision() != 0 {
		t.Errorf("Revision() with no runner = %d, want 0", srcNoRunner.Revision())
	}
}
