// Package factor provides the factor evaluation service with backpressure-aware
// NATS subscriber (M10-BASE-B6). The subscriber bridges NATS bar events to the
// DSL evaluation engine with a bounded channel — when the channel is full,
// events are dropped and the backpressure metric is incremented.
package factor

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// MarketStateReader is the interface factor/subscriber needs from mdgateway.MarketStateTracker.
type MarketStateReader interface {
	Get(broker, canonical string) *MarketStateSnapshot
}

// MarketStateSnapshot is a read-only snapshot of a market state for a symbol.
type MarketStateSnapshot struct {
	IsTradeable bool
}

// SubscriberConfig holds configuration for the bar event subscriber.
type SubscriberConfig struct {
	BufferSize    int           // capacity of the bounded channel (default 1024)
	FinalityDelay time.Duration // minimum age of bar before consumption (M10-BASE-D6)
}

// DefaultSubscriberConfig returns M10 defaults.
func DefaultSubscriberConfig() SubscriberConfig {
	return SubscriberConfig{
		BufferSize:    1024,
		FinalityDelay: 100 * time.Millisecond,
	}
}

// Subscriber receives bar events from NATS with bounded channel backpressure.
type Subscriber struct {
	cfg         SubscriberConfig
	ch          chan *mdtick.Bar
	log         *zap.Logger
	marketState MarketStateReader // M10-BASE-F6: tradability gate

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// NewSubscriber creates a bar event subscriber.
func NewSubscriber(cfg SubscriberConfig, log *zap.Logger) *Subscriber {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1024
	}
	return &Subscriber{
		cfg: cfg,
		ch:  make(chan *mdtick.Bar, cfg.BufferSize),
		log: log,
	}
}

// SetMarketState injects a market state reader for the tradability gate (M10-BASE-F6).
func (s *Subscriber) SetMarketState(ms MarketStateReader) {
	s.marketState = ms
}

// Chan returns the read-side channel for bar events.
func (s *Subscriber) Chan() <-chan *mdtick.Bar { return s.ch }

// Push attempts to enqueue a bar event. Returns false if the channel is full
// (backpressure — the event is dropped and the metric is incremented).
// Also rejects bars that have not met the finality delay (M10-BASE-D6)
// and bars from non-tradeable symbols (M10-BASE-F6).
func (s *Subscriber) Push(bar *mdtick.Bar) bool {
	// Bar finality gate: only consume bars that are old enough.
	if s.cfg.FinalityDelay > 0 && !bar.IsReplay {
		barAge := time.Since(time.UnixMilli(bar.CloseTsUnixMs))
		if barAge < s.cfg.FinalityDelay {
			barFinalitySkipTotal.Add(1)
			return false
		}
	}

	// M10-BASE-F6: MarketState tradability gate.
	if s.marketState != nil {
		if ms := s.marketState.Get(bar.Broker, bar.Canonical); ms != nil && !ms.IsTradeable {
			return false
		}
	}

	select {
	case s.ch <- bar:
		return true
	default:
		recordChanFull()
		return false
	}
}

// Start begins the subscriber loop. In production this subscribes to NATS.
func (s *Subscriber) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.running = true
	ctx, s.cancel = context.WithCancel(ctx)
	go s.loop(ctx)
}

// Stop shuts down the subscriber.
func (s *Subscriber) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Subscriber) loop(ctx context.Context) {
	<-ctx.Done()
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}

// --- Bar finality metric (M10-BASE-D6) ---

var barFinalitySkipTotal atomic.Int64

// BarFinalitySkipTotal returns the count of bars skipped due to finality delay.
func BarFinalitySkipTotal() int64 { return barFinalitySkipTotal.Load() }
