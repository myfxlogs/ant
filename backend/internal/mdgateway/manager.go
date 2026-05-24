package mdgateway

import (
	"context"
	"fmt"
	"sync"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

type Gateway interface {
	Platform() string
	AccountID() string
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Subscribe(ctx context.Context, symbols []string, handler TickHandler) error
	HealthCheck(ctx context.Context) error
	SessionID() string
}

type TickHandler func(t *mdtick.Tick)

type ManagerDeps struct {
	Normalizer  *Normalizer
	Quality     *Quality
	Dedup       *TickDedup
	Aggregator  *BarAggregator
	Publisher   *Publisher
	CHWriter    *CHWriter
	SpillWriter *SpillWriter
}

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

func (m *Manager) AddGateway(ctx context.Context, gw Gateway, syms []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.gateways[gw.AccountID()]; exists {
		return fmt.Errorf("mdgateway: account %s already registered", gw.AccountID())
	}
	m.gateways[gw.AccountID()] = gw
	return nil
}

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

func (m *Manager) HandleTick(t *mdtick.Tick) {
	// ADR-0010 §2.3: trace span for the full tick processing pipeline.
	// ctx, span := m.tracer.StartSpan(context.Background(), "HandleTick")
	// defer span.End()
	t.Canonical = m.normalizer.Resolve(t.Broker, t.SymbolRaw)

	qr := m.quality.Check(t)
	if qr.Dropped {
		return
	}

	if m.dedup.Seen(t) {
		return
	}

	var bars []*mdtick.Bar
	m.aggregator.AddTick(t, func(b *mdtick.Bar) { bars = append(bars, b) })

	_ = m.publisher.PublishTick(t)

	for _, b := range bars {
		_ = m.publisher.PublishBar(b)
	}

	m.chWriter.EnqueueTick(t)
	for _, b := range bars {
		m.chWriter.EnqueueBar(b)
	}
}

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

type AccountHealth struct {
	AccountID    string
	Broker       string
	Platform     string
	State        string
	LastTickAt   int64
	CircuitState string
	TickRate1m   float64
}
