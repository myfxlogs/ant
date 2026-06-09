package secretbox

import (
	"bytes"
	"testing"
)

func TestNew_NilOnEmptyKey(t *testing.T) {
	t.Parallel()
	b := New([]byte{})
	if b != nil {
		t.Fatal("expected nil Box for empty master key")
	}
}

func TestNew_NonEmptyKey(t *testing.T) {
	t.Parallel()
	b := New([]byte("my-master-key"))
	if b == nil {
		t.Fatal("expected non-nil Box for non-empty master key")
	}
}

func TestSealOpen_RoundTrip(t *testing.T) {
	t.Parallel()
	box := New([]byte("super-secret-master-key-32bytes!!"))
	plaintext := []byte("sensitive data: api_key=sk-1234567890")

	ct, salt, nonce, err := box.Seal(plaintext)
	if err != nil {
		t.Fatalf("Seal failed: %v", err)
	}
	if len(ct) == 0 {
		t.Fatal("expected non-empty ciphertext")
	}
	if len(salt) != saltLen {
		t.Fatalf("expected salt length %d, got %d", saltLen, len(salt))
	}
	if len(nonce) != nonceLen {
		t.Fatalf("expected nonce length %d, got %d", nonceLen, len(nonce))
	}
	// Ciphertext should differ from plaintext.
	if bytes.Equal(ct, plaintext) {
		t.Fatal("ciphertext should not equal plaintext")
	}

	decrypted, err := box.Open(ct, salt, nonce)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted data doesn't match: got %q, want %q", decrypted, plaintext)
	}
}

func TestSeal_NilBox(t *testing.T) {
	t.Parallel()
	var box *Box // nil
	_, _, _, err := box.Seal([]byte("data"))
	if err == nil {
		t.Fatal("expected error for nil Box")
	}
}

func TestOpen_NilBox(t *testing.T) {
	t.Parallel()
	var box *Box
	_, err := box.Open([]byte("ct"), make([]byte, saltLen), make([]byte, nonceLen))
	if err == nil {
		t.Fatal("expected error for nil Box")
	}
}

func TestSeal_DifferentOutputs(t *testing.T) {
	t.Parallel()
	box := New([]byte("master-key"))
	ct1, s1, n1, _ := box.Seal([]byte("same-data"))
	ct2, s2, n2, _ := box.Seal([]byte("same-data"))
	if bytes.Equal(ct1, ct2) && bytes.Equal(s1, s2) && bytes.Equal(n1, n2) {
		t.Fatal("two encryptions of same plaintext should produce different outputs (random salt/nonce)")
	}
}

func TestOpen_TamperedCiphertext(t *testing.T) {
	t.Parallel()
	box := New([]byte("master-key-12345678901234567890"))
	ct, salt, nonce, err := box.Seal([]byte("secret"))
	if err != nil {
		t.Fatalf("Seal failed: %v", err)
	}
	// Flip a byte in the ciphertext.
	ct[0] ^= 0x01
	_, err = box.Open(ct, salt, nonce)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext (GCM auth tag check)")
	}
}

func TestOpen_InvalidSaltLength(t *testing.T) {
	t.Parallel()
	box := New([]byte("key"))
	_, err := box.Open([]byte("ct"), []byte("short"), make([]byte, nonceLen))
	if err == nil {
		t.Fatal("expected error for invalid salt length")
	}
}

func TestOpen_InvalidNonceLength(t *testing.T) {
	t.Parallel()
	box := New([]byte("key"))
	_, err := box.Open([]byte("ct"), make([]byte, saltLen), []byte("short"))
	if err == nil {
		t.Fatal("expected error for invalid nonce length")
	}
}

func TestOpen_WrongKey(t *testing.T) {
	t.Parallel()
	box1 := New([]byte("key-alpha"))
	box2 := New([]byte("key-beta"))
	ct, salt, nonce, _ := box1.Seal([]byte("data"))
	_, err := box2.Open(ct, salt, nonce)
	if err == nil {
		t.Fatal("expected decryption to fail with wrong key")
	}
}
