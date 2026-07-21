package marketplace

import (
	"context"
	"fmt"
	"time"
)

// getRevenueTrend returns daily revenue for the last 30 days, split by sale vs subscription.
func (s *Service) getRevenueTrend(ctx context.Context, userID string) ([]RevenueTrendPoint, error) {
	rows, err := s.pg.Query(ctx,
		`SELECT date_trunc('day', created_at) AS day,
		        COALESCE(SUM(amount::numeric) FILTER (WHERE description LIKE 'Strategy sale: %' AND description NOT LIKE 'Strategy renewal%'), 0)::text,
		        COALESCE(SUM(amount::numeric) FILTER (WHERE description LIKE 'Strategy renewal%'), 0)::text
		 FROM wallet_transactions
		 WHERE user_id::text = $1 AND tx_type = 'sale'
		   AND created_at > now() - INTERVAL '30 days'
		 GROUP BY day ORDER BY day`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: revenue trend: %w", err)
	}
	defer rows.Close()

	var points []RevenueTrendPoint
	for rows.Next() {
		var day time.Time
		var saleRev, subRev string
		if err := rows.Scan(&day, &saleRev, &subRev); err != nil {
			return nil, fmt.Errorf("marketplace: revenue trend scan: %w", err)
		}
		points = append(points, RevenueTrendPoint{
			DateMs:              day.UnixMilli(),
			SaleRevenue:         saleRev,
			SubscriptionRevenue: subRev,
		})
	}
	return points, nil
}

// getSubscriberTrend returns daily subscriber changes for the last 30 days.
func (s *Service) getSubscriberTrend(ctx context.Context, userID string) ([]SubscriberTrendPoint, error) {
	// Daily new subscribers (created) and churned (deactivated).
	rows, err := s.pg.Query(ctx,
		`WITH target_strategies AS (
			SELECT strategy_id FROM marketplace_strategies WHERE publisher_id::text = $1 AND status = 'published'
		)
		 SELECT day,
		        SUM(new_count) AS new_count,
		        SUM(churn_count) AS churn_count
		 FROM (
			SELECT date_trunc('day', created_at) AS day,
			       COUNT(*) AS new_count, 0 AS churn_count
			FROM user_subscriptions us
			JOIN target_strategies ts ON ts.strategy_id = us.target_strategy_id
			WHERE us.created_at > now() - INTERVAL '30 days'
			GROUP BY day
			UNION ALL
			SELECT date_trunc('day', updated_at) AS day,
			       0 AS new_count, COUNT(*) AS churn_count
			FROM user_subscriptions us
			JOIN target_strategies ts ON ts.strategy_id = us.target_strategy_id
			WHERE us.active = false AND us.updated_at > now() - INTERVAL '30 days'
			GROUP BY day
		 ) combined
		 GROUP BY day ORDER BY day`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: subscriber trend: %w", err)
	}
	defer rows.Close()

	// Build a map for quick lookup.
	type dayData struct {
		newCount, churnCount int32
	}
	dayMap := make(map[int64]dayData)
	for rows.Next() {
		var day time.Time
		var nc, cc int32
		if err := rows.Scan(&day, &nc, &cc); err != nil {
			return nil, fmt.Errorf("marketplace: subscriber trend scan: %w", err)
		}
		dayMap[day.UnixMilli()] = dayData{nc, cc}
	}

	// Calculate baseline: active count at start of 30-day window.
	// baseline = currentActive - totalNew + totalChurned (undo 30 days of changes)
	var currentActive int32
	_ = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_subscriptions us
		 JOIN marketplace_strategies ms ON ms.strategy_id = us.target_strategy_id
		 WHERE ms.publisher_id::text = $1 AND us.active = true`,
		userID).Scan(&currentActive)

	var totalNew, totalChurned int32
	for _, dd := range dayMap {
		totalNew += dd.newCount
		totalChurned += dd.churnCount
	}
	runningActive := currentActive - totalNew + totalChurned
	if runningActive < 0 {
		runningActive = 0
	}

	// Fill in all 30 days, forward from oldest to newest.
	now := time.Now().UTC().Truncate(24 * time.Hour)
	var points []SubscriberTrendPoint
	for i := 29; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		ms := day.UnixMilli()
		dd := dayMap[ms]
		runningActive = runningActive + dd.newCount - dd.churnCount
		if runningActive < 0 {
			runningActive = 0
		}
		points = append(points, SubscriberTrendPoint{
			DateMs:         ms,
			NewSubscribers: dd.newCount,
			Churned:        dd.churnCount,
			Active:         runningActive,
		})
	}
	return points, nil
}

// getStrategyBreakdown returns per-strategy analytics for the publisher.
func (s *Service) getStrategyBreakdown(ctx context.Context, userID string) ([]StrategyBreakdown, error) {
	rows, err := s.pg.Query(ctx,
		`SELECT ms.strategy_id::text,
		        COALESCE(ms.title, ''),
		        ms.total_subscribers,
		        COALESCE(SUM(wt.amount::numeric), 0)::text,
		        COALESCE(AVG(mr.rating), 0),
		        COUNT(mr.rating),
		        ms.price_model,
		        COALESCE(ms.price_amount::text, '0')
		 FROM marketplace_strategies ms
		 LEFT JOIN wallet_transactions wt ON wt.user_id::text = $1 AND wt.tx_type = 'sale'
		      AND wt.description LIKE 'Strategy sale: ' || COALESCE(ms.title, '') || '%'
		 LEFT JOIN marketplace_ratings mr ON mr.strategy_id = ms.strategy_id
		 WHERE ms.publisher_id::text = $1 AND ms.status = 'published'
		 GROUP BY ms.strategy_id, ms.title, ms.total_subscribers, ms.price_model, ms.price_amount
		 ORDER BY ms.total_subscribers DESC`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: strategy breakdown: %w", err)
	}
	defer rows.Close()

	var items []StrategyBreakdown
	for rows.Next() {
		var item StrategyBreakdown
		if err := rows.Scan(&item.StrategyID, &item.Title, &item.TotalSubscribers,
			&item.Revenue, &item.AvgRating, &item.RatingCount,
			&item.PriceModel, &item.PriceAmount); err != nil {
			return nil, fmt.Errorf("marketplace: strategy breakdown scan: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("marketplace: strategy breakdown rows: %w", err)
	}
	return items, nil
}
