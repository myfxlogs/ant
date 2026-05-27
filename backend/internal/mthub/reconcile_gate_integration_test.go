//go:build integration

package mthub

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func TestReconcileBeforeAccept(t *testing.T) {
	t.Parallel()
	gate := NewReconcileGate()
	hub := NewHub()

	// Simulate: account connects → enters reconciling
	gate.EnterReconciling("test-account")

	svc := NewMtHubService(hub, nil, nil, nil, nil, gate, nil)

	req := &OrderRequest{
		AccountID: "test-account",
		Canonical: "EURUSD",
		Side:      SideBuy,
		OrderType: OrderMarket,
		Volume:    decimal.NewFromFloat(0.01),
		ClientID:  "client-1",
	}

	// PlaceOrder should be rejected while reconciling.
	_, err := svc.PlaceOrder(context.Background(), req)
	if err == nil {
		t.Fatal("PlaceOrder should fail while account is reconciling")
	}
	if !errors.Is(err, ErrReconciling) {
		t.Fatalf("expected ErrReconciling, got %v", err)
	}

	// Mark reconciled → should now accept (though executor is nil, different error).
	gate.MarkReconciled("test-account")

	_, err = svc.PlaceOrder(context.Background(), req)
	if err == nil {
		t.Fatal("PlaceOrder should fail with session not found after reconcile")
	}
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound after reconcile, got %v", err)
	}
}
