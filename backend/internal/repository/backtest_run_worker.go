package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ClaimNextForWork claims a pending/cancel-requested/stale run for execution.
func (r *BacktestRunRepository) ClaimNextForWork(ctx context.Context, leaseUntil time.Time) (*BacktestRun, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var out BacktestRun
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
			b.strategy_code_hash,
			b.cost_model_snapshot,
			b.status, b.error, b.started_at, b.finished_at, b.strategy_code, b.initial_capital,
			b.extra_symbols, b.parameter_overrides, b.proto_response,
			b.commission, b.slippage, b.leverage, b.trade_direction, b.strict_mode, b.config_snapshot,
			b.strategy_id,
			b.created_at
	`
	err := r.db.QueryRow(ctx, query, leaseUntil).Scan(
		&out.ID, &out.UserID, &out.AccountID, &out.Symbol, &out.Timeframe, &out.DatasetID, &out.TemplateID, &out.TemplateDraftID,
		&out.Mode, &out.FromTs, &out.ToTs,
		&out.CancelRequestedAt, &out.LeaseUntil,
		&out.StrategyCodeHash,
		&out.CostModelSnapshot,
		&out.Status, &out.Error, &out.StartedAt, &out.FinishedAt, &out.StrategyCode, &out.InitialCapital,
		&out.ExtraSymbols, &out.ParameterOverrides, &out.ProtoResponse,
		&out.Commission, &out.Slippage, &out.Leverage, &out.TradeDirection, &out.StrictMode, &out.ConfigSnapshot,
		&out.StrategyID,
		&out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ExtendLease extends the lease on a running backtest run.
func (r *BacktestRunRepository) ExtendLease(ctx context.Context, userID, runID uuid.UUID, leaseUntil time.Time) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	query := `
		UPDATE backtest_runs
		SET lease_until = $3
		WHERE id = $1 AND user_id = $2 AND finished_at IS NULL
	`
	_, err := r.db.Exec(ctx, query, runID, userID, leaseUntil)
	if err != nil {
		return fmt.Errorf("extend lease: %w", err)
	}
	return nil
}

// RequestCancel requests cancellation of a backtest run.
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
	_, err := r.db.Exec(ctx, query, runID, userID)
	if err != nil {
		return fmt.Errorf("request cancel: %w", err)
	}
	// Push-first: notify active workers of cancel request.
	_, _ = r.db.Exec(ctx, "SELECT pg_notify('backtest_cancel', $1)", runID.String())
	return nil
}

// CountActiveByUser counts active backtest runs for a user.
func (r *BacktestRunRepository) CountActiveByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("repository not initialized")
	}
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(1) FROM backtest_runs WHERE user_id = $1 AND status IN ('PENDING','RUNNING','CANCEL_REQUESTED')`,
		userID).Scan(&n)
	return n, err
}

// CountPendingByUser counts pending backtest runs for a user.
func (r *BacktestRunRepository) CountPendingByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("repository not initialized")
	}
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(1) FROM backtest_runs WHERE user_id = $1 AND status = 'PENDING'`,
		userID).Scan(&n)
	return n, err
}

// CountRecentStartsByUser counts recently started backtest runs for a user.
func (r *BacktestRunRepository) CountRecentStartsByUser(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("repository not initialized")
	}
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(1) FROM backtest_runs WHERE user_id = $1 AND created_at >= $2`,
		userID, since).Scan(&n)
	return n, err
}

// CountActiveByAccount counts active backtest runs for a specific account.
func (r *BacktestRunRepository) CountActiveByAccount(ctx context.Context, userID, accountID uuid.UUID) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("repository not initialized")
	}
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(1) FROM backtest_runs WHERE user_id = $1 AND account_id = $2 AND status IN ('PENDING','RUNNING','CANCEL_REQUESTED')`,
		userID, accountID).Scan(&n)
	return n, err
}

// CountPendingByAccount counts pending backtest runs for a specific account.
func (r *BacktestRunRepository) CountPendingByAccount(ctx context.Context, userID, accountID uuid.UUID) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("repository not initialized")
	}
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(1) FROM backtest_runs WHERE user_id = $1 AND account_id = $2 AND status = 'PENDING'`,
		userID, accountID).Scan(&n)
	return n, err
}

// GetStatusAndCancelRequestedAt returns the current status and cancel-requested timestamp.
func (r *BacktestRunRepository) GetStatusAndCancelRequestedAt(ctx context.Context, userID, runID uuid.UUID) (string, *time.Time, error) {
	if r == nil || r.db == nil {
		return "", nil, errors.New("repository not initialized")
	}
	var status string
	var cancelAt *time.Time
	err := r.db.QueryRow(ctx,
		`SELECT status, cancel_requested_at FROM backtest_runs WHERE id = $1 AND user_id = $2`,
		runID, userID).Scan(&status, &cancelAt)
	return status, cancelAt, err
}

// UpdateAsyncFields updates status, error, timestamps, and optional columns atomically.
// protoResponse is the serialized ant.v1.ExecuteBacktestResponse proto binary (canonical wire format).
// backtestSnapshot is the serialized ant.v1.BacktestSnapshot proto binary (server-generated, tamper-proof).
func (r *BacktestRunRepository) UpdateAsyncFields(ctx context.Context, userID, runID uuid.UUID, status string, errMsg string, startedAt, finishedAt *time.Time, protoResponse []byte, backtestSnapshot []byte) error {
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
			proto_response = COALESCE($7, proto_response),
			backtest_snapshot = COALESCE($8, backtest_snapshot)
		WHERE id = $1 AND user_id = $2
	`
	_, err := r.db.Exec(ctx, query, runID, userID, status, errMsg, startedAt, finishedAt, protoResponse, backtestSnapshot)
	if err != nil {
		return fmt.Errorf("update async fields: %w", err)
	}
	if status == "SUCCEEDED" || status == "FAILED" || status == "CANCELED" {
		_, _ = r.db.Exec(ctx, "SELECT pg_notify('backtest_status', $1)", runID.String())
	}
	return nil
}
