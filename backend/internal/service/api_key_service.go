package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/repository"
)

type APIKeyService struct {
	repo *repository.APIKeyRepository
}

func NewAPIKeyService(repo *repository.APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

// Create issues a new API key and returns the raw key once.
// Store ONLY the hash in database.
func (s *APIKeyService) Create(ctx context.Context, userID uuid.UUID, name string, scopes []string, expiresAt *time.Time) (rawKey string, keyID uuid.UUID, err error) {
	if err := ValidateSandboxScopes(name, scopes); err != nil {
		return "", uuid.Nil, err
	}

	rawKey, err = generateRawAPIKey()
	if err != nil {
		return "", uuid.Nil, err
	}

	h := hashAPIKey(rawKey)

	k := &repository.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		KeyHash:   h,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, k); err != nil {
		return "", uuid.Nil, err
	}

	return rawKey, k.ID, nil
}

// Validate returns user id if key is valid.
func (s *APIKeyService) Validate(ctx context.Context, rawKey string) (uuid.UUID, []string, error) {
	if rawKey == "" {
		return uuid.Nil, nil, repository.ErrAPIKeyNotFound
	}

	h := hashAPIKey(rawKey)
	k, err := s.repo.GetByHash(ctx, h)
	if err != nil {
		return uuid.Nil, nil, err
	}

	if k.RevokedAt != nil {
		return uuid.Nil, nil, repository.ErrAPIKeyRevoked
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return uuid.Nil, nil, repository.ErrAPIKeyExpired
	}

	_ = s.repo.MarkUsed(ctx, k.ID, time.Now())
	return k.UserID, k.Scopes, nil
}

func generateRawAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashAPIKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

var ErrInvalidAPIKey = errors.New("invalid api key")

var ErrSandboxForbiddenScope = errors.New("sandbox key must not have trade:write or account:write")

var sandboxForbiddenScopes = []string{"trade:write", "account:write"}

func ValidateSandboxScopes(name string, scopes []string) error {
	if !strings.Contains(strings.ToLower(name), "sandbox") {
		return nil
	}
	for _, s := range scopes {
		for _, f := range sandboxForbiddenScopes {
			if strings.EqualFold(s, f) {
				return ErrSandboxForbiddenScope
			}
		}
	}
	return nil
}
