package model

import (
	"time"

	"github.com/google/uuid"
)

// WebAuthnCredential is a user's registered passkey public key.
// The online server stores this for the registration ceremony.
// coldsign maintains its own copy (synced via USB, Q2/R12).
type WebAuthnCredential struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	CredentialID    string
	PublicKey       []byte
	AttestationType string
	AAGUID          string
	SignCount       int64
	Transports      []string
	Name            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// WithdrawalRequest tracks a user withdrawal through its lifecycle.
// PENDING → SIGNED → BROADCASTING → DONE / FAILED / CANCELLED
type WithdrawalRequest struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Amount       string
	DestAddress  string
	Nonce        int64
	CredentialID string
	Assertion    []byte
	Status       string
	BundleID     *uuid.UUID
	TxHash       *string
	IdemKey      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CompletedAt  *time.Time
}

// WithdrawalWhitelistEntry is a user-approved withdrawal destination (R12).
type WithdrawalWhitelistEntry struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Address      string
	Label        string
	Status       string // PENDING_CONFIRMATION / ACTIVE / REMOVED
	ConfirmedAt  *time.Time
	CooldownUntil *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CredentialChangeLog tracks all credential/whitelist mutations (R12).
type CredentialChangeLog struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	ChangeType string // CREDENTIAL_ADD / CREDENTIAL_REMOVE / WHITELIST_ADD / WHITELIST_REMOVE
	TargetID   string
	Status     string // PENDING / CONFIRMED / EXPIRED
	IdemKey    string
	CreatedAt  time.Time
	ConfirmedAt *time.Time
}
