package secrets_test

import (
	"context"
	"testing"

	"anttrader/internal/secrets"
)

// TestRotateKey verifies the L-3 multi-key rotation:
// 1. Encrypt with v1 → decrypt succeeds.
// 2. RotateKey → new version generated, old key retained.
// 3. Encrypt with new version → decrypt succeeds.
// 4. Old v1 ciphertext still decryptable after rotation.
func TestRotateKey(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	client, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// 1. Encrypt with v1.
	plaintext := []byte("secret-mt-password")
	cipherV1, err := client.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
	if err != nil {
		t.Fatalf("Encrypt v1: %v", err)
	}
	if cipherV1[0] != 1 {
		t.Fatalf("version byte: want 1, got %d", cipherV1[0])
	}

	// 2. Decrypt v1.
	decrypted, err := client.Decrypt(context.Background(), secrets.PurposeMTPassword, cipherV1)
	if err != nil {
		t.Fatalf("Decrypt v1: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("round-trip v1: got %q, want %q", decrypted, plaintext)
	}

	// 3. Rotate to v2.
	rc, ok := client.(secrets.RotateClient)
	if !ok {
		t.Fatal("client does not implement RotateClient")
	}
	newVer, newKeyB64, err := rc.RotateKey()
	if err != nil {
		t.Fatalf("RotateKey: %v", err)
	}
	if newVer != 2 {
		t.Fatalf("new version: want 2, got %d", newVer)
	}
	if len(newKeyB64) != 44 {
		t.Fatalf("new key b64 length: want 44, got %d", len(newKeyB64))
	}
	t.Logf("RotateKey: version=%d, key_len=%d", newVer, len(newKeyB64))

	// 4. Encrypt with v2.
	cipherV2, err := client.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
	if err != nil {
		t.Fatalf("Encrypt v2: %v", err)
	}
	if cipherV2[0] != 2 {
		t.Fatalf("version byte after rotation: want 2, got %d", cipherV2[0])
	}

	// 5. Decrypt v2.
	decrypted2, err := client.Decrypt(context.Background(), secrets.PurposeMTPassword, cipherV2)
	if err != nil {
		t.Fatalf("Decrypt v2: %v", err)
	}
	if string(decrypted2) != string(plaintext) {
		t.Fatalf("round-trip v2: got %q, want %q", decrypted2, plaintext)
	}

	// 6. Old v1 ciphertext still decryptable (key retained).
	decryptedOld, err := client.Decrypt(context.Background(), secrets.PurposeMTPassword, cipherV1)
	if err != nil {
		t.Fatalf("Decrypt old v1 after rotation: %v", err)
	}
	if string(decryptedOld) != string(plaintext) {
		t.Fatalf("backward compat: got %q, want %q", decryptedOld, plaintext)
	}

	t.Log("Round-trip: v1 → rotate → v2, old v1 still decryptable (PASS)")
}
