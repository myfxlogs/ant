// Package hdwallet provides BIP44 HD wallet derivation for TRON addresses.
// This file implements watch-only (xpub-only) address derivation per ADR-0026 §10.1.
// The online server uses ONLY the account-level extended public key — no private keys
// ever touch the online machine (R1).
package hdwallet

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
)

// DeriveAddressFromXpub derives a TRON Base58 address from an account-level
// extended public key (xpub at m/44'/195'/0'/0) and a non-hardened child index.
// Uses BIP32 CKDpub (public-only derivation) — no private key needed (R1).
func DeriveAddressFromXpub(accountXpub string, index uint32) (string, error) {
	ext, err := hdkeychain.NewKeyFromString(accountXpub)
	if err != nil {
		return "", fmt.Errorf("hdwallet: parse xpub: %w", err)
	}
	return DeriveAddressFromExtKey(ext, index)
}

// DeriveAddressFromExtKey derives a TRON Base58 address from a pre-parsed
// account-level extended public key. Use this when calling DeriveAddress in a
// hot loop to avoid re-parsing the xpub string on every call.
func DeriveAddressFromExtKey(ext *hdkeychain.ExtendedKey, index uint32) (string, error) {
	child, err := ext.Derive(index)
	if err != nil {
		return "", fmt.Errorf("hdwallet: derive index %d: %w", index, err)
	}

	pubKey, err := child.ECPubKey()
	if err != nil {
		return "", fmt.Errorf("hdwallet: get pubkey: %w", err)
	}

	addr := address.BTCECPubkeyToAddress(pubKey)
	return addr.String(), nil
}

// ParseXpub parses a base58-encoded extended public key string.
// The returned *ExtendedKey can be reused across multiple DeriveAddressFromExtKey
// calls to avoid repeated base58 decoding + checksum verification.
func ParseXpub(accountXpub string) (*hdkeychain.ExtendedKey, error) {
	ext, err := hdkeychain.NewKeyFromString(accountXpub)
	if err != nil {
		return nil, fmt.Errorf("hdwallet: parse xpub: %w", err)
	}
	return ext, nil
}

// XpubFingerprint computes the SHA-256 fingerprint of an xpub for startup
// integrity verification (R5). Per ADR-0026 §10.1: hex(sha256(xpub)).
// Returns the full 64-character hex encoding (256-bit collision resistance).
func XpubFingerprint(accountXpub string) (string, error) {
	h := sha256.Sum256([]byte(accountXpub))
	return hex.EncodeToString(h[:]), nil
}

// ValidateXpub checks that the given string is a valid BIP32 extended public key
// on the Bitcoin mainnet (TRON uses the same secp256k1 curve; BIP44 coin type
// 195 is encoded in the derivation path, not in the network params).
func ValidateXpub(accountXpub string) error {
	_, err := hdkeychain.NewKeyFromString(accountXpub)
	if err != nil {
		return fmt.Errorf("hdwallet: invalid xpub: %w", err)
	}
	return nil
}

// DeriveAccountXpubAndFingerprint derives the account-level xpub
// (m/44'/195'/0'/0) and its SHA-256 fingerprint from a BIP39 seed.
// Used by cmd/hdgen and cmd/coldsign (air-gapped tools only).
// All intermediate private key material is zeroed AFTER xpub extraction.
func DeriveAccountXpubAndFingerprint(seed []byte) (xpubStr, fingerprint string, err error) {
	account, err := deriveAccountExtKey(seed)
	if err != nil {
		return "", "", err
	}
	defer account.Zero()

	external, err := account.Derive(0)
	if err != nil {
		return "", "", fmt.Errorf("hdwallet: derive external chain: %w", err)
	}
	defer external.Zero()

	xpubKey, err := external.Neuter()
	if err != nil {
		return "", "", fmt.Errorf("hdwallet: neuter to xpub: %w", err)
	}
	xpubStr = xpubKey.String()

	fingerprint, err = XpubFingerprint(xpubStr)
	if err != nil {
		return "", "", fmt.Errorf("hdwallet: compute fingerprint: %w", err)
	}
	return xpubStr, fingerprint, nil
}
