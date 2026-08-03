package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// CreditAccount holds a user's credit balance (1 credit = $0.01).
type CreditAccount struct {
	ID            uuid.UUID `db:"id"`
	UserID        uuid.UUID `db:"user_id"`
	Balance       string    `db:"balance"`
	FrozenBalance string    `db:"frozen_balance"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// CreditTransaction is an immutable audit trail entry for credit changes.
type CreditTransaction struct {
	ID            uuid.UUID  `db:"id"`
	AccountID     uuid.UUID  `db:"account_id"`
	UserID        uuid.UUID  `db:"user_id"`
	TxType        string     `db:"tx_type"`
	Amount        string     `db:"amount"`
	BalanceBefore string     `db:"balance_before"`
	BalanceAfter  string     `db:"balance_after"`
	Source        *string    `db:"source"`
	Description   *string    `db:"description"`
	OperatorID    *uuid.UUID `db:"operator_id"`
	RelatedTxID   *uuid.UUID `db:"related_tx_id"`
	CreatedAt     time.Time  `db:"created_at"`
}

// CreditRepository manages credit accounts and transactions.
type CreditRepository struct {
	db *pgxpool.Pool
}

func NewCreditRepository(db *pgxpool.Pool) *CreditRepository {
	return &CreditRepository{db: db}
}

// GetOrCreateAccount returns the user's credit account, creating one if needed.
func (r *CreditRepository) GetOrCreateAccount(ctx context.Context, userID uuid.UUID) (*CreditAccount, error) {
	var acc CreditAccount
	err := r.db.QueryRow(ctx,
		`INSERT INTO credit_accounts (user_id) VALUES ($1)
		 ON CONFLICT (user_id) DO UPDATE SET updated_at = NOW()
		 RETURNING id, user_id, balance, frozen_balance, created_at, updated_at`,
		userID).Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.FrozenBalance, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get or create credit account: %w", err)
	}
	return &acc, nil
}

// GetBalance returns the user's current credit balance as decimal.
func (r *CreditRepository) GetBalance(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	acc, err := r.GetOrCreateAccount(ctx, userID)
	if err != nil {
		return decimal.Zero, err
	}
	bal, err := decimal.NewFromString(acc.Balance)
	if err != nil {
		return decimal.Zero, err
	}
	return bal, nil
}

// AddCredits adds credits to the user's account (deposit, grant, etc.).
func (r *CreditRepository) AddCredits(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, txType, source, description string, operatorID *uuid.UUID) (*CreditTransaction, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be positive for AddCredits")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var acc CreditAccount
	err = tx.QueryRow(ctx,
		`INSERT INTO credit_accounts (user_id, balance) VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET balance = credit_accounts.balance + $2, updated_at = NOW()
		 RETURNING id, user_id, balance, frozen_balance, created_at, updated_at`,
		userID, amount.StringFixed(8)).Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.FrozenBalance, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("add credits: %w", err)
	}

	balBefore, _ := decimal.NewFromString(acc.Balance)
	balBefore = balBefore.Sub(amount)

	var ct CreditTransaction
	err = tx.QueryRow(ctx,
		`INSERT INTO credit_transactions (account_id, user_id, tx_type, amount, balance_before, balance_after, source, description, operator_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, account_id, user_id, tx_type, amount, balance_before, balance_after, source, description, operator_id, related_tx_id, created_at`,
		acc.ID, userID, txType, amount.StringFixed(8), balBefore.StringFixed(8), acc.Balance, source, description, operatorID).
		Scan(&ct.ID, &ct.AccountID, &ct.UserID, &ct.TxType, &ct.Amount, &ct.BalanceBefore, &ct.BalanceAfter, &ct.Source, &ct.Description, &ct.OperatorID, &ct.RelatedTxID, &ct.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert credit tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &ct, nil
}

// HoldCredits freezes credits for a pending AI session (pre-deduction).
// Moves amount from balance to frozen_balance.
func (r *CreditRepository) HoldCredits(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, description string) (*CreditTransaction, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("hold amount must be positive")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var acc CreditAccount
	err = tx.QueryRow(ctx,
		`UPDATE credit_accounts
		 SET balance = balance - $2, frozen_balance = frozen_balance + $2, updated_at = NOW()
		 WHERE user_id = $1 AND balance >= $2
		 RETURNING id, user_id, balance, frozen_balance, created_at, updated_at`,
		userID, amount.StringFixed(8)).Scan(&acc.ID, &acc.UserID, &acc.Balance, &acc.FrozenBalance, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insufficient balance for hold: %w", err)
	}

	balBefore, _ := decimal.NewFromString(acc.Balance)
	balBefore = balBefore.Add(amount)

	var ct CreditTransaction
	err = tx.QueryRow(ctx,
		`INSERT INTO credit_transactions (account_id, user_id, tx_type, amount, balance_before, balance_after, source, description)
		 VALUES ($1, $2, 'ai_hold', $3, $4, $5, 'pre_deduction', $6)
		 RETURNING id, account_id, user_id, tx_type, amount, balance_before, balance_after, source, description, operator_id, related_tx_id, created_at`,
		acc.ID, userID, amount.StringFixed(8), balBefore.StringFixed(8), acc.Balance, description).
		Scan(&ct.ID, &ct.AccountID, &ct.UserID, &ct.TxType, &ct.Amount, &ct.BalanceBefore, &ct.BalanceAfter, &ct.Source, &ct.Description, &ct.OperatorID, &ct.RelatedTxID, &ct.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert hold tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &ct, nil
}

// SettleCredits settles a hold: deducts actual cost from frozen, releases remainder to balance.
// holdAmount is the original hold, actualCost is the real cost. If actualCost < holdAmount,
// the difference is released back to balance.
func (r *CreditRepository) SettleCredits(ctx context.Context, userID uuid.UUID, holdAmount, actualCost decimal.Decimal, description string) error {
	if actualCost.LessThan(decimal.Zero) {
		return fmt.Errorf("actual cost cannot be negative")
	}
	if actualCost.GreaterThan(holdAmount) {
		// Need to deduct more than held — take extra from balance.
		extra := actualCost.Sub(holdAmount)
		return r.settleWithExtra(ctx, userID, holdAmount, extra, description)
	}

	release := holdAmount.Sub(actualCost)
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Release unused hold back to balance.
	if release.GreaterThan(decimal.Zero) {
		_, err = tx.Exec(ctx,
			`UPDATE credit_accounts
			 SET frozen_balance = frozen_balance - $2, balance = balance + $2, updated_at = NOW()
			 WHERE user_id = $1`,
			userID, release.StringFixed(8))
		if err != nil {
			return fmt.Errorf("release frozen: %w", err)
		}
	}

	// Deduct actual cost from frozen.
	if actualCost.GreaterThan(decimal.Zero) {
		_, err = tx.Exec(ctx,
			`UPDATE credit_accounts
			 SET frozen_balance = frozen_balance - $2, updated_at = NOW()
			 WHERE user_id = $1`,
			userID, actualCost.StringFixed(8))
		if err != nil {
			return fmt.Errorf("deduct frozen: %w", err)
		}

		var accID uuid.UUID
		var balAfter string
		_ = tx.QueryRow(ctx,
			`SELECT id, balance FROM credit_accounts WHERE user_id = $1`, userID).Scan(&accID, &balAfter)

		_, err = tx.Exec(ctx,
			`INSERT INTO credit_transactions (account_id, user_id, tx_type, amount, balance_before, balance_after, source, description)
			 VALUES ($1, $2, 'ai_usage', $3, $4, $5, 'settlement', $6)`,
			accID, userID, actualCost.StringFixed(8), balAfter, balAfter, description)
		if err != nil {
			return fmt.Errorf("insert settlement tx: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *CreditRepository) settleWithExtra(ctx context.Context, userID uuid.UUID, holdAmount, extra decimal.Decimal, description string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Deduct entire hold from frozen.
	_, err = tx.Exec(ctx,
		`UPDATE credit_accounts SET frozen_balance = frozen_balance - $2, updated_at = NOW() WHERE user_id = $1`,
		userID, holdAmount.StringFixed(8))
	if err != nil {
		return fmt.Errorf("deduct frozen: %w", err)
	}

	// Deduct extra from balance.
	_, err = tx.Exec(ctx,
		`UPDATE credit_accounts SET balance = balance - $2, updated_at = NOW() WHERE user_id = $1 AND balance >= $2`,
		userID, extra.StringFixed(8))
	if err != nil {
		return fmt.Errorf("deduct extra: %w", err)
	}

	var accID uuid.UUID
	var balAfter string
	_ = tx.QueryRow(ctx, `SELECT id, balance FROM credit_accounts WHERE user_id = $1`, userID).Scan(&accID, &balAfter)

	totalCost := holdAmount.Add(extra)
	_, err = tx.Exec(ctx,
		`INSERT INTO credit_transactions (account_id, user_id, tx_type, amount, balance_before, balance_after, source, description)
		 VALUES ($1, $2, 'ai_usage', $3, $4, $5, 'settlement', $6)`,
		accID, userID, totalCost.StringFixed(8), balAfter, balAfter, description)
	if err != nil {
		return fmt.Errorf("insert settlement tx: %w", err)
	}

	return tx.Commit(ctx)
}

// ListTransactions returns paginated credit transactions for a user.
func (r *CreditRepository) ListTransactions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*CreditTransaction, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM credit_transactions WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, account_id, user_id, tx_type, amount, balance_before, balance_after, source, description, operator_id, related_tx_id, created_at
		 FROM credit_transactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []*CreditTransaction
	for rows.Next() {
		var ct CreditTransaction
		if err := rows.Scan(&ct.ID, &ct.AccountID, &ct.UserID, &ct.TxType, &ct.Amount, &ct.BalanceBefore, &ct.BalanceAfter, &ct.Source, &ct.Description, &ct.OperatorID, &ct.RelatedTxID, &ct.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, &ct)
	}
	return out, total, rows.Err()
}
