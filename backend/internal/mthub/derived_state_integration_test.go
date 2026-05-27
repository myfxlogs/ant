//go:build integration

package mthub

import (
	"testing"
	"time"
)

func TestDerivedQuantities(t *testing.T) {
	t.Parallel()
	cache := NewStateCache(nil, testLogger())

	// Populate cache with order fills from multiple accounts.
	cache.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "BUY", Volume: 1.0, Price: 1.0850,
		ToState: "FILLED", Timestamp: time.Now(),
	})
	cache.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 2,
		Canonical: "GBPUSD", Side: "SELL", Volume: 0.5, Price: 1.2650,
		ToState: "FILLED", Timestamp: time.Now(),
	})
	cache.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-2", Ticket: 3,
		Canonical: "USDJPY", Side: "BUY", Volume: 2.0, Price: 150.00,
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
	if totalExposure <= 0 {
		t.Fatalf("totalExposure should be > 0, got %f", totalExposure)
	}
	if totalMargin <= 0 {
		t.Fatalf("totalMargin should be > 0, got %f", totalMargin)
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
	if acc1.Exposure <= 0 {
		t.Fatalf("acc-1 exposure should be > 0, got %f", acc1.Exposure)
	}

	acc2 := state.GetAccount("acc-2")
	if acc2 == nil {
		t.Fatal("acc-2 should exist")
	}

	t.Logf("DerivedQuantities: acc-1 exposure=%.4f margin=%.4f | acc-2 exposure=%.4f | total exposure=%.4f margin=%.4f",
		acc1.Exposure, acc1.MarginUsed, acc2.Exposure, totalExposure, totalMargin)
}
