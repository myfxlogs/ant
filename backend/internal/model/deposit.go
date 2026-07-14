package model

import (
	"time"

	"github.com/google/uuid"
)

// DepositRequest represents a USDT deposit request from a user.
type DepositRequest struct {
	ID          uuid.UUID  `db:"id"`
	UserID      uuid.UUID  `db:"user_id"`
	Amount      string     `db:"amount"`       // USDT amount (decimal string)
	AmountUSD   string     `db:"amount_usd"`   // USD amount to credit (decimal string)
	TxHash      *string    `db:"tx_hash"`      // optional on-chain tx hash
	Status      string     `db:"status"`       // PENDING / APPROVED / REJECTED
	ReviewerID  *uuid.UUID `db:"reviewer_id"`  // admin who approved/rejected
	ReviewNote  *string    `db:"review_note"`  // admin note
	ReviewedAt  *time.Time `db:"reviewed_at"`
	WalletTxID  *uuid.UUID `db:"wallet_tx_id"` // wallet_transactions.id on approval
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	// Joined field for admin view
	UserEmail *string `db:"user_email"`
}
