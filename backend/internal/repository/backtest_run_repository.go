package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BacktestRunRepository struct {
	db *pgxpool.Pool
}

type BacktestRun struct {
	ID                  uuid.UUID  `db:"id"`
	UserID              uuid.UUID  `db:"user_id"`
	AccountID           uuid.UUID  `db:"account_id"`
	Symbol              string     `db:"symbol"`
	Timeframe           string     `db:"timeframe"`
	DatasetID           *uuid.UUID `db:"dataset_id"`
	TemplateID          *uuid.UUID `db:"template_id"`
	TemplateDraftID     *uuid.UUID `db:"template_draft_id"`
	Mode                string     `db:"mode"`
	FromTs              *time.Time `db:"from_ts"`
	ToTs                *time.Time `db:"to_ts"`
	CancelRequestedAt   *time.Time `db:"cancel_requested_at"`
	LeaseUntil          *time.Time `db:"lease_until"`
	StrategyCodeHash    string     `db:"strategy_code_hash"`
	PythonServiceVersion *string   `db:"python_service_version"`
	CostModelSnapshot   []byte     `db:"cost_model_snapshot"`
	Metrics             []byte     `db:"metrics"`
	EquityCurve         []byte     `db:"equity_curve"`
	Status              string     `db:"status"`
	Error               string     `db:"error"`
	StartedAt           *time.Time `db:"started_at"`
	FinishedAt          *time.Time `db:"finished_at"`
	StrategyCode        *string    `db:"strategy_code"`
	InitialCapital      *float64   `db:"initial_capital"`
	ExtraSymbols        []string   `db:"extra_symbols"`
	CreatedAt           time.Time  `db:"created_at"`
}

func NewBacktestRunRepository(db *pgxpool.Pool) *BacktestRunRepository {
	return &BacktestRunRepository{db: db}
}

func (r *BacktestRunRepository) Create(ctx context.Context, run *BacktestRun) (uuid.UUID, error) {
	query := `
		INSERT INTO backtest_runs (
			id, user_id, account_id, symbol, timeframe, dataset_id, template_id, template_draft_id,
			mode, from_ts, to_ts,
			cancel_requested_at, lease_until,
			strategy_code_hash, python_service_version,
			cost_model_snapshot, metrics, equity_curve,
			status, error, started_at, finished_at, strategy_code, initial_capital,
			extra_symbols,
			created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,CURRENT_TIMESTAMP)
		RETURNING id
	`
	id := run.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	var out uuid.UUID
	err := r.db.QueryRow(ctx, query,
		id,
		run.UserID,
		run.AccountID,
		run.Symbol,
		run.Timeframe,
		run.DatasetID,
		run.TemplateID,
		run.TemplateDraftID,
		run.Mode,
		run.FromTs,
		run.ToTs,
		run.CancelRequestedAt,
		run.LeaseUntil,
		run.StrategyCodeHash,
		run.PythonServiceVersion,
		run.CostModelSnapshot,
		run.Metrics,
		run.EquityCurve,
		run.Status,
		run.Error,
		run.StartedAt,
		run.FinishedAt,
		run.StrategyCode,
		run.InitialCapital,
		run.ExtraSymbols,
	).Scan(&out)
	return out, err
}

func (r *BacktestRunRepository) GetByID(ctx context.Context, userID, runID uuid.UUID) (*BacktestRun, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var out BacktestRun
	err := r.db.QueryRow(ctx,
		`SELECT
			id, user_id, account_id, symbol, timeframe, dataset_id, template_id, template_draft_id,
			mode, from_ts, to_ts,
			cancel_requested_at, lease_until,
			strategy_code_hash, python_service_version,
			cost_model_snapshot, metrics, equity_curve,
			status, error, started_at, finished_at, strategy_code, initial_capital,
			extra_symbols,
			created_at
		FROM backtest_runs
		WHERE id = $1 AND user_id = $2`,
		runID, userID,
	).Scan(
		&out.ID, &out.UserID, &out.AccountID, &out.Symbol, &out.Timeframe, &out.DatasetID, &out.TemplateID, &out.TemplateDraftID,
		&out.Mode, &out.FromTs, &out.ToTs,
		&out.CancelRequestedAt, &out.LeaseUntil,
		&out.StrategyCodeHash, &out.PythonServiceVersion,
		&out.CostModelSnapshot, &out.Metrics, &out.EquityCurve,
		&out.Status, &out.Error, &out.StartedAt, &out.FinishedAt, &out.StrategyCode, &out.InitialCapital,
		&out.ExtraSymbols,
		&out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *BacktestRunRepository) ListByUser(ctx context.Context, userID uuid.UUID, accountID *uuid.UUID, limit, offset int) ([]*BacktestRun, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	if accountID == nil || *accountID == uuid.Nil {
		return r.listByUserAll(ctx, userID, limit, offset)
	}
	return r.listByUserAccount(ctx, userID, *accountID, limit, offset)
}

func (r *BacktestRunRepository) listByUserAll(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*BacktestRun, error) {
	return r.scanBacktestRunRows(ctx,
		`SELECT id, user_id, account_id, symbol, timeframe, dataset_id, template_id, template_draft_id,
			mode, from_ts, to_ts, cancel_requested_at, lease_until,
			strategy_code_hash, python_service_version,
			cost_model_snapshot, metrics, equity_curve,
			status, error, started_at, finished_at, strategy_code, initial_capital,
			extra_symbols, created_at
		FROM backtest_runs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
}

func (r *BacktestRunRepository) listByUserAccount(ctx context.Context, userID, accountID uuid.UUID, limit, offset int) ([]*BacktestRun, error) {
	return r.scanBacktestRunRows(ctx,
		`SELECT id, user_id, account_id, symbol, timeframe, dataset_id, template_id, template_draft_id,
			mode, from_ts, to_ts, cancel_requested_at, lease_until,
			strategy_code_hash, python_service_version,
			cost_model_snapshot, metrics, equity_curve,
			status, error, started_at, finished_at, strategy_code, initial_capital,
			extra_symbols, created_at
		FROM backtest_runs
		WHERE user_id = $1 AND account_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`, userID, accountID, limit, offset)
}

func (r *BacktestRunRepository) scanBacktestRunRows(ctx context.Context, query string, args ...interface{}) ([]*BacktestRun, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*BacktestRun
	for rows.Next() {
		var out BacktestRun
		if err := rows.Scan(
			&out.ID, &out.UserID, &out.AccountID, &out.Symbol, &out.Timeframe, &out.DatasetID, &out.TemplateID, &out.TemplateDraftID,
			&out.Mode, &out.FromTs, &out.ToTs,
			&out.CancelRequestedAt, &out.LeaseUntil,
			&out.StrategyCodeHash, &out.PythonServiceVersion,
			&out.CostModelSnapshot, &out.Metrics, &out.EquityCurve,
			&out.Status, &out.Error, &out.StartedAt, &out.FinishedAt, &out.StrategyCode, &out.InitialCapital,
			&out.ExtraSymbols,
			&out.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &out)
	}
	return items, rows.Err()
}

func (r *BacktestRunRepository) Delete(ctx context.Context, userID, runID uuid.UUID) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("repository not initialized")
	}
	query := `
		DELETE FROM backtest_runs
		WHERE id = $1 AND user_id = $2
	`
	ct, err := r.db.Exec(ctx, query, runID, userID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}
