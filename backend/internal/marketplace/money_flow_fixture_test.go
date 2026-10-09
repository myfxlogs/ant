//go:build integration

package marketplace

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

func moneyFlowTestPG(t *testing.T) *pgxpool.Pool {
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

type moneyFlowFixture struct {
	pool       *pgxpool.Pool
	svc        *Service
	buyerID    uuid.UUID
	sellerID   uuid.UUID
	stratID    uuid.UUID
	walletRepo *repository.WalletRepository
}

func setupMoneyFlow(t *testing.T) *moneyFlowFixture {
	t.Helper()
	pool := moneyFlowTestPG(t)
	ctx := context.Background()
	walletRepo := repository.NewWalletRepository(pool)
	svc := New(pool, walletRepo, zap.NewNop())

	buyerID := uuid.New()
	sellerID := uuid.New()
	stratID := uuid.New()

	// Create test users.
	for _, u := range []struct {
		id    uuid.UUID
		email string
	}{
		{buyerID, "mf-buyer-" + uuid.NewString()[:8] + "@anttest.io"},
		{sellerID, "mf-seller-" + uuid.NewString()[:8] + "@anttest.io"},
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
			 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', 'user', 'active', NOW(), NOW())`,
			u.id, u.email,
		); err != nil {
			t.Fatalf("insert test user: %v", err)
		}
	}

	// Create wallets with initial balance.
	for _, u := range []struct {
		id      uuid.UUID
		balance string
	}{
		{buyerID, "1000.00"},
		{sellerID, "0.00"},
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO user_wallets (user_id, balance) VALUES ($1, $2::numeric)
			 ON CONFLICT (user_id) DO UPDATE SET balance = $2::numeric`,
			u.id, u.balance,
		); err != nil {
			t.Fatalf("insert wallet: %v", err)
		}
	}

	// Create a strategy template owned by seller (marketplace_strategies FK → strategy_templates).
	if _, err := pool.Exec(ctx,
		`INSERT INTO strategy_templates (id, user_id, name, description, code, status, created_at, updated_at)
		 VALUES ($1, $2, 'Test Strategy', 'Test', '// test', 'published', NOW(), NOW())`,
		stratID, sellerID,
	); err != nil {
		t.Fatalf("insert strategy template: %v", err)
	}

	// Publish strategy on marketplace with price_model=once, price=50.00.
	if _, err := pool.Exec(ctx,
		`INSERT INTO marketplace_strategies (strategy_id, publisher_id, title, description, price_model, price_amount, asset_class, symbols, timeframe, risk_level, status, refund_window_days)
		 VALUES ($1, $2, 'Test Strategy', 'Test', 'once', 50.00, 'forex', '{EURUSD}', 'H1', 'medium', 'published', 7)`,
		stratID, sellerID,
	); err != nil {
		t.Fatalf("insert marketplace strategy: %v", err)
	}

	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM marketplace_settlements WHERE buyer_id = $1 OR provider_id = $2`, buyerID, sellerID)
		pool.Exec(ctx, `DELETE FROM user_subscriptions WHERE subscriber_user_id = $1`, buyerID)
		pool.Exec(ctx, `DELETE FROM marketplace_strategies WHERE strategy_id = $1`, stratID)
		pool.Exec(ctx, `DELETE FROM strategy_templates WHERE id = $1`, stratID)
		pool.Exec(ctx, `DELETE FROM user_wallets WHERE user_id IN ($1, $2)`, buyerID, sellerID)
		pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, buyerID, sellerID)
	})

	return &moneyFlowFixture{
		pool:       pool,
		svc:        svc,
		buyerID:    buyerID,
		sellerID:   sellerID,
		stratID:    stratID,
		walletRepo: walletRepo,
	}
}

func (f *moneyFlowFixture) getBalance(ctx context.Context, t *testing.T, userID uuid.UUID) decimal.Decimal {
	t.Helper()
	var bal string
	err := f.pool.QueryRow(ctx, `SELECT balance::text FROM user_wallets WHERE user_id = $1`, userID).Scan(&bal)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	d, err := decimal.NewFromString(bal)
	if err != nil {
		t.Fatalf("parse balance %q: %v", bal, err)
	}
	return d
}

func (f *moneyFlowFixture) getSettlementStatus(ctx context.Context, t *testing.T, purchaseID uuid.UUID) string {
	t.Helper()
	var status string
	err := f.pool.QueryRow(ctx, `SELECT status FROM marketplace_settlements WHERE purchase_id = $1`, purchaseID).Scan(&status)
	if err != nil {
		t.Fatalf("get settlement status: %v", err)
	}
	return status
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestMoneyFlow_PurchaseOnce_HappyPath verifies the complete purchase → settle flow:
// buyer is charged, settlement is frozen, then settled after refund window.
