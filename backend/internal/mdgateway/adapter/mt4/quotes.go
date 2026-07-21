package mt4

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt4"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

func (g *Gateway) Subscribe(ctx context.Context, syms []string, handler mdtick.TickHandler) error {
	g.mu.RLock()
	sc := g.streamCli
	sub := g.subCli
	sid := g.sessionID
	g.mu.RUnlock()
	if sc == nil {
		return fmt.Errorf("mt4: not connected")
	}
	// Persist symbols for re-subscription after reconnect.
	g.mu.Lock()
	g.subscribedSymbols = append(g.subscribedSymbols[:0], syms...)
	g.mu.Unlock()
	if sub != nil && len(syms) > 0 {
		subMd := metadata.New(map[string]string{"id": sid})
		if tok := g.token(); tok != "" {
			subMd.Set("authorization", "Bearer "+tok)
		}
		subCtx := metadata.NewOutgoingContext(ctx, subMd)
		if _, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: syms}); err != nil {
			g.log.Warn("mt4: subscribe symbols failed", zap.Strings("syms", syms), zap.Error(err))
		} else {
			g.log.Info("mt4: subscribed symbols", zap.Strings("syms", syms))
		}
	}
	go g.recvLoop(ctx, handler)
	return nil
}

// AddSymbols subscribes to additional symbols on the existing MT4 session
// without starting a new quote stream. The existing recvLoop OnQuote stream
// will automatically deliver ticks for the newly added symbols.
func (g *Gateway) AddSymbols(ctx context.Context, symbols []string) error {
	g.mu.RLock()
	sub := g.subCli
	sid := g.sessionID
	g.mu.RUnlock()
	if sub == nil || sid == "" {
		return fmt.Errorf("mt4 AddSymbols: not connected")
	}
	if len(symbols) == 0 {
		return nil
	}
	subMd := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		subMd.Set("authorization", "Bearer "+tok)
	}
	subCtx := metadata.NewOutgoingContext(ctx, subMd)
	_, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: symbols})
	if err != nil {
		return fmt.Errorf("mt4 AddSymbols: %w", err)
	}
	// Persist for re-subscription after reconnect.
	g.mu.Lock()
	g.subscribedSymbols = append(g.subscribedSymbols, symbols...)
	g.mu.Unlock()
	g.log.Info("mt4: added symbols", zap.Strings("syms", symbols))
	return nil
}

func (g *Gateway) recvLoop(ctx context.Context, handler mdtick.TickHandler) {
	const maxBackoff = 5 * time.Minute
	backoff := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := g.ensureConnected(ctx, &backoff, maxBackoff); err != nil {
			return
		}

		// Re-subscribe symbols after a reconnect. ensureConnected may have
		// performed a fresh Connect (new session), which loses prior subscriptions.
		g.mu.RLock()
		syms := append([]string{}, g.subscribedSymbols...)
		sub := g.subCli
		sid := g.sessionID
		g.mu.RUnlock()
		if sub != nil && len(syms) > 0 {
			subMd := metadata.New(map[string]string{"id": sid})
			if tok := g.token(); tok != "" {
				subMd.Set("authorization", "Bearer "+tok)
			}
			subCtx := metadata.NewOutgoingContext(ctx, subMd)
			if _, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: syms}); err != nil {
				g.log.Warn("mt4: re-subscribe symbols failed", zap.Strings("syms", syms), zap.Error(err))
			}
		}

		g.mu.RLock()
		sc := g.streamCli
		sid = g.sessionID
		g.mu.RUnlock()

		subCtx, cancel := context.WithCancel(ctx)
		g.mu.Lock()
		g.cancelSub = cancel
		g.mu.Unlock()

		md := metadata.New(map[string]string{"id": sid})
		if tok := g.token(); tok != "" {
			md.Set("authorization", "Bearer "+tok)
		}
		subCtx = metadata.NewOutgoingContext(subCtx, md)
		stream, err := sc.OnQuote(subCtx, &pb.OnQuoteRequest{Id: sid})
		if err != nil {
			g.log.Warn("mt4 subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			// Force disconnect so ensureConnected will do a fresh Connect
			// with a new session on the next iteration.
			// Skip on context cancellation — normal teardown, not a stream error.
			if err != context.Canceled && err != context.DeadlineExceeded {
				g.reportStatus("reconnecting", err.Error())
				_ = g.Disconnect(ctx)
				if g.breaker != nil {
					g.breaker.OnFailure()
				}
			}
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		g.reportStatus("connected", "")
		g.log.Info("mt4: quote stream active")
		for {
			quote, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt4 recv", zap.Error(err))
				cancel()
				// Force disconnect so ensureConnected will do a fresh Connect
				// with a new session on the next iteration.
				// Skip on context cancellation — normal teardown, not a stream error.
				if err != context.Canceled && err != context.DeadlineExceeded {
					g.reportStatus("reconnecting", err.Error())
					_ = g.Disconnect(ctx)
					if g.breaker != nil {
						g.breaker.OnFailure()
					}
				}
				break
			}
			q := quote.GetResult()
			if q == nil {
				continue
			}
			handler(&mdtick.Tick{
				UserID:        g.cfg.UserID,
				AccountID:     g.cfg.AccountID,
				Broker:        g.cfg.Broker,
				Platform:      "mt4",
				SymbolRaw:     q.GetSymbol(),
				Canonical:     "",
				TsUnixMs:      q.GetTime().AsTime().UnixMilli(),
				ArrivedUnixMs: Clk.Now().UTC().UnixMilli(),
				Bid:           decimal.NewFromFloat(q.GetBid()),
				Ask:           decimal.NewFromFloat(q.GetAsk()),
			})
		}
	}
}

func (g *Gateway) SubscribeProfit(ctx context.Context, handler mdtick.ProfitHandler) error {
	g.mu.RLock()
	sc := g.streamCli
	g.mu.RUnlock()
	if sc == nil {
		return fmt.Errorf("mt4: not connected")
	}
	go g.profitRecvLoop(ctx, handler)
	return nil
}

func (g *Gateway) profitRecvLoop(ctx context.Context, handler mdtick.ProfitHandler) {
	const maxBackoff = 5 * time.Minute
	backoff := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := g.ensureConnected(ctx, &backoff, maxBackoff); err != nil {
			return
		}

		g.mu.RLock()
		sc := g.streamCli
		sid := g.sessionID
		g.mu.RUnlock()

		subCtx, cancel := context.WithCancel(ctx)
		g.mu.Lock()
		g.cancelProfitSub = cancel
		g.mu.Unlock()

		md := metadata.New(map[string]string{"id": sid})
		if tok := g.token(); tok != "" {
			md.Set("authorization", "Bearer "+tok)
		}
		subCtx = metadata.NewOutgoingContext(subCtx, md)
		stream, err := sc.OnOrderProfit(subCtx, &pb.OnOrderProfitRequest{Id: sid})
		if err != nil {
			g.log.Warn("mt4 profit subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			// Force disconnect so ensureConnected will do a fresh Connect
			// with a new session on the next iteration.
			// Skip on context cancellation — normal teardown, not a stream error.
			if err != context.Canceled && err != context.DeadlineExceeded {
				g.reportStatus("reconnecting", err.Error())
				_ = g.Disconnect(ctx)
				if g.breaker != nil {
					g.breaker.OnFailure()
				}
			}
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		g.reportStatus("connected", "")
		g.log.Info("mt4: profit stream active")
		for {
			resp, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt4 profit recv", zap.Error(err))
				cancel()
				// Force disconnect so ensureConnected will do a fresh Connect
				// with a new session on the next iteration.
				// Skip on context cancellation — normal teardown, not a stream error.
				if err != context.Canceled && err != context.DeadlineExceeded {
					g.reportStatus("reconnecting", err.Error())
					_ = g.Disconnect(ctx)
					if g.breaker != nil {
						g.breaker.OnFailure()
					}
				}
				break
			}
			p := resp.GetResult()
			if p == nil {
				continue
			}
			// Use Decimal for financial arithmetic to avoid float64 rounding.
			equityD := decimal.NewFromFloat(p.GetEquity())
			balanceD := decimal.NewFromFloat(p.GetBalance())
			profitD := equityD.Sub(balanceD)
			var profitPercent float64
			if !balanceD.IsZero() {
				profitPercent = profitD.Div(balanceD).Mul(decimal.NewFromInt(100)).InexactFloat64()
			}
			positions := make([]mdtick.ProfitPosition, 0, len(p.GetOrders()))
			for _, o := range p.GetOrders() {
				positions = append(positions, mdtick.ProfitPosition{
					Ticket:       int64(o.GetTicket()),
					Symbol:       o.GetSymbol(),
					Profit:       decimal.NewFromFloat(o.GetProfit()),
					Volume:       decimal.NewFromFloat(o.GetLots()),
					CurrentPrice: decimal.NewFromFloat(o.GetClosePrice()),
				})
			}
			handler(&mdtick.ProfitUpdate{
				AccountID:     g.cfg.AccountID,
				Platform:      "mt4",
				Balance:       decimal.NewFromFloat(p.GetBalance()),
				Credit:        decimal.NewFromFloat(p.GetCredit()),
				Equity:        decimal.NewFromFloat(p.GetEquity()),
				Margin:        decimal.NewFromFloat(p.GetMargin()),
				FreeMargin:    decimal.NewFromFloat(p.GetFreeMargin()),
				MarginLevel:   decimal.NewFromFloat(p.GetMarginLevel()),
				Profit:        profitD,
				ProfitPercent: profitPercent,
				Positions:     positions,
			})
		}
	}
}
