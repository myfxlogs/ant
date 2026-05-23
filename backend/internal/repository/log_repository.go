package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"anttrader/internal/model"
)

type LogRepository struct {
	db *sqlx.DB
}

func NewLogRepository(db *sqlx.DB) *LogRepository {
	return &LogRepository{db: db}
}

func (r *LogRepository) CreateConnectionLog(ctx context.Context, log *model.AccountConnectionLog) error {
	query := `
		INSERT INTO account_connection_logs (
			id, user_id, account_id, event_type, status, message, error_detail,
			server_host, server_port, login_id, connection_duration_seconds, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.db.ExecContext(ctx, query,
		log.ID, log.UserID, log.AccountID, log.EventType, log.Status, log.Message, log.ErrorDetail,
		log.ServerHost, log.ServerPort, log.LoginID, log.ConnectionDurationSecs, log.CreatedAt)
	return err
}

type ScheduleRunLogRow struct {
	ID          uuid.UUID `db:"id"`
	Kind        string    `db:"kind"`
	Action      string    `db:"action"`
	Status      string    `db:"status"`
	DurationMs  int64     `db:"duration_ms"`
	ErrorMessage string   `db:"error_message"`
	SignalType  string    `db:"signal_type"`
	SignalVolume float64  `db:"signal_volume"`
	CreatedAt   time.Time `db:"created_at"`
}

func (r *LogRepository) GetScheduleRunLogs(ctx context.Context, userID uuid.UUID, scheduleID uuid.UUID, page, pageSize int) ([]*ScheduleRunLogRow, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	// NOTE: operation logs are stored in system_operation_logs; schedule toggle logs use:
	// module=strategy_schedule, resource_type=schedule, resource_id=<schedule uuid>
	base := `
WITH merged AS (
  SELECT
    id,
    'execution'::text AS kind,
    COALESCE(signal_type, '') AS action,
    status::text AS status,
    COALESCE(execution_time_ms, 0) AS duration_ms,
    COALESCE(error_message, '') AS error_message,
    COALESCE(signal_type, '') AS signal_type,
    COALESCE(signal_volume, 0) AS signal_volume,
    created_at
  FROM strategy_execution_logs
  WHERE user_id = $1 AND schedule_id = $2

  UNION ALL

  SELECT
    id,
    'operation'::text AS kind,
    COALESCE(action, '') AS action,
    status::text AS status,
    COALESCE(duration_ms, 0) AS duration_ms,
    COALESCE(error_message, '') AS error_message,
    '' AS signal_type,
    0 AS signal_volume,
    created_at
  FROM system_operation_logs
  WHERE user_id = $1 AND module = 'strategy_schedule' AND resource_type = 'schedule' AND resource_id = $2
)
`

	countQuery := base + `SELECT COUNT(*) FROM merged`
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, userID, scheduleID); err != nil {
		return nil, 0, err
	}

	dataQuery := base + fmt.Sprintf(`SELECT * FROM merged ORDER BY created_at DESC LIMIT $3 OFFSET $4`)
	rows := make([]*ScheduleRunLogRow, 0)
	err := r.db.SelectContext(ctx, &rows, dataQuery, userID, scheduleID, pageSize, offset)
	return rows, total, err
}

func (r *LogRepository) GetConnectionLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.AccountConnectionLog, int, error) {
	baseQuery := `FROM account_connection_logs WHERE user_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	if params != nil {
		if params.AccountID != "" {
			baseQuery += fmt.Sprintf(` AND account_id = $%d`, argIndex)
			accountID, _ := uuid.Parse(params.AccountID)
			args = append(args, accountID)
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
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
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

	var logs []*model.AccountConnectionLog
	err = r.db.SelectContext(ctx, &logs, dataQuery, args...)
	return logs, total, err
}

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

	_, err := r.db.ExecContext(ctx, query,
		log.ID, log.UserID, log.ScheduleID, log.TemplateID, log.AccountID, log.Symbol, log.Timeframe, log.Status,
		log.SignalType, log.SignalPrice, log.SignalVolume, log.SignalStopLoss, log.SignalTakeProfit,
		log.ExecutedOrderID, log.ExecutedPrice, log.ExecutedVolume, log.Profit, log.ErrorMessage,
		log.ExecutionTimeMs, klineData, strategyParams, log.CreatedAt)
	return err
}

func (r *LogRepository) UpdateExecutionLog(ctx context.Context, log *model.StrategyExecutionLog) error {
	query := `
		UPDATE strategy_execution_logs SET
			status = $2, signal_type = $3, signal_price = $4, signal_volume = $5,
			signal_stop_loss = $6, signal_take_profit = $7, executed_order_id = $8,
			executed_price = $9, executed_volume = $10, profit = $11, error_message = $12,
			execution_time_ms = $13
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		log.ID, log.Status, log.SignalType, log.SignalPrice, log.SignalVolume,
		log.SignalStopLoss, log.SignalTakeProfit, log.ExecutedOrderID,
		log.ExecutedPrice, log.ExecutedVolume, log.Profit, log.ErrorMessage,
		log.ExecutionTimeMs)
	return err
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
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
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

	var logs []*model.StrategyExecutionLog
	err = r.db.SelectContext(ctx, &logs, dataQuery, args...)
	return logs, total, err
}

func (r *LogRepository) CreateOrderHistory(ctx context.Context, order *model.OrderHistory) error {
	query := `
		INSERT INTO order_history (
			id, user_id, account_id, ticket, order_type, symbol, volume,
			open_price, close_price, open_time, close_time, stop_loss, take_profit,
			profit, commission, swap, comment, magic_number, is_auto_trade, schedule_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`

	_, err := r.db.ExecContext(ctx, query,
		order.ID, order.UserID, order.AccountID, order.Ticket, order.OrderType, order.Symbol, order.Volume,
		order.OpenPrice, order.ClosePrice, order.OpenTime, order.CloseTime, order.StopLoss, order.TakeProfit,
		order.Profit, order.Commission, order.Swap, order.Comment, order.MagicNumber, order.IsAutoTrade, order.ScheduleID, order.CreatedAt)
	return err
}

// UpdateOrderHistoryClose fills close_* / PnL on a row previously inserted for this schedule ticket (first close only).
func (r *LogRepository) UpdateOrderHistoryClose(ctx context.Context, userID, accountID, scheduleID uuid.UUID, ticket int64, closePrice, profit, swap, commission float64, closeTime time.Time) (int64, error) {
	const q = `
		UPDATE order_history
		SET close_price = $5,
			close_time = $6,
			profit = $7,
			swap = $8,
			commission = $9
		WHERE user_id = $1 AND account_id = $2 AND schedule_id = $3 AND ticket = $4
		  AND close_time IS NULL`
	res, err := r.db.ExecContext(ctx, q, userID, accountID, scheduleID, ticket, closePrice, closeTime, profit, swap, commission)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *LogRepository) GetOrderHistory(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.OrderHistory, int, error) {
	baseQuery := `FROM order_history WHERE user_id = $1`
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
		if params.Type != "" {
			baseQuery += fmt.Sprintf(` AND order_type = $%d`, argIndex)
			args = append(args, params.Type)
			argIndex++
		}
		if params.StartDate != "" {
			baseQuery += fmt.Sprintf(` AND open_time >= $%d`, argIndex)
			args = append(args, params.StartDate)
			argIndex++
		}
		if params.EndDate != "" {
			baseQuery += fmt.Sprintf(` AND open_time <= $%d`, argIndex)
			args = append(args, params.EndDate)
			argIndex++
		}
	}

	countQuery := `SELECT COUNT(*) ` + baseQuery
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
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
	dataQuery := fmt.Sprintf(`SELECT * %s ORDER BY open_time DESC LIMIT $%d OFFSET $%d`, baseQuery, argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	var orders []*model.OrderHistory
	err = r.db.SelectContext(ctx, &orders, dataQuery, args...)
	return orders, total, err
}

func (r *LogRepository) CreateOperationLog(ctx context.Context, log *model.SystemOperationLog) error {
	query := `
		INSERT INTO system_operation_logs (
			id, user_id, operation_type, module, resource_type, resource_id, action,
			old_value, new_value, ip_address, user_agent, status, error_message, duration_ms, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			error_message = EXCLUDED.error_message,
			duration_ms = EXCLUDED.duration_ms,
			new_value = EXCLUDED.new_value,
			created_at = EXCLUDED.created_at`

	var oldValue, newValue []byte
	if log.OldValue != nil {
		oldValue, _ = json.Marshal(log.OldValue)
	}
	if log.NewValue != nil {
		newValue, _ = json.Marshal(log.NewValue)
	}

	_, err := r.db.ExecContext(ctx, query,
		log.ID, log.UserID, log.OperationType, log.Module, log.ResourceType, log.ResourceID, log.Action,
		oldValue, newValue, log.IPAddress, log.UserAgent, log.Status, log.ErrorMessage, log.DurationMs, log.CreatedAt)
	return err
}

func (r *LogRepository) GetOperationLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.SystemOperationLog, int, error) {
	baseQuery := `FROM system_operation_logs WHERE user_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	if params != nil {
		if params.Module != "" {
			baseQuery += fmt.Sprintf(` AND module = $%d`, argIndex)
			args = append(args, params.Module)
			argIndex++
		}
		if params.Action != "" {
			baseQuery += fmt.Sprintf(` AND action = $%d`, argIndex)
			args = append(args, params.Action)
			argIndex++
		}
		if params.Type != "" {
			baseQuery += fmt.Sprintf(` AND operation_type = $%d`, argIndex)
			args = append(args, params.Type)
			argIndex++
		}
		if params.ResourceType != "" {
			baseQuery += fmt.Sprintf(` AND resource_type = $%d`, argIndex)
			args = append(args, params.ResourceType)
			argIndex++
		}
		if params.ResourceID != "" {
			baseQuery += fmt.Sprintf(` AND resource_id = $%d`, argIndex)
			rid, _ := uuid.Parse(params.ResourceID)
			args = append(args, rid)
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
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
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

	var logs []*model.SystemOperationLog
	err = r.db.SelectContext(ctx, &logs, dataQuery, args...)
	return logs, total, err
}
