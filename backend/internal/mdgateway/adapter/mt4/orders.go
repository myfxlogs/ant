package mt4

import (
	"context"
	"fmt"
	"time"

	pb "anttrader/mt4"
	"anttrader/internal/mthub"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

func mt4Op(side mthub.Side, ot mthub.OrderType) pb.Op {
	switch {
	case side == mthub.SideBuy && ot == mthub.OrderMarket:
		return pb.Op_Op_Buy
	case side == mthub.SideSell && ot == mthub.OrderMarket:
		return pb.Op_Op_Sell
	case side == mthub.SideBuy && ot == mthub.OrderLimit:
		return pb.Op_Op_BuyLimit
	case side == mthub.SideSell && ot == mthub.OrderLimit:
		return pb.Op_Op_SellLimit
	case side == mthub.SideBuy && ot == mthub.OrderStop:
		return pb.Op_Op_BuyStop
	case side == mthub.SideSell && ot == mthub.OrderStop:
		return pb.Op_Op_SellStop
	default:
		return pb.Op_Op_Buy
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
	op := mt4Op(req.Side, req.OrderType)
	price := req.Price.InexactFloat64()
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := tc.OrderSend(ctx, &pb.OrderSendRequest{
		Id: sid, Symbol: req.Canonical, Operation: op,
		Volume:     req.Volume.InexactFloat64(),
		Price:      price,
		Stoploss:   req.StopLoss.InexactFloat64(),
		Takeprofit: req.TakeProfit.InexactFloat64(),
	})
	if err != nil {
		return 0, fmt.Errorf("mt4 OrderSend: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return 0, fmt.Errorf("mt4 OrderSend: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	if resp.GetResult() == nil {
		return 0, fmt.Errorf("mt4 OrderSend: nil result")
	}
	return int64(resp.GetResult().GetTicket()), nil
}

func (g *Gateway) CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error {
	g.mu.RLock()
	tc := g.tradingCli
	sid := g.sessionID
	g.mu.RUnlock()
	if tc == nil || sid == "" {
		return fmt.Errorf("mt4 CloseOrder: not connected")
	}
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	l := lots.InexactFloat64()
	resp, err := tc.OrderClose(ctx, &pb.OrderCloseRequest{Id: sid, Ticket: int32(ticket), Lots: l})
	if err != nil {
		return fmt.Errorf("mt4 OrderClose: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return fmt.Errorf("mt4 OrderClose: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
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
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := tc.OrderModify(ctx, &pb.OrderModifyRequest{
		Id: sid, Ticket: int32(ticket),
		Stoploss: sl.InexactFloat64(), Takeprofit: tp.InexactFloat64(),
		Price: price.InexactFloat64(),
	})
	if err != nil {
		return fmt.Errorf("mt4 OrderModify: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return fmt.Errorf("mt4 OrderModify: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return nil
}

func (g *Gateway) FetchOpenedOrders(ctx context.Context) ([]*mthub.OrderRecord, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt4: not connected")
	}
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := client.OpenedOrders(ctx, &pb.OpenedOrdersRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt4 OpenedOrders: %w", err)
	}
	orders := resp.GetResult()
	out := make([]*mthub.OrderRecord, 0, len(orders))
	for _, o := range orders {
		side := mthub.SideBuy
		ot := mthub.OrderMarket
		switch o.GetType() {
		case pb.Op_Op_Sell:
			side = mthub.SideSell
		case pb.Op_Op_BuyLimit:
			ot = mthub.OrderLimit
		case pb.Op_Op_SellLimit:
			side = mthub.SideSell
			ot = mthub.OrderLimit
		case pb.Op_Op_BuyStop:
			ot = mthub.OrderStop
		case pb.Op_Op_SellStop:
			side = mthub.SideSell
			ot = mthub.OrderStop
		}
		out = append(out, &mthub.OrderRecord{
			Ticket:     int64(o.GetTicket()),
			SymbolRaw:  o.GetSymbol(),
			Canonical:  o.GetSymbol(),
			Side:       side,
			OrderType:  ot,
			Volume:     decimal.NewFromFloat(o.GetLots()),
			OpenPrice:  decimal.NewFromFloat(o.GetOpenPrice()),
			ClosePrice: decimal.NewFromFloat(o.GetClosePrice()),
			OpenTime:   o.GetOpenTime().AsTime(),
			CloseTime:  o.GetCloseTime().AsTime(),
			Profit:     decimal.NewFromFloat(o.GetProfit()),
			Swap:       decimal.NewFromFloat(o.GetSwap()),
			Commission: decimal.NewFromFloat(o.GetCommission()),
			Comment:    o.GetComment(),
			Magic:      o.GetMagicNumber(),
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
		return nil, fmt.Errorf("mt4 FetchOrderHistory: not connected")
	}
	fromStr := from.UTC().Format("2006-01-02T15:04:05")
	toStr := to.UTC().Format("2006-01-02T15:04:05")
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := client.OrderHistory(ctx, &pb.OrderHistoryRequest{Id: sid, From: fromStr, To: toStr})
	if err != nil {
		return nil, fmt.Errorf("mt4 OrderHistory: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt4 OrderHistory: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	orders := resp.GetResult()
	out := make([]*mthub.OrderRecord, 0, len(orders))
	for _, o := range orders {
		side := mthub.SideBuy
		ot := mthub.OrderMarket
		switch o.GetType() {
		case pb.Op_Op_Sell:
			side = mthub.SideSell
		case pb.Op_Op_BuyLimit:
			ot = mthub.OrderLimit
		case pb.Op_Op_SellLimit:
			side = mthub.SideSell
			ot = mthub.OrderLimit
		case pb.Op_Op_BuyStop:
			ot = mthub.OrderStop
		case pb.Op_Op_SellStop:
			side = mthub.SideSell
			ot = mthub.OrderStop
		}
		state := mthub.OrderStateClosed
		if o.GetCloseTime().GetSeconds() == 0 {
			state = mthub.OrderStateOpen
		}
		out = append(out, &mthub.OrderRecord{
			Ticket:     int64(o.GetTicket()),
			SymbolRaw:  o.GetSymbol(),
			Canonical:  o.GetSymbol(),
			Side:       side,
			OrderType:  ot,
			Volume:     decimal.NewFromFloat(o.GetLots()),
			OpenPrice:  decimal.NewFromFloat(o.GetOpenPrice()),
			ClosePrice: decimal.NewFromFloat(o.GetClosePrice()),
			OpenTime:   o.GetOpenTime().AsTime(),
			CloseTime:  o.GetCloseTime().AsTime(),
			Profit:     decimal.NewFromFloat(o.GetProfit()),
			Swap:       decimal.NewFromFloat(o.GetSwap()),
			Commission: decimal.NewFromFloat(o.GetCommission()),
			Comment:    o.GetComment(),
			Magic:      o.GetMagicNumber(),
			State:      state,
		})
	}
	return out, nil
}

func (g *Gateway) FetchSymbolParams(ctx context.Context, canonicals []string) ([]*mthub.SymbolParam, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt4 FetchSymbolParams: not connected")
	}
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	out := make([]*mthub.SymbolParam, 0, len(canonicals))
	for _, c := range canonicals {
		ctx2 := metadata.NewOutgoingContext(ctx, md)
		resp, err := client.SymbolParams(ctx2, &pb.SymbolParamsRequest{Id: sid, Symbol: c})
		if err != nil {
			return out, fmt.Errorf("mt4 SymbolParams(%s): %w", c, err)
		}
		if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
			return out, fmt.Errorf("mt4 SymbolParams(%s): code=%d msg=%s", c, resp.GetError().GetCode(), resp.GetError().GetMessage())
		}
		r := resp.GetResult()
		if r == nil {
			continue
		}
		si := r.GetSymbol()
		gp := r.GetGroupParams()
		param := &mthub.SymbolParam{
			Canonical:   c,
			SymbolRaw:   c,
			SpreadFloat: si.GetSpread() > 0,
		}
		if si != nil {
			param.Digits = si.GetDigits()
			param.StopLevel = si.GetStopsLevel()
			param.PointValue = decimal.NewFromFloat(si.GetPoint())
		}
		if gp != nil {
			param.LotMin = decimal.NewFromFloat(gp.GetMinLot())
			param.LotMax = decimal.NewFromFloat(gp.GetMaxLot())
			param.LotStep = decimal.NewFromFloat(gp.GetLotStep())
			param.TradeMode = gp.GetExecution()
		}
		// MT4 does not expose ContractSize (LotSize) via SymbolParams; default is 1.
		if param.LotSize.IsZero() {
			param.LotSize = decimal.NewFromInt(1)
		}
		out = append(out, param)
	}
	return out, nil
}

func (g *Gateway) SubscribeOrderEvents(ctx context.Context, h mthub.OrderEventHandler) error {
	g.mu.RLock()
	streamCli := g.streamCli
	sid := g.sessionID
	g.mu.RUnlock()
	if streamCli == nil || sid == "" {
		return fmt.Errorf("mt4 SubscribeOrderEvents: not connected")
	}
	md := metadata.New(map[string]string{"id": sid, "authorization": "Bearer " + g.token()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	stream, err := streamCli.OnOrderUpdate(ctx, &pb.OnOrderUpdateRequest{Id: sid})
	if err != nil {
		return fmt.Errorf("mt4 OnOrderUpdate: %w", err)
	}
	ctx, g.cancelOrderUpdateSub = context.WithCancel(ctx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				g.log.Error("mt4 order event recv panic", zap.Any("panic", r))
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
				g.log.Warn("mt4 order event recv error", zap.Error(err))
				return
			}
			if h == nil || msg.GetResult() == nil || msg.GetResult().GetUpdate() == nil {
				continue
			}
			upd := msg.GetResult().GetUpdate()
			o := upd.GetOrder()
			event := &mthub.OrderEvent{
				AccountID: g.cfg.AccountID,
				EventType: upd.GetAction().String(),
				Timestamp: time.Now(),
			}
			if o != nil {
				event.Ticket = int64(o.GetTicket())
			}
			h(event)
		}
	}()
	return nil
}
