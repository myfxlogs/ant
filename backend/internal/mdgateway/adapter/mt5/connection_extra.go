package mt5

import (
	"context"
	"fmt"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt5"
	"google.golang.org/grpc/metadata"
)

// IsQuoteSession returns true if the broker is currently quoting the symbol.
func (g *Gateway) IsQuoteSession(ctx context.Context, symbol string) (bool, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return false, fmt.Errorf("mt5: not connected")
	}

	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	resp, err := client.IsQuoteSession(metadata.NewOutgoingContext(ctx, md), &pb.IsQuoteSessionRequest{
		Id:     sid,
		Symbol: symbol,
	})
	if err != nil {
		return false, fmt.Errorf("mt5 IsQuoteSession: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return false, fmt.Errorf("mt5 IsQuoteSession: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetResult(), nil
}

// IsTradeSession returns true if the broker is currently trading the symbol.
func (g *Gateway) IsTradeSession(ctx context.Context, symbol string) (bool, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return false, fmt.Errorf("mt5: not connected")
	}

	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	resp, err := client.IsTradeSession(metadata.NewOutgoingContext(ctx, md), &pb.IsTradeSessionRequest{
		Id:     sid,
		Symbol: symbol,
	})
	if err != nil {
		return false, fmt.Errorf("mt5 IsTradeSession: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return false, fmt.Errorf("mt5 IsTradeSession: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetResult(), nil
}

// SymbolSessionsEx returns the raw exchange sessions for a symbol.
func (g *Gateway) SymbolSessionsEx(ctx context.Context, symbol string) (*pb.SymbolSessionsEx, error) {
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
	resp, err := client.SymbolSessionsEx(metadata.NewOutgoingContext(ctx, md), &pb.SymbolSessionsExRequest{
		Id:     sid,
		Symbol: symbol,
	})
	if err != nil {
		return nil, fmt.Errorf("mt5 SymbolSessionsEx: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt5 SymbolSessionsEx: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetResult(), nil
}

// TickValueWithSize returns the tick value and size for the requested symbols.
func (g *Gateway) TickValueWithSize(ctx context.Context, symbols []string) ([]*pb.TickValueWithSize, error) {
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
	resp, err := client.TickValueWithSize(metadata.NewOutgoingContext(ctx, md), &pb.TickValueWithSizeRequest{
		Id:     sid,
		Symbol: symbols,
	})
	if err != nil {
		return nil, fmt.Errorf("mt5 TickValueWithSize: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt5 TickValueWithSize: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetResult(), nil
}

// PriceHistoryToday returns today's bars for a symbol/timeframe via the QuoteHistory client.
func (g *Gateway) PriceHistoryToday(ctx context.Context, symbol, period string) ([]*mdtick.Bar, error) {
	g.mu.RLock()
	qhCli := g.qhCli
	sid := g.sessionID
	g.mu.RUnlock()
	if qhCli == nil || sid == "" {
		return nil, fmt.Errorf("mt5 GetPriceHistory: not connected")
	}

	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	resp, err := qhCli.PriceHistoryToday(metadata.NewOutgoingContext(ctx, md), &pb.PriceHistoryTodayRequest{
		Id:        sid,
		Symbol:    symbol,
		TimeFrame: mt5PeriodToTimeframe(period),
	})
	if err != nil {
		return nil, fmt.Errorf("mt5 PriceHistoryToday: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt5 PriceHistoryToday: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return convertMT5Bars(resp.GetResult(), g.cfg.AccountID, period), nil
}

// SubscribeMarketWatch subscribes to market watch updates for the session.
func (g *Gateway) SubscribeMarketWatch(ctx context.Context) (string, error) {
	g.mu.RLock()
	sub := g.subCli
	sid := g.sessionID
	g.mu.RUnlock()
	if sub == nil || sid == "" {
		return "", fmt.Errorf("mt5: not connected")
	}

	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	resp, err := sub.SubscribeMarketWatch(metadata.NewOutgoingContext(ctx, md), &pb.SubscribeMarketWatchRequest{Id: sid})
	if err != nil {
		return "", fmt.Errorf("mt5 SubscribeMarketWatch: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return "", fmt.Errorf("mt5 SubscribeMarketWatch: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetResult(), nil
}

// SubscribeOpenedOrdersTickets subscribes to opened-orders ticket updates with an optional interval in milliseconds.
func (g *Gateway) SubscribeOpenedOrdersTickets(ctx context.Context, intervalMs int32) (string, error) {
	g.mu.RLock()
	sub := g.subCli
	sid := g.sessionID
	g.mu.RUnlock()
	if sub == nil || sid == "" {
		return "", fmt.Errorf("mt5: not connected")
	}

	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	req := &pb.SubscribeOpenedOrdersTicketsRequest{Id: sid}
	if intervalMs > 0 {
		v := intervalMs
		req.Interval = &v
	}
	resp, err := sub.SubscribeOpenedOrdersTickets(metadata.NewOutgoingContext(ctx, md), req)
	if err != nil {
		return "", fmt.Errorf("mt5 SubscribeOpenedOrdersTickets: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return "", fmt.Errorf("mt5 SubscribeOpenedOrdersTickets: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetResult(), nil
}
