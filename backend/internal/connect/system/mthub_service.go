package system

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

// MtHubServer implements ant.v1.MtHubServiceHandler.
type MtHubServer struct {
	svc          *mthub.MtHubService
	platform     *service.PlatformService
	marketData   *repository.MarketDataRepository
	tradeRecords *repository.TradeRecordRepository
	log          *zap.Logger
}

var _ antv1c.MtHubServiceHandler = (*MtHubServer)(nil)

func NewMtHubServer(svc *mthub.MtHubService, platform *service.PlatformService, marketData *repository.MarketDataRepository, tradeRecords *repository.TradeRecordRepository, log *zap.Logger) *MtHubServer {
	return &MtHubServer{svc: svc, platform: platform, marketData: marketData, tradeRecords: tradeRecords, log: log}
}

func (s *MtHubServer) PlaceOrder(ctx context.Context, req *connect.Request[antv1.PlaceOrderRequest]) (*connect.Response[antv1.PlaceOrderResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	m := req.Msg
	ok, err := s.platform.UserOwnsAccount(ctx, userID, m.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}
	vol, err := decimal.NewFromString(m.Volume)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// Default to decimal.Zero for empty Price/StopLoss/TakeProfit (proto3 default values).
	price, sl, tp := decimal.Zero, decimal.Zero, decimal.Zero
	if m.Price != "" {
		price, err = decimal.NewFromString(m.Price)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid price: %w", err))
		}
	}
	if m.StopLoss != "" {
		sl, err = decimal.NewFromString(m.StopLoss)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid stop_loss: %w", err))
		}
	}
	if m.TakeProfit != "" {
		tp, err = decimal.NewFromString(m.TakeProfit)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid take_profit: %w", err))
		}
	}

	rec, err := s.svc.PlaceOrder(ctx, &mthub.OrderRequest{
		AccountID: m.AccountId, Canonical: m.Canonical,
		Side: sideFromProto(m.Side), OrderType: orderTypeFromProto(m.OrderType),
		Volume: vol, Price: price, StopLoss: sl, TakeProfit: tp,
		Comment: m.Comment, ClientID: m.ClientId, Magic: m.Magic,
	})
	if err != nil {
		s.log.Error("PlaceOrder", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.PlaceOrderResponse{Ticket: rec.Ticket, Status: "submitted"}), nil
}

func (s *MtHubServer) CloseOrder(ctx context.Context, req *connect.Request[antv1.CloseOrderRequest]) (*connect.Response[antv1.CloseOrderResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	m := req.Msg
	ok, err := s.platform.UserOwnsAccount(ctx, userID, m.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}
	lots := decimal.Zero
	if m.Lots != "" {
		// #11: Check decimal parse error for lots.
		lots, err = decimal.NewFromString(m.Lots)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid lots: %w", err))
		}
	}
	if err := s.svc.CloseOrder(ctx, m.AccountId, m.Ticket, lots); err != nil {
		s.log.Error("CloseOrder", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.CloseOrderResponse{Status: "closed"}), nil
}

func (s *MtHubServer) OpenedOrders(ctx context.Context, req *connect.Request[antv1.OpenedOrdersRequest]) (*connect.Response[antv1.OpenedOrdersResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	ok, err := s.platform.UserOwnsAccount(ctx, userID, req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}
	list, err := s.svc.OpenedOrders(ctx, req.Msg.AccountId)
	if err != nil {
		s.log.Error("OpenedOrders", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.OpenedOrdersResponse{Orders: toProtoOrders(list)}), nil
}

func (s *MtHubServer) OrderHistory(ctx context.Context, req *connect.Request[antv1.OrderHistoryRequest]) (*connect.Response[antv1.OrderHistoryResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	ok, err := s.platform.UserOwnsAccount(ctx, userID, req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}
	list, err := s.svc.OrderHistory(ctx, req.Msg.AccountId, req.Msg.From.AsTime(), req.Msg.To.AsTime())
	if err != nil {
		s.log.Error("OrderHistory", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.OrderHistoryResponse{Orders: toProtoOrders(list)}), nil
}

func (s *MtHubServer) SymbolParams(ctx context.Context, req *connect.Request[antv1.SymbolParamsRequest]) (*connect.Response[antv1.SymbolParamsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	ok, err := s.platform.UserOwnsAccount(ctx, userID, req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}
	list, err := s.svc.SymbolParams(ctx, req.Msg.AccountId, req.Msg.Canonicals)
	if err != nil {
		s.log.Error("SymbolParams", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.SymbolParamsResponse{Params: toProtoParams(list)}), nil
}

func sideFromProto(s antv1.Side) mthub.Side {
	if s == antv1.Side_SIDE_SELL {
		return mthub.SideSell
	}
	return mthub.SideBuy
}

func orderTypeFromProto(t antv1.OrderType) mthub.OrderType {
	switch t {
	case antv1.OrderType_ORDER_TYPE_LIMIT:
		return mthub.OrderLimit
	case antv1.OrderType_ORDER_TYPE_STOP:
		return mthub.OrderStop
	case antv1.OrderType_ORDER_TYPE_STOP_LIMIT:
		return mthub.OrderStopLimit
	default:
		return mthub.OrderMarket
	}
}

func toProtoOrders(list []*mthub.OrderRecord) []*antv1.OrderRecord {
	out := make([]*antv1.OrderRecord, 0, len(list))
	for _, r := range list {
		out = append(out, &antv1.OrderRecord{
			Ticket: r.Ticket, AccountId: r.AccountID,
			SymbolRaw: r.SymbolRaw, Canonical: r.Canonical,
			Side: toProtoSide(r.Side), OrderType: toProtoOrderType(r.OrderType),
			Volume: r.Volume.String(), OpenPrice: r.OpenPrice.String(),
			ClosePrice: r.ClosePrice.String(), Profit: r.Profit.String(),
			Commission: r.Commission.String(), Swap: r.Swap.String(),
			OpenTime: timestamppb.New(r.OpenTime), CloseTime: timestamppb.New(r.CloseTime),
			Comment: r.Comment, Magic: r.Magic, State: toProtoState(r.State),
		})
	}
	return out
}

func toProtoSide(s mthub.Side) antv1.Side {
	if s == mthub.SideSell {
		return antv1.Side_SIDE_SELL
	}
	return antv1.Side_SIDE_BUY
}

func toProtoOrderType(t mthub.OrderType) antv1.OrderType {
	switch t {
	case mthub.OrderMarket:
		return antv1.OrderType_ORDER_TYPE_MARKET
	case mthub.OrderLimit:
		return antv1.OrderType_ORDER_TYPE_LIMIT
	case mthub.OrderStop:
		return antv1.OrderType_ORDER_TYPE_STOP
	case mthub.OrderStopLimit:
		return antv1.OrderType_ORDER_TYPE_STOP_LIMIT
	default:
		return antv1.OrderType_ORDER_TYPE_MARKET
	}
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
	case mthub.OrderStateOpen:
		return antv1.OrderState_ORDER_STATE_OPEN
	case mthub.OrderStateClosed:
		return antv1.OrderState_ORDER_STATE_CLOSED
	case mthub.OrderStateCancelled:
		return antv1.OrderState_ORDER_STATE_CANCELLED
	case mthub.OrderStateRejected:
		return antv1.OrderState_ORDER_STATE_REJECTED
	default:
		return antv1.OrderState_ORDER_STATE_PENDING
	}
}
