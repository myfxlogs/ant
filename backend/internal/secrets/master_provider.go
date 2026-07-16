package secrets

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
// L-3: rotation writes new keys to ANT_KEY_DIR (default /var/lib/ant/keys)
// so operators can deploy the new key into the env var on restart.
type EnvMasterKey struct{}

func (EnvMasterKey) MasterKey(_ context.Context) ([]byte, error) {
	// ANT_MASTER_KEY is injected by main() via config.Load().AntMasterKey
	return decodeMasterKey(os.Getenv("ANT_MASTER_KEY"))
}

func (e EnvMasterKey) Rotate(_ context.Context) (uint8, []byte, error) {
	// 1. Read current key to determine current version.
	currentKey, err := e.MasterKey(context.Background())
	if err != nil {
		return 0, nil, fmt.Errorf("rotate: cannot read current key: %w", err)
	}

	// 2. Determine next version by scanning existing key files.
	nextVer := nextKeyVersion(keyDir())

	// 3. Generate new random 32-byte key.
	newKey := make([]byte, 32)
	if _, err := io.ReadFull(cryptoRand, newKey); err != nil {
		return 0, nil, fmt.Errorf("rotate: %w", err)
	}
	_ = currentKey // old key retained in env var for decryption

	// 4. Persist new key to ANT_KEY_DIR.
	dir := keyDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return 0, nil, fmt.Errorf("rotate: mkdir %s: %w", dir, err)
	}
	keyPath := filepath.Join(dir, fmt.Sprintf("ant-master-key-v%d.key", nextVer))
	b64 := base64.StdEncoding.EncodeToString(newKey)
	if err := os.WriteFile(keyPath, []byte(b64+"\n"), 0600); err != nil {
		return 0, nil, fmt.Errorf("rotate: write key file: %w", err)
	}

	return nextVer, newKey, nil
}

// keyDir returns the directory for persisted key versions.
func keyDir() string {
	if d := os.Getenv("ANT_KEY_DIR"); d != "" {
		return d
	}
	return "/var/lib/ant/keys"
}

// nextKeyVersion scans the key directory and returns the next available version.
func nextKeyVersion(dir string) uint8 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 2 // start at version 2 (v1 is the env var)
	}
	maxVer := uint8(1)
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "ant-master-key-v") && strings.HasSuffix(name, ".key") {
			// Extract version number: "ant-master-key-v{N}.key" → N.
			trimmed := strings.TrimPrefix(name, "ant-master-key-v")
			trimmed = strings.TrimSuffix(trimmed, ".key")
			if v, err := strconv.ParseUint(trimmed, 10, 8); err == nil {
				if uint8(v) > maxVer {
					maxVer = uint8(v)
				}
			}
		}
	}
	return maxVer + 1
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
