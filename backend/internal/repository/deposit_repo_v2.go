package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"alphaforge/internal/model"
)

// DepositRepository manages deposits records (new HD wallet system).
type DepositRepository struct {
	db DBTX
}

func NewDepositRepository(db DBTX) *DepositRepository {
	return &DepositRepository{db: db}
}

// Create inserts a new deposit record.
func (r *DepositRepository) Create(ctx context.Context, d *model.Deposit) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO deposits (id, user_id, deposit_address_id, tx_hash, amount, block_number, confirmations, status, confirmed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tx_hash) DO NOTHING
	`, d.ID, d.UserID, d.DepositAddressID, d.TxHash, d.Amount, d.BlockNumber, d.Confirmations, d.Status, d.ConfirmedAt)
	if err != nil {
		return fmt.Errorf("deposit repo v2: create: %w", err)
	}
	return nil
}

// ExistsByTxHash checks if a deposit with the given tx_hash already exists.
func (r *DepositRepository) ExistsByTxHash(ctx context.Context, txHash string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM deposits WHERE tx_hash = $1)`, txHash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("deposit repo v2: exists: %w", err)
	}
	return exists, nil
}

// ListByUser returns paginated deposit history for a user.
func (r *DepositRepository) ListByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]model.Deposit, int64, error) {
	var total int64
	err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM deposits WHERE user_id = $1 AND status = 'CONFIRMED'`, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("deposit repo v2: count: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}
	offset := (page - 1) * pageSize
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, deposit_address_id, tx_hash, amount::text, block_number,
		       confirmations, status, confirmed_at, created_at
		FROM deposits
		WHERE user_id = $1 AND status = 'CONFIRMED'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("deposit repo v2: list: %w", err)
	}
	defer rows.Close()
	var out []model.Deposit
	for rows.Next() {
		var d model.Deposit
		if err := rows.Scan(&d.ID, &d.UserID, &d.DepositAddressID, &d.TxHash, &d.Amount,
			&d.BlockNumber, &d.Confirmations, &d.Status, &d.ConfirmedAt, &d.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("deposit repo v2: scan: %w", err)
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

// ListManualReview returns deposits with MANUAL_REVIEW status for admin review.
func (r *DepositRepository) ListManualReview(ctx context.Context, page, pageSize int) ([]model.Deposit, int64, error) {
	var total int64
	err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM deposits WHERE status = 'MANUAL_REVIEW'`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("deposit repo v2: count manual review: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}
	offset := (page - 1) * pageSize
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, deposit_address_id, tx_hash, amount::text, block_number,
		       confirmations, status, confirmed_at, created_at
		FROM deposits
		WHERE status = 'MANUAL_REVIEW'
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("deposit repo v2: list manual review: %w", err)
	}
	defer rows.Close()
	var out []model.Deposit
	for rows.Next() {
		var d model.Deposit
		if err := rows.Scan(&d.ID, &d.UserID, &d.DepositAddressID, &d.TxHash, &d.Amount,
			&d.BlockNumber, &d.Confirmations, &d.Status, &d.ConfirmedAt, &d.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("deposit repo v2: scan: %w", err)
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

var ErrDepositExists = errors.New("deposit already exists")

// UnsweptAddress represents an address with unswept confirmed deposit balance.
type UnsweptAddress struct {
	AddrID uuid.UUID
	Amount string
}

// ListUnsweptAddresses finds addresses with confirmed deposits that haven't been fully swept.
// Correctly handles multiple deposits to the same address by comparing total deposits
// minus total DONE sweep amounts, rather than just checking for any DONE sweep_log.
// Excludes addresses with active PENDING/SWEEPING sweep tasks.
func (r *DepositRepository) ListUnsweptAddresses(ctx context.Context, threshold string, limit int) ([]UnsweptAddress, error) {
	rows, err := r.db.Query(ctx, `
		WITH addr_balance AS (
			SELECT d.deposit_address_id,
			       SUM(d.amount) AS total_deposits,
			       COALESCE(SUM(sl.amount) FILTER (WHERE sl.status = 'DONE'), 0) AS total_swept
			FROM deposits d
			LEFT JOIN sweep_logs sl ON sl.deposit_address_id = d.deposit_address_id
			WHERE d.status = 'CONFIRMED'
			GROUP BY d.deposit_address_id
		)
		SELECT deposit_address_id, (total_deposits - total_swept)::text AS unswept_amount
		FROM addr_balance
		WHERE (total_deposits - total_swept) >= $1::numeric
		AND NOT EXISTS (
			SELECT 1 FROM sweep_logs sl2
			WHERE sl2.deposit_address_id = addr_balance.deposit_address_id
			AND sl2.status IN ('PENDING', 'SWEEPING')
		)
		AND NOT EXISTS (
			SELECT 1 FROM sweep_logs sl3
			WHERE sl3.deposit_address_id = addr_balance.deposit_address_id
			AND sl3.status = 'FAILED'
			AND sl3.updated_at > NOW() - INTERVAL '1 hour'
		)
		LIMIT $2
	`, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("deposit repo v2: list unswept: %w", err)
	}
	defer rows.Close()

	var out []UnsweptAddress
	for rows.Next() {
		var u UnsweptAddress
		if err := rows.Scan(&u.AddrID, &u.Amount); err != nil {
			return nil, fmt.Errorf("deposit repo v2: scan unswept: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
