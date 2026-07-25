package marketplace

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func (s *MarketplaceServer) GetMarketplaceAnalytics(
	ctx context.Context,
	req *connect.Request[antv1.GetMarketplaceAnalyticsRequest],
) (*connect.Response[antv1.MarketplaceAnalytics], error) {
	if _, err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}

	result, err := s.svc.GetMarketplaceAnalytics(ctx, req.Msg.Period)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	daily := make([]*antv1.DailyAnalytics, 0, len(result.Daily))
	for _, d := range result.Daily {
		daily = append(daily, &antv1.DailyAnalytics{
			Date:           d.Date,
			Gmv:            d.GMV.String(),
			Transactions:   d.Transactions,
			NewSubscribers: d.NewSubscribers,
		})
	}

	return connect.NewResponse(&antv1.MarketplaceAnalytics{
		TotalGmv:        result.TotalGMV.String(),
		PlatformRevenue: result.PlatformRevenue.String(),
		ProviderRevenue: result.ProviderRevenue.String(),
		TotalTransactions: result.TotalTx,
		ActiveBuyers:    result.ActiveBuyers,
		NewSubscribers:  result.NewSubscribers,
		Arpu:            result.ARPU.String(),
		TotalStrategies: result.TotalStrategies,
		NewStrategies:   result.NewStrategies,
		RefundRate:      result.RefundRate.String(),
		Daily:           daily,
	}), nil
}

func (s *MarketplaceServer) GetTopStrategies(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[antv1.TopStrategiesResponse], error) {
	if _, err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}

	byRev, bySub, err := s.svc.GetTopStrategies(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	revItems := make([]*antv1.TopItem, 0, len(byRev))
	for _, item := range byRev {
		revItems = append(revItems, &antv1.TopItem{
			Id: item.ID, Name: item.Name, Value: item.Value.String(), Rank: item.Rank,
		})
	}
	subItems := make([]*antv1.TopItem, 0, len(bySub))
	for _, item := range bySub {
		subItems = append(subItems, &antv1.TopItem{
			Id: item.ID, Name: item.Name, Value: item.Value.String(), Rank: item.Rank,
		})
	}

	return connect.NewResponse(&antv1.TopStrategiesResponse{
		ByRevenue:     revItems,
		BySubscribers: subItems,
	}), nil
}

func (s *MarketplaceServer) GetTopProviders(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[antv1.TopProvidersResponse], error) {
	if _, err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}

	byRev, byStrat, err := s.svc.GetTopProviders(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	revItems := make([]*antv1.TopItem, 0, len(byRev))
	for _, item := range byRev {
		revItems = append(revItems, &antv1.TopItem{
			Id: item.ID, Name: item.Name, Value: item.Value.String(), Rank: item.Rank,
		})
	}
	stratItems := make([]*antv1.TopItem, 0, len(byStrat))
	for _, item := range byStrat {
		stratItems = append(stratItems, &antv1.TopItem{
			Id: item.ID, Name: item.Name, Value: item.Value.String(), Rank: item.Rank,
		})
	}

	return connect.NewResponse(&antv1.TopProvidersResponse{
		ByRevenue:    revItems,
		ByStrategies: stratItems,
	}), nil
}
