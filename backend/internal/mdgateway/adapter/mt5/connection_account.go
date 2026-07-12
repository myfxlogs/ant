package mt5

import (
	"context"
	"fmt"
	"time"

	pb "alphaforge/mt5"
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/metadata"
)

// FetchAccountInfo calls AccountSummary and returns basic account details.
func (g *Gateway) FetchAccountInfo(ctx context.Context) (*mdtick.MTAccountInfo, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()

	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5: not connected")
	}

	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	asCtx := metadata.NewOutgoingContext(ctx, md)
	resp, err := client.AccountSummary(asCtx, &pb.AccountSummaryRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt5 AccountSummary: %w", err)
	}
	if resp.GetResult() == nil {
		g.log.Warn("mt5 AccountSummary: result nil, assuming investor (read-only) account")
		return &mdtick.MTAccountInfo{IsInvestor: true}, nil
	}

	s := resp.GetResult()
	return &mdtick.MTAccountInfo{
		Balance:    decimal.NewFromFloat(s.GetBalance()),
		Credit:     decimal.NewFromFloat(s.GetCredit()),
		Equity:     decimal.NewFromFloat(s.GetEquity()),
		Margin:     decimal.NewFromFloat(s.GetMargin()),
		FreeMargin: decimal.NewFromFloat(s.GetFreeMargin()),
		Leverage:   int32(s.GetLeverage()),
		Currency:   s.GetCurrency(),
		IsInvestor: s.GetIsInvestor(),
	}, nil
}

func (g *Gateway) HealthCheck(ctx context.Context) error {
	g.mu.RLock()
	client := g.serviceCli
	sid := g.sessionID
	g.mu.RUnlock()

	if client == nil || sid == "" {
		return fmt.Errorf("mt5: not connected")
	}

	hcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	_, err := client.Health(metadata.NewOutgoingContext(hcCtx, md), &pb.HealthRequest{})
	if err != nil {
		return fmt.Errorf("mt5: health check failed: %w", err)
	}
	return nil
}

func (g *Gateway) SessionID() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.sessionID
}

// SetReconnecting guards against recvLoop races during managed reconnection.
func (g *Gateway) SetReconnecting(v bool) {
	g.mu.Lock()
	g.reconnecting = v
	g.mu.Unlock()
}

// SetStatusCallback registers a callback for connection state changes.
// The callback must not block — it is called from recvLoop goroutines.
func (g *Gateway) SetStatusCallback(fn func(status, message string)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.onStatusChange = fn
}

// reportStatus invokes the registered status callback if set.
// Concurrency-safe: reads the callback field under RLock.
func (g *Gateway) reportStatus(status, message string) {
	g.mu.RLock()
	fn := g.onStatusChange
	g.mu.RUnlock()
	if fn != nil {
		fn(status, message)
	}
}

func (g *Gateway) MT5Client() pb.MT5Client {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.client
}
