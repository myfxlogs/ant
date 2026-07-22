package marketplace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// AnalyticsResult holds the marketplace analytics data.
type AnalyticsResult struct {
	TotalGMV        decimal.Decimal
	PlatformRevenue decimal.Decimal
	ProviderRevenue decimal.Decimal
	TotalTx         int32
	ActiveBuyers    int32
	NewSubscribers  int32
	ARPU            decimal.Decimal
	TotalStrategies int32
	NewStrategies   int32
	RefundRate      decimal.Decimal
	Daily           []DailyAnalyticsRow
}

type DailyAnalyticsRow struct {
	Date           string
	GMV            decimal.Decimal
	Transactions   int32
	NewSubscribers int32
}

// TopItemRow represents a top strategy or provider.
type TopItemRow struct {
	ID    string
	Name  string
	Value decimal.Decimal
	Rank  int32
}

// GetMarketplaceAnalytics computes marketplace analytics for the given period.
func (s *Service) GetMarketplaceAnalytics(ctx context.Context, period string) (*AnalyticsResult, error) {
	interval := analyticsPeriodToInterval(period)
	since := time.Now().Add(-interval)

	// Total GMV and transactions from wallet_transactions.
	// Purchase tx amounts are negative (buyer debit), so use ABS().
	var totalGMV decimal.Decimal
	var totalTx int32
	err := s.pg.QueryRow(ctx,
		`SELECT COALESCE(SUM(ABS(amount)),0), COUNT(*)
		 FROM wallet_transactions
		 WHERE tx_type = 'purchase' AND created_at >= $1`,
		since,
	).Scan(&totalGMV, &totalTx)
	if err != nil {
		return nil, fmt.Errorf("marketplace: analytics gmv: %w", err)
	}

	// Platform revenue = sum of fee_settlement transactions (Phase 5.4: replaces platform_fee).
	var platformRev decimal.Decimal
	err = s.pg.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount),0)
		 FROM wallet_transactions
		 WHERE tx_type = 'fee_settlement' AND created_at >= $1`,
		since,
	).Scan(&platformRev)
	if err != nil {
		return nil, fmt.Errorf("marketplace: analytics platform revenue: %w", err)
	}

	// Provider revenue = sum of settlement transactions (Phase 5.4: directly from settled amounts,
	// not derived from GMV - platformFee, since GMV includes frozen purchases not yet settled).
	var providerRev decimal.Decimal
	err = s.pg.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount),0)
		 FROM wallet_transactions
		 WHERE tx_type = 'settlement' AND created_at >= $1`,
		since,
	).Scan(&providerRev)
	if err != nil {
		return nil, fmt.Errorf("marketplace: analytics provider revenue: %w", err)
	}

	// Active buyers = distinct users with purchases in period.
	var activeBuyers int32
	err = s.pg.QueryRow(ctx,
		`SELECT COUNT(DISTINCT user_id)
		 FROM wallet_transactions
		 WHERE tx_type = 'purchase' AND created_at >= $1`,
		since,
	).Scan(&activeBuyers)
	if err != nil {
		return nil, fmt.Errorf("marketplace: analytics active buyers: %w", err)
	}

	// New subscribers in period.
	var newSubs int32
	err = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_subscriptions WHERE created_at >= $1`,
		since,
	).Scan(&newSubs)
	if err != nil {
		return nil, fmt.Errorf("marketplace: analytics new subscribers: %w", err)
	}

	// ARPU = GMV / active buyers.
	arpu := decimal.Zero
	if activeBuyers > 0 {
		arpu = totalGMV.Div(decimal.NewFromInt(int64(activeBuyers)))
	}

	// Total strategies.
	var totalStrategies int32
	err = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM marketplace_strategies`,
	).Scan(&totalStrategies)
	if err != nil {
		return nil, fmt.Errorf("marketplace: analytics total strategies: %w", err)
	}

	// New strategies in period.
	var newStrategies int32
	err = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM marketplace_strategies WHERE created_at >= $1`,
		since,
	).Scan(&newStrategies)
	if err != nil {
		return nil, fmt.Errorf("marketplace: analytics new strategies: %w", err)
	}

	// Refund rate = refunds / purchases.
	var refundCount int32
	err = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM wallet_transactions
		 WHERE tx_type = 'refund' AND created_at >= $1`,
		since,
	).Scan(&refundCount)
	if err != nil {
		return nil, fmt.Errorf("marketplace: analytics refund count: %w", err)
	}
	refundRate := decimal.Zero
	if totalTx > 0 {
		refundRate = decimal.NewFromInt(int64(refundCount)).Div(decimal.NewFromInt(int64(totalTx)))
	}

	// Daily breakdown.
	dailyRows, err := s.pg.Query(ctx,
		`SELECT DATE(wt.created_at)::text as d,
		        COALESCE(SUM(ABS(CASE WHEN wt.tx_type='purchase' THEN wt.amount ELSE 0 END)),0),
		        COUNT(CASE WHEN wt.tx_type='purchase' THEN 1 END)::int,
		        COALESCE(MAX(subs.cnt), 0)::int
		 FROM wallet_transactions wt
		 LEFT JOIN (
		   SELECT DATE(created_at)::date as sub_date, COUNT(*)::int as cnt
		   FROM user_subscriptions
		   WHERE created_at >= $1
		   GROUP BY sub_date
		 ) subs ON subs.sub_date = DATE(wt.created_at)
		 WHERE wt.created_at >= $1
		 GROUP BY d ORDER BY d`,
		since,
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace: analytics daily: %w", err)
	}
	defer dailyRows.Close()

	var daily []DailyAnalyticsRow
	for dailyRows.Next() {
		var d DailyAnalyticsRow
		if err := dailyRows.Scan(&d.Date, &d.GMV, &d.Transactions, &d.NewSubscribers); err != nil {
			return nil, fmt.Errorf("marketplace: analytics daily scan: %w", err)
		}
		daily = append(daily, d)
	}
	if err := dailyRows.Err(); err != nil {
		return nil, err
	}

	return &AnalyticsResult{
		TotalGMV:        totalGMV,
		PlatformRevenue: platformRev,
		ProviderRevenue: providerRev,
		TotalTx:         totalTx,
		ActiveBuyers:    activeBuyers,
		NewSubscribers:  newSubs,
		ARPU:            arpu,
		TotalStrategies: totalStrategies,
		NewStrategies:   newStrategies,
		RefundRate:      refundRate,
		Daily:           daily,
	}, nil
}

// GetTopStrategies returns top strategies by revenue and subscribers.
func (s *Service) GetTopStrategies(ctx context.Context) ([]TopItemRow, []TopItemRow, error) {
	// By revenue from settlement transactions (Phase 5.4).
	revRows, err := s.pg.Query(ctx,
		`SELECT ms.strategy_id::text, COALESCE(ms.title,''),
		        COALESCE(SUM(wt.amount),0)
		 FROM wallet_transactions wt
		 JOIN user_subscriptions us`+subJoinOnClause+`
		 JOIN marketplace_strategies ms ON ms.strategy_id = us.target_strategy_id
		 WHERE wt.tx_type = 'settlement'
		 GROUP BY ms.strategy_id, ms.title
		 ORDER BY COALESCE(SUM(wt.amount),0) DESC LIMIT 10`)
	if err != nil {
		return nil, nil, fmt.Errorf("marketplace: top strategies by revenue: %w", err)
	}
	defer revRows.Close()

	var byRev []TopItemRow
	rank := int32(1)
	for revRows.Next() {
		var item TopItemRow
		if err := revRows.Scan(&item.ID, &item.Name, &item.Value); err != nil {
			return nil, nil, err
		}
		item.Rank = rank
		byRev = append(byRev, item)
		rank++
	}
	if err := revRows.Err(); err != nil {
		return nil, nil, err
	}

	// By subscribers.
	subRows, err := s.pg.Query(ctx,
		`SELECT ms.strategy_id::text, COALESCE(ms.title,''),
		        COALESCE(ms.total_subscribers,0)
		 FROM marketplace_strategies ms
		 WHERE ms.status = 'published'
		 ORDER BY COALESCE(ms.total_subscribers,0) DESC LIMIT 10`)
	if err != nil {
		return nil, nil, fmt.Errorf("marketplace: top strategies by subscribers: %w", err)
	}
	defer subRows.Close()

	var bySub []TopItemRow
	rank = 1
	for subRows.Next() {
		var item TopItemRow
		if err := subRows.Scan(&item.ID, &item.Name, &item.Value); err != nil {
			return nil, nil, err
		}
		item.Rank = rank
		bySub = append(bySub, item)
		rank++
	}
	if err := subRows.Err(); err != nil {
		return nil, nil, err
	}

	return byRev, bySub, nil
}

// GetTopProviders returns top providers by revenue and strategy count.
func (s *Service) GetTopProviders(ctx context.Context) ([]TopItemRow, []TopItemRow, error) {
	// By revenue from settlement transactions (Phase 5.4).
	revRows, err := s.pg.Query(ctx,
		`SELECT ms.publisher_id::text,
		        COALESCE(u.nickname, u.email, ms.publisher_id::text),
		        COALESCE(SUM(wt.amount),0)
		 FROM wallet_transactions wt
		 JOIN user_subscriptions us`+subJoinOnClause+`
		 JOIN marketplace_strategies ms ON ms.strategy_id = us.target_strategy_id
		 LEFT JOIN users u ON u.id = ms.publisher_id
		 WHERE wt.tx_type = 'settlement' AND ms.publisher_id IS NOT NULL
		 GROUP BY ms.publisher_id, u.nickname, u.email
		 ORDER BY COALESCE(SUM(wt.amount),0) DESC LIMIT 10`)
	if err != nil {
		return nil, nil, fmt.Errorf("marketplace: top providers by revenue: %w", err)
	}
	defer revRows.Close()

	var byRev []TopItemRow
	rank := int32(1)
	for revRows.Next() {
		var item TopItemRow
		if err := revRows.Scan(&item.ID, &item.Name, &item.Value); err != nil {
			return nil, nil, err
		}
		item.Rank = rank
		byRev = append(byRev, item)
		rank++
	}
	if err := revRows.Err(); err != nil {
		return nil, nil, err
	}

	// By strategy count.
	stratRows, err := s.pg.Query(ctx,
		`SELECT ms.publisher_id::text,
		        COALESCE(u.nickname, u.email, ms.publisher_id::text),
		        COUNT(*)::numeric
		 FROM marketplace_strategies ms
		 LEFT JOIN users u ON u.id = ms.publisher_id
		 WHERE ms.status = 'published' AND ms.publisher_id IS NOT NULL
		 GROUP BY ms.publisher_id, u.nickname, u.email
		 ORDER BY COUNT(*) DESC LIMIT 10`)
	if err != nil {
		return nil, nil, fmt.Errorf("marketplace: top providers by strategies: %w", err)
	}
	defer stratRows.Close()

	var byStrat []TopItemRow
	rank = 1
	for stratRows.Next() {
		var item TopItemRow
		if err := stratRows.Scan(&item.ID, &item.Name, &item.Value); err != nil {
			return nil, nil, err
		}
		item.Rank = rank
		byStrat = append(byStrat, item)
		rank++
	}
	if err := stratRows.Err(); err != nil {
		return nil, nil, err
	}

	return byRev, byStrat, nil
}

func analyticsPeriodToInterval(period string) time.Duration {
	switch strings.ToLower(period) {
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	case "90d":
		return 90 * 24 * time.Hour
	default:
		return 365 * 24 * time.Hour
	}
}
