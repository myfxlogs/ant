package mthub

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestIsSessionError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		err  string
		want bool
	}{
		{"Invalid account or password", true},
		{"session not connected", true},
		{"DeadlineExceeded", true},
		{"connection reset by peer", true},
		{"transport is closing", true},
		{"rpc error: code = Unavailable", true},
		{"some other error", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isSessionError(errors.New(tt.err))
		if got != tt.want {
			t.Errorf("isSessionError(%q) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestReconnectAndRetry_NoReconnectGateway(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{logger: zap.NewNop(), hub: &Hub{}}
	called := false
	err := svc.reconnectAndRetry("acc-1", func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("op should be called when ReconnectGateway is nil")
	}
}

func TestReconnectAndRetry_Cooldown(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{
		logger: zap.NewNop(),
		hub: &Hub{
			ReconnectGateway: func(_ context.Context, _ string) error { return nil },
		},
		reconnectLastAt: map[string]time.Time{},
	}
	svc.reconnectLastAt["acc-1"] = Clk.Now()
	called := false
	err := svc.reconnectAndRetry("acc-1", func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("op should still be called during cooldown")
	}
}

func TestReconnectAndRetry_Success(t *testing.T) {
	t.Parallel()
	reconnectCalled := false
	svc := &MtHubService{
		logger: zap.NewNop(),
		hub: &Hub{
			ReconnectGateway: func(_ context.Context, _ string) error {
				reconnectCalled = true
				return nil
			},
		},
		reconnectLastAt: map[string]time.Time{},
	}
	opCalled := false
	err := svc.reconnectAndRetry("acc-1", func() error {
		opCalled = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reconnectCalled {
		t.Fatal("ReconnectGateway should be called")
	}
	if !opCalled {
		t.Fatal("op should be called after successful reconnect")
	}
}

func TestReconnectAndRetry_ReconnectFails(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{
		logger: zap.NewNop(),
		hub: &Hub{
			ReconnectGateway: func(_ context.Context, _ string) error {
				return errors.New("reconnect failed")
			},
		},
		reconnectLastAt: map[string]time.Time{},
	}
	opCalled := false
	err := svc.reconnectAndRetry("acc-1", func() error {
		opCalled = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opCalled {
		t.Fatal("op should still be called even if reconnect fails")
	}
}

func TestOpenedOrders_SessionErrorReconnect(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	reconnectCalled := false
	svc.hub.ReconnectGateway = func(_ context.Context, _ string) error {
		reconnectCalled = true
		return nil
	}
	exec := &mockExecutor{
		platform: "MT5",
		fetchOpenedOrdersFn: func(_ context.Context) ([]*OrderRecord, error) {
			return nil, errors.New("rpc error: code = Unavailable")
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	_, err := svc.OpenedOrders(context.Background(), "acc-1")
	if err == nil {
		t.Fatal("expected error from session failure")
	}
	if !reconnectCalled {
		t.Fatal("ReconnectGateway should be called on session error")
	}
}

func TestSymbolParams_SessionErrorReconnect(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	reconnectCalled := false
	svc.hub.ReconnectGateway = func(_ context.Context, _ string) error {
		reconnectCalled = true
		return nil
	}
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(_ context.Context, _ []string) ([]*SymbolParam, error) {
			return nil, errors.New("transport is closing")
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	_, err := svc.SymbolParams(context.Background(), "acc-1", []string{"EURUSD"})
	if err == nil {
		t.Fatal("expected error from session failure")
	}
	if !reconnectCalled {
		t.Fatal("ReconnectGateway should be called on session error")
	}
}
