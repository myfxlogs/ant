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

	svc := newTestService()
	svc.hub = hub

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

// mockExecutor is a configurable OrderExecutor stub for unit tests.
// All fn fields are optional — nil means use the default success behavior.
type mockExecutor struct {
	platform            string
	placeOrderFn        func(context.Context, *OrderRequest) (int64, error)
	closeOrderFn        func(context.Context, int64, decimal.Decimal) error
	deleteOrderFn       func(context.Context, int64) error
	modifyOrderFn       func(context.Context, int64, decimal.Decimal, decimal.Decimal, decimal.Decimal) error
	fetchSymbolParamsFn func(context.Context, []string) ([]*SymbolParam, error)
	fetchOpenedOrdersFn func(context.Context) ([]*OrderRecord, error)
	fetchOrderHistoryFn func(context.Context, time.Time, time.Time) ([]*OrderRecord, error)
	fetchAllSymbolsFn   func(context.Context) ([]string, error)
	fetchPriceHistoryFn func(context.Context, string, string, int64, int64, int) ([]*Bar, error)
	addSymbolsFn        func(context.Context, []string) error
}

func (m *mockExecutor) Platform() string {
	if m.platform != "" {
		return m.platform
	}
	return "mock"
}
func (m *mockExecutor) PlaceOrder(ctx context.Context, req *OrderRequest) (int64, error) {
	if m.placeOrderFn != nil {
		return m.placeOrderFn(ctx, req)
	}
	return 99999, nil
}
func (m *mockExecutor) CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error {
	if m.closeOrderFn != nil {
		return m.closeOrderFn(ctx, ticket, lots)
	}
	return nil
}
func (m *mockExecutor) DeleteOrder(ctx context.Context, ticket int64) error {
	if m.deleteOrderFn != nil {
		return m.deleteOrderFn(ctx, ticket)
	}
	return nil
}
func (m *mockExecutor) ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
	if m.modifyOrderFn != nil {
		return m.modifyOrderFn(ctx, ticket, sl, tp, price)
	}
	return nil
}
func (m *mockExecutor) FetchOpenedOrders(ctx context.Context) ([]*OrderRecord, error) {
	if m.fetchOpenedOrdersFn != nil {
		return m.fetchOpenedOrdersFn(ctx)
	}
	return nil, nil
}
func (m *mockExecutor) FetchOrderHistory(ctx context.Context, from, to time.Time) ([]*OrderRecord, error) {
	if m.fetchOrderHistoryFn != nil {
		return m.fetchOrderHistoryFn(ctx, from, to)
	}
	return nil, nil
}
func (m *mockExecutor) FetchSymbolParams(ctx context.Context, canonicals []string) ([]*SymbolParam, error) {
	if m.fetchSymbolParamsFn != nil {
		return m.fetchSymbolParamsFn(ctx, canonicals)
	}
	if len(canonicals) == 0 {
		return nil, nil
	}
	return []*SymbolParam{{Canonical: canonicals[0], ContractSize: decimal.NewFromInt(100000), LotSize: decimal.NewFromInt(100000)}}, nil
}
func (m *mockExecutor) FetchAllSymbols(ctx context.Context) ([]string, error) {
	if m.fetchAllSymbolsFn != nil {
		return m.fetchAllSymbolsFn(ctx)
	}
	return nil, nil
}
func (m *mockExecutor) FetchPriceHistory(ctx context.Context, symbol, period string, from, to int64, count int) ([]*Bar, error) {
	if m.fetchPriceHistoryFn != nil {
		return m.fetchPriceHistoryFn(ctx, symbol, period, from, to, count)
	}
	return nil, nil
}
func (m *mockExecutor) AddSymbols(ctx context.Context, symbols []string) error {
	if m.addSymbolsFn != nil {
		return m.addSymbolsFn(ctx, symbols)
	}
	return nil
}
func (m *mockExecutor) SubscribeOrderEvents(_ context.Context, _ OrderEventHandler) error { return nil }
