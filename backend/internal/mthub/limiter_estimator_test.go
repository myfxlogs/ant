package mthub

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/internal/costsvc"
	"alphaforge/internal/usermgr"
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
		SpreadPips:       decimal.NewFromFloat(1.5),
		PipSize:          decimal.NewFromFloat(0.0001),
		PipValue:         decimal.NewFromFloat(10.0),
		CommissionPerLot: decimal.NewFromFloat(7.0),
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

// mockExecutor is a minimal OrderExecutor stub for unit tests.
type mockExecutor struct{}

func (m *mockExecutor) Platform() string                        { return "mock" }
func (m *mockExecutor) PlaceOrder(_ context.Context, _ *OrderRequest) (int64, error) { return 99999, nil }
func (m *mockExecutor) CloseOrder(_ context.Context, _ int64, _ decimal.Decimal) error { return nil }
func (m *mockExecutor) DeleteOrder(_ context.Context, _ int64) error                   { return nil }
func (m *mockExecutor) ModifyOrder(_ context.Context, _ int64, _, _, _ decimal.Decimal) error { return nil }
func (m *mockExecutor) FetchOpenedOrders(_ context.Context) ([]*OrderRecord, error)  { return nil, nil }
func (m *mockExecutor) FetchOrderHistory(_ context.Context, _, _ time.Time) ([]*OrderRecord, error) { return nil, nil }
func (m *mockExecutor) FetchSymbolParams(_ context.Context, _ []string) ([]*SymbolParam, error) { return nil, nil }
func (m *mockExecutor) FetchAllSymbols(_ context.Context) ([]string, error)               { return nil, nil }
func (m *mockExecutor) FetchPriceHistory(_ context.Context, _, _ string, _, _ int64, _ int) ([]*Bar, error) {
	return nil, nil
}
func (m *mockExecutor) AddSymbols(_ context.Context, _ []string) error                  { return nil }
func (m *mockExecutor) SubscribeOrderEvents(_ context.Context, _ OrderEventHandler) error { return nil }
