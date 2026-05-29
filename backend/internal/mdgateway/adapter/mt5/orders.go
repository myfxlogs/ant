package mt5

import (
	"context"
	"fmt"
	"time"

	pb "anttrader/mt5"
	"anttrader/internal/mthub"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

// --- OrderExecutor interface (mthub) ---

func (g *Gateway) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		return 0, fmt.Errorf("mt5 PlaceOrder: not connected")
	}
	ot := mt5OrderType(req.Side, req.OrderType)
	price := req.Price.InexactFloat64()
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := tc.OrderSend(ctx, &pb.OrderSendRequest{
		Id: sid, Symbol: req.Canonical, Operation: ot,
		Volume:    req.Volume.InexactFloat64(),
		Price:     &price,
		Stoploss:  pfloat64(req.StopLoss),
		Takeprofit: pfloat64(req.TakeProfit),
		Comment:   &req.Comment,
		ExpertID:  pInt64(int64(req.Magic)),
	})
	if err != nil {
		return 0, fmt.Errorf("mt5 OrderSend: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return 0, fmt.Errorf("mt5 OrderSend: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	if resp.GetResult() == nil {
		return 0, fmt.Errorf("mt5 OrderSend: nil result")
	}
	return resp.GetResult().GetTicket(), nil
}

// openTimeFromOrder extracts the open time from an MT5 Order, falling back
// to OpenTimestampUTC when the proto Timestamp is nil or zero (some MT5
// brokers only populate the int64 field).
func openTimeFromOrder(o *pb.Order) time.Time {
	if t := o.GetOpenTime(); t != nil && t.GetSeconds() > 0 {
		return t.AsTime()
	}
	if ts := o.GetOpenTimestampUTC(); ts > 0 {
		return time.Unix(ts, 0)
	}
	return time.Time{}
}

// closeTimeFromOrder extracts the close time from an MT5 Order, falling back
// to CloseTimestampUTC when the proto Timestamp is nil or zero.
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
		return fmt.Errorf("mt5 CloseOrder: not connected")
	}
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	l := lots.InexactFloat64()
	resp, err := tc.OrderClose(ctx, &pb.OrderCloseRequest{Id: sid, Ticket: ticket, Lots: &l})
	if err != nil {
		return fmt.Errorf("mt5 OrderClose: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return fmt.Errorf("mt5 OrderClose: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
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
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := tc.OrderModify(ctx, &pb.OrderModifyRequest{
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

func (g *Gateway) FetchOpenedOrders(ctx context.Context) ([]*mthub.OrderRecord, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5: not connected")
	}
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := client.OpenedOrders(ctx, &pb.OpenedOrdersRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt5 OpenedOrders: %w", err)
	}
	orders := resp.GetResult()
	out := make([]*mthub.OrderRecord, 0, len(orders))
	for _, o := range orders {
		side := mthub.SideBuy
		ot := mthub.OrderMarket
		switch o.GetOrderType() {
		case pb.OrderType_OrderType_Sell:
			side = mthub.SideSell
		case pb.OrderType_OrderType_BuyLimit:
			ot = mthub.OrderLimit
		case pb.OrderType_OrderType_SellLimit:
			side = mthub.SideSell
			ot = mthub.OrderLimit
		case pb.OrderType_OrderType_BuyStop:
			ot = mthub.OrderStop
		case pb.OrderType_OrderType_SellStop:
			side = mthub.SideSell
			ot = mthub.OrderStop
		case pb.OrderType_OrderType_BuyStopLimit:
			ot = mthub.OrderStopLimit
		case pb.OrderType_OrderType_SellStopLimit:
			side = mthub.SideSell
			ot = mthub.OrderStopLimit
		}
		out = append(out, &mthub.OrderRecord{
			Ticket:     o.GetTicket(),
			SymbolRaw:  o.GetSymbol(),
			Canonical:  o.GetSymbol(),
			Side:       side,
			OrderType:  ot,
			Volume:     decimal.NewFromFloat(o.GetLots()),
			OpenPrice:  decimal.NewFromFloat(o.GetOpenPrice()),
			ClosePrice: decimal.NewFromFloat(o.GetClosePrice()),
			OpenTime:   openTimeFromOrder(o),
			CloseTime:  closeTimeFromOrder(o),
			Profit:     decimal.NewFromFloat(o.GetProfit()),
			Swap:       decimal.NewFromFloat(o.GetSwap()),
			Commission: decimal.NewFromFloat(o.GetCommission()),
			Comment:    o.GetComment(),
			Magic:      int32(o.GetExpertId()),
			State:      mthub.OrderStateOpen,
		})
	}
	return out, nil
}

func (g *Gateway) FetchOrderHistory(ctx context.Context, from, to time.Time) ([]*mthub.OrderRecord, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5 FetchOrderHistory: not connected")
	}
	fromStr := from.UTC().Format("2006-01-02T15:04:05")
	toStr := to.UTC().Format("2006-01-02T15:04:05")
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := client.OrderHistory(ctx, &pb.OrderHistoryRequest{Id: sid, From: fromStr, To: toStr})
	if err != nil {
		return nil, fmt.Errorf("mt5 OrderHistory: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt5 OrderHistory: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	orders := resp.GetResult()
	out := make([]*mthub.OrderRecord, 0, len(orders))
	for _, o := range orders {
		side, ot := mt5OrderTypeToSideAndOrderType(o.GetOrderType())
		state := mthub.OrderStateClosed
		if ct := o.GetCloseTime(); ct == nil || ct.GetSeconds() == 0 {
			state = mthub.OrderStateOpen
		}
		out = append(out, &mthub.OrderRecord{
			Ticket:     o.GetTicket(),
			SymbolRaw:  o.GetSymbol(),
			Canonical:  o.GetSymbol(),
			Side:       side,
			OrderType:  ot,
			Volume:     decimal.NewFromFloat(o.GetLots()),
			OpenPrice:  decimal.NewFromFloat(o.GetOpenPrice()),
			ClosePrice: decimal.NewFromFloat(o.GetClosePrice()),
			OpenTime:   openTimeFromOrder(o),
			CloseTime:  closeTimeFromOrder(o),
			Profit:     decimal.NewFromFloat(o.GetProfit()),
			Swap:       decimal.NewFromFloat(o.GetSwap()),
			Commission: decimal.NewFromFloat(o.GetCommission()),
			Comment:    o.GetComment(),
			Magic:      int32(o.GetExpertId()),
			State:      state,
		})
	}
	return out, nil
}

func mt5OrderTypeToSideAndOrderType(ot pb.OrderType) (mthub.Side, mthub.OrderType) {
	switch ot {
	case pb.OrderType_OrderType_Sell:
		return mthub.SideSell, mthub.OrderMarket
	case pb.OrderType_OrderType_BuyLimit:
		return mthub.SideBuy, mthub.OrderLimit
	case pb.OrderType_OrderType_SellLimit:
		return mthub.SideSell, mthub.OrderLimit
	case pb.OrderType_OrderType_BuyStop:
		return mthub.SideBuy, mthub.OrderStop
	case pb.OrderType_OrderType_SellStop:
		return mthub.SideSell, mthub.OrderStop
	case pb.OrderType_OrderType_BuyStopLimit:
		return mthub.SideBuy, mthub.OrderStopLimit
	case pb.OrderType_OrderType_SellStopLimit:
		return mthub.SideSell, mthub.OrderStopLimit
	default:
		return mthub.SideBuy, mthub.OrderMarket
	}
}

func (g *Gateway) FetchSymbolParams(ctx context.Context, canonicals []string) ([]*mthub.SymbolParam, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5 FetchSymbolParams: not connected")
	}
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	out := make([]*mthub.SymbolParam, 0, len(canonicals))
	for _, c := range canonicals {
		ctx2 := metadata.NewOutgoingContext(ctx, md)
		resp, err := client.SymbolParams(ctx2, &pb.SymbolParamsRequest{Id: sid, Symbol: c})
		if err != nil {
			return out, fmt.Errorf("mt5 SymbolParams(%s): %w", c, err)
		}
		if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
			return out, fmt.Errorf("mt5 SymbolParams(%s): code=%d msg=%s", c, resp.GetError().GetCode(), resp.GetError().GetMessage())
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

func (g *Gateway) SubscribeOrderEvents(ctx context.Context, h mthub.OrderEventHandler) error {
	g.mu.RLock()
	streamCli := g.streamCli
	sid := g.sessionID
	g.mu.RUnlock()
	if streamCli == nil || sid == "" {
		return fmt.Errorf("mt5 SubscribeOrderEvents: not connected")
	}
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	stream, err := streamCli.OnOrderUpdate(ctx, &pb.OnOrderUpdateRequest{Id: sid})
	if err != nil {
		return fmt.Errorf("mt5 OnOrderUpdate: %w", err)
	}
	ctx, g.cancelOrderUpdateSub = context.WithCancel(ctx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				g.log.Error("mt5 order event recv panic", zap.Any("panic", r))
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
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
