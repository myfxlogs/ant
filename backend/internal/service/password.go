package service

import (
	"strings"

	"golang.org/x/crypto/bcrypt"

	"anttrader/internal/pkg/hash"
)

// VerifyPassword verifies a password against a stored hash, auto-detecting the format.
// Supports argon2id (preferred) and bcrypt (backward compat for legacy users).
func VerifyPassword(storedHash, password string) bool {
	if strings.HasPrefix(storedHash, "$argon2id$") {
		valid, err := hash.VerifyPassword(password, storedHash)
		return err == nil && valid
	}
	// bcrypt hashes start with "$2a$", "$2b$", or "$2y$"
	if strings.HasPrefix(storedHash, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
	}
	return false
}
