package marketplace

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// CompareStrategies returns side-by-side comparison data for up to 4 strategies (Phase 3.3).
func (s *MarketplaceServer) CompareStrategies(
	ctx context.Context,
	req *connect.Request[antv1.CompareStrategiesRequest],
) (*connect.Response[antv1.CompareStrategiesResponse], error) {
	if len(req.Msg.StrategyIds) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("strategy_ids is required"))
	}

	results, err := s.svc.CompareStrategies(ctx, req.Msg.StrategyIds)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("compare strategies: %w", err))
	}

	out := make([]*antv1.StrategyComparison, 0, len(results))
	for _, c := range results {
		out = append(out, &antv1.StrategyComparison{
			StrategyId:          c.StrategyID,
			Title:               c.Title,
			PublisherName:       c.PublisherName,
			PriceModel:          c.PriceModel,
			PriceAmount:         c.PriceAmount,
			AssetClass:          c.AssetClass,
			Timeframe:           c.Timeframe,
			RiskLevel:           c.RiskLevel,
			TotalSubscribers:    c.TotalSubscribers,
			AvgRating:           c.AvgRating,
			RatingCount:         c.RatingCount,
			BacktestTotalReturn: c.BacktestTotalReturn,
			BacktestMaxDrawdown: c.BacktestMaxDD,
			BacktestSharpeRatio: c.BacktestSharpe,
			BacktestWinRate:     c.BacktestWinRate,
			BacktestTotalTrades: c.BacktestTotalTrades,
			LiveTotalReturn:     c.LiveTotalReturn,
			LiveMaxDrawdown:     c.LiveMaxDD,
			LiveSharpeRatio:     c.LiveSharpe,
			LiveTotalTrades:     c.LiveTotalTrades,
			TrackingSince:       c.TrackingSince,
		})
	}

	return connect.NewResponse(&antv1.CompareStrategiesResponse{Strategies: out}), nil
}
