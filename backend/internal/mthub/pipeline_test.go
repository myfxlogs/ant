package mthub

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/internal/risksvc"
)

type mockRiskPipeline struct {
	allow  bool
	stage  string
	reason string
	lots   float64
}

func (m *mockRiskPipeline) Process(_ context.Context, _ *risksvc.SignalRequest) *risksvc.SignalResult {
	return &risksvc.SignalResult{
		Allowed: m.allow,
		Stage:   m.stage,
		Reason:  m.reason,
		Lots:    m.lots,
		Method:  "mock",
	}
}

func TestSetRiskPipeline(t *testing.T) {
	svc := &MtHubService{}
	if svc.riskPipeline != nil {
		t.Error("expected nil riskPipeline by default")
	}

	p := &mockRiskPipeline{allow: true, stage: "complete", lots: 0.1}
	svc.SetRiskPipeline(p)

	if svc.riskPipeline == nil {
		t.Fatal("expected non-nil riskPipeline after SetRiskPipeline")
	}
}

func TestRiskPipeline_Reject(t *testing.T) {
	svc := &MtHubService{}
	p := &mockRiskPipeline{
		allow:  false,
		stage:  "hardlimit",
		reason: "jurisdiction restricted",
	}
	svc.SetRiskPipeline(p)

	req := &OrderRequest{
		AccountID: "test-account",
		Canonical: "EURUSD",
		Side:      SideBuy,
	}
	_, err := svc.PlaceOrder(context.Background(), req)
	if err == nil {
		t.Fatal("expected error from rejected pipeline, got nil")
	}
	expected := "risk rejected at hardlimit: jurisdiction restricted"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestRiskPipeline_Allowed(t *testing.T) {
	svc := &MtHubService{}

	// Register a mock executor so PlaceOrder can call exec.PlaceOrder.
	hub := NewHub()
	mockExec := &mockExecutor{}
	hub.Register("test-account", &Session{AccountID: "test-account"}, mockExec)
	svc.hub = hub

	p := &mockRiskPipeline{
		allow: true,
		stage: "complete",
		lots:  0.25,
	}
	svc.SetRiskPipeline(p)

	req := &OrderRequest{
		AccountID: "test-account",
		Canonical: "EURUSD",
		Side:      SideBuy,
		Volume:    decimal.NewFromFloat(1.0),
		Price:     decimal.NewFromFloat(1.12345),
	}
	order, err := svc.PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if order.Ticket != 99999 {
		t.Errorf("expected ticket 99999, got %d", order.Ticket)
	}
	// Volume should have been overridden by sizer's lots.
	if req.Volume.InexactFloat64() != 0.25 {
		t.Errorf("expected volume 0.25 (from sizer), got %v", req.Volume)
	}
}

// mockExecutor implements OrderExecutor for testing.
type mockExecutor struct{}

func (m *mockExecutor) Platform() string                        { return "mock" }
func (m *mockExecutor) PlaceOrder(_ context.Context, _ *OrderRequest) (int64, error) { return 99999, nil }
func (m *mockExecutor) CloseOrder(_ context.Context, _ int64, _ decimal.Decimal) error { return nil }
func (m *mockExecutor) ModifyOrder(_ context.Context, _ int64, _, _, _ decimal.Decimal) error { return nil }
func (m *mockExecutor) FetchOpenedOrders(_ context.Context) ([]*OrderRecord, error)  { return nil, nil }
func (m *mockExecutor) FetchOrderHistory(_ context.Context, _, _ time.Time) ([]*OrderRecord, error) { return nil, nil }
func (m *mockExecutor) FetchSymbolParams(_ context.Context, _ []string) ([]*SymbolParam, error) { return nil, nil }
func (m *mockExecutor) SubscribeOrderEvents(_ context.Context, _ OrderEventHandler) error { return nil }
