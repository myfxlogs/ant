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

// GetByID retrieves a single schedule by its primary key.
func (r *StrategyScheduleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
	var s model.StrategySchedule
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, template_id, account_id, name, symbol, timeframe,
			parameters, schedule_type, schedule_config, backtest_metrics,
			risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
			is_active, last_run_at, next_run_at, run_count, last_error, enable_count,
			manual_run_count, last_manual_run_at, last_manual_error,
			created_at, updated_at
		FROM strategy_schedules WHERE id = $1`, id,
	).Scan(
		&s.ID, &s.UserID, &s.TemplateID, &s.AccountID, &s.Name, &s.Symbol, &s.Timeframe,
		&s.Parameters, &s.ScheduleType, &s.ScheduleConfig, &s.BacktestMetrics,
		&s.RiskScore, &s.RiskLevel, &s.RiskReasons, &s.RiskWarnings, &s.LastBacktestAt,
		&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.RunCount, &s.LastError, &s.EnableCount,
		&s.ManualRunCount, &s.LastManualRunAt, &s.LastManualError,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}
	return &s, nil
}

// GetByUserID returns all schedules for a user, newest first.
func (r *StrategyScheduleRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.StrategySchedule, error) {
	return querySchedules(ctx, r,
		`SELECT id, user_id, template_id, account_id, name, symbol, timeframe,
			parameters, schedule_type, schedule_config, backtest_metrics,
			risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
			is_active, last_run_at, next_run_at, run_count, last_error, enable_count,
			manual_run_count, last_manual_run_at, last_manual_error,
			created_at, updated_at
		FROM strategy_schedules WHERE user_id = $1 ORDER BY created_at DESC`, userID)
}

// GetByTemplateID returns all schedules for a template.
func (r *StrategyScheduleRepository) GetByTemplateID(ctx context.Context, templateID uuid.UUID) ([]*model.StrategySchedule, error) {
	return querySchedules(ctx, r,
		`SELECT id, user_id, template_id, account_id, name, symbol, timeframe,
			parameters, schedule_type, schedule_config, backtest_metrics,
			risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
			is_active, last_run_at, next_run_at, run_count, last_error, enable_count,
			manual_run_count, last_manual_run_at, last_manual_error,
			created_at, updated_at
		FROM strategy_schedules WHERE template_id = $1 ORDER BY created_at DESC`, templateID)
}

// GetByAccountID returns all schedules for an account.
func (r *StrategyScheduleRepository) GetByAccountID(ctx context.Context, accountID uuid.UUID) ([]*model.StrategySchedule, error) {
	return querySchedules(ctx, r,
		`SELECT id, user_id, template_id, account_id, name, symbol, timeframe,
			parameters, schedule_type, schedule_config, backtest_metrics,
			risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
			is_active, last_run_at, next_run_at, run_count, last_error, enable_count,
			manual_run_count, last_manual_run_at, last_manual_error,
			created_at, updated_at
		FROM strategy_schedules WHERE account_id = $1 ORDER BY created_at DESC`, accountID)
}

// GetByUniqueKey looks up a schedule by its natural key.
func (r *StrategyScheduleRepository) GetByUniqueKey(ctx context.Context, userID, accountID, templateID uuid.UUID, symbol, timeframe string) (*model.StrategySchedule, error) {
	var s model.StrategySchedule
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, template_id, account_id, name, symbol, timeframe,
			parameters, schedule_type, schedule_config, backtest_metrics,
			risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
			is_active, last_run_at, next_run_at, run_count, last_error, enable_count,
			manual_run_count, last_manual_run_at, last_manual_error,
			created_at, updated_at
		FROM strategy_schedules WHERE user_id = $1 AND account_id = $2 AND template_id = $3 AND symbol = $4 AND timeframe = $5 LIMIT 1`,
		userID, accountID, templateID, symbol, timeframe,
	).Scan(
		&s.ID, &s.UserID, &s.TemplateID, &s.AccountID, &s.Name, &s.Symbol, &s.Timeframe,
		&s.Parameters, &s.ScheduleType, &s.ScheduleConfig, &s.BacktestMetrics,
		&s.RiskScore, &s.RiskLevel, &s.RiskReasons, &s.RiskWarnings, &s.LastBacktestAt,
		&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.RunCount, &s.LastError, &s.EnableCount,
		&s.ManualRunCount, &s.LastManualRunAt, &s.LastManualError,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}
	return &s, nil
}

// GetActiveSchedules returns all active schedules ordered by next_run_at.
func (r *StrategyScheduleRepository) GetActiveSchedules(ctx context.Context) ([]*model.StrategySchedule, error) {
	return querySchedules(ctx, r,
		`SELECT id, user_id, template_id, account_id, name, symbol, timeframe,
			parameters, schedule_type, schedule_config, backtest_metrics,
			risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
			is_active, last_run_at, next_run_at, run_count, last_error, enable_count,
			manual_run_count, last_manual_run_at, last_manual_error,
			created_at, updated_at
		FROM strategy_schedules WHERE is_active = true ORDER BY next_run_at ASC`)
}

// GetEarliestNextRunAt returns the earliest next_run_at among all active schedules.
// Returns zero time if no active schedule has a next_run_at set.
func (r *StrategyScheduleRepository) GetEarliestNextRunAt(ctx context.Context) (time.Time, error) {
	var earliest *time.Time
	err := r.db.QueryRow(ctx,
		`SELECT MIN(next_run_at) FROM strategy_schedules WHERE is_active = true AND next_run_at IS NOT NULL`,
	).Scan(&earliest)
	if err != nil {
		return time.Time{}, fmt.Errorf("GetEarliestNextRunAt: %w", err)
	}
	if earliest == nil {
		return time.Time{}, nil
	}
	return *earliest, nil
}

// GetDueSchedules returns active schedules whose next_run_at is <= the given time.
func (r *StrategyScheduleRepository) GetDueSchedules(ctx context.Context, before time.Time) ([]*model.StrategySchedule, error) {
	return querySchedules(ctx, r,
		`SELECT id, user_id, template_id, account_id, name, symbol, timeframe,
			parameters, schedule_type, schedule_config, backtest_metrics,
			risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
			is_active, last_run_at, next_run_at, run_count, last_error, enable_count,
			manual_run_count, last_manual_run_at, last_manual_error,
			created_at, updated_at
		FROM strategy_schedules WHERE is_active = true AND next_run_at IS NOT NULL AND next_run_at <= $1 ORDER BY next_run_at ASC`,
		before)
}

// CountByUserID returns the total number of schedules for a user.
func (r *StrategyScheduleRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM strategy_schedules WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

// CountByTemplateID returns the total number of schedules for a template.
func (r *StrategyScheduleRepository) CountByTemplateID(ctx context.Context, templateID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM strategy_schedules WHERE template_id = $1`, templateID).Scan(&count)
	return count, err
}

// querySchedules is a shared helper that scans schedule rows into a slice.
func querySchedules(ctx context.Context, r *StrategyScheduleRepository, query string, args ...interface{}) ([]*model.StrategySchedule, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []*model.StrategySchedule
	for rows.Next() {
		var s model.StrategySchedule
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.TemplateID, &s.AccountID, &s.Name, &s.Symbol, &s.Timeframe,
			&s.Parameters, &s.ScheduleType, &s.ScheduleConfig, &s.BacktestMetrics,
			&s.RiskScore, &s.RiskLevel, &s.RiskReasons, &s.RiskWarnings, &s.LastBacktestAt,
			&s.IsActive, &s.LastRunAt, &s.NextRunAt, &s.RunCount, &s.LastError, &s.EnableCount,
			&s.ManualRunCount, &s.LastManualRunAt, &s.LastManualError,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		schedules = append(schedules, &s)
	}
	return schedules, rows.Err()
}
