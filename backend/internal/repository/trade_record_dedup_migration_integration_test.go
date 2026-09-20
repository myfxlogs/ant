//go:build integration

package repository

// TRADE-RECORDS-DUP-1 T5/T6: migration 281 up/down against a real database.
//
// T5 runs the actual up migration file (multi-statement, one transaction —
// the docker-entrypoint runner shape), asserts the archive-then-delete
// coupling on the seeded duplicate classes, then the down migration must
// restore every removed row with its original chain material
// (seq/prev_hash/entry_hash). Delete counts are asserted RELATIVELY: the
// expected values are measured with the migration's own predicates right
// before it runs (the database may already contain production-scale
// duplicates — on the 2026-09-20 production copy that is 879 + 3024 + 2
// seeded).
//
// T6 re-runs a full up→down→up cycle and asserts identical delete counts on
// the restored data (deterministic re-application), and that VerifyChain
// still works after down dropped the log table (missing-table fallback, not
// an error).
//
// Guard rails: if trade_record_dedup_log already exists at start (migration
// applied, or a crashed earlier run left it), the test skips rather than
// mutating an applied state; if a run fails after up applied, cleanup
// re-runs the down file best-effort so the database ends as found.

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dedupUpPath   = "../../migrations/281_trade_records_dedup.up.sql"
	dedupDownPath = "../../migrations/281_trade_records_dedup.down.sql"
)

// execMigrationFile executes a whole migration file as one multi-statement
// string over the simple protocol (implicit transaction, SET LOCAL scoped)
// and returns one result per statement, in file order.
func execMigrationFile(t *testing.T, pool *pgxpool.Pool, path string) []*pgconn.Result {
	t.Helper()
	results, err := execMigrationFileQuiet(context.Background(), pool, path)
	if err != nil {
		t.Fatalf("migration %s: %v", path, err)
	}
	return results
}

func execMigrationFileQuiet(ctx context.Context, pool *pgxpool.Pool, path string) ([]*pgconn.Result, error) {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	return conn.Conn().PgConn().Exec(ctx, string(sqlBytes)).ReadAll()
}

func dedupLogTableExists(t *testing.T, pool *pgxpool.Pool) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(),
		`SELECT to_regclass('trade_record_dedup_log') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatalf("probe dedup log table: %v", err)
	}
	return exists
}

type dedupSeedRow struct {
	id        uuid.UUID
	ticket    int64
	closeTime time.Time
	seq       int64
	prevHash  []byte
	entryHash []byte
}

// setupDedupMigrationTest creates the FK parents, skips when the dedup-log
// table already exists (migration applied / leftover), and registers cleanup
// that leaves the database exactly as found: best-effort down-restore, row
// purge, user + account removal.
func setupDedupMigrationTest(t *testing.T, pool *pgxpool.Pool) (userID, accountID uuid.UUID) {
	t.Helper()
	if dedupLogTableExists(t, pool) {
		t.Skip("trade_record_dedup_log already exists — migration 281 applied or leftover; run on a pre-281 database")
	}
	ctx := context.Background()
	userID, accountID = uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, status) VALUES ($1, $2, 'test', 'active') ON CONFLICT DO NOTHING`,
		userID, "dedup-mig-"+userID.String()[:8]+"@test.local"); err != nil {
		t.Skipf("skipping: cannot insert test user: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO mt_accounts (id, user_id, mt_type, broker_host, login, account_status) VALUES ($1, $2, 'mt4', 'test', '12345', 'disconnected') ON CONFLICT DO NOTHING`,
		accountID, userID); err != nil {
		t.Skipf("skipping: cannot insert test account: %v", err)
	}
	t.Cleanup(func() {
		cctx := context.Background()
		if dedupLogTableExists(t, pool) {
			// A failed run left up applied — best-effort restore.
			if _, err := execMigrationFileQuiet(cctx, pool, dedupDownPath); err != nil {
				t.Logf("cleanup: down restore failed: %v", err)
			}
		}
		cleanupTestTradeRecords(t, pool, userID, accountID)
		pool.Exec(cctx, `DELETE FROM mt_accounts WHERE id = $1`, accountID)
		pool.Exec(cctx, `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID, accountID
}

// seedDedupClasses inserts, for a fresh account: the UTC row of a twin pair
// (ticket 9201 @ T), its +8h CST twin (identical amounts, both timestamps
// +8h), and an epoch-zero phantom row (ticket 9202). Rows are hash-chained
// with pinned negative seqs; hashes use the exact numeric scale VerifyChain
// rescans (volume (10,4), prices (18,8), profit (18,4)).
func seedDedupClasses(t *testing.T, pool *pgxpool.Pool, userID, accountID uuid.UUID) (utcRow, cstRow, epochRow dedupSeedRow) {
	t.Helper()
	ctx := context.Background()
	base := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC)
	// Seqs ascending in chain order — VerifyChain walks ORDER BY seq ASC.
	rows := []dedupSeedRow{
		{ticket: 9201, closeTime: base, seq: -920002},
		{ticket: 9201, closeTime: base.Add(8 * time.Hour), seq: -920001},
		{ticket: 9202, closeTime: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), seq: -920000},
	}
	var prev []byte
	for i := range rows {
		r := &rows[i]
		r.id = uuid.New()
		openTime := r.closeTime.Add(-1 * time.Hour)
		r.prevHash = prev
		r.entryHash = computeTradeEntryHash(prev, r.seq, accountID, r.ticket, "EURUSD",
			"0.1000", "1.10000000", "1.10500000", "50.0000",
			[2]int64{openTime.UnixMilli(), r.closeTime.UnixMilli()})
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
				'dedup-mig-test', 1, 'mt4',
				$7, $7, $8, $9, $10
			)`,
			r.id, userID, accountID, r.ticket, openTime, r.closeTime, base.Add(48*time.Hour), r.seq, prev, r.entryHash,
		); err != nil {
			t.Fatalf("seed dedup row %s: %v", r.id, err)
		}
		prev = r.entryHash
	}
	return rows[0], rows[1], rows[2]
}

// expectedDedupCounts measures the migration's own predicates — including the
// just-seeded rows — so delete counts are asserted relatively.
func expectedDedupCounts(t *testing.T, pool *pgxpool.Pool) (d1, d2 int64) {
	t.Helper()
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM trade_records a JOIN trade_records b
		  ON b.account_id = a.account_id AND b.ticket = a.ticket
		 AND b.close_time = a.close_time + interval '8 hours'
		 AND b.open_time  = a.open_time  + interval '8 hours'
		 AND b.volume = a.volume AND b.open_price = a.open_price
		 AND b.close_price = a.close_price AND b.profit = a.profit
	`).Scan(&d1); err != nil {
		t.Fatalf("count D1: %v", err)
	}
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM trade_records WHERE close_time = TIMESTAMP '1970-01-01'`,
	).Scan(&d2); err != nil {
		t.Fatalf("count D2: %v", err)
	}
	return d1, d2
}

func assertDeleteTag(t *testing.T, tag pgconn.CommandTag, label string, want int64) {
	t.Helper()
	if got := tag.String(); got != fmt.Sprintf("DELETE %d", want) {
		t.Fatalf("%s deleted %q, want DELETE %d", label, got, want)
	}
}

func assertUpResults(t *testing.T, up []*pgconn.Result, d1, d2 int64) {
	t.Helper()
	if len(up) != 4 {
		t.Fatalf("up file must yield 4 statements (SET/CREATE/DELETE/DELETE), got %d", len(up))
	}
	assertDeleteTag(t, up[2].CommandTag, "D1 (cst_utc_8h)", d1)
	assertDeleteTag(t, up[3].CommandTag, "D2 (epoch_zero)", d2)
}

func assertDownResults(t *testing.T, down []*pgconn.Result, restored int64) {
	t.Helper()
	if len(down) != 3 {
		t.Fatalf("down file must yield 3 statements (SET/INSERT/DROP), got %d", len(down))
	}
	if got := down[1].CommandTag.String(); got != fmt.Sprintf("INSERT 0 %d", restored) {
		t.Fatalf("down restored %q, want INSERT 0 %d", got, restored)
	}
}

// T5: up archives-then-deletes the seeded classes; down restores them with
// original chain material.
func TestMigration281_UpArchiveThenDownRestore_Integration(t *testing.T) {
	pool := getTestPool(t)
	userID, accountID := setupDedupMigrationTest(t, pool)
	ctx := context.Background()

	utcRow, cstRow, epochRow := seedDedupClasses(t, pool, userID, accountID)
	d1Exp, d2Exp := expectedDedupCounts(t, pool)

	assertUpResults(t, execMigrationFile(t, pool, dedupUpPath), d1Exp, d2Exp)

	// Log rows: kept_id points at the UTC twin; epoch row archived unkept.
	// kept_id is *uuid.UUID — a NULL must scan as nil, never a stale value.
	var keptID *uuid.UUID
	var dupClass string
	if err := pool.QueryRow(ctx,
		`SELECT kept_id, dup_class FROM trade_record_dedup_log WHERE removed_id = $1`, cstRow.id,
	).Scan(&keptID, &dupClass); err != nil {
		t.Fatalf("D1 log row missing: %v", err)
	}
	if keptID == nil || *keptID != utcRow.id || dupClass != "cst_utc_8h" {
		t.Fatalf("D1 log: kept_id=%v class=%s, want UTC twin %s/cst_utc_8h", keptID, dupClass, utcRow.id)
	}
	if err := pool.QueryRow(ctx,
		`SELECT kept_id, dup_class FROM trade_record_dedup_log WHERE removed_id = $1`, epochRow.id,
	).Scan(&keptID, &dupClass); err != nil {
		t.Fatalf("D2 log row missing: %v", err)
	}
	if keptID != nil || dupClass != "epoch_zero" {
		t.Fatalf("D2 log: kept_id=%v class=%s, want NULL/epoch_zero", keptID, dupClass)
	}

	// The UTC anchor survives.
	var alive int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM trade_records WHERE id = $1`, utcRow.id).Scan(&alive); err != nil || alive != 1 {
		t.Fatalf("UTC anchor must survive dedup (alive=%d err=%v)", alive, err)
	}

	assertDownResults(t, execMigrationFile(t, pool, dedupDownPath), d1Exp+d2Exp)

	// Spot-check chain material: every seeded row is back at its original
	// seq/hash values.
	for _, r := range []dedupSeedRow{utcRow, cstRow, epochRow} {
		var seq int64
		var prevHash, entryHash []byte
		if err := pool.QueryRow(ctx,
			`SELECT seq, prev_hash, entry_hash FROM trade_records WHERE id = $1`, r.id,
		).Scan(&seq, &prevHash, &entryHash); err != nil {
			t.Fatalf("restored row %s missing: %v", r.id, err)
		}
		if seq != r.seq || string(prevHash) != string(r.prevHash) || string(entryHash) != string(r.entryHash) {
			t.Fatalf("restored row %s drifted: seq=%d want=%d prev ok=%v entry ok=%v",
				r.id, seq, r.seq, string(prevHash) == string(r.prevHash), string(entryHash) == string(r.entryHash))
		}
	}
	var tableGone bool
	if err := pool.QueryRow(ctx,
		`SELECT to_regclass('trade_record_dedup_log') IS NULL`).Scan(&tableGone); err != nil || !tableGone {
		t.Fatalf("down must drop the log table (tableGone=%v err=%v)", tableGone, err)
	}
}

// T6: the delete CTEs re-apply deterministically on restored data, and
// VerifyChain keeps working after down dropped the log table.
// Mutation M4 (D1 signature loosened) → the production-copy count check
// REDs; this test pins the seeded-class equivalent (still exact, never more).
func TestMigration281_RerunDeterministic_VerifyChainAfterDown_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := NewTradeRecordRepository(pool)
	userID, accountID := setupDedupMigrationTest(t, pool)

	seedDedupClasses(t, pool, userID, accountID)
	d1Exp, d2Exp := expectedDedupCounts(t, pool)

	up1 := execMigrationFile(t, pool, dedupUpPath)
	execMigrationFile(t, pool, dedupDownPath)
	up2 := execMigrationFile(t, pool, dedupUpPath)
	execMigrationFile(t, pool, dedupDownPath)

	assertUpResults(t, up1, d1Exp, d2Exp)
	assertUpResults(t, up2, d1Exp, d2Exp)

	// Log table dropped: VerifyChain must fall back to strict mode, not error.
	breaks, err := repo.VerifyChain(context.Background(), userID, accountID)
	if err != nil {
		t.Fatalf("VerifyChain after down: %v", err)
	}
	if len(breaks) != 0 {
		t.Fatalf("restored chain must verify clean, got: %+v", breaks)
	}
}
