// V1-LEGACY: will be replaced by M7.1-7.4 cards. Do not extend; new code goes alongside.
// Package mdgateway — MT4 gateway adapter.
package mdgateway

import (
	"context"

	"anttrader/internal/mdgateway/adapter/mdtick"
	mt4adapter "anttrader/internal/mdgateway/adapter/mt4"
	"anttrader/internal/mt4client"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// mt4Gateway wraps the adapter/mt4 package to implement Gateway.
type mt4Gateway struct {
	inner *mt4adapter.Gateway
}

func newMT4Gateway(cfg AccountConfig, normalizer *Normalizer, mt4c *mt4client.MT4Client) Gateway {
	ac := mdtick.AccountConfig{
		Broker:   cfg.Broker,
		Login:    cfg.Login,
		Password: cfg.Password,
		Server:   cfg.Server,
		Host:     cfg.Host,
		Port:     cfg.Port,
	}
	var an *mdtick.Normalizer
	if normalizer != nil {
		an = &mdtick.Normalizer{
			Resolver: &resolverBridge{inner: normalizer.resolver},
		}
	}
	return &mt4Gateway{inner: mt4adapter.New(ac, an, zap.NewNop(), mt4c)}
}

// resolverBridge adapts mdgateway.CanonicalResolver to mdtick.CanonicalResolver.
type resolverBridge struct {
	inner CanonicalResolver
}

func (r *resolverBridge) Resolve(brokerID, symbolRaw string) string {
	if r.inner != nil {
		return r.inner.Resolve(brokerID, symbolRaw)
	}
	return symbolRaw
}

func (g *mt4Gateway) Platform() string                { return g.inner.Platform() }
func (g *mt4Gateway) Conn() *grpc.ClientConn           { return nil }
func (g *mt4Gateway) SessionID() string                { return g.inner.SessionID() }
func (g *mt4Gateway) BrokerID() string                 { return g.inner.BrokerID() }
func (g *mt4Gateway) Connect(ctx context.Context) error    { return g.inner.Connect(ctx) }
func (g *mt4Gateway) Disconnect(ctx context.Context) error { return g.inner.Disconnect(ctx) }
func (g *mt4Gateway) HealthCheck(ctx context.Context) error { return g.inner.HealthCheck(ctx) }

func (g *mt4Gateway) Subscribe(ctx context.Context, symbols []string, handler TickHandler) error {
	return g.inner.Subscribe(ctx, symbols, func(t *mdtick.Tick) {
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
