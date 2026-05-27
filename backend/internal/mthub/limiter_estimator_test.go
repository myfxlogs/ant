package mthub

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"anttrader/internal/costsvc"
	"anttrader/internal/usermgr"
)

func TestUserLimiterEffective(t *testing.T) {
	svc := &MtHubService{}
	limiter := usermgr.NewUserLimiter(usermgr.DefaultConfig())
	svc.SetUserLimiter(limiter)

	if svc.userLimiter == nil {
		t.Fatal("expected non-nil userLimiter after SetUserLimiter")
	}

	// Verify AllowOrder returns true for a fresh user (under limit).
	if !limiter.AllowOrder("test-user-1") {
		t.Error("expected AllowOrder to return true for fresh user")
	}

	// Flood the limiter: DefaultConfig allows 10 orders/sec.
	for i := 0; i < 10; i++ {
		limiter.AllowOrder("test-user-2")
	}
	if limiter.AllowOrder("test-user-2") {
		t.Error("expected AllowOrder to return false after exceeding rate limit")
	}
}

func TestCostEstimator_Wired(t *testing.T) {
	hub := NewHub()
	mockExec := &mockExecutor{}
	hub.Register("test-account", &Session{AccountID: "test-account"}, mockExec)

	svc := NewMtHubService(hub, nil, nil, nil, nil, nil, nil)

	model := &costsvc.CostModel{
		Symbol:           "EURUSD",
		SpreadPips:       1.5,
		PipSize:          0.0001,
		PipValue:         10.0,
		CommissionPerLot: 7.0,
	}
	estimator := &costsvc.StaticEstimator{Model: model}
	svc.SetCostEstimator(estimator)

	if svc.costEstimator == nil {
		t.Fatal("expected non-nil costEstimator after SetCostEstimator")
	}

	// Place an order and verify it succeeds (cost estimator runs internally).
	req := &OrderRequest{
		AccountID: "test-account",
		Canonical: "EURUSD",
		Side:      SideBuy,
		Volume:    decimal.NewFromFloat(0.1),
		Price:     decimal.NewFromFloat(1.1000),
	}
	order, err := svc.PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("PlaceOrder failed: %v", err)
	}
	if order.Ticket != 99999 {
		t.Errorf("expected ticket 99999, got %d", order.Ticket)
	}
}

func TestUserLimiter_RateLimitKicksIn(t *testing.T) {
	// Test the limiter directly — PlaceOrder only calls AllowOrder when
	// usermgr.GetUserID(ctx) returns a non-empty user ID, which requires
	// the auth interceptor to set it in the context.
	limiter := usermgr.NewUserLimiter(usermgr.Config{
		MaxEntries:      10,
		OrderPerUserMax: 3,
	})

	// First 3 should pass.
	for i := 0; i < 3; i++ {
		if !limiter.AllowOrder("user-1") {
			t.Errorf("order %d should have been allowed", i+1)
		}
	}
	// 4th should be blocked (exceeds 3/sec).
	if limiter.AllowOrder("user-1") {
		t.Error("expected 4th order to be rate-limited")
	}
}
