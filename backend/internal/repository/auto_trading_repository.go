package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"anttrader/internal/model"
)

var (
	ErrLegacyScheduleNotFound = errors.New("schedule not found")
	ErrExecutionNotFound      = errors.New("execution not found")
	ErrRiskConfigNotFound     = errors.New("risk config not found")
	ErrGlobalSettingsNotFound = errors.New("global settings not found")
)

type AutoTradingRepository struct {
	db *sqlx.DB
}

func NewAutoTradingRepository(db *sqlx.DB) *AutoTradingRepository {
	return &AutoTradingRepository{db: db}
}

func (r *AutoTradingRepository) CreateSchedule(ctx context.Context, schedule *model.StrategyScheduleLegacy) error {
	query := `
		INSERT INTO strategy_schedules (
			id, user_id, template_id, account_id, name, symbol, timeframe,
			parameters, schedule_type, schedule_config,
			is_active, last_run_at, next_run_at, last_error, run_count,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

	now := time.Now()
	if schedule.ID == uuid.Nil {
		schedule.ID = uuid.New()
	}
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		schedule.ID, schedule.UserID, schedule.TemplateID, schedule.AccountID, schedule.Name, schedule.Symbol, schedule.Timeframe,
		schedule.Parameters, schedule.ScheduleType, schedule.ScheduleConfig,
		schedule.IsActive, schedule.LastRunAt, schedule.NextRunAt,
		schedule.LastError, schedule.RunCount, schedule.CreatedAt, schedule.UpdatedAt,
	)
	return err
}

func (r *AutoTradingRepository) GetScheduleByID(ctx context.Context, id uuid.UUID) (*model.StrategyScheduleLegacy, error) {
	query := `SELECT * FROM strategy_schedules WHERE id = $1`
	var schedule model.StrategyScheduleLegacy
	err := r.db.GetContext(ctx, &schedule, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLegacyScheduleNotFound
		}
		return nil, err
	}
	return &schedule, nil
}

func (r *AutoTradingRepository) GetSchedulesByTemplateID(ctx context.Context, templateID uuid.UUID) ([]*model.StrategyScheduleLegacy, error) {
	query := `SELECT * FROM strategy_schedules WHERE template_id = $1 ORDER BY created_at DESC`
	var schedules []*model.StrategyScheduleLegacy
	err := r.db.SelectContext(ctx, &schedules, query, templateID)
	return schedules, err
}

func (r *AutoTradingRepository) GetSchedulesByUserID(ctx context.Context, userID uuid.UUID) ([]*model.StrategyScheduleLegacy, error) {
	query := `SELECT * FROM strategy_schedules WHERE user_id = $1 ORDER BY created_at DESC`
	var schedules []*model.StrategyScheduleLegacy
	err := r.db.SelectContext(ctx, &schedules, query, userID)
	return schedules, err
}

func (r *AutoTradingRepository) GetSchedulesByAccountID(ctx context.Context, accountID uuid.UUID) ([]*model.StrategyScheduleLegacy, error) {
	query := `SELECT * FROM strategy_schedules WHERE account_id = $1 ORDER BY created_at DESC`
	var schedules []*model.StrategyScheduleLegacy
	err := r.db.SelectContext(ctx, &schedules, query, accountID)
	return schedules, err
}

func (r *AutoTradingRepository) GetActiveSchedules(ctx context.Context) ([]*model.StrategyScheduleLegacy, error) {
	query := `SELECT * FROM strategy_schedules WHERE is_active = true ORDER BY next_run_at ASC`
	var schedules []*model.StrategyScheduleLegacy
	err := r.db.SelectContext(ctx, &schedules, query)
	return schedules, err
}

func (r *AutoTradingRepository) GetDueSchedules(ctx context.Context, before time.Time) ([]*model.StrategyScheduleLegacy, error) {
	query := `SELECT * FROM strategy_schedules WHERE is_active = true AND next_run_at <= $1 ORDER BY next_run_at ASC`
	var schedules []*model.StrategyScheduleLegacy
	err := r.db.SelectContext(ctx, &schedules, query, before)
	return schedules, err
}

func (r *AutoTradingRepository) UpdateSchedule(ctx context.Context, schedule *model.StrategyScheduleLegacy) error {
	query := `
		UPDATE strategy_schedules SET
			schedule_type = $2, schedule_config = $3, is_active = $4,
			last_run_at = $5, next_run_at = $6, last_error = $7,
			run_count = $8, updated_at = $9
		WHERE id = $1`

	schedule.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		schedule.ID, schedule.ScheduleType, schedule.ScheduleConfig, schedule.IsActive,
		schedule.LastRunAt, schedule.NextRunAt, schedule.LastError, schedule.RunCount,
		schedule.UpdatedAt,
	)
	return err
}

func (r *AutoTradingRepository) UpdateScheduleStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	query := `UPDATE strategy_schedules SET is_active = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, isActive, time.Now())
	return err
}

func (r *AutoTradingRepository) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM strategy_schedules WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrLegacyScheduleNotFound
	}
	return nil
}

func (r *AutoTradingRepository) CreateExecution(ctx context.Context, execution *model.StrategyExecution) error {
	query := `
		INSERT INTO strategy_executions (
			id, user_id, template_id, schedule_id, account_id, status,
			signals, orders, error_message, started_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	if execution.ID == uuid.Nil {
		execution.ID = uuid.New()
	}
	execution.StartedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		execution.ID, execution.UserID, execution.TemplateID, execution.ScheduleID, execution.AccountID,
		execution.Status, execution.Signals, execution.Orders, execution.ErrorMessage,
		execution.StartedAt, execution.CompletedAt,
	)
	return err
}

func (r *AutoTradingRepository) GetExecutionByID(ctx context.Context, id uuid.UUID) (*model.StrategyExecution, error) {
	query := `SELECT * FROM strategy_executions WHERE id = $1`
	var execution model.StrategyExecution
	err := r.db.GetContext(ctx, &execution, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrExecutionNotFound
		}
		return nil, err
	}
	return &execution, nil
}

func (r *AutoTradingRepository) GetExecutionsByTemplateID(ctx context.Context, templateID uuid.UUID, limit int) ([]*model.StrategyExecution, error) {
	query := `SELECT * FROM strategy_executions WHERE template_id = $1 ORDER BY started_at DESC LIMIT $2`
	var executions []*model.StrategyExecution
	err := r.db.SelectContext(ctx, &executions, query, templateID, limit)
	return executions, err
}

func (r *AutoTradingRepository) GetExecutionsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*model.StrategyExecution, error) {
	query := `SELECT * FROM strategy_executions WHERE user_id = $1 ORDER BY started_at DESC LIMIT $2`
	var executions []*model.StrategyExecution
	err := r.db.SelectContext(ctx, &executions, query, userID, limit)
	return executions, err
}

func (r *AutoTradingRepository) GetExecutionsByAccountID(ctx context.Context, accountID uuid.UUID, limit int) ([]*model.StrategyExecution, error) {
	query := `SELECT * FROM strategy_executions WHERE account_id = $1 ORDER BY started_at DESC LIMIT $2`
	var executions []*model.StrategyExecution
	err := r.db.SelectContext(ctx, &executions, query, accountID, limit)
	return executions, err
}

func (r *AutoTradingRepository) UpdateExecution(ctx context.Context, execution *model.StrategyExecution) error {
	query := `
		UPDATE strategy_executions SET
			status = $2, signals = $3, orders = $4, error_message = $5, completed_at = $6
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		execution.ID, execution.Status, execution.Signals, execution.Orders,
		execution.ErrorMessage, execution.CompletedAt,
	)
	return err
}

func (r *AutoTradingRepository) GetTodayExecutionCount(ctx context.Context, accountID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM strategy_executions WHERE account_id = $1 AND started_at >= CURRENT_DATE`
	var count int
	err := r.db.GetContext(ctx, &count, query, accountID)
	return count, err
}

func (r *AutoTradingRepository) GetTodayProfit(ctx context.Context, accountID uuid.UUID) (float64, error) {
	query := `
		SELECT COALESCE(SUM((orders->>'profit')::float), 0)
		FROM strategy_executions
		WHERE account_id = $1 AND started_at >= CURRENT_DATE AND status = 'completed'`
	var profit float64
	err := r.db.GetContext(ctx, &profit, query, accountID)
	return profit, err
}

func (r *AutoTradingRepository) CreateRiskConfig(ctx context.Context, config *model.RiskConfig) error {
	query := `
		INSERT INTO risk_configs (
			id, user_id, account_id, max_risk_percent, max_daily_loss,
			max_drawdown_percent, max_positions, max_lot_size, daily_loss_used,
			trailing_stop_enabled, trailing_stop_pips, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	now := time.Now()
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}
	config.CreatedAt = now
	config.UpdatedAt = now

	var accountID any = config.AccountID
	if config.AccountID == uuid.Nil {
		accountID = nil
	}

	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.UserID, accountID, config.MaxRiskPercent,
		config.MaxDailyLoss, config.MaxDrawdownPercent, config.MaxPositions,
		config.MaxLotSize, config.DailyLossUsed, config.TrailingStopEnabled,
		config.TrailingStopPips, config.CreatedAt, config.UpdatedAt,
	)
	return err
}

func (r *AutoTradingRepository) GetRiskConfigByID(ctx context.Context, id uuid.UUID) (*model.RiskConfig, error) {
	query := `SELECT * FROM risk_configs WHERE id = $1`
	var config model.RiskConfig
	err := r.db.GetContext(ctx, &config, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRiskConfigNotFound
		}
		return nil, err
	}
	return &config, nil
}

func (r *AutoTradingRepository) GetRiskConfigByUserID(ctx context.Context, userID uuid.UUID) (*model.RiskConfig, error) {
	query := `SELECT * FROM risk_configs WHERE user_id = $1 AND (account_id IS NULL OR account_id = '00000000-0000-0000-0000-000000000000')`
	var config model.RiskConfig
	err := r.db.GetContext(ctx, &config, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRiskConfigNotFound
		}
		return nil, err
	}
	return &config, nil
}

func (r *AutoTradingRepository) GetRiskConfigByAccountID(ctx context.Context, accountID uuid.UUID) (*model.RiskConfig, error) {
	query := `SELECT * FROM risk_configs WHERE account_id = $1`
	var config model.RiskConfig
	err := r.db.GetContext(ctx, &config, query, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRiskConfigNotFound
		}
		return nil, err
	}
	return &config, nil
}

func (r *AutoTradingRepository) UpdateRiskConfig(ctx context.Context, config *model.RiskConfig) error {
	query := `
		UPDATE risk_configs SET
			max_risk_percent = $2, max_daily_loss = $3, max_drawdown_percent = $4,
			max_positions = $5, max_lot_size = $6, daily_loss_used = $7,
			trailing_stop_enabled = $8, trailing_stop_pips = $9, updated_at = $10
		WHERE id = $1`

	config.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.MaxRiskPercent, config.MaxDailyLoss, config.MaxDrawdownPercent,
		config.MaxPositions, config.MaxLotSize, config.DailyLossUsed,
		config.TrailingStopEnabled, config.TrailingStopPips, config.UpdatedAt,
	)
	return err
}

func (r *AutoTradingRepository) UpdateDailyLossUsed(ctx context.Context, id uuid.UUID, dailyLossUsed float64) error {
	query := `UPDATE risk_configs SET daily_loss_used = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, dailyLossUsed, time.Now())
	return err
}

func (r *AutoTradingRepository) ResetDailyLossUsed(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE risk_configs SET daily_loss_used = 0, updated_at = $2 WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID, time.Now())
	return err
}

func (r *AutoTradingRepository) CreateGlobalSettings(ctx context.Context, settings *model.GlobalSettings) error {
	query := `
		INSERT INTO global_settings (
			id, user_id, auto_trade_enabled, max_risk_percent,
			max_positions, max_lot_size, max_daily_loss, max_drawdown_percent, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	now := time.Now()
	if settings.ID == uuid.Nil {
		settings.ID = uuid.New()
	}
	settings.CreatedAt = now
	settings.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		settings.ID, settings.UserID, settings.AutoTradeEnabled, settings.MaxRiskPercent,
		settings.MaxPositions, settings.MaxLotSize, settings.MaxDailyLoss, settings.MaxDrawdownPercent, settings.CreatedAt, settings.UpdatedAt,
	)
	return err
}

func (r *AutoTradingRepository) GetGlobalSettingsByUserID(ctx context.Context, userID uuid.UUID) (*model.GlobalSettings, error) {
	query := `SELECT * FROM global_settings WHERE user_id = $1`
	var settings model.GlobalSettings
	err := r.db.GetContext(ctx, &settings, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGlobalSettingsNotFound
		}
		return nil, err
	}
	return &settings, nil
}

func (r *AutoTradingRepository) UpdateGlobalSettings(ctx context.Context, settings *model.GlobalSettings) error {
	query := `
		UPDATE global_settings SET
			auto_trade_enabled = $2, max_risk_percent = $3,
			max_positions = $4, max_lot_size = $5, max_daily_loss = $6, max_drawdown_percent = $7, updated_at = $8
		WHERE id = $1`

	settings.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		settings.ID, settings.AutoTradeEnabled, settings.MaxRiskPercent,
		settings.MaxPositions, settings.MaxLotSize, settings.MaxDailyLoss, settings.MaxDrawdownPercent, settings.UpdatedAt,
	)
	return err
}

func (r *AutoTradingRepository) UpdateAutoTradeEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	query := `UPDATE global_settings SET auto_trade_enabled = $2, updated_at = $3 WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID, enabled, time.Now())
	return err
}

func (r *AutoTradingRepository) CreateTradingLog(ctx context.Context, log *model.TradingLog) error {
	query := `
		INSERT INTO trade_logs (
			id, user_id, account_id, action, symbol, order_type, volume, price, ticket, profit, message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	log.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		log.ID, log.UserID, log.AccountID, log.Action, log.Symbol, log.LogType, 0, 0, 0, 0, log.Message, log.CreatedAt,
	)
	return err
}

func (r *AutoTradingRepository) GetTradingLogs(ctx context.Context, userID uuid.UUID, params *model.LogListParams) ([]*model.TradingLog, int, error) {
	baseQuery := `FROM trade_logs WHERE user_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	if params != nil {
		if params.Module != "" {
			baseQuery += ` AND order_type = $` + string(rune('0'+argIndex))
			args = append(args, params.Module)
			argIndex++
		}
		if params.StartDate != "" {
			baseQuery += ` AND created_at >= $` + string(rune('0'+argIndex))
			args = append(args, params.StartDate)
			argIndex++
		}
		if params.EndDate != "" {
			baseQuery += ` AND created_at <= $` + string(rune('0'+argIndex))
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

	pageSize := 20
	page := 1
	if params != nil {
		if params.PageSize > 0 {
			pageSize = params.PageSize
		}
		if params.Page > 0 {
			page = params.Page
		}
	}
	offset := (page - 1) * pageSize

	dataQuery := `SELECT id, user_id, account_id, action, symbol, order_type as log_type, volume, price, ticket, profit, message, created_at ` + baseQuery + ` ORDER BY created_at DESC LIMIT $` + string(rune('0'+argIndex)) + ` OFFSET $` + string(rune('0'+argIndex+1))
	args = append(args, pageSize, offset)

	var logs []*model.TradingLog
	err = r.db.SelectContext(ctx, &logs, dataQuery, args...)
	return logs, total, err
}

func (r *AutoTradingRepository) GetRecentTradingLogs(ctx context.Context, userID uuid.UUID, limit int) ([]*model.TradingLog, error) {
	query := `SELECT id, user_id, account_id, action, symbol, order_type as log_type, volume, price, ticket, profit, message, created_at FROM trade_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`
	var logs []*model.TradingLog
	err := r.db.SelectContext(ctx, &logs, query, userID, limit)
	return logs, err
}
