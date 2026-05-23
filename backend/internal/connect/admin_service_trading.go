package connect

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
)

func (s *AdminService) ListPositions(ctx context.Context, req *connect.Request[v1.ListPositionsRequest]) (*connect.Response[v1.ListPositionsResponse], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	page := int(req.Msg.Page)
	pageSize := int(req.Msg.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	result, err := s.adminSvc.ListPositions(ctx, "", req.Msg.AccountId, "", page, pageSize)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	positions := make([]*v1.Order, len(result.Data.([]*model.Position)))
	for i, p := range result.Data.([]*model.Position) {
		positions[i] = convertPositionToOrderProto(p)
	}

	return connect.NewResponse(&v1.ListPositionsResponse{
		Positions: positions,
		Total:     int32(result.Total),
	}), nil
}

func (s *AdminService) ListOrders(ctx context.Context, req *connect.Request[v1.ListOrdersRequest]) (*connect.Response[v1.ListOrdersResponse], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	page := int(req.Msg.Page)
	pageSize := int(req.Msg.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	result, err := s.adminSvc.ListOrders(ctx, "", req.Msg.AccountId, "", "", "", page, pageSize)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	orders := make([]*v1.Order, len(result.Data.([]*model.Order)))
	for i, o := range result.Data.([]*model.Order) {
		orders[i] = convertOrderToProto(o)
	}

	return connect.NewResponse(&v1.ListOrdersResponse{
		Orders: orders,
		Total:  int32(result.Total),
	}), nil
}

func (s *AdminService) GetTradingSummary(ctx context.Context, req *connect.Request[v1.GetTradingSummaryRequest]) (*connect.Response[v1.TradingSummary], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	summary, err := s.adminSvc.GetTradingSummary(ctx, req.Msg.StartDate, req.Msg.EndDate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(convertTradingSummaryToProto(summary)), nil
}

func convertPositionToOrderProto(p *model.Position) *v1.Order {
	return &v1.Order{
		Ticket:      p.Ticket,
		Symbol:      p.Symbol,
		Type:        getOrderTypeString(p.OrderType),
		Volume:      p.Volume,
		OpenPrice:   p.OpenPrice,
		StopLoss:    p.StopLoss,
		TakeProfit:  p.TakeProfit,
		Profit:      p.Profit,
		Swap:        p.Swap,
		Commission:  p.Commission,
		OpenTime:    timestamppb.New(p.OpenTime),
		Comment:     p.OrderComment,
		MagicNumber: int64(p.MagicNumber),
		AccountId:   p.MTAccountID.String(),
	}
}

func convertOrderToProto(o *model.Order) *v1.Order {
	var expiration *timestamppb.Timestamp
	if o.Expiration != nil {
		expiration = timestamppb.New(*o.Expiration)
	}
	return &v1.Order{
		Ticket:      o.Ticket,
		Symbol:      o.Symbol,
		Type:        getOrderTypeString(o.OrderType),
		Volume:      o.Volume,
		OpenPrice:   o.Price,
		StopLoss:    o.StopLoss,
		TakeProfit:  o.TakeProfit,
		OpenTime:    expiration,
		Comment:     o.OrderComment,
		MagicNumber: int64(o.MagicNumber),
		AccountId:   o.MTAccountID.String(),
	}
}

func getOrderTypeString(orderType int16) string {
	switch orderType {
	case 0, 1:
		return "buy"
	case 2, 3:
		return "sell"
	case 4, 5:
		return "buy_limit"
	case 6, 7:
		return "sell_limit"
	case 8, 9:
		return "buy_stop"
	case 10, 11:
		return "sell_stop"
	default:
		return "unknown"
	}
}

func convertTradingSummaryToProto(s *model.TradingSummary) *v1.TradingSummary {
	proto := &v1.TradingSummary{
		Overview: &v1.TradingOverview{
			TotalUsers:        int32(s.Overview.TotalUsers),
			ActiveUsers:       int32(s.Overview.ActiveUsers),
			TotalAccounts:     int32(s.Overview.TotalAccounts),
			ConnectedAccounts: int32(s.Overview.ConnectedAccounts),
		},
		Trading: &v1.TradingDetails{
			TotalOrders:   int32(s.Trading.TotalOrders),
			ClosedOrders:  int32(s.Trading.ClosedOrders),
			PendingOrders: int32(s.Trading.PendingOrders),
			TotalVolume:   s.Trading.TotalVolume,
			TotalProfit:   s.Trading.TotalProfit,
			TotalLoss:     s.Trading.TotalLoss,
			NetProfit:     s.Trading.NetProfit,
		},
	}

	proto.ByPlatform = make(map[string]*v1.PlatformStats)
	for k, v := range s.ByPlatform {
		proto.ByPlatform[k] = &v1.PlatformStats{
			Trades: int32(v.Orders),
			Profit: v.Volume,
		}
	}

	return proto
}

func (s *AdminService) ResolveAlert(ctx context.Context, req *connect.Request[v1.ResolveAlertRequest]) (*connect.Response[emptypb.Empty], error) {
	operatorID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.adminSvc.ResolveAlert(ctx, req.Msg.AlertId, operatorID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}
