//go:build integration

package service

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

func balanceSnapshotTestPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		dsn = "postgres://ant:ant@localhost:5432/ant?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping integration test: pg connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// DATA-TRUTH-4: RecordBalanceSnapshot must write into `account_balance_history`,
// the single table the whole analytics stack reads (equity curve, hourly equity,
// monthly detail, starting balance behind ReturnPercent).
//
// A refactor had pointed this insert at `account_balance_snapshots`, a table that
// does not exist in the schema. Every write failed, the error was swallowed at
// Debug level, and the readers starved for 28 days (production data stopped
// 2026-07-22 while accounts kept streaming).
//
// Adversarial proof: change the INSERT target in RecordBalanceSnapshot back to
// `account_balance_snapshots` → the insert errors on a missing relation → RED.
func TestDATATRUTH4_RecordBalanceSnapshot_LandsInAnalyticsTable(t *testing.T) {
	pool := balanceSnapshotTestPG(t)
	ctx := context.Background()
	svc := NewAccountService(pool, NewTestSecretsClient(t))

	userID := uuid.New()
	accID := uuid.New()

	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', 'user', 'active', NOW(), NOW())`,
		userID, "test-balsnap-"+userID.String()[:8]+"@anttest.io",
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO mt_accounts (id, user_id, mt_type, login, broker_company, broker_server, broker_host, account_status)
		 VALUES ($1, $2, 'mt4', $3, 'TestBroker', 'TestServer', 'demo.mt4.com', 'connected')`,
		accID, userID, "balsnap-"+userID.String()[:8],
	); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM account_balance_history WHERE account_id = $1`, accID)
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE id = $1`, accID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	// Values as the broker reports them — never recomputed locally.
	balance := decimal.RequireFromString("4109.63")
	equity := decimal.RequireFromString("37328.306")
	margin := decimal.RequireFromString("1867.23")
	freeMargin := decimal.RequireFromString("35461.076")

	if err := svc.RecordBalanceSnapshot(ctx, accID.String(), userID.String(), balance, equity, margin, freeMargin); err != nil {
		t.Fatalf("RecordBalanceSnapshot failed — RED: the analytics time-series receives nothing, "+
			"every equity/ReturnPercent reader starves: %v", err)
	}

	// Read it back exactly the way the equity-curve reader does
	// (analytics_repository_equity.go:155).
	var gotEquity, gotBalance, gotMargin, gotFreeMargin decimal.Decimal
	err := pool.QueryRow(ctx,
		`SELECT equity, balance, margin, free_margin FROM account_balance_history
		 WHERE account_id = $1 ORDER BY recorded_at DESC LIMIT 1`, accID,
	).Scan(&gotEquity, &gotBalance, &gotMargin, &gotFreeMargin)
	if err != nil {
		t.Fatalf("analytics reader found no snapshot in account_balance_history — RED: "+
			"writer and readers point at different tables: %v", err)
	}
	if !gotEquity.Equal(equity) {
		t.Fatalf("equity mismatch: got %s want %s", gotEquity, equity)
	}
	if !gotBalance.Equal(balance) {
		t.Fatalf("balance mismatch: got %s want %s", gotBalance, balance)
	}
	// Margin must be persisted verbatim from the broker frame: the risk gate
	// (margin_call / stop_out) reads it, and a silently-zeroed margin blinds it.
	if !gotMargin.Equal(margin) {
		t.Fatalf("margin mismatch: got %s want %s — RED: broker-reported margin lost on persist", gotMargin, margin)
	}
	if !gotFreeMargin.Equal(freeMargin) {
		t.Fatalf("free_margin mismatch: got %s want %s", gotFreeMargin, freeMargin)
	}
}

// The per-account throttle must not silently drop the very first sample.
func TestDATATRUTH4_RecordBalanceSnapshot_ThrottleKeepsFirstSample(t *testing.T) {
	pool := balanceSnapshotTestPG(t)
	ctx := context.Background()
	svc := NewAccountService(pool, NewTestSecretsClient(t))

	userID := uuid.New()
	accID := uuid.New()

	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', 'user', 'active', NOW(), NOW())`,
		userID, "test-balthr-"+userID.String()[:8]+"@anttest.io",
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO mt_accounts (id, user_id, mt_type, login, broker_company, broker_server, broker_host, account_status)
		 VALUES ($1, $2, 'mt4', $3, 'TestBroker', 'TestServer', 'demo.mt4.com', 'connected')`,
		accID, userID, "balthr-"+userID.String()[:8],
	); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM account_balance_history WHERE account_id = $1`, accID)
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE id = $1`, accID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	one := decimal.NewFromInt(1)
	// Two back-to-back calls: the first must persist, the second is throttled.
	if err := svc.RecordBalanceSnapshot(ctx, accID.String(), userID.String(), one, one, one, one); err != nil {
		t.Fatalf("first snapshot must persist: %v", err)
	}
	if err := svc.RecordBalanceSnapshot(ctx, accID.String(), userID.String(), one, one, one, one); err != nil {
		t.Fatalf("throttled snapshot must be a no-op, not an error: %v", err)
	}

	var rows int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM account_balance_history WHERE account_id = $1`, accID,
	).Scan(&rows); err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected exactly 1 persisted snapshot within the throttle window, got %d", rows)
	}
}
