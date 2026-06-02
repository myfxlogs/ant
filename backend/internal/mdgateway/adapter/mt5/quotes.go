package mt5

import (
	"context"
	"fmt"
	"time"

	pb "anttrader/mt5"
	"anttrader/internal/mdgateway/adapter/mdtick"
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
	if sub != nil && len(syms) > 0 {
		subMd := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
		subCtx := metadata.NewOutgoingContext(ctx, subMd)
		if _, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: syms}); err != nil {
			g.log.Warn("mt5: subscribe symbols failed", zap.Strings("syms", syms), zap.Error(err))
		} else {
			g.log.Info("mt5: subscribed symbols", zap.Strings("syms", syms))
		}
	}
	// #nosec G118 — recvLoop runs for the gateway full connection lifetime
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
	subMd := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	subCtx := metadata.NewOutgoingContext(ctx, subMd)
	_, err := sub.SubscribeMany(subCtx, &pb.SubscribeManyRequest{Id: sid, Symbols: symbols})
	if err != nil {
		return fmt.Errorf("mt5 AddSymbols: %w", err)
	}
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

		g.mu.RLock()
		sc := g.streamCli
		sid := g.sessionID
		g.mu.RUnlock()

		subCtx, cancel := context.WithCancel(ctx)
		g.mu.Lock()
		g.cancelSub = cancel
		g.mu.Unlock()

		md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
		subCtx = metadata.NewOutgoingContext(subCtx, md)
		stream, err := sc.OnQuote(subCtx, &pb.OnQuoteRequest{Id: sid})
		if err != nil {
			g.log.Warn("mt5 subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		g.log.Info("mt5: quote stream active")
		for {
			tick, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt5 recv", zap.Error(err))
				cancel()
				break
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
	}
}

// fetchAndPublish calls AccountSummary (canonical MQL5 values) and publishes
// via the handler. If AccountSummary fails, falls back to stream-derived values
// from p; if p is nil (initial call before any stream frame), skips.
func (g *Gateway) fetchAndPublish(ctx context.Context, sid string, p *pb.ProfitUpdate, handler mdtick.ProfitHandler) {
	var balance, equity, profit, margin, freeMargin, marginLevel, credit float64

	asMd := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	sctx, scancel := context.WithTimeout(ctx, 3*time.Second)
	defer scancel()
	asCtx := metadata.NewOutgoingContext(sctx, asMd)
	acct, err := g.client.AccountSummary(asCtx, &pb.AccountSummaryRequest{Id: sid})

	if err == nil && acct != nil && acct.GetResult() != nil {
		s := acct.GetResult()
		balance = s.GetBalance()
		equity = s.GetEquity()
		profit = s.GetProfit()
		margin = s.GetMargin()
		freeMargin = s.GetFreeMargin()
		marginLevel = s.GetMarginLevel()
		credit = s.GetCredit()
	} else if p != nil {
		g.log.Debug("mt5 AccountSummary failed; falling back to stream frame",
			zap.String("account_id", g.cfg.AccountID), zap.Error(err))
		balance = p.GetBalance()
		equity = p.GetEquity()
		profit = equity - balance
		margin = p.GetMargin()
		freeMargin = p.GetFreeMargin()
		marginLevel = p.GetMarginLevel()
		credit = p.GetCredit()
	} else {
		g.log.Warn("mt5 initial AccountSummary failed; no data to publish",
			zap.String("account_id", g.cfg.AccountID), zap.Error(err))
		return
	}

	var profitPercent float64
	if balance > 0 {
		profitPercent = profit / balance * 100
	}

	var positions []mdtick.ProfitPosition
	if p != nil {
		positions = make([]mdtick.ProfitPosition, 0, len(p.GetOrders()))
		for _, o := range p.GetOrders() {
			positions = append(positions, mdtick.ProfitPosition{
				Ticket:       o.GetTicket(),
				Symbol:       o.GetSymbol(),
				Profit:       o.GetProfit(),
				Volume:       o.GetLots(),
				CurrentPrice: o.GetOpenPrice(),
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
	// #nosec G118 — profitRecvLoop runs for the gateway full connection lifetime
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

		md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
		subCtx = metadata.NewOutgoingContext(subCtx, md)
		stream, err := sc.OnOrderProfit(subCtx, &pb.OnOrderProfitRequest{Id: sid})
		if err != nil {
			g.log.Warn("mt5 profit subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		g.log.Info("mt5: profit stream active")

		// Call AccountSummary once for initial snapshot. MT5 OnOrderProfit
		// only fires when positions change, so without this the frontend
		// would see stale data for idle accounts. AccountSummary is the
		// canonical source (MQL5 AccountInfoDouble).
		g.fetchAndPublish(ctx, sid, nil, handler)

		for {
			resp, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt5 profit recv", zap.Error(err))
				cancel()
				break
			}
			p := resp.GetResult()
			if p == nil {
				continue
			}

			// On each stream frame, fetch canonical AccountSummary.
			// Falls back to stream-derived values on RPC failure.
			g.fetchAndPublish(ctx, sid, p, handler)
		}
	}
}
