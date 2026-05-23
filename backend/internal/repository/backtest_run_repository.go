package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type BacktestRunRepository struct {
	db *sqlx.DB
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
	ExtraSymbols        pq.StringArray `db:"extra_symbols"`
	CreatedAt           time.Time  `db:"created_at"`
}

func NewBacktestRunRepository(db *sqlx.DB) *BacktestRunRepository {
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
	err := r.db.QueryRowxContext(ctx, query,
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
	query := `
		SELECT
			id, user_id, account_id, symbol, timeframe, dataset_id, template_id, template_draft_id,
			mode, from_ts, to_ts,
			cancel_requested_at, lease_until,
			strategy_code_hash, python_service_version,
			cost_model_snapshot, metrics, equity_curve,
			status, error, started_at, finished_at, strategy_code, initial_capital,
			extra_symbols,
			created_at
		FROM backtest_runs
		WHERE id = $1 AND user_id = $2
	`
	err := r.db.GetContext(ctx, &out, query, runID, userID)
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

	items := []*BacktestRun{}
	if accountID == nil || *accountID == uuid.Nil {
		query := `
			SELECT
				id, user_id, account_id, symbol, timeframe, dataset_id, template_id, template_draft_id,
				mode, from_ts, to_ts,
				cancel_requested_at, lease_until,
				strategy_code_hash, python_service_version,
				cost_model_snapshot, metrics, equity_curve,
				status, error, started_at, finished_at, strategy_code, initial_capital,
				extra_symbols,
				created_at
			FROM backtest_runs
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		err := r.db.SelectContext(ctx, &items, query, userID, limit, offset)
		return items, err
	}

	query := `
		SELECT
			id, user_id, account_id, symbol, timeframe, dataset_id, template_id, template_draft_id,
			mode, from_ts, to_ts,
			cancel_requested_at, lease_until,
			strategy_code_hash, python_service_version,
			cost_model_snapshot, metrics, equity_curve,
			status, error, started_at, finished_at, strategy_code, initial_capital,
			extra_symbols,
			created_at
		FROM backtest_runs
		WHERE user_id = $1 AND account_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	err := r.db.SelectContext(ctx, &items, query, userID, *accountID, limit, offset)
	return items, err
}

func (r *BacktestRunRepository) ClaimNextForWork(ctx context.Context, leaseUntil time.Time) (*BacktestRun, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var out BacktestRun
	// Claim a single run that is pending or a stale running lease.
	// Also claim cancel-requested runs so the worker can finalize cancellation.
	query := `
		WITH candidate AS (
			SELECT b.id
			FROM backtest_runs b
			WHERE
				(status = 'PENDING')
				OR (status = 'CANCEL_REQUESTED' AND finished_at IS NULL)
				OR (
					status = 'RUNNING'
					AND finished_at IS NULL
					AND lease_until IS NOT NULL
					AND lease_until < CURRENT_TIMESTAMP
				)
			ORDER BY (status = 'CANCEL_REQUESTED') DESC, created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE backtest_runs b
		SET
			status = 'RUNNING',
			started_at = COALESCE(b.started_at, CURRENT_TIMESTAMP),
			lease_until = $1
		FROM candidate c
		WHERE b.id = c.id
		RETURNING
			b.id, b.user_id, b.account_id, b.symbol, b.timeframe, b.dataset_id, b.template_id, b.template_draft_id,
			b.mode, b.from_ts, b.to_ts,
			b.cancel_requested_at, b.lease_until,
			b.strategy_code_hash, b.python_service_version,
			b.cost_model_snapshot, b.metrics, b.equity_curve,
			b.status, b.error, b.started_at, b.finished_at, b.strategy_code, b.initial_capital,
			b.extra_symbols,
			b.created_at
	`
	err := r.db.GetContext(ctx, &out, query, leaseUntil)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *BacktestRunRepository) ExtendLease(ctx context.Context, userID, runID uuid.UUID, leaseUntil time.Time) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	query := `
		UPDATE backtest_runs
		SET lease_until = $3
		WHERE id = $1 AND user_id = $2 AND finished_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, runID, userID, leaseUntil)
	return err
}

func (r *BacktestRunRepository) RequestCancel(ctx context.Context, userID, runID uuid.UUID) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	query := `
		UPDATE backtest_runs
		SET
			status = CASE
				WHEN status IN ('SUCCEEDED','FAILED','CANCELED') THEN status
				ELSE 'CANCEL_REQUESTED'
			END,
			cancel_requested_at = COALESCE(cancel_requested_at, CURRENT_TIMESTAMP)
		WHERE id = $1 AND user_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, runID, userID)
	return err
}

func (r *BacktestRunRepository) Delete(ctx context.Context, userID, runID uuid.UUID) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("repository not initialized")
	}
	query := `
		DELETE FROM backtest_runs
		WHERE id = $1 AND user_id = $2
	`
	res, err := r.db.ExecContext(ctx, query, runID, userID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *BacktestRunRepository) CountActiveByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("repository not initialized")
	}
	var n int
	query := `
		SELECT COUNT(1)
		FROM backtest_runs
		WHERE user_id = $1 AND status IN ('PENDING','RUNNING','CANCEL_REQUESTED')
	`
	err := r.db.GetContext(ctx, &n, query, userID)
	return n, err
}

func (r *BacktestRunRepository) CountPendingByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("repository not initialized")
	}
	var n int
	query := `
		SELECT COUNT(1)
		FROM backtest_runs
		WHERE user_id = $1 AND status = 'PENDING'
	`
	err := r.db.GetContext(ctx, &n, query, userID)
	return n, err
}

func (r *BacktestRunRepository) CountRecentStartsByUser(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("repository not initialized")
	}
	var n int
	query := `
		SELECT COUNT(1)
		FROM backtest_runs
		WHERE user_id = $1 AND created_at >= $2
	`
	err := r.db.GetContext(ctx, &n, query, userID, since)
	return n, err
}

func (r *BacktestRunRepository) CountActiveByAccount(ctx context.Context, userID, accountID uuid.UUID) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("repository not initialized")
	}
	var n int
	query := `
		SELECT COUNT(1)
		FROM backtest_runs
		WHERE user_id = $1 AND account_id = $2 AND status IN ('PENDING','RUNNING','CANCEL_REQUESTED')
	`
	err := r.db.GetContext(ctx, &n, query, userID, accountID)
	return n, err
}

func (r *BacktestRunRepository) CountPendingByAccount(ctx context.Context, userID, accountID uuid.UUID) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("repository not initialized")
	}
	var n int
	query := `
		SELECT COUNT(1)
		FROM backtest_runs
		WHERE user_id = $1 AND account_id = $2 AND status = 'PENDING'
	`
	err := r.db.GetContext(ctx, &n, query, userID, accountID)
	return n, err
}

func (r *BacktestRunRepository) GetStatusAndCancelRequestedAt(ctx context.Context, userID, runID uuid.UUID) (string, *time.Time, error) {
	if r == nil || r.db == nil {
		return "", nil, errors.New("repository not initialized")
	}
	var status string
	var cancelAt *time.Time
	query := `
		SELECT status, cancel_requested_at
		FROM backtest_runs
		WHERE id = $1 AND user_id = $2
	`
	err := r.db.QueryRowxContext(ctx, query, runID, userID).Scan(&status, &cancelAt)
	return status, cancelAt, err
}

func (r *BacktestRunRepository) UpdateAsyncFields(ctx context.Context, userID, runID uuid.UUID, status string, errMsg string, startedAt, finishedAt *time.Time, metrics, equityCurve []byte) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	query := `
		UPDATE backtest_runs
		SET
			status = COALESCE(NULLIF($3, ''), status),
			error = $4,
			started_at = COALESCE($5, started_at),
			finished_at = COALESCE($6, finished_at),
			lease_until = CASE
				WHEN COALESCE(NULLIF($3, ''), status) IN ('SUCCEEDED','FAILED','CANCELED') THEN NULL
				ELSE lease_until
			END,
			metrics = COALESCE($7, metrics),
			equity_curve = COALESCE($8, equity_curve)
		WHERE id = $1 AND user_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, runID, userID, status, errMsg, startedAt, finishedAt, metrics, equityCurve)
	return err
}
