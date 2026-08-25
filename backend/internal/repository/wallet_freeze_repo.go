package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"alphaforge/internal/model"
)

// FreezeForWithdrawal moves amount from balance to frozen_balance atomically (R9).
// balance -= amount, frozen_balance += amount in the same transaction.
// Records a wallet_transaction with hash chain linkage (R8) and idempotency (R7).
// Returns ErrIdempotentReplay if idemKey already exists.
func (r *WalletRepository) FreezeForWithdrawal(ctx context.Context, tx pgx.Tx, walletID, userID uuid.UUID, amount, idemKey string) (*model.Wallet, error) {
	return r.freezeOp(ctx, tx, walletID, userID, amount, idemKey, "withdrawal_freeze",
		"Freeze for withdrawal", true)
}

// CompleteWithdrawal deducts the frozen amount after successful on-chain broadcast (R9).
// frozen_balance -= amount. Balance is NOT changed (funds already left on-chain).
// Records a wallet_transaction with hash chain linkage.
func (r *WalletRepository) CompleteWithdrawal(ctx context.Context, tx pgx.Tx, walletID, userID uuid.UUID, amount, idemKey string) (*model.Wallet, error) {
	return r.freezeOp(ctx, tx, walletID, userID, amount, idemKey, "withdrawal_complete",
		"Withdrawal completed (broadcast confirmed)", false)
}

// CancelWithdrawal returns frozen amount to balance (R9).
// frozen_balance -= amount, balance += amount.
// Records a wallet_transaction with hash chain linkage.
func (r *WalletRepository) CancelWithdrawal(ctx context.Context, tx pgx.Tx, walletID, userID uuid.UUID, amount, idemKey string) (*model.Wallet, error) {
	return r.freezeOp(ctx, tx, walletID, userID, amount, idemKey, "withdrawal_cancel",
		"Withdrawal cancelled (funds returned to balance)", false)
}

// freezeOp is the shared implementation for all freeze-mode operations.
// freeze=true: balance -= amount, frozen += amount (initiate withdrawal).
// freeze=false & txType="withdrawal_complete": frozen -= amount (complete withdrawal).
// freeze=false & txType="withdrawal_cancel": frozen -= amount, balance += amount (cancel).
func (r *WalletRepository) freezeOp(ctx context.Context, tx pgx.Tx, walletID, userID uuid.UUID, amount, idemKey, txType, description string, freeze bool) (*model.Wallet, error) {
	// 1. Lock wallet row.
	var balanceBefore string
	err := tx.QueryRow(ctx,
		`SELECT balance::text FROM user_wallets WHERE id = $1 FOR UPDATE`, walletID,
	).Scan(&balanceBefore)
	if err != nil {
		return nil, fmt.Errorf("wallet freeze: lock wallet: %w", err)
	}

	// 2. Update balance and frozen_balance.
	var balanceAfter string
	if freeze {
		// balance -= amount, frozen += amount
		err = tx.QueryRow(ctx, `
			UPDATE user_wallets
			SET balance = balance - ($1)::numeric,
			    frozen_balance = frozen_balance + ($1)::numeric,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND balance >= ($1)::numeric
			RETURNING balance::text
		`, amount, walletID).Scan(&balanceAfter)
	} else if txType == "withdrawal_cancel" {
		// frozen -= amount, balance += amount
		err = tx.QueryRow(ctx, `
			UPDATE user_wallets
			SET frozen_balance = frozen_balance - ($1)::numeric,
			    balance = balance + ($1)::numeric,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND frozen_balance >= ($1)::numeric
			RETURNING balance::text
		`, amount, walletID).Scan(&balanceAfter)
	} else {
		// withdrawal_complete: frozen -= amount only
		err = tx.QueryRow(ctx, `
			UPDATE user_wallets
			SET frozen_balance = frozen_balance - ($1)::numeric,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND frozen_balance >= ($1)::numeric
			RETURNING balance::text
		`, amount, walletID).Scan(&balanceAfter)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrInsufficientBalance
		}
		return nil, fmt.Errorf("wallet freeze: update: %w", err)
	}

	// 3. Insert transaction with hash chain + idempotency + outbox (shared helper).
	txID, err := r.ledgerChainInsert(ctx, ledgerInsertParams{
		Tx: tx, WalletID: walletID, UserID: userID, Amount: amount,
		TxType: txType, Description: description, IdemKey: idemKey,
		BalanceBefore: balanceBefore, BalanceAfter: balanceAfter,
	})
	if err != nil {
		if errors.Is(err, model.ErrIdempotentReplay) {
			// Undo the balance/frozen update to prevent inconsistency.
			// The caller's defer rollback will also undo, but this keeps the
			// transaction state consistent if the caller continues using it.
			if freeze {
				_, _ = tx.Exec(ctx,
					`UPDATE user_wallets SET balance = balance + ($1)::numeric, frozen_balance = frozen_balance - ($1)::numeric WHERE id = $2`,
					amount, walletID)
			} else if txType == "withdrawal_cancel" {
				_, _ = tx.Exec(ctx,
					`UPDATE user_wallets SET frozen_balance = frozen_balance + ($1)::numeric, balance = balance - ($1)::numeric WHERE id = $2`,
					amount, walletID)
			} else {
				// withdrawal_complete: only frozen was decremented
				_, _ = tx.Exec(ctx,
					`UPDATE user_wallets SET frozen_balance = frozen_balance + ($1)::numeric WHERE id = $2`,
					amount, walletID)
			}
			return r.walletAfterUpdate(ctx, tx, walletID, uuid.Nil)
		}
		return nil, err
	}

	// 4. Return updated wallet.
	return r.walletAfterUpdate(ctx, tx, walletID, txID)
}
