package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"alphaforge/internal/model"
)

// WithdrawalRepository manages withdrawal requests and whitelist entries.
type WithdrawalRepository struct {
	db DBTX
}

func NewWithdrawalRepository(db DBTX) *WithdrawalRepository {
	return &WithdrawalRepository{db: db}
}

// CreateWithdrawal inserts a new withdrawal request with PENDING status.
func (r *WithdrawalRepository) CreateWithdrawal(ctx context.Context, w *model.WithdrawalRequest) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO withdrawal_requests (id, user_id, amount, dest_address, nonce, credential_id, assertion, status, idem_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING', $8)
	`, w.ID, w.UserID, w.Amount, w.DestAddress, w.Nonce, w.CredentialID, w.Assertion, w.IdemKey)
	if err != nil {
		return fmt.Errorf("withdrawal repo: create: %w", err)
	}
	return nil
}

// GetWithdrawal retrieves a withdrawal request by ID.
func (r *WithdrawalRepository) GetWithdrawal(ctx context.Context, id uuid.UUID) (*model.WithdrawalRequest, error) {
	var w model.WithdrawalRequest
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, amount::text, dest_address, nonce, credential_id, assertion, status,
		       bundle_id, tx_hash, idem_key, created_at, updated_at, completed_at
		FROM withdrawal_requests WHERE id = $1
	`, id).Scan(
		&w.ID, &w.UserID, &w.Amount, &w.DestAddress, &w.Nonce, &w.CredentialID, &w.Assertion,
		&w.Status, &w.BundleID, &w.TxHash, &w.IdemKey, &w.CreatedAt, &w.UpdatedAt, &w.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("withdrawal repo: get: %w", err)
	}
	return &w, nil
}

// ListWithdrawalsByUser returns paginated withdrawal requests for a user.
func (r *WithdrawalRepository) ListWithdrawalsByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]model.WithdrawalRequest, int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM withdrawal_requests WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("withdrawal repo: count: %w", err)
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, amount::text, dest_address, nonce, credential_id, assertion, status,
		       bundle_id, tx_hash, idem_key, created_at, updated_at, completed_at
		FROM withdrawal_requests WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, userID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("withdrawal repo: list: %w", err)
	}
	defer rows.Close()

	var out []model.WithdrawalRequest
	for rows.Next() {
		var w model.WithdrawalRequest
		if err := rows.Scan(
			&w.ID, &w.UserID, &w.Amount, &w.DestAddress, &w.Nonce, &w.CredentialID, &w.Assertion,
			&w.Status, &w.BundleID, &w.TxHash, &w.IdemKey, &w.CreatedAt, &w.UpdatedAt, &w.CompletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("withdrawal repo: scan: %w", err)
		}
		out = append(out, w)
	}
	return out, total, rows.Err()
}

// UpdateWithdrawalStatus transitions a withdrawal to a new status.
func (r *WithdrawalRepository) UpdateWithdrawalStatus(ctx context.Context, id uuid.UUID, status string, txHash *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE withdrawal_requests
		SET status = $2, tx_hash = COALESCE($3, tx_hash), updated_at = NOW(),
		    completed_at = CASE WHEN $2 IN ('DONE', 'FAILED', 'CANCELLED') THEN NOW() ELSE completed_at END
		WHERE id = $1
	`, id, status, txHash)
	if err != nil {
		return fmt.Errorf("withdrawal repo: update status: %w", err)
	}
	return nil
}

// UpdateWithdrawalBundle links a withdrawal to a sweep bundle.
func (r *WithdrawalRepository) UpdateWithdrawalBundle(ctx context.Context, id uuid.UUID, bundleID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE withdrawal_requests SET bundle_id = $2, status = 'SIGNED', updated_at = NOW() WHERE id = $1
	`, id, bundleID)
	if err != nil {
		return fmt.Errorf("withdrawal repo: update bundle: %w", err)
	}
	return nil
}

// CancelWithdrawalBundle marks the sweep bundle for a cancelled withdrawal as MANUAL_REVIEW.
// This prevents coldsign from signing a bundle whose withdrawal has been cancelled.
func (r *WithdrawalRepository) CancelWithdrawalBundle(ctx context.Context, bundleID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sweep_bundles SET status = 'MANUAL_REVIEW', updated_at = NOW()
		WHERE batch_id = $1 AND status = 'PENDING_SIGN'
	`, bundleID)
	if err != nil {
		return fmt.Errorf("withdrawal repo: cancel bundle: %w", err)
	}
	return nil
}

// StoreWithdrawalAssertion stores the WebAuthn assertion and transitions to SIGNED_WAITING_BUNDLE.
func (r *WithdrawalRepository) StoreWithdrawalAssertion(ctx context.Context, id uuid.UUID, assertion []byte, credentialID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE withdrawal_requests
		SET assertion = $2, credential_id = $3, status = 'SIGNED_WAITING_BUNDLE', updated_at = NOW()
		WHERE id = $1
	`, id, assertion, credentialID)
	if err != nil {
		return fmt.Errorf("withdrawal repo: store assertion: %w", err)
	}
	return nil
}

// CreateWhitelistEntry inserts a new whitelist address with PENDING_CONFIRMATION status.
func (r *WithdrawalRepository) CreateWhitelistEntry(ctx context.Context, e *model.WithdrawalWhitelistEntry) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO withdrawal_whitelist (id, user_id, address, label, status, cooldown_until)
		VALUES ($1, $2, $3, $4, 'PENDING_CONFIRMATION', NOW() + INTERVAL '24 hours')
	`, e.ID, e.UserID, e.Address, e.Label)
	if err != nil {
		return fmt.Errorf("withdrawal repo: create whitelist: %w", err)
	}
	return nil
}

// ListWhitelistByUser returns all whitelist entries for a user.
func (r *WithdrawalRepository) ListWhitelistByUser(ctx context.Context, userID uuid.UUID) ([]model.WithdrawalWhitelistEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, address, label, status, confirmed_at, cooldown_until, created_at, updated_at
		FROM withdrawal_whitelist WHERE user_id = $1 ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("withdrawal repo: list whitelist: %w", err)
	}
	defer rows.Close()

	var out []model.WithdrawalWhitelistEntry
	for rows.Next() {
		var e model.WithdrawalWhitelistEntry
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.Address, &e.Label, &e.Status, &e.ConfirmedAt,
			&e.CooldownUntil, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("withdrawal repo: scan whitelist: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetActiveWhitelistAddresses returns ACTIVE whitelist addresses for a user.
func (r *WithdrawalRepository) GetActiveWhitelistAddresses(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT address FROM withdrawal_whitelist WHERE user_id = $1 AND status = 'ACTIVE'
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("withdrawal repo: active whitelist: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var addr string
		if err := rows.Scan(&addr); err != nil {
			return nil, fmt.Errorf("withdrawal repo: scan active whitelist: %w", err)
		}
		out = append(out, addr)
	}
	return out, rows.Err()
}

// ListAllActiveWhitelist returns all ACTIVE whitelist entries across all users.
// Used by ExportWhitelist RPC for coldsign USB sync (R12).
func (r *WithdrawalRepository) ListAllActiveWhitelist(ctx context.Context) ([]model.WithdrawalWhitelistEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, address, label, status, confirmed_at, cooldown_until, created_at, updated_at
		FROM withdrawal_whitelist WHERE status = 'ACTIVE' ORDER BY user_id, created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("withdrawal repo: list all active whitelist: %w", err)
	}
	defer rows.Close()

	var out []model.WithdrawalWhitelistEntry
	for rows.Next() {
		var e model.WithdrawalWhitelistEntry
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.Address, &e.Label, &e.Status, &e.ConfirmedAt,
			&e.CooldownUntil, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("withdrawal repo: scan all active whitelist: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ActivateWhitelistEntry transitions a whitelist entry to ACTIVE after cooldown.
func (r *WithdrawalRepository) ActivateWhitelistEntry(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE withdrawal_whitelist SET status = 'ACTIVE', confirmed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'PENDING_CONFIRMATION' AND cooldown_until <= NOW()
	`, id)
	if err != nil {
		return fmt.Errorf("withdrawal repo: activate whitelist: %w", err)
	}
	return nil
}

// RemoveWhitelistEntry marks a whitelist entry as REMOVED.
func (r *WithdrawalRepository) RemoveWhitelistEntry(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE withdrawal_whitelist SET status = 'REMOVED', updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return fmt.Errorf("withdrawal repo: remove whitelist: %w", err)
	}
	return nil
}

// CreateCredentialChangeLog inserts a credential/whitelist change record (R12).
func (r *WithdrawalRepository) CreateCredentialChangeLog(ctx context.Context, log *model.CredentialChangeLog) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO credential_change_log (user_id, change_type, target_id, status, idem_key)
		VALUES ($1, $2, $3, 'PENDING', $4)
	`, log.UserID, log.ChangeType, log.TargetID, log.IdemKey)
	if err != nil {
		return fmt.Errorf("withdrawal repo: create change log: %w", err)
	}
	return nil
}

// ConfirmCredentialChangeLog marks a change log entry as confirmed.
func (r *WithdrawalRepository) ConfirmCredentialChangeLog(ctx context.Context, idemKey string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE credential_change_log SET status = 'CONFIRMED', confirmed_at = NOW() WHERE idem_key = $1
	`, idemKey)
	if err != nil {
		return fmt.Errorf("withdrawal repo: confirm change log: %w", err)
	}
	return nil
}

// ListPendingActivations returns whitelist entries that have passed their cooldown
// and are ready to be activated.
func (r *WithdrawalRepository) ListPendingActivations(ctx context.Context, now time.Time) ([]model.WithdrawalWhitelistEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, address, label, status, confirmed_at, cooldown_until, created_at, updated_at
		FROM withdrawal_whitelist
		WHERE status = 'PENDING_CONFIRMATION' AND cooldown_until <= $1
	`, now)
	if err != nil {
		return nil, fmt.Errorf("withdrawal repo: list pending activations: %w", err)
	}
	defer rows.Close()

	var out []model.WithdrawalWhitelistEntry
	for rows.Next() {
		var e model.WithdrawalWhitelistEntry
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.Address, &e.Label, &e.Status, &e.ConfirmedAt,
			&e.CooldownUntil, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("withdrawal repo: scan pending activation: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListWithdrawalsByStatus returns withdrawals matching the given status,
// with assertion and credential_id populated (for WithdrawalBuilder).
func (r *WithdrawalRepository) ListWithdrawalsByStatus(ctx context.Context, status string) ([]model.WithdrawalRequest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, amount::text, dest_address, nonce, credential_id, assertion, status,
		       bundle_id, tx_hash, idem_key, created_at, updated_at, completed_at
		FROM withdrawal_requests WHERE status = $1 ORDER BY created_at
	`, status)
	if err != nil {
		return nil, fmt.Errorf("withdrawal repo: list by status: %w", err)
	}
	defer rows.Close()

	var out []model.WithdrawalRequest
	for rows.Next() {
		var w model.WithdrawalRequest
		if err := rows.Scan(
			&w.ID, &w.UserID, &w.Amount, &w.DestAddress, &w.Nonce, &w.CredentialID, &w.Assertion,
			&w.Status, &w.BundleID, &w.TxHash, &w.IdemKey, &w.CreatedAt, &w.UpdatedAt, &w.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("withdrawal repo: scan by status: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
