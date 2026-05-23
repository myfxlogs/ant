package mthub

import (
	"context"
	"testing"

	"anttrader/internal/mdgateway"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// mockGateway implements mdgateway.Gateway for testing.
type mockGateway struct {
	platform  string
	conn      *grpc.ClientConn
	sessionID string
	brokerID  string
}

func (m *mockGateway) Platform() string                         { return m.platform }
func (m *mockGateway) Conn() *grpc.ClientConn                   { return m.conn }
func (m *mockGateway) SessionID() string                        { return m.sessionID }
func (m *mockGateway) BrokerID() string                         { return m.brokerID }
func (m *mockGateway) Connect(ctx context.Context) error        { return nil }
func (m *mockGateway) Disconnect(ctx context.Context) error     { return nil }
func (m *mockGateway) Subscribe(ctx context.Context, symbols []string, h mdgateway.TickHandler) error {
	return nil
}
func (m *mockGateway) HealthCheck(ctx context.Context) error { return nil }

// ── NewHub ──

func TestNewHub(t *testing.T) {
	h := NewHub(nil, nil, nil)
	if h == nil {
		t.Fatal("NewHub returned nil")
	}
	if h.sessions == nil {
		t.Fatal("sessions map not initialized")
	}
	if h.SessionCount() != 0 {
		t.Fatalf("expected 0 sessions, got %d", h.SessionCount())
	}
}

// ── EnsureSession: register + reuse ──

func TestHubEnsureSession(t *testing.T) {
	gw := &mockGateway{platform: "mt5", sessionID: "sess-1", brokerID: "b1"}
	lookup := func(brokerID string) (mdgateway.Gateway, bool) {
		if brokerID == "b1" {
			return gw, true
		}
		return nil, false
	}
	hub := NewHub(lookup, nil, zap.NewNop())

	// First call: should register session.
	ses, err := hub.EnsureSession("acc-1", "b1")
	if err != nil {
		t.Fatal(err)
	}
	if ses == nil {
		t.Fatal("expected non-nil session")
	}
	if ses.Platform() != "mt5" {
		t.Errorf("Platform = %s, want mt5", ses.Platform())
	}
	if ses.SessionID() != "sess-1" {
		t.Errorf("SessionID = %s, want sess-1", ses.SessionID())
	}

	// Second call: should return cached.
	ses2, err := hub.EnsureSession("acc-1", "b1")
	if err != nil {
		t.Fatal(err)
	}
	if ses2 != ses {
		t.Error("expected same session instance on second call")
	}

	if n := hub.SessionCount(); n != 1 {
		t.Errorf("SessionCount = %d, want 1", n)
	}

	// Missing broker.
	ses3, err := hub.EnsureSession("acc-2", "b2")
	if err != nil {
		t.Fatal(err)
	}
	if ses3 != nil {
		t.Error("expected nil session for missing broker")
	}
}

// ── CloseSession ──

func TestHubCloseSession(t *testing.T) {
	gw := &mockGateway{platform: "mt5", brokerID: "b1"}
	hub := NewHub(func(brokerID string) (mdgateway.Gateway, bool) {
		return gw, true
	}, nil, zap.NewNop())

	hub.EnsureSession("acc-1", "b1")
	if hub.SessionCount() != 1 {
		t.Fatalf("SessionCount = %d, want 1", hub.SessionCount())
	}
	hub.CloseSession("acc-1")
	if hub.SessionCount() != 0 {
		t.Errorf("SessionCount = %d after close, want 0", hub.SessionCount())
	}
}

// ── UpdatePrice + LatestBid/LatestAsk/LatestPriceForSide ──

func TestHubUpdatePrice(t *testing.T) {
	hub := NewHub(nil, nil, nil)

	hub.UpdatePrice("EURUSD", 1.0850, 1.0852)

	if bid := hub.LatestBid("EURUSD"); bid != 1.0850 {
		t.Errorf("LatestBid = %f, want 1.0850", bid)
	}
	if ask := hub.LatestAsk("EURUSD"); ask != 1.0852 {
		t.Errorf("LatestAsk = %f, want 1.0852", ask)
	}
}

func TestHubLatestPriceForSide(t *testing.T) {
	hub := NewHub(nil, nil, nil)

	hub.UpdatePrice("EURUSD", 1.0850, 1.0852)

	// buy side → bid
	if p := hub.LatestPriceForSide("EURUSD", "buy"); p != 1.0850 {
		t.Errorf("buy side price = %f, want 1.0850", p)
	}
	// sell side → ask
	if p := hub.LatestPriceForSide("EURUSD", "sell"); p != 1.0852 {
		t.Errorf("sell side price = %f, want 1.0852", p)
	}
	// unknown symbol → 0
	if p := hub.LatestPriceForSide("XAUUSD", "buy"); p != 0 {
		t.Errorf("unknown symbol price = %f, want 0", p)
	}
}

func TestHubUpdatePriceOverwrite(t *testing.T) {
	hub := NewHub(nil, nil, nil)

	hub.UpdatePrice("EURUSD", 1.0850, 1.0852)
	hub.UpdatePrice("EURUSD", 1.0860, 1.0862)

	if bid := hub.LatestBid("EURUSD"); bid != 1.0860 {
		t.Errorf("LatestBid after overwrite = %f, want 1.0860", bid)
	}
}

func TestHubLatestPriceForSideFallback(t *testing.T) {
	hub := NewHub(nil, nil, nil)

	// Only bid available, no ask
	hub.UpdatePrice("EURUSD", 1.0850, 0)

	// sell side should fall back to bid when ask is 0
	if p := hub.LatestPriceForSide("EURUSD", "sell"); p != 1.0850 {
		t.Errorf("sell side fallback price = %f, want 1.0850", p)
	}
}

// ── ActiveSessions ──

func TestHubActiveSessions(t *testing.T) {
	hub := NewHub(func(brokerID string) (mdgateway.Gateway, bool) {
		return &mockGateway{platform: "mt5", brokerID: brokerID}, true
	}, nil, zap.NewNop())

	hub.EnsureSession("a1", "b1")
	hub.EnsureSession("a2", "b2")
	hub.EnsureSession("a3", "b3")

	active := hub.ActiveSessions()
	if active["mt5"] != 3 {
		t.Errorf("mt5 = %d, want 3", active["mt5"])
	}
}

func TestHubActiveSessionsEmpty(t *testing.T) {
	hub := NewHub(nil, nil, nil)
	active := hub.ActiveSessions()
	if len(active) != 0 {
		t.Errorf("expected empty map, got %d entries", len(active))
	}
}
