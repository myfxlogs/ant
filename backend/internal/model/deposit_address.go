package model

import (
	"time"

	"github.com/google/uuid"
)

// DepositAddress represents a TRC20 address in the HD wallet address pool.
type DepositAddress struct {
	ID              uuid.UUID  `db:"id"`
	UserID          *uuid.UUID `db:"user_id"`
	Address         string     `db:"address"`
	DerivationIndex int        `db:"derivation_index"`
	EncryptedPrivkey []byte    `db:"encrypted_privkey"`
	Network         string     `db:"network"`
	Status          string     `db:"status"` // AVAILABLE / ASSIGNED / RETIRED
	HasReceivedUSDT bool       `db:"has_received_usdt"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
	AssignedAt      *time.Time `db:"assigned_at"`
}

// Deposit represents an on-chain USDT deposit confirmed by the chain monitor.
type Deposit struct {
	ID               uuid.UUID `db:"id"`
	UserID           uuid.UUID `db:"user_id"`
	DepositAddressID uuid.UUID `db:"deposit_address_id"`
	TxHash           string    `db:"tx_hash"`
	Amount           string    `db:"amount"`
	BlockNumber      int64     `db:"block_number"`
	Confirmations    int       `db:"confirmations"`
	Status           string    `db:"status"` // CONFIRMED / MANUAL_REVIEW
	ConfirmedAt      *time.Time `db:"confirmed_at"`
	CreatedAt        time.Time `db:"created_at"`
}

// SweepLog represents a sweep (fund consolidation) operation record.
type SweepLog struct {
	ID               uuid.UUID  `db:"id"`
	DepositAddressID uuid.UUID  `db:"deposit_address_id"`
	TxHash           string     `db:"tx_hash"`
	Amount           string     `db:"amount"`
	EnergyUsed       int64      `db:"energy_used"`
	Status           string     `db:"status"` // PENDING / SWEEPING / DONE / FAILED
	ErrorMessage     *string    `db:"error_message"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
	CompletedAt      *time.Time `db:"completed_at"`
}
