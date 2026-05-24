// Package mdgateway — Gateway interface and Manager types for v2.
package mdgateway

import (
	"context"
	"sync"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// Gateway is the contract every MT adapter must implement.
type Gateway interface {
	Platform() string
	AccountID() string
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Subscribe(ctx context.Context, symbols []string, handler TickHandler) error
	HealthCheck(ctx context.Context) error
	SessionID() string
}

// TickHandler is called by the adapter for each received tick.
// Must not block; the handler dispatches to Manager.HandleTick synchronously.
type TickHandler func(t *mdtick.Tick)

// Manager orchestrates connections, platform dispatch, circuit breaker
// protection, and tick fan-out to publisher + ClickHouse writer.
type Manager struct {
	normalizer  *Normalizer
	quality     *Quality
	dedup       *TickDedup
	aggregator  *BarAggregator
	publisher   *Publisher
	chWriter    *CHWriter
	spillWriter *SpillWriter
	breakers    map[string]*CircuitBreaker // brokerKey → breaker

	mu       sync.RWMutex
	gateways map[string]Gateway // accountID → Gateway
}

// AccountHealth summarizes a single account's connection state.
type AccountHealth struct {
	AccountID    string
	Broker       string
	Platform     string
	State        string // disconnected, connecting, connected, degraded
	LastTickAt   int64
	CircuitState string // closed, open, half_open
	TickRate1m   float64
}
