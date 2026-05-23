package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

func (r *AnalyticsRepository) GetEquityCurve(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.EquityPoint, error) {
	// 获取账户初始余额
	initialBalance, err := r.GetAccountInitialBalance(ctx, accountID)
	if err != nil {
		initialBalance = 0
	}

	// 获取每日交易盈亏（排除入金出金）
	query := `
		SELECT
			DATE(close_time) as date,
			COALESCE(SUM(CASE WHEN order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') THEN profit ELSE 0 END), 0) as profit,
			COALESCE(SUM(CASE WHEN order_type IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit') THEN profit ELSE 0 END), 0) as deposit_withdrawal
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
	var dailyDataList []dailyData
	err = r.db.SelectContext(ctx, &dailyDataList, query, accountID, start, end)
	if err != nil {
		return nil, err
	}

	var result []*model.EquityPoint
	runningBalance := initialBalance
	runningEquity := initialBalance
	cumulativeTradingPnL := 0.0

	for _, dd := range dailyDataList {
		// 累计入金出金影响余额
		runningBalance += dd.DepositWithdrawal
		// 累计交易盈亏
		cumulativeTradingPnL += dd.Profit
		// Balance = 初始余额 + 入金出金 + 已平仓盈亏
		runningBalance += dd.Profit
		// Equity = Balance（没有实时浮动盈亏数据时两者相等）
		runningEquity = runningBalance

		result = append(result, &model.EquityPoint{
			Date:    dd.Date.Format("01/02"),
			Equity:  runningEquity,
			Balance: runningBalance,
			Profit:  dd.Profit,
		})
	}

	// 如果没有数据，返回初始余额点
	if len(result) == 0 {
		result = append(result, &model.EquityPoint{
			Date:    time.Now().Format("01/02"),
			Equity:  initialBalance,
			Balance: initialBalance,
			Profit:  0,
		})
	}

	return result, nil
}
