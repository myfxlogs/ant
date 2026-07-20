package hdwallet

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/fbsobreira/go-bip39"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deriveAccountXpubFromMnemonic derives the account-level xpub (m/44'/195'/0'/0)
// from a BIP39 mnemonic. Uses the shared hdwallet function.
// The online server never calls this — it receives the xpub via config.
func deriveAccountXpubFromMnemonic(t *testing.T, mnemonic string) string {
	t.Helper()
	seed := bip39.NewSeed(mnemonic, "")
	require.NotEmpty(t, seed)

	xpubStr, _, err := DeriveAccountXpubAndFingerprint(seed)
	require.NoError(t, err)
	return xpubStr
}

// TestXpubDerivationMatchesPrivKey verifies that xpub-only derivation produces
// the same address as private key derivation (BIP44 CKDpub == CKDpriv for non-hardened).
func TestXpubDerivationMatchesPrivKey(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"
	seed := bip39.NewSeed(mnemonic, "")

	xpub := deriveAccountXpubFromMnemonic(t, mnemonic)

	// Derive address at index 0 using xpub (watch-only, no private key)
	xpubAddr0, err := DeriveAddressFromXpub(xpub, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, xpubAddr0)
	assert.True(t, len(xpubAddr0) > 30, "address should be a valid TRON base58 string")

	// Derive address at index 0 using private key (offline path)
	sk0, err := DeriveDepositPrivKey(seed, 0)
	require.NoError(t, err)
	privAddr0 := address.BTCECPrivkeyToAddress(sk0).String()

	// The two must match — this proves xpub derivation is consistent with
	// the offline private key derivation (BIP44 CKDpub == CKDpriv for non-hardened)
	assert.Equal(t, privAddr0, xpubAddr0,
		"xpub-derived address must match privkey-derived address at index 0")

	// Verify index 1 also matches
	xpubAddr1, err := DeriveAddressFromXpub(xpub, 1)
	require.NoError(t, err)
	sk1, err := DeriveDepositPrivKey(seed, 1)
	require.NoError(t, err)
	privAddr1 := address.BTCECPrivkeyToAddress(sk1).String()
	assert.Equal(t, privAddr1, xpubAddr1,
		"xpub-derived address must match privkey-derived address at index 1")
}

// TestXpubFingerprintDeterministic verifies fingerprint is stable.
func TestXpubFingerprintDeterministic(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"
	xpub := deriveAccountXpubFromMnemonic(t, mnemonic)

	fp1, err := XpubFingerprint(xpub)
	require.NoError(t, err)
	assert.Len(t, fp1, 64, "fingerprint should be 64 hex chars (full SHA-256)")

	fp2, err := XpubFingerprint(xpub)
	require.NoError(t, err)
	assert.Equal(t, fp1, fp2, "fingerprint must be deterministic")
}

// TestXpubFingerprintDifferentXpubs verifies different xpubs produce different fingerprints.
func TestXpubFingerprintDifferentXpubs(t *testing.T) {
	mnemonic1 := "test test test test test test test test test test test junk"
	mnemonic2 := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

	xpub1 := deriveAccountXpubFromMnemonic(t, mnemonic1)
	xpub2 := deriveAccountXpubFromMnemonic(t, mnemonic2)

	fp1, err := XpubFingerprint(xpub1)
	require.NoError(t, err)
	fp2, err := XpubFingerprint(xpub2)
	require.NoError(t, err)

	assert.NotEqual(t, fp1, fp2, "different xpubs must have different fingerprints")
}

// TestDeriveAddressFromXpubInvalid verifies error handling.
func TestDeriveAddressFromXpubInvalid(t *testing.T) {
	_, err := DeriveAddressFromXpub("invalid-xpub", 0)
	assert.Error(t, err)
}

// TestValidateXpub verifies xpub validation.
func TestValidateXpub(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"
	xpub := deriveAccountXpubFromMnemonic(t, mnemonic)

	err := ValidateXpub(xpub)
	assert.NoError(t, err)

	err = ValidateXpub("not-a-valid-xpub")
	assert.Error(t, err)
}

// TestXpubDerivationMultipleIndices verifies uniqueness across indices.
func TestXpubDerivationMultipleIndices(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"
	xpub := deriveAccountXpubFromMnemonic(t, mnemonic)

	seen := make(map[string]bool)
	for i := uint32(0); i < 10; i++ {
		addr, err := DeriveAddressFromXpub(xpub, i)
		require.NoError(t, err)
		assert.False(t, seen[addr], "duplicate address at index %d", i)
		seen[addr] = true
	}
}

// TestXpubFingerprintFormat verifies the fingerprint is valid hex.
func TestXpubFingerprintFormat(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"
	xpub := deriveAccountXpubFromMnemonic(t, mnemonic)

	fp, err := XpubFingerprint(xpub)
	require.NoError(t, err)
	_, err = hex.DecodeString(fp)
	assert.NoError(t, err, "fingerprint must be valid hex")
}

// TestXpubFingerprintMatchesSha256 verifies the fingerprint algorithm.
func TestXpubFingerprintMatchesSha256(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"
	xpub := deriveAccountXpubFromMnemonic(t, mnemonic)

	fp, err := XpubFingerprint(xpub)
	require.NoError(t, err)

	h := sha256.Sum256([]byte(xpub))
	expected := hex.EncodeToString(h[:])
	assert.Equal(t, expected, fp)
	assert.Len(t, fp, 64, "fingerprint must be full SHA-256 hex (64 chars)")
}
