package repository

import (
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

	// Initialize unrealized P&L from snapshot at or before period start.
	// hasInitSnap distinguishes "no snapshot found" from "legitimate zero PnL".
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

	// Daily trade P&L.
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

	// Daily equity snapshots (ground truth from broker, includes floating P&L).
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

	// If no snapshot at period start, seed unrealizedPnL from the first
	// snapshot within the period so the equity line is meaningful from day 1.
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
		// Balance = cumulative from trade_records (internally consistent).
		runningBalance += deposit + profit

		// At a snapshot: correct Equity to broker's ground truth.
		// Between snapshots: unrealizedPnL stays constant, Equity moves with Balance.
		if snapIdx < len(snapshots) {
			snapDate := time.Date(snapshots[snapIdx].Date.Year(), snapshots[snapIdx].Date.Month(), snapshots[snapIdx].Date.Day(), 0, 0, 0, 0, start.Location())
			if snapDate.Equal(dayCursor) {
				unrealizedPnL = snapshots[snapIdx].Equity - runningBalance
				snapIdx++
			}
		}

		result = append(result, &model.EquityPoint{
			Date:    dayCursor.Format("2006-01-02"),
			Equity:  math.Round((runningBalance+unrealizedPnL)*100) / 100,
			Balance: math.Round(runningBalance*100) / 100,
			Profit:  math.Round(profit*100) / 100,
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
			Equity:  math.Round(eq*100) / 100,
			Balance: math.Round(initialBalance*100) / 100,
			Profit:  0,
		})
	}

	return result, nil
}

	// GetHourlyEquityCurve returns equity curve points grouped by hour for intraday display.
func (r *AnalyticsRepository) GetHourlyEquityCurve(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.EquityPoint, error) {
	initialBalance, err := r.GetBalanceAtTime(ctx, accountID, start)
	if err != nil {
		return nil, fmt.Errorf("get balance at start time: %w", err)
	}

	// All snapshots in range — drive equity variation hour by hour.
	snapshotQuery := `
		SELECT recorded_at, equity FROM account_balance_history
		WHERE account_id = $1 AND recorded_at >= $2 AND recorded_at <= $3
		ORDER BY recorded_at ASC
	`
	type snapPoint struct {
		RecordedAt time.Time `db:"recorded_at"`
		Equity     float64   `db:"equity"`
	}
	snapRows, err := r.db.Query(ctx, snapshotQuery, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer snapRows.Close()
	var snapshots []snapPoint
	for snapRows.Next() {
		var sp snapPoint
		if err := snapRows.Scan(&sp.RecordedAt, &sp.Equity); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, sp)
	}
	if err := snapRows.Err(); err != nil {
		return nil, err
	}

	// Hourly trade data.
	query := `
		SELECT
			DATE_TRUNC('hour', close_time) as hour,
			COALESCE(SUM(CASE WHEN LOWER(order_type) NOT IN ('balance', 'credit') THEN profit ELSE 0 END), 0) as profit,
			COALESCE(SUM(CASE WHEN LOWER(order_type) IN ('balance', 'credit') THEN profit ELSE 0 END), 0) as deposit_withdrawal
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
		GROUP BY DATE_TRUNC('hour', close_time)
		ORDER BY hour ASC
	`
	type hourlyData struct {
		Hour              time.Time `db:"hour"`
		Profit            float64   `db:"profit"`
		DepositWithdrawal float64   `db:"deposit_withdrawal"`
	}
	rows, err := r.db.Query(ctx, query, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hourlyDataList []hourlyData
	for rows.Next() {
		var hd hourlyData
		if err := rows.Scan(&hd.Hour, &hd.Profit, &hd.DepositWithdrawal); err != nil {
			return nil, err
		}
		hourlyDataList = append(hourlyDataList, hd)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var result []*model.EquityPoint
	runningBalance := initialBalance
	currentEquity := initialBalance // before first snapshot, equity = balance (no floating P&L data)

	hourCursor := time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), 0, 0, 0, start.Location())
	endHour := time.Date(end.Year(), end.Month(), end.Day(), end.Hour(), 0, 0, 0, end.Location())
	dataIdx := 0
	snapIdx := 0

	for !hourCursor.After(endHour) {
		profit := 0.0
		deposit := 0.0
		if dataIdx < len(hourlyDataList) && hourlyDataList[dataIdx].Hour.Equal(hourCursor) {
			profit = hourlyDataList[dataIdx].Profit
			deposit = hourlyDataList[dataIdx].DepositWithdrawal
			dataIdx++
		}
		runningBalance += deposit + profit

		// Advance snapshots: consume all snapshots recorded before the END of this hour.
		// The latest snapshot's equity becomes the equity for this hour.
		hourEnd := hourCursor.Add(time.Hour)
		for snapIdx < len(snapshots) && snapshots[snapIdx].RecordedAt.Before(hourEnd) {
			currentEquity = snapshots[snapIdx].Equity
			snapIdx++
		}

		result = append(result, &model.EquityPoint{
			Date:    hourCursor.Format("2006-01-02 15:04"),
			Equity:  math.Round(currentEquity*100) / 100,
			Balance: math.Round(runningBalance*100) / 100,
			Profit:  math.Round(profit*100) / 100,
		})
		hourCursor = hourCursor.Add(time.Hour)
	}

	if len(result) == 0 {
		result = append(result, &model.EquityPoint{
			Date:    time.Now().Format("2006-01-02 15:04"),
			Equity:  math.Round(initialBalance*100) / 100,
			Balance: math.Round(initialBalance*100) / 100,
			Profit:  0,
		})
	}

	return result, nil
}
