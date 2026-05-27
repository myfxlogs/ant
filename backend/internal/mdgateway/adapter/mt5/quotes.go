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
	go g.recvLoop(ctx, handler)
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

// SubscribeOrderUpdate subscribes to MT5 OnOrderUpdate stream.
// Each event contains account metrics + full OpenedOrders list,
// providing real-time position updates (open/close/modify).
func (g *Gateway) SubscribeOrderUpdate(ctx context.Context, handler mdtick.OrderUpdateHandler) error {
	g.mu.RLock()
	sc := g.streamCli
	g.mu.RUnlock()
	if sc == nil {
		return fmt.Errorf("mt5: not connected")
	}
	go g.orderUpdateRecvLoop(ctx, handler)
	return nil
}

func (g *Gateway) orderUpdateRecvLoop(ctx context.Context, handler mdtick.OrderUpdateHandler) {
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
		g.cancelOrderUpdateSub = cancel
		g.mu.Unlock()

		md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
		subCtx = metadata.NewOutgoingContext(subCtx, md)
		stream, err := sc.OnOrderUpdate(subCtx, &pb.OnOrderUpdateRequest{Id: sid})
		if err != nil {
			g.log.Warn("mt5 order update subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		g.log.Info("mt5: order update stream active")
		for {
			resp, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt5 order update recv", zap.Error(err))
				cancel()
				break
			}
			s := resp.GetResult()
			if s == nil {
				continue
			}

			update := s.GetUpdate()
			var updateTicket int64
			var updateType string
			var updateSymbol string
			var updateVolume float64
			var updateOpenPrice float64
			var updateClosePrice float64
			var updateProfit float64
			var updateSwap float64
			var updateCommission float64
			var updateComment string
			var updateOpenTime int64
			var updateCloseTime int64
			var updateSL float64
			var updateTP float64
			if update != nil {
				o := update.GetOrder()
				updateTicket = o.GetTicket()
				updateType = mt5UpdateTypeLabel(update.GetType())
				updateSymbol = o.GetSymbol()
				updateVolume = o.GetLots()
				updateOpenPrice = o.GetOpenPrice()
				updateClosePrice = o.GetClosePrice()
				updateProfit = o.GetProfit()
				updateSwap = o.GetSwap()
				updateCommission = o.GetCommission()
				updateComment = o.GetComment()
				updateOpenTime = o.GetOpenTime().GetSeconds()
				updateCloseTime = o.GetCloseTime().GetSeconds()
				updateSL = o.GetStopLoss()
				updateTP = o.GetTakeProfit()
			}

			// Convert OpenedOrders to mdtick format.
			positions := make([]mdtick.OrderUpdatePosition, 0, len(s.GetOpenedOrders()))
			for _, o := range s.GetOpenedOrders() {
				positions = append(positions, mdtick.OrderUpdatePosition{
					Ticket:       o.GetTicket(),
					Symbol:       o.GetSymbol(),
					Type:         mt5OrderTypeLabel(o.GetOrderType()),
					Volume:       o.GetLots(),
					OpenPrice:    o.GetOpenPrice(),
					CurrentPrice: o.GetClosePrice(),
					StopLoss:     o.GetStopLoss(),
					TakeProfit:   o.GetTakeProfit(),
					Profit:       o.GetProfit(),
					Swap:         o.GetSwap(),
					Commission:   o.GetCommission(),
					Comment:      o.GetComment(),
					OpenTime:     o.GetOpenTime().GetSeconds(),
				})
			}

			profit := s.GetEquity() - s.GetBalance()
			handler(&mdtick.OrderUpdate{
				AccountID:        g.cfg.AccountID,
				Platform:         "mt5",
				UpdateTicket:     updateTicket,
				UpdateType:       updateType,
				UpdateSymbol:     updateSymbol,
				UpdateVolume:     updateVolume,
				UpdateOpenPrice:  updateOpenPrice,
				UpdateClosePrice: updateClosePrice,
				UpdateProfit:     updateProfit,
				UpdateSwap:       updateSwap,
				UpdateCommission: updateCommission,
				UpdateComment:    updateComment,
				UpdateOpenTime:   updateOpenTime,
				UpdateCloseTime:  updateCloseTime,
				UpdateSL:         updateSL,
				UpdateTP:         updateTP,
				Balance:          s.GetBalance(),
				Equity:           s.GetEquity(),
				Margin:           s.GetMargin(),
				FreeMargin:       s.GetFreeMargin(),
				MarginLevel:      s.GetMarginLevel(),
				Profit:           profit,
				Positions:        positions,
			})
		}
	}
}

func mt5UpdateTypeLabel(t pb.UpdateType) string {
	switch t {
	case pb.UpdateType_UpdateType_MarketOpen:
		return "open"
	case pb.UpdateType_UpdateType_MarketClose:
		return "close"
	case pb.UpdateType_UpdateType_PartialClose:
		return "close"
	case pb.UpdateType_UpdateType_PendingOpen:
		return "pending_open"
	case pb.UpdateType_UpdateType_PendingClose:
		return "pending_close"
	case pb.UpdateType_UpdateType_MarketModify:
		return "modify"
	case pb.UpdateType_UpdateType_PendingModify:
		return "modify"
	default:
		return "unknown"
	}
}

func mt5OrderTypeLabel(ot pb.OrderType) string {
	switch ot {
	case pb.OrderType_OrderType_Sell:
		return "sell"
	case pb.OrderType_OrderType_BuyLimit:
		return "buy_limit"
	case pb.OrderType_OrderType_SellLimit:
		return "sell_limit"
	case pb.OrderType_OrderType_BuyStop:
		return "buy_stop"
	case pb.OrderType_OrderType_SellStop:
		return "sell_stop"
	case pb.OrderType_OrderType_BuyStopLimit:
		return "buy_stop_limit"
	case pb.OrderType_OrderType_SellStopLimit:
		return "sell_stop_limit"
	default:
		return "buy"
	}
}

// GetPriceHistory implements backfiller.MTAPIBarSource via MT5 PriceHistory RPC.
func (g *Gateway) GetPriceHistory(ctx context.Context, accountID, symbolRaw, period string, from, to int64) ([]*mdtick.Bar, error) {
	g.mu.RLock()
	qhCli := g.qhCli
	sid := g.sessionID
	g.mu.RUnlock()

	if qhCli == nil || sid == "" {
		return nil, fmt.Errorf("mt5 GetPriceHistory: not connected")
	}

	tf := mt5PeriodToTimeframe(period)
	fromStr := time.UnixMilli(from).UTC().Format("2006-01-02T15:04:05")
	toStr := time.UnixMilli(to).UTC().Format("2006-01-02T15:04:05")

	resp, err := qhCli.PriceHistory(ctx, &pb.PriceHistoryRequest{
		Id: sid, Symbol: symbolRaw, From: fromStr, To: toStr, TimeFrame: tf,
	})
	if err != nil {
		return nil, fmt.Errorf("mt5 PriceHistory: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt5 PriceHistory: code=%d msg=%s",
			resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return convertMT5Bars(resp.GetResult(), accountID, period), nil
}

func mt5PeriodToTimeframe(period string) int32 {
	switch period {
	case "1m":
		return 1
	case "5m":
		return 5
	case "15m":
		return 15
	case "30m":
		return 30
	case "1h":
		return 60
	case "4h":
		return 240
	case "1d":
		return 1440
	default:
		return 60
	}
}

func convertMT5Bars(bars []*pb.Bar, accountID, period string) []*mdtick.Bar {
	pm := periodMs(period)
	var out []*mdtick.Bar
	for _, b := range bars {
		t := b.GetTime().AsTime()
		out = append(out, &mdtick.Bar{
			AccountID:     accountID,
			Period:        period,
			OpenTsUnixMs:  t.UnixMilli(),
			CloseTsUnixMs: t.UnixMilli() + pm,
			Open:          decimal.NewFromFloat(b.GetOpenPrice()),
			High:          decimal.NewFromFloat(b.GetHighPrice()),
			Low:           decimal.NewFromFloat(b.GetLowPrice()),
			Close:         decimal.NewFromFloat(b.GetClosePrice()),
			Volume:        float64(b.GetVolume()),
			TickCount:     uint32(b.GetTickVolume()),
		})
	}
	return out
}

func periodMs(period string) int64 {
	switch period {
	case "1m":
		return 60_000
	case "5m":
		return 300_000
	case "15m":
		return 900_000
	case "1h":
		return 3_600_000
	case "4h":
		return 14_400_000
	case "1d":
		return 86_400_000
	default:
		return 60_000
	}
}
