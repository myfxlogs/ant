package repository

import (
	"github.com/shopspring/decimal"
	"context"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

func (r *AnalyticsRepository) GetDailyReturns(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]float64, error) {
	query := `
		SELECT COALESCE(SUM(profit), 0) as daily_return
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
		GROUP BY DATE(close_time)
		ORDER BY DATE(close_time)
	`
	rows, err := r.db.Query(ctx, query, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var returns []float64
	for rows.Next() {
		var ret float64
		if err := rows.Scan(&ret); err != nil {
			return nil, err
		}
		returns = append(returns, ret)
	}
	return returns, rows.Err()
}

func (r *AnalyticsRepository) GetMaxDrawdown(ctx context.Context, accountID uuid.UUID, start, end time.Time) (maxDrawdown float64, maxDrawdownPercent float64, err error) {
	query := `
		WITH daily_pnl AS (
			SELECT
				DATE(close_time) as date,
				SUM(profit) as daily_pnl
			FROM trade_records
			WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
				AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
			GROUP BY DATE(close_time)
			ORDER BY date
		),
		cumulative AS (
			SELECT
				date,
				daily_pnl,
				SUM(daily_pnl) OVER (ORDER BY date) as cumulative_pnl
			FROM daily_pnl
		),
		with_running_max AS (
			SELECT
				cumulative_pnl,
				MAX(cumulative_pnl) OVER (ORDER BY date ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) as running_max
			FROM cumulative
		)
		SELECT
			COALESCE(MIN(cumulative_pnl - running_max), 0) as max_drawdown,
			CASE
				WHEN MAX(running_max) > 0
				THEN ABS(MIN(cumulative_pnl - running_max)) / MAX(running_max) * 100
				ELSE 0
			END as max_drawdown_percent
		FROM with_running_max
	`
	err = r.db.QueryRow(ctx, query, accountID, start, end).Scan(&maxDrawdown, &maxDrawdownPercent)
	return
}

func (r *AnalyticsRepository) GetMonthlyPnL(ctx context.Context, accountID uuid.UUID, year int) ([]*model.MonthlyPnL, error) {
	query := `
		SELECT
			EXTRACT(MONTH FROM close_time) as month_num,
			TO_CHAR(close_time, 'Mon') as month,
			COALESCE(SUM(profit), 0) as profit,
			COUNT(*) as trades,
			SUM(CASE WHEN profit > 0 THEN 1 ELSE 0 END) as win_trades,
			SUM(CASE WHEN profit < 0 THEN 1 ELSE 0 END) as loss_trades
		FROM trade_records
		WHERE account_id = $1 AND EXTRACT(YEAR FROM close_time) = $2
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
		GROUP BY EXTRACT(MONTH FROM close_time), TO_CHAR(close_time, 'Mon')
		ORDER BY month_num
	`
	rows, err := r.db.Query(ctx, query, accountID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type monthStat struct {
		MonthNum   int     `db:"month_num"`
		Month      string  `db:"month"`
		Profit     float64 `db:"profit"`
		Trades     int     `db:"trades"`
		WinTrades  int     `db:"win_trades"`
		LossTrades int     `db:"loss_trades"`
	}
	var stats []*monthStat
	for rows.Next() {
		s := &monthStat{}
		if err := rows.Scan(&s.MonthNum, &s.Month, &s.Profit, &s.Trades, &s.WinTrades, &s.LossTrades); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	monthNames := []string{"1月", "2月", "3月", "4月", "5月", "6月", "7月", "8月", "9月", "10月", "11月", "12月"}
	result := make([]*model.MonthlyPnL, 12)
	for i := 0; i < 12; i++ {
		result[i] = &model.MonthlyPnL{
			Month:      monthNames[i],
			MonthNum:   i + 1,
			Profit:     decimal.Zero,
			Trades:     0,
			WinTrades:  0,
			LossTrades: 0,
		}
	}
	for _, s := range stats {
		idx := s.MonthNum - 1
		result[idx].Profit = decimal.NewFromFloat(s.Profit)
		result[idx].Trades = s.Trades
		result[idx].WinTrades = s.WinTrades
		result[idx].LossTrades = s.LossTrades
	}

	return result, nil
}

// GetWeekdayPnL aggregates closed-trade P/L by ISO weekday (1=Mon … 7=Sun) in [start, end].
func (r *AnalyticsRepository) GetWeekdayPnL(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.WeekdayPnL, error) {
	query := `
		SELECT
			EXTRACT(ISODOW FROM close_time)::int AS weekday,
			COALESCE(SUM(profit), 0) AS pnl,
			COUNT(*)::int AS trades
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
		GROUP BY EXTRACT(ISODOW FROM close_time)
		ORDER BY weekday
	`
	rows, err := r.db.Query(ctx, query, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type weekdayStat struct {
		Weekday int     `db:"weekday"`
		PnL     float64 `db:"pnl"`
		Trades  int     `db:"trades"`
	}
	var stats []*weekdayStat
	for rows.Next() {
		s := &weekdayStat{}
		if err := rows.Scan(&s.Weekday, &s.PnL, &s.Trades); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]*model.WeekdayPnL, 7)
	for i := 0; i < 7; i++ {
		out[i] = &model.WeekdayPnL{Weekday: i + 1, PnL: 0, Trades: 0}
	}
	for _, s := range stats {
		if s.Weekday >= 1 && s.Weekday <= 7 {
			out[s.Weekday-1].PnL = s.PnL
			out[s.Weekday-1].Trades = s.Trades
		}
	}
	return out, nil
}
