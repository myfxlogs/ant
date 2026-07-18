package hdwallet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeriveFromMnemonic verifies that a known mnemonic produces deterministic addresses.
func TestDeriveFromMnemonic(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"

	addr0, err := DeriveFromMnemonic(mnemonic, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, addr0.Address)
	assert.Len(t, addr0.PrivateKey, 32)
	assert.Equal(t, 0, addr0.Index)

	addr1, err := DeriveFromMnemonic(mnemonic, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, addr1.Address)
	assert.NotEqual(t, addr0.Address, addr1.Address)
	assert.Equal(t, 1, addr1.Index)
}

// TestDeriveDeterminism verifies the same mnemonic + index always produces the same address.
func TestDeriveDeterminism(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"

	a1, err := DeriveFromMnemonic(mnemonic, 5)
	require.NoError(t, err)

	a2, err := DeriveFromMnemonic(mnemonic, 5)
	require.NoError(t, err)

	assert.Equal(t, a1.Address, a2.Address)
	assert.Equal(t, a1.PrivateKey, a2.PrivateKey)
}

// TestDeriveBatch verifies batch derivation.
func TestDeriveBatch(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"

	addrs, err := DeriveBatch(mnemonic, 0, 5)
	require.NoError(t, err)
	assert.Len(t, addrs, 5)

	seen := make(map[string]bool)
	for i, a := range addrs {
		assert.False(t, seen[a.Address], "duplicate address at index %d", i)
		seen[a.Address] = true
		assert.Equal(t, i, a.Index)
	}
}

// TestAddressFromPrivateKey verifies address derivation from private key matches.
func TestAddressFromPrivateKey(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"

	derived, err := DeriveFromMnemonic(mnemonic, 0)
	require.NoError(t, err)

	addr, err := AddressFromPrivateKey(derived.PrivateKey)
	require.NoError(t, err)
	assert.Equal(t, derived.Address, addr)
}

// TestInvalidIndex verifies error handling for out-of-range index.
func TestInvalidIndex(t *testing.T) {
	_, err := DeriveFromMnemonic("test test test test test test test test test test test junk", -1)
	assert.Error(t, err)
}
