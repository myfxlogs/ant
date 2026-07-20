// Package hdwallet — derive_priv.go provides private key derivation from a
// BIP39 seed for OFFLINE USE ONLY (cmd/coldsign, cmd/hdgen).
//
// The online server MUST NOT call these functions (ADR-0026 R1).
// The CI zero-private-key assertion excludes the hdwallet package from scanning
// because these functions inherently handle private key material.
package hdwallet

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
)

// deriveAccountExtKey derives the account-level extended private key at
// m/44'/195'/0' from a BIP39 seed. Shared by DeriveDepositPrivKey and
// DeriveEnergyAccountPrivKey to avoid duplicating the hardened derivation chain.
// Caller must zero the returned key after use.
func deriveAccountExtKey(seed []byte) (*hdkeychain.ExtendedKey, error) {
	master, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, fmt.Errorf("hdwallet: create master key: %w", err)
	}

	purpose, err := master.Derive(hdkeychain.HardenedKeyStart + 44)
	if err != nil {
		return nil, fmt.Errorf("hdwallet: derive purpose: %w", err)
	}

	coinType, err := purpose.Derive(hdkeychain.HardenedKeyStart + 195)
	if err != nil {
		return nil, fmt.Errorf("hdwallet: derive coin type: %w", err)
	}

	account, err := coinType.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, fmt.Errorf("hdwallet: derive account: %w", err)
	}

	// Zero intermediates — account holds a copy of the key material.
	master.Zero()
	purpose.Zero()
	coinType.Zero()

	return account, nil
}

// DeriveDepositPrivKey derives the private key at m/44'/195'/0'/0/{index}
// from a BIP39 seed. OFFLINE USE ONLY — cmd/coldsign.
// The returned PrivateKey is an independent copy; intermediate ExtendedKey
// material is zeroed internally.
func DeriveDepositPrivKey(seed []byte, index uint32) (*btcec.PrivateKey, error) {
	account, err := deriveAccountExtKey(seed)
	if err != nil {
		return nil, err
	}
	defer account.Zero()

	external, err := account.Derive(0)
	if err != nil {
		return nil, fmt.Errorf("hdwallet: derive external chain: %w", err)
	}
	defer external.Zero()

	child, err := external.Derive(index)
	if err != nil {
		return nil, fmt.Errorf("hdwallet: derive index %d: %w", index, err)
	}
	defer child.Zero()

	privKey, err := child.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("hdwallet: get private key: %w", err)
	}
	return privKey, nil
}

// DeriveEnergyAccountPrivKey derives the private key at m/44'/195'/0'/1/0
// (change=1, index=0 — the energy account fixed path) from a BIP39 seed.
// OFFLINE USE ONLY — cmd/coldsign.
func DeriveEnergyAccountPrivKey(seed []byte) (*btcec.PrivateKey, error) {
	account, err := deriveAccountExtKey(seed)
	if err != nil {
		return nil, err
	}
	defer account.Zero()

	change, err := account.Derive(1)
	if err != nil {
		return nil, fmt.Errorf("hdwallet: derive change chain: %w", err)
	}
	defer change.Zero()

	energy, err := change.Derive(0)
	if err != nil {
		return nil, fmt.Errorf("hdwallet: derive energy index: %w", err)
	}
	defer energy.Zero()

	privKey, err := energy.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("hdwallet: get energy private key: %w", err)
	}
	return privKey, nil
}

// ZeroSeed securely zeroes a BIP39 seed slice.
func ZeroSeed(seed []byte) {
	common.ZeroBytes(seed)
}
