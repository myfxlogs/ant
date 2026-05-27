package repository

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"anttrader/internal/pkg/hash"

	"github.com/google/uuid"
)

// PasswordResetToken represents a one-time password reset token.
type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	Consumed  bool
	CreatedAt time.Time
}

// PasswordResetRepo manages password reset tokens.
type PasswordResetRepo struct {
	db *pgxpool.Pool
}

func NewPasswordResetRepo(db *pgxpool.Pool) *PasswordResetRepo {
	return &PasswordResetRepo{db: db}
}

// GenerateToken creates a cryptographically random token string (32 bytes, base64).
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CreateResetToken creates a new password reset token valid for 24 hours.
// Returns the plaintext token (only shown to the admin who triggered the reset).
func (r *PasswordResetRepo) CreateResetToken(ctx context.Context, userID uuid.UUID) (string, error) {
	token, err := GenerateToken()
	if err != nil {
		return "", err
	}
	_, err = r.db.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token, expires_at)
		 VALUES ($1, $2, $3)`,
		userID, token, time.Now().Add(24*time.Hour),
	)
	if err != nil {
		return "", fmt.Errorf("create reset token: %w", err)
	}
	return token, nil
}

// ValidateResetToken returns the user ID if the token is valid and unused.
func (r *PasswordResetRepo) ValidateResetToken(ctx context.Context, token string) (uuid.UUID, error) {
	var userID uuid.UUID
	var consumed bool
	var expiresAt time.Time
	err := r.db.QueryRow(ctx,
		`SELECT user_id, consumed, expires_at FROM password_reset_tokens WHERE token = $1`,
		token,
	).Scan(&userID, &consumed, &expiresAt)
	if err != nil {
		return uuid.Nil, fmt.Errorf("validate reset token: %w", err)
	}
	if consumed {
		return uuid.Nil, fmt.Errorf("reset token already used")
	}
	if time.Now().After(expiresAt) {
		return uuid.Nil, fmt.Errorf("reset token expired")
	}
	return userID, nil
}

// ConsumeResetToken marks a token as consumed after successful password reset.
func (r *PasswordResetRepo) ConsumeResetToken(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE password_reset_tokens SET consumed = TRUE WHERE token = $1`,
		token,
	)
	return err
}

// HashPassword hashes a plaintext password using argon2id.
func HashPassword(password string) (string, error) {
	return hash.HashPassword(password)
}
