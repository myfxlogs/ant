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
		CapturedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
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
		AccountID:        "acct-1",
		Positions:        []mthub.PositionSnapshotItem{{Ticket: 7}},
		FinancialsSource: "order_stream",
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
