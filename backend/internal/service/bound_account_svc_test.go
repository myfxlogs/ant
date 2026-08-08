package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/repository"
)

// mockSubRepo is a minimal SubscriptionRepository stub for BoundAccountService tests.
// We can't use the real repo without a DB, so we test the logic via the service layer
// with a test that exercises the EnsureBoundAccount path against a real database
// in integration mode, or via mock for unit tests.

// TestEnsureBoundAccount_FreeTierRejectsSecondAccount is the adversarial proof:
// A free-tier user (max_mt_accounts=1) with one account already bound
// must be rejected when attempting to bind a second account.
//
// Adversarial proof: remove the limit check in EnsureBoundAccount → this test goes red.
func TestEnsureBoundAccount_FreeTierRejectsSecondAccount(t *testing.T) {
	if testing.Short() {
		t.Skip("requires database")
	}
	ctx := context.Background()
	pool := testPool(t)
	defer pool.Close()

	log := zap.NewNop()
	subRepo := repository.NewSubscriptionRepository(pool)
	boundRepo := repository.NewBoundAccountRepository(pool)
	svc := NewBoundAccountService(boundRepo, subRepo, pool, log)

	// Create a test user with free subscription.
	userID := createTestUser(t, pool)
	defer cleanupTestUser(t, pool, userID)

	// Ensure free subscription exists.
	subSvc := NewSubscriptionService(subRepo, nil, pool, log)
	if err := subSvc.EnsureFreeSubscription(ctx, userID); err != nil {
		t.Fatalf("ensure free subscription: %v", err)
	}

	// Create two test MT accounts for the user.
	acc1 := createTestMTAccount(t, pool, userID, "1001")
	acc2 := createTestMTAccount(t, pool, userID, "1002")
	defer cleanupTestMTAccount(t, pool, acc1)
	defer cleanupTestMTAccount(t, pool, acc2)

	// Clean any pre-existing bindings.
	_ = boundRepo.UnbindAccount(ctx, userID, acc1)
	_ = boundRepo.UnbindAccount(ctx, userID, acc2)

	// Binding first account should succeed (free tier = 1).
	if err := svc.EnsureBoundAccount(ctx, userID, acc1); err != nil {
		t.Fatalf("bind first account should succeed: %v", err)
	}

	// Binding second account must fail with ErrAccountLimitExceeded.
	err := svc.EnsureBoundAccount(ctx, userID, acc2)
	if err == nil {
		t.Fatal("expected ErrAccountLimitExceeded, got nil — adversarial proof: limit check is missing!")
	}
	if err != ErrAccountLimitExceeded {
		t.Fatalf("expected ErrAccountLimitExceeded, got: %v", err)
	}

	// Verify the first account is still bound (idempotent re-check).
	if err := svc.EnsureBoundAccount(ctx, userID, acc1); err != nil {
		t.Fatalf("re-bind first account should succeed (idempotent): %v", err)
	}
}

// TestEnsureBoundAccount_ProTierAllowsMultipleAccounts verifies that pro tier (max=5)
// allows binding multiple accounts up to the limit.
func TestEnsureBoundAccount_ProTierAllowsMultipleAccounts(t *testing.T) {
	if testing.Short() {
		t.Skip("requires database")
	}
	ctx := context.Background()
	pool := testPool(t)
	defer pool.Close()

	log := zap.NewNop()
	subRepo := repository.NewSubscriptionRepository(pool)
	boundRepo := repository.NewBoundAccountRepository(pool)
	svc := NewBoundAccountService(boundRepo, subRepo, pool, log)

	userID := createTestUser(t, pool)
	defer cleanupTestUser(t, pool, userID)

	// Create pro subscription by directly inserting.
	proPlan, err := subRepo.GetPlanByName(ctx, "pro")
	if err != nil {
		t.Fatalf("get pro plan: %v", err)
	}
	if proPlan == nil {
		t.Skip("pro plan not found in DB — skipping")
	}

	// Deactivate any existing subscription and create pro.
	_, _ = pool.Exec(ctx, `UPDATE user_platform_subscriptions SET status='expired' WHERE user_id=$1`, userID)
	_, _ = pool.Exec(ctx,
		`INSERT INTO user_platform_subscriptions (id, user_id, plan_id, status, billing_cycle, current_period_start, current_period_end, auto_renew)
		 VALUES ($1, $2, $3, 'active', 'monthly', NOW(), NOW() + interval '1 month', false)`,
		uuid.New(), userID, proPlan.ID)

	// Create 6 test MT accounts.
	var accIDs []uuid.UUID
	for i := 0; i < 6; i++ {
		accID := createTestMTAccount(t, pool, userID, "200"+string(rune('1'+i)))
		accIDs = append(accIDs, accID)
		defer cleanupTestMTAccount(t, pool, accID)
		_ = boundRepo.UnbindAccount(ctx, userID, accID)
	}

	// Bind accounts 1-5 should succeed (pro = 5).
	for i := 0; i < 5; i++ {
		if err := svc.EnsureBoundAccount(ctx, userID, accIDs[i]); err != nil {
			t.Fatalf("bind account %d should succeed on pro tier: %v", i+1, err)
		}
	}

	// Account 6 must be rejected.
	err = svc.EnsureBoundAccount(ctx, userID, accIDs[5])
	if err == nil {
		t.Fatal("expected ErrAccountLimitExceeded for 6th account on pro tier, got nil")
	}
	if err != ErrAccountLimitExceeded {
		t.Fatalf("expected ErrAccountLimitExceeded, got: %v", err)
	}
}

// TestEnsureBoundAccount_NotOwnedAccountRejected verifies that binding
// an account that doesn't belong to the user is rejected.
func TestEnsureBoundAccount_NotOwnedAccountRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("requires database")
	}
	ctx := context.Background()
	pool := testPool(t)
	defer pool.Close()

	log := zap.NewNop()
	subRepo := repository.NewSubscriptionRepository(pool)
	boundRepo := repository.NewBoundAccountRepository(pool)
	svc := NewBoundAccountService(boundRepo, subRepo, pool, log)

	userID := createTestUser(t, pool)
	defer cleanupTestUser(t, pool, userID)
	subSvc := NewSubscriptionService(subRepo, nil, pool, log)
	_ = subSvc.EnsureFreeSubscription(ctx, userID)

	otherUserID := createTestUser(t, pool)
	defer cleanupTestUser(t, pool, otherUserID)
	otherAcc := createTestMTAccount(t, pool, otherUserID, "9999")
	defer cleanupTestMTAccount(t, pool, otherAcc)

	err := svc.EnsureBoundAccount(ctx, userID, otherAcc)
	if err == nil {
		t.Fatal("expected ErrAccountNotOwned, got nil")
	}
	if err != ErrAccountNotOwned {
		t.Fatalf("expected ErrAccountNotOwned, got: %v", err)
	}
}

// --- helpers ---

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://alphaforge:alphaforge@localhost:5432/alphaforge?sslmode=disable")
	if err != nil {
		t.Skipf("no database: %v", err)
	}
	return pool
}

func createTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	uid := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, password_hash, status, created_at, updated_at)
		 VALUES ($1, $2, 'testhash', 'active', NOW(), NOW())`,
		uid, "test-"+uid.String()+"@example.com")
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return uid
}

func cleanupTestUser(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) {
	t.Helper()
	_, _ = pool.Exec(context.Background(), `DELETE FROM subscription_bound_accounts WHERE user_id=$1`, userID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM user_platform_subscriptions WHERE user_id=$1`, userID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM mt_accounts WHERE user_id=$1`, userID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)
}

func createTestMTAccount(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, login string) uuid.UUID {
	t.Helper()
	accID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO mt_accounts (id, user_id, login, broker, server, mt_type, account_status, balance, leverage, created_at, updated_at)
		 VALUES ($1, $2, $3, 'TestBroker', 'TestServer', 'mt4', 'active', $4, 100, NOW(), NOW())`,
		accID, userID, login, decimal.NewFromInt(10000).String())
	if err != nil {
		t.Fatalf("create test mt account: %v", err)
	}
	return accID
}

func cleanupTestMTAccount(t *testing.T, pool *pgxpool.Pool, accID uuid.UUID) {
	t.Helper()
	_, _ = pool.Exec(context.Background(), `DELETE FROM subscription_bound_accounts WHERE mt_account_id=$1`, accID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM mt_accounts WHERE id=$1`, accID)
}

// Ensure model package is referenced (avoids unused import in some build configs).
var _ = model.SubscriptionPlan{}
