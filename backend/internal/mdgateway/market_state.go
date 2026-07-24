// Package mdgateway provides MarketState — a per-symbol "tradability" declaration
// aggregating quality metrics into a single go/no-go signal for downstream pipelines.
//
// MarketState upgrades Quality from "drop bad data" to "declare tradability"
// (M10-BASE-F1). Strategy pipelines consume IsTradeable rather than raw ticks.
package mdgateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
)

// SessionPhase describes the current trading session phase.
type SessionPhase string

const (
	PhasePreMarket  SessionPhase = "pre_market"
	PhaseOpen       SessionPhase = "open"
	PhaseLunch      SessionPhase = "lunch"
	PhaseClose      SessionPhase = "close"
	PhasePostMarket SessionPhase = "post_market"
	PhaseWeekend    SessionPhase = "weekend"
	PhaseHoliday    SessionPhase = "holiday"
)

// MarketState aggregates quality signals into a tradability declaration per symbol.
type MarketState struct {
	Symbol string
	Broker string

	// Quote freshness.
	LastQuote  time.Time
	QuoteAgeMs int64

	// Quality Z-scores (> 3 means anomalous).
	SpreadZscore   float64
	TickRateZscore float64
	GapMarker      bool

	// Session awareness.
	SwapWindow    bool
	SessionPhase  SessionPhase
	HolidayMarker bool

	// Triangulation delta (cross-rate consistency check).
	TriangulationDelta float64

	// Derived verdict.
	IsTradeable  bool
	FreezeReason string
}

// MarketStateConfig tunes MarketState evaluation.
type MarketStateConfig struct {
	MaxQuoteAgeMs       int64   // quotes older than this → not tradeable (default 5000)
	MaxSpreadZscore     float64 // spread Z-score above this → anomaly (default 3.0)
	MaxTickRateZscore   float64 // tick rate Z-score above this → quote stuffing (default 4.0)
	MaxTriangulationDelta float64 // max cross-rate deviation (default 0.005 = 0.5%)
}

// DefaultMarketStateConfig returns sensible defaults.
func DefaultMarketStateConfig() MarketStateConfig {
	return MarketStateConfig{
		MaxQuoteAgeMs:         5000,
		MaxSpreadZscore:       3.0,
		MaxTickRateZscore:     4.0,
		MaxTriangulationDelta: 0.005,
	}
}

// MarketStateTracker maintains per-symbol MarketState snapshots.
type MarketStateTracker struct {
	cfg    MarketStateConfig
	mu     sync.RWMutex
	states map[string]*MarketState // key: "broker:canonical"
	conds  map[string]*sync.Cond   // key: "broker:canonical" — signaled on tradability change
}

// NewMarketStateTracker creates a new tracker.
func NewMarketStateTracker(cfg MarketStateConfig) *MarketStateTracker {
	return &MarketStateTracker{
		cfg:    cfg,
		states: make(map[string]*MarketState),
		conds:  make(map[string]*sync.Cond),
	}
}

// Update applies a tick to the market state for its symbol.
// Signals any waiters blocked on WaitTradeable if tradability changed.
func (t *MarketStateTracker) Update(tick *mdtick.Tick) *MarketState {
	key := tick.Broker + ":" + tick.Canonical

	t.mu.Lock()

	ms, ok := t.states[key]
	if !ok {
		ms = &MarketState{
			Symbol: tick.Canonical,
			Broker: tick.Broker,
		}
		t.states[key] = ms
		t.conds[key] = sync.NewCond(&sync.Mutex{})
	}

	prevTradeable := ms.IsTradeable

	ms.LastQuote = Clk.Now()
	ms.QuoteAgeMs = 0

	// Spread in basis points for Z-score tracking.
	bid, _ := tick.Bid.Float64()
	ask, _ := tick.Ask.Float64()
	if bid > 0 {
		ms.SpreadZscore = (ask - bid) / bid * 10000 // spread in bps
	}

	// Evaluate tradability.
	ms.IsTradeable = t.evaluateTradeable(ms)

	t.mu.Unlock()

	// Signal waiters if tradability flipped from false to true.
	if !prevTradeable && ms.IsTradeable {
		if cond, ok := t.conds[key]; ok {
			cond.L.Lock()
			cond.Broadcast()
			cond.L.Unlock()
		}
	}

	return ms
}

// Get returns the current market state for a symbol.
func (t *MarketStateTracker) Get(broker, canonical string) *MarketState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	key := broker + ":" + canonical
	return t.states[key]
}

// All returns all current market states.
func (t *MarketStateTracker) All() []*MarketState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]*MarketState, 0, len(t.states))
	for _, ms := range t.states {
		result = append(result, ms)
	}
	return result
}

// RefreshAges updates QuoteAgeMs for all tracked symbols and re-evaluates tradability.
// Signals any waiters if tradability flips to true.
func (t *MarketStateTracker) RefreshAges(now time.Time) {
	t.mu.Lock()
	for _, ms := range t.states {
		prev := ms.IsTradeable
		ms.QuoteAgeMs = now.Sub(ms.LastQuote).Milliseconds()
		ms.IsTradeable = t.evaluateTradeable(ms)
		if !prev && ms.IsTradeable {
			key := ms.Broker + ":" + ms.Symbol
			if cond, ok := t.conds[key]; ok {
				cond.L.Lock()
				cond.Broadcast()
				cond.L.Unlock()
			}
		}
	}
	t.mu.Unlock()
}

// WaitTradeable blocks until the symbol (broker:canonical) becomes tradeable or ctx is cancelled.
// Uses a condition variable signaled by Update/RefreshAges — no polling.
func (t *MarketStateTracker) WaitTradeable(ctx context.Context, broker, canonical string) error {
	key := broker + ":" + canonical

	t.mu.RLock()
	cond, ok := t.conds[key]
	ms := t.states[key]
	t.mu.RUnlock()

	if !ok || ms == nil {
		return fmt.Errorf("market state not tracked: %s", key)
	}

	// Fast path: already tradeable.
	if ms.IsTradeable {
		return nil
	}

	// Wait on condition variable, with context cancellation via goroutine.
	done := make(chan struct{})
	go func() {
		cond.L.Lock()
		cond.Wait()
		cond.L.Unlock()
		close(done)
	}()

	select {
	case <-ctx.Done():
		// Wake the goroutine so it doesn't leak.
		cond.L.Lock()
		cond.Broadcast()
		cond.L.Unlock()
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (t *MarketStateTracker) evaluateTradeable(ms *MarketState) bool {
	if ms.SessionPhase == PhaseHoliday || ms.SessionPhase == PhaseWeekend {
		ms.FreezeReason = "market closed: " + string(ms.SessionPhase)
		return false
	}
	if ms.QuoteAgeMs > t.cfg.MaxQuoteAgeMs {
		ms.FreezeReason = "stale quote"
		return false
	}
	if ms.SpreadZscore > t.cfg.MaxSpreadZscore {
		ms.FreezeReason = "spread anomaly"
		return false
	}
	if ms.TickRateZscore > t.cfg.MaxTickRateZscore {
		ms.FreezeReason = "quote stuffing detected"
		return false
	}
	if ms.TriangulationDelta > t.cfg.MaxTriangulationDelta {
		ms.FreezeReason = "triangulation deviation"
		return false
	}
	ms.FreezeReason = ""
	return true
}
