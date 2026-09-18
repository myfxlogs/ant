package strategy

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/internal/mthub"
)

func cacheSnapshot() *mthub.PositionSnapshot {
	return &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(10000),
		Margin: decimal.NewFromInt(10), FreeMargin: decimal.NewFromInt(9990), Leverage: 100,
		FinancialsAuthoritative: true, FinancialsSource: "account_summary",
		PositionsAuthoritative: true,
		CapturedAt:             time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		PositionsCapturedAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		PositionsSource:        "order_stream",
	}
}

func TestPositionCacheRejectsNonAuthoritativeSnapshot(t *testing.T) {
	cache := NewPositionCache(nil)
	snap := cacheSnapshot()
	snap.FinancialsAuthoritative = false
	cache.PutSnapshot(snap, snap.CapturedAt)
	if _, ok := cache.GetFreshSnapshot("acct-1", snap.CapturedAt.Add(time.Second)); ok {
		t.Fatal("non-authoritative snapshot must not become usable account state")
	}
}

func TestPositionCacheRejectsStaleSnapshot(t *testing.T) {
	cache := NewPositionCache(nil)
	snap := cacheSnapshot()
	cache.PutSnapshot(snap, snap.CapturedAt)
	if _, ok := cache.GetFreshSnapshot("acct-1", snap.CapturedAt.Add(AccountSnapshotMaxAge+time.Second)); ok {
		t.Fatal("stale snapshot must not become usable account state")
	}
}

func TestPositionCacheOrderUpdateMergesPositionsWithoutOverwritingFinancials(t *testing.T) {
	cache := NewPositionCache(nil)
	snap := cacheSnapshot()
	cache.PutSnapshot(snap, snap.CapturedAt)
	order := &mthub.PositionSnapshot{
		AccountID:              "acct-1",
		Positions:              []mthub.PositionSnapshotItem{{Ticket: 7}},
		PositionsAuthoritative: true,
		FinancialsSource:       "order_stream",
		PositionsCapturedAt:    snap.CapturedAt.Add(time.Second),
		PositionsSource:        "order_stream",
	}
	cache.PutSnapshot(order, snap.CapturedAt.Add(time.Second))
	got, ok := cache.GetFreshSnapshot("acct-1", snap.CapturedAt.Add(2*time.Second))
	if !ok {
		t.Fatal("merged snapshot should remain fresh")
	}
	if !got.Balance.Equal(snap.Balance) || !got.FreeMargin.Equal(snap.FreeMargin) || len(got.Positions) != 1 || got.Positions[0].Ticket != 7 {
		t.Fatalf("order update changed authoritative financials or positions were not merged: %+v", got)
	}
}

func TestPositionCacheFinancialRefreshPreservesPositions(t *testing.T) {
	cache := NewPositionCache(nil)
	snap := cacheSnapshot()
	snap.Positions = []mthub.PositionSnapshotItem{{Ticket: 7}}
	cache.PutSnapshot(snap, snap.CapturedAt)
	refresh := cacheSnapshot()
	refresh.CapturedAt = snap.CapturedAt.Add(30 * time.Second)
	refresh.Positions = nil
	refresh.PositionsAuthoritative = false
	cache.PutSnapshot(refresh, refresh.CapturedAt)
	got, ok := cache.GetFreshSnapshot("acct-1", refresh.CapturedAt.Add(time.Second))
	if !ok || len(got.Positions) != 1 || got.Positions[0].Ticket != 7 {
		t.Fatalf("financial-only refresh cleared authoritative positions: %+v", got)
	}
}

// SNAPSHOT-SLICE-ALIAS-1: the cache's stored state must never share mutable
// backing arrays with caller-owned snapshots, and returned snapshots must be
// cache-private.

// TestPositionCache_PutIsolation — mutating the caller's snapshot AFTER
// PutSnapshot must not corrupt the stored cache state.
//
// Adversarial: delete put's else-branch copy → the stored merged carries the
// incoming snap's slices → the mutation leaks into GetSnapshot → RED.
func TestPositionCache_PutIsolation(t *testing.T) {
	cache := NewPositionCache(nil)
	snap := cacheSnapshot()
	snap.Positions = []mthub.PositionSnapshotItem{{Ticket: 42}}
	cache.PutSnapshot(snap, snap.CapturedAt)

	snap.Positions[0].Ticket = 999 // caller owns this snapshot; cache must be immune

	got := cache.GetSnapshot("acct-1")
	if got == nil || len(got.Positions) != 1 || got.Positions[0].Ticket != 42 {
		t.Fatalf("stored state corrupted by caller mutation: %+v, want Ticket 42", got)
	}
}

// TestPositionCache_GetCopyIsolation — mutating a returned snapshot must not
// corrupt the cache (Get* hand out private copies).
//
// Adversarial: restore GetSnapshot to return the stored pointer → both reads
// alias one object → the mutation leaks → RED.
func TestPositionCache_GetCopyIsolation(t *testing.T) {
	cache := NewPositionCache(nil)
	snap := cacheSnapshot()
	snap.Positions = []mthub.PositionSnapshotItem{{Ticket: 42}}
	cache.PutSnapshot(snap, snap.CapturedAt)

	first := cache.GetSnapshot("acct-1")
	first.Positions[0].Ticket = 999

	second := cache.GetSnapshot("acct-1")
	if second == nil || len(second.Positions) != 1 || second.Positions[0].Ticket != 42 {
		t.Fatalf("cache corrupted via returned-snapshot mutation: %+v, want Ticket 42", second)
	}
}
