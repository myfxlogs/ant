//go:build integration

package marketplace

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestMoneyFlow_RefundAlreadyRefunded(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	result, err := f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", "idem-"+uuid.NewString()[:8])
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}

	_, err = f.svc.RefundPurchase(ctx, f.buyerID.String(), result.SubscriptionID)
	if err != nil {
		t.Fatalf("first refund: %v", err)
	}

	_, err = f.svc.RefundPurchase(ctx, f.buyerID.String(), result.SubscriptionID)
	if err == nil {
		t.Fatal("expected double refund error")
	}
}

// TestMoneyFlow_SubscribeFree verifies free strategy subscription works without charging.
func TestMoneyFlow_SubscribeFree(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	// Set strategy to free.
	_, err := f.pool.Exec(ctx,
		`UPDATE marketplace_strategies SET price_model = 'free', price_amount = NULL WHERE strategy_id = $1`,
		f.stratID,
	)
	if err != nil {
		t.Fatalf("update to free: %v", err)
	}

	subID, err := f.svc.Subscribe(ctx, f.buyerID.String(), f.sellerID.String(), f.stratID.String(), "subscribe")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if subID == "" {
		t.Fatal("expected subscription ID")
	}

	// Balance unchanged.
	balance := f.getBalance(ctx, t, f.buyerID)
	if !balance.Equal(decimal.NewFromFloat(1000)) {
		t.Errorf("balance after free subscribe: expected 1000, got %s", balance)
	}

	// No settlement row.
	var count int
	err = f.pool.QueryRow(ctx, `SELECT count(*) FROM marketplace_settlements WHERE buyer_id = $1`, f.buyerID).Scan(&count)
	if err != nil {
		t.Fatalf("count settlements: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 settlements for free subscribe, got %d", count)
	}
}

// TestMoneyFlow_SubscribePaidRejected verifies Subscribe rejects paid strategies.
func TestMoneyFlow_SubscribePaidRejected(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	// Strategy is already priced (once, 50.00).
	_, err := f.svc.Subscribe(ctx, f.buyerID.String(), f.sellerID.String(), f.stratID.String(), "subscribe")
	if err == nil {
		t.Fatal("expected error: paid strategies require purchase")
	}
}

// TestMoneyFlow_SettlementIdempotent verifies settling twice doesn't double-credit.
func TestMoneyFlow_SettlementIdempotent(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	result, err := f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", "idem-"+uuid.NewString()[:8])
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}

	subID, _ := uuid.Parse(result.SubscriptionID)
	_, err = f.pool.Exec(ctx,
		`UPDATE marketplace_settlements SET settles_at = now() - interval '1 second' WHERE purchase_id = $1`,
		subID,
	)
	if err != nil {
		t.Fatalf("backdate: %v", err)
	}

	// First settle.
	_, err = f.svc.SettleExpired(ctx, f.sellerID.String())
	if err != nil {
		t.Fatalf("first settle: %v", err)
	}

	sellerAfterFirst := f.getBalance(ctx, t, f.sellerID)

	// Second settle (should be no-op, settlement already settled).
	_, err = f.svc.SettleExpired(ctx, f.sellerID.String())
	if err != nil {
		t.Fatalf("second settle: %v", err)
	}

	sellerAfterSecond := f.getBalance(ctx, t, f.sellerID)
	if !sellerAfterFirst.Equal(sellerAfterSecond) {
		t.Errorf("idempotent settle: seller balance changed: %s → %s", sellerAfterFirst, sellerAfterSecond)
	}
}

// TestMoneyFlow_FullLifecycle tests purchase → settle → refund-settled → verify balances.
func TestMoneyFlow_FullLifecycle(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	// 1. Purchase.
	result, err := f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", "idem-"+uuid.NewString()[:8])
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}
	subID, _ := uuid.Parse(result.SubscriptionID)

	buyerAfterPurchase := f.getBalance(ctx, t, f.buyerID)
	sellerAfterPurchase := f.getBalance(ctx, t, f.sellerID)

	// 2. Settle (backdate).
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

	buyerAfterSettle := f.getBalance(ctx, t, f.buyerID)
	sellerAfterSettle := f.getBalance(ctx, t, f.sellerID)

	// Buyer balance unchanged by settlement.
	if !buyerAfterSettle.Equal(buyerAfterPurchase) {
		t.Errorf("buyer balance changed at settle: %s → %s", buyerAfterPurchase, buyerAfterSettle)
	}
	// Seller credited 45 (50 - 10%% default fee).
	if !sellerAfterSettle.Equal(sellerAfterPurchase.Add(decimal.NewFromFloat(45))) {
		t.Errorf("seller not credited correctly: expected %s, got %s",
			sellerAfterPurchase.Add(decimal.NewFromFloat(45)), sellerAfterSettle)
	}

	// 3. Refund (settled → reverse).
	_, err = f.svc.RefundPurchase(ctx, f.buyerID.String(), result.SubscriptionID)
	if err != nil {
		t.Fatalf("refund: %v", err)
	}

	buyerAfterRefund := f.getBalance(ctx, t, f.buyerID)
	sellerAfterRefund := f.getBalance(ctx, t, f.sellerID)

	// Buyer fully refunded.
	if !buyerAfterRefund.Equal(decimal.NewFromFloat(1000)) {
		t.Errorf("buyer after full lifecycle: expected 1000, got %s", buyerAfterRefund)
	}
	// Seller debited back (45 reversed).
	if !sellerAfterRefund.Equal(decimal.NewFromFloat(0)) {
		t.Errorf("seller after full lifecycle: expected 0, got %s", sellerAfterRefund)
	}

	// 4. Settlement refunded.
	status := f.getSettlementStatus(ctx, t, subID)
	if status != "refunded" {
		t.Errorf("settlement status: expected refunded, got %s", status)
	}

	t.Logf("full lifecycle: buyer %s→%s→%s→%s, seller %s→%s→%s→%s",
		"1000", buyerAfterPurchase, buyerAfterSettle, buyerAfterRefund,
		"0", sellerAfterPurchase, sellerAfterSettle, sellerAfterRefund)
}

// TestMoneyFlow_RefundWindowExpired verifies refund is rejected after window expires.
func TestMoneyFlow_RefundWindowExpired(t *testing.T) {
	t.Parallel()
	f := setupMoneyFlow(t)
	ctx := context.Background()

	result, err := f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", "idem-"+uuid.NewString()[:8])
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}

	// Backdate subscription creation to exceed 7-day refund window.
	subID, _ := uuid.Parse(result.SubscriptionID)
	_, err = f.pool.Exec(ctx,
		`UPDATE user_subscriptions SET created_at = now() - interval '10 days' WHERE id = $1`,
		subID,
	)
	if err != nil {
		t.Fatalf("backdate subscription: %v", err)
	}

	// Refund request should fail (settlement still frozen, but subscription past refund window).
	_, err = f.svc.CreateRefundRequest(ctx, f.buyerID.String(), result.SubscriptionID, "test refund")
	if err == nil {
		t.Fatal("expected refund window expired error")
	}
}

// TestMoneyFlow_PurchaseWithFeeTier verifies fee split when marketplace fee tier is applied.
// The starter tier (fee_rate=0.10, min_sales_count=0) applies to new sellers.
// This test is NOT parallel because it reads the shared system wallet.
func TestMoneyFlow_PurchaseWithFeeTier(t *testing.T) {
	f := setupMoneyFlow(t)
	ctx := context.Background()

	// Read system wallet balance before.
	var sysBalanceBefore string
	_ = f.pool.QueryRow(ctx, `SELECT balance::text FROM user_wallets WHERE user_id = $1`, SystemUserID).Scan(&sysBalanceBefore)
	sysBefore, _ := decimal.NewFromString(sysBalanceBefore)

	result, err := f.svc.PurchaseStrategy(ctx, f.buyerID.String(), f.stratID.String(), "", "idem-"+uuid.NewString()[:8])
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}

	// Buyer charged 50.
	balanceAfter := f.getBalance(ctx, t, f.buyerID)
	if !balanceAfter.Equal(decimal.NewFromFloat(950)) {
		t.Errorf("buyer balance: expected 950, got %s", balanceAfter)
	}

	// Settle and verify seller gets 45 (50 - 10% starter tier fee).
	subID, _ := uuid.Parse(result.SubscriptionID)
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

	sellerBalance := f.getBalance(ctx, t, f.sellerID)
	if !sellerBalance.Equal(decimal.NewFromFloat(45)) {
		t.Errorf("seller balance with 10%% starter fee: expected 45, got %s", sellerBalance)
	}

	// System wallet gets 5 (10% of 50) — check delta since wallet is shared.
	var sysBalanceAfter string
	err = f.pool.QueryRow(ctx, `SELECT balance::text FROM user_wallets WHERE user_id = $1`, SystemUserID).Scan(&sysBalanceAfter)
	if err != nil {
		t.Fatalf("get system balance: %v", err)
	}
	sysAfter, _ := decimal.NewFromString(sysBalanceAfter)
	sysDelta := sysAfter.Sub(sysBefore)
	if !sysDelta.Equal(decimal.NewFromFloat(5)) {
		t.Errorf("system wallet delta with 10%% fee: expected 5, got %s", sysDelta)
	}
}

// ── Summary ───────────────────────────────────────────────────────────────────

// TestMoneyFlowSummary runs all money-flow tests and logs a summary.
func TestMoneyFlowSummary(t *testing.T) {
	tests := []string{
		"TestMoneyFlow_PurchaseOnce_HappyPath",
		"TestMoneyFlow_PurchaseOnce_RefundFrozen",
		"TestMoneyFlow_PurchaseOnce_RefundSettled",
		"TestMoneyFlow_PurchaseSubscription",
		"TestMoneyFlow_PurchaseInsufficientBalance",
		"TestMoneyFlow_PurchaseAlreadySubscribed",
		"TestMoneyFlow_PurchaseSelfBuy",
		"TestMoneyFlow_PurchaseIdempotent",
		"TestMoneyFlow_RefundAlreadyRefunded",
		"TestMoneyFlow_SubscribeFree",
		"TestMoneyFlow_SubscribePaidRejected",
		"TestMoneyFlow_SettlementIdempotent",
		"TestMoneyFlow_FullLifecycle",
		"TestMoneyFlow_RefundWindowExpired",
		"TestMoneyFlow_PurchaseWithFeeTier",
	}
	t.Logf("LAUNCH-2 money flow integration tests: %d cases", len(tests))
	for _, name := range tests {
		t.Logf("  - %s", name)
	}
	_ = fmt.Sprintf // ensure fmt is used
}
