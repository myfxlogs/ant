package mdgateway

import (
	"context"
	"testing"
	"time"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// fakeGateway implements Gateway for testing Add/Remove/Health lifecycle.
type fakeGateway struct {
	platform   string
	accountID  string
	sessionID  string
	connected  bool
	failHealth bool
}

func (g *fakeGateway) Platform() string   { return g.platform }
func (g *fakeGateway) AccountID() string  { return g.accountID }
func (g *fakeGateway) SessionID() string  { return g.sessionID }

func (g *fakeGateway) Connect(ctx context.Context) error {
	g.connected = true
	return nil
}

func (g *fakeGateway) Disconnect(ctx context.Context) error {
	g.connected = false
	return nil
}

func (g *fakeGateway) Subscribe(ctx context.Context, symbols []string, handler mdtick.TickHandler) error {
	return nil
}

func (g *fakeGateway) SubscribeProfit(ctx context.Context, handler mdtick.ProfitHandler) error {
	return nil
}

func (g *fakeGateway) SubscribeOrderUpdate(ctx context.Context, handler mdtick.OrderUpdateHandler) error {
	return nil
}

func (g *fakeGateway) HealthCheck(ctx context.Context) error {
	if g.failHealth {
		return context.DeadlineExceeded
	}
	return nil
}

func TestManagerAddGateway(t *testing.T) {
	t.Parallel()
	mgr := NewManager(ManagerDeps{})
	gw := &fakeGateway{platform: "mt4", accountID: "acc-1"}

	if err := mgr.AddGateway(context.Background(), gw, nil); err != nil {
		t.Fatalf("AddGateway: %v", err)
	}

	// Duplicate add must fail.
	if err := mgr.AddGateway(context.Background(), gw, nil); err == nil {
		t.Fatal("duplicate AddGateway must return error")
	}

	// Verify gateway appears in Health output.
	health := mgr.Health()
	if len(health) != 1 {
		t.Fatalf("Health() after add: got %d entries, want 1", len(health))
	}
	h := health[0]
	if h.AccountID != "acc-1" || h.Platform != "mt4" {
		t.Fatalf("Health entry = %+v, want acc-1/mt4", h)
	}
	if h.State != "no_data" {
		t.Fatalf("Health state with no ticks = %q, want no_data", h.State)
	}

	t.Log("AddGateway: registered, duplicate rejected, health visible")
}

func TestManagerRemoveGateway(t *testing.T) {
	t.Parallel()
	mgr := NewManager(ManagerDeps{})
	gw := &fakeGateway{platform: "mt5", accountID: "acc-2"}
	mgr.AddGateway(context.Background(), gw, nil)

	if err := mgr.RemoveGateway(context.Background(), "acc-2"); err != nil {
		t.Fatalf("RemoveGateway: %v", err)
	}
	if gw.connected {
		t.Fatal("gateway must be disconnected after removal")
	}

	// Remove non-existent should be no-op.
	if err := mgr.RemoveGateway(context.Background(), "nonexistent"); err != nil {
		t.Fatalf("RemoveGateway nonexistent: %v", err)
	}

	if len(mgr.Health()) != 0 {
		t.Fatal("Health() after remove must be empty")
	}

	t.Log("RemoveGateway: disconnected, double remove no-op, health empty")
}

func TestManagerHealthStates(t *testing.T) {
	t.Parallel()
	mgr := NewManager(ManagerDeps{})
	gw1 := &fakeGateway{platform: "mt4", accountID: "acc-a"}
	gw2 := &fakeGateway{platform: "mt5", accountID: "acc-b"}
	mgr.AddGateway(context.Background(), gw1, nil)
	mgr.AddGateway(context.Background(), gw2, nil)

	// No ticks → no_data.
	for _, h := range mgr.Health() {
		if h.State != "no_data" {
			t.Fatalf("account %s state = %q, want no_data", h.AccountID, h.State)
		}
	}

	// Simulate recent tick → healthy.
	mgr.mu.Lock()
	mgr.lastTickAt["acc-a"] = time.Now().UnixMilli()
	mgr.mu.Unlock()

	h1 := mgr.Health()[0]
	if h1.AccountID == "acc-a" && h1.State != "healthy" {
		t.Fatalf("recent tick must be healthy, got %q", h1.State)
	}

	// Simulate stale tick (>5 min ago).
	mgr.mu.Lock()
	mgr.lastTickAt["acc-b"] = time.Now().UnixMilli() - 6*60*1000
	mgr.mu.Unlock()

	for _, h := range mgr.Health() {
		if h.AccountID == "acc-b" && h.State != "stale" {
			t.Fatalf("6m old tick must be stale, got %q", h.State)
		}
	}

	// Simulate dead tick (>15 min ago).
	mgr.mu.Lock()
	mgr.lastTickAt["acc-a"] = time.Now().UnixMilli() - 16*60*1000
	mgr.mu.Unlock()

	for _, h := range mgr.Health() {
		if h.AccountID == "acc-a" && h.State != "dead" {
			t.Fatalf("16m old tick must be dead, got %q", h.State)
		}
	}

	t.Log("Health states: no_data → healthy → stale → dead transitions verified")
}

func TestManagerGatewayCount(t *testing.T) {
	t.Parallel()
	mgr := NewManager(ManagerDeps{})
	for i := 0; i < 5; i++ {
		gw := &fakeGateway{accountID: "acc-" + string(rune('0'+i))}
		if err := mgr.AddGateway(context.Background(), gw, nil); err != nil {
			t.Fatalf("AddGateway %d: %v", i, err)
		}
	}
	if len(mgr.gateways) != 5 {
		t.Fatalf("gateway count = %d, want 5", len(mgr.gateways))
	}
	if len(mgr.Health()) != 5 {
		t.Fatalf("Health entries = %d, want 5", len(mgr.Health()))
	}
}
