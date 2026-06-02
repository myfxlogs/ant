package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ScheduleRow struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	TemplateID      uuid.UUID
	AccountID       uuid.UUID
	Name            string
	Symbol          string
	Timeframe       string
	Parameters      []byte
	ScheduleType    string
	ScheduleConfig  []byte
	BacktestMetrics []byte
	RiskScore       *int32
	RiskLevel       string
	RiskReasons     []byte
	RiskWarnings    []byte
	LastBacktestAt  *time.Time
	IsActive        bool
	LastRunAt       *time.Time
	NextRunAt       *time.Time
	RunCount        int32
	LastError       string
	EnableCount     int32
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (s *StrategySvc) ListSchedules(ctx context.Context, userID uuid.UUID) ([]ScheduleRow, error) {
	rows, err := s.pg.Query(ctx,
		`SELECT id, user_id, template_id, account_id, name, symbol, timeframe, parameters, schedule_type, schedule_config,
		 backtest_metrics, risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
		 is_active, last_run_at, next_run_at, run_count, last_error, enable_count, created_at, updated_at
		 FROM strategy_schedules WHERE user_id = $1 ORDER BY created_at DESC LIMIT 100`, userID)
	if err != nil {
		return nil, fmt.Errorf("ListSchedules: %w", err)
	}
	defer rows.Close()
	return scanScheduleRows(rows)
}

func (s *StrategySvc) GetSchedule(ctx context.Context, id, userID uuid.UUID) (*ScheduleRow, error) {
	var r ScheduleRow
	err := s.pg.QueryRow(ctx,
		`SELECT id, user_id, template_id, account_id, name, symbol, timeframe, parameters, schedule_type, schedule_config,
		 backtest_metrics, risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
		 is_active, last_run_at, next_run_at, run_count, last_error, enable_count, created_at, updated_at
		 FROM strategy_schedules WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&r.ID, &r.UserID, &r.TemplateID, &r.AccountID, &r.Name, &r.Symbol, &r.Timeframe,
		&r.Parameters, &r.ScheduleType, &r.ScheduleConfig,
		&r.BacktestMetrics, &r.RiskScore, &r.RiskLevel, &r.RiskReasons, &r.RiskWarnings, &r.LastBacktestAt,
		&r.IsActive, &r.LastRunAt, &r.NextRunAt, &r.RunCount, &r.LastError, &r.EnableCount, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, fmt.Errorf("GetSchedule: %w", err)
	}
	return &r, nil
}

func (s *StrategySvc) CreateSchedule(ctx context.Context, r *ScheduleRow) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.Parameters == nil {
		r.Parameters = []byte("{}")
	}
	if r.ScheduleConfig == nil {
		r.ScheduleConfig = []byte("{}")
	}
	if r.RiskReasons == nil {
		r.RiskReasons = []byte("[]")
	}
	if r.RiskWarnings == nil {
		r.RiskWarnings = []byte("[]")
	}
	_, err := s.pg.Exec(ctx,
		`INSERT INTO strategy_schedules (id, user_id, template_id, account_id, name, symbol, timeframe, parameters, schedule_type, schedule_config,
		 backtest_metrics, risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
		 is_active, last_run_at, next_run_at, run_count, last_error, enable_count, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`,
		r.ID, r.UserID, r.TemplateID, r.AccountID, r.Name, r.Symbol, r.Timeframe,
		r.Parameters, r.ScheduleType, r.ScheduleConfig,
		r.BacktestMetrics, r.RiskScore, r.RiskLevel, r.RiskReasons, r.RiskWarnings, r.LastBacktestAt,
		r.IsActive, r.LastRunAt, r.NextRunAt, r.RunCount, r.LastError, r.EnableCount, r.CreatedAt, r.UpdatedAt)
	if err != nil {
		return fmt.Errorf("CreateSchedule: %w", err)
	}
	return nil
}

func (s *StrategySvc) UpdateSchedule(ctx context.Context, r *ScheduleRow) error {
	r.UpdatedAt = time.Now()
	_, err := s.pg.Exec(ctx,
		`UPDATE strategy_schedules SET name=$2, symbol=$3, timeframe=$4, parameters=$5, schedule_type=$6, schedule_config=$7,
		 backtest_metrics=$8, risk_score=$9, risk_level=$10, risk_reasons=$11, risk_warnings=$12, last_backtest_at=$13,
		 is_active=$14, last_run_at=$15, next_run_at=$16, run_count=$17, last_error=$18, updated_at=$19 WHERE id=$1 AND user_id=$20`,
		r.ID, r.Name, r.Symbol, r.Timeframe, r.Parameters, r.ScheduleType, r.ScheduleConfig,
		r.BacktestMetrics, r.RiskScore, r.RiskLevel, r.RiskReasons, r.RiskWarnings, r.LastBacktestAt,
		r.IsActive, r.LastRunAt, r.NextRunAt, r.RunCount, r.LastError, r.UpdatedAt, r.UserID)
	if err != nil {
		return fmt.Errorf("UpdateSchedule: %w", err)
	}
	return nil
}

func (s *StrategySvc) DeleteSchedule(ctx context.Context, id, userID uuid.UUID) error {
	s.pg.Exec(ctx, `DELETE FROM strategy_execution_logs WHERE schedule_id = $1`, id)
	tag, err := s.pg.Exec(ctx, `DELETE FROM strategy_schedules WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("DeleteSchedule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrScheduleNotFound
	}
	return nil
}

func (s *StrategySvc) SetScheduleActive(ctx context.Context, id, userID uuid.UUID, active bool) error {
	_, err := s.pg.Exec(ctx,
		`UPDATE strategy_schedules SET is_active=$2, enable_count=enable_count+CASE WHEN $2=true AND is_active=false THEN 1 ELSE 0 END, updated_at=$3 WHERE id=$1 AND user_id=$4`,
		id, active, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("SetScheduleActive: %w", err)
	}
	return nil
}

func scanScheduleRows(rows pgx.Rows) ([]ScheduleRow, error) {
	var out []ScheduleRow
	for rows.Next() {
		var r ScheduleRow
		err := rows.Scan(&r.ID, &r.UserID, &r.TemplateID, &r.AccountID, &r.Name, &r.Symbol, &r.Timeframe,
			&r.Parameters, &r.ScheduleType, &r.ScheduleConfig,
			&r.BacktestMetrics, &r.RiskScore, &r.RiskLevel, &r.RiskReasons, &r.RiskWarnings, &r.LastBacktestAt,
			&r.IsActive, &r.LastRunAt, &r.NextRunAt, &r.RunCount, &r.LastError, &r.EnableCount, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan schedule row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
