package repository

import (
	"github.com/shopspring/decimal"
	"context"
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
	rows, err := r.db.Query(ctx, query, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stats []*model.SymbolStats
	for rows.Next() {
		s := &model.SymbolStats{}
		// Scan monetary fields as decimal.Decimal to preserve NUMERIC precision.
		if err := rows.Scan(&s.Symbol, &s.TotalTrades, &s.WinningTrades, &s.LosingTrades, &s.WinRate, &s.TotalProfit, &s.TotalLoss, &s.NetProfit, &s.ProfitFactor, &s.AverageProfit, &s.TotalVolume, &s.AverageVolume, &s.LargestWin, &s.LargestLoss, &s.AverageHoldingTime); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
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
		Date   time.Time       `db:"date"`
		Profit decimal.Decimal `db:"profit"`
	}
	rows, err := r.db.Query(ctx, query, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dailyProfits []dailyProfit
	for rows.Next() {
		var dp dailyProfit
		if err := rows.Scan(&dp.Date, &dp.Profit); err != nil {
			return nil, err
		}
		dailyProfits = append(dailyProfits, dp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var result []*model.DailyEquity
	runningBalance := decimal.Zero
	for _, dp := range dailyProfits {
		runningBalance = runningBalance.Add(dp.Profit)
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
	query := `SELECT balance FROM mt_accounts WHERE deleted_at IS NULL AND id = $1`
	var balance float64
	err := r.db.QueryRow(ctx, query, accountID).Scan(&balance)
	return balance, err
}

func (r *AnalyticsRepository) GetAccountInitialBalance(ctx context.Context, accountID uuid.UUID) (float64, error) {
	query := `
		SELECT COALESCE(
			(SELECT balance FROM account_balance_history
			 WHERE account_id = $1
			 ORDER BY created_at ASC
			 LIMIT 1),
			(SELECT balance FROM mt_accounts WHERE deleted_at IS NULL AND id = $1)
		) as initial_balance
	`
	var balance float64
	err := r.db.QueryRow(ctx, query, accountID).Scan(&balance)
	return balance, err
}

// GetBalanceAtTime returns the account balance closest to but not after `at`.
func (r *AnalyticsRepository) GetBalanceAtTime(ctx context.Context, accountID uuid.UUID, at time.Time) (decimal.Decimal, error) {
	query := `
		SELECT balance FROM account_balance_history
		WHERE account_id = $1 AND recorded_at <= $2
		ORDER BY recorded_at DESC
		LIMIT 1
	`
	var balance decimal.Decimal
	err := r.db.QueryRow(ctx, query, accountID, at).Scan(&balance)
	if err == nil {
		return balance, nil
	}

	// Fallback: compute from all trade records before `at`.
	query = `SELECT COALESCE(SUM(profit), 0) FROM trade_records WHERE account_id = $1 AND close_time <= $2`
	err = r.db.QueryRow(ctx, query, accountID, at).Scan(&balance)
	if err != nil {
		return decimal.Zero, err
	}
	return balance, nil
}

// RecordBalanceSnapshot inserts a periodic equity/balance snapshot.
func (r *AnalyticsRepository) RecordBalanceSnapshot(ctx context.Context, accountID, userID uuid.UUID, balance, equity, margin, freeMargin decimal.Decimal) error {
	query := `
		INSERT INTO account_balance_history (account_id, user_id, balance, equity, margin, free_margin, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err := r.db.Exec(ctx, query, accountID, userID, balance, equity, margin, freeMargin)
	return err
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
	err = r.db.QueryRow(ctx, query, accountID, start, end).Scan(&maxWins, &maxLosses)
	return
}

func (r *AnalyticsRepository) GetHoldingTimeStats(ctx context.Context, accountID uuid.UUID, start, end time.Time) (avgHoldingSeconds float64, err error) {
	query := `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (close_time - open_time))), 0)
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
	`
	err = r.db.QueryRow(ctx, query, accountID, start, end).Scan(&avgHoldingSeconds)
	return
}

