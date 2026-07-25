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
		return 0, fmt.Errorf("mt5 OrderSend: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
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
		return fmt.Errorf("mt5 OrderClose: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
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
		return fmt.Errorf("mt5 DeleteOrder: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
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
		return fmt.Errorf("mt5 OrderModify: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return nil
}

func (g *Gateway) FetchSymbolParams(ctx context.Context, canonicals []string) ([]*mthub.SymbolParam, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5 FetchSymbolParams: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	out := make([]*mthub.SymbolParam, 0, len(canonicals))
	for _, c := range canonicals {
		ctx2 := metadata.NewOutgoingContext(ctx, md)
		resp, err := client.SymbolParams(ctx2, &pb.SymbolParamsRequest{Id: sid, Symbol: c})
		if err != nil {
			return nil, fmt.Errorf("mt5 SymbolParams(%s): %w", c, err)
		}
		if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
			return nil, fmt.Errorf("mt5 SymbolParams(%s): code=%d msg=%s", c, resp.GetError().GetCode(), resp.GetError().GetMessage())
		}
		r := resp.GetResult()
		if r == nil {
			continue
		}
		si := r.GetSymbolInfo()
		sg := r.GetSymbolGroup()
		out = append(out, &mthub.SymbolParam{
			Canonical:   c,
			SymbolRaw:   c,
			Digits:      si.GetDigits(),
			TradeMode:   int32(sg.GetTradeMode()),
			StopLevel:   sg.GetSL(),
			PointValue:  decimal.NewFromFloat(si.GetTickValue()),
			LotSize:     decimal.NewFromFloat(si.GetContractSize()),
			LotStep:     decimal.NewFromFloat(sg.GetLotsStep()),
			LotMin:      decimal.NewFromFloat(sg.GetMinLots()),
			LotMax:      decimal.NewFromFloat(sg.GetMaxLots()),
			SpreadFloat: si.GetSpread() > 0,
		})
	}
	return out, nil
}

// FetchPriceHistory fetches K-line bars via the broker PriceHistory RPC in quotes.go.
func (g *Gateway) FetchPriceHistory(ctx context.Context, symbol, period string, from, to int64, count int) ([]*mthub.Bar, error) {
	bars, err := g.GetPriceHistory(ctx, "", symbol, period, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]*mthub.Bar, 0, len(bars))
	for _, b := range bars {
		out = append(out, &mthub.Bar{
			Time:   time.UnixMilli(b.OpenTsUnixMs),
			Open:   b.Open,
			High:   b.High,
			Low:    b.Low,
			Close:  b.Close,
			Volume: decimal.NewFromFloat(b.Volume),
		})
	}
	return out, nil
}

// FetchAllSymbols returns all available symbol names from the broker.
func (g *Gateway) FetchAllSymbols(ctx context.Context) ([]string, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5 FetchAllSymbols: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	ctx2 := metadata.NewOutgoingContext(ctx, md)
	resp, err := client.SymbolList(ctx2, &pb.SymbolListRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt5 SymbolList: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt5 SymbolList: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetResult(), nil
}

func (g *Gateway) SubscribeOrderEvents(ctx context.Context, h mthub.OrderEventHandler) error {
	g.mu.RLock()
	streamCli := g.streamCli
	sid := g.sessionID
	g.mu.RUnlock()
	if streamCli == nil || sid == "" {
		return fmt.Errorf("mt5 SubscribeOrderEvents: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	ctx = metadata.NewOutgoingContext(ctx, md)
	stream, err := streamCli.OnOrderUpdate(ctx, &pb.OnOrderUpdateRequest{Id: sid})
	if err != nil {
		return fmt.Errorf("mt5 OnOrderUpdate: %w", err)
	}
	g.mu.Lock()
	if g.cancelHubOrderSub != nil {
		g.cancelHubOrderSub()
	}
	ctx, g.cancelHubOrderSub = context.WithCancel(ctx)
	g.mu.Unlock()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				g.log.Error("mt5 order event recv panic", zap.Any("panic", r))
			}
		}()
		for {
			if ctx.Err() != nil {
				return
			}
			msg, err := stream.Recv()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				g.log.Warn("mt5 order event recv error", zap.Error(err))
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
	}()
	return nil
}

func truncSid(s string) string {
	if len(s) > 8 {
		return s[:8] + "..."
	}
	return s
}
