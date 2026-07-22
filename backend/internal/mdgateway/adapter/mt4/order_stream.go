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

func (g *Gateway) SubscribeOrderUpdate(ctx context.Context, handler mdtick.OrderUpdateHandler) error {
	g.mu.RLock()
	sc := g.streamCli
	g.mu.RUnlock()
	if sc == nil {
		return fmt.Errorf("mt4: not connected")
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

		md := metadata.New(map[string]string{"id": sid})
		if tok := g.token(); tok != "" {
			md.Set("authorization", "Bearer "+tok)
		}
		subCtx = metadata.NewOutgoingContext(subCtx, md)
		stream, err := sc.OnOrderUpdate(subCtx, &pb.OnOrderUpdateRequest{Id: sid})
		if err != nil {
			g.log.Warn("mt4 order update subscribe", zap.Error(err), zap.Duration("backoff", backoff))
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
		g.log.Info("mt4: order update stream active")
		for {
			resp, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt4 order update recv", zap.Error(err))
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
			s := resp.GetResult()
			if s == nil {
				continue
			}

			update := s.GetUpdate()
			var updateTicket int64
			var updateType string
			var updateSymbol string
			var updateVolume decimal.Decimal
			var updateOpenPrice decimal.Decimal
			var updateClosePrice decimal.Decimal
			var updateProfit decimal.Decimal
			var updateSwap decimal.Decimal
			var updateCommission decimal.Decimal
			var updateComment string
			var updateOpenTime int64
			var updateCloseTime int64
			var updateSL decimal.Decimal
			var updateTP decimal.Decimal
			var updateOrderType string
			if update != nil && update.GetOrder() != nil {
				o := update.GetOrder()
				updateTicket = int64(o.GetTicket())
				updateType = mt4UpdateActionLabel(update.GetAction())
				updateOrderType = mt4OrderOpLabel(o.GetType())
				updateSymbol = o.GetSymbol()
				updateVolume = decimal.NewFromFloat(o.GetLots())
				updateOpenPrice = decimal.NewFromFloat(o.GetOpenPrice())
				updateClosePrice = decimal.NewFromFloat(o.GetClosePrice())
				updateProfit = decimal.NewFromFloat(o.GetProfit())
				updateSwap = decimal.NewFromFloat(o.GetSwap())
				updateCommission = decimal.NewFromFloat(o.GetCommission())
				updateComment = o.GetComment()
				updateOpenTime = o.GetOpenTime().GetSeconds()
				updateCloseTime = o.GetCloseTime().GetSeconds()
				updateSL = decimal.NewFromFloat(o.GetStopLoss())
				updateTP = decimal.NewFromFloat(o.GetTakeProfit())
			}

			positions := make([]mdtick.OrderUpdatePosition, 0, len(s.GetOpenedOrders()))
			for _, o := range s.GetOpenedOrders() {
				positions = append(positions, mdtick.OrderUpdatePosition{
					Ticket:       int64(o.GetTicket()),
					Symbol:       o.GetSymbol(),
					Type:         mt4OrderOpLabel(o.GetType()),
					Volume:       decimal.NewFromFloat(o.GetLots()),
					OpenPrice:    decimal.NewFromFloat(o.GetOpenPrice()),
					CurrentPrice: decimal.NewFromFloat(o.GetClosePrice()),
					StopLoss:     decimal.NewFromFloat(o.GetStopLoss()),
					TakeProfit:   decimal.NewFromFloat(o.GetTakeProfit()),
					Profit:       decimal.NewFromFloat(o.GetProfit()),
					Swap:         decimal.NewFromFloat(o.GetSwap()),
					Commission:   decimal.NewFromFloat(o.GetCommission()),
					Comment:      o.GetComment(),
					OpenTime:     o.GetOpenTime().GetSeconds(),
				})
			}

			balance := s.GetBalance()
			// Use Decimal for financial arithmetic to avoid float64 rounding.
			equityD := decimal.NewFromFloat(s.GetEquity())
			balanceD := decimal.NewFromFloat(balance)
			profitD := equityD.Sub(balanceD)
			var profitPct float64
			if balance > 0 {
				profitPct = profitD.Div(balanceD).Mul(decimal.NewFromInt(100)).InexactFloat64()
			}
			handler(&mdtick.OrderUpdate{
				AccountID:        g.cfg.AccountID,
				Platform:         "mt4",
				UpdateTicket:     updateTicket,
				UpdateType:       updateType,
				UpdateOrderType:  updateOrderType,
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
				Balance:          balanceD,
				Credit:           decimal.NewFromFloat(s.GetCredit()),
				Equity:           equityD,
				Margin:           decimal.NewFromFloat(s.GetMargin()),
				FreeMargin:       decimal.NewFromFloat(s.GetFreeMargin()),
				MarginLevel:      decimal.NewFromFloat(s.GetMarginLevel()),
				Profit:           profitD,
				ProfitPercent:    profitPct,
				Positions:        positions,
			})
		}
	}
}

func mt4UpdateActionLabel(a pb.UpdateAction) string {
	switch a {
	case pb.UpdateAction_UpdateAction_PositionOpen:
		return "open"
	case pb.UpdateAction_UpdateAction_PositionClose:
		return "close"
	case pb.UpdateAction_UpdateAction_PositionModify:
		return "modify"
	case pb.UpdateAction_UpdateAction_PendingOpen:
		return "pending_open"
	case pb.UpdateAction_UpdateAction_PendingClose:
		return "pending_close"
	case pb.UpdateAction_UpdateAction_PendingModify:
		return "pending_modify"
	case pb.UpdateAction_UpdateAction_PendingFill:
		return "open"
	case pb.UpdateAction_UpdateAction_Balance:
		return "balance"
	case pb.UpdateAction_UpdateAction_Credit:
		return "credit"
	default:
		return "unknown"
	}
}

func mt4OrderOpLabel(op pb.Op) string {
	switch op {
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
	case pb.Op_Op_Balance:
		return "balance"
	case pb.Op_Op_Credit:
		return "credit"
	default:
		return "buy"
	}
}
