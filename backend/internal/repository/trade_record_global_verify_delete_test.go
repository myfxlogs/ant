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

	"github.com/google/uuid"
)

// T3 — unarchived mid-chain deletion: the follower loses its prev and the
// break surfaces.
func TestVerifyGlobalChain_UnarchivedDelete_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := newGlobalVerifyRepo(t)
	ctx := context.Background()
	acct := newGVAccount(t, pool, "gv-t3")

	var ids []uuid.UUID
	for i := int64(0); i < 3; i++ {
		r := makeTestTradeRecord(acct.userID, acct.accountID, 7201+i)
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
		ids = append(ids, r.ID)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SET LOCAL session_replication_role = 'replica'`); err != nil {
		t.Fatalf("set replica: %v", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM trade_records WHERE id = $1`, ids[1]); err != nil {
		t.Fatalf("delete middle: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	breaks, err := repo.VerifyChain(ctx, acct.userID, acct.accountID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if countBreaks(breaks, "chain_break") != 1 {
		t.Fatalf("want exactly 1 chain_break after unarchived delete, got %+v", breaks)
	}
}

// T4 — archived deletion is self-healing: archive the removed row with the
// FULL migration-281 column set and the union walk closes the gap
// (supersedes the deleted_link exemption tests).

// T4 — archived deletion is self-healing: archive the removed row with the
// FULL migration-281 column set and the union walk closes the gap
// (supersedes the deleted_link exemption tests).
func TestVerifyGlobalChain_ArchivedDeleteSelfHeals_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := newGlobalVerifyRepo(t)
	ctx := context.Background()
	acct := newGVAccount(t, pool, "gv-t4")

	if !dedupLogTableExists(t, pool) {
		// Harness: pre-281 database — create the log with the migration's
		// original full-column DDL (a subset-column harness would not satisfy
		// the union query).
		if _, err := pool.Exec(ctx, migration281LogDDL(t)); err != nil {
			t.Fatalf("harness log DDL: %v", err)
		}
	}

	var ids []uuid.UUID
	for i := int64(0); i < 3; i++ {
		r := makeTestTradeRecord(acct.userID, acct.accountID, 7301+i)
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
		ids = append(ids, r.ID)
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
		`INSERT INTO trade_record_dedup_log (removed_id, kept_id, dup_class, removed_by,
			id, account_id, ticket, symbol, order_type, volume, open_price, close_price,
			profit, swap, commission, open_time, close_time, stop_loss, take_profit,
			order_comment, magic_number, platform, created_at, updated_at,
			schedule_id, user_id, seq, prev_hash, entry_hash)
		 SELECT id, NULL, 'epoch_zero', 'gv-t4',
			id, account_id, ticket, symbol, order_type, volume, open_price, close_price,
			profit, swap, commission, open_time, close_time, stop_loss, take_profit,
			order_comment, magic_number, platform, created_at, updated_at,
			schedule_id, user_id, seq, prev_hash, entry_hash
		 FROM trade_records WHERE id = $1`, ids[1]); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM trade_records WHERE id = $1`, ids[1]); err != nil {
		t.Fatalf("delete middle: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	breaks, err := repo.VerifyChain(ctx, acct.userID, acct.accountID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(breaks) != 0 {
		t.Fatalf("archived deletion must leave the chain whole, got: %+v", breaks)
	}
}

// T5 — per-account filtering: a break in account B is invisible to
// VerifyChain(acctA) and visible to VerifyChain(acctB).
