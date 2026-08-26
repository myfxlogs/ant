// broker_trade_context_test.go — VM-TRADE-CONTEXT-2 brokerImpl error recording tests.
//
// These tests verify that brokerImpl records query errors to lastError
// so Runner can fail-closed (VM-TRADE-CONTEXT-2 S13/S14).

package runner

import (
	"context"
	"errors"
	"testing"

	"alphaforge/strategy/sdk"
)

// failingExecutor is a mockExecutor that returns errors for query methods.
type failingExecutor struct {
	mockExecutor
	openedErr   error
	pendingErr  error
}

func (m *failingExecutor) OpenedOrders(ctx context.Context) ([]sdk.Position, error) {
	return nil, m.openedErr
}

func (m *failingExecutor) PendingOrders(ctx context.Context) ([]sdk.PendingOrder, error) {
	return nil, m.pendingErr
}

// TestBrokerImpl_PositionsQueryError_RecordsLastError verifies that when
// executor.OpenedOrders returns an error, brokerImpl.Positions records it
// to lastError (VM-TRADE-CONTEXT-2).
//
// Adversarial: restore silent error swallow (`_ = err; return nil`)
// → LastError stays nil → RED.
func TestBrokerImpl_PositionsQueryError_RecordsLastError(t *testing.T) {
	r := New(Config{})
	exec := &failingExecutor{
		openedErr: errors.New("connection refused"),
	}
	r.broker.executor = exec
	r.broker.resetError()

	positions := r.broker.Positions(0)
	if positions != nil {
		t.Fatalf("Positions() = %v, want nil (on error)", positions)
	}
	err := r.broker.LastError()
	if err == nil {
		t.Fatal("LastError() = nil, want error (Positions query error not recorded)")
	}
}

// TestBrokerImpl_OrdersQueryError_RecordsLastError verifies that when
// executor.PendingOrders returns an error, brokerImpl.Orders records it
// to lastError (VM-TRADE-CONTEXT-2).
func TestBrokerImpl_OrdersQueryError_RecordsLastError(t *testing.T) {
	r := New(Config{})
	exec := &failingExecutor{
		pendingErr: errors.New("timeout"),
	}
	r.broker.executor = exec
	r.broker.resetError()

	orders := r.broker.Orders(0)
	if orders != nil {
		t.Fatalf("Orders() = %v, want nil (on error)", orders)
	}
	err := r.broker.LastError()
	if err == nil {
		t.Fatal("LastError() = nil, want error (Orders query error not recorded)")
	}
}

// TestBrokerImpl_HistoryOrders_NotAvailable_RecordsError verifies that
// in live mode (executor != nil), HistoryOrders records an error because
// it's not available (VM-TRADE-CONTEXT-2).
func TestBrokerImpl_HistoryOrders_NotAvailable_RecordsError(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{}
	r.broker.executor = exec
	r.broker.resetError()

	history := r.broker.HistoryOrders(0, 0)
	if history != nil {
		t.Fatalf("HistoryOrders() = %v, want nil (not available in live)", history)
	}
	err := r.broker.LastError()
	if err == nil {
		t.Fatal("LastError() = nil, want error (HistoryOrders not available in live mode)")
	}
}

// TestRunner_OnBar_FailClosed_OnBrokerError verifies that Runner.OnBar
// returns an error when brokerImpl has a recorded LastError after strategy
// execution (VM-TRADE-CONTEXT-2 S14).
//
// Adversarial: remove the LastError check in OnBar → returns nil error → RED.
func TestRunner_OnBar_FailClosed_OnBrokerError(t *testing.T) {
	r := New(Config{})
	exec := &failingExecutor{
		openedErr: errors.New("connection refused"),
	}
	r.broker.executor = exec

	// Use a minimal strategy that calls Positions (triggering the error).
	strat := &errorTriggerStrategy{}
	r.SetStrategy(strat)

	_, err := r.OnBar(context.Background(), nil, "M5")
	if err == nil {
		t.Fatal("OnBar returned nil error, want fail-closed error on broker query failure")
	}
}

// errorTriggerStrategy is a minimal strategy that queries broker.Positions
// in OnBar, triggering a broker query error.
type errorTriggerStrategy struct{}

func (s *errorTriggerStrategy) OnInit(ctx sdk.Context) error { return nil }
func (s *errorTriggerStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
	// Trigger a Positions query which will fail.
	ctx.Broker().Positions(0)
	return nil, nil
}
func (s *errorTriggerStrategy) OnDeinit(ctx sdk.Context, reason string) error { return nil }
