//go:build integration

package marketplace

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

// ── Test harness ──────────────────────────────────────────────────────────────

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
func TestMoneyFlow_PurchaseOnce_HappyPath(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	balanceBefore := f.getBalance(ctx, t, f.buyerID)
	if !balanceBefore.Equal(decimal.NewFromFloat(1000)) {
		t.Fatalf("expected initial balance 1000, got %s", balanceBefore)
	}

	// Purchase.
	result, err := f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", "idem-"+uuid.NewString()[:8])
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}

	// Verify buyer was charged 50.00.
	balanceAfter := f.getBalance(ctx, t, f.buyerID)
	expected := decimal.NewFromFloat(950)
	if !balanceAfter.Equal(expected) {
		t.Errorf("buyer balance after purchase: expected %s, got %s", expected, balanceAfter)
	}

	// Verify settlement is frozen.
	subID, err := uuid.Parse(result.SubscriptionID)
	if err != nil {
		t.Fatalf("parse sub ID: %v", err)
	}
	status := f.getSettlementStatus(ctx, t, subID)
	if status != "frozen" {
		t.Errorf("settlement status: expected frozen, got %s", status)
	}

	// Force settle by backdating the settlement.
	_, err = f.pool.Exec(ctx,
		`UPDATE marketplace_settlements SET settles_at = now() - interval '1 second' WHERE purchase_id = $1`,
		subID,
	)
	if err != nil {
		t.Fatalf("backdate settlement: %v", err)
	}

	// Settle expired.
	settleResult, err := f.svc.SettleExpired(ctx, f.sellerID.String())
	if err != nil {
		t.Fatalf("settle expired: %v", err)
	}
	if settleResult.SettledCount != 1 {
		t.Errorf("expected 1 settled, got %d", settleResult.SettledCount)
	}

	// Verify seller received funds. Default fee rate is 10% (fallback in getEffectiveFeeRateTx
	// when system_config=0 and no marketplace_fee_tiers rows), so seller gets 45.
	sellerBalance := f.getBalance(ctx, t, f.sellerID)
	if !sellerBalance.Equal(decimal.NewFromFloat(45)) {
		t.Errorf("seller balance after settlement: expected 45 (50 - 10%% fee), got %s", sellerBalance)
	}

	// Verify settlement is now settled.
	status = f.getSettlementStatus(ctx, t, subID)
	if status != "settled" {
		t.Errorf("settlement status after settle: expected settled, got %s", status)
	}
}

// TestMoneyFlow_PurchaseOnce_RefundFrozen verifies refund of a frozen settlement:
// buyer gets money back, settlement marked refunded, subscription deactivated.
func TestMoneyFlow_PurchaseOnce_RefundFrozen(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	idemKey := "idem-" + uuid.NewString()[:8]
	result, err := f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", idemKey)
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}

	balanceAfterPurchase := f.getBalance(ctx, t, f.buyerID)
	if !balanceAfterPurchase.Equal(decimal.NewFromFloat(950)) {
		t.Fatalf("buyer balance after purchase: expected 950, got %s", balanceAfterPurchase)
	}

	// Refund.
	refundResult, err := f.svc.RefundPurchase(ctx, f.buyerID.String(), result.SubscriptionID)
	if err != nil {
		t.Fatalf("refund: %v", err)
	}

	// Verify buyer got money back.
	balanceAfterRefund := f.getBalance(ctx, t, f.buyerID)
	if !balanceAfterRefund.Equal(decimal.NewFromFloat(1000)) {
		t.Errorf("buyer balance after refund: expected 1000, got %s", balanceAfterRefund)
	}

	// Verify settlement is refunded.
	subID, _ := uuid.Parse(result.SubscriptionID)
	status := f.getSettlementStatus(ctx, t, subID)
	if status != "refunded" {
		t.Errorf("settlement status after refund: expected refunded, got %s", status)
	}

	// Verify subscription is inactive.
	var active bool
	err = f.pool.QueryRow(ctx, `SELECT active FROM user_subscriptions WHERE id = $1`, subID).Scan(&active)
	if err != nil {
		t.Fatalf("query subscription: %v", err)
	}
	if active {
		t.Error("subscription should be inactive after refund")
	}

	_ = refundResult
}

// TestMoneyFlow_PurchaseOnce_RefundSettled verifies refund of a settled settlement:
// buyer gets money back, seller is debited, settlement marked refunded.
func TestMoneyFlow_PurchaseOnce_RefundSettled(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	result, err := f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", "idem-"+uuid.NewString()[:8])
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}

	subID, _ := uuid.Parse(result.SubscriptionID)

	// Backdate and settle.
	_, err = f.pool.Exec(ctx,
		`UPDATE marketplace_settlements SET settles_at = now() - interval '1 second' WHERE purchase_id = $1`,
		subID,
	)
	if err != nil {
		t.Fatalf("backdate: %v", err)
	}
	_, err = f.svc.SettleExpired(ctx, f.sellerID.String())
	if err != nil {
		t.Fatalf("settle: %v", err)
	}

	sellerBalanceAfterSettle := f.getBalance(ctx, t, f.sellerID)
	if !sellerBalanceAfterSettle.Equal(decimal.NewFromFloat(45)) {
		t.Fatalf("seller balance after settle: expected 45 (50 - 10%% fee), got %s", sellerBalanceAfterSettle)
	}

	// Refund.
	_, err = f.svc.RefundPurchase(ctx, f.buyerID.String(), result.SubscriptionID)
	if err != nil {
		t.Fatalf("refund settled: %v", err)
	}

	// Buyer gets full refund.
	balanceAfterRefund := f.getBalance(ctx, t, f.buyerID)
	if !balanceAfterRefund.Equal(decimal.NewFromFloat(1000)) {
		t.Errorf("buyer balance after refund: expected 1000, got %s", balanceAfterRefund)
	}

	// Seller is debited (45 reversed).
	sellerAfterRefund := f.getBalance(ctx, t, f.sellerID)
	if !sellerAfterRefund.Equal(decimal.NewFromFloat(0)) {
		t.Errorf("seller balance after refund: expected 0, got %s", sellerAfterRefund)
	}

	// Settlement is refunded.
	status := f.getSettlementStatus(ctx, t, subID)
	if status != "refunded" {
		t.Errorf("settlement status: expected refunded, got %s", status)
	}
}

// TestMoneyFlow_PurchaseSubscription verifies subscription model purchase:
// charges buyer, creates subscription with expiry, frozen settlement.
func TestMoneyFlow_PurchaseSubscription(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	// Change price model to subscription.
	_, err := f.pool.Exec(ctx,
		`UPDATE marketplace_strategies SET price_model = 'subscription', price_amount = 30.00 WHERE strategy_id = $1`,
		f.stratID,
	)
	if err != nil {
		t.Fatalf("update price model: %v", err)
	}

	result, err := f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", "idem-"+uuid.NewString()[:8])
	if err != nil {
		t.Fatalf("purchase subscription: %v", err)
	}

	// Buyer charged 30.
	balanceAfter := f.getBalance(ctx, t, f.buyerID)
	if !balanceAfter.Equal(decimal.NewFromFloat(970)) {
		t.Errorf("buyer balance: expected 970, got %s", balanceAfter)
	}

	// Subscription has expiry.
	var expiresAt *time.Time
	subID, _ := uuid.Parse(result.SubscriptionID)
	err = f.pool.QueryRow(ctx, `SELECT expires_at FROM user_subscriptions WHERE id = $1`, subID).Scan(&expiresAt)
	if err != nil {
		t.Fatalf("query subscription: %v", err)
	}
	if expiresAt == nil {
		t.Error("subscription should have expiry date")
	}

	// Settlement is frozen.
	status := f.getSettlementStatus(ctx, t, subID)
	if status != "frozen" {
		t.Errorf("settlement status: expected frozen, got %s", status)
	}
}

// TestMoneyFlow_PurchaseInsufficientBalance verifies purchase fails when buyer has no funds.
func TestMoneyFlow_PurchaseInsufficientBalance(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	// Drain buyer wallet.
	_, err := f.pool.Exec(ctx, `UPDATE user_wallets SET balance = 10.00 WHERE user_id = $1`, f.buyerID)
	if err != nil {
		t.Fatalf("drain wallet: %v", err)
	}

	_, err = f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", "idem-"+uuid.NewString()[:8])
	if err == nil {
		t.Fatal("expected insufficient balance error")
	}
}

// TestMoneyFlow_PurchaseAlreadySubscribed verifies duplicate purchase is rejected.
func TestMoneyFlow_PurchaseAlreadySubscribed(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	idemKey := "idem-" + uuid.NewString()[:8]
	_, err := f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", idemKey)
	if err != nil {
		t.Fatalf("first purchase: %v", err)
	}

	_, err = f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", "idem-"+uuid.NewString()[:8])
	if err == nil {
		t.Fatal("expected already subscribed error")
	}
}

// TestMoneyFlow_PurchaseSelfBuy verifies buyer cannot purchase own strategy.
func TestMoneyFlow_PurchaseSelfBuy(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	_, err := f.svc.PurchaseStrategy(ctx, f.sellerID.String(), f.stratID.String(), "", "idem-"+uuid.NewString()[:8])
	if err == nil {
		t.Fatal("expected self-purchase error")
	}
}

// TestMoneyFlow_PurchaseIdempotent verifies idempotent replay returns same result.
func TestMoneyFlow_PurchaseIdempotent(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	idemKey := "idem-" + uuid.NewString()[:8]
	result1, err := f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", idemKey)
	if err != nil {
		t.Fatalf("first purchase: %v", err)
	}

	result2, err := f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", idemKey)
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}

	if result1.SubscriptionID != result2.SubscriptionID {
		t.Errorf("idempotent: subscription IDs differ: %s vs %s", result1.SubscriptionID, result2.SubscriptionID)
	}

	// Balance should not change on replay.
	balance := f.getBalance(ctx, t, f.buyerID)
	if !balance.Equal(decimal.NewFromFloat(950)) {
		t.Errorf("balance after idempotent replay: expected 950, got %s", balance)
	}
}

// TestMoneyFlow_RefundAlreadyRefunded verifies double refund is rejected.
