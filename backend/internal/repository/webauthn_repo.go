package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"alphaforge/internal/model"
)

// WebAuthnRepository manages WebAuthn credentials, withdrawal requests, and whitelist.
type WebAuthnRepository struct {
	db DBTX
}

func NewWebAuthnRepository(db DBTX) *WebAuthnRepository {
	return &WebAuthnRepository{db: db}
}

// CreateCredential stores a new WebAuthn credential.
func (r *WebAuthnRepository) CreateCredential(ctx context.Context, c *model.WebAuthnCredential) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO webauthn_credentials (user_id, credential_id, public_key, attestation_type, aaguid, sign_count, transports, name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, c.UserID, c.CredentialID, c.PublicKey, c.AttestationType, c.AAGUID, c.SignCount, c.Transports, c.Name)
	if err != nil {
		return fmt.Errorf("webauthn repo: create credential: %w", err)
	}
	return nil
}

// GetCredential retrieves a credential by credential_id.
func (r *WebAuthnRepository) GetCredential(ctx context.Context, credentialID string) (*model.WebAuthnCredential, error) {
	var c model.WebAuthnCredential
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, credential_id, public_key, attestation_type, aaguid, sign_count, transports, name, created_at, updated_at
		FROM webauthn_credentials WHERE credential_id = $1
	`, credentialID).Scan(
		&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey, &c.AttestationType, &c.AAGUID,
		&c.SignCount, &c.Transports, &c.Name, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("webauthn repo: get credential: %w", err)
	}
	return &c, nil
}

// ListCredentialsByUser returns all credentials for a user.
func (r *WebAuthnRepository) ListCredentialsByUser(ctx context.Context, userID uuid.UUID) ([]model.WebAuthnCredential, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, credential_id, public_key, attestation_type, aaguid, sign_count, transports, name, created_at, updated_at
		FROM webauthn_credentials WHERE user_id = $1 ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("webauthn repo: list credentials: %w", err)
	}
	defer rows.Close()

	var out []model.WebAuthnCredential
	for rows.Next() {
		var c model.WebAuthnCredential
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey, &c.AttestationType, &c.AAGUID,
			&c.SignCount, &c.Transports, &c.Name, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("webauthn repo: scan credential: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateSignCount updates the sign count for clone detection.
func (r *WebAuthnRepository) UpdateSignCount(ctx context.Context, credentialID string, signCount int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE webauthn_credentials SET sign_count = $2, updated_at = NOW() WHERE credential_id = $1
	`, credentialID, signCount)
	if err != nil {
		return fmt.Errorf("webauthn repo: update sign count: %w", err)
	}
	return nil
}

// DeleteCredential removes a credential.
func (r *WebAuthnRepository) DeleteCredential(ctx context.Context, userID uuid.UUID, credentialID string) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM webauthn_credentials WHERE user_id = $1 AND credential_id = $2
	`, userID, credentialID)
	if err != nil {
		return fmt.Errorf("webauthn repo: delete credential: %w", err)
	}
	return nil
}

// ListAllCredentials returns all credentials for coldsign USB export (Q2).
func (r *WebAuthnRepository) ListAllCredentials(ctx context.Context) ([]model.WebAuthnCredential, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, credential_id, public_key, attestation_type, aaguid, sign_count, transports, name, created_at, updated_at
		FROM webauthn_credentials ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("webauthn repo: list all credentials: %w", err)
	}
	defer rows.Close()

	var out []model.WebAuthnCredential
	for rows.Next() {
		var c model.WebAuthnCredential
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey, &c.AttestationType, &c.AAGUID,
			&c.SignCount, &c.Transports, &c.Name, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("webauthn repo: scan all credentials: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
