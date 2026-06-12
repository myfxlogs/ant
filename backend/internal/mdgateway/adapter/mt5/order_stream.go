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

func (g *Gateway) SubscribeOrderUpdate(ctx context.Context, handler mdtick.OrderUpdateHandler) error {
	g.mu.RLock()
	sc := g.streamCli
	g.mu.RUnlock()
	if sc == nil {
		return fmt.Errorf("mt5: not connected")
	}
	// #nosec G118 — orderUpdateRecvLoop runs for the gateway full connection lifetime
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
			// Force disconnect so ensureConnected will do a fresh Connect
			// with a new session on the next iteration.
			// Skip on context cancellation — normal teardown, not a stream error.
			if err != context.Canceled && err != context.DeadlineExceeded {
				g.reportStatus("reconnecting", err.Error())
				g.Disconnect(ctx)
			}
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		g.reportStatus("connected", "")
		g.log.Info("mt5: order update stream active")
		for {
			resp, err := stream.Recv()
			if err != nil {
				g.log.Warn("mt5 order update recv", zap.Error(err))
				cancel()
				// Force disconnect so ensureConnected will do a fresh Connect
				// with a new session on the next iteration.
				// Skip on context cancellation — normal teardown, not a stream error.
				if err != context.Canceled && err != context.DeadlineExceeded {
					g.reportStatus("reconnecting", err.Error())
					g.Disconnect(ctx)
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
				updateTicket = o.GetTicket()
				updateType = mt5UpdateTypeLabel(update.GetType())
				updateOrderType = mt5OrderTypeLabel(o.GetOrderType())
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
				Platform:         "mt5",
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
				Credit:           0, // MT5 OrderUpdateSummary lacks GetCredit; credit arrives via OnAccountProfit stream
				Equity:           s.GetEquity(),
				Margin:           s.GetMargin(),
				FreeMargin:       s.GetFreeMargin(),
				MarginLevel:      s.GetMarginLevel(),
				Profit:           profitD.InexactFloat64(),
				ProfitPercent:    profitPct,
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
	case pb.UpdateType_UpdateType_Balance:
		return "balance"
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
	case pb.OrderType_OrderType_Balance:
		return "balance"
	case pb.OrderType_OrderType_Credit:
		return "credit"
	default:
		return "buy"
	}
}
