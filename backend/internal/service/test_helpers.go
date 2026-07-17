package service

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"testing"

	"alphaforge/internal/secrets"
)

// NewTestSecretsClient creates a secrets.Client with a random key for integration tests.
func NewTestSecretsClient(t *testing.T) secrets.Client {
	t.Helper()
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	client, err := secrets.New(base64.StdEncoding.EncodeToString(key), 1)
	if err != nil {
		t.Fatalf("create test secrets client: %v", err)
	}
	return client
}
