//go:build integration

package mthub

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestDerivedQuantities(t *testing.T) {
	t.Parallel()
	cache := NewStateCache(nil, testLogger())

	// Populate cache with order fills from multiple accounts.
	cache.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "BUY", Volume: decimal.NewFromFloat(1.0), Price: decimal.NewFromFloat(1.0850),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	cache.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 2,
		Canonical: "GBPUSD", Side: "SELL", Volume: decimal.NewFromFloat(0.5), Price: decimal.NewFromFloat(1.2650),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	cache.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-2", Ticket: 3,
		Canonical: "USDJPY", Side: "BUY", Volume: decimal.NewFromFloat(2.0), Price: decimal.NewFromFloat(150.0),
		ToState: "FILLED", Timestamp: time.Now(),
	})

	computer := NewDerivedComputer(cache, 50*time.Millisecond)
	computer.Start()
	defer computer.Stop()

	// Wait for at least one recalc cycle.
	time.Sleep(200 * time.Millisecond)

	state := computer.State()
	accs, totalExposure, totalMargin, _, _, _, lastUpdated := state.Get()

	if len(accs) != 2 {
		t.Fatalf("want 2 accounts, got %d", len(accs))
	}
	if !totalExposure.GreaterThan(decimal.Zero) {
		t.Fatalf("totalExposure should be > 0, got %s", totalExposure.String())
	}
	if !totalMargin.GreaterThan(decimal.Zero) {
		t.Fatalf("totalMargin should be > 0, got %s", totalMargin.String())
	}
	if lastUpdated.IsZero() {
		t.Fatal("lastUpdated should not be zero")
	}

	// Verify per-account data.
	acc1 := state.GetAccount("acc-1")
	if acc1 == nil {
		t.Fatal("acc-1 should exist")
	}
	// EURUSD: 1.0 * 1.085 = 1.085 notional
	// GBPUSD: 0.5 * 1.265 = 0.6325 notional
	// Total: ~1.7175
	if !acc1.Exposure.GreaterThan(decimal.Zero) {
		t.Fatalf("acc-1 exposure should be > 0, got %s", acc1.Exposure.String())
	}

	acc2 := state.GetAccount("acc-2")
	if acc2 == nil {
		t.Fatal("acc-2 should exist")
	}

	t.Logf("DerivedQuantities: acc-1 exposure=%s margin=%s | acc-2 exposure=%s | total exposure=%s margin=%s",
		acc1.Exposure.String(), acc1.MarginUsed.String(), acc2.Exposure.String(), totalExposure.String(), totalMargin.String())
}
