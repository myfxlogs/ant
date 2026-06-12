package mt5

import (
	"context"
	"fmt"
	"time"

	pb "anttrader/mt5"
	"anttrader/internal/mthub"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/metadata"
)

func (g *Gateway) FetchOpenedOrders(ctx context.Context) ([]*mthub.OrderRecord, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt5: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := client.OpenedOrders(ctx, &pb.OpenedOrdersRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt5 OpenedOrders: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt5 OpenedOrders: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
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
	if from.IsZero() || to.IsZero() {
		return nil, fmt.Errorf("mt5 FetchOrderHistory: from and to are required")
	}
	if from.After(to) {
		return nil, fmt.Errorf("mt5 FetchOrderHistory: from must be before to")
	}
	fromStr := from.UTC().Format("2006-01-02T15:04:05")
	toStr := to.UTC().Format("2006-01-02T15:04:05")
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
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
	case pb.OrderType_OrderType_SellStopLimit:
		return mthub.SideSell, mthub.OrderStopLimit
	case pb.OrderType_OrderType_Balance:
		return mthub.SideBuy, mthub.OrderBalance
	case pb.OrderType_OrderType_Credit:
		return mthub.SideBuy, mthub.OrderCredit
	default:
		return mthub.SideBuy, mthub.OrderMarket
	}
}
