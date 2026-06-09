package controlplane

import (
	"testing"
	"time"
)

func TestStrategyBreaker_InitiallyClosed(t *testing.T) {
	t.Parallel()
	sb := NewStrategyBreaker(DefaultBreakerConfig())
	if sb.state != BreakerClosed {
		t.Fatalf("expected closed, got %s", sb.state)
	}
}

func TestStrategyBreaker_DoesNotTripWithFewTrades(t *testing.T) {
	t.Parallel()
	sb := NewStrategyBreaker(DefaultBreakerConfig())
	now := time.Now()
	// Record losses but below MinSampleTrades (5).
	for i := 0; i < 4; i++ {
		state := sb.RecordPnL(-100, now.Add(time.Duration(i)*time.Minute))
		if state != BreakerClosed {
			t.Fatalf("expected closed with %d trades, got %s", i+1, state)
		}
	}
}

func TestStrategyBreaker_TripsOnExcessiveLoss(t *testing.T) {
	t.Parallel()
	cfg := BreakerConfig{
		MaxLossPercent:  10.0,
		WindowDuration:  30 * time.Minute,
		CoolDown:        1 * time.Hour,
		MinSampleTrades: 3,
	}
	sb := NewStrategyBreaker(cfg)
	now := time.Now()
	// Each trade loses 20 (loss percent = 20 which exceeds 10% threshold).
	for i := 0; i < 3; i++ {
		sb.RecordPnL(-20, now.Add(time.Duration(i)*time.Minute))
	}
	// After 3 trades losing 20 each, avg loss = 20 > 10% → trip.
	if sb.state != BreakerOpen {
		t.Fatalf("expected breaker open, got %s", sb.state)
	}
}

func TestStrategyBreaker_ProfitableTradesKeepsClosed(t *testing.T) {
	t.Parallel()
	sb := NewStrategyBreaker(DefaultBreakerConfig())
	now := time.Now()
	for i := 0; i < 10; i++ {
		state := sb.RecordPnL(50, now.Add(time.Duration(i)*time.Minute))
		if state != BreakerClosed {
			t.Fatalf("expected closed on profitable trades, got %s at trade %d", state, i+1)
		}
	}
}

func TestStrategyBreaker_HalfOpenAfterCooldown(t *testing.T) {
	t.Parallel()
	cfg := BreakerConfig{
		MaxLossPercent:  5.0,
		WindowDuration:  30 * time.Minute,
		CoolDown:        10 * time.Millisecond,
		MinSampleTrades: 2,
	}
	sb := NewStrategyBreaker(cfg)
	now := time.Now()
	sb.RecordPnL(-100, now)
	sb.RecordPnL(-100, now.Add(time.Minute))
	if sb.state != BreakerOpen {
		t.Fatal("expected breaker to be open")
	}
	// After cooldown (10ms) past the tripped time (now+1min), next trade transitions to half-open.
	future := now.Add(time.Minute + 20*time.Millisecond)
	state := sb.RecordPnL(10, future)
	if state != BreakerHalfOpen {
		t.Fatalf("expected half_open after cooldown, got %s", state)
	}
}

func TestStrategyBreaker_Reset(t *testing.T) {
	t.Parallel()
	sb := NewStrategyBreaker(DefaultBreakerConfig())
	now := time.Now()
	for i := 0; i < 10; i++ {
		sb.RecordPnL(-100, now.Add(time.Duration(i)*time.Minute))
	}
	if sb.state != BreakerOpen {
		t.Fatal("expected breaker to be open")
	}
	sb.Reset()
	if sb.state != BreakerClosed {
		t.Fatalf("expected closed after reset, got %s", sb.state)
	}
	if len(sb.samples) != 0 {
		t.Fatalf("expected 0 samples after reset, got %d", len(sb.samples))
	}
}

func TestBreakerRegistry_GetOrCreate(t *testing.T) {
	t.Parallel()
	r := NewBreakerRegistry(DefaultBreakerConfig())
	b1 := r.GetOrCreate("strat-1")
	if b1 == nil {
		t.Fatal("expected non-nil breaker")
	}
	b2 := r.GetOrCreate("strat-1")
	if b1 != b2 {
		t.Fatal("expected same breaker instance")
	}
	b3 := r.GetOrCreate("strat-2")
	if b1 == b3 {
		t.Fatal("expected different breaker for different strategy")
	}
}

func TestBreakerRegistry_List(t *testing.T) {
	t.Parallel()
	r := NewBreakerRegistry(DefaultBreakerConfig())
	r.GetOrCreate("s1")
	r.GetOrCreate("s2")
	statuses := r.List()
	if len(statuses) != 2 {
		t.Fatalf("expected 2 breakers, got %d", len(statuses))
	}
}

func TestBreakerRegistry_Reset(t *testing.T) {
	t.Parallel()
	r := NewBreakerRegistry(DefaultBreakerConfig())
	sb := r.GetOrCreate("s1")
	now := time.Now()
	for i := 0; i < 10; i++ {
		sb.RecordPnL(-100, now.Add(time.Duration(i)*time.Minute))
	}
	if sb.state != BreakerOpen {
		t.Fatal("expected open")
	}
	r.Reset("s1")
	if sb.state != BreakerClosed {
		t.Fatal("expected closed after registry reset")
	}
}
