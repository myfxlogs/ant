package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"alphaforge/internal/model"
)

// SweepLogRepository manages sweep_logs records (3-leg model, ADR §2.3).
type SweepLogRepository struct {
	db DBTX
}

func NewSweepLogRepository(db DBTX) *SweepLogRepository {
	return &SweepLogRepository{db: db}
}

// CreateLeg inserts a single PENDING sweep log leg for a batch.
func (r *SweepLogRepository) CreateLeg(
	ctx context.Context,
	batchID, addrID uuid.UUID,
	legType string, legSeq int,
	amount string,
) (*model.SweepLog, error) {
	var sl model.SweepLog
	err := r.db.QueryRow(ctx, `
		INSERT INTO sweep_logs (batch_id, deposit_address_id, leg_type, leg_seq, amount, status)
		VALUES ($1, $2, $3, $4, $5::numeric, 'PENDING')
		RETURNING id, batch_id, deposit_address_id, leg_type, leg_seq,
		          tx_hash, amount::text, energy_used, status, error_message,
		          created_at, updated_at, completed_at
	`, batchID, addrID, legType, legSeq, amount).Scan(
		&sl.ID, &sl.BatchID, &sl.DepositAddressID, &sl.LegType, &sl.LegSeq,
		&sl.TxHash, &sl.Amount, &sl.EnergyUsed, &sl.Status, &sl.ErrorMessage,
		&sl.CreatedAt, &sl.UpdatedAt, &sl.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("sweep log repo: create leg: %w", err)
	}
	return &sl, nil
}

// UpdateToSweeping transitions a sweep log to SWEEPING and records energy used.
// Accepts both PENDING and FAILED status — FAILED legs are re-broadcast after
// chain verification confirms the previous tx is not on chain (D9: safe re-broadcast).
func (r *SweepLogRepository) UpdateToSweeping(ctx context.Context, id uuid.UUID, txHash string, energyUsed int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sweep_logs
		SET status = 'SWEEPING', tx_hash = $2, energy_used = $3, updated_at = NOW()
		WHERE id = $1 AND status IN ('PENDING', 'FAILED')
	`, id, txHash, energyUsed)
	if err != nil {
		return fmt.Errorf("sweep log repo: update to sweeping: %w", err)
	}
	return nil
}

// UpdateToDone transitions a sweep log to DONE.
// Accepts PENDING, SWEEPING, and FAILED — a FAILED leg may be found confirmed
// on chain during recovery (D9 chain check), transitioning directly to DONE.
func (r *SweepLogRepository) UpdateToDone(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sweep_logs
		SET status = 'DONE', completed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status IN ('PENDING', 'SWEEPING', 'FAILED')
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

// UpdateToManualReview transitions a sweep log to MANUAL_REVIEW.
// Used when a leg is chain-confirmed FAILED — the operator must investigate
// (e.g. manually undelegate energy, resolve transfer failure) before re-sweeping.
func (r *SweepLogRepository) UpdateToManualReview(ctx context.Context, id uuid.UUID, reason string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sweep_logs
		SET status = 'MANUAL_REVIEW', error_message = $2, updated_at = NOW()
		WHERE id = $1
	`, id, reason)
	if err != nil {
		return fmt.Errorf("sweep log repo: update to manual review: %w", err)
	}
	return nil
}

// MarkStuckSweepingAsFailed transitions SWEEPING or PENDING sweep_logs older than the given threshold.
// - SWEEPING with tx_hash: set to MANUAL_REVIEW (broadcast succeeded, funds may have moved — not safe to re-sweep)
// - SWEEPING without tx_hash: set to FAILED (stuck before broadcast — safe to re-sweep)
// - PENDING: set to FAILED (stuck before broadcast — safe to re-sweep)
func (r *SweepLogRepository) MarkStuckSweepingAsFailed(ctx context.Context, maxAge time.Duration) (int64, error) {
	// PENDING legs wait for cold signing (human USB round-trip, potentially hours).
	// Use 24h — matching raw_tx expiry — so legs aren't prematurely FAILED while
	// the operator still holds the unsigned bundle for signing.
	pendingTimeout := 24 * time.Hour
	result, err := r.db.Exec(ctx, `
		UPDATE sweep_logs
		SET status = CASE WHEN tx_hash IS NOT NULL AND tx_hash != '' THEN 'MANUAL_REVIEW' ELSE 'FAILED' END,
		    error_message = 'sweep timed out (stuck in ' || status || ')',
		    completed_at = NOW(), updated_at = NOW()
		WHERE (status = 'SWEEPING' AND updated_at < NOW() - make_interval(secs => $1))
		   OR (status = 'PENDING' AND updated_at < NOW() - make_interval(secs => $2))
	`, int64(maxAge.Seconds()), int64(pendingTimeout.Seconds()))
	if err != nil {
		return 0, fmt.Errorf("sweep log repo: mark stuck: %w", err)
	}
	return result.RowsAffected(), nil
}

// ListSweepingWithTxHash returns SWEEPING and FAILED sweep_logs that have a tx_hash.
// SWEEPING: broadcast succeeded but confirmation timed out — chain check for final status.
// FAILED+tx_hash: broadcast may have partially succeeded (network error after tx accepted) —
// chain check as safety net, especially for expired bundles where resumeBroadcasting won't run.
func (r *SweepLogRepository) ListSweepingWithTxHash(ctx context.Context) ([]model.SweepLog, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, batch_id, deposit_address_id, leg_type, leg_seq,
		       tx_hash, amount::text, energy_used, status, error_message,
		       created_at, updated_at, completed_at
		FROM sweep_logs
		WHERE status IN ('SWEEPING', 'FAILED') AND tx_hash IS NOT NULL AND tx_hash != ''
		ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("sweep log repo: list sweeping with tx_hash: %w", err)
	}
	defer rows.Close()
	var out []model.SweepLog
	for rows.Next() {
		var sl model.SweepLog
		if err := rows.Scan(&sl.ID, &sl.BatchID, &sl.DepositAddressID, &sl.LegType, &sl.LegSeq,
			&sl.TxHash, &sl.Amount, &sl.EnergyUsed, &sl.Status, &sl.ErrorMessage,
			&sl.CreatedAt, &sl.UpdatedAt, &sl.CompletedAt); err != nil {
			return nil, fmt.Errorf("sweep log repo: scan sweeping: %w", err)
		}
		out = append(out, sl)
	}
	return out, rows.Err()
}

// ListBatchLegs returns all 3 legs for a given batch_id, ordered by leg_seq.
func (r *SweepLogRepository) ListBatchLegs(ctx context.Context, batchID uuid.UUID) ([]model.SweepLog, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, batch_id, deposit_address_id, leg_type, leg_seq,
		       tx_hash, amount::text, energy_used, status, error_message,
		       created_at, updated_at, completed_at
		FROM sweep_logs
		WHERE batch_id = $1
		ORDER BY leg_seq
	`, batchID)
	if err != nil {
		return nil, fmt.Errorf("sweep log repo: list batch legs: %w", err)
	}
	defer rows.Close()
	var out []model.SweepLog
	for rows.Next() {
		var sl model.SweepLog
		if err := rows.Scan(&sl.ID, &sl.BatchID, &sl.DepositAddressID, &sl.LegType, &sl.LegSeq,
			&sl.TxHash, &sl.Amount, &sl.EnergyUsed, &sl.Status, &sl.ErrorMessage,
			&sl.CreatedAt, &sl.UpdatedAt, &sl.CompletedAt); err != nil {
			return nil, fmt.Errorf("sweep log repo: scan batch legs: %w", err)
		}
		out = append(out, sl)
	}
	return out, rows.Err()
}

// GetLegByTxHash returns a sweep log leg by its tx_hash.
func (r *SweepLogRepository) GetLegByTxHash(ctx context.Context, txHash string) (*model.SweepLog, error) {
	var sl model.SweepLog
	err := r.db.QueryRow(ctx, `
		SELECT id, batch_id, deposit_address_id, leg_type, leg_seq,
		       tx_hash, amount::text, energy_used, status, error_message,
		       created_at, updated_at, completed_at
		FROM sweep_logs
		WHERE tx_hash = $1
	`, txHash).Scan(
		&sl.ID, &sl.BatchID, &sl.DepositAddressID, &sl.LegType, &sl.LegSeq,
		&sl.TxHash, &sl.Amount, &sl.EnergyUsed, &sl.Status, &sl.ErrorMessage,
		&sl.CreatedAt, &sl.UpdatedAt, &sl.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("sweep log repo: get by tx_hash: %w", err)
	}
	return &sl, nil
}

// GetLatestDoneTransferLeg returns the most recent DONE transfer leg for a deposit address.
// Used by double-spend prevention to distinguish expected (already swept) vs unexpected
// outgoing transfers. If a DONE transfer leg exists, an outgoing transfer to cold wallet
// is expected and should not block re-sweeping (ADR §2.7: "归集后地址保留，支持多次充值").
func (r *SweepLogRepository) GetLatestDoneTransferLeg(ctx context.Context, addrID uuid.UUID) (*model.SweepLog, error) {
	var sl model.SweepLog
	err := r.db.QueryRow(ctx, `
		SELECT id, batch_id, deposit_address_id, leg_type, leg_seq,
		       tx_hash, amount::text, energy_used, status, error_message,
		       created_at, updated_at, completed_at
		FROM sweep_logs
		WHERE deposit_address_id = $1 AND leg_type = 'transfer' AND status = 'DONE'
		ORDER BY completed_at DESC
		LIMIT 1
	`, addrID).Scan(
		&sl.ID, &sl.BatchID, &sl.DepositAddressID, &sl.LegType, &sl.LegSeq,
		&sl.TxHash, &sl.Amount, &sl.EnergyUsed, &sl.Status, &sl.ErrorMessage,
		&sl.CreatedAt, &sl.UpdatedAt, &sl.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("sweep log repo: get latest done transfer: %w", err)
	}
	return &sl, nil
}

// FailLegsByBatch marks all PENDING legs for a given batch_id as FAILED.
// Used when a PENDING_SIGN bundle expires (D1: orphaned bundle recovery).
func (r *SweepLogRepository) FailLegsByBatch(ctx context.Context, batchID uuid.UUID, reason string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sweep_logs
		SET status = 'FAILED', error_message = $2, completed_at = NOW(), updated_at = NOW()
		WHERE batch_id = $1 AND status = 'PENDING'
	`, batchID, reason)
	if err != nil {
		return fmt.Errorf("sweep log repo: fail legs by batch: %w", err)
	}
	return nil
}
