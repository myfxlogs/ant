package repository

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"
	"github.com/google/uuid"

	"anttrader/internal/model"
)

// GetHourlyEquityCurve returns equity curve points grouped by hour for intraday display.
func (r *AnalyticsRepository) GetHourlyEquityCurve(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.EquityPoint, error) {
	initialBalance, err := r.GetBalanceAtTime(ctx, accountID, start)
	if err != nil {
		return nil, fmt.Errorf("get balance at start time: %w", err)
	}

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
	currentEquity := initialBalance

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

		hourEnd := hourCursor.Add(time.Hour)
		for snapIdx < len(snapshots) && snapshots[snapIdx].RecordedAt.Before(hourEnd) {
			currentEquity = snapshots[snapIdx].Equity
			snapIdx++
		}

		result = append(result, &model.EquityPoint{
			Date:    hourCursor.Format("2006-01-02 15:04"),
			Equity:  decimal.NewFromFloat(math.Round(currentEquity*100) / 100),
			Balance:  decimal.NewFromFloat(math.Round(runningBalance*100) / 100),
			Profit:  decimal.NewFromFloat(math.Round(profit*100) / 100),
		})
		hourCursor = hourCursor.Add(time.Hour)
	}

	if len(result) == 0 {
		result = append(result, &model.EquityPoint{
			Date:    time.Now().Format("2006-01-02 15:04"),
			Equity:  decimal.NewFromFloat(math.Round(initialBalance*100) / 100),
			Balance:  decimal.NewFromFloat(math.Round(initialBalance*100) / 100),
			Profit:  decimal.Zero,
		})
	}

	return result, nil
}
