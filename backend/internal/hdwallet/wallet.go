// Package hdwallet provides BIP44 HD wallet derivation for TRON addresses.
// Uses gotron-sdk's mnemonic package for BIP39 → BIP44 (coin type 195) derivation.
package hdwallet

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
)

// DerivedAddress holds the result of HD derivation for a single index.
type DerivedAddress struct {
	Address    string // TRON Base58 address (T...)
	PrivateKey []byte // raw 32-byte private key
	Index      int    // derivation index
}

// GenerateMnemonic generates a new 24-word BIP39 mnemonic (256-bit entropy).
func GenerateMnemonic() (string, error) {
	return mnemonic.Generate(256)
}

// DeriveFromMnemonic derives a single address from a mnemonic at the given index.
// Path: m/44'/195'/0'/0/{index}
func DeriveFromMnemonic(mnemonicPhrase string, index int) (*DerivedAddress, error) {
	sk, _ := mnemonic.FromSeedAndPassphrase(mnemonicPhrase, "", index)
	if sk == nil {
		return nil, fmt.Errorf("hdwallet: derive index %d: invalid mnemonic or index", index)
	}

	privKeyBytes := sk.Serialize()
	addr := address.BTCECPrivkeyToAddress(sk)

	result := &DerivedAddress{
		Address:    addr.String(),
		PrivateKey: make([]byte, 32),
		Index:      index,
	}
	copy(result.PrivateKey, privKeyBytes)

	common.ZeroBytes(privKeyBytes)

	return result, nil
}

// DeriveBatch derives addresses from index `start` to `start+count-1`.
// Returns all derived addresses or an error if any derivation fails.
func DeriveBatch(mnemonicPhrase string, start, count int) ([]*DerivedAddress, error) {
	result := make([]*DerivedAddress, 0, count)
	for i := start; i < start+count; i++ {
		addr, err := DeriveFromMnemonic(mnemonicPhrase, i)
		if err != nil {
			return nil, fmt.Errorf("hdwallet: batch derive at index %d: %w", i, err)
		}
		result = append(result, addr)
	}
	return result, nil
}

// AddressFromPrivateKey computes the TRON Base58 address from a raw 32-byte private key.
func AddressFromPrivateKey(privKey []byte) (string, error) {
	sk, _ := btcec.PrivKeyFromBytes(privKey)
	if sk == nil {
		return "", fmt.Errorf("hdwallet: parse private key: nil key")
	}
	addr := address.BTCECPrivkeyToAddress(sk)
	return addr.String(), nil
}
