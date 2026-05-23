// Package mdgateway — MT5 gateway adapter.
package mdgateway

import (
	"context"

	mt5adapter "anttrader/internal/mdgateway/adapter/mt5"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// mt5Gateway wraps the adapter/mt5 package to implement Gateway.
type mt5Gateway struct {
	inner *mt5adapter.Gateway
}

func newMT5Gateway(cfg AccountConfig, normalizer *Normalizer) Gateway {
	ac := mt5adapter.AccountConfig{
		Broker:   cfg.Broker,
		Login:    cfg.Login,
		Password: cfg.Password,
		Server:   cfg.Server,
		Host:     cfg.Host,
		Port:     cfg.Port,
	}
	var an *mt5adapter.Normalizer
	if normalizer != nil {
		an = &mt5adapter.Normalizer{
			Resolver: &resolverBridgeMT5{inner: normalizer.resolver},
		}
	}
	return &mt5Gateway{inner: mt5adapter.New(ac, an, zap.NewNop())}
}

// resolverBridgeMT5 adapts mdgateway.CanonicalResolver to mt5adapter.CanonicalResolver.
type resolverBridgeMT5 struct {
	inner CanonicalResolver
}

func (r *resolverBridgeMT5) Resolve(brokerID, symbolRaw string) string {
	if r.inner != nil {
		return r.inner.Resolve(brokerID, symbolRaw)
	}
	return symbolRaw
}

func (g *mt5Gateway) Platform() string                { return g.inner.Platform() }
func (g *mt5Gateway) Conn() *grpc.ClientConn           { return nil }
func (g *mt5Gateway) SessionID() string                { return g.inner.SessionID() }
func (g *mt5Gateway) BrokerID() string                 { return g.inner.BrokerID() }
func (g *mt5Gateway) Connect(ctx context.Context) error    { return g.inner.Connect(ctx) }
func (g *mt5Gateway) Disconnect(ctx context.Context) error { return g.inner.Disconnect(ctx) }
func (g *mt5Gateway) HealthCheck(ctx context.Context) error { return g.inner.HealthCheck(ctx) }

func (g *mt5Gateway) Subscribe(ctx context.Context, symbols []string, handler TickHandler) error {
	return g.inner.Subscribe(ctx, symbols, func(t *mt5adapter.Tick) {
		handler(&Tick{
			UserID:        t.UserID,
			Broker:        t.Broker,
			Symbol:        t.Symbol,
			Canonical:     t.Canonical,
			TsUnixMs:      t.TsUnixMs,
			ArrivedUnixMs: t.ArrivedUnixMs,
			Bid:           &Money{Value: t.Bid.GetValue()},
			Ask:           &Money{Value: t.Ask.GetValue()},
			BidVolume:     t.BidVolume,
			AskVolume:     t.AskVolume,
		})
	})
}
