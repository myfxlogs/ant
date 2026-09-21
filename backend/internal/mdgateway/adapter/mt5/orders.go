package mt5

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/mthub"
	pb "alphaforge/mt5"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const orderTimeout = 30 * time.Second

func (g *Gateway) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		return nil, fmt.Errorf("mt5 PlaceOrder: not connected")
	}
	if g.breaker != nil && !g.breaker.Allow() {
		return nil, mthub.ErrCircuitOpen
	}
	ot := mt5OrderType(req.Side, req.OrderType)
	price := req.Price.InexactFloat64()
	slippage := uint64(req.Deviation) // negative clamps to 0 (broker default)
	if req.Deviation < 0 {
		slippage = 0
	}
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
		Slippage:   &slippage,
		Stoploss:   pfloat64(req.StopLoss),
		Takeprofit: pfloat64(req.TakeProfit),
		Comment:    &req.Comment,
		ExpertID:   pInt64(int64(req.Magic)),
	})
	if err != nil {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return nil, fmt.Errorf("mt5 OrderSend: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return nil, &mthub.BrokerRejectError{Op: "mt5 OrderSend", Code: int32(resp.GetError().GetCode()), Message: resp.GetError().GetMessage()}
	}
	o := resp.GetResult()
	if o == nil {
		if g.breaker != nil {
			g.breaker.OnFailure()
		}
		return nil, fmt.Errorf("mt5 OrderSend: nil result")
	}
	if g.breaker != nil {
		g.breaker.OnSuccess()
	}
	rec := &mthub.OrderRecord{
		// Broker receipt verbatim; absent fields stay zero (= unknown).
		// mt5 pb Order has no magic field → rec.Magic stays 0 (= unknown).
		Ticket:     o.GetTicket(),
		OpenPrice:  decimal.NewFromFloat(o.GetOpenPrice()),
		Volume:     decimal.NewFromFloat(o.GetLots()),
		StopLoss:   decimal.NewFromFloat(o.GetStopLoss()),
		TakeProfit: decimal.NewFromFloat(o.GetTakeProfit()),
		Comment:    o.GetComment(),
		AccountID:  req.AccountID,
		Canonical:  req.Canonical,
		SymbolRaw:  req.Canonical,
	}
	if ot2 := o.GetOpenTime(); ot2 != nil {
		rec.OpenTime = ot2.AsTime()
	}
	// Side/OrderType come from the broker's stated order type, not the
	// request — zero values would inject a buy-market record downstream.
	rec.Side, rec.OrderType = mt5OrderTypeToSideAndOrderType(o.GetOrderType())
	// Same explicit derivation as mt4: zero State would misreport a market
	// fill as pending (OrderStatePending is the zero value).
	if o.GetOrderType() == pb.OrderType_OrderType_Buy || o.GetOrderType() == pb.OrderType_OrderType_Sell {
		rec.State = mthub.OrderStateOpen
	} else {
		rec.State = mthub.OrderStatePending
	}
	return rec, nil
}

// openTimeFromOrder extracts open time from MT5 Order, falling back to OpenTimestampUTC.
func openTimeFromOrder(o *pb.Order) time.Time {
	if t := o.GetOpenTime(); t != nil && t.GetSeconds() > 0 {
		return t.AsTime()
	}
	if ts := o.GetOpenTimestampUTC(); ts > 0 {
		return time.Unix(ts, 0).UTC()
	}
	return time.Time{}
}

// closeTimeFromOrder extracts close time from MT5 Order, falling back to CloseTimestampUTC.
func closeTimeFromOrder(o *pb.Order) time.Time {
	if t := o.GetCloseTime(); t != nil && t.GetSeconds() > 0 {
		return t.AsTime()
	}
	if ts := o.GetCloseTimestampUTC(); ts > 0 {
		return time.Unix(ts, 0).UTC()
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
		return &mthub.BrokerRejectError{Op: "mt5 OrderClose", Code: int32(resp.GetError().GetCode()), Message: resp.GetError().GetMessage()}
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
		return &mthub.BrokerRejectError{Op: "mt5 DeleteOrder", Code: int32(resp.GetError().GetCode()), Message: resp.GetError().GetMessage()}
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
		return &mthub.BrokerRejectError{Op: "mt5 OrderModify", Code: int32(resp.GetError().GetCode()), Message: resp.GetError().GetMessage()}
	}
	return nil
}

func (g *Gateway) SubscribeOrderEvents(ctx context.Context, h mthub.OrderEventHandler) error {
	g.mu.RLock()
	streamCli := g.streamCli
	sid := g.sessionID
	g.mu.RUnlock()
	if streamCli == nil || sid == "" {
		return fmt.Errorf("mt5 SubscribeOrderEvents: not connected")
	}
	g.mu.Lock()
	if g.cancelHubOrderSub != nil {
		g.cancelHubOrderSub()
	}
	ctx, g.cancelHubOrderSub = context.WithCancel(ctx)
	g.mu.Unlock()
	go g.orderEventLoop(ctx, h)
	return nil
}

func (g *Gateway) orderEventLoop(ctx context.Context, h mthub.OrderEventHandler) {
	defer func() {
		if r := recover(); r != nil {
			g.log.Error("mt5 order event recv panic", zap.Any("panic", r))
		}
	}()
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		g.mu.RLock()
		streamCli := g.streamCli
		sid := g.sessionID
		g.mu.RUnlock()
		if streamCli == nil || sid == "" {
			g.sleep(ctx, backoff)
			backoff = minDuration(backoff*2, streamMaxBackoff)
			continue
		}
		md := metadata.New(map[string]string{"id": sid})
		if tok := g.token(); tok != "" {
			md.Set("authorization", "Bearer "+tok)
		}
		subCtx, cancel := context.WithCancel(ctx)
		subCtx = metadata.NewOutgoingContext(subCtx, md)
		stream, err := streamCli.OnOrderUpdate(subCtx, &pb.OnOrderUpdateRequest{Id: sid})
		if err != nil {
			g.log.Warn("mt5 order event subscribe", zap.Error(err), zap.Duration("backoff", backoff))
			cancel()
			g.handleStreamError(ctx, err, &backoff)
			continue
		}
		backoff = time.Second
		g.recvOrderUpdates(ctx, cancel, stream, h, &backoff)
	}
}

func (g *Gateway) recvOrderUpdates(ctx context.Context, cancel context.CancelFunc,
	stream grpc.ServerStreamingClient[pb.OnOrderUpdateReply], h mthub.OrderEventHandler, backoff *time.Duration,
) {
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msg, err := stream.Recv()
		if err != nil {
			g.log.Warn("mt5 order event recv error", zap.Error(err))
			g.handleStreamError(ctx, err, backoff)
			return
		}
		if h == nil || msg.GetResult() == nil || msg.GetResult().GetUpdate() == nil {
			continue
		}
		upd := msg.GetResult().GetUpdate()
		o := upd.GetOrder()
		event := &mthub.OrderEvent{
			AccountID: g.cfg.AccountID,
			EventType: upd.GetType().String(),
			Timestamp: time.Now(),
		}
		if o != nil {
			event.Ticket = o.GetTicket()
		}
		h(event)
	}
}

func truncSid(s string) string {
	if len(s) > 8 {
		return s[:8] + "..."
	}
	return s
}
