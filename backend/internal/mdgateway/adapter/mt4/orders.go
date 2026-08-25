package mt4

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mthub"
	pb "alphaforge/mt4"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

const orderTimeout = 30 * time.Second

func mt4Op(side mthub.Side, ot mthub.OrderType) (pb.Op, error) {
	switch {
	case side == mthub.SideBuy && ot == mthub.OrderMarket:
		return pb.Op_Op_Buy, nil
	case side == mthub.SideSell && ot == mthub.OrderMarket:
		return pb.Op_Op_Sell, nil
	case side == mthub.SideBuy && ot == mthub.OrderLimit:
		return pb.Op_Op_BuyLimit, nil
	case side == mthub.SideSell && ot == mthub.OrderLimit:
		return pb.Op_Op_SellLimit, nil
	case side == mthub.SideBuy && ot == mthub.OrderStop:
		return pb.Op_Op_BuyStop, nil
	case side == mthub.SideSell && ot == mthub.OrderStop:
		return pb.Op_Op_SellStop, nil
	default:
		return 0, fmt.Errorf("mt4 unsupported order type: side=%d orderType=%d", side, ot)
	}
}

func (g *Gateway) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		return 0, fmt.Errorf("mt4 PlaceOrder: not connected")
	}
	if g.breaker != nil && !g.breaker.Allow() {
		return 0, mthub.ErrCircuitOpen
	}
	op, err := mt4Op(req.Side, req.OrderType)
	if err != nil {
		return 0, fmt.Errorf("mt4 PlaceOrder: %w", err)
	}
	price := req.Price.InexactFloat64()
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	callCtx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	callCtx = metadata.NewOutgoingContext(callCtx, md)
	resp, err := tc.OrderSend(callCtx, &pb.OrderSendRequest{
		Id: sid, Symbol: req.Canonical, Operation: op,
		Volume:     req.Volume.InexactFloat64(),
		Price:      price,
		Stoploss:   req.StopLoss.InexactFloat64(),
		Takeprofit: req.TakeProfit.InexactFloat64(),
		Magic:      req.Magic,
		Slippage:   req.Deviation, // VM-TRADE-CONTEXT-7: EA deviation → MT4 slippage
	})
	if err != nil {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return 0, fmt.Errorf("mt4 OrderSend: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return 0, fmt.Errorf("%w: mt4 OrderSend: code=%d msg=%s", mthub.ErrBrokerRejected, resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	if resp.GetResult() == nil {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return 0, fmt.Errorf("mt4 OrderSend: nil result")
	}
	if g.breaker != nil {
		g.breaker.OnSuccess()
	}
	return int64(resp.GetResult().GetTicket()), nil
}

func (g *Gateway) CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		g.log.Warn("mt4 CloseOrder: not connected", zap.Bool("hasCli", tc != nil), zap.Bool("hasSid", sid != ""))
		return fmt.Errorf("mt4 CloseOrder: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	callCtx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	callCtx = metadata.NewOutgoingContext(callCtx, md)
	l := lots.InexactFloat64()
	g.log.Info("mt4 CloseOrder: sending", zap.Int64("ticket", ticket), zap.Float64("lots", l), zap.String("sid", truncSid(sid)))
	resp, err := tc.OrderClose(callCtx, &pb.OrderCloseRequest{Id: sid, Ticket: int32(ticket), Lots: l})
	if err != nil {
		g.log.Error("mt4 OrderClose: gRPC error", zap.Error(err))
		return fmt.Errorf("mt4 OrderClose: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		g.log.Error("mt4 OrderClose: broker error", zap.Int32("code", int32(resp.GetError().GetCode())), zap.String("msg", resp.GetError().GetMessage()))
		return fmt.Errorf("%w: mt4 OrderClose: code=%d msg=%s", mthub.ErrBrokerRejected, resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	g.log.Info("mt4 CloseOrder: success", zap.Int64("ticket", ticket))
	return nil
}

// DeleteOrder cancels a pending order using MT4 OrderDelete.
// MT4 has a dedicated OrderDelete RPC — OrderClose only works for open positions.
func (g *Gateway) DeleteOrder(ctx context.Context, ticket int64) error {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		g.log.Warn("mt4 DeleteOrder: not connected", zap.Bool("hasCli", tc != nil), zap.Bool("hasSid", sid != ""))
		return fmt.Errorf("mt4 DeleteOrder: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	callCtx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	callCtx = metadata.NewOutgoingContext(callCtx, md)
	g.log.Info("mt4 DeleteOrder: sending", zap.Int64("ticket", ticket), zap.String("sid", truncSid(sid)))
	resp, err := tc.OrderDelete(callCtx, &pb.OrderDeleteRequest{Id: sid, Ticket: int32(ticket)})
	if err != nil {
		g.log.Error("mt4 OrderDelete: gRPC error", zap.Error(err))
		return fmt.Errorf("mt4 OrderDelete: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		g.log.Error("mt4 OrderDelete: broker error", zap.Int32("code", int32(resp.GetError().GetCode())), zap.String("msg", resp.GetError().GetMessage()))
		return fmt.Errorf("%w: mt4 OrderDelete: code=%d msg=%s", mthub.ErrBrokerRejected, resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	g.log.Info("mt4 DeleteOrder: success", zap.Int64("ticket", ticket))
	return nil
}

func (g *Gateway) ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		return fmt.Errorf("mt4 ModifyOrder: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	callCtx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	callCtx = metadata.NewOutgoingContext(callCtx, md)
	resp, err := tc.OrderModify(callCtx, &pb.OrderModifyRequest{
		Id: sid, Ticket: int32(ticket),
		Stoploss: sl.InexactFloat64(), Takeprofit: tp.InexactFloat64(),
		Price: price.InexactFloat64(),
	})
	if err != nil {
		return fmt.Errorf("mt4 OrderModify: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return fmt.Errorf("%w: mt4 OrderModify: code=%d msg=%s", mthub.ErrBrokerRejected, resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return nil
}

func truncSid(s string) string {
	if len(s) > 8 {
		return s[:8] + "..."
	}
	return s
}
