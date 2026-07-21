package marketplace

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// ListLeaderboard returns ranked strategies by type and period (Phase 3.1).
func (s *MarketplaceServer) ListLeaderboard(
	ctx context.Context,
	req *connect.Request[antv1.ListLeaderboardRequest],
) (*connect.Response[antv1.ListLeaderboardResponse], error) {
	entries, err := s.svc.ListLeaderboard(ctx, req.Msg.Type, req.Msg.Period, req.Msg.AssetClass, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("leaderboard: %w", err))
	}

	out := make([]*antv1.LeaderboardEntry, 0, len(entries))
	for _, e := range entries {
		entry := &antv1.LeaderboardEntry{
			StrategyId:       e.StrategyID,
			PublishId:        e.PublishID,
			Title:            e.Title,
			PublisherName:    e.PublisherName,
			PublisherId:      e.PublisherID,
			PriceModel:       e.PriceModel,
			PriceAmount:      e.PriceAmount,
			AssetClass:       e.AssetClass,
			Timeframe:        e.Timeframe,
			RiskLevel:        e.RiskLevel,
			TotalSubscribers: e.TotalSubscribers,
			AvgRating:        e.AvgRating,
			RatingCount:      e.RatingCount,
			Rank:             e.Rank,
			TotalReturn:      e.TotalReturn,
			MaxDrawdown:      e.MaxDrawdown,
			SharpeRatio:      e.SharpeRatio,
			WinRate:          e.WinRate,
			TotalTrades:      e.TotalTrades,
			TrackingSince:    e.TrackingSince,
			PublishedAtMs:    e.PublishedAtMs,
		}
		if e.BacktestSnapshot != nil {
			entry.Backtest = e.BacktestSnapshot
		}
		out = append(out, entry)
	}

	return connect.NewResponse(&antv1.ListLeaderboardResponse{Entries: out}), nil
}
