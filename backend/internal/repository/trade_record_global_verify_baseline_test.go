//go:build integration

package repository

// VERIFY-CHAIN-SEMANTIC-1 T1-T9: global union chain verification.
//
// The chain is global (writes chain to the global tail); the archived
// dedup_log rows are chain members whose full-column snapshots make deleted
// links whole. T1 pins the production-clone baseline (chain_break hard 0,
// hash_mismatch == the measured legacy-unconfirmable count — count drift is
// a regression signal). T2-T9 are self-contained: seeds chain from the real
// global tail via the canonical write path (or explicit SQL where the test's
// discriminating power requires a hand-built hash), and cleanup restores the
// database exactly.
//
// Supersedes trade_record_dedup_verify_integration_test.go (the deleted_link
// exemption is replaced by union correctness).

import (
	"context"
	"testing"

	"alphaforge/internal/model"
)

// T1 — production-clone baseline: the union walk is self-consistent
// (chain_break hard 0) and the legacy-unconfirmable count is exactly the
// measured baseline; drift is a regression signal.
func TestVerifyGlobalChain_ProductionBaseline_Integration(t *testing.T) {
	repo := newGlobalVerifyRepo(t)

	breaks, err := repo.VerifyGlobalChain(context.Background())
	if err != nil {
		t.Fatalf("VerifyGlobalChain: %v", err)
	}
	if got := countBreaks(breaks, "chain_break"); got != 0 {
		t.Fatalf("chain_break = %d, want 0 (union walk must be self-consistent): %s",
			got, firstBreaks(breaks, 3))
	}
	if got := countBreaks(breaks, "hash_mismatch"); got != wantLegacyHashMismatch {
		t.Fatalf("hash_mismatch = %d, want %d (legacy baseline drift — regression signal)",
			got, wantLegacyHashMismatch)
	}
	t.Logf("baseline: hash_mismatch=%d unhashed=%d",
		countBreaks(breaks, "hash_mismatch"), countBreaks(breaks, "unhashed"))
}

// T2 — tamper discrimination on canonical rows: profit update without hash
// maintenance → hash_mismatch; prev_hash tamper (replica bypass) →
// chain_break.

// T2 — tamper discrimination on canonical rows: profit update without hash
// maintenance → hash_mismatch; prev_hash tamper (replica bypass) →
// chain_break.
func TestVerifyGlobalChain_TamperDiscrimination_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := newGlobalVerifyRepo(t)
	ctx := context.Background()
	acct := newGVAccount(t, pool, "gv-t2")

	r1 := makeTestTradeRecord(acct.userID, acct.accountID, 7101)
	r2 := makeTestTradeRecord(acct.userID, acct.accountID, 7102)
	for _, r := range []*model.TradeRecord{r1, r2} {
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// Sanity: canonical rows verify.
	breaks, err := repo.VerifyChain(ctx, acct.userID, acct.accountID)
	if err != nil || len(breaks) != 0 {
		t.Fatalf("pre-tamper verify: breaks=%+v err=%v", breaks, err)
	}

	// Tamper profit (hash-covered but not trigger-protected).
	if _, err := pool.Exec(ctx,
		`UPDATE trade_records SET profit = profit + 1 WHERE id = $1`, r2.ID); err != nil {
		t.Fatalf("tamper profit: %v", err)
	}
	breaks, err = repo.VerifyChain(ctx, acct.userID, acct.accountID)
	if err != nil {
		t.Fatalf("verify after profit tamper: %v", err)
	}
	if countBreaks(breaks, "hash_mismatch") != 1 {
		t.Fatalf("want exactly 1 hash_mismatch after profit tamper, got %+v", breaks)
	}

	// Restore profit, then tamper prev_hash (hash-chain field: replica bypass).
	if _, err := pool.Exec(ctx,
		`UPDATE trade_records SET profit = $2 WHERE id = $1`, r2.ID, r2.Profit); err != nil {
		t.Fatalf("restore profit: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SET LOCAL session_replication_role = 'replica'`); err != nil {
		t.Fatalf("set replica: %v", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE trade_records SET prev_hash = decode('ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff', 'hex') WHERE id = $1`,
		r2.ID); err != nil {
		t.Fatalf("tamper prev_hash: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	breaks, err = repo.VerifyChain(ctx, acct.userID, acct.accountID)
	if err != nil {
		t.Fatalf("verify after prev tamper: %v", err)
	}
	if countBreaks(breaks, "chain_break") != 1 {
		t.Fatalf("want exactly 1 chain_break after prev_hash tamper, got %+v", breaks)
	}
}

// T3 — unarchived mid-chain deletion: the follower loses its prev and the
// break surfaces.
