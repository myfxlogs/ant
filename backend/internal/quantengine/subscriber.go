// Package quantengine provides the quantitative signal inference engine
// with backpressure-aware NATS subscriber (M10-BASE-B6).
package quantengine

import (
	"context"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

// Signal represents a trading signal routed from factor inference to OMS.
type Signal struct {
	SignalID  string
	AccountID string
	Symbol    string
	Side      string
	Volume    float64
	Price     float64
	Strategy  string
}

// SubscriberConfig holds configuration for the signal subscriber.
type SubscriberConfig struct {
	BufferSize int // default 1000
}

// DefaultSubscriberConfig returns M10 defaults.
func DefaultSubscriberConfig() SubscriberConfig {
	return SubscriberConfig{BufferSize: 1000}
}

// Subscriber receives trading signals via bounded channel with backpressure.
type Subscriber struct {
	cfg SubscriberConfig
	ch  chan *Signal
	log *zap.Logger

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// NewSubscriber creates a signal subscriber.
func NewSubscriber(cfg SubscriberConfig, log *zap.Logger) *Subscriber {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1000
	}
	return &Subscriber{
		cfg: cfg,
		ch:  make(chan *Signal, cfg.BufferSize),
		log: log,
	}
}

// Chan returns the read-side channel for signals (consumed by the OMS router).
func (s *Subscriber) Chan() <-chan *Signal { return s.ch }

// Push attempts to enqueue a signal. Returns false on backpressure drop.
func (s *Subscriber) Push(sig *Signal) bool {
	select {
	case s.ch <- sig:
		return true
	default:
		signalDroppedTotal.Add(1)
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

// --- Backpressure metrics ---

var signalDroppedTotal atomic.Int64

// SignalDroppedTotal returns the total number of dropped signals.
func SignalDroppedTotal() int64 { return signalDroppedTotal.Load() }
