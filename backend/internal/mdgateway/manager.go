// Package mdgateway implements the market data gateway service.
//
// Adapter layout:
//
//	internal/mdgateway/adapter/mt4/   → raw mtapi gRPC client wrapper
//	internal/mdgateway/adapter/mt5/   → raw mtapi gRPC client wrapper
//
// This manager orchestrates connections, platform dispatch,
// circuit breaker protection, and tick fan-out to publisher + ClickHouse writer.
package mdgateway

import (
	"context"
	"fmt"
	"sync"

	"anttrader/internal/mt4client"
	"anttrader/internal/mt5client"

	"google.golang.org/grpc"
)

// TickHandler is called for each normalized Tick.
type TickHandler func(tick *Tick)

// Gateway wraps a single broker connection with tick streaming.
type Gateway interface {
	Platform() string
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Subscribe(ctx context.Context, symbols []string, handler TickHandler) error
	HealthCheck(ctx context.Context) error
	// Conn returns the underlying gRPC connection (may be nil before Connect).
	Conn() *grpc.ClientConn
	// SessionID returns the MT session token (empty before Connect).
	SessionID() string
	// BrokerID returns the broker UUID for this connection.
	BrokerID() string
}

// AccountConfig holds connection parameters for a broker account.
type AccountConfig struct {
	Broker     string
	Platform   string // "mt4" or "mt5"
	Login      string
	Password   string
	Server     string
	Host       string
	Port       string
	MtapiToken string
	UserID     string
	Status     string
	IsDisabled bool
}

// Config holds the md-gateway service configuration.
type Config struct {
	Accounts []AccountEntry
	Log      LogConfig
}

// AccountEntry maps a broker account from config.
type AccountEntry struct {
	UserID   string
	Broker   string
	Platform string
	Login    string
	Password string
	Server   string
	Host     string
	Port     string
	Symbols  []string
}

// LogConfig holds log settings.
type LogConfig struct {
	Level string
}

// Manager manages broker connections with circuit breaker protection.
type Manager struct {
	mu         sync.Mutex
	gateways   map[string]Gateway
	breakers   map[string]*CircuitBreaker // per-gateway circuit breaker
	normalizer *Normalizer
	metrics    *MDMetrics
}

// NewEmptyManager creates a Manager with no gateways (populated dynamically).
func NewEmptyManager() *Manager {
	return &Manager{
		gateways: make(map[string]Gateway),
		breakers: make(map[string]*CircuitBreaker),
	}
}

// SetNormalizer sets the canonical resolver for future gateway connections.
func (m *Manager) SetNormalizer(n *Normalizer) {
	m.normalizer = n
}

// SetMetrics attaches Prometheus metrics to the manager.
func (m *Manager) SetMetrics(metrics *MDMetrics) {
	m.metrics = metrics
}

// NewManager creates a Manager and instantiates platform adapters.
func NewManager(cfg Config) *Manager {
	m := &Manager{
		gateways: make(map[string]Gateway),
		breakers: make(map[string]*CircuitBreaker),
	}
	for _, entry := range cfg.Accounts {
		ac := AccountConfig{
			Broker:   entry.Broker,
			Platform: entry.Platform,
			Login:    entry.Login,
			Password: entry.Password,
			Server:   entry.Server,
			Host:     entry.Host,
			Port:     entry.Port,
		}
		key := entry.Broker + "-" + entry.Login
		gw := newGateway(ac, m.normalizer)
		m.gateways[key] = gw
		m.breakers[key] = NewCircuitBreaker(DefaultCircuitBreakerConfig())
	}
	return m
}

// AddGateway adds or replaces a gateway connection with circuit breaker.
func (m *Manager) AddGateway(cfg AccountConfig) {
	m.AddGatewayWithClient(cfg, nil, nil)
}

// AddGatewayWithClient adds a gateway with real MT clients.
func (m *Manager) AddGatewayWithClient(cfg AccountConfig, mt4c *mt4client.MT4Client, mt5c *mt5client.MT5Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := cfg.Broker + "-" + cfg.Login
	m.gateways[key] = newGatewayWithClient(cfg, m.normalizer, mt4c, mt5c)
	m.breakers[key] = NewCircuitBreaker(DefaultCircuitBreakerConfig())
}

// RemoveGateway removes a gateway connection.
func (m *Manager) RemoveGateway(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.gateways, key)
	delete(m.breakers, key)
}

// ConnectGateway connects a specific gateway with circuit breaker protection.
func (m *Manager) ConnectGateway(ctx context.Context, key string) error {
	m.mu.Lock()
	gw, ok := m.gateways[key]
	cb, cbOk := m.breakers[key]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("mdgateway: gateway %q not found", key)
	}

	if cbOk && !cb.Allow() {
		if m.metrics != nil {
			m.metrics.CircuitState.Set(1)
		}
		return fmt.Errorf("mdgateway: circuit open for %q", key)
	}

	if m.metrics != nil {
		m.metrics.GatewayConnect.WithLabelValues(gw.Platform(), "attempt").Inc()
	}

	if err := gw.Connect(ctx); err != nil {
		if cbOk {
			cb.RecordFailure()
		}
		if m.metrics != nil {
			m.metrics.GatewayConnect.WithLabelValues(gw.Platform(), "failure").Inc()
		}
		return err
	}

	if cbOk {
		cb.RecordSuccess()
	}
	if m.metrics != nil {
		m.metrics.CircuitState.Set(0)
		m.metrics.GatewayConnect.WithLabelValues(gw.Platform(), "success").Inc()
	}
	return nil
}

// HealthCheckGateway checks a specific gateway with breaker integration.
func (m *Manager) HealthCheckGateway(ctx context.Context, key string) error {
	m.mu.Lock()
	gw, ok := m.gateways[key]
	cb, cbOk := m.breakers[key]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("mdgateway: gateway %q not found", key)
	}

	if cbOk && !cb.Allow() {
		if m.metrics != nil {
			m.metrics.CircuitState.Set(1)
		}
		return fmt.Errorf("mdgateway: circuit open for %q", key)
	}

	if err := gw.HealthCheck(ctx); err != nil {
		if cbOk {
			cb.RecordFailure()
		}
		if m.metrics != nil {
			m.metrics.CircuitState.Set(1)
		}
		return err
	}

	if cbOk {
		cb.RecordSuccess()
	}
	if m.metrics != nil {
		m.metrics.CircuitState.Set(0)
	}
	return nil
}

func newGateway(cfg AccountConfig, normalizer *Normalizer) Gateway {
	return newGatewayWithClient(cfg, normalizer, nil, nil)
}

func newGatewayWithClient(cfg AccountConfig, normalizer *Normalizer, mt4c *mt4client.MT4Client, mt5c *mt5client.MT5Client) Gateway {
	switch cfg.Platform {
	case "mt4":
		return newMT4Gateway(cfg, normalizer, mt4c)
	case "mt5":
		return newMT5Gateway(cfg, normalizer, mt5c)
	default:
		panic(fmt.Sprintf("mdgateway: unknown platform %q", cfg.Platform))
	}
}

// Connections returns all managed gateways.
func (m *Manager) Connections() map[string]Gateway {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]Gateway, len(m.gateways))
	for k, v := range m.gateways {
		out[k] = v
	}
	return out
}

// CircuitState returns the circuit breaker state for a gateway.
func (m *Manager) CircuitState(key string) CircuitState {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cb, ok := m.breakers[key]; ok {
		return cb.State()
	}
	return CircuitClosed
}
