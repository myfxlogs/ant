package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"alphaforge/internal/model"
)

// WalletRepository provides database access for user wallets and transactions.
type WalletRepository struct {
	pg *pgxpool.Pool
}

func NewWalletRepository(pg *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{pg: pg}
}

// GetByUserID returns the wallet for a given user, or nil if not found.
func (r *WalletRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
	var w model.Wallet
	err := r.pg.QueryRow(ctx, `
		SELECT w.id, w.user_id, w.balance::text, w.frozen_balance::text, w.currency,
		       w.created_at, w.updated_at, u.account_number
		FROM user_wallets w
		JOIN users u ON u.id = w.user_id
		WHERE w.user_id = $1
	`, userID).Scan(&w.ID, &w.UserID, &w.Balance, &w.FrozenBalance, &w.Currency,
		&w.CreatedAt, &w.UpdatedAt, &w.AccountNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("wallet repo: get by user id: %w", err)
	}
	return &w, nil
}

// GetByUserIDTx returns the wallet for a given user within a transaction, or nil if not found.
func (r *WalletRepository) GetByUserIDTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.Wallet, error) {
	var w model.Wallet
	err := tx.QueryRow(ctx, `
		SELECT w.id, w.user_id, w.balance::text, w.frozen_balance::text, w.currency,
		       w.created_at, w.updated_at
		FROM user_wallets w
		WHERE w.user_id = $1
		FOR UPDATE
	`, userID).Scan(&w.ID, &w.UserID, &w.Balance, &w.FrozenBalance, &w.Currency,
		&w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("wallet repo: get by user id (tx): %w", err)
	}
	return &w, nil
}

// CreateWallet inserts a new wallet for a user (initial balance 0).
// Uses ON CONFLICT DO NOTHING to be idempotent — safe for concurrent calls.
// Returns the wallet (newly created or existing if a race occurred).
func (r *WalletRepository) CreateWallet(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
	var w model.Wallet
	err := r.pg.QueryRow(ctx, `
		INSERT INTO user_wallets (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
		RETURNING id, user_id, balance::text, frozen_balance::text, currency, created_at, updated_at
	`, userID).Scan(&w.ID, &w.UserID, &w.Balance, &w.FrozenBalance, &w.Currency,
		&w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.GetByUserID(ctx, userID)
		}
		return nil, fmt.Errorf("wallet repo: create: %w", err)
	}
	var acctNum *string
	_ = r.pg.QueryRow(ctx, `SELECT account_number FROM users WHERE id = $1`, userID).Scan(&acctNum)
	w.AccountNumber = acctNum
	return &w, nil
}

// CreateWalletTx inserts a new wallet within a transaction (initial balance 0).
// Uses ON CONFLICT DO NOTHING to be idempotent. If the wallet already exists,
// re-queries it within the same transaction. The caller must commit/rollback.
func (r *WalletRepository) CreateWalletTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.Wallet, error) {
	var w model.Wallet
	err := tx.QueryRow(ctx, `
		INSERT INTO user_wallets (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
		RETURNING id, user_id, balance::text, frozen_balance::text, currency, created_at, updated_at
	`, userID).Scan(&w.ID, &w.UserID, &w.Balance, &w.FrozenBalance, &w.Currency,
		&w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.GetByUserIDTx(ctx, tx, userID)
		}
		return nil, fmt.Errorf("wallet repo: create (tx): %w", err)
	}
	return &w, nil
}

// AdjustBalanceTx updates the wallet balance within a transaction and records
// the transaction with hash chain linkage and idempotency (R7/R8/R9).
// idemKey must be unique per logical operation; empty = auto-generate UUID.
// Returns ErrIdempotentReplay if idemKey already exists (no-op, safe to ignore).
func (r *WalletRepository) AdjustBalanceTx(ctx context.Context, tx pgx.Tx, walletID, userID uuid.UUID, amount, txType, description string, operatorID *uuid.UUID, idemKey string) (*model.Wallet, error) {
	if idemKey == "" {
		idemKey = "auto-" + uuid.New().String()
	}

	// 1. Idempotency check (R7) — must happen BEFORE balance update to prevent
	// double-credit on retry. If idem_key exists, return the existing wallet state.
	var existingWalletID uuid.UUID
	err := tx.QueryRow(ctx,
		`SELECT wallet_id FROM wallet_transactions WHERE idem_key = $1`, idemKey,
	).Scan(&existingWalletID)
	if err == nil {
		// Idempotent replay — return current wallet state without modifying balance.
		return r.walletAfterUpdate(ctx, tx, walletID, uuid.Nil)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("wallet repo: idem check: %w", err)
	}

	// 2. Lock wallet row and update balance (R9: CHECK ensures >= 0).
	var balanceBefore string
	err = tx.QueryRow(ctx,
		`SELECT balance::text FROM user_wallets WHERE id = $1 FOR UPDATE`, walletID,
	).Scan(&balanceBefore)
	if err != nil {
		return nil, fmt.Errorf("wallet repo: lock wallet: %w", err)
	}

	var balanceAfter string
	err = tx.QueryRow(ctx, `
		UPDATE user_wallets
		SET balance = balance + ($1)::numeric, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING balance::text
	`, amount, walletID).Scan(&balanceAfter)
	if err != nil {
		if isCheckViolation(err) {
			return nil, model.ErrInsufficientBalance
		}
		return nil, fmt.Errorf("wallet repo: adjust balance: %w", err)
	}

	// 3. Insert transaction with hash chain + idempotency + outbox (shared helper).
	txID, err := r.ledgerChainInsert(ctx, ledgerInsertParams{
		Tx: tx, WalletID: walletID, UserID: userID, Amount: amount,
		TxType: txType, Description: description, OperatorID: operatorID,
		IdemKey: idemKey, BalanceBefore: balanceBefore, BalanceAfter: balanceAfter,
	})
	if err != nil {
		// Concurrent race: another transaction inserted the same idem_key
		// between our idempotency check and INSERT. Undo the balance update
		// to prevent double-credit, then return as idempotent replay.
		if errors.Is(err, model.ErrIdempotentReplay) {
			_, _ = tx.Exec(ctx,
				`UPDATE user_wallets SET balance = balance - ($1)::numeric WHERE id = $2`,
				amount, walletID)
			return r.walletAfterUpdate(ctx, tx, walletID, uuid.Nil)
		}
		return nil, err
	}

	// 4. Return updated wallet.
	return r.walletAfterUpdate(ctx, tx, walletID, txID)
}

// isUniqueViolation checks if err is a PostgreSQL unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isCheckViolation checks if err is a PostgreSQL CHECK constraint violation (SQLSTATE 23514).
func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}
