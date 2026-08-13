//go:build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"alphaforge/internal/model"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping integration test: pg ping: %v", err)
	}
	return pool
}

func makeTestTradeRecord(userID, accountID uuid.UUID, ticket int64) *model.TradeRecord {
	return &model.TradeRecord{
		UserID:       userID,
		AccountID:    accountID,
		Ticket:       ticket,
		Symbol:       "EURUSD",
		OrderType:    "buy",
		Volume:       decimal.NewFromFloat(0.1),
		OpenPrice:    decimal.NewFromFloat(1.1000),
		ClosePrice:   decimal.NewFromFloat(1.1050),
		Profit:       decimal.NewFromFloat(50.0),
		Swap:         decimal.Zero,
		Commission:   decimal.Zero,
		OpenTime:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		CloseTime:    time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
		StopLoss:     decimal.NewFromFloat(1.0950),
		TakeProfit:   decimal.NewFromFloat(1.1100),
		OrderComment: "test",
		MagicNumber:  12345,
		Platform:     "mt4",
	}
}

func cleanupTestTradeRecords(t *testing.T, pool *pgxpool.Pool, userID, accountID uuid.UUID) {
	t.Helper()
	// Disable the append-only trigger temporarily for cleanup.
	_, _ = pool.Exec(context.Background(),
		`ALTER TABLE trade_records DISABLE TRIGGER prevent_trade_delete`)
	_, _ = pool.Exec(context.Background(),
		`DELETE FROM trade_records WHERE user_id = $1 AND account_id = $2`,
		userID, accountID)
	_, _ = pool.Exec(context.Background(),
		`ALTER TABLE trade_records ENABLE TRIGGER prevent_trade_delete`)
}

func TestTradeRecordHashChain_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := NewTradeRecordRepository(pool)

	userID := uuid.New()
	accountID := uuid.New()

	cleanupTestTradeRecords(t, pool, userID, accountID)
	t.Cleanup(func() { cleanupTestTradeRecords(t, pool, userID, accountID) })

	ctx := context.Background()

	// Insert 3 records.
	for i := int64(1); i <= 3; i++ {
		rec := makeTestTradeRecord(userID, accountID, 100+i)
		if err := repo.Create(ctx, rec); err != nil {
			t.Fatalf("insert record %d: %v", i, err)
		}
		if rec.Seq == 0 {
			t.Fatalf("insert record %d: seq not set", i)
		}
		if len(rec.EntryHash) != 32 {
			t.Fatalf("insert record %d: entry_hash not set (len=%d)", i, len(rec.EntryHash))
		}
	}

	// VerifyChain should return no breaks.
	breaks, err := repo.VerifyChain(ctx, userID, accountID)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if len(breaks) != 0 {
		t.Fatalf("expected 0 breaks, got %d: %+v", len(breaks), breaks)
	}
}

func TestTradeRecordHashChain_TamperEntryHash_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := NewTradeRecordRepository(pool)

	userID := uuid.New()
	accountID := uuid.New()

	cleanupTestTradeRecords(t, pool, userID, accountID)
	t.Cleanup(func() { cleanupTestTradeRecords(t, pool, userID, accountID) })

	ctx := context.Background()

	// Insert 2 records.
	for i := int64(1); i <= 2; i++ {
		rec := makeTestTradeRecord(userID, accountID, 200+i)
		if err := repo.Create(ctx, rec); err != nil {
			t.Fatalf("insert record %d: %v", i, err)
		}
	}

	// Tamper: corrupt entry_hash of the second record via direct SQL.
	// Must drop the protective trigger first, then re-enable.
	_, _ = pool.Exec(ctx, `ALTER TABLE trade_records DISABLE TRIGGER protect_trade_hash`)
	_, err := pool.Exec(ctx,
		`UPDATE trade_records SET entry_hash = decode('0000000000000000000000000000000000000000000000000000000000000000', 'hex')
		 WHERE user_id = $1 AND account_id = $2 AND ticket = $3`,
		userID, accountID, int64(202))
	_, _ = pool.Exec(ctx, `ALTER TABLE trade_records ENABLE TRIGGER protect_trade_hash`)
	if err != nil {
		t.Fatalf("tamper entry_hash: %v", err)
	}

	// VerifyChain should detect hash_mismatch.
	breaks, err := repo.VerifyChain(ctx, userID, accountID)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if len(breaks) == 0 {
		t.Fatal("expected hash_mismatch break, got none")
	}
	found := false
	for _, b := range breaks {
		if b.Type == "hash_mismatch" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected hash_mismatch break, got: %+v", breaks)
	}
}

func TestTradeRecordHashChain_TamperPrevHash_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := NewTradeRecordRepository(pool)

	userID := uuid.New()
	accountID := uuid.New()

	cleanupTestTradeRecords(t, pool, userID, accountID)
	t.Cleanup(func() { cleanupTestTradeRecords(t, pool, userID, accountID) })

	ctx := context.Background()

	for i := int64(1); i <= 3; i++ {
		rec := makeTestTradeRecord(userID, accountID, 300+i)
		if err := repo.Create(ctx, rec); err != nil {
			t.Fatalf("insert record %d: %v", i, err)
		}
	}

	// Tamper: corrupt prev_hash of the third record.
	_, _ = pool.Exec(ctx, `ALTER TABLE trade_records DISABLE TRIGGER protect_trade_hash`)
	_, err := pool.Exec(ctx,
		`UPDATE trade_records SET prev_hash = decode('ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff', 'hex')
		 WHERE user_id = $1 AND account_id = $2 AND ticket = $3`,
		userID, accountID, int64(303))
	_, _ = pool.Exec(ctx, `ALTER TABLE trade_records ENABLE TRIGGER protect_trade_hash`)
	if err != nil {
		t.Fatalf("tamper prev_hash: %v", err)
	}

	breaks, err := repo.VerifyChain(ctx, userID, accountID)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	foundChainBreak := false
	for _, b := range breaks {
		if b.Type == "chain_break" {
			foundChainBreak = true
		}
	}
	if !foundChainBreak {
		t.Fatalf("expected chain_break, got: %+v", breaks)
	}
}

func TestTradeRecordHashChain_DeleteBlocked_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := NewTradeRecordRepository(pool)

	userID := uuid.New()
	accountID := uuid.New()

	cleanupTestTradeRecords(t, pool, userID, accountID)
	t.Cleanup(func() { cleanupTestTradeRecords(t, pool, userID, accountID) })

	ctx := context.Background()

	rec := makeTestTradeRecord(userID, accountID, 400)
	if err := repo.Create(ctx, rec); err != nil {
		t.Fatalf("insert record: %v", err)
	}

	// Attempt DELETE — should be blocked by append-only trigger.
	_, err := pool.Exec(ctx,
		`DELETE FROM trade_records WHERE user_id = $1 AND account_id = $2 AND ticket = $3`,
		userID, accountID, int64(400))
	if err == nil {
		t.Fatal("expected DELETE to be blocked by append-only trigger, but it succeeded")
	}

	// Verify the record still exists.
	var count int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM trade_records WHERE user_id = $1 AND account_id = $2 AND ticket = $3`,
		userID, accountID, int64(400)).Scan(&count)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 record after blocked DELETE, got %d", count)
	}
}

func TestTradeRecordHashChain_BatchCreate_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := NewTradeRecordRepository(pool)

	userID := uuid.New()
	accountID := uuid.New()

	cleanupTestTradeRecords(t, pool, userID, accountID)
	t.Cleanup(func() { cleanupTestTradeRecords(t, pool, userID, accountID) })

	ctx := context.Background()

	var records []*model.TradeRecord
	for i := int64(1); i <= 5; i++ {
		rec := makeTestTradeRecord(userID, accountID, 500+i)
		records = append(records, rec)
	}

	if err := repo.BatchCreate(ctx, records); err != nil {
		t.Fatalf("BatchCreate: %v", err)
	}

	// All records should have seq and entry_hash set.
	for i, rec := range records {
		if rec.Seq == 0 {
			t.Fatalf("record %d: seq not set", i)
		}
		if len(rec.EntryHash) != 32 {
			t.Fatalf("record %d: entry_hash not set", i)
		}
	}

	// Chain should be valid.
	breaks, err := repo.VerifyChain(ctx, userID, accountID)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if len(breaks) != 0 {
		t.Fatalf("expected 0 breaks after BatchCreate, got %d: %+v", len(breaks), breaks)
	}
}

func TestTradeRecordHashChain_ConcurrentInsert_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := NewTradeRecordRepository(pool)

	userID := uuid.New()
	accountID := uuid.New()

	cleanupTestTradeRecords(t, pool, userID, accountID)
	t.Cleanup(func() { cleanupTestTradeRecords(t, pool, userID, accountID) })

	ctx := context.Background()

	// Insert from 5 goroutines concurrently.
	done := make(chan error, 5)
	for g := 0; g < 5; g++ {
		go func(grp int) {
			rec := makeTestTradeRecord(userID, accountID, int64(600+grp))
			rec.OrderComment = fmt.Sprintf("concurrent-%d", grp)
			done <- repo.Create(ctx, rec)
		}(g)
	}

	for i := 0; i < 5; i++ {
		if err := <-done; err != nil {
			t.Fatalf("concurrent insert %d: %v", i, err)
		}
	}

	// Chain should be valid (advisory lock serializes inserts).
	breaks, err := repo.VerifyChain(ctx, userID, accountID)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if len(breaks) != 0 {
		t.Fatalf("expected 0 breaks after concurrent inserts, got %d: %+v", len(breaks), breaks)
	}

	// Verify all 5 records are present.
	count, err := repo.CountByAccount(ctx, userID, accountID)
	if err != nil {
		t.Fatalf("CountByAccount: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5 records, got %d", count)
	}
}

// TestTradeRecordHashChain_DuplicateTicketIdempotent_Integration verifies EXEC-1:
// Writing the same (account_id, ticket, close_time) twice should succeed (idempotent
// skip via ON CONFLICT DO NOTHING), not fail with "hash chain fields are immutable".
//
// Adversarial proof: Revert to ON CONFLICT DO UPDATE + unconditional entry_hash UPDATE
// → second write hits trigger RAISE "immutable" → test fails (RED).
// With DO NOTHING + early return on conflict → second write returns nil (GREEN).
func TestTradeRecordHashChain_DuplicateTicketIdempotent_Integration(t *testing.T) {
	pool := getTestPool(t)
	repo := NewTradeRecordRepository(pool)

	userID := uuid.New()
	accountID := uuid.New()

	// Insert test user and mt_account to satisfy FK constraints.
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, status) VALUES ($1, $2, 'test', 'active') ON CONFLICT DO NOTHING`,
		userID, "test-exec1-"+userID.String()[:8]+"@test.local"); err != nil {
		t.Skipf("skipping: cannot insert test user: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO mt_accounts (id, user_id, mt_type, broker_host, login, account_status) VALUES ($1, $2, 'mt4', 'test', '12345', 'disconnected') ON CONFLICT DO NOTHING`,
		accountID, userID); err != nil {
		t.Skipf("skipping: cannot insert test account: %v", err)
	}
	t.Cleanup(func() {
		cleanupTestTradeRecords(t, pool, userID, accountID)
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE id = $1`, accountID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	// First write — should succeed and set entry_hash.
	rec1 := makeTestTradeRecord(userID, accountID, 999)
	if err := repo.Create(ctx, rec1); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	if len(rec1.EntryHash) != 32 {
		t.Fatalf("first write: entry_hash not set (len=%d)", len(rec1.EntryHash))
	}

	// Second write — same (account_id, ticket, close_time). Should be idempotent skip.
	rec2 := makeTestTradeRecord(userID, accountID, 999)
	if err := repo.Create(ctx, rec2); err != nil {
		t.Fatalf("second write (duplicate) failed — RED: ON CONFLICT DO UPDATE + entry_hash UPDATE hits immutable trigger: %v", err)
	}

	// Verify only 1 record exists (no duplicate).
	count, err := repo.CountByAccount(ctx, userID, accountID)
	if err != nil {
		t.Fatalf("CountByAccount: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 record after duplicate write, got %d", count)
	}

	// Verify entry_hash was set on the first write (NULL→non-NULL allowed by fixed trigger).
	var entryHash []byte
	if err := pool.QueryRow(ctx,
		`SELECT entry_hash FROM trade_records WHERE user_id = $1 AND account_id = $2 AND ticket = 999`,
		userID, accountID).Scan(&entryHash); err != nil {
		t.Fatalf("query entry_hash: %v", err)
	}
	if len(entryHash) != 32 {
		t.Fatalf("entry_hash not set (len=%d) — trigger still blocks NULL→non-NULL", len(entryHash))
	}
}
