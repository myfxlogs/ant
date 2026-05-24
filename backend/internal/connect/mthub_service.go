package connect

import (
	"context"

	"connectrpc.com/connect"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/mthub"
)

// MtHubServer implements ant.v1.MtHubServiceHandler.
type MtHubServer struct {
	svc *mthub.MtHubService
}

var _ antv1c.MtHubServiceHandler = (*MtHubServer)(nil)

// NewMtHubServer creates a ConnectRPC server for mthub.
func NewMtHubServer(svc *mthub.MtHubService) *MtHubServer {
	return &MtHubServer{svc: svc}
}

func (s *MtHubServer) PlaceOrder(ctx context.Context, req *connect.Request[antv1.PlaceOrderRequest]) (*connect.Response[antv1.PlaceOrderResponse], error) {
	m := req.Msg
	vol, err := decimal.NewFromString(m.Volume)
	if err != nil { return nil, connect.NewError(connect.CodeInvalidArgument, err) }
	price, _ := decimal.NewFromString(m.Price)
	sl, _ := decimal.NewFromString(m.StopLoss)
	tp, _ := decimal.NewFromString(m.TakeProfit)

	rec, err := s.svc.PlaceOrder(ctx, &mthub.OrderRequest{
		AccountID: m.AccountId, Canonical: m.Canonical,
		Side: sideFromProto(m.Side), OrderType: orderTypeFromProto(m.OrderType),
		Volume: vol, Price: price, StopLoss: sl, TakeProfit: tp,
		Comment: m.Comment, ClientID: m.ClientId, Magic: m.Magic,
	})
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&antv1.PlaceOrderResponse{Ticket: rec.Ticket, Status: "submitted"}), nil
}

func (s *MtHubServer) CloseOrder(ctx context.Context, req *connect.Request[antv1.CloseOrderRequest]) (*connect.Response[antv1.CloseOrderResponse], error) {
	m := req.Msg
	lots := decimal.Zero
	if m.Lots != "" { lots, _ = decimal.NewFromString(m.Lots) }
	if err := s.svc.CloseOrder(ctx, m.AccountId, m.Ticket, lots); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.CloseOrderResponse{Status: "closed"}), nil
}

func (s *MtHubServer) OpenedOrders(ctx context.Context, req *connect.Request[antv1.OpenedOrdersRequest]) (*connect.Response[antv1.OpenedOrdersResponse], error) {
	list, err := s.svc.OpenedOrders(ctx, req.Msg.AccountId)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&antv1.OpenedOrdersResponse{Orders: toProtoOrders(list)}), nil
}

func (s *MtHubServer) OrderHistory(ctx context.Context, req *connect.Request[antv1.OrderHistoryRequest]) (*connect.Response[antv1.OrderHistoryResponse], error) {
	list, err := s.svc.OrderHistory(ctx, req.Msg.AccountId, req.Msg.From.AsTime(), req.Msg.To.AsTime())
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&antv1.OrderHistoryResponse{Orders: toProtoOrders(list)}), nil
}

func (s *MtHubServer) SymbolParams(ctx context.Context, req *connect.Request[antv1.SymbolParamsRequest]) (*connect.Response[antv1.SymbolParamsResponse], error) {
	list, err := s.svc.SymbolParams(ctx, req.Msg.AccountId, req.Msg.Canonicals)
	if err != nil { return nil, connect.NewError(connect.CodeInternal, err) }
	return connect.NewResponse(&antv1.SymbolParamsResponse{Params: toProtoParams(list)}), nil
}

func (s *MtHubServer) PriceHistory(ctx context.Context, req *connect.Request[antv1.PriceHistoryRequest]) (*connect.Response[antv1.PriceHistoryResponse], error) {
	return connect.NewResponse(&antv1.PriceHistoryResponse{}), nil
}

func (s *MtHubServer) GetAccountStatus(ctx context.Context, req *connect.Request[antv1.GetAccountStatusRequest]) (*connect.Response[antv1.AccountStatus], error) {
	return connect.NewResponse(&antv1.AccountStatus{
		AccountId: req.Msg.AccountId, State: "connected",
		LastTickAt: timestamppb.Now(),
	}), nil
}

func (s *MtHubServer) StreamOrderEvents(ctx context.Context, req *connect.Request[antv1.StreamOrderEventsRequest], stream *connect.ServerStream[antv1.OrderEvent]) error {
	userID := "default" // placeholder; extract from interceptor later
	ch, cancel := s.svc.SubscribeUserOrderEvents(ctx, userID)
	defer cancel()
	for {
		select {
		case <-ctx.Done(): return nil
		case ev, ok := <-ch:
			if !ok { return nil }
			if err := stream.Send(toProtoOrderEvent(ev)); err != nil { return err }
		}
	}
}

func sideFromProto(s antv1.Side) mthub.Side {
	if s == antv1.Side_SIDE_SELL { return mthub.SideSell }
	return mthub.SideBuy
}

func orderTypeFromProto(t antv1.OrderType) mthub.OrderType {
	switch t {
	case antv1.OrderType_ORDER_TYPE_LIMIT: return mthub.OrderLimit
	case antv1.OrderType_ORDER_TYPE_STOP: return mthub.OrderStop
	case antv1.OrderType_ORDER_TYPE_STOP_LIMIT: return mthub.OrderStopLimit
	default: return mthub.OrderMarket
	}
}

func toProtoOrders(list []*mthub.OrderRecord) []*antv1.OrderRecord {
	out := make([]*antv1.OrderRecord, 0, len(list))
	for _, r := range list {
		out = append(out, &antv1.OrderRecord{
			Ticket: r.Ticket, AccountId: r.AccountID,
			SymbolRaw: r.SymbolRaw, Canonical: r.Canonical,
			Volume: r.Volume.String(), OpenPrice: r.OpenPrice.String(),
			State: toProtoState(r.State),
		})
	}
	return out
}

func toProtoParams(list []*mthub.SymbolParam) []*antv1.SymbolParam {
	out := make([]*antv1.SymbolParam, 0, len(list))
	for _, p := range list {
		out = append(out, &antv1.SymbolParam{
			Canonical: p.Canonical, Digits: p.Digits,
			LotSize: p.LotSize.String(), LotMin: p.LotMin.String(),
			TradeMode: p.TradeMode, StopLevel: p.StopLevel,
		})
	}
	return out
}

func toProtoState(s mthub.OrderState) antv1.OrderState {
	switch s {
	case mthub.OrderStateOpen: return antv1.OrderState_ORDER_STATE_OPEN
	case mthub.OrderStateClosed: return antv1.OrderState_ORDER_STATE_CLOSED
	case mthub.OrderStateCancelled: return antv1.OrderState_ORDER_STATE_CANCELLED
	case mthub.OrderStateRejected: return antv1.OrderState_ORDER_STATE_REJECTED
	default: return antv1.OrderState_ORDER_STATE_PENDING
	}
}

func toProtoOrderEvent(ev *mthub.OrderEvent) *antv1.OrderEvent {
	return &antv1.OrderEvent{
		AccountId: ev.AccountID, Ticket: ev.Ticket,
		EventType: ev.EventType, Timestamp: timestamppb.New(ev.Timestamp),
		Order: &antv1.OrderRecord{Ticket: ev.Order.Ticket},
	}
}
