package hash

import (
	"testing"
)

func TestHashPassword_Success(t *testing.T) {
	t.Parallel()
	hash, err := HashPassword("my-secret-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
}

func TestVerifyPassword_Correct(t *testing.T) {
	t.Parallel()
	password := "correct-horse-battery-staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected hash error: %v", err)
	}
	ok, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("unexpected verify error: %v", err)
	}
	if !ok {
		t.Fatal("expected password to be verified")
	}
}

func TestVerifyPassword_Wrong(t *testing.T) {
	t.Parallel()
	hash, _ := HashPassword("right-password")
	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to be rejected")
	}
}

func TestVerifyPassword_InvalidFormat(t *testing.T) {
	t.Parallel()
	_, err := VerifyPassword("anything", "not-a-valid-hash")
	if err == nil {
		t.Fatal("expected error for invalid hash format")
	}
}

func TestHashPasswordWithParams_CustomParams(t *testing.T) {
	t.Parallel()
	custom := &Argon2idParams{
		Memory:      32 * 1024,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
	hash, err := HashPasswordWithParams("test", custom)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ok, _ := VerifyPassword("test", hash)
	if !ok {
		t.Fatal("expected custom-params password to verify")
	}
}

func TestHashPassword_DifferentSalts(t *testing.T) {
	t.Parallel()
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")
	if h1 == h2 {
		t.Fatal("two hashes of the same password should differ due to random salt")
	}
}
