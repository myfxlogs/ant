package main

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

// TestBuildChallenge verifies coldsign's buildChallenge matches the online server's
// buildWithdrawalChallenge. This is the critical consistency test (R11).
func TestBuildChallenge(t *testing.T) {
	amount := "100.000000"
	dest := "TXYZ1234567890"
	nonce := int64(1700000000000000000)
	userID := "user-abc-123"

	// coldsign side: nonce passed as string.
	got := buildChallenge(amount, dest, fmt.Sprintf("%d", nonce), userID)

	// Manually reconstruct expected hash (same as server-side buildWithdrawalChallenge).
	h := sha256.New()
	h.Write([]byte(amount))
	h.Write([]byte("|"))
	h.Write([]byte(dest))
	h.Write([]byte("|"))
	h.Write([]byte(fmt.Sprintf("%d", nonce)))
	h.Write([]byte("|"))
	h.Write([]byte(userID))
	want := h.Sum(nil)

	if string(got) != string(want) {
		t.Fatalf("coldsign challenge mismatch:\n got  %x\n want %x", got, want)
	}
}

// TestCheckWithdrawalLimit verifies the per-withdrawal limit check.
func TestCheckWithdrawalLimit(t *testing.T) {
	tests := []struct {
		amount    string
		maxAmount string
		wantErr   bool
	}{
		{"100", "1000", false},
		{"1000", "1000", false},
		{"1000.000001", "1000", true},
		{"0", "1000", false},
		{"500.5", "500.5", false},
		{"500.6", "500.5", true},
	}

	for _, tt := range tests {
		err := checkWithdrawalLimit(tt.amount, tt.maxAmount)
		if (err != nil) != tt.wantErr {
			t.Errorf("checkWithdrawalLimit(%q, %q) = %v, wantErr=%v",
				tt.amount, tt.maxAmount, err, tt.wantErr)
		}
	}
}

// TestCheckWithdrawalLimitInvalidInputs verifies error handling for invalid decimal strings.
func TestCheckWithdrawalLimitInvalidInputs(t *testing.T) {
	tests := []struct {
		amount    string
		maxAmount string
		wantErr   bool
	}{
		{"", "1000", true},
		{"abc", "1000", true},
		{"100", "abc", true},
		{"100", "", true},
	}

	for _, tt := range tests {
		err := checkWithdrawalLimit(tt.amount, tt.maxAmount)
		if (err != nil) != tt.wantErr {
			t.Errorf("checkWithdrawalLimit(%q, %q) = %v, wantErr=%v",
				tt.amount, tt.maxAmount, err, tt.wantErr)
		}
	}
}

// TestLoadCredentialDBInvalidPath verifies error handling for missing file.
func TestLoadCredentialDBInvalidPath(t *testing.T) {
	_, err := LoadCredentialDB("/nonexistent/path/file.bin")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

// TestLoadCredentialDBEmptyPath verifies error for empty path.
func TestLoadCredentialDBEmptyPath(t *testing.T) {
	_, err := LoadCredentialDB("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}
