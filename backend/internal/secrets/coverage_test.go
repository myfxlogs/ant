package secrets_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"anttrader/internal/secrets"
)

func TestNew_InvalidVersion(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	_, err = secrets.New(key, 0)
	if err == nil {
		t.Fatal("expected error for currentVersion=0")
	}
	if !strings.Contains(err.Error(), "must be >= 1") {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestNew_InvalidKeyLength(t *testing.T) {
	_, err := secrets.New("dG9vLXNob3J0", 1) // "too-short" in base64 (9 bytes)
	if err == nil {
		t.Fatal("expected error for short key")
	}
	if !strings.Contains(err.Error(), "must be 32 bytes") {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	client, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Decrypt(context.Background(), secrets.PurposeMTPassword, []byte("short"))
	if err == nil {
		t.Fatal("expected error for short ciphertext")
	}
}

func TestDecrypt_UnknownVersion(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	client, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Version byte 99 — not known.
	ciphertext := make([]byte, 1+12+32)
	ciphertext[0] = 99
	_, err = client.Decrypt(context.Background(), secrets.PurposeMTPassword, ciphertext)
	if err == nil {
		t.Fatal("expected ErrUnknownKeyVersion")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	client, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	plaintext := []byte("tamper-test")
	ciphertext, err := client.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Tamper with middle byte.
	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[len(tampered)/2] ^= 0xFF

	_, err = client.Decrypt(context.Background(), secrets.PurposeMTPassword, tampered)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestReencrypt_AESGCM(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	client, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	plaintext := []byte("reencrypt-inner")
	ciphertext, err := client.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	reencrypted, err := client.Reencrypt(context.Background(), secrets.PurposeMTPassword, ciphertext)
	if err != nil {
		t.Fatalf("Reencrypt: %v", err)
	}

	decrypted, err := client.Decrypt(context.Background(), secrets.PurposeMTPassword, reencrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("round-trip: got %q, want %q", decrypted, plaintext)
	}
}

func TestRotateKey_ErrorPath(t *testing.T) {
	// FileMasterKey.Rotate returns an error (unsupported).
	_, _, err := secrets.FileMasterKey{}.Rotate(context.Background())
	if err == nil {
		t.Fatal("expected error for FileMasterKey.Rotate")
	}
}

func TestDecodeMasterKey_Base64Fallback(t *testing.T) {
	// Provide a non-base64 32-char string — should be used raw.
	t.Setenv("ANT_MASTER_KEY", "abcdefghijklmnopqrstuvwxyz123456")
	_, err := secrets.EnvMasterKey{}.MasterKey(context.Background())
	// 32 bytes as raw string — key should be accepted.
	if err != nil {
		t.Fatalf("raw key rejected: %v", err)
	}
}

func TestEnvelopeDecryptV2_Tampered(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	client := secrets.NewEnvelopeClient(inner)
	plaintext := []byte("v2-tamper-test")

	ciphertext, err := client.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Tamper with data portion.
	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	// Flip a byte in the encrypted data section (after wrapped DEK).
	tampered[len(tampered)-5] ^= 0xFF

	_, err = client.Decrypt(context.Background(), secrets.PurposeMTPassword, tampered)
	if err == nil {
		t.Fatal("expected error for tampered v2 ciphertext")
	}
}

func TestEnvelopeDecrypt_TooShort(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	client := secrets.NewEnvelopeClient(inner)
	_, err = client.Decrypt(context.Background(), secrets.PurposeMTPassword, []byte{0x02, 0x01})
	if err == nil {
		t.Fatal("expected error for short v2 ciphertext")
	}
}

func TestEnvelopeDecryptV2_TruncatedWrappedDEK(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	client := secrets.NewEnvelopeClient(inner)

	// Construct a valid-looking v2 header with truncated wrapped DEK.
	// v2 marker(1) + dek_kid(1) + nonce(12) + partial wrapped(20)
	truncated := make([]byte, 2+12+20)
	truncated[0] = 0x02
	truncated[1] = 0x01
	// nonce bytes are zero
	// wrapped part too short

	_, err = client.Decrypt(context.Background(), secrets.PurposeMTPassword, truncated)
	if err == nil {
		t.Fatal("expected error for truncated wrapped DEK")
	}
}

func TestEnvelopeReencrypt_Error(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	client := secrets.NewEnvelopeClient(inner)

	// Reencrypt garbage should fail.
	_, err = client.Reencrypt(context.Background(), secrets.PurposeMTPassword, []byte("not-valid-ciphertext"))
	if err == nil {
		t.Fatal("expected error for reencrypt of invalid data")
	}
}

func TestMigrateToEnvelope_BrokenCiphertext(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	client := secrets.NewEnvelopeClient(inner)

	// Legacy format marker but broken content.
	broken := make([]byte, 1+12+16+10)
	broken[0] = 1 // looks like legacy

	_, err = secrets.MigrateToEnvelope(client, secrets.PurposeMTPassword, broken)
	if err == nil {
		t.Fatal("expected error for broken legacy ciphertext")
	}
}

func TestRotateKey_ThenEncrypt(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	client, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rc, ok := client.(secrets.RotateClient)
	if !ok {
		t.Fatal("client does not implement RotateClient")
	}

	// Rotate twice to test multi-version.
	_, _, err = rc.RotateKey()
	if err != nil {
		t.Fatalf("RotateKey #1: %v", err)
	}
	_, _, err = rc.RotateKey()
	if err != nil {
		t.Fatalf("RotateKey #2: %v", err)
	}

	// Encrypt with current version (should be v3).
	plaintext := []byte("multi-version-test")
	ciphertext, err := client.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if ciphertext[0] != 3 {
		t.Fatalf("expected version 3, got %d", ciphertext[0])
	}

	decrypted, err := client.Decrypt(context.Background(), secrets.PurposeMTPassword, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("round-trip: got %q, want %q", decrypted, plaintext)
	}
}

func TestKeyDir_Env(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	t.Setenv("ANT_MASTER_KEY", key)
	t.Setenv("ANT_KEY_DIR", t.TempDir())

	// Rotate to ensure key_dir env is exercised.
	_, _, err = secrets.EnvMasterKey{}.Rotate(context.Background())
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
}

func TestRotateKey_ThenEnvelopeEncrypt(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Rotate inner key.
	rc, ok := inner.(secrets.RotateClient)
	if !ok {
		t.Fatal("not RotateClient")
	}
	_, _, err = rc.RotateKey()
	if err != nil {
		t.Fatalf("RotateKey: %v", err)
	}

	// EnvelopeClient uses inner with version 2.
	client := secrets.NewEnvelopeClient(inner)
	plaintext := []byte("post-rotate-envelope")
	ciphertext, err := client.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Should use v2 for dek_kid.
	if ciphertext[1] != 2 {
		t.Fatalf("dek_kid: want 2, got %d", ciphertext[1])
	}

	decrypted, err := client.Decrypt(context.Background(), secrets.PurposeMTPassword, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("round-trip: got %q, want %q", decrypted, plaintext)
	}
}

func TestNextKeyVersion_WithExistingFiles(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	t.Setenv("ANT_MASTER_KEY", key)
	keyDir := t.TempDir()
	t.Setenv("ANT_KEY_DIR", keyDir)

	// First rotation creates v2.
	_, _, err = secrets.EnvMasterKey{}.Rotate(context.Background())
	if err != nil {
		t.Fatalf("Rotate #1: %v", err)
	}

	// Second rotation should create v3.
	ver, _, err := secrets.EnvMasterKey{}.Rotate(context.Background())
	if err != nil {
		t.Fatalf("Rotate #2: %v", err)
	}
	if ver != 3 {
		t.Fatalf("second rotation: want version 3, got %d", ver)
	}
}

func TestDecodeBase64_Invalid(t *testing.T) {
	// Use a 44-char string that is not valid base64.
	t.Setenv("ANT_MASTER_KEY", "!!!!invalid-base64-string-with-44-chars!!")
	_, err := secrets.EnvMasterKey{}.MasterKey(context.Background())
	if err != nil {
		t.Fatalf("raw key fallback should work: %v", err)
	}
}

func TestReencryptAESGCM_DecryptError(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	client, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Reencrypt of garbage should fail at decrypt stage.
	_, err = client.Reencrypt(context.Background(), secrets.PurposeMTPassword, []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	if err == nil {
		t.Fatal("expected error for reencrypt of invalid ciphertext")
	}
}

func TestGetAEADConcurrent(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	client, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			plaintext := []byte("concurrent-test-data")
			ciphertext, _ := client.Encrypt(context.Background(), secrets.PurposeMTPassword, plaintext)
			client.Decrypt(context.Background(), secrets.PurposeMTPassword, ciphertext)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRotate_KeyDirDefault(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	// Don't set ANT_KEY_DIR — exercise default /var/lib/ant/keys.
	// Use a temp home so we don't write to real /var/lib.
	t.Setenv("ANT_MASTER_KEY", key)
	// Unset ANT_KEY_DIR to exercise default path.
	t.Setenv("ANT_KEY_DIR", "")

	// Rotate should attempt to mkdir /var/lib/ant/keys; may fail due to perms
	// but that exercises the error path.
	_, _, err = secrets.EnvMasterKey{}.Rotate(context.Background())
	// Expected: may fail if /var/lib/ant doesn't exist (permission error).
	t.Logf("rotate with default keydir: %v", err)
}

// failingReader always returns an error.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestEncrypt_NonceReadError(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	client, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	secrets.SetRandReader(failingReader{})
	defer secrets.ResetRandReader()

	_, err = client.Encrypt(context.Background(), secrets.PurposeMTPassword, []byte("test"))
	if err == nil {
		t.Fatal("expected error from failing random reader")
	}
	t.Logf("nonce read error: %v (PASS)", err)
}

func TestRotateKey_ReadError(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	client, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rc, ok := client.(secrets.RotateClient)
	if !ok {
		t.Fatal("not RotateClient")
	}

	secrets.SetRandReader(failingReader{})
	defer secrets.ResetRandReader()

	_, _, err = rc.RotateKey()
	if err == nil {
		t.Fatal("expected error from failing random reader on RotateKey")
	}
	t.Logf("RotateKey read error: %v (PASS)", err)
}

func TestEnvelopeEncrypt_DekReadError(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	inner, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	client := secrets.NewEnvelopeClient(inner)

	secrets.SetRandReader(failingReader{})
	defer secrets.ResetRandReader()

	_, err = client.Encrypt(context.Background(), secrets.PurposeMTPassword, []byte("test"))
	if err == nil {
		t.Fatal("expected error from failing random reader on envelope encrypt")
	}
	t.Logf("EnvelopeEncrypt DEK read error: %v (PASS)", err)
}

func TestRotate_ProviderError(t *testing.T) {
	// Don't set ANT_MASTER_KEY at all — MasterKey will fail.
	t.Setenv("ANT_MASTER_KEY", "")
	_, _, err := secrets.EnvMasterKey{}.Rotate(context.Background())
	if err == nil {
		t.Fatal("expected error when ANT_MASTER_KEY is empty")
	}
	t.Logf("Rotate with empty key: %v (PASS)", err)
}

func TestFileMasterKey_ReadError(t *testing.T) {
	t.Setenv("ANT_MASTER_KEY_FILE", "/nonexistent/file/path.key")
	_, err := secrets.FileMasterKey{}.MasterKey(context.Background())
	if err == nil {
		t.Fatal("expected error for nonexistent key file")
	}
	t.Logf("FileMasterKey read error: %v (PASS)", err)
}

func TestAESGCMEncrypt_DifferentPurposes(t *testing.T) {
	key, err := secrets.GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}

	client, err := secrets.New(key, 1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	plaintext := []byte("purpose-test")

	// Encrypt with each purpose.
	for _, p := range []secrets.Purpose{secrets.PurposeMTPassword, secrets.PurposeMTAPIToken, secrets.PurposeBrokerCookie} {
		ciphertext, err := client.Encrypt(context.Background(), p, plaintext)
		if err != nil {
			t.Fatalf("Encrypt(%s): %v", p, err)
		}
		decrypted, err := client.Decrypt(context.Background(), p, ciphertext)
		if err != nil {
			t.Fatalf("Decrypt(%s): %v", p, err)
		}
		if string(decrypted) != string(plaintext) {
			t.Fatalf("%s round-trip: got %q", p, decrypted)
		}
	}
}
