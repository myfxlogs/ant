//go:build integration

package repository

// TRADE-RECORDS-DUP-1 T3/T4: VerifyChain dedup_log exemption.
//
// T3: a row whose prev_hash points at a dedup-removed row's entry_hash is a
// documented removed chain segment — reported as "deleted_link"
// (informational), not "chain_break".
// T4: the exemption is discriminative — remove the dedup_log row and the
// same gap surfaces as "chain_break" again.
//
// Chain rows are inserted via direct SQL with pinned (negative) seq and
// self-computed hashes, so the account chain is deterministic regardless of
// the shared database's global chain tail. The dedup deletion under test
// goes through SET LOCAL session_replication_role='replica' — the same path
// the migration uses (design: the step under test must not tamper with
// triggers; trigger DISABLE is only used by the pre-existing cleanup helper).

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type dedupChainFixture struct {
	ids       []uuid.UUID
	seqs      []int64
	hashes    [][]byte
	userID    uuid.UUID
	accountID uuid.UUID
}

// seedDedupChain inserts the deterministic A→B→C chain for a fresh test
// account and returns row ids, seqs and entry hashes.
func seedDedupChain(t *testing.T, pool *pgxpool.Pool, userID, accountID uuid.UUID) dedupChainFixture {
	t.Helper()
	ctx := context.Background()
	fx := dedupChainFixture{userID: userID, accountID: accountID}
	base := time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		seq := int64(i) - 900002 // ascending in chain order (VerifyChain walks seq ASC)
		ticket := int64(9100 + i)
		openTime := base.Add(time.Duration(i) * time.Hour)
		closeTime := openTime.Add(30 * time.Minute)
		var prev []byte
		if i > 0 {
			prev = fx.hashes[i-1]
		}
		entry := computeTradeEntryHash(prev, seq, accountID, ticket, "EURUSD",
			"0.1000", "1.10000000", "1.10500000", "50.0000",
			[2]int64{openTime.UnixMilli(), closeTime.UnixMilli()})
		id := uuid.New()
		if _, err := pool.Exec(ctx, `
			INSERT INTO trade_records (
				id, user_id, account_id, ticket, symbol, order_type, volume,
				open_price, close_price, profit, swap, commission,
				open_time, close_time, stop_loss, take_profit,
				order_comment, magic_number, platform,
				created_at, updated_at, seq, prev_hash, entry_hash
			) OVERRIDING SYSTEM VALUE VALUES (
				$1, $2, $3, $4, 'EURUSD', 'buy', 0.1000,
				1.1000, 1.1050, 50.0000, 0, 0,
				$5, $6, 1.0950, 1.1100,
				'dedup-test', 1, 'mt4',
				$5, $6, $7, $8, $9
			)`,
			id, userID, accountID, ticket, openTime, closeTime, seq, prev, entry,
		); err != nil {
			t.Fatalf("seed chain row %d: %v", i, err)
		}
		fx.ids = append(fx.ids, id)
		fx.seqs = append(fx.seqs, seq)
		fx.hashes = append(fx.hashes, entry)
	}
	return fx
}

// setupDedupVerifyTest creates the FK parents, provisions the dedup-log
// harness table when the database doesn't have it yet (self-sufficiency on
// pre-281 databases; IF NOT EXISTS semantics via the regclass probe keep an
// already-applied migration's table untouched), and registers cleanup.
func setupDedupVerifyTest(t *testing.T, pool *pgxpool.Pool) func(t *testing.T) dedupChainFixture {
	t.Helper()
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, status) VALUES ($1, $2, 'test', 'active') ON CONFLICT DO NOTHING`,
		userID, "dedup-verify-"+userID.String()[:8]+"@test.local"); err != nil {
		t.Skipf("skipping: cannot insert test user: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO mt_accounts (id, user_id, mt_type, broker_host, login, account_status) VALUES ($1, $2, 'mt4', 'test', '12345', 'disconnected') ON CONFLICT DO NOTHING`,
		accountID, userID); err != nil {
		t.Skipf("skipping: cannot insert test account: %v", err)
	}
	var logTableExists bool
	if err := pool.QueryRow(ctx,
		`SELECT to_regclass('trade_record_dedup_log') IS NOT NULL`).Scan(&logTableExists); err != nil {
		t.Skipf("skipping: cannot probe dedup log table: %v", err)
	}
	harnessCreated := false
	if !logTableExists {
		// Test harness subset — VerifyChain only reads entry_hash; the real
		// DDL (full-row snapshot columns) lives in migration 281.
		if _, err := pool.Exec(ctx, `
			CREATE TABLE trade_record_dedup_log (
				log_id     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				removed_id UUID NOT NULL,
				kept_id    UUID,
				dup_class  TEXT NOT NULL,
				removed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
				removed_by TEXT NOT NULL DEFAULT 'migration_281',
				entry_hash BYTEA
			)`); err != nil {
			t.Skipf("skipping: cannot create dedup log harness: %v", err)
		}
		harnessCreated = true
	}
	t.Cleanup(func() {
		cleanupTestTradeRecords(t, pool, userID, accountID)
		pool.Exec(ctx, `DELETE FROM trade_record_dedup_log WHERE removed_by = 'dedup-test'`)
		if harnessCreated {
			pool.Exec(ctx, `DROP TABLE trade_record_dedup_log`)
		}
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE id = $1`, accountID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})
	return func(t *testing.T) dedupChainFixture {
		return seedDedupChain(t, pool, userID, accountID)
	}
}

// deleteViaReplicaBypass removes the middle chain row and archives it in the
// dedup log inside one transaction using the migration's own bypass path.
func deleteViaReplicaBypass(t *testing.T, pool *pgxpool.Pool, fx dedupChainFixture, dupClass string) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := tx.Exec(ctx, `SET LOCAL session_replication_role = 'replica'`); err != nil {
		t.Fatalf("set replica role: %v", err)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM trade_records WHERE id = $1`, fx.ids[1]); err != nil {
		t.Fatalf("dedup delete: %v", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO trade_record_dedup_log (removed_id, kept_id, dup_class, removed_by, entry_hash)
		 VALUES ($1, $2, $3, 'dedup-test', $4)`,
		fx.ids[1], fx.ids[0], dupClass, fx.hashes[1]); err != nil {
		t.Fatalf("archive dedup row: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

// T3: deleted_link exemption — archived removal is not a chain break.
func TestVerifyChain_DeletedLinkExemption_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := NewTradeRecordRepository(pool)
	seed := setupDedupVerifyTest(t, pool)

	fx := seed(t)
	if breaks, err := repo.VerifyChain(context.Background(), fx.userID, fx.accountID); err != nil {
		t.Fatalf("VerifyChain intact: %v", err)
	} else if len(breaks) != 0 {
		t.Fatalf("intact chain produced breaks: %+v", breaks)
	}

	deleteViaReplicaBypass(t, pool, fx, "epoch_zero")

	breaks, err := repo.VerifyChain(context.Background(), fx.userID, fx.accountID)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	for _, b := range breaks {
		if b.Type == "chain_break" {
			t.Fatalf("archived removal surfaced as chain_break: %+v", breaks)
		}
	}
	var deletedLinks int
	for _, b := range breaks {
		if b.Type == "deleted_link" && b.Seq == fx.seqs[2] {
			deletedLinks++
		}
	}
	if deletedLinks != 1 {
		t.Fatalf("want exactly 1 deleted_link at seq %d (the row after the removal), got %d: %+v",
			fx.seqs[2], deletedLinks, breaks)
	}
}

// T4: exemption is discriminative — without the log entry the gap is a
// chain_break again. Mutation M3 (drop the exemption branch) → this test RED.
func TestVerifyChain_ExemptionDiscriminative_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := NewTradeRecordRepository(pool)
	seed := setupDedupVerifyTest(t, pool)

	fx := seed(t)
	deleteViaReplicaBypass(t, pool, fx, "epoch_zero")

	// Remove the archival proof: the gap is now undocumented tampering.
	if _, err := pool.Exec(context.Background(),
		`DELETE FROM trade_record_dedup_log WHERE removed_by = 'dedup-test'`); err != nil {
		t.Fatalf("purge dedup log: %v", err)
	}

	breaks, err := repo.VerifyChain(context.Background(), fx.userID, fx.accountID)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	found := false
	for _, b := range breaks {
		if b.Type == "chain_break" && b.Seq == fx.seqs[2] {
			found = true
		}
	}
	if !found {
		t.Fatalf("want chain_break at seq %d after log purge, got: %+v", fx.seqs[2], breaks)
	}
}
