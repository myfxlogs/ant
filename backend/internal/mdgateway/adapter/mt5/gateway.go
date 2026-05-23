// Package mt5 provides the MT5 gateway adapter for mdgateway.
// It bridges the existing anttrader/internal/mt5client infrastructure
// to mdgateway, converting proto-based quotes to local Tick types.
//
// This package does NOT import mdgateway to avoid circular imports.
// The calling package (mdgateway) performs the final adaptation.
package mt5

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	pb "anttrader/mt5"
	"anttrader/internal/mt5client"

	"go.uber.org/zap"
)

// AccountConfig mirrors mdgateway.AccountConfig to avoid circular imports.
type AccountConfig struct {
	Broker   string
	Login    string
	Password string
	Server   string
	Host     string
	Port     string
}

// Money is a monetary value with string-preserved decimal precision.
type Money struct {
	Value string
}

// GetValue returns the string value.
func (m *Money) GetValue() string {
	if m == nil {
		return ""
	}
	return m.Value
}

// Tick is a normalized market data tick.
type Tick struct {
	UserID        string
	Broker        string
	Symbol        string
	Canonical     string
	TsUnixMs      int64
	ArrivedUnixMs int64
	Bid           *Money
	Ask           *Money
	BidVolume     float64
	AskVolume     float64
}

// TickHandler is called for each normalized Tick.
type TickHandler func(tick *Tick)

// CanonicalResolver resolves (broker, symbol_raw) -> canonical name.
type CanonicalResolver interface {
	Resolve(brokerID, symbolRaw string) string
}

// Normalizer converts broker-specific quote types to Tick.
type Normalizer struct {
	Resolver CanonicalResolver
}

// Tick creates a Tick with common fields filled.
func (n *Normalizer) Tick(userID, broker, symbol string, tsMs int64, bid, ask string) *Tick {
	canon := symbol
	if n.Resolver != nil {
		canon = n.Resolver.Resolve(broker, symbol)
	}
	return &Tick{
		UserID:    userID,
		Broker:    broker,
		Symbol:    symbol,
		Canonical: canon,
		TsUnixMs:  tsMs,
		Bid:       &Money{Value: bid},
		Ask:       &Money{Value: ask},
	}
}

// Gateway is the MT5 gateway implementation.
type Gateway struct {
	cfg        AccountConfig
	normalizer *Normalizer
	log        *zap.Logger

	mu     sync.Mutex
	conn   *mt5client.MT5Connection
	client *mt5client.MT5Client
	cancel context.CancelFunc
}

// New creates an MT5 gateway adapter.
func New(cfg AccountConfig, normalizer *Normalizer, log *zap.Logger) *Gateway {
	return &Gateway{
		cfg:        cfg,
		normalizer: normalizer,
		log:        log,
	}
}

// Platform returns "mt5".
func (g *Gateway) Platform() string { return "mt5" }

// SessionID returns the MT5 session id.
func (g *Gateway) SessionID() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn != nil {
		return g.conn.GetID()
	}
	return ""
}

// BrokerID returns the broker identifier.
func (g *Gateway) BrokerID() string { return g.cfg.Broker }

// Connect establishes an MT5 connection via the mt5client pool.
func (g *Gateway) Connect(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.conn != nil {
		if g.conn.IsConnected() {
			return nil
		}
	}

	login, err := strconv.ParseUint(g.cfg.Login, 10, 64)
	if err != nil {
		return fmt.Errorf("mt5: invalid login %q: %w", g.cfg.Login, err)
	}
	port, err := strconv.ParseInt(g.cfg.Port, 10, 32)
	if err != nil {
		return fmt.Errorf("mt5: invalid port %q: %w", g.cfg.Port, err)
	}

	conn, err := g.client.Connect(ctx, login, g.cfg.Password, g.cfg.Host, int32(port))
	if err != nil {
		return fmt.Errorf("mt5: connect failed: %w", err)
	}
	g.conn = conn
	return nil
}

// Disconnect removes the MT5 connection.
func (g *Gateway) Disconnect(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.conn != nil {
		g.conn.MarkDisconnected()
		g.conn = nil
	}
	return nil
}

// Subscribe starts the quote stream and forwards normalized ticks to handler.
func (g *Gateway) Subscribe(ctx context.Context, symbols []string, handler TickHandler) error {
	g.mu.Lock()
	if g.conn == nil {
		g.mu.Unlock()
		return fmt.Errorf("mt5: not connected")
	}
	conn := g.conn
	normalizer := g.normalizer
	g.mu.Unlock()

	for _, sym := range symbols {
		if err := conn.Subscribe(ctx, sym, 0); err != nil {
			g.log.Warn("mt5: subscribe failed",
				zap.String("symbol", sym),
				zap.Error(err),
			)
		}
	}

	if err := conn.SubscribeQuoteStream(ctx); err != nil {
		return fmt.Errorf("mt5: quote stream failed: %w", err)
	}

	quoteCh := conn.GetQuoteChannel()

	ctx, cancel := context.WithCancel(ctx)
	g.cancel = cancel

	go func() {
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case quote, ok := <-quoteCh:
				if !ok {
					return
				}
				tick := g.normalize(quote, conn.GetAccountID(), normalizer)
				handler(tick)
			}
		}
	}()

	return nil
}

// HealthCheck verifies the connection is alive.
func (g *Gateway) HealthCheck(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn == nil || !g.conn.IsConnected() {
		return fmt.Errorf("mt5: not connected")
	}
	return nil
}

// normalize converts a protobuf Quote to a local Tick.
func (g *Gateway) normalize(q *pb.Quote, userID string, normalizer *Normalizer) *Tick {
	if normalizer != nil {
		return normalizer.Tick(userID, g.cfg.Broker, q.Symbol,
			time.Now().UnixMilli(),
			fmt.Sprintf("%.6f", q.Bid), fmt.Sprintf("%.6f", q.Ask))
	}
	return &Tick{
		UserID:        userID,
		Broker:        g.cfg.Broker,
		Symbol:        q.Symbol,
		Canonical:     q.Symbol,
		TsUnixMs:      time.Now().UnixMilli(),
		ArrivedUnixMs: time.Now().UnixMilli(),
		Bid:           &Money{Value: fmt.Sprintf("%.6f", q.Bid)},
		Ask:           &Money{Value: fmt.Sprintf("%.6f", q.Ask)},
	}
}
