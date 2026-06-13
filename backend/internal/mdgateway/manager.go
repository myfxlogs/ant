package mdgateway

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	// SetStatusCallback registers a callback that fires when the gateway's
	// connection state changes (connected → reconnecting → disconnected).
	// The callback is concurrency-safe and must not block the caller.
	SetStatusCallback(func(status, message string))
}

type ManagerDeps struct {
	Normalizer       *Normalizer
	Quality          *Quality
	Dedup            *TickDedup
	Aggregator       *BarAggregator
	Publisher        *Publisher
	PgWriter         *PgWriter
	MarketState      *MarketStateTracker
	StuffingDetector *StuffingDetector
	OnBar            func(*mdtick.Bar)
	Log              *zap.Logger
}

type Manager struct {
	normalizer       *Normalizer
	quality          *Quality
	dedup            *TickDedup
	aggregator       *BarAggregator
	publisher        *Publisher
	pgWriter         *PgWriter
	marketState      *MarketStateTracker
	stuffingDetector *StuffingDetector
	onBar            func(*mdtick.Bar)
	breakers         map[string]*CircuitBreaker
	otelTracer       *anttrace.Tracer
	log              *zap.Logger

	mu            sync.RWMutex
	gateways      map[string]Gateway
	lastTickAt    map[string]int64
	disconnecting map[string]bool
	baseCtx       context.Context
}

func NewManager(deps ManagerDeps) *Manager {
	return &Manager{
		normalizer:       deps.Normalizer,
		quality:          deps.Quality,
		dedup:            deps.Dedup,
		aggregator:       deps.Aggregator,
		publisher:        deps.Publisher,
		pgWriter:         deps.PgWriter,
		marketState:      deps.MarketState,
		stuffingDetector: deps.StuffingDetector,
		onBar:            deps.OnBar,
		breakers:         make(map[string]*CircuitBreaker),
		gateways:         make(map[string]Gateway),
		lastTickAt:       make(map[string]int64),
		disconnecting:    make(map[string]bool),
		log:              deps.Log,
	}
}

// SetOTelTracer injects the OTel tracer for HandleTick span generation.
func (m *Manager) SetOTelTracer(t *anttrace.Tracer) { m.otelTracer = t }

// SetBaseContext sets the base context for tick processing.
func (m *Manager) SetBaseContext(ctx context.Context) { m.baseCtx = ctx }

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

// --- Gateway management ---

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
	gw, ok := m.gateways[accountID]
	if !ok {
		m.mu.Unlock()
		return nil
	}
	delete(m.gateways, accountID)
	m.mu.Unlock()
	return gw.Disconnect(ctx)
}

// MarkDisconnecting records that an account is being disconnected by the
// healthMonitor to prevent the NATS subscriber from racing to reconnect it.
func (m *Manager) MarkDisconnecting(accountID string) {
	m.mu.Lock()
	m.disconnecting[accountID] = true
	m.mu.Unlock()
}

// UnmarkDisconnecting removes the disconnecting flag for an account.
func (m *Manager) UnmarkDisconnecting(accountID string) {
	m.mu.Lock()
	delete(m.disconnecting, accountID)
	m.mu.Unlock()
}

// IsDisconnecting returns true if the account is currently being disconnected.
func (m *Manager) IsDisconnecting(accountID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.disconnecting[accountID]
}

// GetGateway returns the registered gateway for accountID, or nil if not found.
func (m *Manager) GetGateway(accountID string) Gateway {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gateways[accountID]
}

// reconnectable is the interface gateways implement to guard against
// recvLoop races during managed reconnection.
type reconnectable interface {
	SetReconnecting(bool)
}

// ReconnectGateway disconnects and reconnects a gateway in-place.
// Used by healthMonitor to attempt recovery of dead connections before removing them.
// Sets a reconnecting flag to prevent recvLoop goroutines from racing a second Connect.
func (m *Manager) ReconnectGateway(ctx context.Context, accountID string) error {
	m.mu.RLock()
	gw, ok := m.gateways[accountID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("mdgateway: gateway %s not found", accountID)
	}
	// Signal recvLoop goroutines to wait instead of racing.
	if rc, ok := gw.(reconnectable); ok {
		rc.SetReconnecting(true)
		defer rc.SetReconnecting(false)
	}
	// Graceful disconnect first.
	if err := gw.Disconnect(ctx); err != nil {
		m.log.Warn("mdgateway: disconnect during reconnect",
			zap.String("account", accountID), zap.Error(err))
	}
	// Re-connect with fresh session.
	if err := gw.Connect(ctx); err != nil {
		return fmt.Errorf("mdgateway: reconnect %s: %w", accountID, err)
	}
	return nil
}

// ResetLastTickAt resets the last-tick timestamp for an account after a
// successful reconnection so the health monitor doesn't immediately flag it again.
func (m *Manager) ResetLastTickAt(accountID string) {
	m.mu.Lock()
	m.lastTickAt[accountID] = time.Now().Unix()
	m.mu.Unlock()
}
