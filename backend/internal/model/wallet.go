package model

import (
	"time"

	"github.com/google/uuid"
)

// ErrIdempotentReplay indicates an AdjustBalanceTx call was a no-op due to a duplicate idem_key.
var ErrIdempotentReplay = errIdempotentReplay{}

type errIdempotentReplay struct{}

func (errIdempotentReplay) Error() string { return "idempotent replay: idem_key already exists" }
func (errIdempotentReplay) Is(target error) bool {
	_, ok := target.(errIdempotentReplay)
	return ok
}

// ErrInsufficientBalance indicates a freeze/withdrawal operation failed because
// the wallet balance or frozen_balance was less than the required amount.
var ErrInsufficientBalance = errInsufficientBalance{}

type errInsufficientBalance struct{}

func (errInsufficientBalance) Error() string { return "insufficient balance for operation" }
func (errInsufficientBalance) Is(target error) bool {
	_, ok := target.(errInsufficientBalance)
	return ok
}

// Wallet represents a user's platform wallet.
type Wallet struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	UserID            uuid.UUID  `json:"user_id" db:"user_id"`
	Balance           string     `json:"balance" db:"balance"`
	FrozenBalance     string     `json:"frozen_balance" db:"frozen_balance"`
	Currency          string     `json:"currency" db:"currency"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
	AccountNumber     *string    `json:"account_number,omitempty" db:"-"`
	LastTransactionID *uuid.UUID `json:"last_transaction_id,omitempty" db:"-"`
}

// WalletTransaction is an immutable (append-only) record of a balance change.
// Hash chain: entry_hash = SHA256(prev_hash || seq || wallet_id || tx_type || amount || balance_before || balance_after || idem_key).
type WalletTransaction struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	WalletID      uuid.UUID  `json:"wallet_id" db:"wallet_id"`
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	TxType        string     `json:"tx_type" db:"tx_type"`
	Amount        string     `json:"amount" db:"amount"`
	BalanceBefore string     `json:"balance_before" db:"balance_before"`
	BalanceAfter  string     `json:"balance_after" db:"balance_after"`
	Description   *string    `json:"description,omitempty" db:"description"`
	OperatorID    *uuid.UUID `json:"operator_id,omitempty" db:"operator_id"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	Seq           int64      `json:"seq" db:"seq"`
	PrevHash      []byte     `json:"prev_hash,omitempty" db:"prev_hash"`
	EntryHash     []byte     `json:"entry_hash,omitempty" db:"entry_hash"`
	IdemKey       *string    `json:"idem_key,omitempty" db:"idem_key"`
}
