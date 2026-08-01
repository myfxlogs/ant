package oms

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/internal/mthub"
)

// mockExecutor implements mthub.OrderExecutor for testing.
type mockExecutor struct {
	platform     string
	placeTicket  int64
	placeErr     error
	openedOrders []*mthub.OrderRecord
	closeErr     error
	modifyErr    error
}

func (m *mockExecutor) Platform() string { return m.platform }
func (m *mockExecutor) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
	return m.placeTicket, m.placeErr
}
func (m *mockExecutor) CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error {
	return m.closeErr
}
func (m *mockExecutor) DeleteOrder(ctx context.Context, ticket int64) error { return nil }
func (m *mockExecutor) ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
	return m.modifyErr
}
func (m *mockExecutor) FetchOpenedOrders(ctx context.Context) ([]*mthub.OrderRecord, error) {
	return m.openedOrders, nil
}
func (m *mockExecutor) FetchOrderHistory(ctx context.Context, from, to time.Time) ([]*mthub.OrderRecord, error) {
	return nil, nil
}
func (m *mockExecutor) FetchSymbolParams(ctx context.Context, canonicals []string) ([]*mthub.SymbolParam, error) {
	return nil, nil
}
func (m *mockExecutor) FetchAllSymbols(ctx context.Context) ([]string, error) { return nil, nil }
func (m *mockExecutor) FetchPriceHistory(ctx context.Context, symbol, period string, from, to int64, count int) ([]*mthub.Bar, error) {
	return nil, nil
}
func (m *mockExecutor) AddSymbols(ctx context.Context, symbols []string) error { return nil }
func (m *mockExecutor) SubscribeOrderEvents(ctx context.Context, h mthub.OrderEventHandler) error {
	return nil
}

func TestNewMTBrokerAdapter(t *testing.T) {
	t.Parallel()
	a := NewMTBrokerAdapter(&mockExecutor{platform: "mt5"})
	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
}

func TestMTBrokerAdapter_Submit(t *testing.T) {
	t.Parallel()
	exec := &mockExecutor{platform: "mt5", placeTicket: 12345}
	a := NewMTBrokerAdapter(exec)

	resp, err := a.Submit(context.Background(), &OrderRequest{
		AccountID: "acc-1",
		Symbol:    "EURUSD",
		Side:      "buy",
		Volume:    decF(0.1),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ticket != "12345" {
		t.Fatalf("expected ticket 12345, got %s", resp.Ticket)
	}
	if resp.State != StateSubmitted {
		t.Fatalf("expected state SUBMITTED, got %s", resp.State)
	}
}

func TestMTBrokerAdapter_Submit_Sell(t *testing.T) {
	t.Parallel()
	exec := &mockExecutor{platform: "mt5", placeTicket: 99}
	a := NewMTBrokerAdapter(exec)

	resp, err := a.Submit(context.Background(), &OrderRequest{
		AccountID: "acc-1",
		Symbol:    "EURUSD",
		Side:      "sell",
		Volume:    decF(0.1),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ticket != "99" {
		t.Fatalf("expected ticket 99, got %s", resp.Ticket)
	}
}

func TestMTBrokerAdapter_Cancel(t *testing.T) {
	t.Parallel()
	exec := &mockExecutor{
		platform: "mt5",
		openedOrders: []*mthub.OrderRecord{
			{Ticket: 100, Volume: decF(0.5), State: mthub.OrderStateOpen},
		},
	}
	a := NewMTBrokerAdapter(exec)

	if err := a.Cancel(context.Background(), "100"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMTBrokerAdapter_Cancel_NotFound(t *testing.T) {
	t.Parallel()
	exec := &mockExecutor{
		platform:     "mt5",
		openedOrders: []*mthub.OrderRecord{},
	}
	a := NewMTBrokerAdapter(exec)

	err := a.Cancel(context.Background(), "999")
	if err == nil {
		t.Fatal("expected error for not-found order")
	}
}

func TestMTBrokerAdapter_Cancel_InvalidTicket(t *testing.T) {
	t.Parallel()
	a := NewMTBrokerAdapter(&mockExecutor{})

	err := a.Cancel(context.Background(), "abc")
	if err == nil {
		t.Fatal("expected error for invalid ticket")
	}
}

func TestMTBrokerAdapter_Modify(t *testing.T) {
	t.Parallel()
	exec := &mockExecutor{platform: "mt5"}
	a := NewMTBrokerAdapter(exec)

	if err := a.Modify(context.Background(), "100", decF(1.1), decF(1.0)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMTBrokerAdapter_Modify_InvalidTicket(t *testing.T) {
	t.Parallel()
	a := NewMTBrokerAdapter(&mockExecutor{})

	err := a.Modify(context.Background(), "abc", decF(1.1), decF(1.0))
	if err == nil {
		t.Fatal("expected error for invalid ticket")
	}
}

func TestMTBrokerAdapter_Query(t *testing.T) {
	t.Parallel()
	exec := &mockExecutor{
		platform: "mt5",
		openedOrders: []*mthub.OrderRecord{
			{Ticket: 200, Canonical: "EURUSD", Volume: decF(0.1), OpenPrice: decF(1.085), State: mthub.OrderStateOpen, AccountID: "acc-1"},
		},
	}
	a := NewMTBrokerAdapter(exec)

	order, err := a.Query(context.Background(), "200")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.Ticket != "200" {
		t.Fatalf("expected ticket 200, got %s", order.Ticket)
	}
	if order.Symbol != "EURUSD" {
		t.Fatalf("expected symbol EURUSD, got %s", order.Symbol)
	}
	if order.State != StateWorking {
		t.Fatalf("expected state WORKING, got %s", order.State)
	}
}

func TestMTBrokerAdapter_Query_NotFound(t *testing.T) {
	t.Parallel()
	exec := &mockExecutor{
		platform:     "mt5",
		openedOrders: []*mthub.OrderRecord{},
	}
	a := NewMTBrokerAdapter(exec)

	_, err := a.Query(context.Background(), "999")
	if err == nil {
		t.Fatal("expected error for not-found order")
	}
}
