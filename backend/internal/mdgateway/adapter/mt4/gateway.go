// Package mt4 provides the MT4 gateway adapter for mdgateway.
// It bridges the existing anttrader/internal/mt4client infrastructure
// to mdgateway, converting proto-based quotes to local Tick types.
//
// This package does NOT import mdgateway to avoid circular imports.
// The calling package (mdgateway) performs the final adaptation.
package mt4

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	pb "anttrader/mt4"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/mt4client"

	"go.uber.org/zap"
)

// Gateway is the MT4 gateway implementation.
type Gateway struct {
	cfg        mdtick.AccountConfig
	normalizer *mdtick.Normalizer
	log        *zap.Logger

	mu     sync.Mutex
	conn   *mt4client.MT4Connection
	client *mt4client.MT4Client
	cancel context.CancelFunc
}

// New creates an MT4 gateway adapter.
func New(cfg mdtick.AccountConfig, normalizer *mdtick.Normalizer, log *zap.Logger, client *mt4client.MT4Client) *Gateway {
	return &Gateway{
		cfg:        cfg,
		normalizer: normalizer,
		log:        log,
		client:     client,
	}
}

// Platform returns "mt4".
func (g *Gateway) Platform() string { return "mt4" }

// SessionID returns the MT4 session token.
func (g *Gateway) SessionID() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn != nil {
		return g.conn.GetToken()
	}
	return ""
}

// BrokerID returns the broker identifier.
func (g *Gateway) BrokerID() string { return g.cfg.Broker }

// Connect establishes an MT4 connection via the mt4client pool.
func (g *Gateway) Connect(ctx context.Context) error {
	if g.client == nil {
		return fmt.Errorf("mt4: client not configured")
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.conn != nil {
		if g.conn.IsConnected() {
			return nil
		}
	}

	login, err := strconv.ParseInt(g.cfg.Login, 10, 32)
	if err != nil {
		return fmt.Errorf("mt4: invalid login %q: %w", g.cfg.Login, err)
	}
	port, err := strconv.ParseInt(g.cfg.Port, 10, 32)
	if err != nil {
		return fmt.Errorf("mt4: invalid port %q: %w", g.cfg.Port, err)
	}

	conn, err := g.client.Connect(ctx, int32(login), g.cfg.Password, g.cfg.Host, int32(port))
	if err != nil {
		return fmt.Errorf("mt4: connect failed: %w", err)
	}
	g.conn = conn
	return nil
}

// Disconnect removes the MT4 connection.
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
func (g *Gateway) Subscribe(ctx context.Context, symbols []string, handler mdtick.TickHandler) error {
	g.mu.Lock()
	if g.conn == nil {
		g.mu.Unlock()
		return fmt.Errorf("mt4: not connected")
	}
	conn := g.conn
	normalizer := g.normalizer
	g.mu.Unlock()

	for _, sym := range symbols {
		if err := conn.Subscribe(ctx, sym); err != nil {
			g.log.Warn("mt4: subscribe failed",
				zap.String("symbol", sym),
				zap.Error(err),
			)
		}
	}

	if err := conn.SubscribeQuoteStream(ctx); err != nil {
		return fmt.Errorf("mt4: quote stream failed: %w", err)
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
		return fmt.Errorf("mt4: not connected")
	}
	return nil
}

// normalize converts a protobuf QuoteEventArgs to a local Tick.
func (g *Gateway) normalize(q *pb.QuoteEventArgs, userID string, normalizer *mdtick.Normalizer) *mdtick.Tick {
	if normalizer != nil {
		return normalizer.Tick(userID, g.cfg.Broker, q.Symbol,
			time.Now().UnixMilli(),
			fmt.Sprintf("%.6f", q.Bid), fmt.Sprintf("%.6f", q.Ask))
	}
	return &mdtick.Tick{
		UserID:        userID,
		Broker:        g.cfg.Broker,
		Symbol:        q.Symbol,
		Canonical:     q.Symbol,
		TsUnixMs:      time.Now().UnixMilli(),
		ArrivedUnixMs: time.Now().UnixMilli(),
		Bid:           &mdtick.Money{Value: fmt.Sprintf("%.6f", q.Bid)},
		Ask:           &mdtick.Money{Value: fmt.Sprintf("%.6f", q.Ask)},
	}
}
