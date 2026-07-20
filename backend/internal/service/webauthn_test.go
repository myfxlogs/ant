package service

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

// TestBuildWithdrawalChallenge verifies the challenge hash is deterministic
// and matches the coldsign-side buildChallenge exactly.
func TestBuildWithdrawalChallenge(t *testing.T) {
	amount := "100.000000"
	dest := "TXYZ1234567890"
	nonce := int64(1700000000000000000)
	userID := "user-abc-123"

	got := buildWithdrawalChallenge(amount, dest, nonce, userID)

	// Manually reconstruct expected hash.
	h := sha256.New()
	h.Write([]byte(amount))
	h.Write([]byte("|"))
	h.Write([]byte(dest))
	h.Write([]byte("|"))
	h.Write([]byte(fmt.Sprintf("%d", nonce)))
	h.Write([]byte("|"))
	h.Write([]byte(userID))
	want := h.Sum(nil)

	if len(got) != 32 {
		t.Fatalf("expected 32-byte hash, got %d", len(got))
	}
	if string(got) != string(want) {
		t.Fatalf("challenge hash mismatch:\n got  %x\n want %x", got, want)
	}
}

// TestBuildWithdrawalChallengeDeterministic ensures same inputs → same output.
func TestBuildWithdrawalChallengeDeterministic(t *testing.T) {
	a := buildWithdrawalChallenge("1.5", "TABC", 123, "user1")
	b := buildWithdrawalChallenge("1.5", "TABC", 123, "user1")
	if string(a) != string(b) {
		t.Fatal("same inputs should produce same hash")
	}
}

// TestBuildWithdrawalChallengeDifferentInputs ensures different inputs → different output.
func TestBuildWithdrawalChallengeDifferentInputs(t *testing.T) {
	a := buildWithdrawalChallenge("1.5", "TABC", 123, "user1")
	b := buildWithdrawalChallenge("1.5", "TABC", 123, "user2")
	if string(a) == string(b) {
		t.Fatal("different user_id should produce different hash")
	}
}
