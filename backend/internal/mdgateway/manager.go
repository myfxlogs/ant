package mdgateway

import (
	"context"
	"fmt"
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
type TickHandler func(t *mdtick.Tick)

// ManagerDeps holds all dependencies for Manager.
type ManagerDeps struct {
	Normalizer  *Normalizer
	Quality     *Quality
	Dedup       *TickDedup
	Aggregator  *BarAggregator
	Publisher   *Publisher
	CHWriter    *CHWriter
	SpillWriter *SpillWriter
}

// Manager orchestrates connections, platform dispatch, circuit breaker
// protection, and tick fan-out to publisher + CH writer.
type Manager struct {
	normalizer  *Normalizer
	quality     *Quality
	dedup       *TickDedup
	aggregator  *BarAggregator
	publisher   *Publisher
	chWriter    *CHWriter
	spillWriter *SpillWriter
	breakers    map[string]*CircuitBreaker

	mu       sync.RWMutex
	gateways map[string]Gateway
}

// NewManager constructs and returns a Manager.
func NewManager(deps ManagerDeps) *Manager {
	return &Manager{
		normalizer:  deps.Normalizer,
		quality:     deps.Quality,
		dedup:       deps.Dedup,
		aggregator:  deps.Aggregator,
		publisher:   deps.Publisher,
		chWriter:    deps.CHWriter,
		spillWriter: deps.SpillWriter,
		breakers:    make(map[string]*CircuitBreaker),
		gateways:    make(map[string]Gateway),
	}
}

// AddGateway registers and starts a new gateway.
func (m *Manager) AddGateway(ctx context.Context, gw Gateway, syms []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.gateways[gw.AccountID()]; exists {
		return fmt.Errorf("mdgateway: account %s already registered", gw.AccountID())
	}
	m.gateways[gw.AccountID()] = gw
	return nil
}

// RemoveGateway closes and removes a registered gateway.
func (m *Manager) RemoveGateway(ctx context.Context, accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	gw, ok := m.gateways[accountID]
	if !ok {
		return nil
	}
	delete(m.gateways, accountID)
	return gw.Disconnect(ctx)
}

// HandleTick processes a tick through the full pipeline:
// normalizer → quality → dedup → aggregator → publisher → chWriter.
func (m *Manager) HandleTick(t *mdtick.Tick) {
	// 1. canonical
	t.Canonical = m.normalizer.Resolve(t.Broker, t.SymbolRaw)

	// 2. quality
	qr := m.quality.Check(t)
	if qr.Dropped {
		return
	}

	// 3. dedup
	if m.dedup.Seen(t) {
		return
	}

	// 4. aggregate bars
	var bars []*mdtick.Bar
	m.aggregator.AddTick(t, func(b *mdtick.Bar) { bars = append(bars, b) })

	// 5. publish tick
	_ = m.publisher.PublishTick(t)

	// 6. publish bars
	for _, b := range bars {
		_ = m.publisher.PublishBar(b)
	}

	// 7. write to ClickHouse (async via channel)
	m.chWriter.EnqueueTick(t)
	for _, b := range bars {
		m.chWriter.EnqueueBar(b)
	}
}

// Health returns health summaries for all registered gateways.
func (m *Manager) Health() []AccountHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []AccountHealth
	for _, gw := range m.gateways {
		result = append(result, AccountHealth{
			AccountID: gw.AccountID(),
			Platform:  gw.Platform(),
		})
	}
	return result
}

// AccountHealth summarizes a single account state.
type AccountHealth struct {
	AccountID    string
	Broker       string
	Platform     string
	State        string
	LastTickAt   int64
	CircuitState string
	TickRate1m   float64
}
