// Package secrets envelope encryption (M10 ADR-0011 §2.2).
//
// Format v2 (envelope): version(1) + dek_kid(1) + nonce(12) + wrapped_dek(48) + nonce(12) + ciphertext+tag
//   - DEK = random 32B AES-256 key, wrapped with KEK via AES-256-GCM
//   - dek_kid identifies which KEK version wrapped the DEK
//
// Format v1 (legacy, see vault_legacy.go): version(1) + nonce(12) + AES-GCM(plaintext, key=KEK)
package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
)

// EnvelopeClient wraps an existing Client with envelope encryption (DEK wrapping).
// It implements Client by delegating to the underlying client for KEK operations,
// then adds a DEK layer between KEK and plaintext.
type EnvelopeClient struct {
	inner Client
}

// NewEnvelopeClient creates an envelope-encryption wrapper.
func NewEnvelopeClient(inner Client) *EnvelopeClient {
	return &EnvelopeClient{inner: inner}
}

// Encrypt generates a random DEK, encrypts plaintext with DEK, then wraps DEK with KEK.
// Output: version(1) + dek_kid(1) + nonce_wrap(12) + wrapped_dek(32+16) + nonce_data(12) + ciphertext+tag
func (c *EnvelopeClient) Encrypt(ctx context.Context, purpose Purpose, plaintext []byte) ([]byte, error) {
	// 1. Generate random DEK (32B).
	dek := make([]byte, 32)
	if _, err := io.ReadFull(cryptoRand, dek); err != nil {
		return nil, fmt.Errorf("envelope: generate dek: %w", err)
	}

	// 2. Wrap DEK with KEK (via inner client).
	wrapped, err := c.inner.Encrypt(ctx, purpose, dek)
	if err != nil {
		return nil, fmt.Errorf("envelope: wrap dek: %w", err)
	}
	// wrapped = version(1) + nonce(12) + AES-GCM(dek, key=KEK)
	// len = 1 + 12 + 32 + 16 = 61

	// 3. Encrypt plaintext with DEK.
	dataCipher, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("envelope: aes cipher: %w", err)
	}
	dataAEAD, err := cipher.NewGCM(dataCipher)
	if err != nil {
		return nil, fmt.Errorf("envelope: gcm: %w", err)
	}
	dataNonce := make([]byte, dataAEAD.NonceSize()) // 12B
	if _, err := io.ReadFull(cryptoRand, dataNonce); err != nil {
		return nil, fmt.Errorf("envelope: data nonce: %w", err)
	}

	encrypted := dataAEAD.Seal(nil, dataNonce, plaintext, nil) // ciphertext + 16B tag

	// 4. Assemble output.
	dekKid := c.inner.CurrentVersion()
	out := make([]byte, 0, 2+12+len(wrapped)+12+len(encrypted))
	out = append(out, 0x02)        // envelope format marker
	out = append(out, dekKid)       // which KEK wrapped the DEK
	out = append(out, dataNonce...) // nonce for data decryption
	out = append(out, wrapped...)   // version(1)+nonce(12)+AES-GCM(dek)
	out = append(out, encrypted...) // AES-GCM(plaintext, key=DEK)

	return out, nil
}

// Decrypt detects format and decrypts accordingly.
// Envelope format (v2 marker 0x02): unwrap DEK → decrypt data with DEK.
// Legacy format: delegates to vault_legacy.go.
func (c *EnvelopeClient) Decrypt(ctx context.Context, purpose Purpose, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 1+12+16 {
		return nil, &SecretError{Msg: "ciphertext too short"}
	}

	// Detect format by first byte.
	if ciphertext[0] == 0x02 {
		return c.decryptV2(ctx, purpose, ciphertext)
	}

	// Legacy format — delegate to inner client.
	return c.inner.Decrypt(ctx, purpose, ciphertext)
}

func (c *EnvelopeClient) decryptV2(ctx context.Context, purpose Purpose, data []byte) ([]byte, error) {
	if len(data) < 2+12 {
		return nil, &SecretError{Msg: "envelope v2: ciphertext too short"}
	}

	pos := 2                        // skip version(1)+dek_kid(1)
	dataNonce := data[pos : pos+12] // 12B nonce for data
	pos += 12

	// wrapped starts at pos and ends where data was encrypted
	// wrapped = version(1)+nonce(12)+AES-GCM(dek) = 61 bytes
	wrappedEnd := pos + 1 + 12 + 32 + 16 // 61
	if wrappedEnd > len(data) {
		return nil, &SecretError{Msg: "envelope v2: wrapped dek truncated"}
	}
	wrapped := data[pos:wrappedEnd]
	encrypted := data[wrappedEnd:]

	// Unwrap DEK.
	dek, err := c.inner.Decrypt(ctx, purpose, wrapped)
	if err != nil {
		return nil, fmt.Errorf("envelope: unwrap dek: %w", err)
	}
	if len(dek) != 32 {
		return nil, &SecretError{Msg: "envelope: unwrapped dek wrong size"}
	}

	// Decrypt data with DEK.
	dataCipher, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("envelope: aes cipher: %w", err)
	}
	dataAEAD, err := cipher.NewGCM(dataCipher)
	if err != nil {
		return nil, fmt.Errorf("envelope: gcm: %w", err)
	}

	plaintext, err := dataAEAD.Open(nil, dataNonce, encrypted, nil)
	if err != nil {
		return nil, &SecretError{Msg: "envelope decrypt: " + err.Error()}
	}
	return plaintext, nil
}

// Reencrypt decrypts then re-encrypts with latest KEK/DEK.
func (c *EnvelopeClient) Reencrypt(ctx context.Context, purpose Purpose, ciphertext []byte) ([]byte, error) {
	raw, err := c.Decrypt(ctx, purpose, ciphertext)
	if err != nil {
		return nil, err
	}
	return c.Encrypt(ctx, purpose, raw)
}

// CurrentVersion delegates to the inner client.
func (c *EnvelopeClient) CurrentVersion() uint8 {
	return c.inner.CurrentVersion()
}
