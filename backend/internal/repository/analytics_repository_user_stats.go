package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserDailyPnL struct {
	Date time.Time `db:"date"`
	PnL  float64   `db:"pnl"`
}

func (r *AnalyticsRepository) GetUserPnLTradesWinLoss(ctx context.Context, userID uuid.UUID, start, end time.Time, enabledOnly bool) (pnl float64, trades int, winTrades int, lossTrades int, sumProfitPos float64, sumLossAbs float64, err error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN tr.order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') THEN tr.profit ELSE 0 END), 0) AS pnl,
			COALESCE(SUM(CASE WHEN tr.order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') THEN 1 ELSE 0 END), 0) AS trades,
			COALESCE(SUM(CASE WHEN tr.order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') AND tr.profit > 0 THEN 1 ELSE 0 END), 0) AS win_trades,
			COALESCE(SUM(CASE WHEN tr.order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') AND tr.profit < 0 THEN 1 ELSE 0 END), 0) AS loss_trades,
			COALESCE(SUM(CASE WHEN tr.order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') AND tr.profit > 0 THEN tr.profit ELSE 0 END), 0) AS sum_profit_pos,
			COALESCE(ABS(SUM(CASE WHEN tr.order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') AND tr.profit < 0 THEN tr.profit ELSE 0 END)), 0) AS sum_loss_abs
		FROM trade_records tr
		JOIN mt_accounts ma ON tr.account_id = ma.id
		WHERE ma.user_id = $1
			AND tr.close_time >= $2 AND tr.close_time <= $3
			AND ($4 = false OR ma.is_disabled = false)
	`
	err = r.db.QueryRowxContext(ctx, query, userID, start, end, enabledOnly).Scan(&pnl, &trades, &winTrades, &lossTrades, &sumProfitPos, &sumLossAbs)
	return
}

func (r *AnalyticsRepository) GetUserDailyPnL(ctx context.Context, userID uuid.UUID, start, end time.Time, enabledOnly bool) ([]UserDailyPnL, error) {
	query := `
		SELECT
			DATE(tr.close_time) AS date,
			COALESCE(SUM(CASE WHEN tr.order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') THEN tr.profit ELSE 0 END), 0) AS pnl
		FROM trade_records tr
		JOIN mt_accounts ma ON tr.account_id = ma.id
		WHERE ma.user_id = $1
			AND tr.close_time >= $2 AND tr.close_time <= $3
			AND ($4 = false OR ma.is_disabled = false)
		GROUP BY DATE(tr.close_time)
		ORDER BY date ASC
	`
	var rows []UserDailyPnL
	err := r.db.SelectContext(ctx, &rows, query, userID, start, end, enabledOnly)
	return rows, err
}

func (r *AnalyticsRepository) GetUserConsecutiveStats(ctx context.Context, userID uuid.UUID, start, end time.Time, enabledOnly bool) (maxWins, maxLosses int, err error) {
	query := `
		WITH profit_signs AS (
			SELECT
				tr.close_time,
				SIGN(tr.profit) AS sign,
				ROW_NUMBER() OVER (ORDER BY tr.close_time) -
				ROW_NUMBER() OVER (PARTITION BY SIGN(tr.profit) ORDER BY tr.close_time) AS grp
			FROM trade_records tr
			JOIN mt_accounts ma ON tr.account_id = ma.id
			WHERE ma.user_id = $1
				AND tr.close_time >= $2 AND tr.close_time <= $3
				AND tr.order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
				AND ($4 = false OR ma.is_disabled = false)
		),
		groups AS (
			SELECT sign, grp, COUNT(*) AS cnt
			FROM profit_signs
			WHERE sign != 0
			GROUP BY sign, grp
		)
		SELECT
			COALESCE(MAX(CASE WHEN sign = 1 THEN cnt END), 0) AS max_wins,
			COALESCE(MAX(CASE WHEN sign = -1 THEN cnt END), 0) AS max_losses
		FROM groups
	`
	err = r.db.QueryRowxContext(ctx, query, userID, start, end, enabledOnly).Scan(&maxWins, &maxLosses)
	return
}
