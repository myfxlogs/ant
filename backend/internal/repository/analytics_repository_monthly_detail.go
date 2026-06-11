package repository

import (
	"context"
	"math"

	"github.com/google/uuid"
	"anttrader/internal/model"
)

// ── Monthly Detail (drill-down) ──

// GetMonthlyDetailMetrics returns aggregated metrics for a single month.
func (r *AnalyticsRepository) GetMonthlyDetailMetrics(ctx context.Context, accountID uuid.UUID, year, month int) (*model.MonthlyDetailMetrics, error) {
	query := `
		SELECT
			COALESCE(SUM(profit), 0)                            AS net_return,
			COUNT(*)::int                                        AS total_trades,
			CASE WHEN COUNT(*) > 0
				THEN SUM(CASE WHEN profit > 0 THEN 1 ELSE 0 END)::float / COUNT(*) * 100
				ELSE 0
			END                                                  AS win_rate,
			CASE WHEN SUM(CASE WHEN profit < 0 THEN ABS(profit) ELSE 0 END) > 0
				THEN SUM(CASE WHEN profit > 0 THEN profit ELSE 0 END) /
				     SUM(CASE WHEN profit < 0 THEN ABS(profit) ELSE 0 END)
				ELSE 0
			END                                                  AS profit_factor,
			COALESCE(MAX(profit), 0)                             AS best_trade,
			COALESCE(MIN(profit), 0)                             AS worst_trade
		FROM trade_records
		WHERE account_id = $1
			AND EXTRACT(YEAR FROM close_time) = $2
			AND EXTRACT(MONTH FROM close_time) = $3
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
	`
	m := &model.MonthlyDetailMetrics{}
	err := r.db.QueryRow(ctx, query, accountID, year, month).Scan(
		&m.NetReturn, &m.TotalTrades, &m.WinRate, &m.ProfitFactor, &m.BestTrade, &m.WorstTrade,
	)
	if err != nil {
		return nil, err
	}
	m.NetReturn = math.Round(m.NetReturn*100) / 100
	m.WinRate = math.Round(m.WinRate*100) / 100
	m.ProfitFactor = math.Round(m.ProfitFactor*100) / 100
	m.BestTrade = math.Round(m.BestTrade*100) / 100
	m.WorstTrade = math.Round(m.WorstTrade*100) / 100
	return m, nil
}

// GetMonthlySymbolPnL returns per-symbol P&L for a single month, sorted by profit desc.
func (r *AnalyticsRepository) GetMonthlySymbolPnL(ctx context.Context, accountID uuid.UUID, year, month int) ([]*model.MonthlySymbolPnL, error) {
	query := `
		SELECT
			symbol,
			COALESCE(SUM(profit), 0) AS net_profit,
			COUNT(*)::int             AS trades,
			CASE WHEN COUNT(*) > 0
				THEN SUM(CASE WHEN profit > 0 THEN 1 ELSE 0 END)::float / COUNT(*) * 100
				ELSE 0
			END                        AS win_rate
		FROM trade_records
		WHERE account_id = $1
			AND EXTRACT(YEAR FROM close_time) = $2
			AND EXTRACT(MONTH FROM close_time) = $3
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
		GROUP BY symbol
		ORDER BY net_profit DESC
	`
	rows, err := r.db.Query(ctx, query, accountID, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*model.MonthlySymbolPnL
	for rows.Next() {
		s := &model.MonthlySymbolPnL{}
		if err := rows.Scan(&s.Symbol, &s.NetProfit, &s.Trades, &s.WinRate); err != nil {
			return nil, err
		}
		s.NetProfit = math.Round(s.NetProfit*100) / 100
		s.WinRate = math.Round(s.WinRate*100) / 100
		results = append(results, s)
	}
	return results, rows.Err()
}

// GetMonthlyHoldingStats returns holding time stats for a single month (in hours).
func (r *AnalyticsRepository) GetMonthlyHoldingStats(ctx context.Context, accountID uuid.UUID, year, month int) (*model.MonthlyHoldingStats, error) {
	query := `
		SELECT
			COALESCE(AVG(EXTRACT(EPOCH FROM (close_time - open_time))) / 3600.0, 0)   AS avg_hours,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (
				ORDER BY EXTRACT(EPOCH FROM (close_time - open_time))
			) / 3600.0, 0)                                                             AS median_hours,
			COALESCE(MAX(EXTRACT(EPOCH FROM (close_time - open_time))) / 3600.0, 0)   AS max_hours,
			COALESCE(MIN(EXTRACT(EPOCH FROM (close_time - open_time))) / 3600.0, 0)   AS min_hours
		FROM trade_records
		WHERE account_id = $1
			AND EXTRACT(YEAR FROM close_time) = $2
			AND EXTRACT(MONTH FROM close_time) = $3
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
			AND open_time IS NOT NULL
			AND close_time IS NOT NULL
	`
	s := &model.MonthlyHoldingStats{}
	err := r.db.QueryRow(ctx, query, accountID, year, month).Scan(
		&s.AverageHours, &s.MedianHours, &s.MaxHours, &s.MinHours,
	)
	if err != nil {
		return nil, err
	}
	s.AverageHours = math.Round(s.AverageHours*100) / 100
	s.MedianHours = math.Round(s.MedianHours*100) / 100
	s.MaxHours = math.Round(s.MaxHours*100) / 100
	s.MinHours = math.Round(s.MinHours*100) / 100
	return s, nil
}
