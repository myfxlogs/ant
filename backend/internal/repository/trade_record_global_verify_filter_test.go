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

// T5 — per-account filtering: a break in account B is invisible to
// VerifyChain(acctA) and visible to VerifyChain(acctB).
func TestVerifyGlobalChain_AccountFilter_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := newGlobalVerifyRepo(t)
	ctx := context.Background()
	acctA := newGVAccount(t, pool, "gv-t5a")
	acctB := newGVAccount(t, pool, "gv-t5b")

	seed := func(acct gvAccount, ticket int64) uuid.UUID {
		r := makeTestTradeRecord(acct.userID, acct.accountID, ticket)
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("seed: %v", err)
		}
		return r.ID
	}
	seed(acctA, 7401)
	midB := seed(acctB, 7402)
	seed(acctB, 7403)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SET LOCAL session_replication_role = 'replica'`); err != nil {
		t.Fatalf("set replica: %v", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM trade_records WHERE id = $1`, midB); err != nil {
		t.Fatalf("delete B middle: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	breaksA, err := repo.VerifyChain(ctx, acctA.userID, acctA.accountID)
	if err != nil {
		t.Fatalf("verify A: %v", err)
	}
	if len(breaksA) != 0 {
		t.Fatalf("acctA must see none of acctB's breaks: %+v", breaksA)
	}
	breaksB, err := repo.VerifyChain(ctx, acctB.userID, acctB.accountID)
	if err != nil {
		t.Fatalf("verify B: %v", err)
	}
	if countBreaks(breaksB, "chain_break") != 1 {
		t.Fatalf("acctB must see exactly 1 chain_break, got %+v", breaksB)
	}
}

// T6 — dropped log table: 42P01 fallback keeps verification alive and
// reports the unhealed breaks as-is; restoring the log heals the walk again.

// T6 — dropped log table: 42P01 fallback keeps verification alive and
// reports the unhealed breaks as-is; restoring the log heals the walk again.
func TestVerifyGlobalChain_MissingLogFallback_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := newGlobalVerifyRepo(t)
	ctx := context.Background()

	if !dedupLogTableExists(t, pool) {
		t.Skip("log table absent — fallback already the steady state on this database")
	}

	// Preserve the archived rows, then drop the table for real (the verifier
	// runs on the pool, so the fallback must survive a real drop, not a
	// transactional one).
	if _, err := pool.Exec(ctx, `CREATE TABLE trade_record_dedup_log_gvbackup AS SELECT * FROM trade_record_dedup_log`); err != nil {
		t.Fatalf("backup: %v", err)
	}
	if _, err := pool.Exec(ctx, `DROP TABLE trade_record_dedup_log`); err != nil {
		t.Fatalf("drop: %v", err)
	}
	t.Cleanup(func() {
		p := getTestPool(t)
		p.Exec(context.Background(), `DROP TABLE IF EXISTS trade_record_dedup_log`)
		if _, err := p.Exec(context.Background(), migration281LogDDL(t)); err != nil {
			t.Logf("cleanup: recreate log DDL: %v", err)
			return
		}
		if _, err := p.Exec(context.Background(),
			`INSERT INTO trade_record_dedup_log (
				log_id, removed_id, kept_id, dup_class, removed_at, removed_by,
				id, account_id, ticket, symbol, order_type, volume, open_price, close_price,
				profit, swap, commission, open_time, close_time, stop_loss, take_profit,
				order_comment, magic_number, platform, created_at, updated_at,
				schedule_id, user_id, seq, prev_hash, entry_hash)
			 OVERRIDING SYSTEM VALUE SELECT * FROM trade_record_dedup_log_gvbackup`); err != nil {
			t.Logf("cleanup: restore log rows: %v", err)
		}
		resetDedupLogIdentity(t, p)
		p.Exec(context.Background(), `DROP TABLE trade_record_dedup_log_gvbackup`)
	})

	breaks, err := repo.VerifyGlobalChain(ctx)
	if err != nil {
		t.Fatalf("VerifyGlobalChain with dropped log must fall back, got error: %v", err)
	}
	if countBreaks(breaks, "chain_break") == 0 {
		t.Fatal("live-only walk after drop must report the unhealed gaps as chain_break")
	}

	// Write-path fallback: appending while dedup_log is absent (pre-migration
	// / post-down steady state) must succeed via the live-only tail read.
	var liveAcct, liveUser uuid.UUID
	if err := pool.QueryRow(ctx,
		`SELECT account_id, user_id FROM trade_records ORDER BY seq DESC LIMIT 1`,
	).Scan(&liveAcct, &liveUser); err != nil {
		t.Fatalf("pick live account: %v", err)
	}
	rec := makeTestTradeRecord(liveUser, liveAcct, 977000001)
	if err := repo.Create(ctx, rec); err != nil {
		t.Fatalf("Create with dropped log must fall back to live tail: %v", err)
	}
	// Remove the fallback-appended row before the restore-heal assertion:
	// it chained onto the live tail, which would read as a break once the
	// archived rows above it rejoin the union walk.
	if _, err := pool.Exec(ctx, `ALTER TABLE trade_records DISABLE TRIGGER prevent_trade_delete`); err != nil {
		t.Fatalf("disable trigger: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM trade_records WHERE ticket = 977000001`); err != nil {
		t.Fatalf("remove fallback row: %v", err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE trade_records ENABLE TRIGGER prevent_trade_delete`); err != nil {
		t.Fatalf("enable trigger: %v", err)
	}

	// Restore and prove the walk self-heals to the baseline.
	if _, err := pool.Exec(ctx, migration281LogDDL(t)); err != nil {
		t.Fatalf("recreate log: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO trade_record_dedup_log (
			log_id, removed_id, kept_id, dup_class, removed_at, removed_by,
			id, account_id, ticket, symbol, order_type, volume, open_price, close_price,
			profit, swap, commission, open_time, close_time, stop_loss, take_profit,
			order_comment, magic_number, platform, created_at, updated_at,
			schedule_id, user_id, seq, prev_hash, entry_hash)
		 OVERRIDING SYSTEM VALUE SELECT * FROM trade_record_dedup_log_gvbackup`); err != nil {
		t.Fatalf("restore log rows: %v", err)
	}
	resetDedupLogIdentity(t, pool)
	breaks, err = repo.VerifyGlobalChain(ctx)
	if err != nil {
		t.Fatalf("verify after restore: %v", err)
	}
	if got := countBreaks(breaks, "chain_break"); got != 0 {
		t.Fatalf("restored log must heal the walk: chain_break=%d", got)
	}
}

// T7 — legacy dual-encoding (D3 load-bearing): a row hashed over the legacy
// `Decimal.String()` forms (volume "0.05", prices "1.1"/"1.105", profit "50")
// is confirmed through the normalized channel while the canonical form alone
// would miss it.
