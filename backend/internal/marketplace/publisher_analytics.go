package marketplace

import (
	"context"
	"fmt"
	"time"
)

// getRevenueTrend returns daily revenue for the last 30 days, split by one-time vs subscription.
func (s *Service) getRevenueTrend(ctx context.Context, userID string) ([]RevenueTrendPoint, error) {
	rows, err := s.pg.Query(ctx,
		`SELECT date_trunc('day', wt.created_at) AS day,
		        COALESCE(SUM(wt.amount::numeric) FILTER (WHERE us.kind = 'purchase'), 0)::text,
		        COALESCE(SUM(wt.amount::numeric) FILTER (WHERE us.kind = 'subscription'), 0)::text
		 FROM wallet_transactions wt
		 JOIN marketplace_settlements ms ON ms.id::text = REPLACE(wt.idem_key, 'mkt-settle-', '')
		 JOIN user_subscriptions us ON us.id = ms.purchase_id
		 WHERE wt.user_id::text = $1 AND wt.tx_type = 'settlement'
		   AND wt.created_at > now() - INTERVAL '30 days'
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
		`SELECT ms2.strategy_id::text,
		        COALESCE(ms2.title, ''),
		        ms2.total_subscribers,
		        COALESCE((
		          SELECT SUM(wt.amount::numeric)
		          FROM wallet_transactions wt
		          JOIN marketplace_settlements settle ON settle.id::text = REPLACE(wt.idem_key, 'mkt-settle-', '')
		          JOIN user_subscriptions us ON us.id = settle.purchase_id
		          WHERE wt.user_id::text = $1 AND wt.tx_type = 'settlement'
		            AND us.target_strategy_id = ms2.strategy_id
		        ), 0)::text,
		        COALESCE(AVG(mr.rating), 0),
		        COUNT(mr.rating),
		        ms2.price_model,
		        COALESCE(ms2.price_amount::text, '0')
		 FROM marketplace_strategies ms2
		 LEFT JOIN marketplace_ratings mr ON mr.strategy_id = ms2.strategy_id
		 WHERE ms2.publisher_id::text = $1 AND ms2.status = 'published'
		 GROUP BY ms2.strategy_id, ms2.title, ms2.total_subscribers, ms2.price_model, ms2.price_amount
		 ORDER BY ms2.total_subscribers DESC`,
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
