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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

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
