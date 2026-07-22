package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

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

// SweepDashboardRow is a single row in the sweep dashboard (C4).
type SweepDashboardRow struct {
	AddrID          uuid.UUID
	Address         string
	DerivationIndex int32
	UnsweptAmount   string
	SweepStatus     string // highest-priority sweep status for this address
}

// GetUnsweptBalance returns the unswept confirmed deposit balance for a single address.
// Returns empty string if the address has no unswept balance.
func (r *DepositRepository) GetUnsweptBalance(ctx context.Context, addrID uuid.UUID) (string, error) {
	var amount string
	err := r.db.QueryRow(ctx, `
		SELECT (total_deposits - total_swept)::text FROM (
			SELECT
				COALESCE(SUM(d.amount), 0) AS total_deposits,
				COALESCE(SUM(sl.amount) FILTER (WHERE sl.status = 'DONE' AND sl.leg_type = 'transfer'), 0) AS total_swept
			FROM deposits d
			LEFT JOIN sweep_logs sl ON sl.deposit_address_id = d.deposit_address_id
			WHERE d.status = 'CONFIRMED' AND d.deposit_address_id = $1
		) addr_balance
		WHERE (total_deposits - total_swept) > 0
	`, addrID).Scan(&amount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("deposit repo v2: get unswept balance: %w", err)
	}
	return amount, nil
}

// ListSweepDashboard returns all deposit addresses with their unswept balance
// and highest-priority sweep status, ordered by unswept balance descending (C4).
// Includes addresses with zero unswept balance if they have active sweep legs,
// so the operator sees the full picture.
func (r *DepositRepository) ListSweepDashboard(ctx context.Context, page, pageSize int) ([]SweepDashboardRow, int64, string, error) {
	// Get total unswept across all addresses.
	var totalUnswept string
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(unswept), 0)::text FROM (
			SELECT d.deposit_address_id,
			       SUM(d.amount) - COALESCE(SUM(sl.amount) FILTER (WHERE sl.status = 'DONE' AND sl.leg_type = 'transfer'), 0) AS unswept
			FROM deposits d
			LEFT JOIN sweep_logs sl ON sl.deposit_address_id = d.deposit_address_id
			WHERE d.status = 'CONFIRMED'
			GROUP BY d.deposit_address_id
		) bal WHERE bal.unswept > 0
	`).Scan(&totalUnswept)
	if err != nil {
		return nil, 0, "", fmt.Errorf("deposit repo v2: dashboard total: %w", err)
	}

	// Count total rows (addresses with unswept > 0 OR active sweep legs).
	var total int64
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT da.id
			FROM user_deposit_addresses da
			WHERE EXISTS (
				SELECT 1 FROM deposits d
				WHERE d.deposit_address_id = da.id AND d.status = 'CONFIRMED'
				GROUP BY d.deposit_address_id
				HAVING SUM(d.amount) - COALESCE(SUM(0) FILTER (WHERE false), 0) > 0
			)
			OR EXISTS (
				SELECT 1 FROM sweep_logs sl
				WHERE sl.deposit_address_id = da.id AND sl.status IN ('PENDING', 'SWEEPING', 'MANUAL_REVIEW')
			)
		) cnt
	`).Scan(&total)
	if err != nil {
		return nil, 0, "", fmt.Errorf("deposit repo v2: dashboard count: %w", err)
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(ctx, `
		WITH addr_balance AS (
			SELECT d.deposit_address_id,
			       SUM(d.amount) AS total_deposits,
			       COALESCE(SUM(sl.amount) FILTER (WHERE sl.status = 'DONE' AND sl.leg_type = 'transfer'), 0) AS total_swept
			FROM deposits d
			LEFT JOIN sweep_logs sl ON sl.deposit_address_id = d.deposit_address_id
			WHERE d.status = 'CONFIRMED'
			GROUP BY d.deposit_address_id
		),
		addr_status AS (
			SELECT deposit_address_id,
			       CASE
			           WHEN bool_or(status = 'MANUAL_REVIEW') THEN 'MANUAL_REVIEW'
			           WHEN bool_or(status = 'SWEEPING') THEN 'SWEEPING'
			           WHEN bool_or(status = 'PENDING') THEN 'PENDING'
			           WHEN bool_or(status = 'DONE') THEN 'DONE'
			           ELSE 'none'
			       END AS sweep_status
			FROM sweep_logs
			GROUP BY deposit_address_id
		)
		SELECT da.id, da.address, da.derivation_index,
		       COALESCE((ab.total_deposits - ab.total_swept)::text, '0') AS unswept_amount,
		       COALESCE(ast.sweep_status, 'none') AS sweep_status
		FROM user_deposit_addresses da
		LEFT JOIN addr_balance ab ON ab.deposit_address_id = da.id
		LEFT JOIN addr_status ast ON ast.deposit_address_id = da.id
		WHERE COALESCE(ab.total_deposits - ab.total_swept, 0) > 0
		   OR ast.sweep_status IN ('PENDING', 'SWEEPING', 'MANUAL_REVIEW')
		ORDER BY COALESCE(ab.total_deposits - ab.total_swept, 0) DESC
		LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, 0, "", fmt.Errorf("deposit repo v2: dashboard query: %w", err)
	}
	defer rows.Close()

	var out []SweepDashboardRow
	for rows.Next() {
		var row SweepDashboardRow
		if err := rows.Scan(&row.AddrID, &row.Address, &row.DerivationIndex, &row.UnsweptAmount, &row.SweepStatus); err != nil {
			return nil, 0, "", fmt.Errorf("deposit repo v2: dashboard scan: %w", err)
		}
		out = append(out, row)
	}
	return out, total, totalUnswept, rows.Err()
}
