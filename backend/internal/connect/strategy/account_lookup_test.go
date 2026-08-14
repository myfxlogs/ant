package strategy

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TestACCTLOOKUP_SelectedAccountNotOverridden verifies that resolveModeAndAccount
// does NOT override an already-selected account with accountLookup.
//
// Adversarial proof: Revert to old code (always call accountLookup in live mode,
// unconditionally overriding cfg.AccountID) → cfg.AccountID becomes "acct-B"
// (the disconnected fallback) instead of "acct-A" (user's choice) → RED.
func TestACCTLOOKUP_SelectedAccountNotOverridden(t *testing.T) {
	t.Parallel()

	// accountLookup returns "acct-B" (oldest, disconnected — should NOT be chosen
	// when user already selected "acct-A").
	lookupCalled := false
	srv := &StrategyExecutionServer{
		log: zap.NewNop(),
		accountLookup: func(_ context.Context, _ string) string {
			lookupCalled = true
			return "acct-B"
		},
	}

	uid := uuid.New()
	cfg := LiveStrategyConfig{
		AccountID: "acct-A", // user selected this from the panel
		Mode:      modeLive,
	}

	if err := srv.resolveModeAndAccount(context.Background(), uid, modeLive, &cfg); err != nil {
		t.Fatalf("resolveModeAndAccount failed: %v", err)
	}

	if lookupCalled {
		t.Fatalf("ACCT-LOOKUP: accountLookup was called despite cfg.AccountID already set — " +
			"RED: old code unconditionally calls accountLookup and overrides user's selection")
	}

	if cfg.AccountID != "acct-A" {
		t.Fatalf("ACCT-LOOKUP: AccountID = %q, want %q (user's selection must not be overridden)",
			cfg.AccountID, "acct-A")
	}

	if cfg.DataSourceAccountID != "acct-A" {
		t.Fatalf("ACCT-LOOKUP: DataSourceAccountID = %q, want %q (bar source must follow selected account)",
			cfg.DataSourceAccountID, "acct-A")
	}
}

// TestACCTLOOKUP_FallbackWhenAccountIDEmpty verifies that accountLookup is used
// as fallback when cfg.AccountID is empty.
func TestACCTLOOKUP_FallbackWhenAccountIDEmpty(t *testing.T) {
	t.Parallel()

	srv := &StrategyExecutionServer{
		log: zap.NewNop(),
		accountLookup: func(_ context.Context, _ string) string {
			return "acct-fallback"
		},
	}

	uid := uuid.New()
	cfg := LiveStrategyConfig{
		AccountID: "", // no account selected — fallback to accountLookup
		Mode:      modeLive,
	}

	if err := srv.resolveModeAndAccount(context.Background(), uid, modeLive, &cfg); err != nil {
		t.Fatalf("resolveModeAndAccount failed: %v", err)
	}

	if cfg.AccountID != "acct-fallback" {
		t.Fatalf("AccountID = %q, want %q (fallback)", cfg.AccountID, "acct-fallback")
	}
	if cfg.DataSourceAccountID != "acct-fallback" {
		t.Fatalf("DataSourceAccountID = %q, want %q (fallback)", cfg.DataSourceAccountID, "acct-fallback")
	}
}

// TestACCTLOOKUP_PaperModeSelectedAccountNotOverridden verifies that paper mode
// also respects the user's selected account.
func TestACCTLOOKUP_PaperModeSelectedAccountNotOverridden(t *testing.T) {
	t.Parallel()

	lookupCalled := false
	srv := &StrategyExecutionServer{
		log: zap.NewNop(),
		accountLookup: func(_ context.Context, _ string) string {
			lookupCalled = true
			return "acct-B"
		},
	}

	uid := uuid.New()
	cfg := LiveStrategyConfig{
		AccountID: "acct-A",
		Mode:      "paper",
	}

	if err := srv.resolveModeAndAccount(context.Background(), uid, "paper", &cfg); err != nil {
		t.Fatalf("resolveModeAndAccount failed: %v", err)
	}

	if lookupCalled {
		t.Fatalf("ACCT-LOOKUP: accountLookup called in paper mode despite AccountID set — RED")
	}

	if cfg.AccountID != "acct-A" {
		t.Fatalf("AccountID = %q, want %q", cfg.AccountID, "acct-A")
	}
	if cfg.DataSourceAccountID != "acct-A" {
		t.Fatalf("DataSourceAccountID = %q, want %q", cfg.DataSourceAccountID, "acct-A")
	}
}
