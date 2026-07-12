package model

import (
	"time"

	"github.com/google/uuid"
)

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

// WalletTransaction is an immutable record of a balance change.
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
}
