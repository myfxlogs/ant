package model

import (
	"time"

	"github.com/google/uuid"
)

// DepositAddress represents a TRC20 address derived on-demand from the account xpub.
type DepositAddress struct {
	ID              uuid.UUID  `db:"id"`
	UserID          *uuid.UUID `db:"user_id"`
	Address         string     `db:"address"`
	DerivationIndex int        `db:"derivation_index"`
	Network         string     `db:"network"`
	Status          string     `db:"status"` // ASSIGNED / RETIRED
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

// SweepLog represents a single leg of a sweep operation (ADR §2.3).
// One sweep = 3 legs: delegate (leg_seq=0), transfer (leg_seq=1), undelegate (leg_seq=2).
// All 3 legs share the same batch_id.
type SweepLog struct {
	ID               uuid.UUID  `db:"id"`
	BatchID          uuid.UUID  `db:"batch_id"`
	DepositAddressID uuid.UUID  `db:"deposit_address_id"`
	LegType          string     `db:"leg_type"`   // delegate / transfer / undelegate
	LegSeq           int        `db:"leg_seq"`    // 0 / 1 / 2
	TxHash           string     `db:"tx_hash"`
	Amount           string     `db:"amount"`
	EnergyUsed       int64      `db:"energy_used"`
	Status           string     `db:"status"` // PENDING / SWEEPING / DONE / FAILED / MANUAL_REVIEW
	ErrorMessage     *string    `db:"error_message"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
	CompletedAt      *time.Time `db:"completed_at"`
}
