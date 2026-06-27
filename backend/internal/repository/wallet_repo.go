package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"anttrader/internal/model"
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
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("wallet repo: get by user id: %w", err)
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
		// If no row returned (ON CONFLICT DO NOTHING), the wallet already exists — re-query.
		if err == pgx.ErrNoRows {
			return r.GetByUserID(ctx, userID)
		}
		return nil, fmt.Errorf("wallet repo: create: %w", err)
	}
	// Fetch account_number for the response.
	var acctNum *string
	_ = r.pg.QueryRow(ctx, `SELECT account_number FROM users WHERE id = $1`, userID).Scan(&acctNum)
	w.AccountNumber = acctNum
	return &w, nil
}

// AdjustBalanceTx updates the wallet balance within a transaction and records the transaction.
// amount can be positive (credit) or negative (debit).
func (r *WalletRepository) AdjustBalanceTx(ctx context.Context, tx pgx.Tx, walletID, userID uuid.UUID, amount, txType, description string, operatorID *uuid.UUID) (*model.Wallet, error) {
	// Lock the wallet row.
	var balanceBefore, balanceAfter string
	err := tx.QueryRow(ctx,
		`SELECT balance::text FROM user_wallets WHERE id = $1 FOR UPDATE`, walletID,
	).Scan(&balanceBefore)
	if err != nil {
		return nil, fmt.Errorf("wallet repo: lock wallet: %w", err)
	}

	// Update balance.
	err = tx.QueryRow(ctx, `
		UPDATE user_wallets
		SET balance = balance + ($1)::numeric, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING balance::text
	`, amount, walletID).Scan(&balanceAfter)
	if err != nil {
		return nil, fmt.Errorf("wallet repo: adjust balance: %w", err)
	}

	// Record transaction.
	_, err = tx.Exec(ctx, `
		INSERT INTO wallet_transactions (wallet_id, user_id, tx_type, amount, balance_before, balance_after, description, operator_id)
		VALUES ($1, $2, $3, $4::numeric, $5::numeric, $6::numeric, $7, $8)
	`, walletID, userID, txType, amount, balanceBefore, balanceAfter, description, operatorID)
	if err != nil {
		return nil, fmt.Errorf("wallet repo: insert transaction: %w", err)
	}

	// Return full wallet info including account_number, frozen_balance, currency.
	var w model.Wallet
	err = tx.QueryRow(ctx, `
		SELECT w.id, w.user_id, w.balance::text, w.frozen_balance::text, w.currency,
		       w.created_at, w.updated_at, u.account_number
		FROM user_wallets w
		JOIN users u ON u.id = w.user_id
		WHERE w.id = $1
	`, walletID).Scan(&w.ID, &w.UserID, &w.Balance, &w.FrozenBalance, &w.Currency,
		&w.CreatedAt, &w.UpdatedAt, &w.AccountNumber)
	if err != nil {
		return nil, fmt.Errorf("wallet repo: re-query after adjust: %w", err)
	}
	return &w, nil
}

// ListTransactions returns a paginated list of wallet transactions for a user.
func (r *WalletRepository) ListTransactions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]model.WalletTransaction, int64, error) {
	// Count total.
	var total int64
	err := r.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM wallet_transactions WHERE user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("wallet repo: count transactions: %w", err)
	}

	offset := (page - 1) * pageSize
	rows, err := r.pg.Query(ctx, `
		SELECT id, wallet_id, user_id, tx_type, amount::text, balance_before::text,
		       balance_after::text, COALESCE(description, ''), operator_id, created_at
		FROM wallet_transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("wallet repo: list transactions: %w", err)
	}
	defer rows.Close()

	var txs []model.WalletTransaction
	for rows.Next() {
		var t model.WalletTransaction
		if err := rows.Scan(&t.ID, &t.WalletID, &t.UserID, &t.TxType, &t.Amount,
			&t.BalanceBefore, &t.BalanceAfter, &t.Description, &t.OperatorID, &t.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("wallet repo: scan transaction: %w", err)
		}
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("wallet repo: iterate transactions: %w", err)
	}
	return txs, total, nil
}
