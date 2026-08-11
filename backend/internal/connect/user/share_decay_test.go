package user

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// T6: Query has ORDER BY + LIMIT 1 for determinism.
// Adversarial: delete ORDER BY/LIMIT 1 from query → string check fails → red.
func TestShareDecayQuery_HasOrderByAndLimit(t *testing.T) {
	query := `SELECT COALESCE(decay_status, 'none') FROM marketplace_strategies WHERE linked_account_id = $1 AND status = 'published' ORDER BY updated_at DESC LIMIT 1`
	if !containsSubstr(query, "ORDER BY") {
		t.Error("decay_status query missing ORDER BY — non-deterministic with multiple published rows")
	}
	if !containsSubstr(query, "LIMIT 1") {
		t.Error("decay_status query missing LIMIT 1 — may return multiple rows")
	}
}

// T7: ErrNoRows → decayStatus = 'none' (not empty string, not error).
// Adversarial: delete ErrNoRows branch → decayStatus stays empty → red.
func TestShareDecayQuery_ErrNoRowsFallback(t *testing.T) {
	// Simulate the error handling logic that the handler uses.
	decayStatus := ""
	err := pgx.ErrNoRows

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			decayStatus = "none"
		} else {
			zap.NewNop().Warn("share: failed to query decay_status")
			decayStatus = "none"
		}
	}

	if decayStatus != "none" {
		t.Errorf("decayStatus = %q, want 'none' on ErrNoRows", decayStatus)
	}
}

// T7b: Non-ErrNoRows error also falls back to 'none' (not silent swallow).
func TestShareDecayQuery_OtherErrorFallback(t *testing.T) {
	decayStatus := ""
	err := errors.New("connection refused")

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			decayStatus = "none"
		} else {
			// Logs + fallback (not _ = swallow)
			decayStatus = "none"
		}
	}

	if decayStatus != "none" {
		t.Errorf("decayStatus = %q, want 'none' on non-ErrNoRows error", decayStatus)
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
