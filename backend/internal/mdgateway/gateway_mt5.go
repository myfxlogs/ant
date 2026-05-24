// V1-LEGACY: will be replaced by M7.1-7.4 cards. Do not extend; new code goes alongside.
// Package mdgateway — MT5 gateway adapter.
package mdgateway

import (
	"context"

	"anttrader/internal/mdgateway/adapter/mdtick"
	mt5adapter "anttrader/internal/mdgateway/adapter/mt5"
	"anttrader/internal/mt5client"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// mt5Gateway wraps the adapter/mt5 package to implement Gateway.
type mt5Gateway struct {
	inner *mt5adapter.Gateway
}

func newMT5Gateway(cfg AccountConfig, normalizer *Normalizer, mt5c *mt5client.MT5Client) Gateway {
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
	return &mt5Gateway{inner: mt5adapter.New(ac, an, zap.NewNop(), mt5c)}
}

func (g *mt5Gateway) Platform() string                { return g.inner.Platform() }
func (g *mt5Gateway) Conn() *grpc.ClientConn           { return nil }
func (g *mt5Gateway) SessionID() string                { return g.inner.SessionID() }
func (g *mt5Gateway) BrokerID() string                 { return g.inner.BrokerID() }
func (g *mt5Gateway) Connect(ctx context.Context) error    { return g.inner.Connect(ctx) }
func (g *mt5Gateway) Disconnect(ctx context.Context) error { return g.inner.Disconnect(ctx) }
func (g *mt5Gateway) HealthCheck(ctx context.Context) error { return g.inner.HealthCheck(ctx) }

func (g *mt5Gateway) Subscribe(ctx context.Context, symbols []string, handler TickHandler) error {
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
