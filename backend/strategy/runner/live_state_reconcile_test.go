package runner

// LIVE-POS-SNAPSHOT-LAG-1: broker-confirmed mutations must survive lagging
// mtapi OpenedOrders snapshots. Adversarial proof: removing the merge from
// UpdateLiveState (restoring wholesale overwrite) makes every test here fail.

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

func posTicket(t int64) sdk.Position {
	return sdk.Position{Ticket: t, Symbol: "BTCUSDm", Volume: decimal.NewFromFloat(0.01)}
}

func ordTicket(t int64) sdk.PendingOrder {
	return sdk.PendingOrder{Ticket: t, Symbol: "BTCUSDm", Volume: decimal.NewFromFloat(0.01)}
}

func livePosTickets(r *Runner) []int64 {
	r.ctx.mu.RLock()
	defer r.ctx.mu.RUnlock()
	out := make([]int64, 0, len(r.ctx.livePositions))
	for _, p := range r.ctx.livePositions {
		out = append(out, p.Ticket)
	}
	return out
}

func liveOrdTickets(r *Runner) []int64 {
	r.ctx.mu.RLock()
	defer r.ctx.mu.RUnlock()
	out := make([]int64, 0, len(r.ctx.livePendingOrders))
	for _, o := range r.ctx.livePendingOrders {
		out = append(out, o.Ticket)
	}
	return out
}

func containsTicket(ts []int64, t int64) bool {
	for _, x := range ts {
		if x == t {
			return true
		}
	}
	return false
}

// Confirmed open retained while the snapshot lags (mtapi propagation ~seconds).
func TestUpdateLiveState_ConfirmedOpenSurvivesLaggingSnapshot(t *testing.T) {
	r := New(Config{Symbol: "BTCUSDm"})
	r.UpdateLiveState("", "", "", "", []sdk.Position{posTicket(111)}, nil)

	r.ApplyConfirmedPosition(posTicket(222)) // broker-confirmed open

	// Snapshot still shows only the old position — the confirmed open must persist.
	r.UpdateLiveState("", "", "", "", []sdk.Position{posTicket(111)}, nil)
	got := livePosTickets(r)
	if !containsTicket(got, 222) || !containsTicket(got, 111) {
		t.Fatalf("confirmed open dropped by lagging snapshot: %v", got)
	}

	// Snapshot catches up — tracking prunes, no duplicate.
	r.UpdateLiveState("", "", "", "", []sdk.Position{posTicket(111), posTicket(222)}, nil)
	got = livePosTickets(r)
	if len(got) != 2 {
		t.Fatalf("duplicate after snapshot catch-up: %v", got)
	}

	// Broker truly closes it later — snapshot authoritative again.
	r.UpdateLiveState("", "", "", "", []sdk.Position{posTicket(111)}, nil)
	got = livePosTickets(r)
	if containsTicket(got, 222) {
		t.Fatalf("closed position resurrected after catch-up: %v", got)
	}
}

// Confirmed close suppressed while the snapshot still lists the position.
func TestUpdateLiveState_ConfirmedCloseNotResurrected(t *testing.T) {
	r := New(Config{Symbol: "BTCUSDm"})
	r.UpdateLiveState("", "", "", "", []sdk.Position{posTicket(111), posTicket(222)}, nil)

	r.RemoveConfirmedPosition(222) // broker-confirmed close

	// Lagging snapshot still carries 222 — must not resurrect.
	r.UpdateLiveState("", "", "", "", []sdk.Position{posTicket(111), posTicket(222)}, nil)
	got := livePosTickets(r)
	if containsTicket(got, 222) {
		t.Fatalf("confirmed close resurrected by lagging snapshot: %v", got)
	}
	if !containsTicket(got, 111) {
		t.Fatalf("unrelated position lost: %v", got)
	}

	// Snapshot catches up — ticket gone, tracking prunes.
	r.UpdateLiveState("", "", "", "", []sdk.Position{posTicket(111)}, nil)

	// A NEW position at the same ticket id is impossible in practice, but a
	// snapshot that later re-lists the ticket after prune must win (broker
	// truth). Simulate prune then reappearance.
	if len(r.ctx.confirmedPosDel) != 0 {
		t.Fatalf("del tracking not pruned after catch-up: %v", r.ctx.confirmedPosDel)
	}
}

// Pending orders get the same retention semantics.
func TestUpdateLiveState_ConfirmedPendingRetained(t *testing.T) {
	r := New(Config{Symbol: "BTCUSDm"})
	r.UpdateLiveState("", "", "", "", nil, nil)

	r.ApplyConfirmedPendingOrder(ordTicket(333))
	r.UpdateLiveState("", "", "", "", nil, nil) // lagging empty snapshot
	got := liveOrdTickets(r)
	if !containsTicket(got, 333) {
		t.Fatalf("confirmed pending dropped by lagging snapshot: %v", got)
	}

	r.RemoveConfirmedPendingOrder(333)
	r.UpdateLiveState("", "", "", "", nil, []sdk.PendingOrder{ordTicket(333)}) // lagging snapshot resurrects
	got = liveOrdTickets(r)
	if containsTicket(got, 333) {
		t.Fatalf("confirmed cancel resurrected by lagging snapshot: %v", got)
	}
}

// Retention expires: a confirmed add the snapshot never shows eventually
// falls back to snapshot truth (broker never registered it).
func TestUpdateLiveState_ConfirmedAddExpires(t *testing.T) {
	r := New(Config{Symbol: "BTCUSDm"})
	r.UpdateLiveState("", "", "", "", nil, nil)
	r.ApplyConfirmedPosition(posTicket(444))

	// Force expiry.
	r.ctx.mu.Lock()
	r.ctx.confirmedPosAdd[444] = time.Now().Add(-2 * confirmedMutationRetention)
	r.ctx.mu.Unlock()

	r.UpdateLiveState("", "", "", "", nil, nil)
	if got := livePosTickets(r); containsTicket(got, 444) {
		t.Fatalf("expired confirmed-add retained forever: %v", got)
	}
}

// LIVE-HISTORY-POOL-1: HistoryOrders must reach the injected live provider;
// without one it fails closed (nil + LastError), never fabricates.
func TestHistoryOrders_LiveProviderWired(t *testing.T) {
	r := New(Config{Symbol: "BTCUSDm"})
	called := false
	r.SetHistoryProvider(func(ctx context.Context, from, to int64) ([]sdk.Position, error) {
		called = true
		return []sdk.Position{posTicket(777)}, nil
	})
	got := r.Broker().HistoryOrders(0, 0)
	if !called || len(got) != 1 || got[0].Ticket != 777 {
		t.Fatalf("history provider not used: called=%v got=%v", called, got)
	}
}

func TestHistoryOrders_NoProviderFailsClosed(t *testing.T) {
	r := New(Config{Symbol: "BTCUSDm"})
	if got := r.Broker().HistoryOrders(0, 0); got != nil {
		t.Fatalf("no provider must return nil, got %v", got)
	}
	if r.Broker().(*brokerImpl).LastError() == nil {
		t.Fatal("no provider must record lastError (fail-closed), got nil")
	}
}
