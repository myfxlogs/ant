package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrAPIKeyNotFound  = errors.New("api key not found")
	ErrAPIKeyRevoked   = errors.New("api key revoked")
	ErrAPIKeyExpired   = errors.New("api key expired")
)

type APIKey struct {
	ID         uuid.UUID   `db:"id"`
	UserID     uuid.UUID   `db:"user_id"`
	Name       string      `db:"name"`
	KeyHash    string      `db:"key_hash"`
	Scopes     []string    `db:"scopes"`
	ExpiresAt  *time.Time  `db:"expires_at"`
	RevokedAt  *time.Time  `db:"revoked_at"`
	LastUsedAt *time.Time  `db:"last_used_at"`
	CreatedAt  time.Time   `db:"created_at"`
}

type APIKeyRepository struct {
	db *sqlx.DB
}

func NewAPIKeyRepository(db *sqlx.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, key *APIKey) error {
	query := `
		INSERT INTO api_keys (
			id, user_id, name, key_hash, scopes, expires_at, revoked_at, last_used_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	if key.ID == uuid.Nil {
		key.ID = uuid.New()
	}
	if key.CreatedAt.IsZero() {
		key.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		key.ID,
		key.UserID,
		key.Name,
		key.KeyHash,
		key.Scopes,
		key.ExpiresAt,
		key.RevokedAt,
		key.LastUsedAt,
		key.CreatedAt,
	)
	return err
}

func (r *APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	query := `SELECT * FROM api_keys WHERE key_hash = $1`
	var key APIKey
	if err := r.db.GetContext(ctx, &key, query, keyHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}
	return &key, nil
}

func (r *APIKeyRepository) MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error {
	query := `UPDATE api_keys SET last_used_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, usedAt)
	return err
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) error {
	query := `UPDATE api_keys SET revoked_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, revokedAt)
	return err
}
