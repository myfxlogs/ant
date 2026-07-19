package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"alphaforge/internal/model"
)

// SweepLogRepository manages sweep_logs records.
type SweepLogRepository struct {
	db DBTX
}

func NewSweepLogRepository(db DBTX) *SweepLogRepository {
	return &SweepLogRepository{db: db}
}

// CreatePending inserts a new PENDING sweep log.
func (r *SweepLogRepository) CreatePending(ctx context.Context, addrID uuid.UUID, amount string) (*model.SweepLog, error) {
	var sl model.SweepLog
	err := r.db.QueryRow(ctx, `
		INSERT INTO sweep_logs (deposit_address_id, amount, status)
		VALUES ($1, $2::numeric, 'PENDING')
		RETURNING id, deposit_address_id, tx_hash, amount::text, energy_used, status, error_message, created_at, updated_at, completed_at
	`, addrID, amount).Scan(
		&sl.ID, &sl.DepositAddressID, &sl.TxHash, &sl.Amount, &sl.EnergyUsed,
		&sl.Status, &sl.ErrorMessage, &sl.CreatedAt, &sl.UpdatedAt, &sl.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("sweep log repo: create pending: %w", err)
	}
	return &sl, nil
}

// UpdateToSweeping transitions a sweep log from PENDING to SWEEPING and records energy used.
// txHash is empty at this stage — the hash is set via UpdateTxHash after broadcast.
func (r *SweepLogRepository) UpdateToSweeping(ctx context.Context, id uuid.UUID, txHash string, energyUsed int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sweep_logs
		SET status = 'SWEEPING', tx_hash = $2, energy_used = $3, updated_at = NOW()
		WHERE id = $1 AND status = 'PENDING'
	`, id, txHash, energyUsed)
	if err != nil {
		return fmt.Errorf("sweep log repo: update to sweeping: %w", err)
	}
	return nil
}

// UpdateToDone transitions a sweep log to DONE.
func (r *SweepLogRepository) UpdateToDone(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sweep_logs
		SET status = 'DONE', completed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status IN ('PENDING', 'SWEEPING')
	`, id)
	if err != nil {
		return fmt.Errorf("sweep log repo: update to done: %w", err)
	}
	return nil
}

// UpdateTxHash sets the transaction hash on an already-SWEEPING sweep log.
// Unlike UpdateToSweeping, this does not change the status and does not
// require the record to be in PENDING state.
func (r *SweepLogRepository) UpdateTxHash(ctx context.Context, id uuid.UUID, txHash string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sweep_logs
		SET tx_hash = $2, updated_at = NOW()
		WHERE id = $1 AND status = 'SWEEPING'
	`, id, txHash)
	if err != nil {
		return fmt.Errorf("sweep log repo: update tx_hash: %w", err)
	}
	return nil
}

// UpdateToFailed transitions a sweep log to FAILED with an error message.
func (r *SweepLogRepository) UpdateToFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sweep_logs
		SET status = 'FAILED', error_message = $2, completed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id, errMsg)
	if err != nil {
		return fmt.Errorf("sweep log repo: update to failed: %w", err)
	}
	return nil
}

// MarkStuckSweepingAsFailed transitions SWEEPING or PENDING sweep_logs older than the given threshold.
// - SWEEPING with tx_hash: set to MANUAL_REVIEW (broadcast succeeded, funds may have moved — not safe to re-sweep)
// - SWEEPING without tx_hash: set to FAILED (stuck before broadcast — safe to re-sweep)
// - PENDING: set to FAILED (stuck before broadcast — safe to re-sweep)
func (r *SweepLogRepository) MarkStuckSweepingAsFailed(ctx context.Context, maxAge time.Duration) (int64, error) {
	result, err := r.db.Exec(ctx, `
		UPDATE sweep_logs
		SET status = CASE WHEN tx_hash IS NOT NULL AND tx_hash != '' THEN 'MANUAL_REVIEW' ELSE 'FAILED' END,
		    error_message = 'sweep timed out (stuck in ' || status || ')',
		    completed_at = NOW(), updated_at = NOW()
		WHERE (status = 'SWEEPING' AND updated_at < NOW() - make_interval(secs => $1))
		   OR (status = 'PENDING' AND updated_at < NOW() - make_interval(secs => $2))
	`, int64(maxAge.Seconds()), int64((2 * time.Minute).Seconds()))
	if err != nil {
		return 0, fmt.Errorf("sweep log repo: mark stuck: %w", err)
	}
	return result.RowsAffected(), nil
}

// ListSweepingWithTxHash returns SWEEPING sweep_logs that have a tx_hash.
// These are sweeps where broadcast succeeded but confirmation timed out.
// The reconfirmation checker queries the chain for their final status.
func (r *SweepLogRepository) ListSweepingWithTxHash(ctx context.Context) ([]model.SweepLog, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, deposit_address_id, tx_hash, amount::text, energy_used, status, error_message, created_at, updated_at, completed_at
		FROM sweep_logs
		WHERE status = 'SWEEPING' AND tx_hash IS NOT NULL AND tx_hash != ''
		ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("sweep log repo: list sweeping with tx_hash: %w", err)
	}
	defer rows.Close()
	var out []model.SweepLog
	for rows.Next() {
		var sl model.SweepLog
		if err := rows.Scan(&sl.ID, &sl.DepositAddressID, &sl.TxHash, &sl.Amount,
			&sl.EnergyUsed, &sl.Status, &sl.ErrorMessage, &sl.CreatedAt, &sl.UpdatedAt, &sl.CompletedAt); err != nil {
			return nil, fmt.Errorf("sweep log repo: scan sweeping: %w", err)
		}
		out = append(out, sl)
	}
	return out, rows.Err()
}
