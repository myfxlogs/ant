package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StrategyRun represents a live/paper strategy run lifecycle record.
type StrategyRun struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	AccountID    string
	Symbol       string
	Timeframe    string
	Mode         string
	StrategyCode string
	Status       string // "running" | "stopped" | "error"
	Error        string
	TotalSignals int
	StartedAt    time.Time
	StoppedAt    *time.Time
}

// StrategyRunRepository provides DB access for strategy_runs.
type StrategyRunRepository struct {
	db *pgxpool.Pool
}

func NewStrategyRunRepository(db *pgxpool.Pool) *StrategyRunRepository {
	return &StrategyRunRepository{db: db}
}

// Create inserts a new strategy run and returns the row with ID + StartedAt.
func (r *StrategyRunRepository) Create(ctx context.Context, run *StrategyRun) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO strategy_runs (user_id, account_id, symbol, timeframe, mode, strategy_code, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, started_at
	`, run.UserID, run.AccountID, run.Symbol, run.Timeframe, run.Mode, run.StrategyCode, run.Status,
	).Scan(&run.ID, &run.StartedAt)
}

// UpdateStopped marks a run as stopped (or errored) and sets stopped_at.
func (r *StrategyRunRepository) UpdateStopped(ctx context.Context, id uuid.UUID, status, runErr string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE strategy_runs SET status = $2, error = $3, stopped_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id, status, runErr)
	return err
}

// IncrementSignalCount atomically bumps total_signals for a run.
func (r *StrategyRunRepository) IncrementSignalCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE strategy_runs SET total_signals = total_signals + 1 WHERE id = $1
	`, id)
	return err
}

// GetByID returns a single strategy run.
func (r *StrategyRunRepository) GetByID(ctx context.Context, id uuid.UUID) (*StrategyRun, error) {
	var run StrategyRun
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, account_id, symbol, timeframe, mode, strategy_code,
		       status, COALESCE(error, ''), total_signals, started_at, stopped_at
		FROM strategy_runs WHERE id = $1
	`, id).Scan(
		&run.ID, &run.UserID, &run.AccountID, &run.Symbol, &run.Timeframe, &run.Mode,
		&run.StrategyCode, &run.Status, &run.Error, &run.TotalSignals, &run.StartedAt, &run.StoppedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStrategyRunNotFound
		}
		return nil, fmt.Errorf("get strategy run: %w", err)
	}
	return &run, nil
}

// ListByUser returns recent strategy runs for a user, paginated.
func (r *StrategyRunRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*StrategyRun, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, account_id, symbol, timeframe, mode, strategy_code,
		       status, COALESCE(error, ''), total_signals, started_at, stopped_at
		FROM strategy_runs WHERE user_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list strategy runs: %w", err)
	}
	defer rows.Close()

	var out []*StrategyRun
	for rows.Next() {
		var run StrategyRun
		if err := rows.Scan(
			&run.ID, &run.UserID, &run.AccountID, &run.Symbol, &run.Timeframe, &run.Mode,
			&run.StrategyCode, &run.Status, &run.Error, &run.TotalSignals, &run.StartedAt, &run.StoppedAt,
		); err != nil {
			return nil, fmt.Errorf("scan strategy run: %w", err)
		}
		out = append(out, &run)
	}
	return out, rows.Err()
}

// ListByAccount returns recent strategy runs for a specific account.
func (r *StrategyRunRepository) ListByAccount(ctx context.Context, accountID string, limit int) ([]*StrategyRun, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, account_id, symbol, timeframe, mode, strategy_code,
		       status, COALESCE(error, ''), total_signals, started_at, stopped_at
		FROM strategy_runs WHERE account_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, accountID, limit)
	if err != nil {
		return nil, fmt.Errorf("list strategy runs by account: %w", err)
	}
	defer rows.Close()

	var out []*StrategyRun
	for rows.Next() {
		var run StrategyRun
		if err := rows.Scan(
			&run.ID, &run.UserID, &run.AccountID, &run.Symbol, &run.Timeframe, &run.Mode,
			&run.StrategyCode, &run.Status, &run.Error, &run.TotalSignals, &run.StartedAt, &run.StoppedAt,
		); err != nil {
			return nil, fmt.Errorf("scan strategy run: %w", err)
		}
		out = append(out, &run)
	}
	return out, rows.Err()
}

// InsertSignal persists a strategy signal linked to a run.
func (r *StrategyRunRepository) InsertSignal(ctx context.Context, params InsertSignalParams) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO strategy_signals (strategy_id, account_id, symbol, signal_type, volume, price, stop_loss, take_profit, reason, status, run_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'executed', $10)
	`,
		params.StrategyID, params.AccountID, params.Symbol, params.SignalType,
		params.Volume, params.Price, params.StopLoss, params.TakeProfit,
		params.Reason, params.RunID,
	)
	return err
}

// InsertSignalParams holds the fields for inserting a strategy signal.
type InsertSignalParams struct {
	StrategyID *uuid.UUID
	AccountID  string
	Symbol     string
	SignalType string
	Volume     string
	Price      string
	StopLoss   string
	TakeProfit string
	Reason     string
	RunID      *uuid.UUID
}

var ErrStrategyRunNotFound = errors.New("strategy run not found")

// CleanupStaleRuns marks all runs with status='running' as 'stopped'.
// Called on server startup to clean up runs orphaned by a crash/restart.
func (r *StrategyRunRepository) CleanupStaleRuns(ctx context.Context) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE strategy_runs
		SET status = 'stopped', error = 'server restarted', stopped_at = CURRENT_TIMESTAMP
		WHERE status = 'running'
	`)
	if err != nil {
		return 0, fmt.Errorf("cleanup stale runs: %w", err)
	}
	return tag.RowsAffected(), nil
}
