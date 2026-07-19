// Command hdgen is an offline CLI tool for batch-generating HD wallet deposit addresses.
// Run on an air-gapped machine. Outputs a proto binary file for import via admin UI.
//
// Usage (one command, all you need):
//
//	hdgen -count 100 -kek-base64 "<ANT_MASTER_KEY>"
//	hdgen -count 100 -kek-base64 "<ANT_MASTER_KEY>" -mnemonic "existing 24 words"
//	hdgen -count 100 -kek-base64 "<ANT_MASTER_KEY>" -out deposit_addresses.bin
//
// The output .bin file contains serialized AddressBatch proto (NOT JSON).
// Upload it via Admin → Wallet Management → Deposit Addresses → Import button.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/hdwallet"
	"alphaforge/internal/secrets"
	"google.golang.org/protobuf/proto"
)

func main() {
	mnemonic := flag.String("mnemonic", "", "BIP39 mnemonic (24 words). If empty, a new one is generated.")
	start := flag.Int("start", 0, "Starting derivation index")
	count := flag.Int("count", 100, "Number of addresses to generate")
	out := flag.String("out", "deposit_addresses.bin", "Output file (proto binary)")
	kekB64 := flag.String("kek-base64", "", "Base64-encoded KEK (must match server's ANT_MASTER_KEY)")
	flag.Parse()

	if *count <= 0 {
		fatal("count must be positive")
	}
	if *kekB64 == "" {
		fatal("-kek-base64 is required (use the same value as server's ANT_MASTER_KEY env)")
	}

	mnemonicPhrase := *mnemonic
	if mnemonicPhrase == "" {
		var err error
		mnemonicPhrase, err = hdwallet.GenerateMnemonic()
		if err != nil {
			fatal("generate mnemonic: %v", err)
		}
		fmt.Fprintf(os.Stderr, "=== WRITE THIS DOWN — NEVER STORE ELECTRONICALLY ===\n")
		fmt.Fprintf(os.Stderr, "Mnemonic: %s\n", mnemonicPhrase)
		fmt.Fprintf(os.Stderr, "======================================================\n\n")
	}

	sec, err := secrets.New(*kekB64, 1)
	if err != nil {
		fatal("create secrets client: %v", err)
	}

	addrs, err := hdwallet.DeriveBatch(mnemonicPhrase, *start, *count)
	if err != nil {
		fatal("derive batch: %v", err)
	}

	batch := &antv1.AddressBatch{Entries: make([]*antv1.AddressBatchEntry, 0, len(addrs))}
	for _, a := range addrs {
		enc, err := sec.Encrypt(context.Background(), secrets.PurposeDepositPrivKey, a.PrivateKey)
		if err != nil {
			fatal("encrypt private key for index %d: %v", a.Index, err)
		}
		batch.Entries = append(batch.Entries, &antv1.AddressBatchEntry{
			Address:          a.Address,
			DerivationIndex:  int32(a.Index),
			EncryptedPrivkey: enc,
			Network:          "TRC20",
		})
		fmt.Fprintf(os.Stderr, "  [%d] %s\n", a.Index, a.Address)
	}

	data, err := proto.Marshal(batch)
	if err != nil {
		fatal("marshal proto: %v", err)
	}
	if err := os.WriteFile(*out, data, 0600); err != nil {
		fatal("write file: %v", err)
	}

	fmt.Fprintf(os.Stderr, "\nDone: %d addresses → %s\n", len(batch.Entries), *out)
	fmt.Fprintf(os.Stderr, "Upload this file via: Admin → Wallet Management → Deposit Addresses → Import\n")
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "hdgen: "+format+"\n", args...)
	os.Exit(1)
}
