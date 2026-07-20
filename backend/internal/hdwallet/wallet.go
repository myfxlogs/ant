// Package hdwallet — wallet.go provides BIP39 mnemonic generation.
// Used by cmd/hdgen (air-gapped tool) to generate the root mnemonic.
package hdwallet

import (
	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
)

// GenerateMnemonic generates a new 24-word BIP39 mnemonic (256-bit entropy).
// Used by cmd/hdgen on an air-gapped machine.
func GenerateMnemonic() (string, error) {
	return mnemonic.Generate(256)
}
