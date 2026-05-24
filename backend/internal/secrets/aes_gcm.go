package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"

	"crypto/sha256"
	"golang.org/x/crypto/hkdf"
)

// New creates a new AES-GCM Client from a base64-encoded master key.
func New(masterB64 string, currentVersion uint8) (Client, error) {
	if currentVersion < 1 {
		return nil, &SecretError{Msg: "currentVersion must be >= 1"}
	}
	kek0, err := base64.StdEncoding.DecodeString(masterB64)
	if err != nil {
		return nil, fmt.Errorf("secrets: decode: %w", err)
	}
	if len(kek0) != 32 {
		return nil, &SecretError{Msg: fmt.Sprintf("master key must be 32 bytes, got %d", len(kek0))}
	}
	return &aesGCMClient{kek0: kek0, currentVersion: currentVersion, cache: make(map[string]cipher.AEAD)}, nil
}

// GenerateMasterKey creates a random 32-byte key and returns it base64-encoded.
func GenerateMasterKey() (string, error) {
	key := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, key) // rand.Reader is infallible on Linux
	return base64.StdEncoding.EncodeToString(key), nil
}

type aesGCMClient struct {
	kek0           []byte
	currentVersion uint8
	mu             sync.RWMutex
	cache          map[string]cipher.AEAD
}

func (c *aesGCMClient) CurrentVersion() uint8 { return c.currentVersion }

func (c *aesGCMClient) getAEAD(version uint8, purpose Purpose) (cipher.AEAD, error) {
	key := fmt.Sprintf("v%d/%s", version, purpose)

	c.mu.RLock()
	if a, ok := c.cache[key]; ok {
		c.mu.RUnlock()
		return a, nil
	}
	c.mu.RUnlock()

	info := fmt.Sprintf("ant/v%d/%s", version, purpose)
	r := hkdf.New(sha256.New, c.kek0, nil, []byte(info))
	kek := make([]byte, 32)
	if _, err := io.ReadFull(r, kek); err != nil {
		return nil, fmt.Errorf("secrets: hkdf: %w", err)
	}
	blk, _ := aes.NewCipher(kek)
	aead, _ := cipher.NewGCM(blk)

	c.mu.Lock()
	if a, ok := c.cache[key]; ok {
		c.mu.Unlock()
		return a, nil
	}
	c.cache[key] = aead
	c.mu.Unlock()
	return aead, nil
}

func (c *aesGCMClient) Encrypt(_ context.Context, purpose Purpose, plaintext []byte) ([]byte, error) {
	aead, err := c.getAEAD(c.currentVersion, purpose)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("secrets: nonce: %w", err)
	}
	out := make([]byte, 1+len(nonce))
	out[0] = c.currentVersion
	copy(out[1:], nonce)
	return aead.Seal(out, nonce, plaintext, nil), nil
}

func (c *aesGCMClient) Decrypt(_ context.Context, purpose Purpose, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 1+12+16 {
		return nil, &SecretError{Msg: "ciphertext too short"}
	}
	version := ciphertext[0]
	if version < 1 || version > c.currentVersion {
		return nil, ErrUnknownKeyVersion
	}
	aead, err := c.getAEAD(version, purpose)
	if err != nil {
		return nil, err
	}
	nonce := ciphertext[1 : 1+aead.NonceSize()]
	raw, err := aead.Open(nil, nonce, ciphertext[1+aead.NonceSize():], nil)
	if err != nil {
		return nil, &SecretError{Msg: "decrypt: " + err.Error()}
	}
	return raw, nil
}

func (c *aesGCMClient) Reencrypt(ctx context.Context, purpose Purpose, ciphertext []byte) ([]byte, error) {
	raw, err := c.Decrypt(ctx, purpose, ciphertext)
	if err != nil {
		return nil, err
	}
	return c.Encrypt(ctx, purpose, raw)
}
