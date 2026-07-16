package mthub

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestDerivedState_UpdateAndGet(t *testing.T) {
	t.Parallel()
	ds := NewDerivedState()
	accounts := map[string]*AccountDerivedState{
		"acc-1": {AccountID: "acc-1", GrossPnL: decimal.NewFromInt(100), NetPnL: decimal.NewFromInt(85), Exposure: decimal.NewFromInt(10000), MarginUsed: decimal.NewFromInt(100)},
		"acc-2": {AccountID: "acc-2", GrossPnL: decimal.NewFromInt(-50), NetPnL: decimal.NewFromInt(-55), Exposure: decimal.NewFromInt(5000), MarginUsed: decimal.NewFromInt(50)},
	}
	ds.Update(accounts, decimal.NewFromInt(15000), decimal.NewFromInt(150), decimal.NewFromInt(50), decimal.NewFromInt(30), 500)

	result, totalExp, totalMargin, gross, net, var95, lastUpdated := ds.Get()
	if len(result) != 2 {
		t.Fatalf("want 2 accounts, got %d", len(result))
	}
	if !totalExp.Equal(decimal.NewFromInt(15000)) {
		t.Fatalf("want 15000 exposure, got %s", totalExp.String())
	}
	if !totalMargin.Equal(decimal.NewFromInt(150)) {
		t.Fatalf("want 150 margin, got %s", totalMargin.String())
	}
	if !gross.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("want 50 gross, got %s", gross.String())
	}
	if !net.Equal(decimal.NewFromInt(30)) {
		t.Fatalf("want 30 net, got %s", net.String())
	}
	if var95 != 500 {
		t.Fatalf("want 500 var95, got %f", var95)
	}
	if lastUpdated.IsZero() {
		t.Fatal("lastUpdated should not be zero")
	}
}

func TestDerivedState_GetAccount(t *testing.T) {
	t.Parallel()
	ds := NewDerivedState()
	accounts := map[string]*AccountDerivedState{
		"acc-1": {AccountID: "acc-1", GrossPnL: decimal.NewFromInt(100), NetPnL: decimal.NewFromInt(85)},
	}
	ds.Update(accounts, decimal.NewFromInt(10000), decimal.NewFromInt(100), decimal.NewFromInt(100), decimal.NewFromInt(85), 500)

	acc := ds.GetAccount("acc-1")
	if acc == nil {
		t.Fatal("acc-1 should exist")
	}
	if !acc.GrossPnL.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("want 100 GrossPnL, got %s", acc.GrossPnL.String())
	}

	missing := ds.GetAccount("acc-nonexistent")
	if missing != nil {
		t.Fatal("nonexistent account should be nil")
	}
}

func TestDerivedComputer_StartStop(t *testing.T) {
	t.Parallel()
	cache := NewStateCache(nil, testLogger())
	computer := NewDerivedComputer(cache, 100*time.Millisecond)
	computer.Start()

	// Let it run a few cycles.
	time.Sleep(350 * time.Millisecond)
	computer.Stop()

	state := computer.State()
	_, _, _, _, _, _, lastUpdated := state.Get()
	if lastUpdated.IsZero() {
		t.Fatal("computer should have updated at least once")
	}
}

func TestDerivedComputer_DefaultInterval(t *testing.T) {
	t.Parallel()
	cache := NewStateCache(nil, testLogger())
	computer := NewDerivedComputer(cache, 0) // 0 → defaults to 5s
	if computer.interval != 5*time.Second {
		t.Fatalf("want default 5s, got %v", computer.interval)
	}
}
