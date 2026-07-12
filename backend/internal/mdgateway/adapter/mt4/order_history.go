package mt4

import (
	"context"
	"fmt"
	"time"

	pb "alphaforge/mt4"
	"alphaforge/internal/mthub"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/metadata"
)

func (g *Gateway) FetchOpenedOrders(ctx context.Context) ([]*mthub.OrderRecord, error) {
	g.mu.RLock()
	client := g.client
	sid := g.sessionID
	g.mu.RUnlock()
	if client == nil || sid == "" {
		return nil, fmt.Errorf("mt4: not connected")
	}
	md := metadata.New(map[string]string{"id": sid})
	if tok := g.token(); tok != "" {
		md.Set("authorization", "Bearer "+tok)
	}
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := client.OpenedOrders(ctx, &pb.OpenedOrdersRequest{Id: sid})
	if err != nil {
		return nil, fmt.Errorf("mt4 OpenedOrders: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != 0 {
		return nil, fmt.Errorf("mt4 OpenedOrders: code=%d msg=%s", resp.GetError().GetCode(), resp.GetError().GetMessage())
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
		case pb.Op_Op_Balance:
			ot = mthub.OrderBalance
		case pb.Op_Op_Credit:
			ot = mthub.OrderCredit
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
			Commission:  decimal.NewFromFloat(o.GetCommission()),
			StopLoss:    decimal.NewFromFloat(o.GetStopLoss()),
			TakeProfit:  decimal.NewFromFloat(o.GetTakeProfit()),
			Comment:     o.GetComment(),
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
	if from.IsZero() || to.IsZero() {
		return nil, fmt.Errorf("mt4 FetchOrderHistory: from and to are required")
	}
	if from.After(to) {
		return nil, fmt.Errorf("mt4 FetchOrderHistory: from must be before to")
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
		case pb.Op_Op_Balance:
			ot = mthub.OrderBalance
		case pb.Op_Op_Credit:
			ot = mthub.OrderCredit
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
			Commission:  decimal.NewFromFloat(o.GetCommission()),
			StopLoss:    decimal.NewFromFloat(o.GetStopLoss()),
			TakeProfit:  decimal.NewFromFloat(o.GetTakeProfit()),
			Comment:     o.GetComment(),
			Magic:      o.GetMagicNumber(),
			State:      state,
		})
	}
	return out, nil
}
