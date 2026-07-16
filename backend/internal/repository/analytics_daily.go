package repository

import (
	"github.com/shopspring/decimal"
	"context"
	"time"

	"github.com/google/uuid"

	"alphaforge/internal/model"
)

// dailyTradeStat holds raw daily PnL data from a single query row.
type dailyTradeStat struct {
	Date        time.Time       `db:"date"`
	PnL         decimal.Decimal `db:"pnl"`
	Trades      int             `db:"trades"`
	Lots        decimal.Decimal `db:"lots"`
	GrossProfit decimal.Decimal `db:"gross_profit"`
	GrossLoss   decimal.Decimal `db:"gross_loss"`
	Cashflow    decimal.Decimal `db:"cashflow"`
}

// dailyMaxFloatingRow holds max intraday floating loss/profit for a single date.
type dailyMaxFloatingRow struct {
	Date              string          `db:"trade_date"`
	MaxFloatingLoss   decimal.Decimal `db:"max_floating_loss"`
	MaxFloatingProfit decimal.Decimal `db:"max_floating_profit"`
}

// GetDailyPnL returns up to 7 recent trading days with PnL, balance, and profit factor.
// Also computes max intraday floating loss/profit from individual trade records.
func (r *AnalyticsRepository) GetDailyPnL(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.DailyPnL, error) {
	initialBalance, err := r.GetBalanceAtTime(ctx, accountID, start)
	if err != nil {
		initialBalance = decimal.Zero
	}

	stats, err := r.fetchDailyStats(ctx, accountID, start, end)
	if err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return make([]*model.DailyPnL, 0, 7), nil
	}

	// Fetch max intraday floating loss/profit via window-function query.
	maxFloating, err := r.fetchDailyMaxFloating(ctx, accountID, start, end)
	if err != nil {
		maxFloating = nil // warn-no-block: computeResult handles nil map
	}

	return computeDailyPnLResult(stats, initialBalance, maxFloating), nil
}

// fetchDailyStats runs the daily PnL aggregation query.
func (r *AnalyticsRepository) fetchDailyStats(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]dailyTradeStat, error) {
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
	rows, err := r.db.Query(ctx, query, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stats []dailyTradeStat
	for rows.Next() {
		var s dailyTradeStat
		if err := rows.Scan(&s.Date, &s.PnL, &s.Trades, &s.Lots, &s.GrossProfit, &s.GrossLoss, &s.Cashflow); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// fetchDailyMaxFloating computes max intraday floating loss/profit for each day
// using SQL window functions over trade-level running P&L.
//
// Max floating loss = max(peak - current) — the deepest drawdown from the
// running peak within the day.
// Max floating profit = max(current - trough) — the highest runup from the
// running trough within the day.
func (r *AnalyticsRepository) fetchDailyMaxFloating(ctx context.Context, accountID uuid.UUID, start, end time.Time) (map[string]dailyMaxFloatingRow, error) {
	query := `
		WITH daily_trades AS (
			SELECT
				DATE(close_time)          AS trade_date,
				close_time,
				id,
				profit,
				SUM(profit) OVER (
					PARTITION BY DATE(close_time)
					ORDER BY close_time, id
				)                          AS running_pnl
			FROM trade_records
			WHERE account_id = $1
				AND close_time >= $2 AND close_time <= $3
				AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
		),
		with_extremes AS (
			SELECT
				trade_date,
				running_pnl,
				MIN(running_pnl) OVER (
					PARTITION BY trade_date
					ORDER BY close_time, id
				)                          AS running_min,
				MAX(running_pnl) OVER (
					PARTITION BY trade_date
					ORDER BY close_time, id
				)                          AS running_max
			FROM daily_trades
		)
		SELECT
			trade_date::text                                          AS trade_date,
			COALESCE(MAX(running_max - running_pnl), 0)              AS max_floating_loss,
			COALESCE(MAX(running_pnl - running_min), 0)              AS max_floating_profit
		FROM with_extremes
		GROUP BY trade_date
	`
	rows, err := r.db.Query(ctx, query, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]dailyMaxFloatingRow)
	for rows.Next() {
		var row dailyMaxFloatingRow
		if err := rows.Scan(&row.Date, &row.MaxFloatingLoss, &row.MaxFloatingProfit); err != nil {
			return nil, err
		}
		result[row.Date] = row
	}
	return result, rows.Err()
}

// computeDailyPnLResult computes running balances, selects the last 7 trading
// days, merges max-floating data, and builds the output slice.
func computeDailyPnLResult(stats []dailyTradeStat, initialBalance decimal.Decimal, maxFloating map[string]dailyMaxFloatingRow) []*model.DailyPnL {
	dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

	// Compute running balances.
	type rowWithBalance struct {
		dailyTradeStat
		Balance decimal.Decimal
	}
	running := initialBalance
	rowsWithBalance := make([]rowWithBalance, 0, len(stats))
	for _, s := range stats {
		running = running.Add(s.Cashflow).Add(s.PnL)
		rowsWithBalance = append(rowsWithBalance, rowWithBalance{dailyTradeStat: s, Balance: running})
	}

	// Select last 7 trading days (days with at least 1 trade).
	selected := make([]rowWithBalance, 0, 7)
	for i := len(rowsWithBalance) - 1; i >= 0 && len(selected) < 7; i-- {
		if rowsWithBalance[i].Trades > 0 {
			selected = append(selected, rowsWithBalance[i])
		}
	}

	// Reverse back to chronological order and build results.
	result := make([]*model.DailyPnL, 0, len(selected))
	for i := len(selected) - 1; i >= 0; i-- {
		s := selected[i]
		dayNum := int(s.Date.Weekday())
		if dayNum == 0 {
			dayNum = 7
		}
		pf := decimal.Zero
		if s.GrossLoss.IsPositive() {
			pf = s.GrossProfit.Div(s.GrossLoss)
		}

		// Merge max intraday floating data.
		dateKey := s.Date.Format("2006-01-02")
		lossAmt := decimal.Zero
		lossRatio := decimal.Zero
		profitAmt := decimal.Zero
		profitRatio := decimal.Zero
		if mf, ok := maxFloating[dateKey]; ok {
			lossAmt = mf.MaxFloatingLoss
			profitAmt = mf.MaxFloatingProfit
			if s.Balance.IsPositive() {
				lossRatio = lossAmt.Div(s.Balance).Mul(decimal.NewFromInt(100))
				profitRatio = profitAmt.Div(s.Balance).Mul(decimal.NewFromInt(100))
			}
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
			MaxFloatingLossAmount:   lossAmt,
			MaxFloatingLossRatio:    lossRatio,
			MaxFloatingProfitAmount: profitAmt,
			MaxFloatingProfitRatio:  profitRatio,
		})
	}
	return result
}
