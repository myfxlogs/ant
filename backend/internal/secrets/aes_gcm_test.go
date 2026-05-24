package secrets_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"anttrader/internal/secrets"
)

func newClient(t *testing.T, version uint8) secrets.Client {
	t.Helper()
	key, err := secrets.GenerateMasterKey()
	require.NoError(t, err)
	c, err := secrets.New(key, version)
	require.NoError(t, err)
	return c
}

func TestNew_InvalidVersion(t *testing.T) {
	_, err := secrets.New("AAAA", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be >= 1")
}

func TestNew_InvalidKeyLength(t *testing.T) {
	_, err := secrets.New(base64.StdEncoding.EncodeToString([]byte("short")), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be 32 bytes")
}

func TestNew_InvalidBase64(t *testing.T) {
	_, err := secrets.New("not-valid-base64!!!", 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestRoundTrip(t *testing.T) {
	c := newClient(t, 1)
	plain := []byte("my-secret-password-123!")

	ct, err := c.Encrypt(t.Context(), secrets.PurposeMTPassword, plain)
	require.NoError(t, err)
	assert.Equal(t, uint8(1), ct[0], "version byte must be 1")

	got, err := c.Decrypt(t.Context(), secrets.PurposeMTPassword, ct)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

func TestDecryptWithDifferentPurpose_Fails(t *testing.T) {
	c := newClient(t, 1)
	plain := []byte("top-secret")

	ct, err := c.Encrypt(t.Context(), secrets.PurposeMTPassword, plain)
	require.NoError(t, err)

	_, err = c.Decrypt(t.Context(), secrets.PurposeMTAPIToken, ct)
	assert.Error(t, err, "different purpose must fail decryption")
}

func TestDecryptCorruptedTag_Fails(t *testing.T) {
	c := newClient(t, 1)
	plain := []byte("data")

	ct, err := c.Encrypt(t.Context(), secrets.PurposeMTPassword, plain)
	require.NoError(t, err)

	// Corrupt the last byte (part of the GCM tag)
	ct[len(ct)-1] ^= 0xFF
	_, err = c.Decrypt(t.Context(), secrets.PurposeMTPassword, ct)
	assert.Error(t, err, "corrupted ciphertext must fail")
}

func TestDecryptTooShort(t *testing.T) {
	c := newClient(t, 1)
	_, err := c.Decrypt(t.Context(), secrets.PurposeMTPassword, []byte{1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestDecryptUnknownVersion(t *testing.T) {
	c := newClient(t, 1)
	plain := []byte("data")

	ct, err := c.Encrypt(t.Context(), secrets.PurposeMTPassword, plain)
	require.NoError(t, err)

	// Tamper version byte to 99
	ct[0] = 99
	_, err = c.Decrypt(t.Context(), secrets.PurposeMTPassword, ct)
	assert.ErrorIs(t, err, secrets.ErrUnknownKeyVersion)
}

func TestOldVersionDecrypt(t *testing.T) {
	// Use a fixed key so we can create two clients sharing the same KEK0
	key, err := secrets.GenerateMasterKey()
	require.NoError(t, err)

	// Create version 1 client and encrypt
	c1, err := secrets.New(key, 1)
	require.NoError(t, err)
	plain := []byte("old-version-data")
	ct, err := c1.Encrypt(t.Context(), secrets.PurposeMTPassword, plain)
	require.NoError(t, err)

	// Create version 2 client with the same key and decrypt v1 ciphertext
	c2, err := secrets.New(key, 2)
	require.NoError(t, err)
	got, err := c2.Decrypt(t.Context(), secrets.PurposeMTPassword, ct)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

func TestNonceUnique(t *testing.T) {
	c := newClient(t, 1)
	plain := []byte("test")
	nonces := make(map[string]struct{})

	for i := 0; i < 1000; i++ {
		ct, err := c.Encrypt(t.Context(), secrets.PurposeMTPassword, plain)
		require.NoError(t, err)
		// nonce is at bytes 1-12 (GCM standard 12-byte nonce)
		nonce := string(ct[1:13])
		if _, exists := nonces[nonce]; exists {
			t.Fatalf("duplicate nonce at iteration %d", i)
		}
		nonces[nonce] = struct{}{}
	}
}

func TestReencrypt(t *testing.T) {
	c := newClient(t, 1)
	plain := []byte("reencrypt-me")

	ct, err := c.Encrypt(t.Context(), secrets.PurposeMTPassword, plain)
	require.NoError(t, err)

	// Reencrypt (same version, round-trip)
	ct2, err := c.Reencrypt(t.Context(), secrets.PurposeMTPassword, ct)
	require.NoError(t, err)

	got, err := c.Decrypt(t.Context(), secrets.PurposeMTPassword, ct2)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

func TestCurrentVersion(t *testing.T) {
	c := newClient(t, 5)
	assert.Equal(t, uint8(5), c.CurrentVersion())

	c2 := newClient(t, 1)
	assert.Equal(t, uint8(1), c2.CurrentVersion())
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	c := newClient(t, 1)
	ct, err := c.Encrypt(t.Context(), secrets.PurposeMTPassword, []byte{})
	require.NoError(t, err)
	assert.NotEmpty(t, ct)

	got, err := c.Decrypt(t.Context(), secrets.PurposeMTPassword, ct)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestDecrypt_VersionZero(t *testing.T) {
	c := newClient(t, 1)
	ct := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	_, err := c.Decrypt(t.Context(), secrets.PurposeMTPassword, ct)
	assert.Error(t, err)
}

func TestReencrypt_OldVersionToLatest(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	require.NoError(t, err)

	c1, err := secrets.New(key, 1)
	require.NoError(t, err)
	plain := []byte("migrate-me")

	ct, err := c1.Encrypt(t.Context(), secrets.PurposeMTPassword, plain)
	require.NoError(t, err)
	assert.Equal(t, uint8(1), ct[0])

	// v2 client reencrypts v1 ciphertext
	c2, err := secrets.New(key, 2)
	require.NoError(t, err)
	ct2, err := c2.Reencrypt(t.Context(), secrets.PurposeMTPassword, ct)
	require.NoError(t, err)
	assert.Equal(t, uint8(2), ct2[0])

	// Both decrypt to same plaintext
	got, err := c2.Decrypt(t.Context(), secrets.PurposeMTPassword, ct2)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

func TestDeriveKey_CacheHit(t *testing.T) {
	// Encrypt twice with same purpose — second call hits cache
	c := newClient(t, 1)
	plain := []byte("cache-test")

	ct1, err := c.Encrypt(t.Context(), secrets.PurposeMTPassword, plain)
	require.NoError(t, err)

	ct2, err := c.Encrypt(t.Context(), secrets.PurposeMTPassword, plain)
	require.NoError(t, err)

	// Both should decrypt successfully
	got1, err := c.Decrypt(t.Context(), secrets.PurposeMTPassword, ct1)
	require.NoError(t, err)
	assert.Equal(t, plain, got1)

	got2, err := c.Decrypt(t.Context(), secrets.PurposeMTPassword, ct2)
	require.NoError(t, err)
	assert.Equal(t, plain, got2)
}

func TestDecrypt_VersionTooHigh(t *testing.T) {
	c := newClient(t, 1)
	plain := []byte("data")
	ct, err := c.Encrypt(t.Context(), secrets.PurposeMTPassword, plain)
	require.NoError(t, err)
	// Set version byte to 99 (> currentVersion 1)
	ct[0] = 99
	_, err = c.Decrypt(t.Context(), secrets.PurposeMTPassword, ct)
	assert.ErrorIs(t, err, secrets.ErrUnknownKeyVersion)
}

func TestReencrypt_InvalidCiphertext(t *testing.T) {
	c := newClient(t, 1)
	_, err := c.Reencrypt(t.Context(), secrets.PurposeMTPassword, []byte("bad"))
	assert.Error(t, err)
}

func TestDeriveKey_Concurrent(t *testing.T) {
	// Trigger concurrent access to exercise cache double-check path
	c := newClient(t, 1)
	plain := []byte("concurrent")
	done := make(chan struct{})
	for range 10 {
		go func() {
			_, _ = c.Encrypt(t.Context(), secrets.PurposeMTPassword, plain)
			done <- struct{}{}
		}()
	}
	for range 10 {
		<-done
	}
}

func TestGenerateMasterKey(t *testing.T) {
	for i := 0; i < 10; i++ {
		k, err := secrets.GenerateMasterKey()
		require.NoError(t, err)
		b, err := base64.StdEncoding.DecodeString(k)
		require.NoError(t, err)
		assert.Len(t, b, 32)
	}
}
