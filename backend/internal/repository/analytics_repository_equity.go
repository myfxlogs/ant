package repository

import (
	"github.com/shopspring/decimal"
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

func (r *AnalyticsRepository) GetEquityCurve(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.EquityPoint, error) {
	initialBalance, err := r.GetBalanceAtTime(ctx, accountID, start)
	if err != nil {
		return nil, fmt.Errorf("get balance at start time: %w", err)
	}

	var unrealizedPnL float64
	hasInitSnap := false
	initSnapQuery := `
		SELECT equity FROM account_balance_history
		WHERE account_id = $1 AND recorded_at <= $2
		ORDER BY recorded_at DESC LIMIT 1
	`
	var initSnapEquity float64
	if err := r.db.QueryRow(ctx, initSnapQuery, accountID, start).Scan(&initSnapEquity); err == nil {
		unrealizedPnL = initSnapEquity - initialBalance
		hasInitSnap = true
	}

	tradeQuery := `
		SELECT
			DATE(close_time) as date,
			COALESCE(SUM(CASE WHEN LOWER(order_type) NOT IN ('balance', 'credit') THEN profit ELSE 0 END), 0) as profit,
			COALESCE(SUM(CASE WHEN LOWER(order_type) IN ('balance', 'credit') THEN profit ELSE 0 END), 0) as deposit_withdrawal
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
		GROUP BY DATE(close_time)
		ORDER BY date ASC
	`
	type dailyData struct {
		Date              time.Time `db:"date"`
		Profit            float64   `db:"profit"`
		DepositWithdrawal float64   `db:"deposit_withdrawal"`
	}
	rows, err := r.db.Query(ctx, tradeQuery, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dailyDataList []dailyData
	for rows.Next() {
		var dd dailyData
		if err := rows.Scan(&dd.Date, &dd.Profit, &dd.DepositWithdrawal); err != nil {
			return nil, err
		}
		dailyDataList = append(dailyDataList, dd)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	snapshotQuery := `
		SELECT DISTINCT ON (DATE(recorded_at))
			DATE(recorded_at) as date,
			equity
		FROM account_balance_history
		WHERE account_id = $1 AND recorded_at >= $2 AND recorded_at <= $3
		ORDER BY DATE(recorded_at), recorded_at DESC
	`
	type dailySnapshot struct {
		Date   time.Time `db:"date"`
		Equity float64   `db:"equity"`
	}
	snapRows, err := r.db.Query(ctx, snapshotQuery, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer snapRows.Close()
	var snapshots []dailySnapshot
	for snapRows.Next() {
		var s dailySnapshot
		if err := snapRows.Scan(&s.Date, &s.Equity); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	if err := snapRows.Err(); err != nil {
		return nil, err
	}

	if !hasInitSnap && len(snapshots) > 0 {
		firstSnap := snapshots[0]
		firstSnapDate := time.Date(firstSnap.Date.Year(), firstSnap.Date.Month(), firstSnap.Date.Day(), 0, 0, 0, 0, start.Location())
		balAtSnap := initialBalance
		for _, dd := range dailyDataList {
			ddDate := time.Date(dd.Date.Year(), dd.Date.Month(), dd.Date.Day(), 0, 0, 0, 0, start.Location())
			if ddDate.After(firstSnapDate) {
				break
			}
			balAtSnap += dd.Profit + dd.DepositWithdrawal
		}
		unrealizedPnL = firstSnap.Equity - balAtSnap
	}

	var result []*model.EquityPoint
	runningBalance := initialBalance

	dayCursor := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
	dataIdx := 0
	snapIdx := 0

	for !dayCursor.After(endDay) {
		profit := 0.0
		deposit := 0.0
		if dataIdx < len(dailyDataList) {
			ddDate := time.Date(dailyDataList[dataIdx].Date.Year(), dailyDataList[dataIdx].Date.Month(), dailyDataList[dataIdx].Date.Day(), 0, 0, 0, 0, start.Location())
			if ddDate.Equal(dayCursor) {
				profit = dailyDataList[dataIdx].Profit
				deposit = dailyDataList[dataIdx].DepositWithdrawal
				dataIdx++
			}
		}
		runningBalance += deposit + profit

		if snapIdx < len(snapshots) {
			snapDate := time.Date(snapshots[snapIdx].Date.Year(), snapshots[snapIdx].Date.Month(), snapshots[snapIdx].Date.Day(), 0, 0, 0, 0, start.Location())
			if snapDate.Equal(dayCursor) {
				unrealizedPnL = snapshots[snapIdx].Equity - runningBalance
				snapIdx++
			}
		}

		result = append(result, &model.EquityPoint{
			Date:    dayCursor.Format("2006-01-02"),
			Equity:  decimal.NewFromFloat(math.Round((runningBalance+unrealizedPnL)*100) / 100),
			Balance:  decimal.NewFromFloat(math.Round(runningBalance*100) / 100),
			Profit:  decimal.NewFromFloat(math.Round(profit*100) / 100),
		})
		dayCursor = dayCursor.AddDate(0, 0, 1)
	}

	if len(result) == 0 {
		eq := initialBalance
		snapQuery := `SELECT equity FROM account_balance_history WHERE account_id = $1 ORDER BY recorded_at DESC LIMIT 1`
		var snapEq float64
		if err := r.db.QueryRow(ctx, snapQuery, accountID).Scan(&snapEq); err == nil {
			eq = snapEq
		}
		result = append(result, &model.EquityPoint{
			Date:    time.Now().Format("2006-01-02"),
			Equity:  decimal.NewFromFloat(math.Round(eq*100) / 100),
			Balance:  decimal.NewFromFloat(math.Round(initialBalance*100) / 100),
			Profit:  decimal.Zero,
		})
	}

	return result, nil
}

// CurrentAccountMetrics holds the live balance/equity/profit from mt_accounts.
type CurrentAccountMetrics struct {
	Balance float64
	Equity  float64
	Profit  float64
}

// GetCurrentAccountMetrics returns the current live balance, equity, and profit
// for an account directly from mt_accounts.
func (r *AnalyticsRepository) GetCurrentAccountMetrics(ctx context.Context, accountID uuid.UUID) (*CurrentAccountMetrics, error) {
	var m CurrentAccountMetrics
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(balance, 0), COALESCE(equity, 0), COALESCE(profit, 0)
		 FROM mt_accounts WHERE id = $1`, accountID,
	).Scan(&m.Balance, &m.Equity, &m.Profit)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
