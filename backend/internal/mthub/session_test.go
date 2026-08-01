package mthub

import (
	"context"
	"testing"
	"time"
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
	if state != "not_found" {
		t.Errorf("expected not_found for expired, got %s", state)
	}
}

func TestHub_WaitSession_Existing(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: Clk.Now()}, nil)
	ch := hub.WaitSession("acc-1")
	select {
	case <-ch:
		// expected: already closed
	default:
		t.Fatal("expected channel to be closed for existing session")
	}
}

func TestHub_WaitSession_Pending(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	ch := hub.WaitSession("acc-pending")
	select {
	case <-ch:
		t.Fatal("channel should not be closed for pending session")
	default:
		// expected
	}
	// Register → channel should close
	hub.Register("acc-pending", &Session{AccountID: "acc-pending", CreatedAt: Clk.Now()}, nil)
	select {
	case <-ch:
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("channel should close after session is registered")
	}
}

func TestHub_RemoveSession(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: Clk.Now()}, nil)
	hub.RemoveSession("acc-1")
	if _, err := hub.EnsureSession(context.Background(), "acc-1"); err == nil {
		t.Fatal("expected error after removal")
	}
}

func TestHub_CloseSession(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: Clk.Now()}, nil)
	if err := hub.CloseSession(context.Background(), "acc-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := hub.EnsureSession(context.Background(), "acc-1"); err == nil {
		t.Fatal("expected error after close")
	}
}

func TestHub_ActiveAccountIDs(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: Clk.Now()}, nil)
	hub.Register("acc-2", &Session{AccountID: "acc-2", CreatedAt: Clk.Now()}, nil)
	ids := hub.ActiveAccountIDs()
	if len(ids) != 2 {
		t.Fatalf("expected 2 active IDs, got %d", len(ids))
	}
}

func TestHub_ActiveAccountIDs_Empty(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	ids := hub.ActiveAccountIDs()
	if len(ids) != 0 {
		t.Fatalf("expected 0 active IDs, got %d", len(ids))
	}
}

func TestHubError_Error(t *testing.T) {
	t.Parallel()
	e := &HubError{Msg: "test error"}
	if e.Error() != "mthub: test error" {
		t.Fatalf("expected 'mthub: test error', got %s", e.Error())
	}
}

func TestErrSessionNotFound(t *testing.T) {
	t.Parallel()
	if ErrSessionNotFound.Error() != "mthub: session not found" {
		t.Fatalf("unexpected error message: %s", ErrSessionNotFound.Error())
	}
}
