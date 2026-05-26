package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTemplateNotFound = errors.New("template not found")
	ErrScheduleNotFound = errors.New("schedule not found")
	ErrSignalNotFound   = errors.New("signal not found")
)

type StrategySvc struct {
	pg *pgxpool.Pool
}

func NewStrategySvc(pg *pgxpool.Pool) *StrategySvc {
	return &StrategySvc{pg: pg}
}

// --- Templates ---

type TemplateRow struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	Description string
	Code        string
	Status      string
	Parameters  []byte
	IsPublic    bool
	IsSystem    bool
	Tags        []string
	UseCount    int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *StrategySvc) ListTemplates(ctx context.Context, userID uuid.UUID) ([]TemplateRow, error) {
	rows, err := s.pg.Query(ctx,
		`SELECT id, user_id, name, description, code, status, parameters, is_public, is_system, tags, use_count, created_at, updated_at
		 FROM strategy_templates WHERE (user_id = $1 OR is_public = true) AND status != 'canceled' ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()
	return scanTemplateRows(rows)
}

func (s *StrategySvc) GetTemplate(ctx context.Context, id uuid.UUID) (*TemplateRow, error) {
	var t TemplateRow
	err := s.pg.QueryRow(ctx,
		`SELECT id, user_id, name, description, code, status, parameters, is_public, is_system, tags, use_count, created_at, updated_at
		 FROM strategy_templates WHERE id = $1`, id,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.Description, &t.Code, &t.Status, &t.Parameters, &t.IsPublic, &t.IsSystem, &t.Tags, &t.UseCount, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("GetTemplate: %w", err)
	}
	return &t, nil
}

func (s *StrategySvc) CreateTemplate(ctx context.Context, t *TemplateRow) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Parameters == nil {
		t.Parameters = []byte("[]")
	}
	if t.Tags == nil {
		t.Tags = []string{}
	}
	_, err := s.pg.Exec(ctx,
		`INSERT INTO strategy_templates (id, user_id, name, description, code, status, parameters, is_public, is_system, tags, use_count, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		t.ID, t.UserID, t.Name, t.Description, t.Code, t.Status, t.Parameters, t.IsPublic, t.IsSystem, t.Tags, t.UseCount, t.CreatedAt, t.UpdatedAt)
	if err != nil { return fmt.Errorf("CreateTemplate: %w", err) }
	return nil
}

func (s *StrategySvc) UpdateTemplate(ctx context.Context, t *TemplateRow) error {
	t.UpdatedAt = time.Now()
	_, err := s.pg.Exec(ctx,
		`UPDATE strategy_templates SET name=$2, description=$3, code=$4, status=$5, parameters=$6, is_public=$7, tags=$8, updated_at=$9 WHERE id=$1`,
		t.ID, t.Name, t.Description, t.Code, t.Status, t.Parameters, t.IsPublic, t.Tags, t.UpdatedAt)
	if err != nil { return fmt.Errorf("UpdateTemplate: %w", err) }
	return nil
}

func (s *StrategySvc) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pg.Exec(ctx, `DELETE FROM strategy_templates WHERE id=$1 AND is_system=false`, id)
	if err != nil {
		return fmt.Errorf("DeleteTemplate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *StrategySvc) SetTemplateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := s.pg.Exec(ctx, `UPDATE strategy_templates SET status=$2, updated_at=$3 WHERE id=$1`, id, status, time.Now())
	if err != nil { return fmt.Errorf("SetTemplateStatus: %w", err) }
	return nil
}

// --- Schedules ---

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
		 FROM strategy_schedules WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("ListSchedules: %w", err)
	}
	defer rows.Close()
	return scanScheduleRows(rows)
}

func (s *StrategySvc) GetSchedule(ctx context.Context, id uuid.UUID) (*ScheduleRow, error) {
	var r ScheduleRow
	err := s.pg.QueryRow(ctx,
		`SELECT id, user_id, template_id, account_id, name, symbol, timeframe, parameters, schedule_type, schedule_config,
		 backtest_metrics, risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
		 is_active, last_run_at, next_run_at, run_count, last_error, enable_count, created_at, updated_at
		 FROM strategy_schedules WHERE id = $1`, id,
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
	if err != nil { return fmt.Errorf("CreateSchedule: %w", err) }
	return nil
}

func (s *StrategySvc) UpdateSchedule(ctx context.Context, r *ScheduleRow) error {
	r.UpdatedAt = time.Now()
	_, err := s.pg.Exec(ctx,
		`UPDATE strategy_schedules SET name=$2, symbol=$3, timeframe=$4, parameters=$5, schedule_type=$6, schedule_config=$7,
		 backtest_metrics=$8, risk_score=$9, risk_level=$10, risk_reasons=$11, risk_warnings=$12, last_backtest_at=$13,
		 is_active=$14, last_run_at=$15, next_run_at=$16, run_count=$17, last_error=$18, updated_at=$19 WHERE id=$1`,
		r.ID, r.Name, r.Symbol, r.Timeframe, r.Parameters, r.ScheduleType, r.ScheduleConfig,
		r.BacktestMetrics, r.RiskScore, r.RiskLevel, r.RiskReasons, r.RiskWarnings, r.LastBacktestAt,
		r.IsActive, r.LastRunAt, r.NextRunAt, r.RunCount, r.LastError, r.UpdatedAt)
	if err != nil { return fmt.Errorf("UpdateSchedule: %w", err) }
	return nil
}

func (s *StrategySvc) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	s.pg.Exec(ctx, `DELETE FROM strategy_execution_logs WHERE schedule_id = $1`, id)
	tag, err := s.pg.Exec(ctx, `DELETE FROM strategy_schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("DeleteSchedule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrScheduleNotFound
	}
	return nil
}

func (s *StrategySvc) SetScheduleActive(ctx context.Context, id uuid.UUID, active bool) error {
	_, err := s.pg.Exec(ctx,
		`UPDATE strategy_schedules SET is_active=$2, enable_count=enable_count+CASE WHEN $2=true AND is_active=false THEN 1 ELSE 0 END, updated_at=$3 WHERE id=$1`,
		id, active, time.Now())
	if err != nil { return fmt.Errorf("SetScheduleActive: %w", err) }
	return nil
}

// --- Signals ---

type SignalRow struct {
	ID         uuid.UUID
	AccountID  uuid.UUID
	Symbol     string
	SignalType string
	Volume     float64
	Price      float64
	StopLoss   float64
	TakeProfit float64
	Reason     string
	Status     string
	ExecutedAt *time.Time
	Ticket     int64
	Profit     float64
	CreatedAt  time.Time
}

func (s *StrategySvc) ListSignals(ctx context.Context, accountID uuid.UUID, status string) ([]SignalRow, error) {
	var rows pgx.Rows
	var err error
	if accountID == uuid.Nil && status == "" {
		rows, err = s.pg.Query(ctx, `SELECT id, account_id, symbol, signal_type, volume, price, stop_loss, take_profit, reason, status, executed_at, ticket, profit, created_at FROM strategy_signals ORDER BY created_at DESC LIMIT 100`)
	} else if status == "" {
		rows, err = s.pg.Query(ctx, `SELECT id, account_id, symbol, signal_type, volume, price, stop_loss, take_profit, reason, status, executed_at, ticket, profit, created_at FROM strategy_signals WHERE account_id = $1 ORDER BY created_at DESC LIMIT 100`, accountID)
	} else if accountID == uuid.Nil {
		rows, err = s.pg.Query(ctx, `SELECT id, account_id, symbol, signal_type, volume, price, stop_loss, take_profit, reason, status, executed_at, ticket, profit, created_at FROM strategy_signals WHERE status = $1 ORDER BY created_at DESC LIMIT 100`, status)
	} else {
		rows, err = s.pg.Query(ctx, `SELECT id, account_id, symbol, signal_type, volume, price, stop_loss, take_profit, reason, status, executed_at, ticket, profit, created_at FROM strategy_signals WHERE account_id = $1 AND status = $2 ORDER BY created_at DESC LIMIT 100`, accountID, status)
	}
	if err != nil {
		return nil, fmt.Errorf("list signals: %w", err)
	}
	defer rows.Close()
	return scanSignalRows(rows)
}

func (s *StrategySvc) GetSignal(ctx context.Context, id uuid.UUID) (*SignalRow, error) {
	var r SignalRow
	err := s.pg.QueryRow(ctx,
		`SELECT id, account_id, symbol, signal_type, volume, price, stop_loss, take_profit, reason, status, executed_at, ticket, profit, created_at FROM strategy_signals WHERE id = $1`, id,
	).Scan(&r.ID, &r.AccountID, &r.Symbol, &r.SignalType, &r.Volume, &r.Price, &r.StopLoss, &r.TakeProfit, &r.Reason, &r.Status, &r.ExecutedAt, &r.Ticket, &r.Profit, &r.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSignalNotFound
		}
		return nil, fmt.Errorf("GetSignal: %w", err)
	}
	return &r, nil
}

func (s *StrategySvc) ExecuteSignal(ctx context.Context, signalID uuid.UUID) (*SignalRow, error) {
	now := time.Now()
	tag, err := s.pg.Exec(ctx,
		`UPDATE strategy_signals SET status='executed', executed_at=$2 WHERE id=$1 AND status='pending'`, signalID, now)
	if err != nil {
		return nil, fmt.Errorf("ExecuteSignal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrSignalNotFound
	}
	return s.GetSignal(ctx, signalID)
}

func (s *StrategySvc) ConfirmSignal(ctx context.Context, signalID uuid.UUID) error {
	tag, err := s.pg.Exec(ctx,
		`UPDATE strategy_signals SET status='confirmed' WHERE id=$1 AND status='pending'`, signalID)
	if err != nil {
		return fmt.Errorf("ConfirmSignal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSignalNotFound
	}
	return nil
}

func (s *StrategySvc) CancelSignal(ctx context.Context, signalID uuid.UUID) error {
	tag, err := s.pg.Exec(ctx,
		`UPDATE strategy_signals SET status='cancelled' WHERE id=$1 AND status IN ('pending','confirmed')`, signalID)
	if err != nil {
		return fmt.Errorf("CancelSignal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSignalNotFound
	}
	return nil
}

// --- Scanners ---

func scanTemplateRows(rows pgx.Rows) ([]TemplateRow, error) {
	var out []TemplateRow
	for rows.Next() {
		var t TemplateRow
		err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Description, &t.Code, &t.Status, &t.Parameters, &t.IsPublic, &t.IsSystem, &t.Tags, &t.UseCount, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan template row: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
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

func scanSignalRows(rows pgx.Rows) ([]SignalRow, error) {
	var out []SignalRow
	for rows.Next() {
		var r SignalRow
		err := rows.Scan(&r.ID, &r.AccountID, &r.Symbol, &r.SignalType, &r.Volume, &r.Price, &r.StopLoss, &r.TakeProfit, &r.Reason, &r.Status, &r.ExecutedAt, &r.Ticket, &r.Profit, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan signal row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// --- JSON helpers ---

func jsonBytes(v any) []byte {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}

func mustParseJSON[T any](raw []byte, fallback T) T {
	if len(raw) == 0 {
		return fallback
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return fallback
	}
	return out
}
