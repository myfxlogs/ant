// Package controlplane provides the SRE control plane (M10-BASE-F7).
//
// Three mechanisms:
//   - Kill Switch: global stop_all + cancel_all, gRPC-accessible
//   - Strategy Breaker: auto circuit-break when loss exceeds threshold in rolling window
//   - Canary: new strategy version runs on small account for N days before full rollout
package controlplane

import (
	"fmt"
	"sync"
	"time"
)

// KillSwitch provides emergency stop for all trading activity.
// All methods are safe for concurrent use.
type KillSwitch struct {
	mu       sync.RWMutex
	engaged  bool
	reason   string
	engagedAt time.Time
}

// NewKillSwitch creates a new kill switch (disarmed by default).
func NewKillSwitch() *KillSwitch {
	return &KillSwitch{}
}

// Engage activates the kill switch with a reason.
func (ks *KillSwitch) Engage(reason string) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.engaged = true
	ks.reason = reason
	ks.engagedAt = time.Now()
}

// Disengage deactivates the kill switch.
func (ks *KillSwitch) Disengage() {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.engaged = false
	ks.reason = ""
}

// IsEngaged returns true if the kill switch is active.
func (ks *KillSwitch) IsEngaged() bool {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.engaged
}

// Status returns the current kill switch state.
func (ks *KillSwitch) Status() (engaged bool, reason string, since time.Time) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.engaged, ks.reason, ks.engagedAt
}

// StrategyBreakerConfig configures automatic strategy circuit breaking.
type StrategyBreakerConfig struct {
	WindowDuration  time.Duration // rolling window for loss calculation (default 30min)
	MaxLossPercent  float64       // max loss % before breaker trips (default 5.0 = 5%)
	CooldownDuration time.Duration // how long breaker stays open before auto-reset (default 1h)
	MinSampleTrades int           // minimum trades in window to evaluate (default 5)
}

// DefaultStrategyBreakerConfig returns sensible defaults.
func DefaultStrategyBreakerConfig() StrategyBreakerConfig {
	return StrategyBreakerConfig{
		WindowDuration:   30 * time.Minute,
		MaxLossPercent:   5.0,
		CooldownDuration: 1 * time.Hour,
		MinSampleTrades:  5,
	}
}

// PnLSample is a single trade P&L record.
type PnLSample struct {
	Timestamp time.Time
	PnL       float64 // positive = profit, negative = loss
}

// StrategyBreaker auto circuit-breaks a strategy when cumulative loss exceeds threshold.
type StrategyBreaker struct {
	cfg    StrategyBreakerConfig
	mu     sync.Mutex
	pnls   []PnLSample
	tripped bool
	openAt time.Time
	reason string
}

// NewStrategyBreaker creates a new strategy breaker.
func NewStrategyBreaker(cfg StrategyBreakerConfig) *StrategyBreaker {
	return &StrategyBreaker{cfg: cfg}
}

// RecordPnL records a trade P&L and checks if the breaker should trip.
func (sb *StrategyBreaker) RecordPnL(pnl float64) (tripped bool, reason string) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	// Auto-reset after cooldown.
	if sb.tripped && time.Since(sb.openAt) > sb.cfg.CooldownDuration {
		sb.tripped = false
		sb.pnls = nil
		sb.reason = ""
	}

	if sb.tripped {
		return true, sb.reason
	}

	now := time.Now()
	sb.pnls = append(sb.pnls, PnLSample{Timestamp: now, PnL: pnl})

	// Purge old samples.
	cutoff := now.Add(-sb.cfg.WindowDuration)
	kept := sb.pnls[:0]
	for _, s := range sb.pnls {
		if s.Timestamp.After(cutoff) {
			kept = append(kept, s)
		}
	}
	sb.pnls = kept

	if len(sb.pnls) < sb.cfg.MinSampleTrades {
		return false, ""
	}

	// Sum cumulative P&L in window.
	var cumPnL, initialEquity float64
	initialEquity = 100000 // baseline
	for _, s := range sb.pnls {
		cumPnL += s.PnL
	}
	lossPercent := -cumPnL / initialEquity * 100

	if lossPercent > sb.cfg.MaxLossPercent {
		sb.tripped = true
		sb.openAt = now
		sb.reason = "loss exceeded " + formatPercent(sb.cfg.MaxLossPercent)
		return true, sb.reason
	}

	return false, ""
}

// IsTripped returns true if the breaker is currently open.
func (sb *StrategyBreaker) IsTripped() bool {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.tripped && time.Since(sb.openAt) > sb.cfg.CooldownDuration {
		sb.tripped = false
		sb.pnls = nil
		return false
	}
	return sb.tripped
}

// Reset manually resets the breaker.
func (sb *StrategyBreaker) Reset() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.tripped = false
	sb.pnls = nil
	sb.reason = ""
}

func formatPercent(pct float64) string {
	return fmt.Sprintf("%.1f%%", pct)
}

// CanaryConfig configures canary deployment for new strategy versions.
type CanaryConfig struct {
	CanaryDuration  time.Duration // minimum canary running time (default 7 * 24h = 1 week)
	MinTrades       int           // minimum trades before promotion (default 50)
	MinSharpeRatio  float64       // minimum Sharpe for promotion (default 0.5)
	MaxDrawdownPct  float64       // maximum drawdown for promotion (default 20%)
	CanaryAccountID string        // small account for canary testing
}

// DefaultCanaryConfig returns standard canary parameters.
func DefaultCanaryConfig() CanaryConfig {
	return CanaryConfig{
		CanaryDuration: 7 * 24 * time.Hour,
		MinTrades:      50,
		MinSharpeRatio: 0.5,
		MaxDrawdownPct: 20,
	}
}

// Canary manages canary deployment of new strategy versions.
type Canary struct {
	cfg       CanaryConfig
	mu        sync.Mutex
	startedAt time.Time
	trades    int
	cumPnL    float64
	promoted  bool
}

// NewCanary creates a new canary tracker.
func NewCanary(cfg CanaryConfig) *Canary {
	return &Canary{cfg: cfg}
}

// Start begins the canary period.
func (c *Canary) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.startedAt = time.Now()
	c.trades = 0
	c.cumPnL = 0
	c.promoted = false
}

// RecordTrade records a completed trade during canary.
func (c *Canary) RecordTrade(pnl float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trades++
	c.cumPnL += pnl
}

// ReadyForPromotion checks if the canary has met all promotion criteria.
func (c *Canary) ReadyForPromotion(sharpeRatio, maxDrawdown float64) (bool, string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.promoted {
		return true, "already promoted"
	}
	if c.startedAt.IsZero() {
		return false, "canary not started"
	}
	if time.Since(c.startedAt) < c.cfg.CanaryDuration {
		return false, "canary period not complete"
	}
	if c.trades < c.cfg.MinTrades {
		return false, "insufficient trades"
	}
	if sharpeRatio < c.cfg.MinSharpeRatio {
		return false, "Sharpe ratio below threshold"
	}
	if maxDrawdown > c.cfg.MaxDrawdownPct {
		return false, "drawdown exceeds limit"
	}
	if c.cumPnL <= 0 {
		return false, "non-positive cumulative P&L"
	}

	c.promoted = true
	return true, "ready for full deployment"
}

// Status returns the current canary status.
func (c *Canary) Status() (elapsed time.Duration, trades int, cumPnL float64, promoted bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.startedAt.IsZero() {
		return 0, 0, 0, false
	}
	return time.Since(c.startedAt), c.trades, c.cumPnL, c.promoted
}
