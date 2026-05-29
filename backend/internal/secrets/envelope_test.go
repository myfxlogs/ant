package secrets_test

import (
	"context"
	"testing"

	"anttrader/internal/secrets"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	client := secrets.NewEnvelopeClient(inner)

	plaintext := []byte("sensitive-broker-password-12345")
	ciphertext, err := client.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Verify v2 format marker.
	if ciphertext[0] != 0x02 {
		t.Fatalf("version marker: want 0x02, got 0x%02x", ciphertext[0])
	}

	// Verify dek_kid matches inner current version.
	if ciphertext[1] != 1 {
		t.Fatalf("dek_kid: want 1, got %d", ciphertext[1])
	}

	decrypted, err := client.Decrypt(context.Background(), secrets.PurposeMTPassword, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("round-trip: got %q, want %q", decrypted, plaintext)
	}

	t.Logf("Envelope round-trip: %d bytes → %d bytes (PASS)", len(plaintext), len(ciphertext))
}

func TestEnvelopeDecryptLegacy(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Encrypt with legacy format (v1).
	plaintext := []byte("legacy-password")
	legacyCipher, err := inner.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
	if err != nil {
		t.Fatalf("Encrypt legacy: %v", err)
	}

	// EnvelopeClient should decrypt legacy format transparently.
	client := secrets.NewEnvelopeClient(inner)
	decrypted, err := client.Decrypt(context.Background(), secrets.PurposeMTPassword, legacyCipher)
	if err != nil {
		t.Fatalf("Decrypt legacy via envelope: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("legacy decrypt: got %q, want %q", decrypted, plaintext)
	}

	// Verify legacy format marker.
	if legacyCipher[0] != 1 {
		t.Fatalf("legacy version: want 1, got %d", legacyCipher[0])
	}

	t.Log("Envelope → legacy backward compat (PASS)")
}

func TestEnvelopeReencrypt(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	client := secrets.NewEnvelopeClient(inner)

	plaintext := []byte("reencrypt-test")
	ciphertext, err := client.Encrypt(context.Background(), secrets.PurposeMTAPIToken, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	reencrypted, err := client.Reencrypt(context.Background(), secrets.PurposeMTAPIToken, ciphertext)
	if err != nil {
		t.Fatalf("Reencrypt: %v", err)
	}

	decrypted, err := client.Decrypt(context.Background(), secrets.PurposeMTAPIToken, reencrypted)
	if err != nil {
		t.Fatalf("Decrypt reencrypted: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("reencrypt round-trip: got %q, want %q", decrypted, plaintext)
	}

	t.Log("Envelope reencrypt (PASS)")
}

func TestEnvelopeDecryptWrongPurpose(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	client := secrets.NewEnvelopeClient(inner)
	plaintext := []byte("test-purpose")

	ciphertext, err := client.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Decrypting with wrong purpose should fail (HKDF derives different key).
	_, err = client.Decrypt(context.Background(), secrets.PurposeMTAPIToken, ciphertext)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong purpose")
	}
	t.Logf("Wrong-purpose rejection: %v (PASS)", err)
}

func TestEnvelopeCurrentVersion(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 5)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	client := secrets.NewEnvelopeClient(inner)
	if v := client.CurrentVersion(); v != 5 {
		t.Fatalf("CurrentVersion: want 5, got %d", v)
	}
	t.Log("Envelope CurrentVersion delegates (PASS)")
}

func TestIsLegacyFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"empty", nil, false},
		{"v1", []byte{1, 0xaa, 0xbb}, true},
		{"v127", []byte{127, 0x00}, true},
		{"v2_envelope", []byte{0x02, 0x01, 0x00}, false},
		{"zero_version", []byte{0, 0x00}, false},
		{"high_version", []byte{128, 0x00}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := secrets.IsLegacyFormat(tt.data)
			if got != tt.want {
				t.Errorf("IsLegacyFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMigrateToEnvelope(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	client := secrets.NewEnvelopeClient(inner)
	plaintext := []byte("migrate-me")

	// Start with legacy format.
	legacy, err := inner.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
	if err != nil {
		t.Fatalf("legacy Encrypt: %v", err)
	}
	if !secrets.IsLegacyFormat(legacy) {
		t.Fatal("expected legacy format")
	}

	// Migrate to envelope.
	migrated, err := secrets.MigrateToEnvelope(client, secrets.PurposeMTPassword, legacy)
	if err != nil {
		t.Fatalf("MigrateToEnvelope: %v", err)
	}
	if secrets.IsLegacyFormat(migrated) {
		t.Fatal("migrated ciphertext should not be legacy format")
	}

	// Decrypt migrated.
	decrypted, err := client.Decrypt(context.Background(), secrets.PurposeMTPassword, migrated)
	if err != nil {
		t.Fatalf("Decrypt migrated: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("migrated round-trip: got %q, want %q", decrypted, plaintext)
	}

	t.Log("MigrateToEnvelope (PASS)")
}

func TestMigrateNonLegacyPassthrough(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	client := secrets.NewEnvelopeClient(inner)
	plaintext := []byte("already-v2")

	// Encrypt in v2 format.
	v2Cipher, err := client.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// MigrateToEnvelope on already-v2 data should be a no-op.
	result, err := secrets.MigrateToEnvelope(client, secrets.PurposeMTPassword, v2Cipher)
	if err != nil {
		t.Fatalf("MigrateToEnvelope on v2: %v", err)
	}
	if len(result) != len(v2Cipher) {
		t.Fatalf("passthrough: len changed from %d to %d", len(v2Cipher), len(result))
	}

	t.Log("MigrateToEnvelope non-legacy passthrough (PASS)")
}
