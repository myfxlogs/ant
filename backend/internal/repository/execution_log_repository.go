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
	baseQ, args, argIdx := buildExecLogFilters(userID, params)
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) `+baseQ, args...).Scan(&total); err != nil { return nil, 0, err }
	page, pageSize := normalizeOpLogPagination(params)
	dataQ := fmt.Sprintf(`SELECT * %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, baseQ, argIdx, argIdx+1)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var logs []*model.StrategyExecutionLog
	for rows.Next() {
		var l model.StrategyExecutionLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.ScheduleID, &l.TemplateID, &l.AccountID, &l.Symbol, &l.Timeframe, &l.Status, &l.SignalType, &l.SignalPrice, &l.SignalVolume, &l.SignalStopLoss, &l.SignalTakeProfit, &l.ExecutedOrderID, &l.ExecutedPrice, &l.ExecutedVolume, &l.Profit, &l.ErrorMessage, &l.ExecutionTimeMs, &l.KlineData, &l.StrategyParams, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}
	return logs, total, rows.Err()
}

func buildExecLogFilters(userID uuid.UUID, params *model.LogQueryParams) (baseQ string, args []interface{}, idx int) {
	baseQ = `FROM strategy_execution_logs WHERE user_id = $1`
	args = []interface{}{userID}
	idx = 2
	if params == nil { return }
	addFilter := func(cond, val string) { baseQ += fmt.Sprintf(` AND %s = $%d`, cond, idx); args = append(args, val); idx++ }
	if params.ScheduleID != "" { sid, _ := uuid.Parse(params.ScheduleID); addFilter("schedule_id", ""); args[len(args)-1] = sid }
	if params.AccountID != "" { aid, _ := uuid.Parse(params.AccountID); addFilter("account_id", ""); args[len(args)-1] = aid }
	if params.Symbol != "" { addFilter("symbol", params.Symbol) }
	if params.Status != "" { addFilter("status", params.Status) }
	if params.StartDate != "" { addFilter("created_at >=", params.StartDate) }
	if params.EndDate != "" { addFilter("created_at <=", params.EndDate) }
	return
}
