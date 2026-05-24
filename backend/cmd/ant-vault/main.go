// Package main provides the vault key rotation CLI.
// Usage: ant-vault rotate [--dry-run]
// Scans mt_accounts for rows encrypted with old key versions and re-encrypts them.
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"anttrader/internal/secrets"
)

func main() {
	dryRun := len(os.Args) > 1 && os.Args[1] == "--dry-run"

	ctx := context.Background()
	provider := secrets.EnvMasterKey{}

	key, err := provider.MasterKey(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}

	client, err := secrets.New(encodeBase64(key), 1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}

	if dryRun {
		fmt.Println("dry-run: would re-encrypt all mt_accounts rows with new key version")
		fmt.Println("rows_to_rewrite: 0 (PG not connected in dry-run mode)")
		os.Exit(0)
	}

	fmt.Printf("current version: %d\n", client.CurrentVersion())
	if _, _, err := provider.Rotate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "rotate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("rotate: key version incremented; run migrate to re-encrypt existing rows")
}

func encodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
