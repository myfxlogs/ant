package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/google/uuid"

	"anttrader/internal/model"
)

type snapPoint struct {
	RecordedAt time.Time
	Equity     decimal.Decimal
}

type hourlyData struct {
	Hour              time.Time
	Profit            decimal.Decimal
	DepositWithdrawal decimal.Decimal
}

// GetHourlyEquityCurve returns equity curve points grouped by hour for intraday display.
func (r *AnalyticsRepository) GetHourlyEquityCurve(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.EquityPoint, error) {
	initialBalance, err := r.GetBalanceAtTime(ctx, accountID, start)
	if err != nil {
		return nil, fmt.Errorf("get balance at start time: %w", err)
	}

	snapshots, err := r.getHourlySnapshots(ctx, accountID, start, end)
	if err != nil {
		return nil, err
	}

	hourlyDataList, err := r.getHourlyTradeData(ctx, accountID, start, end)
	if err != nil {
		return nil, err
	}

	return r.buildHourlyEquityResult(initialBalance, snapshots, hourlyDataList, start, end)
}

func (r *AnalyticsRepository) getHourlySnapshots(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]snapPoint, error) {
	rows, err := r.db.Query(ctx,
		`SELECT recorded_at, equity FROM account_balance_history
		 WHERE account_id=$1 AND recorded_at>=$2 AND recorded_at<=$3 ORDER BY recorded_at ASC`,
		accountID, start, end)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []snapPoint
	for rows.Next() {
		var sp snapPoint
		if err := rows.Scan(&sp.RecordedAt, &sp.Equity); err != nil { return nil, err }
		out = append(out, sp)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) getHourlyTradeData(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]hourlyData, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DATE_TRUNC('hour', close_time),
		        COALESCE(SUM(CASE WHEN LOWER(order_type) NOT IN ('balance','credit') THEN profit ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN LOWER(order_type) IN ('balance','credit') THEN profit ELSE 0 END),0)
		 FROM trade_records WHERE account_id=$1 AND close_time>=$2 AND close_time<=$3
		 GROUP BY DATE_TRUNC('hour', close_time) ORDER BY hour ASC`,
		accountID, start, end)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []hourlyData
	for rows.Next() {
		var hd hourlyData
		if err := rows.Scan(&hd.Hour, &hd.Profit, &hd.DepositWithdrawal); err != nil { return nil, err }
		out = append(out, hd)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) buildHourlyEquityResult(
	initialBalance decimal.Decimal,
	snapshots []snapPoint,
	hourlyDataList []hourlyData,
	start, end time.Time,
) ([]*model.EquityPoint, error) {
	var result []*model.EquityPoint
	runningBalance := initialBalance
	currentEquity := initialBalance
	hourCursor := time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), 0, 0, 0, start.Location())
	endHour := time.Date(end.Year(), end.Month(), end.Day(), end.Hour(), 0, 0, 0, end.Location())
	dataIdx, snapIdx := 0, 0

	for !hourCursor.After(endHour) {
		profit, deposit := decimal.Zero, decimal.Zero
		if dataIdx < len(hourlyDataList) && hourlyDataList[dataIdx].Hour.Equal(hourCursor) {
			profit, deposit = hourlyDataList[dataIdx].Profit, hourlyDataList[dataIdx].DepositWithdrawal
			dataIdx++
		}
		runningBalance = runningBalance.Add(deposit).Add(profit)
		hourEnd := hourCursor.Add(time.Hour)
		for snapIdx < len(snapshots) && snapshots[snapIdx].RecordedAt.Before(hourEnd) {
			currentEquity = snapshots[snapIdx].Equity
			snapIdx++
		}
		result = append(result, &model.EquityPoint{
			Date:    hourCursor.Format("2006-01-02 15:04"),
			Equity:  currentEquity,
			Balance: runningBalance,
			Profit:  profit,
		})
		hourCursor = hourCursor.Add(time.Hour)
	}
	if len(result) == 0 {
		result = append(result, &model.EquityPoint{
			Date:    time.Now().Format("2006-01-02 15:04"),
			Equity:  initialBalance,
			Balance: initialBalance,
			Profit:  decimal.Zero,
		})
	}
	return result, nil
}
