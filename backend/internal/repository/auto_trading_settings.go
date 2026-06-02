package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"anttrader/internal/model"
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
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
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

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*model.TradingLog
	for rows.Next() {
		var l model.TradingLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.AccountID, &l.Action, &l.Symbol, &l.LogType, &l.Volume, &l.Price, &l.Ticket, &l.Profit, &l.Message, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return logs, total, nil
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
