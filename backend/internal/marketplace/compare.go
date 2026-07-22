package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

type StrategyComparison struct {
	StrategyID          string
	Title               string
	PublisherName       string
	PriceModel          string
	PriceAmount         string
	AssetClass          string
	Timeframe           string
	RiskLevel           string
	TotalSubscribers    int32
	AvgRating           float64
	RatingCount         int32
	BacktestTotalReturn string
	BacktestMaxDD       string
	BacktestSharpe      string
	BacktestWinRate     string
	BacktestTotalTrades int32
	LiveTotalReturn     string
	LiveMaxDD           string
	LiveSharpe          string
	LiveTotalTrades     int32
	TrackingSince       string
}

// CompareStrategies returns standardized comparison data for up to 4 strategies.
func (s *Service) CompareStrategies(ctx context.Context, strategyIDs []string) ([]StrategyComparison, error) {
	if len(strategyIDs) == 0 {
		return nil, fmt.Errorf("marketplace: no strategy IDs provided")
	}
	if len(strategyIDs) > 4 {
		strategyIDs = strategyIDs[:4]
	}

	var results []StrategyComparison
	for _, sid := range strategyIDs {
		if _, err := uuid.Parse(sid); err != nil {
			continue
		}
		c, err := s.compareOne(ctx, sid)
		if err != nil {
			continue
		}
		results = append(results, c)
	}
	return results, nil
}

func (s *Service) compareOne(ctx context.Context, strategyID string) (StrategyComparison, error) {
	var c StrategyComparison
	var snapshotRaw []byte
	err := s.pg.QueryRow(ctx,
		`SELECT ms.strategy_id::text, COALESCE(ms.title,''),
		        COALESCE(u.email, u.nickname, usp.user_id::text),
		        COALESCE(ms.price_model,''), COALESCE(ms.price_amount::text,'0'),
		        COALESCE(ms.asset_class,''), COALESCE(ms.timeframe,''), COALESCE(ms.risk_level,''),
		        COALESCE(ms.total_subscribers,0),
		        COALESCE(r.avg_rating,0), COALESCE(r.rating_count,0),
		        ms.backtest_snapshot
		 FROM marketplace_strategies ms
		 JOIN user_strategy_publishes usp ON usp.platform_strategy_id = ms.strategy_id
		 LEFT JOIN users u ON u.id = usp.user_id
		 LEFT JOIN (SELECT strategy_id, AVG(rating) AS avg_rating, COUNT(*)::int AS rating_count FROM marketplace_ratings GROUP BY strategy_id) r ON r.strategy_id = ms.strategy_id
		 WHERE ms.strategy_id = $1 AND ms.status = 'published'`,
		strategyID,
	).Scan(&c.StrategyID, &c.Title, &c.PublisherName, &c.PriceModel, &c.PriceAmount,
		&c.AssetClass, &c.Timeframe, &c.RiskLevel, &c.TotalSubscribers,
		&c.AvgRating, &c.RatingCount, &snapshotRaw)
	if err != nil {
		return c, err
	}

	// Parse backtest snapshot for metrics.
	if len(snapshotRaw) > 0 {
		var snap antv1.BacktestSnapshot
		if err := proto.Unmarshal(snapshotRaw, &snap); err == nil {
			c.BacktestTotalReturn = snap.TotalReturn
			c.BacktestMaxDD = snap.MaxDrawdown
			c.BacktestSharpe = snap.SharpeRatio
			c.BacktestWinRate = snap.WinRate
			c.BacktestTotalTrades = snap.TotalTrades
		}
	}

	// Get live performance summary if available.
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(total_return::text,'0'), COALESCE(max_drawdown::text,'0'),
		        COALESCE(sharpe_ratio::text,''), COALESCE(total_trades,0),
		        COALESCE(tracking_since::text,'')
		 FROM marketplace_live_performance_summary
		 WHERE strategy_id = $1`,
		strategyID,
	).Scan(&c.LiveTotalReturn, &c.LiveMaxDD, &c.LiveSharpe, &c.LiveTotalTrades, &c.TrackingSince)

	return c, nil
}
