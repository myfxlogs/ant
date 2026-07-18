// Command hdgen is an offline CLI tool for batch-generating HD wallet deposit addresses.
// Run on an air-gapped machine. Outputs a JSON file for import into the server DB.
//
// Usage:
//
//	hdgen -mnemonic "word1 word2 ..." -start 0 -count 100 -out addresses.json
//	hdgen -generate-mnemonic -count 100 -out addresses.json
//	hdgen -import addresses.json -db-dsn "postgres://user:pass@host:5432/db?sslmode=disable"
//
// The output JSON contains address, derivation_index, and encrypted_privkey (hex).
// Private keys are encrypted using AES-256-GCM with a KEK provided via -kek-base64.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"alphaforge/internal/hdwallet"
	"alphaforge/internal/model"
	"alphaforge/internal/repository"
	"alphaforge/internal/secrets"
)

type addressEntry struct {
	Address          string `json:"address"`
	DerivationIndex  int    `json:"derivation_index"`
	EncryptedPrivkey string `json:"encrypted_privkey"` // hex-encoded AES-256-GCM ciphertext
}

func main() {
	mnemonic := flag.String("mnemonic", "", "BIP39 mnemonic phrase (24 words)")
	generateMnemonic := flag.Bool("generate-mnemonic", false, "Generate a new random mnemonic")
	start := flag.Int("start", 0, "Starting derivation index")
	count := flag.Int("count", 100, "Number of addresses to generate")
	out := flag.String("out", "addresses.json", "Output JSON file")
	kekB64 := flag.String("kek-base64", "", "Base64-encoded KEK for encrypting private keys (required for generation)")
	importFile := flag.String("import", "", "Import addresses from JSON file into DB (requires -db-dsn)")
	dbDSN := flag.String("db-dsn", "", "PostgreSQL DSN for import (e.g. postgres://user:pass@host:5432/db?sslmode=disable)")
	flag.Parse()

	if *importFile != "" {
		runImport(*importFile, *dbDSN)
		return
	}

	if *count <= 0 {
		fatal("count must be positive")
	}
	if *kekB64 == "" {
		fatal("-kek-base64 is required (use the same KEK as the server's secrets.Client)")
	}

	mnemonicPhrase := *mnemonic
	if *generateMnemonic {
		var err error
		mnemonicPhrase, err = hdwallet.GenerateMnemonic()
		if err != nil {
			fatal("generate mnemonic: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Generated mnemonic (WRITE THIS DOWN — never stored):\n%s\n\n", mnemonicPhrase)
	} else if mnemonicPhrase == "" {
		fatal("either -mnemonic or -generate-mnemonic is required")
	}

	sec, err := secrets.New(*kekB64, 1)
	if err != nil {
		fatal("create secrets client: %v", err)
	}

	addrs, err := hdwallet.DeriveBatch(mnemonicPhrase, *start, *count)
	if err != nil {
		fatal("derive batch: %v", err)
	}

	entries := make([]addressEntry, 0, len(addrs))
	for _, a := range addrs {
		enc, err := sec.Encrypt(context.Background(), secrets.PurposeDepositPrivKey, a.PrivateKey)
		if err != nil {
			fatal("encrypt private key for index %d: %v", a.Index, err)
		}
		entries = append(entries, addressEntry{
			Address:          a.Address,
			DerivationIndex:  a.Index,
			EncryptedPrivkey: hex.EncodeToString(enc),
		})
		fmt.Fprintf(os.Stderr, "  [%d] %s\n", a.Index, a.Address)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		fatal("marshal JSON: %v", err)
	}
	if err := os.WriteFile(*out, data, 0600); err != nil {
		fatal("write file: %v", err)
	}

	fmt.Fprintf(os.Stderr, "\nWrote %d addresses to %s\n", len(entries), *out)
	fmt.Fprintf(os.Stderr, "Import this file into the server DB via: hdgen -import %s -db-dsn <DSN>\n", *out)
}

func runImport(filename, dsn string) {
	if filename == "" {
		fatal("-import requires a filename")
	}
	if dsn == "" {
		fatal("-import requires -db-dsn")
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		fatal("read import file: %v", err)
	}

	var entries []addressEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		fatal("parse import file: %v", err)
	}

	if len(entries) == 0 {
		fatal("no addresses in import file")
	}

	// Convert hex-encoded encrypted privkeys to []byte for DB insertion.
	addrs := make([]model.DepositAddress, 0, len(entries))
	for _, e := range entries {
		encBytes, err := hex.DecodeString(e.EncryptedPrivkey)
		if err != nil {
			fatal("decode hex for address %s (index %d): %v", e.Address, e.DerivationIndex, err)
		}
		addrs = append(addrs, model.DepositAddress{
			Address:          e.Address,
			DerivationIndex:  e.DerivationIndex,
			EncryptedPrivkey: encBytes,
			Network:          "TRC20",
			Status:           "AVAILABLE",
		})
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fatal("connect to database: %v", err)
	}
	defer pool.Close()

	repo := repository.NewDepositAddressRepository(pool)
	if err := repo.ImportBatch(ctx, addrs); err != nil {
		fatal("import batch: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully imported %d addresses into the pool.\n", len(addrs))
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "hdgen: "+format+"\n", args...)
	os.Exit(1)
}
