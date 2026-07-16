package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"alphaforge/internal/model"
)

// Create inserts a new strategy schedule.
func (r *StrategyScheduleRepository) Create(ctx context.Context, s *model.StrategySchedule) error {
	now := time.Now()
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	s.CreatedAt = now
	s.UpdatedAt = now

	_, err := r.db.Exec(ctx,
		`INSERT INTO strategy_schedules (
			id, user_id, template_id, account_id, name, symbol, timeframe,
			parameters, schedule_type, schedule_config, backtest_metrics,
			risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
			is_active, last_run_at, next_run_at, run_count, last_error, enable_count,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)`,
		s.ID, s.UserID, s.TemplateID, s.AccountID, s.Name, s.Symbol, s.Timeframe,
		s.Parameters, s.ScheduleType, s.ScheduleConfig, s.BacktestMetrics,
		s.RiskScore, s.RiskLevel, s.RiskReasons, s.RiskWarnings, s.LastBacktestAt,
		s.IsActive, s.LastRunAt, s.NextRunAt, s.RunCount, s.LastError, s.EnableCount,
		s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create schedule: %w", err)
	}
	return nil
}

// Update modifies all mutable fields of an existing schedule.
func (r *StrategyScheduleRepository) Update(ctx context.Context, s *model.StrategySchedule) error {
	s.UpdatedAt = time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE strategy_schedules SET
			name = $2, symbol = $3, timeframe = $4, parameters = $5,
			schedule_type = $6, schedule_config = $7, backtest_metrics = $8,
			risk_score = $9, risk_level = $10, risk_reasons = $11, risk_warnings = $12,
			last_backtest_at = $13, is_active = $14, last_run_at = $15, next_run_at = $16,
			run_count = $17, last_error = $18, updated_at = $19
		WHERE id = $1`,
		s.ID, s.Name, s.Symbol, s.Timeframe, s.Parameters,
		s.ScheduleType, s.ScheduleConfig, s.BacktestMetrics,
		s.RiskScore, s.RiskLevel, s.RiskReasons, s.RiskWarnings,
		s.LastBacktestAt, s.IsActive, s.LastRunAt, s.NextRunAt,
		s.RunCount, s.LastError, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}
	return nil
}

// UpdateRiskAssessment updates the backtest metrics and risk assessment for a schedule.
func (r *StrategyScheduleRepository) UpdateRiskAssessment(ctx context.Context, id uuid.UUID, a *model.RiskAssessment, m *model.BacktestMetrics) error {
	now := time.Now()

	tmp := &model.StrategySchedule{}
	if err := tmp.SetBacktestMetrics(m); err != nil {
		return fmt.Errorf("encode backtest metrics: %w", err)
	}
	if err := tmp.SetRiskReasons(a.Reasons); err != nil {
		return fmt.Errorf("encode risk reasons: %w", err)
	}
	if err := tmp.SetRiskWarnings(a.Warnings); err != nil {
		return fmt.Errorf("encode risk warnings: %w", err)
	}

	_, err := r.db.Exec(ctx,
		`UPDATE strategy_schedules SET
			backtest_metrics = $2, risk_score = $3, risk_level = $4,
			risk_reasons = $5, risk_warnings = $6, last_backtest_at = $7, updated_at = $8
		WHERE id = $1`,
		id, tmp.BacktestMetrics, a.Score, a.Level, tmp.RiskReasons, tmp.RiskWarnings, now, now,
	)
	if err != nil {
		return fmt.Errorf("update risk assessment: %w", err)
	}
	return nil
}

// UpdateNextRunAt sets the next scheduled run time.
func (r *StrategyScheduleRepository) UpdateNextRunAt(ctx context.Context, id uuid.UUID, nextRunAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE strategy_schedules SET next_run_at = $2, updated_at = $3 WHERE id = $1`,
		id, nextRunAt, time.Now())
	if err != nil {
		return fmt.Errorf("update next run at: %w", err)
	}
	return nil
}

// UpdateLastRun records the last run time and error state, incrementing run_count.
func (r *StrategyScheduleRepository) UpdateLastRun(ctx context.Context, id uuid.UUID, runErr error) error {
	now := time.Now()
	var errMsg string
	if runErr != nil {
		errMsg = runErr.Error()
	}
	_, err := r.db.Exec(ctx,
		`UPDATE strategy_schedules SET last_run_at = $2, run_count = run_count + 1, last_error = $3, updated_at = $4 WHERE id = $1`,
		id, now, errMsg, now)
	if err != nil {
		return fmt.Errorf("update last run: %w", err)
	}
	return nil
}

// SetActive toggles the is_active flag and increments enable_count on fresh activation.
func (r *StrategyScheduleRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	_, err := r.db.Exec(ctx,
		`UPDATE strategy_schedules SET
			is_active = $2,
			enable_count = enable_count + CASE WHEN $2 = true AND is_active = false THEN 1 ELSE 0 END,
			updated_at = $3
		WHERE id = $1`,
		id, active, time.Now())
	if err != nil {
		return fmt.Errorf("set schedule active: %w", err)
	}
	return nil
}

// Delete removes a schedule and its execution logs.
func (r *StrategyScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM strategy_execution_logs WHERE schedule_id = $1`, id); err != nil {
		if err != nil {
			return fmt.Errorf("delete schedule: %w", err)
		}
		return nil
	}
	ct, err := r.db.Exec(ctx, `DELETE FROM strategy_schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrScheduleNotFound
	}
	return nil
}
