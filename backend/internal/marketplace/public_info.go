package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// GetStrategyPublicInfo returns public strategy data for the share landing page.
// No authentication required — only published strategies with status='published' are returned.
// Code snippet is truncated to first 500 chars for preview.
func (s *Service) GetStrategyPublicInfo(ctx context.Context, strategyID string) (*antv1.GetStrategyPublicInfoResponse, error) {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}

	var (
		title, description, publisherName, priceModel, priceAmount,
		assetClass, timeframe, riskLevel, codeSnippet string
		totalSubscribers int32
		avgRating        float64
		ratingCount      int32
		backtestProto    []byte
	)
	err = s.pg.QueryRow(ctx,
		`SELECT ms.title, COALESCE(ms.description,''), COALESCE(u.nickname, 'Publisher'),
		        ms.price_model, COALESCE(ms.price_amount::text,'0'),
		        COALESCE(ms.asset_class,''), COALESCE(ms.timeframe,''),
		        COALESCE(ms.risk_level,''), COALESCE(ms.code_snippet,''),
		        COALESCE(ms.total_subscribers,0), COALESCE(ms.avg_rating,0),
		        COALESCE(ms.rating_count,0), ms.backtest_snapshot
		 FROM marketplace_strategies ms
		 LEFT JOIN users u ON u.id = ms.publisher_id
		 WHERE ms.strategy_id = $1 AND ms.status = 'published'`,
		sid,
	).Scan(&title, &description, &publisherName, &priceModel, &priceAmount,
		&assetClass, &timeframe, &riskLevel, &codeSnippet,
		&totalSubscribers, &avgRating, &ratingCount, &backtestProto)
	if err != nil {
		return nil, fmt.Errorf("marketplace: strategy not found: %w", err)
	}

	// Truncate code snippet for public preview.
	if len(codeSnippet) > 500 {
		codeSnippet = codeSnippet[:500] + "\n// ..."
	}

	resp := &antv1.GetStrategyPublicInfoResponse{
		StrategyId:       strategyID,
		Title:            title,
		Description:      description,
		PublisherName:    publisherName,
		PriceModel:       priceModel,
		PriceAmount:      priceAmount,
		AssetClass:       assetClass,
		Timeframe:        timeframe,
		RiskLevel:        riskLevel,
		TotalSubscribers: totalSubscribers,
		AvgRating:        avgRating,
		RatingCount:      ratingCount,
		CodeSnippet:      codeSnippet,
	}

	// Parse backtest snapshot.
	if len(backtestProto) > 0 {
		snap := &antv1.BacktestSnapshot{}
		if err := proto.Unmarshal(backtestProto, snap); err == nil {
			resp.Backtest = snap
		}
	}

	// Fetch live performance summary.
	var liveReturn, liveDD, liveSharpe, trackingSince string
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(total_return::text,''), COALESCE(max_drawdown::text,''),
		        COALESCE(sharpe_ratio::text,''), COALESCE(TO_CHAR(tracking_since, 'YYYY-MM-DD'),'')
		 FROM marketplace_live_performance_summary
		 WHERE strategy_id = $1`,
		sid,
	).Scan(&liveReturn, &liveDD, &liveSharpe, &trackingSince)
	resp.LiveTotalReturn = liveReturn
	resp.LiveMaxDrawdown = liveDD
	resp.LiveSharpeRatio = liveSharpe
	resp.TrackingSince = trackingSince

	return resp, nil
}
