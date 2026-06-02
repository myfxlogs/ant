package mt4

import (
	"context"
	"fmt"
	"time"

	pb "anttrader/mt4"
	"anttrader/internal/mdgateway/adapter/mdtick"
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
	// #nosec G118 — orderUpdateRecvLoop runs for the gateway's full connection lifetime
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
			g.log.Warn("mt4 order update subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		g.log.Info("mt4: order update stream active")
		for {
			resp, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt4 order update recv", zap.Error(err))
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
			var updateOrderType string
			if update != nil && update.GetOrder() != nil {
				o := update.GetOrder()
				updateTicket = int64(o.GetTicket())
				updateType = mt4UpdateActionLabel(update.GetAction())
				updateOrderType = mt4OrderOpLabel(pb.Op(o.GetType()))
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

			positions := make([]mdtick.OrderUpdatePosition, 0, len(s.GetOpenedOrders()))
			for _, o := range s.GetOpenedOrders() {
				positions = append(positions, mdtick.OrderUpdatePosition{
					Ticket:       int64(o.GetTicket()),
					Symbol:       o.GetSymbol(),
					Type:         mt4OrderOpLabel(o.GetType()),
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

			balance := s.GetBalance()
			profit := s.GetEquity() - balance
			var profitPct float64
			if balance > 0 {
				profitPct = (profit / balance) * 100
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
				Balance:          balance,
				Credit:           s.GetCredit(),
				Equity:           s.GetEquity(),
				Margin:           s.GetMargin(),
				FreeMargin:       s.GetFreeMargin(),
				MarginLevel:      s.GetMarginLevel(),
				Profit:           profit,
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
	default:
		return "buy"
	}
}
