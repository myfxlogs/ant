package mthub

import (
	"context"
	"testing"
)

func TestSessionState_Connected(t *testing.T) {
	hub := NewHub()
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: Clk.Now()}, nil)
	svc := &MtHubService{hub: hub}

	state := svc.SessionState(context.Background(), "acc-1")
	if state != "connected" {
		t.Errorf("expected connected, got %s", state)
	}
}

func TestSessionState_NotFound(t *testing.T) {
	hub := NewHub()
	svc := &MtHubService{hub: hub}

	state := svc.SessionState(context.Background(), "no-such-account")
	if state != "not_found" {
		t.Errorf("expected not_found, got %s", state)
	}
}

func TestSessionState_ExpiredRefreshes(t *testing.T) {
	hub := NewHub()
	hub.Register("acc-old", &Session{
		AccountID: "acc-old",
		CreatedAt: Clk.Now().Add(-5 * 3600e9), // ~5 hours ago
		MaxAge:    1,                            // 1 nanosecond → immediately expired
	}, nil)
	svc := &MtHubService{hub: hub}

	// EnsureSession auto-refreshes expired sessions → returns connected.
	state := svc.SessionState(context.Background(), "acc-old")
	if state != "connected" {
		t.Errorf("expected connected (auto-refreshed), got %s", state)
	}
}
