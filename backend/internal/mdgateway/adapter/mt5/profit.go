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

// fetchAndPublish calls AccountSummary (canonical MQL5 values) and publishes
// via the handler. If AccountSummary fails, no financial snapshot is published;
// p contributes positions only.
// If p is nil (initial call or refreshAccountSummary), OpenedOrders is fetched
// to provide authoritative positions — without this, accounts with 0 positions
// never receive OnOrderUpdate events, so positionsReceivedAt goes stale and
// GetFreshTradingSnapshot blocks all strategy execution after 90s.
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

	// When p is nil (initial call or refreshAccountSummary), fetch OpenedOrders
	// to provide authoritative positions. Without this, accounts with 0 positions
	// never get positionsReceivedAt updated → snapshot goes stale after 90s →
	// GetFreshTradingSnapshot blocks all strategy execution.
	positionsAuth := p != nil
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
	} else {
		ooCtx, ooCancel := context.WithTimeout(ctx, 3*time.Second)
		ooMd := metadata.New(map[string]string{"id": sid})
		if tok := g.token(); tok != "" {
			ooMd.Set("authorization", "Bearer "+tok)
		}
		ooCtx = metadata.NewOutgoingContext(ooCtx, ooMd)
		orders, ooErr := client.OpenedOrders(ooCtx, &pb.OpenedOrdersRequest{Id: sid})
		ooCancel()
		if ooErr != nil {
			g.log.Warn("mt5 OpenedOrders failed during fetchAndPublish; positions not authoritative",
				zap.String("account_id", g.cfg.AccountID), zap.Error(ooErr))
		} else if orders.GetError() != nil && orders.GetError().GetCode() != 0 {
			g.log.Warn("mt5 OpenedOrders error during fetchAndPublish",
				zap.String("account_id", g.cfg.AccountID),
				zap.Int32("code", int32(orders.GetError().GetCode())),
				zap.String("msg", orders.GetError().GetMessage()))
		} else {
			positions = profitPositionsFromMt5OpenedOrders(orders.GetResult())
			positionsAuth = true
		}
	}

	handler(&mdtick.ProfitUpdate{
		AccountID:              g.cfg.AccountID,
		Platform:               "mt5",
		Balance:                balance,
		Credit:                 credit,
		Equity:                 equity,
		Margin:                 margin,
		FreeMargin:             freeMargin,
		MarginLevel:            marginLevel,
		Profit:                 profit,
		ProfitPercent:          profitPercent,
		Leverage:               int32(s.GetLeverage()),
		FinancialSource:        mdtick.FinancialsSourceAccountSummary,
		CapturedAt:             Clk.Now(),
		PositionsAuthoritative: positionsAuth,
		Positions:              positions,
	})
}

// profitPositionsFromMt5OpenedOrders converts MT5 OpenedOrders proto to ProfitPosition
// for use in fetchAndPublish when no stream frame is available.
func profitPositionsFromMt5OpenedOrders(orders []*pb.Order) []mdtick.ProfitPosition {
	if len(orders) == 0 {
		return nil
	}
	out := make([]mdtick.ProfitPosition, 0, len(orders))
	for _, o := range orders {
		var openTimeUnix int64
		if ot := o.GetOpenTime(); ot != nil {
			openTimeUnix = ot.AsTime().Unix()
		}
		out = append(out, mdtick.ProfitPosition{
			Ticket:       o.GetTicket(),
			Symbol:       o.GetSymbol(),
			Type:         mt5OrderTypeString(o.GetOrderType()),
			Magic:        int32(o.GetExpertId()),
			Volume:       decimal.NewFromFloat(o.GetLots()),
			OpenPrice:    decimal.NewFromFloat(o.GetOpenPrice()),
			CurrentPrice: decimal.Zero,
			StopLoss:     decimal.NewFromFloat(o.GetStopLoss()),
			TakeProfit:   decimal.NewFromFloat(o.GetTakeProfit()),
			Profit:       decimal.NewFromFloat(o.GetProfit()),
			Swap:         decimal.NewFromFloat(o.GetSwap()),
			Commission:   decimal.NewFromFloat(o.GetCommission()),
			Comment:      o.GetComment(),
			OpenTime:     openTimeUnix,
		})
	}
	return out
}

// mt5OrderTypeString converts a MT5 OrderType enum to a lowercase order type string.
func mt5OrderTypeString(ot pb.OrderType) string {
	switch ot {
	case pb.OrderType_OrderType_Buy:
		return "buy"
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
		return "unknown"
	}
}

const accountSummaryRefreshInterval = 45 * time.Second

func (g *Gateway) refreshAccountSummary(ctx context.Context, sid string, interval time.Duration, handler mdtick.ProfitHandler) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.fetchAndPublish(ctx, sid, nil, handler)
		}
	}
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
		go g.refreshAccountSummary(subCtx, sid, accountSummaryRefreshInterval, handler)

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
