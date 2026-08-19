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
		// LIVE-PRICE-4: Subscribe per-symbol to avoid atomic batch failure.
		subscribed := 0
		for _, sym := range syms {
			subCtx := metadata.NewOutgoingContext(ctx, subMd)
			resp, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: []string{sym}})
			if err != nil {
				g.log.Warn("mt5: subscribe symbol RPC failed", zap.String("sym", sym), zap.Error(err))
				continue
			}
			if e := resp.GetError(); e != nil && e.GetCode() != 0 {
				g.log.Warn("mt5: subscribe symbol rejected by mtapi", zap.String("sym", sym),
					zap.Int32("code", int32(e.GetCode())), zap.String("msg", e.GetMessage()))
				continue
			}
			subscribed++
		}
		g.log.Info("mt5: subscribed symbols", zap.Int("requested", len(syms)), zap.Int("subscribed", subscribed))
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
		if sc == nil || sid == "" {
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
			tick, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt5 recv", zap.Error(err))
				cancel()
				g.handleStreamError(ctx, err, &backoff)
				goto mt5QuoteLoopEnd
			}
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
		}
	mt5QuoteLoopEnd:
	}
}

func (g *Gateway) reSubscribeSymbols(ctx context.Context) {
	g.mu.RLock()
	syms := append([]string{}, g.subscribedSymbols...)
	sub := g.subCli
	sid := g.sessionID
	g.mu.RUnlock()
	if sub == nil || sid == "" || len(syms) == 0 {
		return
	}
	subMd := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		subMd.Set("authorization", "Bearer "+tok)
	}
	// LIVE-PRICE-4: Per-symbol re-subscribe, skip failures.
	subscribed := 0
	for _, sym := range syms {
		subCtx := metadata.NewOutgoingContext(ctx, subMd)
		resp, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: []string{sym}})
		if err != nil {
			g.log.Warn("mt5: re-subscribe symbol RPC failed", zap.String("sym", sym), zap.Error(err))
			continue
		}
		if e := resp.GetError(); e != nil && e.GetCode() != 0 {
			g.log.Warn("mt5: re-subscribe symbol rejected by mtapi", zap.String("sym", sym),
				zap.Int32("code", int32(e.GetCode())), zap.String("msg", e.GetMessage()))
			continue
		}
		subscribed++
	}
	g.log.Info("mt5: re-subscribed symbols", zap.Int("requested", len(syms)), zap.Int("subscribed", subscribed))
}

// fetchAndPublish calls AccountSummary (canonical MQL5 values) and publishes
// via the handler. If AccountSummary fails, no financial snapshot is published;
// p contributes positions only.
func (g *Gateway) fetchAndPublish(ctx context.Context, sid string, p *pb.ProfitUpdate, handler mdtick.ProfitHandler) {
	g.mu.RLock()
	client := g.client
	g.mu.RUnlock()
	if client == nil {
		g.log.Error("mt5 AccountSummary unavailable; financial snapshot rejected",
			zap.String("account_id", g.cfg.AccountID))
		return
	}

	asMd := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		asMd.Set("authorization", "Bearer "+tok)
	}
	sctx, scancel := context.WithTimeout(ctx, 3*time.Second)
	defer scancel()
	asCtx := metadata.NewOutgoingContext(sctx, asMd)
	acct, err := client.AccountSummary(asCtx, &pb.AccountSummaryRequest{Id: sid})
	if err != nil || acct == nil || acct.GetResult() == nil || (acct.GetError() != nil && acct.GetError().GetCode() != 0) {
		g.log.Error("mt5 AccountSummary failed; financial snapshot rejected",
			zap.String("account_id", g.cfg.AccountID), zap.Error(err))
		return
	}

	s := acct.GetResult()
	balance := decimal.NewFromFloat(s.GetBalance())
	equity := decimal.NewFromFloat(s.GetEquity())
	profit := decimal.NewFromFloat(s.GetProfit())
	margin := decimal.NewFromFloat(s.GetMargin())
	freeMargin := decimal.NewFromFloat(s.GetFreeMargin())
	marginLevel := decimal.NewFromFloat(s.GetMarginLevel())
	credit := decimal.NewFromFloat(s.GetCredit())

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
				Magic:        int32(o.GetExpertId()),
				Profit:       decimal.NewFromFloat(o.GetProfit()),
				Volume:       decimal.NewFromFloat(o.GetLots()),
				CurrentPrice: decimal.NewFromFloat(o.GetOpenPrice()),
			})
		}
	}

	handler(&mdtick.ProfitUpdate{
		AccountID:       g.cfg.AccountID,
		Platform:        "mt5",
		Balance:         balance,
		Credit:          credit,
		Equity:          equity,
		Margin:          margin,
		FreeMargin:      freeMargin,
		MarginLevel:     marginLevel,
		Profit:          profit,
		ProfitPercent:   profitPercent,
		Leverage:        int32(s.GetLeverage()),
		FinancialSource: mdtick.FinancialsSourceAccountSummary,
		CapturedAt:      Clk.Now(),
		Positions:       positions,
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
		if sc == nil || sid == "" {
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
			resp, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt5 profit recv", zap.Error(err))
				cancel()
				g.handleStreamError(ctx, err, &backoff)
				goto mt5ProfitLoopEnd
			}
			p := resp.GetResult()
			if p == nil {
				continue
			}
			g.fetchAndPublish(ctx, sid, p, handler)
		}
	mt5ProfitLoopEnd:
	}
}
