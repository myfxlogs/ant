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
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"alphaforge/internal/model"
)

// Legacy baseline measured on the 2026-09-20 production clone (design D3):
// rows whose stored hash no candidate encoding reproduces.
const wantLegacyHashMismatch = 4605

func newGlobalVerifyRepo(t *testing.T) *TradeRecordRepository {
	t.Helper()
	return NewTradeRecordRepository(getTestPool(t))
}

type gvAccount struct {
	userID, accountID uuid.UUID
}

// newGVAccount creates the FK parents and registers full cleanup (ledger
// rows, log rows by marker, account, user).
func newGVAccount(t *testing.T, pool *pgxpool.Pool, tag string) gvAccount {
	t.Helper()
	ctx := context.Background()
	a := gvAccount{userID: uuid.New(), accountID: uuid.New()}
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, status) VALUES ($1, $2, 'test', 'active') ON CONFLICT DO NOTHING`,
		a.userID, tag+"-"+a.userID.String()[:8]+"@test.local"); err != nil {
		t.Skipf("skipping: cannot insert test user: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO mt_accounts (id, user_id, mt_type, broker_host, login, account_status) VALUES ($1, $2, 'mt4', 'test', '12345', 'disconnected') ON CONFLICT DO NOTHING`,
		a.accountID, a.userID); err != nil {
		t.Skipf("skipping: cannot insert test account: %v", err)
	}
	t.Cleanup(func() {
		p := getTestPool(t)
		cleanupTestTradeRecords(t, p, a.userID, a.accountID)
		p.Exec(context.Background(), `DELETE FROM trade_record_dedup_log WHERE removed_by = $1`, tag)
		p.Exec(context.Background(), `DELETE FROM mt_accounts WHERE id = $1`, a.accountID)
		p.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, a.userID)
	})
	return a
}

func countBreaks(breaks []model.ChainBreak, typ string) int {
	n := 0
	for _, b := range breaks {
		if b.Type == typ {
			n++
		}
	}
	return n
}

func firstBreaks(breaks []model.ChainBreak, n int) string {
	if len(breaks) > n {
		breaks = breaks[:n]
	}
	out := make([]string, 0, len(breaks))
	for _, b := range breaks {
		out = append(out, fmt.Sprintf("{seq=%d type=%s}", b.Seq, b.Type))
	}
	return strings.Join(out, ", ")
}

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
func TestVerifyGlobalChain_LegacyNormalizedEncoding_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := newGlobalVerifyRepo(t)
	ctx := context.Background()
	acct := newGVAccount(t, pool, "gv-t7")

	var tailSeq int64
	var tailEntry []byte
	// Union tail, not live tail: archived rows can sit above the live max
	// seq (dedup removes tail-most rows), and a seed inserted mid-chain would
	// break the archived follower's stored prev link.
	if err := pool.QueryRow(ctx,
		`SELECT seq, entry_hash FROM (
			SELECT seq, entry_hash FROM trade_records WHERE entry_hash IS NOT NULL
			UNION ALL
			SELECT seq, entry_hash FROM trade_record_dedup_log WHERE entry_hash IS NOT NULL
		) t ORDER BY seq DESC LIMIT 1`,
	).Scan(&tailSeq, &tailEntry); err != nil {
		t.Fatalf("read union tail: %v", err)
	}

	id := uuid.New()
	openTime := time.Date(2026, 9, 19, 5, 0, 0, 0, time.UTC)
	closeTime := openTime.Add(30 * time.Minute)
	seq := tailSeq + 1
	// Legacy write-time representation: decimal.NewFromFloat(...).String().
	legacyHash := computeTradeEntryHash(tailEntry, seq, acct.accountID, 7501, "EURUSD",
		"0.05", "1.1", "1.105", "50",
		[2]int64{openTime.UnixMilli(), closeTime.UnixMilli()})
	if _, err := pool.Exec(ctx, `
		INSERT INTO trade_records (
			id, user_id, account_id, ticket, symbol, order_type, volume,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, stop_loss, take_profit,
			order_comment, magic_number, platform,
			created_at, updated_at, seq, prev_hash, entry_hash
		) OVERRIDING SYSTEM VALUE VALUES (
			$1, $2, $3, 7501, 'EURUSD', 'buy', 0.05,
			1.1, 1.105, 50, 0, 0,
			$4, $5, 1.095, 1.11,
			'gv-t7', 1, 'mt4',
			$4, $4, $6, $7, $8
		)`,
		id, acct.userID, acct.accountID, openTime, closeTime, seq, tailEntry, legacyHash,
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	// Discriminator: the canonical ::text form alone must NOT match; the
	// normalized form must.
	columnVolume, columnOpen, columnClose, columnProfit := "0.0500", "1.10000000", "1.10500000", "50.0000"
	canonical := computeTradeEntryHash(tailEntry, seq, acct.accountID, 7501, "EURUSD",
		columnVolume, columnOpen, columnClose, columnProfit,
		[2]int64{openTime.UnixMilli(), closeTime.UnixMilli()})
	if string(canonical) == string(legacyHash) {
		t.Fatal("test setup: canonical and legacy encodings coincide — discriminator lost")
	}
	if !tradeEntryHashVerifies(legacyHash, tailEntry, seq, acct.accountID, 7501, "EURUSD",
		columnVolume, columnOpen, columnClose, columnProfit, openTime, closeTime) {
		t.Fatal("normalized encoding channel failed to confirm the legacy row")
	}

	breaks, err := repo.VerifyChain(ctx, acct.userID, acct.accountID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if countBreaks(breaks, "hash_mismatch") != 0 {
		t.Fatalf("legacy row must be confirmed via the normalized channel, got: %+v", breaks)
	}
}

// T8 — canonical write path (S2): a noisy decimal ("0.30000000000000004")
// rounds to the column form 0.3000; the hash now covers the RETURNED ::text
// form, so the verifier's canonical channel confirms it verbatim.
func TestVerifyGlobalChain_NoisyDecimalCanonicalWrite_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := newGlobalVerifyRepo(t)
	ctx := context.Background()
	acct := newGVAccount(t, pool, "gv-t8")

	r := makeTestTradeRecord(acct.userID, acct.accountID, 7601)
	r.Volume = decimal.RequireFromString("0.30000000000000004")
	if err := repo.Create(ctx, r); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var stored string
	if err := pool.QueryRow(ctx,
		`SELECT volume::text FROM trade_records WHERE id = $1`, r.ID).Scan(&stored); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if stored != "0.3000" {
		t.Fatalf("column form = %q, want 0.3000", stored)
	}

	breaks, err := repo.VerifyChain(ctx, acct.userID, acct.accountID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if countBreaks(breaks, "hash_mismatch") != 0 {
		t.Fatalf("canonical write must verify via ::text channel, got: %+v", breaks)
	}
}

// T9 — unhashed rows are disclosed, not dropped, and do not pollute linkage.
func TestVerifyGlobalChain_UnhashedDisclosed_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := newGlobalVerifyRepo(t)
	ctx := context.Background()
	acct := newGVAccount(t, pool, "gv-t9")

	// Hashed rows first; the NULL-hash row goes LAST so the linkage walk
	// (which skips NULL-hash rows) is unaffected by its position.
	r := makeTestTradeRecord(acct.userID, acct.accountID, 7701)
	if err := repo.Create(ctx, r); err != nil {
		t.Fatalf("seed hashed: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO trade_records (
			id, user_id, account_id, ticket, symbol, order_type, volume,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, stop_loss, take_profit,
			order_comment, magic_number, platform,
			created_at, updated_at, prev_hash, entry_hash
		) VALUES (
			$1, $2, $3, 7702, 'EURUSD', 'buy', 0.1,
			1.1, 1.105, 50, 0, 0,
			'2026-09-19 06:00', '2026-09-19 06:30', 1.095, 1.11,
			'gv-t9', 1, 'mt4',
			'2026-09-19 06:00', '2026-09-19 06:00', NULL, NULL
		)`, uuid.New(), acct.userID, acct.accountID); err != nil {
		t.Fatalf("seed unhashed: %v", err)
	}

	all, err := repo.VerifyGlobalChain(ctx)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	unhashedForAccount := 0
	for _, b := range all {
		if b.Type == "unhashed" && b.AccountID == acct.accountID {
			unhashedForAccount++
		}
	}
	if unhashedForAccount != 1 {
		t.Fatalf("want exactly 1 unhashed finding for the account, got %d", unhashedForAccount)
	}
	breaks, err := repo.VerifyChain(ctx, acct.userID, acct.accountID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if countBreaks(breaks, "chain_break") != 0 || countBreaks(breaks, "hash_mismatch") != 0 {
		t.Fatalf("unhashed row must not pollute linkage: %+v", breaks)
	}
}

// resetDedupLogIdentity re-seats the log_id identity sequence above the
// restored rows — OVERRIDING SYSTEM VALUE inserts do not advance it, and a
// stale sequence would collide future inserts with restored keys.
func resetDedupLogIdentity(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`SELECT setval(pg_get_serial_sequence('trade_record_dedup_log', 'log_id'),
		              GREATEST((SELECT COALESCE(MAX(log_id), 1) FROM trade_record_dedup_log), 1))`); err != nil {
		t.Logf("reset dedup log identity: %v", err)
	}
}

// migration281LogDDL extracts the migration's original full-column CREATE
// TABLE statement for the dedup log (subset harnesses would not satisfy the
// union query).
func migration281LogDDL(t *testing.T) string {
	t.Helper()
	sqlBytes, err := os.ReadFile("../../migrations/281_trade_records_dedup.up.sql")
	if err != nil {
		t.Fatalf("read migration 281: %v", err)
	}
	lines := strings.Split(string(sqlBytes), "\n")
	var out []string
	capturing := false
	for _, line := range lines {
		if strings.HasPrefix(line, "CREATE TABLE trade_record_dedup_log") {
			capturing = true
		}
		if capturing {
			out = append(out, line)
			if line == ");" {
				break
			}
		}
	}
	if !capturing {
		t.Fatal("CREATE TABLE trade_record_dedup_log not found in migration 281")
	}
	return strings.Join(out, "\n")
}
