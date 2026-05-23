package mthub

import (
	"context"
	"testing"

	"anttrader/internal/oms"
)

// stubBrokerAdapter implements oms.BrokerAdapter for testing.
type stubBrokerAdapter struct {
	submitErr error
	ticket    string
}

func (s *stubBrokerAdapter) Submit(ctx context.Context, req *oms.OrderRequest) (*oms.BrokerResp, error) {
	if s.submitErr != nil {
		return nil, s.submitErr
	}
	return &oms.BrokerResp{
		Ticket:    s.ticket,
		State:     oms.StateSubmitted,
		FilledQty: req.Volume,
		FillPrice: req.Price,
	}, nil
}

func (s *stubBrokerAdapter) Cancel(ctx context.Context, ticket string) error { return nil }
func (s *stubBrokerAdapter) Modify(ctx context.Context, ticket string, price, stopPrice float64) error {
	return nil
}
func (s *stubBrokerAdapter) Query(ctx context.Context, ticket string) (*oms.Order, error) {
	return nil, nil
}

func TestNewBrokerOrderExecutor(t *testing.T) {
	e := NewBrokerOrderExecutor(&stubBrokerAdapter{ticket: "123"})
	if e == nil {
		t.Fatal("NewBrokerOrderExecutor returned nil")
	}
}

func TestBrokerOrderExecutor_PlaceOrder(t *testing.T) {
	e := NewBrokerOrderExecutor(&stubBrokerAdapter{ticket: "123"})
	ticket, err := e.PlaceOrder(context.Background(), nil, "mt5", "s1", &OrderRequest{
		Symbol: "EURUSD", Side: "buy", Volume: 0.1, Price: 1.1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ticket != 123 {
		t.Fatalf("expected ticket=123, got %d", ticket)
	}
}

func TestBrokerOrderExecutor_PlaceOrderError(t *testing.T) {
	e := NewBrokerOrderExecutor(&stubBrokerAdapter{
		submitErr: context.DeadlineExceeded,
	})
	_, err := e.PlaceOrder(context.Background(), nil, "mt5", "s1", &OrderRequest{})
	if err == nil {
		t.Fatal("expected error from broker")
	}
}

func TestBrokerOrderExecutor_CloseOrder(t *testing.T) {
	e := NewBrokerOrderExecutor(&stubBrokerAdapter{})
	err := e.CloseOrder(context.Background(), nil, "mt4", "s1", 123, 0.1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBrokerOrderExecutor_NoAdapter(t *testing.T) {
	e := NewBrokerOrderExecutor(nil)
	_, err := e.PlaceOrder(context.Background(), nil, "", "", &OrderRequest{})
	if err == nil {
		t.Fatal("expected error without adapter")
	}
}

func TestParseTicket(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"123", 123},
		{"ticket-456", 456},
		{"", 0},
		{"abc", 0},
	}
	for _, tt := range tests {
		got := parseTicket(tt.input)
		if got != tt.want {
			t.Errorf("parseTicket(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
