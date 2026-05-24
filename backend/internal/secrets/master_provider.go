package secrets

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
)

// MasterProvider supplies the master encryption key (KEK).
// Supports multiple sources: env var, file, KMS (future).
type MasterProvider interface {
	// MasterKey returns the current master key bytes (32B for AES-256).
	MasterKey(ctx context.Context) ([]byte, error)

	// Rotate generates a new master key version and returns it.
	// The old key is retained for decryption of previously-encrypted data.
	Rotate(ctx context.Context) (newVersion uint8, newKey []byte, err error)
}

// EnvMasterKey reads ANT_MASTER_KEY from environment.
type EnvMasterKey struct{}

func (EnvMasterKey) MasterKey(_ context.Context) ([]byte, error) {
	return decodeMasterKey(os.Getenv("ANT_MASTER_KEY"))
}

func (EnvMasterKey) Rotate(_ context.Context) (uint8, []byte, error) {
	return 0, nil, &SecretError{Msg: "rotate not supported for env-backed master key; use file or KMS"}
}

// FileMasterKey reads ANT_MASTER_KEY_FILE from a file path.
type FileMasterKey struct{}

func (FileMasterKey) MasterKey(_ context.Context) ([]byte, error) {
	path := os.Getenv("ANT_MASTER_KEY_FILE")
	if path == "" {
		return nil, &SecretError{Msg: "ANT_MASTER_KEY_FILE not set"}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &SecretError{Msg: "read master key file: " + err.Error()}
	}
	return decodeMasterKey(string(data))
}

func (FileMasterKey) Rotate(_ context.Context) (uint8, []byte, error) {
	return 0, nil, &SecretError{Msg: "rotate not supported for file-backed master key; use KMS"}
}

func decodeMasterKey(s string) ([]byte, error) {
	if s == "" {
		return nil, &SecretError{Msg: "master key is empty"}
	}
	// Try base64 decode first (32B → 44 base64 chars).
	if len(s) == 44 {
		raw, err := decodeBase64(s)
		if err == nil {
			return raw, nil
		}
	}
	return []byte(s), nil
}

func decodeBase64(s string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	return b, nil
}
