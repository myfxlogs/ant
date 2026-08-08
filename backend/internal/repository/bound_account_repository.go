package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BoundAccountRow is the denormalized view of a bound MT account.
type BoundAccountRow struct {
	MTAccountID   uuid.UUID
	Login         string
	Broker        string
	Server        string
	MTType        string
	AccountStatus string
	BoundAt       time.Time
}

// BoundAccountRepository manages subscription_bound_accounts.
type BoundAccountRepository struct {
	pg *pgxpool.Pool
}

func NewBoundAccountRepository(pg *pgxpool.Pool) *BoundAccountRepository {
	return &BoundAccountRepository{pg: pg}
}

// CountBoundAccounts returns the number of MT accounts bound to a user's subscription.
func (r *BoundAccountRepository) CountBoundAccounts(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM subscription_bound_accounts WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("bound account repo: count: %w", err)
	}
	return count, nil
}

// IsAccountBound returns true if the MT account is bound to the user's subscription.
func (r *BoundAccountRepository) IsAccountBound(ctx context.Context, userID, accountID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pg.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM subscription_bound_accounts WHERE user_id = $1 AND mt_account_id = $2)`,
		userID, accountID).Scan(&exists)
	return exists, err
}

// BindAccount binds an MT account to the user's subscription.
func (r *BoundAccountRepository) BindAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	_, err := r.pg.Exec(ctx,
		`INSERT INTO subscription_bound_accounts (user_id, mt_account_id) VALUES ($1, $2)
		 ON CONFLICT (user_id, mt_account_id) DO NOTHING`,
		userID, accountID)
	if err != nil {
		return fmt.Errorf("bound account repo: bind: %w", err)
	}
	return nil
}

// UnbindAccount removes an MT account binding and returns the number of active schedules
// that were using it (caller should stop those schedules).
func (r *BoundAccountRepository) UnbindAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	_, err := r.pg.Exec(ctx,
		`DELETE FROM subscription_bound_accounts WHERE user_id = $1 AND mt_account_id = $2`,
		userID, accountID)
	if err != nil {
		return fmt.Errorf("bound account repo: unbind: %w", err)
	}
	return nil
}

// ListBoundAccounts returns all bound MT accounts for a user with denormalized account info.
func (r *BoundAccountRepository) ListBoundAccounts(ctx context.Context, userID uuid.UUID) ([]BoundAccountRow, error) {
	rows, err := r.pg.Query(ctx,
		`SELECT sba.mt_account_id, ma.login, ma.broker_company, ma.broker_server, ma.mt_type, ma.account_status, sba.bound_at
		 FROM subscription_bound_accounts sba
		 JOIN mt_accounts ma ON ma.id = sba.mt_account_id
		 WHERE sba.user_id = $1 AND ma.deleted_at IS NULL
		 ORDER BY sba.bound_at ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("bound account repo: list: %w", err)
	}
	defer rows.Close()

	var out []BoundAccountRow
	for rows.Next() {
		var r BoundAccountRow
		if err := rows.Scan(&r.MTAccountID, &r.Login, &r.Broker, &r.Server, &r.MTType, &r.AccountStatus, &r.BoundAt); err != nil {
			return nil, fmt.Errorf("bound account repo: scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
