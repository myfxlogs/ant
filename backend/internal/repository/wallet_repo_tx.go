package repository

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"alphaforge/internal/model"
)

// ledgerChainInsert is the shared helper for inserting a wallet_transaction with
// hash chain linkage, idempotency, and ledger_outbox write. Used by both
// AdjustBalanceTx and freezeOp to avoid code duplication (R7/R8).
//
// Steps: idempotency check → advisory lock → read chain tail → (caller updates balance) →
// insert transaction → compute entry_hash → update entry_hash → write outbox.
//
// Returns (prevHash, txID, seq, error) from the chain tail lock + insert.
// If idemKey already exists, returns ErrIdempotentReplay.
// If a concurrent insert races on idem_key unique index, returns ErrIdempotentReplay.
// ledgerInsertParams holds parameters for ledgerChainInsert.
type ledgerInsertParams struct {
	Tx            pgx.Tx
	WalletID      uuid.UUID
	UserID        uuid.UUID
	Amount        string
	TxType        string
	Description   string
	OperatorID    *uuid.UUID
	IdemKey       string
	BalanceBefore string
	BalanceAfter  string
}

func (r *WalletRepository) ledgerChainInsert(
	ctx context.Context, p ledgerInsertParams,
) (txID uuid.UUID, seq int64, err error) {
	// 1. Idempotency check (R7).
	var existing uuid.UUID
	err = p.Tx.QueryRow(ctx,
		`SELECT wallet_id FROM wallet_transactions WHERE idem_key = $1`, p.IdemKey,
	).Scan(&existing)
	if err == nil {
		return uuid.Nil, 0, model.ErrIdempotentReplay
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, 0, fmt.Errorf("ledger: idem check: %w", err)
	}

	// 2. Advisory lock to serialize chain operations (R8).
	// pg_advisory_xact_lock works even on an empty table where FOR UPDATE would
	// have no row to lock, preventing a race where two concurrent first-ever
	// transactions both get prev_hash = nil.
	const chainLockKey = 20826
	if _, err := p.Tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, chainLockKey); err != nil {
		return uuid.Nil, 0, fmt.Errorf("ledger: advisory lock: %w", err)
	}

	var prevHash []byte
	err = p.Tx.QueryRow(ctx,
		`SELECT entry_hash FROM wallet_transactions ORDER BY seq DESC LIMIT 1`,
	).Scan(&prevHash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, 0, fmt.Errorf("ledger: read chain tail: %w", err)
	}

	// 3. Insert transaction with hash chain.
	err = p.Tx.QueryRow(ctx, `
		INSERT INTO wallet_transactions (wallet_id, user_id, tx_type, amount, balance_before, balance_after, description, operator_id, prev_hash, idem_key)
		VALUES ($1, $2, $3, $4::numeric, $5::numeric, $6::numeric, $7, $8, $9, $10)
		RETURNING id, seq
	`, p.WalletID, p.UserID, p.TxType, p.Amount, p.BalanceBefore, p.BalanceAfter, p.Description, p.OperatorID, prevHash, p.IdemKey).Scan(&txID, &seq)
	if err != nil {
		// Concurrent race on idem_key unique index → idempotent replay (R7).
		if isUniqueViolation(err) {
			return uuid.Nil, 0, model.ErrIdempotentReplay
		}
		return uuid.Nil, 0, fmt.Errorf("ledger: insert transaction: %w", err)
	}

	// 4. Compute entry_hash = SHA256(prev_hash || seq || wallet_id || tx_type || amount || balance_before || balance_after || idem_key).
	entryHash := computeEntryHash(prevHash, seq, p.WalletID, p.TxType, p.Amount, p.BalanceBefore, p.BalanceAfter, p.IdemKey)

	// 5. Update entry_hash (separate UPDATE because seq is GENERATED ALWAYS AS IDENTITY).
	_, err = p.Tx.Exec(ctx,
		`UPDATE wallet_transactions SET entry_hash = $1 WHERE id = $2`, entryHash, txID,
	)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("ledger: set entry_hash: %w", err)
	}

	// 6. Write to ledger_outbox for external notification (R8).
	// Split INSERT and NOTIFY into separate calls — pgx prepared statements
	// do not support multiple SQL commands in a single Exec (SQLSTATE 42601).
	_, err = p.Tx.Exec(ctx,
		`INSERT INTO ledger_outbox (seq, entry_hash) VALUES ($1, $2)`,
		seq, entryHash,
	)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("ledger: outbox insert: %w", err)
	}
	_, err = p.Tx.Exec(ctx, `NOTIFY ledger_outbox`)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("ledger: outbox notify: %w", err)
	}

	return txID, seq, nil
}

// walletAfterUpdate re-queries the wallet row after a balance/frozen update.
func (r *WalletRepository) walletAfterUpdate(ctx context.Context, tx pgx.Tx, walletID, txID uuid.UUID) (*model.Wallet, error) {
	var w model.Wallet
	err := tx.QueryRow(ctx, `
		SELECT w.id, w.user_id, w.balance::text, w.frozen_balance::text, w.currency,
		       w.created_at, w.updated_at, u.account_number
		FROM user_wallets w
		JOIN users u ON u.id = w.user_id
		WHERE w.id = $1
	`, walletID).Scan(&w.ID, &w.UserID, &w.Balance, &w.FrozenBalance, &w.Currency,
		&w.CreatedAt, &w.UpdatedAt, &w.AccountNumber)
	if err != nil {
		return nil, fmt.Errorf("ledger: re-query wallet: %w", err)
	}
	w.LastTransactionID = &txID
	return &w, nil
}

// computeEntryHash calculates SHA256(prev_hash || seq || wallet_id || tx_type || amount || balance_before || balance_after || idem_key).
func computeEntryHash(prevHash []byte, seq int64, walletID uuid.UUID, txType, amount, balanceBefore, balanceAfter, idemKey string) []byte {
	h := sha256.New()
	h.Write(prevHash)
	var seqBuf [8]byte
	binary.BigEndian.PutUint64(seqBuf[:], uint64(seq))
	h.Write(seqBuf[:])
	h.Write(walletID[:])
	h.Write([]byte(txType))
	h.Write([]byte(amount))
	h.Write([]byte(balanceBefore))
	h.Write([]byte(balanceAfter))
	h.Write([]byte(idemKey))
	return h.Sum(nil)
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

// WriteCredentialChangeLedger inserts a credential/whitelist change into the hash chain
// with zero amount (no balance impact) so it's tamper-evident and forwarded to the
// off-host ledger mirror via ledger_outbox (R12). coldsign compares its mirror to detect
// unauthorized credential changes.
func (r *WalletRepository) WriteCredentialChangeLedger(
	ctx context.Context, userID uuid.UUID, txType, description, idemKey string,
) error {
	tx, err := r.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("credential change ledger: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	wallet, err := r.getWalletByUserIDTx(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("credential change ledger: get wallet: %w", err)
	}

	_, _, err = r.ledgerChainInsert(ctx, ledgerInsertParams{
		Tx: tx, WalletID: wallet.ID, UserID: userID, Amount: "0",
		TxType: txType, Description: description, IdemKey: idemKey,
		BalanceBefore: wallet.Balance, BalanceAfter: wallet.Balance,
	})
	if err != nil {
		if errors.Is(err, model.ErrIdempotentReplay) {
			return nil
		}
		return fmt.Errorf("credential change ledger: chain insert: %w", err)
	}

	return tx.Commit(ctx)
}

// getWalletByUserIDTx reads the wallet within an existing transaction.
func (r *WalletRepository) getWalletByUserIDTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.Wallet, error) {
	var w model.Wallet
	err := tx.QueryRow(ctx, `
		SELECT w.id, w.user_id, w.balance::text, w.frozen_balance::text, w.currency,
		       w.created_at, w.updated_at
		FROM user_wallets w WHERE w.user_id = $1
	`, userID).Scan(&w.ID, &w.UserID, &w.Balance, &w.FrozenBalance, &w.Currency,
		&w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("wallet repo: get wallet by user: %w", err)
	}
	return &w, nil
}
