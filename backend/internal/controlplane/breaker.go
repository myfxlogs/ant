package controlplane

import (
	"sync"
	"time"
)

// BreakerConfig configures automatic strategy circuit breaking.
type BreakerConfig struct {
	MaxLossPercent  float64
	WindowDuration  time.Duration
	CoolDown        time.Duration
	MinSampleTrades int
}

// DefaultBreakerConfig returns sensible defaults.
func DefaultBreakerConfig() BreakerConfig {
	return BreakerConfig{
		MaxLossPercent:  5.0,
		WindowDuration:  30 * time.Minute,
		CoolDown:        1 * time.Hour,
		MinSampleTrades: 5,
	}
}

// PnLSample is a single trade profit/loss record.
type PnLSample struct {
	Timestamp time.Time
	PnL       float64
}

// BreakerState is the current state of a strategy breaker.
type BreakerState string

const (
	BreakerClosed   BreakerState = "closed"
	BreakerOpen     BreakerState = "open"
	BreakerHalfOpen BreakerState = "half_open"
)

// StrategyBreakerStatus is the observable state of one breaker.
type StrategyBreakerStatus struct {
	StrategyID      string       `json:"strategy_id"`
	State           BreakerState `json:"state"`
	TotalPnL        float64      `json:"total_pnl"`
	LossPercent     float64      `json:"loss_percent"`
	TradeCount      int          `json:"trade_count"`
	TrippedAt       string       `json:"tripped_at,omitempty"`
	TripReason      string       `json:"trip_reason,omitempty"`
	AllowProbeTrade bool         `json:"allow_probe_trade"`
}

// StrategyBreaker implements per-strategy circuit breaking.
type StrategyBreaker struct {
	mu         sync.RWMutex
	config     BreakerConfig
	state      BreakerState
	samples    []PnLSample
	trippedAt  time.Time
	tripReason string
}

// NewStrategyBreaker creates a breaker with the given config.
func NewStrategyBreaker(cfg BreakerConfig) *StrategyBreaker {
	return &StrategyBreaker{config: cfg, state: BreakerClosed}
}

// RecordPnL adds a trade result and evaluates the breaker.
func (sb *StrategyBreaker) RecordPnL(pnl float64, ts time.Time) BreakerState {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.samples = append(sb.samples, PnLSample{Timestamp: ts, PnL: pnl})
	cutoff := ts.Add(-sb.config.WindowDuration)
	filtered := make([]PnLSample, 0, len(sb.samples))
	var totalPnL float64
	for _, s := range sb.samples {
		if s.Timestamp.After(cutoff) {
			filtered = append(filtered, s)
			totalPnL += s.PnL
		}
	}
	sb.samples = filtered

	if sb.state == BreakerOpen {
		if ts.After(sb.trippedAt.Add(sb.config.CoolDown)) {
			sb.state = BreakerHalfOpen
		}
		return sb.state
	}

	count := len(filtered)
	if count >= sb.config.MinSampleTrades && totalPnL < 0 {
		avgLoss := -totalPnL / float64(count)
		if avgLoss > sb.config.MaxLossPercent {
			sb.state = BreakerOpen
			sb.trippedAt = ts
			sb.tripReason = "loss exceeded threshold"
		}
	}
	return sb.state
}

// Status returns the current breaker status.
func (sb *StrategyBreaker) Status(strategyID string) StrategyBreakerStatus {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	var totalPnL float64
	for _, s := range sb.samples {
		totalPnL += s.PnL
	}
	lossPct := 0.0
	if len(sb.samples) > 0 && totalPnL < 0 {
		lossPct = -totalPnL / float64(len(sb.samples))
	}

	s := StrategyBreakerStatus{
		StrategyID:  strategyID,
		State:       sb.state,
		TotalPnL:    totalPnL,
		LossPercent: lossPct,
		TradeCount:  len(sb.samples),
	}
	if sb.state == BreakerOpen || sb.state == BreakerHalfOpen {
		s.TrippedAt = sb.trippedAt.Format(time.RFC3339)
		s.TripReason = sb.tripReason
	}
	if sb.state == BreakerHalfOpen {
		s.AllowProbeTrade = true
	}
	return s
}

// Reset manually resets the breaker to closed state.
func (sb *StrategyBreaker) Reset() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.state = BreakerClosed
	sb.samples = nil
	sb.tripReason = ""
}

// BreakerRegistry manages per-strategy breakers.
type BreakerRegistry struct {
	mu       sync.RWMutex
	config   BreakerConfig
	breakers map[string]*StrategyBreaker
}

// NewBreakerRegistry creates a breaker registry.
func NewBreakerRegistry(cfg BreakerConfig) *BreakerRegistry {
	return &BreakerRegistry{config: cfg, breakers: make(map[string]*StrategyBreaker)}
}

// GetOrCreate returns the breaker for a strategy, creating one if needed.
func (r *BreakerRegistry) GetOrCreate(strategyID string) *StrategyBreaker {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.breakers[strategyID]; ok {
		return b
	}
	b := NewStrategyBreaker(r.config)
	r.breakers[strategyID] = b
	return b
}

// List returns status for all registered breakers.
func (r *BreakerRegistry) List() []StrategyBreakerStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []StrategyBreakerStatus
	for id, b := range r.breakers {
		out = append(out, b.Status(id))
	}
	return out
}

// Reset resets the breaker for a strategy.
func (r *BreakerRegistry) Reset(strategyID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.breakers[strategyID]; ok {
		b.Reset()
	}
}
