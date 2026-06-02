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

	_, err := r.db.Exec(ctx, query,
		schedule.ID, schedule.UserID, schedule.TemplateID, schedule.AccountID, schedule.Name, schedule.Symbol, schedule.Timeframe,
		schedule.Parameters, schedule.ScheduleType, schedule.ScheduleConfig,
		schedule.IsActive, schedule.LastRunAt, schedule.NextRunAt,
		schedule.LastError, schedule.RunCount, schedule.CreatedAt, schedule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create schedule: %w", err)
	}
	return nil
}

func (r *AutoTradingRepository) GetScheduleByID(ctx context.Context, id uuid.UUID) (*model.StrategyScheduleLegacy, error) {
	query := `SELECT id, strategy_id, user_id, template_id, account_id, name, symbol, timeframe, parameters, schedule_type, schedule_config, is_active, last_run_at, next_run_at, last_error, run_count, created_at, updated_at FROM strategy_schedules WHERE id = $1`
	var schedule model.StrategyScheduleLegacy
	err := r.db.QueryRow(ctx, query, id).Scan(
		&schedule.ID, &schedule.StrategyID, &schedule.UserID, &schedule.TemplateID, &schedule.AccountID,
		&schedule.Name, &schedule.Symbol, &schedule.Timeframe, &schedule.Parameters, &schedule.ScheduleType, &schedule.ScheduleConfig,
		&schedule.IsActive, &schedule.LastRunAt, &schedule.NextRunAt, &schedule.LastError, &schedule.RunCount,
		&schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLegacyScheduleNotFound
		}
		return nil, err
	}
	return &schedule, nil
}

func (r *AutoTradingRepository) GetSchedulesByTemplateID(ctx context.Context, templateID uuid.UUID) ([]*model.StrategyScheduleLegacy, error) {
	query := `SELECT id, strategy_id, user_id, template_id, account_id, name, symbol, timeframe, parameters, schedule_type, schedule_config, is_active, last_run_at, next_run_at, last_error, run_count, created_at, updated_at FROM strategy_schedules WHERE template_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []*model.StrategyScheduleLegacy
	for rows.Next() {
		var s model.StrategyScheduleLegacy
		if err := rows.Scan(
			&s.ID, &s.StrategyID, &s.UserID, &s.TemplateID, &s.AccountID,
			&s.Name, &s.Symbol, &s.Timeframe, &s.Parameters, &s.ScheduleType, &s.ScheduleConfig,
			&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.LastError, &s.RunCount,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		schedules = append(schedules, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return schedules, nil
}

func (r *AutoTradingRepository) GetSchedulesByUserID(ctx context.Context, userID uuid.UUID) ([]*model.StrategyScheduleLegacy, error) {
	query := `SELECT id, strategy_id, user_id, template_id, account_id, name, symbol, timeframe, parameters, schedule_type, schedule_config, is_active, last_run_at, next_run_at, last_error, run_count, created_at, updated_at FROM strategy_schedules WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []*model.StrategyScheduleLegacy
	for rows.Next() {
		var s model.StrategyScheduleLegacy
		if err := rows.Scan(
			&s.ID, &s.StrategyID, &s.UserID, &s.TemplateID, &s.AccountID,
			&s.Name, &s.Symbol, &s.Timeframe, &s.Parameters, &s.ScheduleType, &s.ScheduleConfig,
			&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.LastError, &s.RunCount,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		schedules = append(schedules, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return schedules, nil
}

func (r *AutoTradingRepository) GetSchedulesByAccountID(ctx context.Context, accountID uuid.UUID) ([]*model.StrategyScheduleLegacy, error) {
	query := `SELECT id, strategy_id, user_id, template_id, account_id, name, symbol, timeframe, parameters, schedule_type, schedule_config, is_active, last_run_at, next_run_at, last_error, run_count, created_at, updated_at FROM strategy_schedules WHERE account_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []*model.StrategyScheduleLegacy
	for rows.Next() {
		var s model.StrategyScheduleLegacy
		if err := rows.Scan(
			&s.ID, &s.StrategyID, &s.UserID, &s.TemplateID, &s.AccountID,
			&s.Name, &s.Symbol, &s.Timeframe, &s.Parameters, &s.ScheduleType, &s.ScheduleConfig,
			&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.LastError, &s.RunCount,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		schedules = append(schedules, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return schedules, nil
}

func (r *AutoTradingRepository) GetActiveSchedules(ctx context.Context) ([]*model.StrategyScheduleLegacy, error) {
	query := `SELECT id, strategy_id, user_id, template_id, account_id, name, symbol, timeframe, parameters, schedule_type, schedule_config, is_active, last_run_at, next_run_at, last_error, run_count, created_at, updated_at FROM strategy_schedules WHERE is_active = true ORDER BY next_run_at ASC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []*model.StrategyScheduleLegacy
	for rows.Next() {
		var s model.StrategyScheduleLegacy
		if err := rows.Scan(
			&s.ID, &s.StrategyID, &s.UserID, &s.TemplateID, &s.AccountID,
			&s.Name, &s.Symbol, &s.Timeframe, &s.Parameters, &s.ScheduleType, &s.ScheduleConfig,
			&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.LastError, &s.RunCount,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		schedules = append(schedules, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return schedules, nil
}

func (r *AutoTradingRepository) GetDueSchedules(ctx context.Context, before time.Time) ([]*model.StrategyScheduleLegacy, error) {
	query := `SELECT id, strategy_id, user_id, template_id, account_id, name, symbol, timeframe, parameters, schedule_type, schedule_config, is_active, last_run_at, next_run_at, last_error, run_count, created_at, updated_at FROM strategy_schedules WHERE is_active = true AND next_run_at <= $1 ORDER BY next_run_at ASC`
	rows, err := r.db.Query(ctx, query, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []*model.StrategyScheduleLegacy
	for rows.Next() {
		var s model.StrategyScheduleLegacy
		if err := rows.Scan(
			&s.ID, &s.StrategyID, &s.UserID, &s.TemplateID, &s.AccountID,
			&s.Name, &s.Symbol, &s.Timeframe, &s.Parameters, &s.ScheduleType, &s.ScheduleConfig,
			&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.LastError, &s.RunCount,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		schedules = append(schedules, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return schedules, nil
}

func (r *AutoTradingRepository) UpdateSchedule(ctx context.Context, schedule *model.StrategyScheduleLegacy) error {
	query := `
			UPDATE strategy_schedules SET
				schedule_type = $2, schedule_config = $3, is_active = $4,
				last_run_at = $5, next_run_at = $6, last_error = $7,
				run_count = $8, updated_at = $9
			WHERE id = $1`

	schedule.UpdatedAt = time.Now()
	_, err := r.db.Exec(ctx, query,
		schedule.ID, schedule.ScheduleType, schedule.ScheduleConfig, schedule.IsActive,
		schedule.LastRunAt, schedule.NextRunAt, schedule.LastError, schedule.RunCount,
		schedule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}
	return nil
}

func (r *AutoTradingRepository) UpdateScheduleStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	query := `UPDATE strategy_schedules SET is_active = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, isActive, time.Now())
	if err != nil {
		return fmt.Errorf("update schedule status: %w", err)
	}
	return nil
}

func (r *AutoTradingRepository) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM strategy_schedules WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrLegacyScheduleNotFound
	}
	return nil
}
