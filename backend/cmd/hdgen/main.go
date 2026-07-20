// cmd/hdgen is an air-gapped tool that generates a BIP39 mnemonic,
// derives the account-level xpub (m/44'/195'/0'/0), and exports
// xpub + fingerprint as proto binary. No private key material is exported.
//
// Usage: hdgen [-o output.bin]
//   -o  output file path (default: stdout as hex)
//
// Run ONLY on an air-gapped machine. The mnemonic must be written down
// and stored offline — it is the root of all deposit address private keys.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/fbsobreira/go-bip39"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/hdwallet"
)

func main() {
	outPath := flag.String("o", "", "output file path (default: stdout hex)")
	flag.Parse()

	if err := run(*outPath); err != nil {
		fmt.Fprintf(os.Stderr, "hdgen: %v\n", err)
		os.Exit(1)
	}
}

func run(outPath string) error {
	// 1. Generate 24-word BIP39 mnemonic (256-bit entropy).
	mnemonic, err := hdwallet.GenerateMnemonic()
	if err != nil {
		return fmt.Errorf("generate mnemonic: %w", err)
	}

	// 2. Derive account-level xpub from mnemonic.
	seed := bip39.NewSeed(mnemonic, "")
	defer hdwallet.ZeroSeed(seed)
	xpub, fingerprint, err := hdwallet.DeriveAccountXpubAndFingerprint(seed)
	if err != nil {
		return fmt.Errorf("derive xpub: %w", err)
	}

	// 3. Print mnemonic to stderr (operator writes it down, never stored digitally).
	fmt.Fprintf(os.Stderr, "=== MNEMONIC (write down, store offline) ===\n")
	fmt.Fprintf(os.Stderr, "%s\n", mnemonic)
	fmt.Fprintf(os.Stderr, "=============================================\n\n")

	// 4. Export xpub + fingerprint as proto binary (no private key).
	export := &antv1.XpubExport{
		Xpub:        xpub,
		Fingerprint: fingerprint,
		Network:     "TRC20",
	}
	data, err := proto.Marshal(export)
	if err != nil {
		return fmt.Errorf("marshal xpub export: %w", err)
	}

	if outPath != "" {
		if err := os.WriteFile(outPath, data, 0600); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		fmt.Fprintf(os.Stderr, "xpub exported to %s\n", outPath)
	} else {
		fmt.Printf("%x\n", data)
	}

	fmt.Fprintf(os.Stderr, "xpub: %s\n", xpub)
	fmt.Fprintf(os.Stderr, "fingerprint: %s\n", fingerprint)
	return nil
}
