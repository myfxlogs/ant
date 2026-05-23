package mthub

import (
	"context"
	"testing"

	"anttrader/internal/mdgateway"

	"go.uber.org/zap"
)

func TestNewMtHubService(t *testing.T) {
	s := NewMtHubService(nil, nil, nil, nil)
	if s == nil {
		t.Fatal("NewMtHubService returned nil")
	}
}

func TestMtHubServiceEnsureSession(t *testing.T) {
	gw := &mockGateway{platform: "mt5", sessionID: "s1", brokerID: "b1"}
	hub := NewHub(func(brokerID string) (mdgateway.Gateway, bool) {
		return gw, true
	}, nil, zap.NewNop())
	svc := NewMtHubService(hub, NewOrderEventBroker(), nil, zap.NewNop())

	res, err := svc.EnsureSession(context.Background(), "acc-1")
	if err != nil {
		t.Fatal(err)
	}
	if res.SessionID != "s1" {
		t.Errorf("SessionID = %s, want s1", res.SessionID)
	}
}

func TestMtHubServiceCloseSession(t *testing.T) {
	gw := &mockGateway{platform: "mt5", brokerID: "b1"}
	hub := NewHub(func(brokerID string) (mdgateway.Gateway, bool) {
		return gw, true
	}, nil, zap.NewNop())
	svc := NewMtHubService(hub, NewOrderEventBroker(), nil, zap.NewNop())

	hub.EnsureSession("acc-1", "b1")
	err := svc.CloseSession(context.Background(), "acc-1")
	if err != nil {
		t.Fatal(err)
	}
	if hub.SessionCount() != 0 {
		t.Error("session not removed")
	}
}

func TestMtHubService_OpsFailWithoutExecutor(t *testing.T) {
	svc := NewMtHubService(
		NewHub(func(string) (mdgateway.Gateway, bool) { return nil, false }, nil, zap.NewNop()),
		NewOrderEventBroker(), nil, zap.NewNop(),
	)

	// OrderSend should fail with "no order executor"
	_, err := svc.OrderSend(context.Background(), "acc-1", &OrderRequest{})
	if err == nil {
		t.Error("OrderSend: expected error without executor")
	}

	// OrderClose should fail
	_, err = svc.OrderClose(context.Background(), "acc-1", &CloseRequest{})
	if err == nil {
		t.Error("OrderClose: expected error without executor")
	}

	// SymbolParamsMany should fail with "no session"
	_, err = svc.SymbolParamsMany(context.Background(), "acc-1", nil)
	if err == nil {
		t.Error("SymbolParamsMany: expected error without session")
	}

	// PriceHistory should fail
	_, err = svc.PriceHistory(context.Background(), "acc-1", "EURUSD")
	if err == nil {
		t.Error("PriceHistory: expected error without session")
	}
}
