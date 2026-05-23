package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

func (r *AnalyticsRepository) GetSymbolStats(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.SymbolStats, error) {
	query := `
		SELECT
			symbol,
			COUNT(*) as total_trades,
			COALESCE(SUM(CASE WHEN profit > 0 THEN 1 ELSE 0 END), 0) as winning_trades,
			COALESCE(SUM(CASE WHEN profit < 0 THEN 1 ELSE 0 END), 0) as losing_trades,
			ROUND(CAST(
				CASE
					WHEN COUNT(*) > 0
					THEN SUM(CASE WHEN profit > 0 THEN 1 ELSE 0 END)::float / COUNT(*) * 100
					ELSE 0
				END AS numeric), 2
			) as win_rate,
			COALESCE(SUM(CASE WHEN profit > 0 THEN profit ELSE 0 END), 0) as total_profit,
			COALESCE(ABS(SUM(CASE WHEN profit < 0 THEN profit ELSE 0 END)), 0) as total_loss,
			COALESCE(SUM(profit), 0) as net_profit,
			CASE
				WHEN ABS(SUM(CASE WHEN profit < 0 THEN profit ELSE 0 END)) > 0
				THEN SUM(CASE WHEN profit > 0 THEN profit ELSE 0 END) / ABS(SUM(CASE WHEN profit < 0 THEN profit ELSE 0 END))
				ELSE 0
			END as profit_factor,
			CASE
				WHEN SUM(CASE WHEN profit > 0 THEN 1 ELSE 0 END) > 0
				THEN SUM(CASE WHEN profit > 0 THEN profit ELSE 0 END) / SUM(CASE WHEN profit > 0 THEN 1 ELSE 0 END)
				ELSE 0
			END as average_profit,
			COALESCE(SUM(volume), 0) as total_volume,
			CASE
				WHEN COUNT(*) > 0
				THEN SUM(volume) / COUNT(*)
				ELSE 0
			END as average_volume,
			COALESCE(MAX(CASE WHEN profit > 0 THEN profit ELSE 0 END), 0) as largest_win,
			COALESCE(ABS(MIN(CASE WHEN profit < 0 THEN profit ELSE 0 END)), 0) as largest_loss,
			'' as average_holding_time
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3 AND symbol != '' AND symbol IS NOT NULL
		GROUP BY symbol
		ORDER BY net_profit DESC
	`
	var stats []*model.SymbolStats
	err := r.db.SelectContext(ctx, &stats, query, accountID, start, end)
	return stats, err
}

func (r *AnalyticsRepository) GetDailyEquity(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.DailyEquity, error) {
	query := `
		SELECT
			DATE(close_time) as date,
			COALESCE(SUM(CASE WHEN order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') THEN profit ELSE 0 END), 0) as profit
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
		GROUP BY DATE(close_time)
		ORDER BY date ASC
	`
	type dailyProfit struct {
		Date   time.Time `db:"date"`
		Profit float64   `db:"profit"`
	}
	var dailyProfits []dailyProfit
	err := r.db.SelectContext(ctx, &dailyProfits, query, accountID, start, end)
	if err != nil {
		return nil, err
	}

	var result []*model.DailyEquity
	runningBalance := 0.0
	for _, dp := range dailyProfits {
		runningBalance += dp.Profit
		result = append(result, &model.DailyEquity{
			Date:     dp.Date.Format("2006-01-02"),
			Profit:   dp.Profit,
			Balance:  runningBalance,
			Equity:   runningBalance,
			Drawdown: 0,
		})
	}

	return result, nil
}

func (r *AnalyticsRepository) GetAccountBalance(ctx context.Context, accountID uuid.UUID) (float64, error) {
	query := `SELECT balance FROM mt_accounts WHERE id = $1`
	var balance float64
	err := r.db.GetContext(ctx, &balance, query, accountID)
	return balance, err
}

func (r *AnalyticsRepository) GetAccountInitialBalance(ctx context.Context, accountID uuid.UUID) (float64, error) {
	query := `
		SELECT COALESCE(
			(SELECT balance FROM account_balance_history
			 WHERE account_id = $1
			 ORDER BY created_at ASC
			 LIMIT 1),
			(SELECT balance FROM mt_accounts WHERE id = $1)
		) as initial_balance
	`
	var balance float64
	err := r.db.GetContext(ctx, &balance, query, accountID)
	return balance, err
}

func (r *AnalyticsRepository) GetConsecutiveStats(ctx context.Context, accountID uuid.UUID, start, end time.Time) (maxWins, maxLosses int, err error) {
	query := `
		WITH profit_signs AS (
			SELECT
				close_time,
				SIGN(profit) as sign,
				ROW_NUMBER() OVER (ORDER BY close_time) -
				ROW_NUMBER() OVER (PARTITION BY SIGN(profit) ORDER BY close_time) as grp
			FROM trade_records
			WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
				AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
		),
		groups AS (
			SELECT sign, grp, COUNT(*) as cnt
			FROM profit_signs
			WHERE sign != 0
			GROUP BY sign, grp
		)
		SELECT
			COALESCE(MAX(CASE WHEN sign = 1 THEN cnt END), 0) as max_wins,
			COALESCE(MAX(CASE WHEN sign = -1 THEN cnt END), 0) as max_losses
		FROM groups
	`
	err = r.db.QueryRowxContext(ctx, query, accountID, start, end).Scan(&maxWins, &maxLosses)
	return
}

func (r *AnalyticsRepository) GetHoldingTimeStats(ctx context.Context, accountID uuid.UUID, start, end time.Time) (avgHoldingSeconds float64, err error) {
	query := `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (close_time - open_time))), 0)
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
	`
	err = r.db.GetContext(ctx, &avgHoldingSeconds, query, accountID, start, end)
	return
}

func (r *AnalyticsRepository) GetDailyReturns(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]float64, error) {
	query := `
		SELECT COALESCE(SUM(profit), 0) as daily_return
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
		GROUP BY DATE(close_time)
		ORDER BY DATE(close_time)
	`
	var returns []float64
	err := r.db.SelectContext(ctx, &returns, query, accountID, start, end)
	return returns, err
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
	err = r.db.QueryRowxContext(ctx, query, accountID, start, end).Scan(&maxDrawdown, &maxDrawdownPercent)
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
	var stats []*struct {
		MonthNum   int     `db:"month_num"`
		Month      string  `db:"month"`
		Profit     float64 `db:"profit"`
		Trades     int     `db:"trades"`
		WinTrades  int     `db:"win_trades"`
		LossTrades int     `db:"loss_trades"`
	}
	err := r.db.SelectContext(ctx, &stats, query, accountID, year)
	if err != nil {
		return nil, err
	}

	monthNames := []string{"1月", "2月", "3月", "4月", "5月", "6月", "7月", "8月", "9月", "10月", "11月", "12月"}
	result := make([]*model.MonthlyPnL, 12)
	for i := 0; i < 12; i++ {
		result[i] = &model.MonthlyPnL{
			Month:      monthNames[i],
			MonthNum:   i + 1,
			Profit:     0,
			Trades:     0,
			WinTrades:  0,
			LossTrades: 0,
		}
	}
	for _, s := range stats {
		idx := s.MonthNum - 1
		result[idx].Profit = s.Profit
		result[idx].Trades = s.Trades
		result[idx].WinTrades = s.WinTrades
		result[idx].LossTrades = s.LossTrades
	}

	return result, nil
}

func (r *AnalyticsRepository) GetDailyPnL(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.DailyPnL, error) {
	result := make([]*model.DailyPnL, 0, 7)
	dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

	initialBalance, err := r.GetAccountInitialBalance(ctx, accountID)
	if err != nil {
		initialBalance = 0
	}

	type dailyStat struct {
		Date       time.Time `db:"date"`
		PnL        float64   `db:"pnl"`
		Trades     int       `db:"trades"`
		Lots       float64   `db:"lots"`
		GrossProfit float64  `db:"gross_profit"`
		GrossLoss  float64   `db:"gross_loss"`
		Cashflow   float64   `db:"cashflow"`
	}
	query := `
		SELECT
			DATE(close_time) AS date,
			COALESCE(SUM(CASE WHEN order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') THEN profit ELSE 0 END), 0) AS pnl,
			COALESCE(COUNT(*) FILTER (WHERE order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')), 0)::int AS trades,
			COALESCE(SUM(CASE WHEN order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') THEN volume ELSE 0 END), 0) AS lots,
			COALESCE(SUM(CASE WHEN order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') AND profit > 0 THEN profit ELSE 0 END), 0) AS gross_profit,
			COALESCE(SUM(CASE WHEN order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') AND profit < 0 THEN ABS(profit) ELSE 0 END), 0) AS gross_loss,
			COALESCE(SUM(CASE WHEN order_type IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') THEN profit ELSE 0 END), 0) AS cashflow
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
		GROUP BY DATE(close_time)
		ORDER BY date ASC
	`
	var stats []dailyStat
	if err := r.db.SelectContext(ctx, &stats, query, accountID, start, end); err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return result, nil
	}

	runningBalance := initialBalance
	rowsWithBalance := make([]struct {
		dailyStat
		Balance float64
	}, 0, len(stats))
	for _, s := range stats {
		runningBalance += s.Cashflow + s.PnL
		rowsWithBalance = append(rowsWithBalance, struct {
			dailyStat
			Balance float64
		}{
			dailyStat: s,
			Balance:   runningBalance,
		})
	}

	selected := make([]struct {
		dailyStat
		Balance float64
	}, 0, 7)
	for i := len(rowsWithBalance) - 1; i >= 0 && len(selected) < 7; i-- {
		row := rowsWithBalance[i]
		if row.Trades > 0 {
			selected = append(selected, row)
		}
	}
	for i := len(selected) - 1; i >= 0; i-- {
		s := selected[i]
		dayNum := int(s.Date.Weekday())
		if dayNum == 0 {
			dayNum = 7
		}
		pf := 0.0
		if s.GrossLoss > 0 {
			pf = s.GrossProfit / s.GrossLoss
		}
		result = append(result, &model.DailyPnL{
			Day:                     dayNames[int(s.Date.Weekday())],
			DayNum:                  dayNum,
			Date:                    s.Date.Format("01-02"),
			PnL:                     s.PnL,
			Trades:                  s.Trades,
			Lots:                    s.Lots,
			Balance:                 s.Balance,
			ProfitFactor:            pf,
			MaxFloatingLossAmount:   0,
			MaxFloatingLossRatio:    0,
			MaxFloatingProfitAmount: 0,
			MaxFloatingProfitRatio:  0,
		})
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
	var stats []*struct {
		Weekday int     `db:"weekday"`
		PnL     float64 `db:"pnl"`
		Trades  int     `db:"trades"`
	}
	if err := r.db.SelectContext(ctx, &stats, query, accountID, start, end); err != nil {
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

	var points []*model.MonthlyAnalysisPoint
	if err := r.db.SelectContext(ctx, &points, query, accountID); err != nil {
		return nil, err
	}
	return points, nil
}

func (r *AnalyticsRepository) GetMonthlyAnalysisYears(ctx context.Context, accountID uuid.UUID) ([]int, error) {
	query := `
		SELECT DISTINCT EXTRACT(YEAR FROM close_time)::int AS year
		FROM trade_records
		WHERE account_id = $1
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
		ORDER BY year ASC
	`

	var years []int
	if err := r.db.SelectContext(ctx, &years, query, accountID); err != nil {
		return nil, err
	}
	return years, nil
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
	var stats []*struct {
		HourStart int     `db:"hour_start"`
		Trades    int     `db:"trades"`
		Lots      float64 `db:"lots"`
		Profit    float64 `db:"profit"`
		GrossProfit float64 `db:"gross_profit"`
		GrossLoss float64 `db:"gross_loss"`
		WinRate   float64 `db:"win_rate"`
	}
	if err := r.db.SelectContext(ctx, &stats, query, accountID, start, end); err != nil {
		return nil, err
	}

	result := make([]*model.HourlyStats, 24)
	for h := 0; h < 24; h++ {
		result[h] = &model.HourlyStats{
			Hour:      fmt.Sprintf("%02d:00", h),
			HourStart: h,
			Trades:    0,
			Profit:    0,
			WinRate:   0,
			AvgPnL:    0,
			Lots:      0,
			Balance:   0,
			ProfitFactor: 0,
			MaxFloatingLossAmount: 0,
			MaxFloatingLossRatio: 0,
			MaxFloatingProfitAmount: 0,
			MaxFloatingProfitRatio: 0,
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
