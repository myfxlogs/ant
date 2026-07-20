package hdwallet

import (
"testing"

"github.com/fbsobreira/go-bip39"
"github.com/fbsobreira/gotron-sdk/pkg/address"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

// TestGenerateMnemonic verifies that GenerateMnemonic produces a valid 24-word mnemonic.
func TestGenerateMnemonic(t *testing.T) {
m, err := GenerateMnemonic()
require.NoError(t, err)
assert.True(t, bip39.IsMnemonicValid(m), "generated mnemonic must be valid")
}

// TestDeriveDepositPrivKeyDeterminism verifies the same seed + index always produces the same private key.
func TestDeriveDepositPrivKeyDeterminism(t *testing.T) {
seed := bip39.NewSeed("test test test test test test test test test test test junk", "")

sk1, err := DeriveDepositPrivKey(seed, 0)
require.NoError(t, err)
sk2, err := DeriveDepositPrivKey(seed, 0)
require.NoError(t, err)
assert.Equal(t, sk1.Serialize(), sk2.Serialize())
}

// TestDeriveDepositPrivKeyUnique verifies different indices produce different keys.
func TestDeriveDepositPrivKeyUnique(t *testing.T) {
seed := bip39.NewSeed("test test test test test test test test test test test junk", "")

sk0, err := DeriveDepositPrivKey(seed, 0)
require.NoError(t, err)
sk1, err := DeriveDepositPrivKey(seed, 1)
require.NoError(t, err)
assert.NotEqual(t, sk0.Serialize(), sk1.Serialize())
}

// TestDeriveDepositPrivKeyAddress verifies the derived private key produces a valid TRON address.
func TestDeriveDepositPrivKeyAddress(t *testing.T) {
seed := bip39.NewSeed("test test test test test test test test test test test junk", "")

sk, err := DeriveDepositPrivKey(seed, 0)
require.NoError(t, err)
addr := address.BTCECPrivkeyToAddress(sk).String()
assert.NotEmpty(t, addr)
assert.True(t, len(addr) > 30, "address should be a valid TRON base58 string")
}

// TestDeriveEnergyAccountPrivKey verifies energy account key derivation.
func TestDeriveEnergyAccountPrivKey(t *testing.T) {
seed := bip39.NewSeed("test test test test test test test test test test test junk", "")

sk, err := DeriveEnergyAccountPrivKey(seed)
require.NoError(t, err)
addr := address.BTCECPrivkeyToAddress(sk).String()
assert.NotEmpty(t, addr)
assert.True(t, len(addr) > 30, "address should be a valid TRON base58 string")
}

// TestDeriveEnergyAccountPrivKeyDeterminism verifies energy key is deterministic.
func TestDeriveEnergyAccountPrivKeyDeterminism(t *testing.T) {
seed := bip39.NewSeed("test test test test test test test test test test test junk", "")

sk1, err := DeriveEnergyAccountPrivKey(seed)
require.NoError(t, err)
sk2, err := DeriveEnergyAccountPrivKey(seed)
require.NoError(t, err)
assert.Equal(t, sk1.Serialize(), sk2.Serialize())
}

// TestDeriveDepositPrivKeyDifferentFromEnergy verifies deposit and energy keys are different.
func TestDeriveDepositPrivKeyDifferentFromEnergy(t *testing.T) {
seed := bip39.NewSeed("test test test test test test test test test test test junk", "")

depositSk, err := DeriveDepositPrivKey(seed, 0)
require.NoError(t, err)
energySk, err := DeriveEnergyAccountPrivKey(seed)
require.NoError(t, err)
assert.NotEqual(t, depositSk.Serialize(), energySk.Serialize())
}
