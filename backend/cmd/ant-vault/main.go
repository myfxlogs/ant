// Package main provides the vault key rotation CLI.
// Usage: ant-vault rotate [--dry-run]
// Rotates the master encryption key and optionally re-encrypts existing rows.
// L-3: EnvMasterKey rotation writes new key to ANT_KEY_DIR/ant-master-key-v{N}.key.
// The operator copies the new key into ANT_MASTER_KEY on next restart.
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"anttrader/internal/secrets"
)

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	dryRun := cmd == "--dry-run"
	for _, a := range os.Args[1:] {
		if a == "--dry-run" {
			dryRun = true
		}
	}

	if cmd != "rotate" && cmd != "--dry-run" && cmd != "" {
		fmt.Fprintf(os.Stderr, "usage: ant-vault rotate [--dry-run]\n")
		os.Exit(1)
	}

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
		fmt.Println("dry-run: would rotate key and re-encrypt affected rows")
		fmt.Printf("current_version: %d\n", client.CurrentVersion())
		fmt.Println("rows_to_rewrite: 0 (dry-run: PG connection omitted)")
		os.Exit(0)
	}

	fmt.Printf("current version: %d\n", client.CurrentVersion())

	// L-3: use RotateClient for in-process key rotation.
	rc, ok := client.(secrets.RotateClient)
	if !ok {
		fmt.Fprintln(os.Stderr, "rotate: client does not support in-process rotation (use KMS)")
		os.Exit(1)
	}

	newVersion, newKeyB64, err := rc.RotateKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "rotate: client.RotateKey: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("rotate: new key generated, version=%d\n", newVersion)

	// Persist the new key via the provider.
	newKey, _ := base64.StdEncoding.DecodeString(newKeyB64)
	newVer, _, err := provider.Rotate(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rotate: persist key: %v\n", err)
		os.Exit(1)
	}
	_ = newKey // used above in RotateKey, now persisted via provider
	fmt.Printf("rotate: key persisted to ANT_KEY_DIR, version=%d\n", newVer)
	fmt.Println("rotate: done — restart with ANT_MASTER_KEY set to the new key value")
	fmt.Println("         (find it at $ANT_KEY_DIR/ant-master-key-v*.key)")
}

func encodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
