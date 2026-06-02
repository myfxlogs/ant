package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

func (r *LogRepository) CreateExecutionLog(ctx context.Context, log *model.StrategyExecutionLog) error {
	query := `
		INSERT INTO strategy_execution_logs (
			id, user_id, schedule_id, template_id, account_id, symbol, timeframe, status,
			signal_type, signal_price, signal_volume, signal_stop_loss, signal_take_profit,
			executed_order_id, executed_price, executed_volume, profit, error_message,
			execution_time_ms, kline_data, strategy_params, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)`

	var klineData, strategyParams []byte
	if log.KlineData != nil {
		klineData, _ = json.Marshal(log.KlineData)
	}
	if log.StrategyParams != nil {
		strategyParams, _ = json.Marshal(log.StrategyParams)
	}

	_, err := r.db.Exec(ctx, query,
		log.ID, log.UserID, log.ScheduleID, log.TemplateID, log.AccountID, log.Symbol, log.Timeframe, log.Status,
		log.SignalType, log.SignalPrice, log.SignalVolume, log.SignalStopLoss, log.SignalTakeProfit,
		log.ExecutedOrderID, log.ExecutedPrice, log.ExecutedVolume, log.Profit, log.ErrorMessage,
		log.ExecutionTimeMs, klineData, strategyParams, log.CreatedAt)
	return fmt.Errorf("create execution log: %w", err)
}

func (r *LogRepository) UpdateExecutionLog(ctx context.Context, log *model.StrategyExecutionLog) error {
	query := `
		UPDATE strategy_execution_logs SET
			status = $2, signal_type = $3, signal_price = $4, signal_volume = $5,
			signal_stop_loss = $6, signal_take_profit = $7, executed_order_id = $8,
			executed_price = $9, executed_volume = $10, profit = $11, error_message = $12,
			execution_time_ms = $13
		WHERE id = $1`

	_, err := r.db.Exec(ctx, query,
		log.ID, log.Status, log.SignalType, log.SignalPrice, log.SignalVolume,
		log.SignalStopLoss, log.SignalTakeProfit, log.ExecutedOrderID,
		log.ExecutedPrice, log.ExecutedVolume, log.Profit, log.ErrorMessage,
		log.ExecutionTimeMs)
	return fmt.Errorf("update execution log: %w", err)
}

func (r *LogRepository) GetExecutionLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.StrategyExecutionLog, int, error) {
	baseQuery := `FROM strategy_execution_logs WHERE user_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	if params != nil {
		if params.ScheduleID != "" {
			baseQuery += fmt.Sprintf(` AND schedule_id = $%d`, argIndex)
			scheduleID, _ := uuid.Parse(params.ScheduleID)
			args = append(args, scheduleID)
			argIndex++
		}
		if params.AccountID != "" {
			baseQuery += fmt.Sprintf(` AND account_id = $%d`, argIndex)
			accountID, _ := uuid.Parse(params.AccountID)
			args = append(args, accountID)
			argIndex++
		}
		if params.Symbol != "" {
			baseQuery += fmt.Sprintf(` AND symbol = $%d`, argIndex)
			args = append(args, params.Symbol)
			argIndex++
		}
		if params.Status != "" {
			baseQuery += fmt.Sprintf(` AND status = $%d`, argIndex)
			args = append(args, params.Status)
			argIndex++
		}
		if params.StartDate != "" {
			baseQuery += fmt.Sprintf(` AND created_at >= $%d`, argIndex)
			args = append(args, params.StartDate)
			argIndex++
		}
		if params.EndDate != "" {
			baseQuery += fmt.Sprintf(` AND created_at <= $%d`, argIndex)
			args = append(args, params.EndDate)
			argIndex++
		}
	}

	countQuery := `SELECT COUNT(*) ` + baseQuery
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	page := 1
	pageSize := 20
	if params != nil {
		if params.Page > 0 {
			page = params.Page
		}
		if params.PageSize > 0 {
			pageSize = params.PageSize
		}
	}

	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(`SELECT * %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, baseQuery, argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	queryRows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer queryRows.Close()
	var logs []*model.StrategyExecutionLog
	for queryRows.Next() {
		var log model.StrategyExecutionLog
		if err := queryRows.Scan(&log.ID, &log.UserID, &log.ScheduleID, &log.TemplateID, &log.AccountID, &log.Symbol, &log.Timeframe, &log.Status, &log.SignalType, &log.SignalPrice, &log.SignalVolume, &log.SignalStopLoss, &log.SignalTakeProfit, &log.ExecutedOrderID, &log.ExecutedPrice, &log.ExecutedVolume, &log.Profit, &log.ErrorMessage, &log.ExecutionTimeMs, &log.KlineData, &log.StrategyParams, &log.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &log)
	}
	if err := queryRows.Err(); err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
