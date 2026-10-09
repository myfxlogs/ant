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
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

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
	if !tradeEntryHashVerifies(legacyHash, tailEntry, tradeEntryRow{
		seq: seq, accountID: acct.accountID, ticket: 7501,
		symbol: "EURUSD", volume: columnVolume, openPrice: columnOpen, closePrice: columnClose, profit: columnProfit,
		openTime: openTime, closeTime: closeTime,
	}) {
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
