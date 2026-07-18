package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"alphaforge/internal/model"
)

// DepositAddressRepository manages the HD wallet address pool and deposit records.
type DepositAddressRepository struct {
	db DBTX
}

func NewDepositAddressRepository(db DBTX) *DepositAddressRepository {
	return &DepositAddressRepository{db: db}
}

// ClaimAddress atomically assigns an AVAILABLE address to a user.
// Uses FOR UPDATE SKIP LOCKED for concurrency safety.
// If the user already has an ASSIGNED address, returns it instead.
func (r *DepositAddressRepository) ClaimAddress(ctx context.Context, userID uuid.UUID) (*model.DepositAddress, error) {
	// First check if user already has an assigned address.
	existing, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// Atomically claim an available address.
	var a model.DepositAddress
	err = r.db.QueryRow(ctx, `
		UPDATE user_deposit_addresses
		SET user_id = $1, status = 'ASSIGNED', assigned_at = NOW(), updated_at = NOW()
		WHERE id = (
			SELECT id FROM user_deposit_addresses
			WHERE status = 'AVAILABLE'
			ORDER BY derivation_index
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, user_id, address, derivation_index, encrypted_privkey,
		          network, status, has_received_usdt, created_at, updated_at, assigned_at
	`, userID).Scan(
		&a.ID, &a.UserID, &a.Address, &a.DerivationIndex, &a.EncryptedPrivkey,
		&a.Network, &a.Status, &a.HasReceivedUSDT, &a.CreatedAt, &a.UpdatedAt, &a.AssignedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAddressPoolEmpty
		}
		return nil, fmt.Errorf("deposit address repo: claim: %w", err)
	}
	return &a, nil
}

// GetByUserID returns the assigned deposit address for a user, or nil if none.
func (r *DepositAddressRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.DepositAddress, error) {
	var a model.DepositAddress
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, address, derivation_index, encrypted_privkey,
		       network, status, has_received_usdt, created_at, updated_at, assigned_at
		FROM user_deposit_addresses
		WHERE user_id = $1 AND status = 'ASSIGNED'
	`, userID).Scan(
		&a.ID, &a.UserID, &a.Address, &a.DerivationIndex, &a.EncryptedPrivkey,
		&a.Network, &a.Status, &a.HasReceivedUSDT, &a.CreatedAt, &a.UpdatedAt, &a.AssignedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("deposit address repo: get by user: %w", err)
	}
	return &a, nil
}

// GetByAddress returns the deposit address record by TRON address string.
func (r *DepositAddressRepository) GetByAddress(ctx context.Context, addr string) (*model.DepositAddress, error) {
	var a model.DepositAddress
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, address, derivation_index, encrypted_privkey,
		       network, status, has_received_usdt, created_at, updated_at, assigned_at
		FROM user_deposit_addresses
		WHERE address = $1
	`, addr).Scan(
		&a.ID, &a.UserID, &a.Address, &a.DerivationIndex, &a.EncryptedPrivkey,
		&a.Network, &a.Status, &a.HasReceivedUSDT, &a.CreatedAt, &a.UpdatedAt, &a.AssignedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("deposit address repo: get by address: %w", err)
	}
	return &a, nil
}

// GetByID returns a deposit address record by its UUID.
func (r *DepositAddressRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.DepositAddress, error) {
	var a model.DepositAddress
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, address, derivation_index, encrypted_privkey,
		       network, status, has_received_usdt, created_at, updated_at, assigned_at
		FROM user_deposit_addresses
		WHERE id = $1
	`, id).Scan(
		&a.ID, &a.UserID, &a.Address, &a.DerivationIndex, &a.EncryptedPrivkey,
		&a.Network, &a.Status, &a.HasReceivedUSDT, &a.CreatedAt, &a.UpdatedAt, &a.AssignedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("deposit address repo: get by id: %w", err)
	}
	return &a, nil
}

// CountAvailable returns the number of AVAILABLE addresses in the pool.
func (r *DepositAddressRepository) CountAvailable(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM user_deposit_addresses WHERE status = 'AVAILABLE'`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("deposit address repo: count available: %w", err)
	}
	return count, nil
}

// ImportBatch inserts a batch of offline-generated addresses into the pool.
// All addresses start with status='AVAILABLE', user_id=NULL.
func (r *DepositAddressRepository) ImportBatch(ctx context.Context, addrs []model.DepositAddress) error {
	tx, ok := r.db.(pgx.Tx)
	if !ok {
		// Not in a transaction context, use individual inserts.
		for _, a := range addrs {
			_, err := r.db.Exec(ctx, `
				INSERT INTO user_deposit_addresses (address, derivation_index, encrypted_privkey, network, status)
				VALUES ($1, $2, $3, $4, 'AVAILABLE')
				ON CONFLICT (address) DO NOTHING
			`, a.Address, a.DerivationIndex, a.EncryptedPrivkey, a.Network)
			if err != nil {
				return fmt.Errorf("deposit address repo: import batch: %w", err)
			}
		}
		return nil
	}

	for _, a := range addrs {
		_, err := tx.Exec(ctx, `
			INSERT INTO user_deposit_addresses (address, derivation_index, encrypted_privkey, network, status)
			VALUES ($1, $2, $3, $4, 'AVAILABLE')
			ON CONFLICT (address) DO NOTHING
		`, a.Address, a.DerivationIndex, a.EncryptedPrivkey, a.Network)
		if err != nil {
			return fmt.Errorf("deposit address repo: import batch: %w", err)
		}
	}
	return nil
}

// AddressInfo holds the user ID and address ID for an assigned deposit address.
type AddressInfo struct {
	UserID uuid.UUID
	AddrID uuid.UUID
}

// ListAssignedAddresses returns all ASSIGNED addresses with their user and address IDs.
// Used by chain monitor to populate the in-memory address map.
func (r *DepositAddressRepository) ListAssignedAddresses(ctx context.Context) (map[string]AddressInfo, error) {
	rows, err := r.db.Query(ctx, `
		SELECT address, user_id, id FROM user_deposit_addresses
		WHERE status = 'ASSIGNED' AND user_id IS NOT NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("deposit address repo: list assigned: %w", err)
	}
	defer rows.Close()

	m := make(map[string]AddressInfo)
	for rows.Next() {
		var addr string
		var info AddressInfo
		if err := rows.Scan(&addr, &info.UserID, &info.AddrID); err != nil {
			return nil, fmt.Errorf("deposit address repo: scan assigned: %w", err)
		}
		m[addr] = info
	}
	return m, rows.Err()
}

// MarkReceivedUSDT updates has_received_usdt to true for an address.
func (r *DepositAddressRepository) MarkReceivedUSDT(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE user_deposit_addresses SET has_received_usdt = true, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deposit address repo: mark received: %w", err)
	}
	return nil
}

// ListAllAddresses returns paginated deposit addresses with optional status filter.
func (r *DepositAddressRepository) ListAllAddresses(ctx context.Context, status string, page, pageSize int) ([]model.DepositAddress, int64, error) {
	offset := (page - 1) * pageSize
	if status != "" {
		var total int64
		err := r.db.QueryRow(ctx,
			`SELECT count(*) FROM user_deposit_addresses WHERE status = $1`, status).Scan(&total)
		if err != nil {
			return nil, 0, fmt.Errorf("deposit address repo: count all: %w", err)
		}
		rows, err := r.db.Query(ctx, `
			SELECT id, user_id, address, derivation_index, encrypted_privkey,
			       network, status, has_received_usdt, created_at, updated_at, assigned_at
			FROM user_deposit_addresses
			WHERE status = $1
			ORDER BY derivation_index
			LIMIT $2 OFFSET $3
		`, status, pageSize, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("deposit address repo: list all: %w", err)
		}
		defer rows.Close()
		return scanAddresses(rows, total)
	}

	var total int64
	err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM user_deposit_addresses`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("deposit address repo: count all: %w", err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, address, derivation_index, encrypted_privkey,
		       network, status, has_received_usdt, created_at, updated_at, assigned_at
		FROM user_deposit_addresses
		ORDER BY derivation_index
		LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("deposit address repo: list all: %w", err)
	}
	defer rows.Close()
	return scanAddresses(rows, total)
}

func scanAddresses(rows pgx.Rows, total int64) ([]model.DepositAddress, int64, error) {
	var addrs []model.DepositAddress
	for rows.Next() {
		var a model.DepositAddress
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.Address, &a.DerivationIndex, &a.EncryptedPrivkey,
			&a.Network, &a.Status, &a.HasReceivedUSDT, &a.CreatedAt, &a.UpdatedAt, &a.AssignedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("deposit address repo: scan: %w", err)
		}
		addrs = append(addrs, a)
	}
	return addrs, total, rows.Err()
}

var ErrAddressPoolEmpty = errors.New("address pool empty: no available addresses")
