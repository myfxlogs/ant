package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"anttrader/internal/model"
)

var ErrScheduleNotFound = errors.New("strategy schedule not found")

type StrategyScheduleRepository struct {
	db *sqlx.DB
}

func NewStrategyScheduleRepository(db *sqlx.DB) *StrategyScheduleRepository {
	return &StrategyScheduleRepository{db: db}
}

func (r *StrategyScheduleRepository) Create(ctx context.Context, s *model.StrategySchedule) error {
	now := time.Now()
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	s.CreatedAt = now
	s.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
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
	return err
}

func (r *StrategyScheduleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
	var s model.StrategySchedule
	if err := r.db.GetContext(ctx, &s, `SELECT * FROM strategy_schedules WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *StrategyScheduleRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.StrategySchedule, error) {
	var schedules []*model.StrategySchedule
	err := r.db.SelectContext(ctx, &schedules,
		`SELECT * FROM strategy_schedules WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	return schedules, err
}

func (r *StrategyScheduleRepository) GetByTemplateID(ctx context.Context, templateID uuid.UUID) ([]*model.StrategySchedule, error) {
	var schedules []*model.StrategySchedule
	err := r.db.SelectContext(ctx, &schedules,
		`SELECT * FROM strategy_schedules WHERE template_id = $1 ORDER BY created_at DESC`, templateID)
	return schedules, err
}

func (r *StrategyScheduleRepository) GetByAccountID(ctx context.Context, accountID uuid.UUID) ([]*model.StrategySchedule, error) {
	var schedules []*model.StrategySchedule
	err := r.db.SelectContext(ctx, &schedules,
		`SELECT * FROM strategy_schedules WHERE account_id = $1 ORDER BY created_at DESC`, accountID)
	return schedules, err
}

func (r *StrategyScheduleRepository) GetByUniqueKey(ctx context.Context, userID, accountID, templateID uuid.UUID, symbol, timeframe string) (*model.StrategySchedule, error) {
	var s model.StrategySchedule
	if err := r.db.GetContext(ctx, &s,
		`SELECT * FROM strategy_schedules WHERE user_id = $1 AND account_id = $2 AND template_id = $3 AND symbol = $4 AND timeframe = $5 LIMIT 1`,
		userID, accountID, templateID, symbol, timeframe); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *StrategyScheduleRepository) GetActiveSchedules(ctx context.Context) ([]*model.StrategySchedule, error) {
	var schedules []*model.StrategySchedule
	err := r.db.SelectContext(ctx, &schedules,
		`SELECT * FROM strategy_schedules WHERE is_active = true ORDER BY next_run_at ASC`)
	return schedules, err
}

func (r *StrategyScheduleRepository) GetDueSchedules(ctx context.Context, before time.Time) ([]*model.StrategySchedule, error) {
	var schedules []*model.StrategySchedule
	err := r.db.SelectContext(ctx, &schedules,
		`SELECT * FROM strategy_schedules WHERE is_active = true AND next_run_at IS NOT NULL AND next_run_at <= $1 ORDER BY next_run_at ASC`,
		before)
	return schedules, err
}

func (r *StrategyScheduleRepository) Update(ctx context.Context, s *model.StrategySchedule) error {
	s.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
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
	return err
}

func (r *StrategyScheduleRepository) UpdateRiskAssessment(ctx context.Context, id uuid.UUID, a *model.RiskAssessment, m *model.BacktestMetrics) error {
	now := time.Now()

	metricsJSON, _ := json.Marshal(m)
	reasonsJSON, _ := json.Marshal(a.Reasons)
	warningsJSON, _ := json.Marshal(a.Warnings)

	_, err := r.db.ExecContext(ctx,
		`UPDATE strategy_schedules SET
			backtest_metrics = $2, risk_score = $3, risk_level = $4,
			risk_reasons = $5, risk_warnings = $6, last_backtest_at = $7, updated_at = $8
		WHERE id = $1`,
		id, metricsJSON, a.Score, a.Level, reasonsJSON, warningsJSON, now, now,
	)
	return err
}

func (r *StrategyScheduleRepository) UpdateNextRunAt(ctx context.Context, id uuid.UUID, nextRunAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE strategy_schedules SET next_run_at = $2, updated_at = $3 WHERE id = $1`,
		id, nextRunAt, time.Now())
	return err
}

func (r *StrategyScheduleRepository) UpdateLastRun(ctx context.Context, id uuid.UUID, runErr error) error {
	now := time.Now()
	var errMsg string
	if runErr != nil {
		errMsg = runErr.Error()
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE strategy_schedules SET last_run_at = $2, run_count = run_count + 1, last_error = $3, updated_at = $4 WHERE id = $1`,
		id, now, errMsg, now)
	return err
}

func (r *StrategyScheduleRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE strategy_schedules SET
			is_active = $2,
			enable_count = enable_count + CASE WHEN $2 = true AND is_active = false THEN 1 ELSE 0 END,
			updated_at = $3
		WHERE id = $1`,
		id, active, time.Now())
	return err
}

func (r *StrategyScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM strategy_execution_logs WHERE schedule_id = $1`, id); err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM strategy_schedules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrScheduleNotFound
	}
	return nil
}

func (r *StrategyScheduleRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM strategy_schedules WHERE user_id = $1`, userID)
	return count, err
}

func (r *StrategyScheduleRepository) CountByTemplateID(ctx context.Context, templateID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM strategy_schedules WHERE template_id = $1`, templateID)
	return count, err
}
