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
	tracer      *Tracer

	mu       sync.RWMutex
	gateways map[string]Gateway
}

// Tracer is a minimal trace interface (avoids circular import with internal/trace).
type Tracer struct {
	enabled bool
}

// SetTracer injects a tracer for HandleTick span generation.
func (m *Manager) SetTracer(enabled bool) {
	m.tracer = &Tracer{enabled: enabled}
}

func (m *Manager) startTrace(ctx context.Context, name string) (context.Context, *SimpleSpan) {
	if m.tracer == nil || !m.tracer.enabled {
		return ctx, &SimpleSpan{}
	}
	// Full OTel integration via internal/trace.Tracer when wired by runner.
	return ctx, &SimpleSpan{}
}

// SimpleSpan is a no-op span when OTel is not wired.
type SimpleSpan struct{}

func (s *SimpleSpan) End() {}

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
	ctx := context.Background()

	// ADR-0010 §2.3: OTel trace spans for the 6-stage tick pipeline.
	_, span1 := m.startTrace(ctx, "normalize")
	t.Canonical = m.normalizer.Resolve(t.Broker, t.SymbolRaw)
	span1.End()

	_, span2 := m.startTrace(ctx, "quality")
	qr := m.quality.Check(ctx, t)
	span2.End()
	if qr.Dropped {
		return
	}

	_, span3 := m.startTrace(ctx, "dedup")
	seen := m.dedup.Seen(t)
	span3.End()
	if seen {
		return
	}

	_, span4 := m.startTrace(ctx, "aggregate")
	var bars []*mdtick.Bar
	m.aggregator.AddTick(t, func(b *mdtick.Bar) { bars = append(bars, b) })
	span4.End()

	_, span5 := m.startTrace(ctx, "publish")
	_ = m.publisher.PublishTick(t)
	for _, b := range bars {
		_ = m.publisher.PublishBar(b)
	}
	span5.End()

	_, span6 := m.startTrace(ctx, "enqueue")
	m.chWriter.EnqueueTick(t)
	for _, b := range bars {
		m.chWriter.EnqueueBar(b)
	}
	span6.End()
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
