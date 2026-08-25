package mt5

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mthub"
	pb "alphaforge/mt5"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

const orderTimeout = 30 * time.Second

func (g *Gateway) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		return 0, fmt.Errorf("mt5 PlaceOrder: not connected")
	}
	if g.breaker != nil && !g.breaker.Allow() {
		return 0, mthub.ErrCircuitOpen
	}
	ot := mt5OrderType(req.Side, req.OrderType)
	price := req.Price.InexactFloat64()
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	callCtx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	callCtx = metadata.NewOutgoingContext(callCtx, md)
	resp, err := tc.OrderSend(callCtx, &pb.OrderSendRequest{
		Id: sid, Symbol: req.Canonical, Operation: ot,
		Volume:     req.Volume.InexactFloat64(),
		Price:      &price,
		Stoploss:   pfloat64(req.StopLoss),
		Takeprofit: pfloat64(req.TakeProfit),
		Comment:    &req.Comment,
		ExpertID:   pInt64(int64(req.Magic)),
		Slippage:   pUint64(uint64(req.Deviation)), // VM-TRADE-CONTEXT-7: EA deviation → MT5 slippage
	})
	if err != nil {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return 0, fmt.Errorf("mt5 OrderSend: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return 0, fmt.Errorf("%w: mt5 OrderSend: code=%d msg=%s", mthub.ErrBrokerRejected, resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	if resp.GetResult() == nil {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return 0, fmt.Errorf("mt5 OrderSend: nil result")
	}
	if g.breaker != nil {
		g.breaker.OnSuccess()
	}
	return resp.GetResult().GetTicket(), nil
}

// openTimeFromOrder extracts open time from MT5 Order, falling back to OpenTimestampUTC.
func openTimeFromOrder(o *pb.Order) time.Time {
	if t := o.GetOpenTime(); t != nil && t.GetSeconds() > 0 {
		return t.AsTime()
	}
	if ts := o.GetOpenTimestampUTC(); ts > 0 {
		return time.Unix(ts, 0)
	}
	return time.Time{}
}

// closeTimeFromOrder extracts close time from MT5 Order, falling back to CloseTimestampUTC.
func closeTimeFromOrder(o *pb.Order) time.Time {
	if t := o.GetCloseTime(); t != nil && t.GetSeconds() > 0 {
		return t.AsTime()
	}
	if ts := o.GetCloseTimestampUTC(); ts > 0 {
		return time.Unix(ts, 0)
	}
	return time.Time{}
}

func mt5OrderType(side mthub.Side, ot mthub.OrderType) pb.OrderType {
	switch {
	case side == mthub.SideBuy && ot == mthub.OrderMarket:
		return pb.OrderType_OrderType_Buy
	case side == mthub.SideSell && ot == mthub.OrderMarket:
		return pb.OrderType_OrderType_Sell
	case side == mthub.SideBuy && ot == mthub.OrderLimit:
		return pb.OrderType_OrderType_BuyLimit
	case side == mthub.SideSell && ot == mthub.OrderLimit:
		return pb.OrderType_OrderType_SellLimit
	case side == mthub.SideBuy && ot == mthub.OrderStop:
		return pb.OrderType_OrderType_BuyStop
	case side == mthub.SideSell && ot == mthub.OrderStop:
		return pb.OrderType_OrderType_SellStop
	case side == mthub.SideBuy && ot == mthub.OrderStopLimit:
		return pb.OrderType_OrderType_BuyStopLimit
	case side == mthub.SideSell && ot == mthub.OrderStopLimit:
		return pb.OrderType_OrderType_SellStopLimit
	default:
		return pb.OrderType_OrderType_Buy
	}
}

func pfloat64(d decimal.Decimal) *float64 {
	if d.IsZero() {
		return nil
	}
	v := d.InexactFloat64()
	return &v
}

func pInt64(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

// pUint64 returns a pointer to v, or nil if v == 0.
// VM-TRADE-CONTEXT-7: used for MT5 Slippage (Deviation) mapping.
func pUint64(v uint64) *uint64 {
	if v == 0 {
		return nil
	}
	return &v
}

func (g *Gateway) CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		g.log.Warn("mt5 CloseOrder: not connected", zap.Bool("hasCli", tc != nil), zap.Bool("hasSid", sid != ""))
		return fmt.Errorf("mt5 CloseOrder: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	callCtx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	callCtx = metadata.NewOutgoingContext(callCtx, md)
	l := lots.InexactFloat64()
	g.log.Info("mt5 CloseOrder: sending", zap.Int64("ticket", ticket), zap.Float64("lots", l), zap.String("sid", truncSid(sid)))
	resp, err := tc.OrderClose(callCtx, &pb.OrderCloseRequest{Id: sid, Ticket: ticket, Lots: &l})
	if err != nil {
		g.log.Error("mt5 OrderClose: gRPC error", zap.Error(err))
		return fmt.Errorf("mt5 OrderClose: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		g.log.Error("mt5 OrderClose: broker error", zap.Int32("code", int32(resp.GetError().GetCode())), zap.String("msg", resp.GetError().GetMessage()))
		return fmt.Errorf("%w: mt5 OrderClose: code=%d msg=%s", mthub.ErrBrokerRejected, resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	g.log.Info("mt5 CloseOrder: success", zap.Int64("ticket", ticket))
	return nil
}

// DeleteOrder cancels a pending order.
// MT5 has no dedicated OrderDelete RPC — OrderClose with lots=0 handles
// both pending order cancellation and position close on MT5.
func (g *Gateway) DeleteOrder(ctx context.Context, ticket int64) error {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		g.log.Warn("mt5 DeleteOrder: not connected", zap.Bool("hasCli", tc != nil), zap.Bool("hasSid", sid != ""))
		return fmt.Errorf("mt5 DeleteOrder: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	callCtx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	callCtx = metadata.NewOutgoingContext(callCtx, md)
	g.log.Info("mt5 DeleteOrder: sending", zap.Int64("ticket", ticket), zap.String("sid", truncSid(sid)))
	l := 0.0
	resp, err := tc.OrderClose(callCtx, &pb.OrderCloseRequest{Id: sid, Ticket: ticket, Lots: &l})
	if err != nil {
		g.log.Error("mt5 OrderClose (delete): gRPC error", zap.Error(err))
		return fmt.Errorf("mt5 DeleteOrder: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		g.log.Error("mt5 OrderClose (delete): broker error", zap.Int32("code", int32(resp.GetError().GetCode())), zap.String("msg", resp.GetError().GetMessage()))
		return fmt.Errorf("%w: mt5 DeleteOrder: code=%d msg=%s", mthub.ErrBrokerRejected, resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	g.log.Info("mt5 DeleteOrder: success", zap.Int64("ticket", ticket))
	return nil
}

func (g *Gateway) ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		return fmt.Errorf("mt5 ModifyOrder: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	callCtx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	callCtx = metadata.NewOutgoingContext(callCtx, md)
	resp, err := tc.OrderModify(callCtx, &pb.OrderModifyRequest{
		Id: sid, Ticket: ticket,
		Stoploss: sl.InexactFloat64(), Takeprofit: tp.InexactFloat64(),
	})
	if err != nil {
		return fmt.Errorf("mt5 OrderModify: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return fmt.Errorf("%w: mt5 OrderModify: code=%d msg=%s", mthub.ErrBrokerRejected, resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return nil
}

// Query and event operations moved to orders_queries.go
