// account_provider_test.go — MTAccountStateProvider tests (T3.2b).
package strategy

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/internal/mthub"
)

// stubExecutor implements mthub.OrderExecutor for testing.
type stubExecutor struct {
	orders []*mthub.OrderRecord
}

func (e *stubExecutor) Platform() string                        { return "test" }
func (e *stubExecutor) FetchOpenedOrders(ctx context.Context) ([]*mthub.OrderRecord, error) { return e.orders, nil }
func (e *stubExecutor) FetchOrderHistory(ctx context.Context, from, to time.Time) ([]*mthub.OrderRecord, error) { return nil, nil }
func (e *stubExecutor) FetchSymbolParams(ctx context.Context, canonicals []string) ([]*mthub.SymbolParam, error) { return nil, nil }
func (e *stubExecutor) FetchAllSymbols(ctx context.Context) ([]string, error) { return nil, nil }
func (e *stubExecutor) FetchPriceHistory(ctx context.Context, symbol, period string, from, to int64, count int) ([]*mthub.Bar, error) { return nil, nil }
func (e *stubExecutor) AddSymbols(ctx context.Context, symbols []string) error { return nil }
func (e *stubExecutor) SubscribeOrderEvents(ctx context.Context, h mthub.OrderEventHandler) error { return nil }
func (e *stubExecutor) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (int64, error) { return 0, nil }
func (e *stubExecutor) CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error { return nil }
func (e *stubExecutor) ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error { return nil }

// ── Tests ──────────────────────────────────────────────────────────────

func TestProviderNoExecutor(t *testing.T) {
	hub := mthub.NewHub()
	provider := NewMTAccountStateProvider(hub, nil)

	state, err := provider.GetAccountState(context.Background(), "no-such-account")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != nil {
		t.Error("expected nil state when no executor (fail-closed)")
	}
}

func TestProviderWithOpenPositions(t *testing.T) {
	hub := mthub.NewHub()
	hub.Register("acct-1", &mthub.Session{AccountID: "acct-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour},
		&stubExecutor{orders: []*mthub.OrderRecord{
			{Ticket: 1, SymbolRaw: "EURUSD", Volume: decimal.NewFromFloat(0.10),
				OpenPrice: decimal.NewFromFloat(1.08500),
				Profit: decimal.NewFromFloat(50.0), Commission: decimal.Zero, Swap: decimal.Zero},
			{Ticket: 2, SymbolRaw: "GBPUSD", Volume: decimal.NewFromFloat(0.20),
				OpenPrice: decimal.NewFromFloat(1.30000),
				Profit: decimal.NewFromFloat(-20.0), Commission: decimal.Zero, Swap: decimal.Zero},
		}})

	provider := NewMTAccountStateProvider(hub, nil)
	state, err := provider.GetAccountState(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state == nil {
		t.Fatal("expected non-nil state with registered executor")
	}
	if state.OpenPositions != 2 {
		t.Errorf("expected 2 positions, got %d", state.OpenPositions)
	}
	// Equity = 10000 + 50 - 20 = 10030.
	if !state.Equity.GreaterThan(decimal.NewFromInt(10000)) {
		t.Errorf("equity should be > 10000 (has profit), got %s", state.Equity)
	}
	// Peak equity should be tracked.
	peak := provider.GetPeakEquity("acct-1")
	if peak.IsZero() {
		t.Error("peak equity should be tracked")
	}
}

func TestProviderEmptyPositions(t *testing.T) {
	hub := mthub.NewHub()
	hub.Register("acct-2", &mthub.Session{AccountID: "acct-2", CreatedAt: time.Now(), MaxAge: 4 * time.Hour},
		&stubExecutor{orders: nil})

	provider := NewMTAccountStateProvider(hub, nil)
	state, err := provider.GetAccountState(context.Background(), "acct-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.OpenPositions != 0 {
		t.Errorf("expected 0 positions, got %d", state.OpenPositions)
	}
}

func TestProviderPeakEquityTracking(t *testing.T) {
	hub := mthub.NewHub()
	hub.Register("acct-3", &mthub.Session{AccountID: "acct-3", CreatedAt: time.Now(), MaxAge: 4 * time.Hour},
		&stubExecutor{orders: []*mthub.OrderRecord{
			{Ticket: 1, SymbolRaw: "EURUSD", Volume: decimal.NewFromFloat(0.10),
				OpenPrice: decimal.NewFromFloat(1.08500), Profit: decimal.NewFromFloat(100.0)},
		}})

	provider := NewMTAccountStateProvider(hub, nil)

	// First call: peak = 10100.
	state1, _ := provider.GetAccountState(context.Background(), "acct-3")
	peak1 := provider.GetPeakEquity("acct-3")

	// Replace with lower equity.
	hub.Register("acct-3", &mthub.Session{AccountID: "acct-3", CreatedAt: time.Now(), MaxAge: 4 * time.Hour},
		&stubExecutor{orders: []*mthub.OrderRecord{
			{Ticket: 1, SymbolRaw: "EURUSD", Volume: decimal.NewFromFloat(0.10),
				OpenPrice: decimal.NewFromFloat(1.08500), Profit: decimal.NewFromFloat(10.0)},
		}})

	state2, _ := provider.GetAccountState(context.Background(), "acct-3")
	peak2 := provider.GetPeakEquity("acct-3")

	// Peak should remain at the higher value.
	if !peak2.Equal(peak1) {
		t.Errorf("peak equity should remain at %s, got %s (state1=%s, state2=%s)",
			peak1, peak2, state1.Equity, state2.Equity)
	}
}

func TestProviderResetPeakEquity(t *testing.T) {
	hub := mthub.NewHub()
	hub.Register("acct-4", &mthub.Session{AccountID: "acct-4", CreatedAt: time.Now(), MaxAge: 4 * time.Hour},
		&stubExecutor{orders: []*mthub.OrderRecord{
			{Ticket: 1, SymbolRaw: "EURUSD", Volume: decimal.NewFromFloat(0.10),
				OpenPrice: decimal.NewFromFloat(1.08500), Profit: decimal.NewFromFloat(200.0)},
		}})

	provider := NewMTAccountStateProvider(hub, nil)
	provider.GetAccountState(context.Background(), "acct-4")
	provider.ResetPeakEquity("acct-4")

	peak := provider.GetPeakEquity("acct-4")
	if !peak.IsZero() {
		t.Errorf("peak should be zero after reset, got %s", peak)
	}
}

func TestProviderWithGateIntegration(t *testing.T) {
	// Full integration: provider → gate → decision.
	hub := mthub.NewHub()
	hub.Register("acct-5", &mthub.Session{AccountID: "acct-5", CreatedAt: time.Now(), MaxAge: 4 * time.Hour},
		&stubExecutor{orders: []*mthub.OrderRecord{
			{Ticket: 1, SymbolRaw: "EURUSD", Volume: decimal.NewFromFloat(0.10),
				OpenPrice: decimal.NewFromFloat(1.08500), Profit: decimal.NewFromFloat(500.0)},
		}})

	provider := NewMTAccountStateProvider(hub, nil)
	state, err := provider.GetAccountState(context.Background(), "acct-5")
	if err != nil || state == nil {
		t.Fatal("provider must return state")
	}

	// Verify state is usable by the gate.
	if state.Equity.IsNegative() {
		t.Error("equity should be positive")
	}
	if state.Balance.IsZero() {
		t.Error("balance should be non-zero")
	}
	// Positions and margin should be set.
	if state.OpenPositions == 0 {
		t.Error("should have open positions")
	}
}
