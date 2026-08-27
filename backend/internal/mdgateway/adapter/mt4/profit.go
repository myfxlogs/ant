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

const accountSummaryRefreshInterval = 45 * time.Second

func (g *Gateway) refreshAccountSummary(ctx context.Context, sid string, interval time.Duration, handler mdtick.ProfitHandler) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.fetchAndPublish(ctx, sid, nil, func(pu *mdtick.ProfitUpdate) {
				g.mu.Lock()
				g.lastProfitUpdate = pu
				g.lastProfitAt = time.Now()
				g.mu.Unlock()
				handler(pu)
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

		// QUOTE-RECONNECT-LOOP S4: ensureConnected never returns an error.
		// The loop only exits on ctx.Done().
		_ = g.ensureConnected(ctx, &backoff, maxBackoff)
		if ctx.Err() != nil {
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

		// Publish initial account snapshot from AccountSummary before any stream
		// frames arrive. MT4's OnOrderProfit may be quiet for accounts with no
		// open positions, so without this the frontend shows stale data until the
		// first frame (which may never come).
		g.fetchAndPublish(ctx, sid, nil, func(pu *mdtick.ProfitUpdate) {
			g.mu.Lock()
			g.lastProfitUpdate = pu
			g.lastProfitAt = time.Now()
			g.mu.Unlock()
			handler(pu)
		})
		go g.refreshAccountSummary(subCtx, sid, accountSummaryRefreshInterval, handler)

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
				g.fetchAndPublish(ctx, sid, p, func(pu *mdtick.ProfitUpdate) {
					g.mu.Lock()
					g.lastProfitUpdate = pu
					g.lastProfitAt = time.Now()
					g.mu.Unlock()
					handler(pu)
				})
			}
		}
	profitLoopEnd:
	}
}

// fetchAndPublish calls AccountSummary for authoritative financial values.
// MT4's OnOrderProfit stream always returns margin=0, freeMargin=equity,
// marginLevel=0 — these fields are simply not filled by the mtapi MT4 adapter.
// AccountSummary returns the real values, so we use it as the single source
// of truth for all financial fields and only take positions from the stream
// frame. If AccountSummary fails, no financial snapshot is published.
// If p is nil (initial call or refreshAccountSummary), OpenedOrders is fetched
// to provide authoritative positions — without this, accounts with 0 positions
// never receive OnOrderUpdate events, so positionsReceivedAt goes stale and
// GetFreshTradingSnapshot blocks all strategy execution after 90s.
func (g *Gateway) fetchAndPublish(ctx context.Context, sid string, p *pb.ProfitUpdate, handler mdtick.ProfitHandler) {
	g.mu.RLock()
	client := g.client
	g.mu.RUnlock()
	if client == nil {
		g.log.Error("mt4 AccountSummary unavailable; financial snapshot rejected",
			zap.String("account_id", g.cfg.AccountID))
		return
	}

	asMd := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		asMd.Set("authorization", "Bearer "+tok)
	}
	sctx, scancel := context.WithTimeout(ctx, 3*time.Second)
	asCtx := metadata.NewOutgoingContext(sctx, asMd)
	acct, err := client.AccountSummary(asCtx, &pb.AccountSummaryRequest{Id: sid})
	scancel()
	if err != nil || acct == nil || acct.GetResult() == nil || (acct.GetError() != nil && acct.GetError().GetCode() != 0) {
		g.log.Error("mt4 AccountSummary failed; financial snapshot rejected",
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
	positions := parseMt4Positions(p)
	if p == nil {
		ooCtx, ooCancel := context.WithTimeout(ctx, 3*time.Second)
		ooMd := metadata.New(map[string]string{"id": sid})
		if tok := g.token(); tok != "" {
			ooMd.Set("authorization", "Bearer "+tok)
		}
		ooCtx = metadata.NewOutgoingContext(ooCtx, ooMd)
		orders, ooErr := client.OpenedOrders(ooCtx, &pb.OpenedOrdersRequest{Id: sid})
		ooCancel()
		if ooErr != nil {
			g.log.Warn("mt4 OpenedOrders failed during fetchAndPublish; positions not authoritative",
				zap.String("account_id", g.cfg.AccountID), zap.Error(ooErr))
		} else if orders.GetError() != nil && orders.GetError().GetCode() != 0 {
			g.log.Warn("mt4 OpenedOrders error during fetchAndPublish",
				zap.String("account_id", g.cfg.AccountID),
				zap.Int32("code", int32(orders.GetError().GetCode())),
				zap.String("msg", orders.GetError().GetMessage()))
		} else {
			positions = profitPositionsFromOpenedOrders(orders.GetResult())
			positionsAuth = true
		}
	}

	handler(&mdtick.ProfitUpdate{
		AccountID:              g.cfg.AccountID,
		Platform:               "mt4",
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

// profitPositionsFromOpenedOrders converts MT4 OpenedOrders proto to ProfitPosition
// for use in fetchAndPublish when no stream frame is available.
func profitPositionsFromOpenedOrders(orders []*pb.Order) []mdtick.ProfitPosition {
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
			Ticket:       int64(o.GetTicket()),
			Symbol:       o.GetSymbol(),
			Type:         mt4OrderTypeString(o.GetType()),
			Magic:        o.GetMagicNumber(),
			Volume:       decimal.NewFromFloat(o.GetLots()),
			OpenPrice:    decimal.NewFromFloat(o.GetOpenPrice()),
			CurrentPrice: decimal.Zero, // MT4 Order proto has no current price field
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

// mt4OrderTypeString converts a MT4 Op enum to a lowercase order type string.
func mt4OrderTypeString(op pb.Op) string {
	switch op {
	case pb.Op_Op_Buy:
		return "buy"
	case pb.Op_Op_Sell:
		return "sell"
	case pb.Op_Op_BuyLimit:
		return "buy_limit"
	case pb.Op_Op_SellLimit:
		return "sell_limit"
	case pb.Op_Op_BuyStop:
		return "buy_stop"
	case pb.Op_Op_SellStop:
		return "sell_stop"
	default:
		return "unknown"
	}
}

func parseMt4Positions(p *pb.ProfitUpdate) []mdtick.ProfitPosition {
	if p == nil {
		return nil
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
	return positions
}
