package repository

import (
	"github.com/shopspring/decimal"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

func (r *AnalyticsRepository) GetEquityCurve(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.EquityPoint, error) {
	initialBalance, err := r.GetBalanceAtTime(ctx, accountID, start)
	if err != nil {
		return nil, fmt.Errorf("get balance at start time: %w", err)
	}

	unrealizedPnL, hasInitSnap, err := r.getInitialUnrealizedPnL(ctx, accountID, start, initialBalance)
	if err != nil {
		return nil, err
	}

	dailyDataList, err := r.getDailyTradeData(ctx, accountID, start, end)
	if err != nil {
		return nil, err
	}

	snapshots, err := r.getDailySnapshots(ctx, accountID, start, end)
	if err != nil {
		return nil, err
	}

	if !hasInitSnap && len(snapshots) > 0 {
		firstSnap := snapshots[0]
		firstSnapDate := time.Date(firstSnap.Date.Year(), firstSnap.Date.Month(), firstSnap.Date.Day(), 0, 0, 0, 0, start.Location())
		balAtSnap := initialBalance
		for _, dd := range dailyDataList {
			ddDate := time.Date(dd.Date.Year(), dd.Date.Month(), dd.Date.Day(), 0, 0, 0, 0, start.Location())
			if ddDate.After(firstSnapDate) { break }
			balAtSnap = balAtSnap.Add(dd.Profit).Add(dd.DepositWithdrawal)
		}
		unrealizedPnL = firstSnap.Equity.Sub(balAtSnap)
	}

	return r.buildEquityCurveResult(ctx, accountID, initialBalance, unrealizedPnL, dailyDataList, snapshots, start, end)
}

// getInitialUnrealizedPnL fetches the equity snapshot before the period start.
func (r *AnalyticsRepository) getInitialUnrealizedPnL(ctx context.Context, accountID uuid.UUID, start time.Time, initialBalance decimal.Decimal) (decimal.Decimal, bool, error) {
	var unrealizedPnL decimal.Decimal
	hasInitSnap := false
	var initSnapEquity decimal.Decimal
	err := r.db.QueryRow(ctx,
		`SELECT equity FROM account_balance_history WHERE account_id = $1 AND recorded_at <= $2 ORDER BY recorded_at DESC LIMIT 1`,
		accountID, start,
	).Scan(&initSnapEquity)
	if err == nil {
		unrealizedPnL = initSnapEquity.Sub(initialBalance)
		hasInitSnap = true
	}
	return unrealizedPnL, hasInitSnap, nil
}

type dailyData struct {
	Date              time.Time
	Profit            decimal.Decimal
	DepositWithdrawal decimal.Decimal
}

func (r *AnalyticsRepository) getDailyTradeData(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]dailyData, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DATE(close_time), COALESCE(SUM(CASE WHEN LOWER(order_type) NOT IN ('balance','credit') THEN profit ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN LOWER(order_type) IN ('balance','credit') THEN profit ELSE 0 END),0)
		 FROM trade_records WHERE account_id=$1 AND close_time>=$2 AND close_time<=$3
		 GROUP BY DATE(close_time) ORDER BY DATE(close_time) ASC`,
		accountID, start, end)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []dailyData
	for rows.Next() {
		var dd dailyData
		if err := rows.Scan(&dd.Date, &dd.Profit, &dd.DepositWithdrawal); err != nil { return nil, err }
		out = append(out, dd)
	}
	return out, rows.Err()
}

type dailySnapshot struct {
	Date   time.Time
	Equity decimal.Decimal
}

func (r *AnalyticsRepository) getDailySnapshots(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]dailySnapshot, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT ON (DATE(recorded_at)) DATE(recorded_at), equity
		 FROM account_balance_history WHERE account_id=$1 AND recorded_at>=$2 AND recorded_at<=$3
		 ORDER BY DATE(recorded_at), recorded_at DESC`,
		accountID, start, end)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []dailySnapshot
	for rows.Next() {
		var s dailySnapshot
		if err := rows.Scan(&s.Date, &s.Equity); err != nil { return nil, err }
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) buildEquityCurveResult(
	ctx context.Context,
	accountID uuid.UUID,
	initialBalance, unrealizedPnL decimal.Decimal,
	dailyDataList []dailyData,
	snapshots []dailySnapshot,
	start, end time.Time,
) ([]*model.EquityPoint, error) {
	var result []*model.EquityPoint
	runningBalance := initialBalance
	dayCursor := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
	dataIdx, snapIdx := 0, 0

	for !dayCursor.After(endDay) {
		profit, deposit := decimal.Zero, decimal.Zero
		if dataIdx < len(dailyDataList) {
			ddDate := time.Date(dailyDataList[dataIdx].Date.Year(), dailyDataList[dataIdx].Date.Month(), dailyDataList[dataIdx].Date.Day(), 0, 0, 0, 0, start.Location())
			if ddDate.Equal(dayCursor) {
				profit, deposit = dailyDataList[dataIdx].Profit, dailyDataList[dataIdx].DepositWithdrawal
				dataIdx++
			}
		}
		runningBalance = runningBalance.Add(deposit).Add(profit)
		if snapIdx < len(snapshots) {
			snapDate := time.Date(snapshots[snapIdx].Date.Year(), snapshots[snapIdx].Date.Month(), snapshots[snapIdx].Date.Day(), 0, 0, 0, 0, start.Location())
			if snapDate.Equal(dayCursor) {
				unrealizedPnL = snapshots[snapIdx].Equity.Sub(runningBalance)
				snapIdx++
			}
		}
		equity := runningBalance.Add(unrealizedPnL)
		result = append(result, &model.EquityPoint{
			Date:    dayCursor.Format("2006-01-02"),
			Equity:  equity,
			Balance: runningBalance,
			Profit:  profit,
		})
		dayCursor = dayCursor.AddDate(0, 0, 1)
	}
	if len(result) == 0 {
		eq := initialBalance
		var snapEq decimal.Decimal
		if err := r.db.QueryRow(ctx, `SELECT equity FROM account_balance_history WHERE account_id=$1 ORDER BY recorded_at DESC LIMIT 1`, accountID).Scan(&snapEq); err == nil {
			eq = snapEq
		}
		result = append(result, &model.EquityPoint{
			Date:    time.Now().Format("2006-01-02"),
			Equity:  eq,
			Balance: initialBalance,
			Profit:  decimal.Zero,
		})
	}
	return result, nil
}

// CurrentAccountMetrics holds the live balance/equity/profit from mt_accounts.
type CurrentAccountMetrics struct {
	Balance decimal.Decimal
	Equity  decimal.Decimal
	Profit  decimal.Decimal
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
