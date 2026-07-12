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

	_, err := r.db.Exec(ctx, query,
		config.ID, config.UserID, accountID, config.MaxRiskPercent,
		config.MaxDailyLoss, config.MaxDrawdownPercent, config.MaxPositions,
		config.MaxLotSize, config.DailyLossUsed, config.TrailingStopEnabled,
		config.TrailingStopPips, config.CreatedAt, config.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create risk config: %w", err)
	}
	return nil
}

func (r *AutoTradingRepository) GetRiskConfigByID(ctx context.Context, id uuid.UUID) (*model.RiskConfig, error) {
	query := `SELECT id, user_id, account_id, max_risk_percent, max_daily_loss, max_drawdown_percent, max_positions, max_lot_size, daily_loss_used, trailing_stop_enabled, trailing_stop_pips, created_at, updated_at FROM risk_configs WHERE id = $1`
	var config model.RiskConfig
	err := r.db.QueryRow(ctx, query, id).Scan(
		&config.ID, &config.UserID, &config.AccountID, &config.MaxRiskPercent,
		&config.MaxDailyLoss, &config.MaxDrawdownPercent, &config.MaxPositions,
		&config.MaxLotSize, &config.DailyLossUsed, &config.TrailingStopEnabled,
		&config.TrailingStopPips, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRiskConfigNotFound
		}
		return nil, err
	}
	return &config, nil
}

func (r *AutoTradingRepository) GetRiskConfigByUserID(ctx context.Context, userID uuid.UUID) (*model.RiskConfig, error) {
	query := `SELECT id, user_id, account_id, max_risk_percent, max_daily_loss, max_drawdown_percent, max_positions, max_lot_size, daily_loss_used, trailing_stop_enabled, trailing_stop_pips, created_at, updated_at FROM risk_configs WHERE user_id = $1 AND (account_id IS NULL OR account_id = '00000000-0000-0000-0000-000000000000')`
	var config model.RiskConfig
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&config.ID, &config.UserID, &config.AccountID, &config.MaxRiskPercent,
		&config.MaxDailyLoss, &config.MaxDrawdownPercent, &config.MaxPositions,
		&config.MaxLotSize, &config.DailyLossUsed, &config.TrailingStopEnabled,
		&config.TrailingStopPips, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRiskConfigNotFound
		}
		return nil, err
	}
	return &config, nil
}

func (r *AutoTradingRepository) GetRiskConfigByAccountID(ctx context.Context, accountID uuid.UUID) (*model.RiskConfig, error) {
	query := `SELECT id, user_id, account_id, max_risk_percent, max_daily_loss, max_drawdown_percent, max_positions, max_lot_size, daily_loss_used, trailing_stop_enabled, trailing_stop_pips, created_at, updated_at FROM risk_configs WHERE account_id = $1`
	var config model.RiskConfig
	err := r.db.QueryRow(ctx, query, accountID).Scan(
		&config.ID, &config.UserID, &config.AccountID, &config.MaxRiskPercent,
		&config.MaxDailyLoss, &config.MaxDrawdownPercent, &config.MaxPositions,
		&config.MaxLotSize, &config.DailyLossUsed, &config.TrailingStopEnabled,
		&config.TrailingStopPips, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	_, err := r.db.Exec(ctx, query,
		config.ID, config.MaxRiskPercent, config.MaxDailyLoss, config.MaxDrawdownPercent,
		config.MaxPositions, config.MaxLotSize, config.DailyLossUsed,
		config.TrailingStopEnabled, config.TrailingStopPips, config.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update risk config: %w", err)
	}
	return nil
}

func (r *AutoTradingRepository) UpdateDailyLossUsed(ctx context.Context, id uuid.UUID, dailyLossUsed float64) error {
	query := `UPDATE risk_configs SET daily_loss_used = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, dailyLossUsed, time.Now())
	if err != nil {
		return fmt.Errorf("update daily loss used: %w", err)
	}
	return nil
}

func (r *AutoTradingRepository) ResetDailyLossUsed(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE risk_configs SET daily_loss_used = 0, updated_at = $2 WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID, time.Now())
	if err != nil {
		return fmt.Errorf("reset daily loss used: %w", err)
	}
	return nil
}
