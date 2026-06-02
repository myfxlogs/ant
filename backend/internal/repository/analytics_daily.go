package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

// dailyTradeStat holds raw daily PnL data from a single query row.
type dailyTradeStat struct {
	Date        time.Time `db:"date"`
	PnL         float64   `db:"pnl"`
	Trades      int       `db:"trades"`
	Lots        float64   `db:"lots"`
	GrossProfit float64   `db:"gross_profit"`
	GrossLoss   float64   `db:"gross_loss"`
	Cashflow    float64   `db:"cashflow"`
}

// GetDailyPnL returns up to 7 recent trading days with PnL, balance, and profit factor.
func (r *AnalyticsRepository) GetDailyPnL(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.DailyPnL, error) {
	initialBalance, err := r.GetBalanceAtTime(ctx, accountID, start)
	if err != nil {
		initialBalance = 0
	}

	stats, err := r.fetchDailyStats(ctx, accountID, start, end)
	if err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return make([]*model.DailyPnL, 0, 7), nil
	}

	return computeDailyPnLResult(stats, initialBalance), nil
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

// computeDailyPnLResult computes running balances, selects the last 7 trading
// days, and builds the output slice.
func computeDailyPnLResult(stats []dailyTradeStat, initialBalance float64) []*model.DailyPnL {
	dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

	// Compute running balances.
	type rowWithBalance struct {
		dailyTradeStat
		Balance float64
	}
	running := initialBalance
	rowsWithBalance := make([]rowWithBalance, 0, len(stats))
	for _, s := range stats {
		running += s.Cashflow + s.PnL
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
	return result
}
