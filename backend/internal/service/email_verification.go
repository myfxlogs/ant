package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/notifier"
)

// EmailVerificationService handles email verification token generation, storage, and validation.
type EmailVerificationService struct {
	pg       *pgxpool.Pool
	email    *notifier.EmailNotifier
	appURL   string
	log      *zap.Logger
}

// NewEmailVerificationService creates a new EmailVerificationService.
// appURL is the base URL for verification links (e.g. https://alfq.org).
func NewEmailVerificationService(pg *pgxpool.Pool, email *notifier.EmailNotifier, appURL string, log *zap.Logger) *EmailVerificationService {
	return &EmailVerificationService{pg: pg, email: email, appURL: appURL, log: log}
}

// GenerateAndSend creates a verification token for the user and sends the verification email.
// Returns nil if SMTP is not configured (verification is optional in that case).
func (s *EmailVerificationService) GenerateAndSend(ctx context.Context, userID uuid.UUID, userEmail string) error {
	if s.email == nil {
		s.log.Debug("email verification skipped — SMTP not configured", zap.String("userID", userID.String()))
		return nil
	}

	token, err := s.generateToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("email verification: generate token: %w", err)
	}

	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.appURL, url.QueryEscape(token))
	subject := "Verify your email — AlphaForge"
	body := fmt.Sprintf(
		"Welcome to AlphaForge!\n\n"+
			"Please verify your email address by clicking the link below:\n\n"+
			"%s\n\n"+
			"This link expires in 24 hours.\n\n"+
			"If you did not create an account, you can safely ignore this email.",
		verifyURL,
	)

	if err := s.email.SendTo(userEmail, subject, body); err != nil {
		s.log.Warn("email verification: send failed", zap.String("userID", userID.String()), zap.Error(err))
		return err
	}
	s.log.Info("verification email sent", zap.String("userID", userID.String()), zap.String("email", userEmail))
	return nil
}

// VerifyToken validates a verification token and marks the user's email as verified.
// Returns the userID on success.
func (s *EmailVerificationService) VerifyToken(ctx context.Context, token string) (uuid.UUID, error) {
	tokenHash := hashToken(token)

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("email verification: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID uuid.UUID
	var usedAt *time.Time
	err = tx.QueryRow(ctx,
		`SELECT user_id, used_at FROM email_verification_tokens
		 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()`,
		tokenHash,
	).Scan(&userID, &usedAt)
	if err != nil {
		return uuid.Nil, fmt.Errorf("email verification: invalid or expired token")
	}

	_, err = tx.Exec(ctx,
		`UPDATE email_verification_tokens SET used_at = now() WHERE token_hash = $1`,
		tokenHash,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("email verification: mark token used: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE users SET email_verified_at = now(), updated_at = now() WHERE id = $1`,
		userID,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("email verification: set verified: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("email verification: commit: %w", err)
	}

	return userID, nil
}

func (s *EmailVerificationService) generateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate random: %w", err)
	}
	token := hex.EncodeToString(raw)
	tokenHash := hashToken(token)

	_, err := s.pg.Exec(ctx,
		`INSERT INTO email_verification_tokens (user_id, token_hash)
		 VALUES ($1, $2)`,
		userID, tokenHash,
	)
	if err != nil {
		return "", fmt.Errorf("insert token: %w", err)
	}
	return token, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
