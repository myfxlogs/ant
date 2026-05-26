package controlplane

import (
	"testing"
	"time"
)

// --- Kill Switch Tests ---

func TestKillSwitch_InitiallyDisengaged(t *testing.T) {
	ks := NewKillSwitch()
	if ks.IsEngaged() {
		t.Fatal("kill switch should be disengaged initially")
	}
	engaged, reason, _ := ks.Status()
	if engaged {
		t.Fatalf("should be disengaged, got engaged with reason=%s", reason)
	}
}

func TestKillSwitch_EngageDisengage(t *testing.T) {
	ks := NewKillSwitch()
	ks.Engage("emergency stop: data corruption detected")
	if !ks.IsEngaged() {
		t.Fatal("kill switch should be engaged")
	}
	engaged, reason, since := ks.Status()
	if !engaged || reason == "" || since.IsZero() {
		t.Fatalf("status: engaged=%v reason=%s since=%v", engaged, reason, since)
	}

	ks.Disengage()
	if ks.IsEngaged() {
		t.Fatal("kill switch should be disengaged after Disengage")
	}
}

func TestKillSwitch_ConcurrentAccess(t *testing.T) {
	ks := NewKillSwitch()
	done := make(chan struct{})

	go func() {
		for i := 0; i < 1000; i++ {
			ks.Engage("test")
			ks.Disengage()
		}
		close(done)
	}()

	for i := 0; i < 1000; i++ {
		ks.IsEngaged()
		ks.Status()
	}
	<-done
}

// --- Strategy Breaker Tests ---

func TestStrategyBreaker_NoTripInitially(t *testing.T) {
	cfg := DefaultStrategyBreakerConfig()
	sb := NewStrategyBreaker(cfg)

	if sb.IsTripped() {
		t.Fatal("breaker should not be tripped initially")
	}
}

func TestStrategyBreaker_TripsOnLoss(t *testing.T) {
	cfg := DefaultStrategyBreakerConfig()
	cfg.MaxLossPercent = 1.0 // trip on >1% loss
	cfg.MinSampleTrades = 3
	sb := NewStrategyBreaker(cfg)

	// Simulate losses.
	for i := 0; i < 10; i++ {
		tripped, _ := sb.RecordPnL(-500) // $500 loss each, cumulative -$5000 on $100k = 5%
		if tripped {
			t.Logf("tripped on iteration %d", i)
			break
		}
	}

	if !sb.IsTripped() {
		t.Fatal("breaker should trip after cumulative losses exceed threshold")
	}
}

func TestStrategyBreaker_AutoResetAfterCooldown(t *testing.T) {
	cfg := DefaultStrategyBreakerConfig()
	cfg.MaxLossPercent = 1.0
	cfg.MinSampleTrades = 3
	cfg.CooldownDuration = 50 * time.Millisecond
	sb := NewStrategyBreaker(cfg)

	// Trip the breaker.
	for i := 0; i < 10; i++ {
		tripped, _ := sb.RecordPnL(-500)
		if tripped {
			break
		}
	}
	if !sb.IsTripped() {
		t.Fatal("breaker should be tripped")
	}

	// Wait for cooldown.
	time.Sleep(100 * time.Millisecond)

	if sb.IsTripped() {
		t.Fatal("breaker should auto-reset after cooldown")
	}
}

func TestStrategyBreaker_ProfitsNoTrip(t *testing.T) {
	cfg := DefaultStrategyBreakerConfig()
	cfg.MinSampleTrades = 3
	sb := NewStrategyBreaker(cfg)

	// Profitable trades should not trip.
	for i := 0; i < 20; i++ {
		tripped, _ := sb.RecordPnL(100) // $100 profit each
		if tripped {
			t.Fatal("profitable trades should not trip breaker")
		}
	}
}

func TestStrategyBreaker_InsufficientTrades(t *testing.T) {
	cfg := DefaultStrategyBreakerConfig()
	cfg.MinSampleTrades = 5
	sb := NewStrategyBreaker(cfg)

	// Only 2 trades — not enough to evaluate.
	for i := 0; i < 2; i++ {
		tripped, _ := sb.RecordPnL(-1000)
		if tripped {
			t.Fatal("insufficient trades should not trip breaker")
		}
	}
}

func TestStrategyBreaker_Reset(t *testing.T) {
	cfg := DefaultStrategyBreakerConfig()
	cfg.MaxLossPercent = 1.0
	cfg.MinSampleTrades = 3
	sb := NewStrategyBreaker(cfg)

	for i := 0; i < 10; i++ {
		sb.RecordPnL(-500)
	}
	if !sb.IsTripped() {
		t.Fatal("should be tripped")
	}

	sb.Reset()
	if sb.IsTripped() {
		t.Fatal("should be reset")
	}
}

// --- Canary Tests ---

func TestCanary_NotReadyWithoutStart(t *testing.T) {
	cfg := DefaultCanaryConfig()
	c := NewCanary(cfg)

	ready, reason := c.ReadyForPromotion(2.0, 5.0)
	if ready {
		t.Fatalf("should not be ready: reason=%s", reason)
	}
}

func TestCanary_NotReadyBeforeDuration(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.CanaryDuration = 1 * time.Hour
	c := NewCanary(cfg)

	c.Start()
	for i := 0; i < 100; i++ {
		c.RecordTrade(50)
	}

	ready, _ := c.ReadyForPromotion(2.0, 5.0)
	if ready {
		t.Fatal("should not be ready before canary duration elapses")
	}
}

func TestCanary_RejectsWithLowSharpe(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.CanaryDuration = 0 // immediate
	cfg.MinTrades = 5
	c := NewCanary(cfg)

	c.Start()
	for i := 0; i < 10; i++ {
		c.RecordTrade(50)
	}

	ready, reason := c.ReadyForPromotion(0.1, 5.0)
	if ready {
		t.Fatalf("low Sharpe should reject: reason=%s", reason)
	}
	t.Logf("Rejected for low Sharpe: %s", reason)
}

func TestCanary_RejectsWithHighDrawdown(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.CanaryDuration = 0
	cfg.MinTrades = 5
	c := NewCanary(cfg)

	c.Start()
	for i := 0; i < 10; i++ {
		c.RecordTrade(50)
	}

	ready, reason := c.ReadyForPromotion(2.0, 30.0) // drawdown 30% > 20% max
	if ready {
		t.Fatalf("high drawdown should reject: reason=%s", reason)
	}
	t.Logf("Rejected for high drawdown: %s", reason)
}

func TestCanary_RejectsNonPositivePnL(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.CanaryDuration = 0
	cfg.MinTrades = 5
	c := NewCanary(cfg)

	c.Start()
	for i := 0; i < 10; i++ {
		c.RecordTrade(-100)
	}

	ready, reason := c.ReadyForPromotion(2.0, 5.0)
	if ready {
		t.Fatalf("negative P&L should reject: reason=%s", reason)
	}
}

func TestCanary_ReadyWhenAllPass(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.CanaryDuration = 0
	cfg.MinTrades = 5
	c := NewCanary(cfg)

	c.Start()
	for i := 0; i < 10; i++ {
		c.RecordTrade(100)
	}

	ready, reason := c.ReadyForPromotion(1.5, 10.0)
	if !ready {
		t.Fatalf("should be ready: reason=%s", reason)
	}
	t.Logf("Promoted: %s", reason)
}

func TestCanary_Status(t *testing.T) {
	cfg := DefaultCanaryConfig()
	c := NewCanary(cfg)

	// Status before start.
	elapsed, trades, cumPnL, promoted := c.Status()
	if elapsed != 0 || trades != 0 || cumPnL != 0 || promoted {
		t.Fatal("status before start should be all zero")
	}

	c.Start()
	c.RecordTrade(50)
	c.RecordTrade(-20)

	elapsed, trades, cumPnL, promoted = c.Status()
	if trades != 2 || cumPnL != 30 || promoted {
		t.Fatalf("status: trades=%d cumPnL=%.0f promoted=%v", trades, cumPnL, promoted)
	}
}
