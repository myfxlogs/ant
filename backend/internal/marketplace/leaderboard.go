package marketplace

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

type LeaderboardEntry struct {
	StrategyID       string
	PublishID        string
	Title            string
	PublisherName    string
	PublisherID      string
	PriceModel       string
	PriceAmount      string
	AssetClass       string
	Timeframe        string
	RiskLevel        string
	TotalSubscribers int32
	AvgRating        float64
	RatingCount      int32
	Rank             int32
	// Return leaderboard
	TotalReturn   string
	MaxDrawdown   string
	SharpeRatio   string
	WinRate       string
	TotalTrades   int32
	TrackingSince string
	// New/rising
	PublishedAtMs int64
	// Backtest
	BacktestSnapshot *antv1.BacktestSnapshot
}

// ListLeaderboard returns ranked strategies by the given type and period.
func (s *Service) ListLeaderboard(ctx context.Context, lbType, period, assetClass string, limit int) ([]LeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	switch lbType {
	case "return":
		return s.leaderboardByReturn(ctx, period, assetClass, limit)
	case "popular":
		return s.leaderboardByPopular(ctx, assetClass, limit)
	case "new":
		return s.leaderboardByNew(ctx, assetClass, limit)
	default:
		return s.leaderboardByPopular(ctx, assetClass, limit)
	}
}

func (s *Service) leaderboardByReturn(ctx context.Context, period, assetClass string, limit int) ([]LeaderboardEntry, error) {
	query, args := buildLeaderboardQuery("return", period, assetClass, limit)
	return s.scanLeaderboard(ctx, query, args)
}

func (s *Service) leaderboardByPopular(ctx context.Context, assetClass string, limit int) ([]LeaderboardEntry, error) {
	query, args := buildLeaderboardQuery("popular", "", assetClass, limit)
	return s.scanLeaderboard(ctx, query, args)
}

func (s *Service) leaderboardByNew(ctx context.Context, assetClass string, limit int) ([]LeaderboardEntry, error) {
	query, args := buildLeaderboardQuery("new", "", assetClass, limit)
	return s.scanLeaderboard(ctx, query, args)
}

func buildLeaderboardQuery(lbType, period, assetClass string, limit int) (string, []interface{}) {
	base := `SELECT ms.strategy_id::text, usp.id::text, COALESCE(ms.title,''),
        COALESCE(u.email, u.nickname, usp.user_id::text), usp.user_id::text,
        COALESCE(ms.price_model,''), COALESCE(ms.price_amount::text,'0'),
        COALESCE(ms.asset_class,''), COALESCE(ms.timeframe,''), COALESCE(ms.risk_level,''),
        COALESCE(ms.total_subscribers,0),
        COALESCE(r.avg_rating,0), COALESCE(r.rating_count,0)`
	var metrics string
	var extraJoins string
	extraWhere := ""
	orderBy := ""
	switch lbType {
	case "return":
		metrics = `,
        COALESCE(lps.total_return::text,'0'), COALESCE(lps.max_drawdown::text,'0'),
        COALESCE(lps.sharpe_ratio::text,''), COALESCE(lps.win_rate::text,''),
        COALESCE(lps.total_trades,0), COALESCE(lps.tracking_since::text,'')`
		extraJoins = ` LEFT JOIN marketplace_live_performance_summary lps ON lps.strategy_id = ms.strategy_id`
		extraWhere = ` AND lps.strategy_id IS NOT NULL AND lps.account_type = 'real'`
		interval := leaderboardPeriodToInterval(period)
		if interval != "" {
			extraWhere += fmt.Sprintf(" AND lps.last_updated >= now() - INTERVAL '%s'", interval)
		}
		orderBy = ` ORDER BY lps.total_return DESC`
	case "new":
		metrics = `,
        '0', '0', '', '', 0, ''`
		extraWhere = ` AND usp.published_at > now() - INTERVAL '30 days'`
		orderBy = ` ORDER BY COALESCE(ms.win_rate,0) DESC, usp.published_at DESC`
	default:
		metrics = `,
        '0', '0', '', '', 0, ''`
		orderBy = ` ORDER BY COALESCE(ms.total_subscribers,0) DESC, COALESCE(r.avg_rating,0) DESC`
	}
	from := ` FROM marketplace_strategies ms
 JOIN user_strategy_publishes usp ON usp.platform_strategy_id = ms.strategy_id
 LEFT JOIN users u ON u.id = usp.user_id
 LEFT JOIN (SELECT strategy_id, AVG(rating) AS avg_rating, COUNT(*)::int AS rating_count FROM marketplace_ratings GROUP BY strategy_id) r ON r.strategy_id = ms.strategy_id` + extraJoins
	query := base + metrics + `,
        EXTRACT(EPOCH FROM usp.published_at) * 1000,
        ms.backtest_snapshot` + from + ` WHERE ms.status = 'published'` + extraWhere
	args := []interface{}{}
	argIdx := 1
	if assetClass != "" {
		query += fmt.Sprintf(" AND ms.asset_class = $%d", argIdx)
		args = append(args, assetClass)
		argIdx++
	}
	query += fmt.Sprintf(orderBy+" LIMIT $%d", argIdx)
	args = append(args, limit)
	return query, args
}

func (s *Service) scanLeaderboard(ctx context.Context, query string, args []interface{}) ([]LeaderboardEntry, error) {
	rows, err := s.pg.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("marketplace: leaderboard query: %w", err)
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	rank := int32(1)
	for rows.Next() {
		var e LeaderboardEntry
		var snapshotRaw []byte
		var publishedAtSec float64
		if err := rows.Scan(&e.StrategyID, &e.PublishID, &e.Title, &e.PublisherName, &e.PublisherID,
			&e.PriceModel, &e.PriceAmount, &e.AssetClass, &e.Timeframe, &e.RiskLevel,
			&e.TotalSubscribers, &e.AvgRating, &e.RatingCount,
			&e.TotalReturn, &e.MaxDrawdown, &e.SharpeRatio, &e.WinRate, &e.TotalTrades, &e.TrackingSince,
			&publishedAtSec, &snapshotRaw); err != nil {
			return nil, fmt.Errorf("marketplace: leaderboard scan: %w", err)
		}
		e.PublishedAtMs = int64(publishedAtSec * 1000)
		e.Rank = rank
		rank++
		if len(snapshotRaw) > 0 {
			var snap antv1.BacktestSnapshot
			if err := proto.Unmarshal(snapshotRaw, &snap); err == nil {
				e.BacktestSnapshot = &snap
			}
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("marketplace: leaderboard rows: %w", err)
	}
	return entries, nil
}

func leaderboardPeriodToInterval(period string) string {
	switch period {
	case "week":
		return "7 days"
	case "month":
		return "30 days"
	case "quarter":
		return "90 days"
	default:
		return ""
	}
}
