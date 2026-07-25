package marketplace

import (
	"context"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func (s *MarketplaceServer) AdminListStrategies(
	ctx context.Context,
	req *connect.Request[antv1.AdminListStrategiesRequest],
) (*connect.Response[antv1.AdminListStrategiesResponse], error) {
	if _, err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}

	m := req.Msg
	rows, total, err := s.svc.AdminListStrategies(ctx, m.Status, m.Keyword, int(m.Limit), int(m.Offset))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*antv1.AdminStrategyItem, 0, len(rows))
	for _, r := range rows {
		item := &antv1.AdminStrategyItem{
			Status:           r.Status,
			TotalSales:       r.TotalSales,
			TotalRevenue:     r.TotalRevenue.String(),
			PlatformRevenue:  r.PlatformRevenue.String(),
			IsFeatured:       r.IsFeatured,
			FeaturedPriority: r.FeaturedPriority,
			Strategy: &antv1.PublishedStrategy{
				StrategyId:   r.StrategyID,
				Title:        r.Title,
				Description:  r.Description,
				PriceModel:   r.PriceModel,
				PriceAmount:  r.PriceAmount.String(),
				AssetClass:   r.AssetClass,
				PublisherUserId: r.PublisherID,
				StrategyName: r.PublisherName,
			},
		}
		if r.LastSaleAt != nil {
			item.LastSaleAt = r.LastSaleAt.Format("2006-01-02T15:04:05Z")
		}
		items = append(items, item)
	}

	return connect.NewResponse(&antv1.AdminListStrategiesResponse{
		Strategies: items,
		Total:      int32(total),
	}), nil
}

func (s *MarketplaceServer) AdminFeatureStrategy(
	ctx context.Context,
	req *connect.Request[antv1.AdminFeatureStrategyRequest],
) (*connect.Response[antv1.AdminFeatureStrategyResponse], error) {
	if _, err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}

	m := req.Msg
	if err := s.svc.AdminFeatureStrategy(ctx, m.StrategyId, m.Featured, m.Priority); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.AdminFeatureStrategyResponse{
		Success: true,
	}), nil
}
