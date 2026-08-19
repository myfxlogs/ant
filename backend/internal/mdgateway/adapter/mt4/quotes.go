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
		// LIVE-PRICE-4: Subscribe per-symbol to avoid atomic batch failure
		// when one symbol doesn't exist on the broker. Non-existent symbols
		// are skipped (logged) instead of failing the entire subscription.
		subscribed := 0
		for _, sym := range syms {
			subCtx := metadata.NewOutgoingContext(ctx, subMd)
			resp, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: []string{sym}})
			if err != nil {
				g.log.Warn("mt4: subscribe symbol RPC failed", zap.String("sym", sym), zap.Error(err))
				continue
			}
			if e := resp.GetError(); e != nil && e.GetCode() != 0 {
				g.log.Warn("mt4: subscribe symbol rejected by mtapi", zap.String("sym", sym),
					zap.Int32("code", int32(e.GetCode())), zap.String("msg", e.GetMessage()))
				continue
			}
			subscribed++
		}
		g.log.Info("mt4: subscribed symbols", zap.Int("requested", len(syms)), zap.Int("subscribed", subscribed))
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
	resp, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: symbols})
	if err != nil {
		return fmt.Errorf("mt4 AddSymbols: %w", err)
	}
	if e := resp.GetError(); e != nil && e.GetCode() != 0 {
		return fmt.Errorf("mt4 AddSymbols: mtapi rejected: code=%d msg=%s", e.GetCode(), e.GetMessage())
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
			g.log.Warn("mt4 subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.handleStreamError(ctx, err, &backoff)
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
				g.handleStreamError(ctx, err, &backoff)
				goto quoteLoopEnd
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
	quoteLoopEnd:
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
			g.log.Warn("mt4: re-subscribe symbol RPC failed", zap.String("sym", sym), zap.Error(err))
			continue
		}
		if e := resp.GetError(); e != nil && e.GetCode() != 0 {
			g.log.Warn("mt4: re-subscribe symbol rejected by mtapi", zap.String("sym", sym),
				zap.Int32("code", int32(e.GetCode())), zap.String("msg", e.GetMessage()))
			continue
		}
		subscribed++
	}
	g.log.Info("mt4: re-subscribed symbols", zap.Int("requested", len(syms)), zap.Int("subscribed", subscribed))
}

func (g *Gateway) profitRecvTimeout() time.Duration {
	g.mu.RLock()
	lu := g.lastProfitUpdate
	g.mu.RUnlock()
	if lu == nil {
		// First frame on a fresh connection; give mtapi a bit of time to push it.
		return 30 * time.Second
	}
	if len(lu.Positions) > 0 || lu.Margin.GreaterThan(decimal.Zero) {
		// Account has open positions/margin: OnOrderProfit should fire very frequently.
		return 15 * time.Second
	}
	// Empty account: profit stream may legitimately be quiet, but still should heartbeat periodically.
	return 60 * time.Second
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
			g.log.Warn("mt4 profit subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.handleStreamError(ctx, err, &backoff)
			continue
		}

		backoff = time.Second
		g.reportStatus("connected", "")
		g.log.Info("mt4: profit stream active")
		for {
			timeout := g.profitRecvTimeout()
			timer := time.NewTimer(timeout)

			type recvResult struct {
				resp *pb.OnOrderProfitReply
				err  error
			}
			ch := make(chan recvResult, 1)
			go func() {
				resp, err := stream.Recv()
				select {
				case ch <- recvResult{resp: resp, err: err}:
				case <-subCtx.Done():
				}
			}()

			select {
			case <-subCtx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				cancel()
				goto profitLoopEnd
			case <-timer.C:
				g.log.Warn("mt4 profit stream silence timeout; retrying profit stream",
					zap.Duration("timeout", timeout), zap.String("account", g.cfg.AccountID))
				cancel()
				// Do NOT call Disconnect() — that tears down the shared connection
				// including the quote stream (OnQuote). For empty accounts (no positions,
				// no margin), OnOrderProfit legitimately never pushes data, so this
				// timeout fires repeatedly. Disconnecting destroys the quote stream
				// which IS receiving ticks, causing data starvation for live strategies.
				// Just sleep with backoff and retry the profit stream on the same connection.
				g.sleep(ctx, backoff)
				backoff = minDuration(backoff*2, maxBackoff)
				goto profitLoopEnd
			case r := <-ch:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				if r.err != nil {
					g.log.Warn("mt4 profit recv", zap.Error(r.err))
					cancel()
					g.handleStreamError(ctx, r.err, &backoff)
					goto profitLoopEnd
				}
				p := r.resp.GetResult()
				if p == nil {
					continue
				}
				pu := parseMt4ProfitUpdate(p, g.cfg.AccountID)
				g.mu.Lock()
				g.lastProfitUpdate = pu
				g.lastProfitAt = time.Now()
				g.mu.Unlock()
				handler(pu)
			}
		}
	profitLoopEnd:
	}
}

func parseMt4ProfitUpdate(p *pb.ProfitUpdate, accountID string) *mdtick.ProfitUpdate {
	equityD := decimal.NewFromFloat(p.GetEquity())
	balanceD := decimal.NewFromFloat(p.GetBalance())
	profitD := equityD.Sub(balanceD)
	var profitPercent float64
	if !balanceD.IsZero() {
		profitPercent = profitD.Div(balanceD).Mul(decimal.NewFromInt(100)).InexactFloat64()
	}
	positions := make([]mdtick.ProfitPosition, 0, len(p.GetOrders()))
	for _, o := range p.GetOrders() {
		typ := "buy"
		if o.GetType() == pb.Op_Op_Sell {
			typ = "sell"
		}
		openTime := int64(0)
		if t := o.GetOpenTime(); t != nil {
			openTime = t.AsTime().Unix()
		}
		positions = append(positions, mdtick.ProfitPosition{
			Ticket:       int64(o.GetTicket()),
			Symbol:       o.GetSymbol(),
			Type:         typ,
			Magic:        o.GetMagicNumber(),
			Volume:       decimal.NewFromFloat(o.GetLots()),
			OpenPrice:    decimal.NewFromFloat(o.GetOpenPrice()),
			CurrentPrice: decimal.NewFromFloat(o.GetClosePrice()),
			StopLoss:     decimal.NewFromFloat(o.GetStopLoss()),
			TakeProfit:   decimal.NewFromFloat(o.GetTakeProfit()),
			Profit:       decimal.NewFromFloat(o.GetProfit()),
			Swap:         decimal.NewFromFloat(o.GetSwap()),
			Commission:   decimal.NewFromFloat(o.GetCommission()),
			Comment:      o.GetComment(),
			OpenTime:     openTime,
		})
	}
	return &mdtick.ProfitUpdate{
		AccountID:     accountID,
		Platform:      "mt4",
		Balance:       balanceD,
		Credit:        decimal.NewFromFloat(p.GetCredit()),
		Equity:        equityD,
		Margin:        decimal.NewFromFloat(p.GetMargin()),
		FreeMargin:    decimal.NewFromFloat(p.GetFreeMargin()),
		MarginLevel:   decimal.NewFromFloat(p.GetMarginLevel()),
		Profit:        profitD,
		ProfitPercent: profitPercent,
		Positions:     positions,
	}
}
