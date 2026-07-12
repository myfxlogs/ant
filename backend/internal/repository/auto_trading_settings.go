package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"alphaforge/internal/model"
)

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

	_, err := r.db.Exec(ctx, query,
		settings.ID, settings.UserID, settings.AutoTradeEnabled, settings.MaxRiskPercent,
		settings.MaxPositions, settings.MaxLotSize, settings.MaxDailyLoss, settings.MaxDrawdownPercent, settings.CreatedAt, settings.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create global settings: %w", err)
	}
	return nil
}

func (r *AutoTradingRepository) GetGlobalSettingsByUserID(ctx context.Context, userID uuid.UUID) (*model.GlobalSettings, error) {
	query := `SELECT id, user_id, auto_trade_enabled, notification_enabled, email_notification, sms_notification, max_risk_percent, max_positions, max_lot_size, max_daily_loss, max_drawdown_percent, created_at, updated_at FROM global_settings WHERE user_id = $1`
	var settings model.GlobalSettings
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&settings.ID, &settings.UserID, &settings.AutoTradeEnabled, &settings.NotificationEnabled, &settings.EmailNotification, &settings.SmsNotification,
		&settings.MaxRiskPercent, &settings.MaxPositions, &settings.MaxLotSize, &settings.MaxDailyLoss, &settings.MaxDrawdownPercent,
		&settings.CreatedAt, &settings.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	_, err := r.db.Exec(ctx, query,
		settings.ID, settings.AutoTradeEnabled, settings.MaxRiskPercent,
		settings.MaxPositions, settings.MaxLotSize, settings.MaxDailyLoss, settings.MaxDrawdownPercent, settings.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update global settings: %w", err)
	}
	return nil
}

func (r *AutoTradingRepository) UpdateAutoTradeEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	query := `UPDATE global_settings SET auto_trade_enabled = $2, updated_at = $3 WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID, enabled, time.Now())
	if err != nil {
		return fmt.Errorf("update auto trade enabled: %w", err)
	}
	return nil
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

	_, err := r.db.Exec(ctx, query,
		log.ID, log.UserID, log.AccountID, log.Action, log.Symbol, log.LogType, 0, 0, 0, 0, log.Message, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create trading log: %w", err)
	}
	return nil
}

func (r *AutoTradingRepository) GetTradingLogs(ctx context.Context, userID uuid.UUID, params *model.LogListParams) ([]*model.TradingLog, int, error) {
	baseQ, args, idx := buildTradingLogFilters(userID, params)
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) `+baseQ, args...).Scan(&total); err != nil { return nil, 0, err }
	page, pageSize := 1, 20
	if params != nil { if params.Page > 0 { page = params.Page }; if params.PageSize > 0 { pageSize = params.PageSize } }
	dataQ := fmt.Sprintf(`SELECT id, user_id, account_id, action, symbol, order_type as log_type, volume, price, ticket, profit, message, created_at %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, baseQ, idx, idx+1)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var logs []*model.TradingLog
	for rows.Next() {
		var l model.TradingLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.AccountID, &l.Action, &l.Symbol, &l.LogType, &l.Volume, &l.Price, &l.Ticket, &l.Profit, &l.Message, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}
	return logs, total, rows.Err()
}

func buildTradingLogFilters(userID uuid.UUID, params *model.LogListParams) (baseQ string, args []interface{}, idx int) {
	baseQ = `FROM trade_logs WHERE user_id = $1`
	args = []interface{}{userID}
	idx = 2
	if params == nil { return }
	addFilter := func(col, val string) { baseQ += fmt.Sprintf(` AND %s = $%d`, col, idx); args = append(args, val); idx++ }
	if params.Module != "" { addFilter("order_type", params.Module) }
	if params.StartDate != "" { addFilter("created_at >=", params.StartDate) }
	if params.EndDate != "" { addFilter("created_at <=", params.EndDate) }
	return
}

func (r *AutoTradingRepository) GetRecentTradingLogs(ctx context.Context, userID uuid.UUID, limit int) ([]*model.TradingLog, error) {
	query := `SELECT id, user_id, account_id, action, symbol, order_type as log_type, volume, price, ticket, profit, message, created_at FROM trade_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`
	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []*model.TradingLog
	for rows.Next() {
		var l model.TradingLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.AccountID, &l.Action, &l.Symbol, &l.LogType, &l.Volume, &l.Price, &l.Ticket, &l.Profit, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, err
}
