package mt5

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt5"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

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
		if sc == nil {
			g.sleep(ctx, time.Second)
			continue
		}

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
			g.log.Warn("mt5 order update subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.handleStreamError(ctx, err, &backoff)
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
				g.handleStreamError(ctx, err, &backoff)
				break
			}
			s := resp.GetResult()
			if s == nil {
				continue
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						g.log.Error("mt5 order update handler panic", zap.Any("panic", r))
					}
				}()
				handler(parseMt5OrderUpdate(s, g.cfg.AccountID))
			}()
		}
	}
}

const streamMaxBackoff = 5 * time.Minute

func (g *Gateway) handleStreamError(ctx context.Context, err error, backoff *time.Duration) {
	// gRPC wraps context.Canceled as "rpc error: code = Canceled desc = context canceled".
	// When one stream's Disconnect cancels another stream's subCtx, the other stream
	// sees this wrapped error. We must NOT call Disconnect again — that would cascade
	// and tear down the shared connection for all streams, creating an infinite loop.
	if status.Code(err) == codes.Canceled {
		g.sleep(ctx, *backoff)
		*backoff = minDuration(*backoff*2, streamMaxBackoff)
		return
	}
	if err != context.Canceled && err != context.DeadlineExceeded {
		g.reportStatus("reconnecting", err.Error())
		_ = g.Disconnect(ctx)
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
	}
	g.sleep(ctx, *backoff)
	*backoff = minDuration(*backoff*2, streamMaxBackoff)
}

func parseMt5OrderUpdate(s *pb.OrderUpdateSummary, accountID string) *mdtick.OrderUpdate {
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

	update := s.GetUpdate()
	if update != nil && update.GetOrder() != nil {
		o := update.GetOrder()
		updateTicket = o.GetTicket()
		updateType = mt5UpdateTypeLabel(update.GetType())
		updateOrderType = mt5OrderTypeLabel(o.GetOrderType())
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
			Ticket:       o.GetTicket(),
			Symbol:       o.GetSymbol(),
			Type:         mt5OrderTypeLabel(o.GetOrderType()),
			Magic:        int32(o.GetExpertId()),
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
	equityD := decimal.NewFromFloat(s.GetEquity())
	balanceD := decimal.NewFromFloat(balance)
	profitD := equityD.Sub(balanceD)
	var profitPct float64
	if balance > 0 {
		profitPct = profitD.Div(balanceD).Mul(decimal.NewFromInt(100)).InexactFloat64()
	}
	return &mdtick.OrderUpdate{
		AccountID:        accountID,
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
		Balance:          balanceD,
		Credit:           decimal.Zero,
		Equity:           equityD,
		Margin:           decimal.NewFromFloat(s.GetMargin()),
		FreeMargin:       decimal.NewFromFloat(s.GetFreeMargin()),
		MarginLevel:      decimal.NewFromFloat(s.GetMarginLevel()),
		Profit:           profitD,
		ProfitPercent:    profitPct,
		Positions:        positions,
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
