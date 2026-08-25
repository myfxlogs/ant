// profit_fetch.go — Fetch and publish logic extracted from profit.go.
package mt4

import (
	"context"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt4"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

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
