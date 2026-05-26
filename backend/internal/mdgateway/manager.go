package mdgateway

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	anttrace "anttrader/internal/trace"
	"anttrader/internal/mdgateway/adapter/mdtick"
)

type Gateway interface {
	Platform() string
	AccountID() string
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Subscribe(ctx context.Context, symbols []string, handler mdtick.TickHandler) error
	SubscribeProfit(ctx context.Context, handler mdtick.ProfitHandler) error
	SubscribeOrderUpdate(ctx context.Context, handler mdtick.OrderUpdateHandler) error
	HealthCheck(ctx context.Context) error
	SessionID() string
}

type ManagerDeps struct {
	Normalizer       *Normalizer
	Quality          *Quality
	Dedup            *TickDedup
	Aggregator       *BarAggregator
	Publisher        *Publisher
	CHWriter         *CHWriter
	SpillWriter      *SpillWriter
	MarketState      *MarketStateTracker  // M10-BASE-F1
	StuffingDetector *StuffingDetector    // M10-BASE-F4
	Log              *zap.Logger
}

type Manager struct {
	normalizer       *Normalizer
	quality          *Quality
	dedup            *TickDedup
	aggregator       *BarAggregator
	publisher        *Publisher
	chWriter         *CHWriter
	spillWriter      *SpillWriter
	marketState      *MarketStateTracker
	stuffingDetector *StuffingDetector
	breakers         map[string]*CircuitBreaker
	otelTracer       *anttrace.Tracer // L-2: real OTel tracer, nil = no-op
	log              *zap.Logger

	mu         sync.RWMutex
	gateways   map[string]Gateway
	lastTickAt map[string]int64 // accountID -> unix ms
	baseCtx    context.Context
}

// SetOTelTracer injects the OTel tracer for HandleTick span generation.
// Pass nil to disable tracing. L-2: replaces the old SetTracer(bool) stub.
func (m *Manager) SetOTelTracer(t *anttrace.Tracer) {
	m.otelTracer = t
}

// SetBaseContext sets the base context for tick processing. If not set,
// context.Background() is used (acceptable for the background pipeline).
func (m *Manager) SetBaseContext(ctx context.Context) {
	m.baseCtx = ctx
}

func (m *Manager) baseContext() context.Context {
	if m.baseCtx != nil {
		return m.baseCtx
	}
	return context.Background()
}

func (m *Manager) startTrace(ctx context.Context, name string) (context.Context, *anttrace.Span) {
	if m.otelTracer == nil {
		return ctx, &anttrace.Span{}
	}
	return m.otelTracer.StartSpan(ctx, name)
}

func NewManager(deps ManagerDeps) *Manager {
	return &Manager{
		normalizer:       deps.Normalizer,
		quality:          deps.Quality,
		dedup:            deps.Dedup,
		aggregator:       deps.Aggregator,
		publisher:        deps.Publisher,
		chWriter:         deps.CHWriter,
		spillWriter:      deps.SpillWriter,
		marketState:      deps.MarketState,
		stuffingDetector: deps.StuffingDetector,
		breakers:         make(map[string]*CircuitBreaker),
		gateways:         make(map[string]Gateway),
		lastTickAt:       make(map[string]int64),
		log:              deps.Log,
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
	ctx := m.baseContext()

	// M10-BASE-F4: quote stuffing check before full pipeline.
	if m.stuffingDetector != nil {
		if m.stuffingDetector.IsPaused(t.Broker, t.Canonical) {
			return // symbol paused due to stuffing
		}
	}

	// ADR-0010 §2.3: OTel trace spans for the 6-stage tick pipeline.
	_, span1 := m.startTrace(ctx, "normalize")
	t.Canonical = m.normalizer.Resolve(ctx, t.Broker, t.SymbolRaw)
	span1.End()

	_, span2 := m.startTrace(ctx, "quality")
	qr := m.quality.Check(ctx, t)
	span2.End()
	if qr.Dropped {
		return
	}

	// M10-BASE-F4: track tick rate for stuffing detection.
	if m.stuffingDetector != nil {
		if stuffed, _ := m.stuffingDetector.Observe(t.Broker, t.Canonical); stuffed {
			return // just stuffed — drop this tick
		}
	}

	// M10-BASE-F5: spread anomaly detection.
	if qr.SpreadBps > 0 && m.quality != nil {
		key := t.Broker + ":" + t.Canonical
		m.quality.trackSpread(key, qr.SpreadBps)
		z := m.quality.SpreadZscore(key, qr.SpreadBps)
		if z > m.quality.cfg.MaxSpreadZscore {
			RecordSpreadAnomaly()
		}
	}

	_, span3 := m.startTrace(ctx, "dedup")
	seen := m.dedup.Seen(t)
	span3.End()
	if seen {
		return
	}

	// Record last tick time for staleness detection.
	m.mu.Lock()
	m.lastTickAt[t.AccountID] = Clk.Now().UnixMilli()
	m.mu.Unlock()

	// M10-BASE-F1: update market state for tradability.
	if m.marketState != nil {
		m.marketState.Update(t)
	}

	_, span4 := m.startTrace(ctx, "aggregate")
	var bars []*mdtick.Bar
	m.aggregator.AddTick(t, func(b *mdtick.Bar) { bars = append(bars, b) })
	span4.End()

	_, span5 := m.startTrace(ctx, "publish")
	if err := m.publisher.PublishTick(t); err != nil && m.log != nil {
		m.log.Warn("mdgateway: PublishTick failed", zap.String("account", t.AccountID), zap.String("symbol", t.Canonical), zap.Error(err))
	}
	for _, b := range bars {
		if err := m.publisher.PublishBar(b); err != nil && m.log != nil {
			m.log.Warn("mdgateway: PublishBar failed", zap.String("account", b.AccountID), zap.String("symbol", b.Canonical), zap.Error(err))
		}
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
	now := Clk.Now().UnixMilli()
	var result []AccountHealth
	for _, gw := range m.gateways {
		lastAt := m.lastTickAt[gw.AccountID()]
		state := "healthy"
		if lastAt == 0 {
			state = "no_data"
		} else if now-lastAt > 15*60*1000 {
			state = "dead"
		} else if now-lastAt > 5*60*1000 {
			state = "stale"
		}
		result = append(result, AccountHealth{
			AccountID:  gw.AccountID(),
			Platform:   gw.Platform(),
			State:      state,
			LastTickAt: lastAt,
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
