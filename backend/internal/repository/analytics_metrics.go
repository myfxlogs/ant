package repository

import (
	"github.com/shopspring/decimal"
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

type hourlyRawStat struct {
	HourStart, Trades int
	Lots, Profit, GrossProfit, GrossLoss decimal.Decimal
	WinRate float64
}

func (r *AnalyticsRepository) GetHourlyStats(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.HourlyStats, error) {
	raw, err := r.queryHourlyStats(ctx, accountID, start, end)
	if err != nil { return nil, err }
	return buildHourlyStatsResult(raw), nil
}

func (r *AnalyticsRepository) queryHourlyStats(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*hourlyRawStat, error) {
	rows, err := r.db.Query(ctx,
		`SELECT EXTRACT(HOUR FROM close_time)::int AS hour_start, COUNT(*)::int,
		        COALESCE(SUM(volume),0), COALESCE(SUM(profit),0),
		        COALESCE(SUM(CASE WHEN profit>0 THEN profit ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN profit<0 THEN ABS(profit) ELSE 0 END),0),
		        CASE WHEN COUNT(*)>0 THEN SUM(CASE WHEN profit>0 THEN 1 ELSE 0 END)::float/COUNT(*)*100 ELSE 0 END
		 FROM trade_records WHERE account_id=$1 AND close_time>=$2 AND close_time<=$3
		   AND order_type NOT IN ('balance','credit','BALANCE','CREDIT','Balance','Credit')
		 GROUP BY hour_start ORDER BY hour_start`,
		accountID, start, end)
	if err != nil { return nil, err }
	defer rows.Close()
	var stats []*hourlyRawStat
	for rows.Next() {
		s := &hourlyRawStat{}
		if err := rows.Scan(&s.HourStart, &s.Trades, &s.Lots, &s.Profit, &s.GrossProfit, &s.GrossLoss, &s.WinRate); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func buildHourlyStatsResult(raw []*hourlyRawStat) []*model.HourlyStats {
	result := make([]*model.HourlyStats, 24)
	for h := 0; h < 24; h++ {
		result[h] = &model.HourlyStats{
			Hour: fmt.Sprintf("%02d:00", h), HourStart: h,
			Profit: decimal.Zero, WinRate: decimal.Zero, Lots: decimal.Zero, Balance: decimal.Zero,
			ProfitFactor: decimal.Zero,
		}
	}
	for _, s := range raw {
		if s.HourStart < 0 || s.HourStart >= 24 { continue }
		result[s.HourStart].Trades = s.Trades
		result[s.HourStart].Lots = s.Lots
		result[s.HourStart].Profit = s.Profit
		result[s.HourStart].WinRate = decimal.NewFromFloat(s.WinRate)
		if s.Trades > 0 { result[s.HourStart].AvgPnL = s.Profit.InexactFloat64() / float64(s.Trades) }
		if s.GrossLoss.IsPositive() { result[s.HourStart].ProfitFactor = s.GrossProfit.Div(s.GrossLoss) }
	}
	return result
}
