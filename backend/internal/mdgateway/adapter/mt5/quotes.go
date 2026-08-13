package mt5

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt5"

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
		return fmt.Errorf("mt5: not connected")
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
		resp, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: syms})
		if err != nil {
			g.log.Warn("mt5: subscribe symbols RPC failed", zap.Strings("syms", syms), zap.Error(err))
		} else if e := resp.GetError(); e != nil && e.GetCode() != 0 {
			g.log.Error("mt5: subscribe symbols rejected by mtapi", zap.Strings("syms", syms),
				zap.Int32("code", int32(e.GetCode())), zap.String("msg", e.GetMessage()))
		} else {
			g.log.Info("mt5: subscribed symbols", zap.Strings("syms", syms))
		}
	}
	go g.recvLoop(ctx, handler)
	return nil
}

// AddSymbols subscribes to additional symbols on the existing MT5 session
// without starting a new quote stream. The existing recvLoop OnQuote stream
// will automatically deliver ticks for the newly added symbols.
func (g *Gateway) AddSymbols(ctx context.Context, symbols []string) error {
	g.mu.RLock()
	sub := g.subCli
	sid := g.sessionID
	g.mu.RUnlock()
	if sub == nil || sid == "" {
		return fmt.Errorf("mt5 AddSymbols: not connected")
	}
	if len(symbols) == 0 {
		return nil
	}
	subMd := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		subMd.Set("authorization", "Bearer "+tok)
	}
	subCtx := metadata.NewOutgoingContext(ctx, subMd)
	resp, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: symbols})
	if err != nil {
		return fmt.Errorf("mt5 AddSymbols: %w", err)
	}
	if e := resp.GetError(); e != nil && e.GetCode() != 0 {
		return fmt.Errorf("mt5 AddSymbols: mtapi rejected: code=%d msg=%s", e.GetCode(), e.GetMessage())
	}
	// Persist for re-subscription after reconnect.
	g.mu.Lock()
	g.subscribedSymbols = append(g.subscribedSymbols, symbols...)
	g.mu.Unlock()
	g.log.Info("mt5: added symbols", zap.Strings("syms", symbols))
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

		g.reSubscribeSymbols(ctx)

		g.mu.RLock()
		sc := g.streamCli
		sid := g.sessionID
		g.mu.RUnlock()
		if sc == nil {
			g.sleep(ctx, time.Second)
			continue
		}

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
			g.log.Warn("mt5 subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.handleStreamError(ctx, err, &backoff)
			continue
		}

		backoff = time.Second
		g.reportStatus("connected", "")
		g.log.Info("mt5: quote stream active")
		for {
			recvCh := make(chan *pb.OnQuoteReply, 1)
			errCh := make(chan error, 1)
			go func() {
				tick, err := stream.Recv()
				if err != nil {
					errCh <- err
					return
				}
				recvCh <- tick
			}()
			select {
			case tick := <-recvCh:
				q := tick.GetResult()
				if q == nil {
					continue
				}
				handler(&mdtick.Tick{
					UserID:        g.cfg.UserID,
					AccountID:     g.cfg.AccountID,
					Broker:        g.cfg.Broker,
					Platform:      "mt5",
					SymbolRaw:     q.GetSymbol(),
					Canonical:     "",
					TsUnixMs:      q.GetTime().AsTime().UnixMilli(),
					ArrivedUnixMs: Clk.Now().UTC().UnixMilli(),
					Bid:           decimal.NewFromFloat(q.GetBid()),
					Ask:           decimal.NewFromFloat(q.GetAsk()),
					BidVolume:     float64(q.GetVolume()),
				})
			case err := <-errCh:
				g.log.Warn("mt5 recv", zap.Error(err))
				cancel()
				g.handleStreamError(ctx, err, &backoff)
				goto mt5QuoteLoopEnd
			case <-time.After(g.quoteTimeoutOrDefault()):
				g.log.Warn("mt5 quote stream: no data — treating as dead", zap.String("account", g.cfg.AccountID), zap.Duration("timeout", g.quoteTimeoutOrDefault()))
				cancel()
				g.handleStreamError(ctx, fmt.Errorf("quote stream: no data timeout"), &backoff)
				goto mt5QuoteLoopEnd
			}
		}
	mt5QuoteLoopEnd:
	}
}

func (g *Gateway) quoteTimeoutOrDefault() time.Duration {
	if g.quoteTimeout > 0 {
		return g.quoteTimeout
	}
	return 90 * time.Second
}

func (g *Gateway) reSubscribeSymbols(ctx context.Context) {
	g.mu.RLock()
	syms := append([]string{}, g.subscribedSymbols...)
	sub := g.subCli
	sid := g.sessionID
	g.mu.RUnlock()
	if sub == nil || len(syms) == 0 {
		return
	}
	subMd := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		subMd.Set("authorization", "Bearer "+tok)
	}
	subCtx := metadata.NewOutgoingContext(ctx, subMd)
	resp, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: syms})
	if err != nil {
		g.log.Warn("mt5: re-subscribe symbols RPC failed", zap.Strings("syms", syms), zap.Error(err))
	} else if e := resp.GetError(); e != nil && e.GetCode() != 0 {
		g.log.Error("mt5: re-subscribe symbols rejected by mtapi", zap.Strings("syms", syms),
			zap.Int32("code", int32(e.GetCode())), zap.String("msg", e.GetMessage()))
	}
}

// fetchAndPublish calls AccountSummary (canonical MQL5 values) and publishes
// via the handler. If AccountSummary fails, falls back to stream-derived values
// from p; if p is nil (initial call before any stream frame), skips.
func (g *Gateway) fetchAndPublish(ctx context.Context, sid string, p *pb.ProfitUpdate, handler mdtick.ProfitHandler) {
	var balance, equity, profit, margin, freeMargin, marginLevel, credit decimal.Decimal

	asMd := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		asMd.Set("authorization", "Bearer "+tok)
	}
	sctx, scancel := context.WithTimeout(ctx, 3*time.Second)
	defer scancel()
	asCtx := metadata.NewOutgoingContext(sctx, asMd)
	acct, err := g.client.AccountSummary(asCtx, &pb.AccountSummaryRequest{Id: sid})

	if err == nil && acct != nil && acct.GetResult() != nil {
		s := acct.GetResult()
		balance = decimal.NewFromFloat(s.GetBalance())
		equity = decimal.NewFromFloat(s.GetEquity())
		profit = decimal.NewFromFloat(s.GetProfit())
		margin = decimal.NewFromFloat(s.GetMargin())
		freeMargin = decimal.NewFromFloat(s.GetFreeMargin())
		marginLevel = decimal.NewFromFloat(s.GetMarginLevel())
		credit = decimal.NewFromFloat(s.GetCredit())
	} else if p != nil {
		g.log.Debug("mt5 AccountSummary failed; falling back to stream frame",
			zap.String("account_id", g.cfg.AccountID), zap.Error(err))
		balance = decimal.NewFromFloat(p.GetBalance())
		equity = decimal.NewFromFloat(p.GetEquity())
		profit = equity.Sub(balance)
		margin = decimal.NewFromFloat(p.GetMargin())
		freeMargin = decimal.NewFromFloat(p.GetFreeMargin())
		marginLevel = decimal.NewFromFloat(p.GetMarginLevel())
		credit = decimal.NewFromFloat(p.GetCredit())
	} else {
		g.log.Warn("mt5 initial AccountSummary failed; no data to publish",
			zap.String("account_id", g.cfg.AccountID), zap.Error(err))
		return
	}

	var profitPercent float64
	if balance.GreaterThan(decimal.Zero) {
		profitPercent = profit.Div(balance).Mul(decimal.NewFromInt(100)).InexactFloat64()
	}

	var positions []mdtick.ProfitPosition
	if p != nil {
		positions = make([]mdtick.ProfitPosition, 0, len(p.GetOrders()))
		for _, o := range p.GetOrders() {
			positions = append(positions, mdtick.ProfitPosition{
				Ticket:       o.GetTicket(),
				Symbol:       o.GetSymbol(),
				Profit:       decimal.NewFromFloat(o.GetProfit()),
				Volume:       decimal.NewFromFloat(o.GetLots()),
				CurrentPrice: decimal.NewFromFloat(o.GetOpenPrice()),
			})
		}
	}

	handler(&mdtick.ProfitUpdate{
		AccountID:     g.cfg.AccountID,
		Platform:      "mt5",
		Balance:       balance,
		Credit:        credit,
		Equity:        equity,
		Margin:        margin,
		FreeMargin:    freeMargin,
		MarginLevel:   marginLevel,
		Profit:        profit,
		ProfitPercent: profitPercent,
		Positions:     positions,
	})
}

func (g *Gateway) SubscribeProfit(ctx context.Context, handler mdtick.ProfitHandler) error {
	g.mu.RLock()
	sc := g.streamCli
	g.mu.RUnlock()
	if sc == nil {
		return fmt.Errorf("mt5: not connected")
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
		if sc == nil {
			g.sleep(ctx, time.Second)
			continue
		}

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
			g.log.Warn("mt5 profit subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.handleStreamError(ctx, err, &backoff)
			continue
		}

		backoff = time.Second
		g.reportStatus("connected", "")
		g.log.Info("mt5: profit stream active")

		g.fetchAndPublish(ctx, sid, nil, handler)

		for {
			recvCh := make(chan *pb.OnOrderProfitReply, 1)
			errCh := make(chan error, 1)
			go func() {
				resp, err := stream.Recv()
				if err != nil {
					errCh <- err
					return
				}
				recvCh <- resp
			}()
			select {
			case resp := <-recvCh:
				p := resp.GetResult()
				if p == nil {
					continue
				}
				g.fetchAndPublish(ctx, sid, p, handler)
			case err := <-errCh:
				g.log.Warn("mt5 profit recv", zap.Error(err))
				cancel()
				g.handleStreamError(ctx, err, &backoff)
				goto mt5ProfitLoopEnd
			case <-time.After(90 * time.Second):
				g.log.Warn("mt5 profit stream: no data for 90s — treating as dead")
				cancel()
				g.handleStreamError(ctx, fmt.Errorf("profit stream: no data timeout"), &backoff)
				goto mt5ProfitLoopEnd
			}
		}
	mt5ProfitLoopEnd:
	}
}
