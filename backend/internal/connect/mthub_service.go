//go:build ignore
// +build ignore

// Package connect — MtHubService ConnectRPC handler.
// Requires `make proto` to generate proto stubs.
// Remove the build ignore tag after running make proto.
// Bridges proto requests to mthub.MtHubService.
package connect

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	mthubv1 "anttrader/gen/proto/anttrader/mthub/v1"
	"anttrader/gen/proto/anttrader/mthub/v1/mthubv1connect"
	"anttrader/internal/mthub"
)

// MtHubService implements the generated ConnectRPC MtHubServiceHandler.
type MtHubService struct {
	svc *mthub.MtHubService
}

// NewMtHubService creates a ConnectRPC handler for the mthub service.
func NewMtHubService(svc *mthub.MtHubService) mthubv1connect.MtHubServiceHandler {
	return &MtHubService{svc: svc}
}

// EnsureSession finds or creates a session for the given account.
func (s *MtHubService) EnsureSession(ctx context.Context, req *connect.Request[mthubv1.EnsureSessionRequest]) (*connect.Response[mthubv1.EnsureSessionResponse], error) {
	result, err := s.svc.EnsureSession(ctx, req.Msg.AccountId)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("session not found"))
	}
	return connect.NewResponse(&mthubv1.EnsureSessionResponse{
		SessionId:     result.SessionID,
		AlreadyActive: result.AlreadyActive,
	}), nil
}

// CloseSession removes a session from the registry.
func (s *MtHubService) CloseSession(ctx context.Context, req *connect.Request[mthubv1.CloseSessionRequest]) (*connect.Response[mthubv1.CloseSessionResponse], error) {
	if err := s.svc.CloseSession(ctx, req.Msg.AccountId); err != nil {
		return nil, err
	}
	return connect.NewResponse(&mthubv1.CloseSessionResponse{}), nil
}

// OrderSend places a new order.
func (s *MtHubService) OrderSend(ctx context.Context, req *connect.Request[mthubv1.OrderSendRequest]) (*connect.Response[mthubv1.OrderSendResponse], error) {
	result, err := s.svc.OrderSend(ctx, req.Msg.AccountId, &mthub.OrderRequest{
		Symbol:  req.Msg.Symbol,
		Side:    req.Msg.Side,
		Volume:  req.Msg.Volume,
		Price:   req.Msg.Price,
		Sl:      req.Msg.Sl,
		Tp:      req.Msg.Tp,
		Comment: req.Msg.Comment,
		Type:    req.Msg.Type,
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&mthubv1.OrderSendResponse{
		Ticket: result.Ticket,
		Error:  result.Error,
	}), nil
}

// OrderClose closes an existing order.
func (s *MtHubService) OrderClose(ctx context.Context, req *connect.Request[mthubv1.OrderCloseRequest]) (*connect.Response[mthubv1.OrderCloseResponse], error) {
	result, err := s.svc.OrderClose(ctx, req.Msg.AccountId, &mthub.CloseRequest{
		Ticket: req.Msg.Ticket,
		Lots:   req.Msg.Lots,
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&mthubv1.OrderCloseResponse{Error: result.Error}), nil
}

// OrderHistory fetches closed orders.
func (s *MtHubService) OrderHistory(ctx context.Context, req *connect.Request[mthubv1.OrderHistoryRequest]) (*connect.Response[mthubv1.OrderHistoryResponse], error) {
	orders, err := s.svc.OrderHistory(ctx, req.Msg.AccountId, &mthub.HistoryRequest{
		From: req.Msg.From,
		To:   req.Msg.To,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*mthubv1.OrderRecord, len(orders))
	for i, o := range orders {
		out[i] = toProtoOrderRecord(o)
	}
	return connect.NewResponse(&mthubv1.OrderHistoryResponse{Orders: out}), nil
}

// OpenedOrders fetches currently open positions.
func (s *MtHubService) OpenedOrders(ctx context.Context, req *connect.Request[mthubv1.OpenedOrdersRequest]) (*connect.Response[mthubv1.OpenedOrdersResponse], error) {
	orders, err := s.svc.OpenedOrders(ctx, req.Msg.AccountId)
	if err != nil {
		return nil, err
	}
	out := make([]*mthubv1.OrderRecord, len(orders))
	for i, o := range orders {
		out[i] = toProtoOrderRecord(o)
	}
	return connect.NewResponse(&mthubv1.OpenedOrdersResponse{Orders: out}), nil
}

// SymbolParamsMany fetches contract parameters.
func (s *MtHubService) SymbolParamsMany(ctx context.Context, req *connect.Request[mthubv1.SymbolParamsManyRequest]) (*connect.Response[mthubv1.SymbolParamsManyResponse], error) {
	params, err := s.svc.SymbolParamsMany(ctx, req.Msg.AccountId, req.Msg.Symbols)
	if err != nil {
		return nil, err
	}
	out := make([]*mthubv1.SymbolParam, len(params))
	for i, p := range params {
		out[i] = &mthubv1.SymbolParam{
			Symbol:       p.Symbol,
			Digits:       p.Digits,
			Point:        p.Point,
			ContractSize: p.ContractSize,
			MinLot:       p.MinLot,
			MaxLot:       p.MaxLot,
			LotStep:      p.LotStep,
		}
	}
	return connect.NewResponse(&mthubv1.SymbolParamsManyResponse{Params: out}), nil
}

// PriceHistory fetches today's price bars.
func (s *MtHubService) PriceHistory(ctx context.Context, req *connect.Request[mthubv1.PriceHistoryRequest]) (*connect.Response[mthubv1.PriceHistoryResponse], error) {
	bars, err := s.svc.PriceHistory(ctx, req.Msg.AccountId, req.Msg.Symbol)
	if err != nil {
		return nil, err
	}
	out := make([]*mthubv1.PriceBar, len(bars))
	for i, b := range bars {
		out[i] = &mthubv1.PriceBar{
			OpenTsMs: b.OpenTsMs,
			Open:     b.Open,
			High:     b.High,
			Low:      b.Low,
			Close:    b.Close,
			Volume:   b.Volume,
		}
	}
	return connect.NewResponse(&mthubv1.PriceHistoryResponse{Bars: out}), nil
}

// SubscribeOrderEvents streams order events.
func (s *MtHubService) SubscribeOrderEvents(ctx context.Context, req *connect.Request[mthubv1.SubscribeOrderEventsRequest], stream *connect.ServerStream[mthubv1.OrderEvent]) error {
	return s.svc.SubscribeOrderEvents(ctx, req.Msg.AccountId, func(ev *mthub.OrderEvent) error {
		oe := &mthubv1.OrderEvent{
			AccountId: ev.AccountId,
			Type:      ev.Type,
		}
		if ev.Order != nil {
			oe.Order = toProtoOrderRecord(ev.Order)
		}
		return stream.Send(oe)
	})
}

func toProtoOrderRecord(o *mthub.OrderRecord) *mthubv1.OrderRecord {
	return &mthubv1.OrderRecord{
		Ticket:       o.Ticket,
		Symbol:       o.Symbol,
		Side:         o.Side,
		Lots:         o.Lots,
		OpenPrice:    o.OpenPrice,
		ClosePrice:   o.ClosePrice,
		Profit:       o.Profit,
		Swap:         o.Swap,
		Commission:   o.Commission,
		OpenTime:     o.OpenTime,
		CloseTime:    o.CloseTime,
		State:        o.State,
		OpenTimeMs:   o.OpenTimeMs,
		CurrentPrice: o.CurrentPrice,
	}
}
