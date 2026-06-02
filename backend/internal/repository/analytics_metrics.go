package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

func (r *AnalyticsRepository) GetMonthlyAnalysisRaw(ctx context.Context, accountID uuid.UUID) ([]*model.MonthlyAnalysisPoint, error) {
	query := `
		SELECT
			EXTRACT(YEAR FROM close_time)::int AS year,
			EXTRACT(MONTH FROM close_time)::int AS month,
			COALESCE(SUM(profit), 0) AS profit,
			COALESCE(SUM(volume), 0) AS lots,
			COALESCE(SUM(
				(CASE
					WHEN LOWER(order_type) LIKE 'buy%' THEN (close_price - open_price)
					ELSE (open_price - close_price)
				END) *
				(CASE
					WHEN symbol ILIKE '%JPY%' THEN 100
					ELSE 10000
				END)
			), 0) AS pips,
			COUNT(*)::int AS trades
		FROM trade_records
		WHERE account_id = $1
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
		GROUP BY EXTRACT(YEAR FROM close_time), EXTRACT(MONTH FROM close_time)
		ORDER BY year ASC, month ASC
	`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var points []*model.MonthlyAnalysisPoint
	for rows.Next() {
		p := &model.MonthlyAnalysisPoint{}
		if err := rows.Scan(&p.Year, &p.Month, &p.Profit, &p.Lots, &p.Pips, &p.Trades); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (r *AnalyticsRepository) GetMonthlyAnalysisYears(ctx context.Context, accountID uuid.UUID) ([]int, error) {
	query := `
		SELECT DISTINCT EXTRACT(YEAR FROM close_time)::int AS year
		FROM trade_records
		WHERE account_id = $1
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
		ORDER BY year ASC
	`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var years []int
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			return nil, err
		}
		years = append(years, year)
	}
	return years, rows.Err()
}

func (r *AnalyticsRepository) GetHourlyStats(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.HourlyStats, error) {
	query := `
		SELECT
			EXTRACT(HOUR FROM close_time)::int AS hour_start,
			COUNT(*)::int AS trades,
			COALESCE(SUM(volume), 0) AS lots,
			COALESCE(SUM(profit), 0) AS profit,
			COALESCE(SUM(CASE WHEN profit > 0 THEN profit ELSE 0 END), 0) AS gross_profit,
			COALESCE(SUM(CASE WHEN profit < 0 THEN ABS(profit) ELSE 0 END), 0) AS gross_loss,
			CASE WHEN COUNT(*) > 0
				THEN SUM(CASE WHEN profit > 0 THEN 1 ELSE 0 END)::float / COUNT(*) * 100
				ELSE 0
			END AS win_rate
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
		GROUP BY EXTRACT(HOUR FROM close_time)
		ORDER BY hour_start
	`
	rows, err := r.db.Query(ctx, query, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stats []*struct {
		HourStart   int     `db:"hour_start"`
		Trades      int     `db:"trades"`
		Lots        float64 `db:"lots"`
		Profit      float64 `db:"profit"`
		GrossProfit float64 `db:"gross_profit"`
		GrossLoss   float64 `db:"gross_loss"`
		WinRate     float64 `db:"win_rate"`
	}
	for rows.Next() {
		s := &struct {
			HourStart   int     `db:"hour_start"`
			Trades      int     `db:"trades"`
			Lots        float64 `db:"lots"`
			Profit      float64 `db:"profit"`
			GrossProfit float64 `db:"gross_profit"`
			GrossLoss   float64 `db:"gross_loss"`
			WinRate     float64 `db:"win_rate"`
		}{}
		if err := rows.Scan(&s.HourStart, &s.Trades, &s.Lots, &s.Profit, &s.GrossProfit, &s.GrossLoss, &s.WinRate); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]*model.HourlyStats, 24)
	for h := 0; h < 24; h++ {
		result[h] = &model.HourlyStats{
			Hour:                  fmt.Sprintf("%02d:00", h),
			HourStart:             h,
			Trades:                0,
			Profit:                0,
			WinRate:               0,
			AvgPnL:                0,
			Lots:                  0,
			Balance:               0,
			ProfitFactor:          0,
			MaxFloatingLossAmount: 0,
			MaxFloatingLossRatio:  0,
			MaxFloatingProfitAmount: 0,
			MaxFloatingProfitRatio:  0,
		}
	}
	for _, s := range stats {
		if s.HourStart >= 0 && s.HourStart < 24 {
			result[s.HourStart].Trades = s.Trades
			result[s.HourStart].Lots = s.Lots
			result[s.HourStart].Profit = s.Profit
			result[s.HourStart].WinRate = s.WinRate
			if s.Trades > 0 {
				result[s.HourStart].AvgPnL = s.Profit / float64(s.Trades)
			}
			if s.GrossLoss > 0 {
				result[s.HourStart].ProfitFactor = s.GrossProfit / s.GrossLoss
			}
		}
	}

	return result, nil
}
