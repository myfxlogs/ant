package strategy

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/internal/mthub"
)

// testAccountSnapshot represents a broker AccountSummary response with non-derived values.
func testAccountSnapshot() *mthub.PositionSnapshot {
	return &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(12000),
		Margin: decimal.NewFromInt(900), FreeMargin: decimal.NewFromInt(11100),
		Profit: decimal.NewFromInt(321), Leverage: 500, FinancialsAuthoritative: true,
		FinancialsSource: "account_summary", CapturedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Positions: []mthub.PositionSnapshotItem{{Ticket: 1}},
	}
}

func testProviderWithSnapshot() *MTAccountStateProvider {
	pc := NewPositionCache(nil)
	snap := testAccountSnapshot()
	pc.PutSnapshot(snap, snap.CapturedAt)
	p := NewMTAccountStateProvider(nil, nil)
	p.SetPositionCache(pc)
	p.now = func() time.Time { return snap.CapturedAt.Add(time.Second) }
	return p
}

func TestProviderMissingSnapshotFailsClosed(t *testing.T) {
	p := NewMTAccountStateProvider(nil, nil)
	p.SetPositionCache(NewPositionCache(nil))
	state, err := p.GetAccountState(context.Background(), "missing")
	if err != nil || state != nil {
		t.Fatalf("missing snapshot must fail closed: state=%v err=%v", state, err)
	}
}

func TestProviderUsesAuthoritativeFinancials(t *testing.T) {
	p := testProviderWithSnapshot()
	state, err := p.GetAccountState(context.Background(), "acct-1")
	if err != nil || state == nil {
		t.Fatalf("expected state, got state=%v err=%v", state, err)
	}
	if !state.Equity.Equal(decimal.NewFromInt(12000)) || !state.FreeMargin.Equal(decimal.NewFromInt(11100)) {
		t.Fatalf("provider recomputed financials: equity=%s free_margin=%s", state.Equity, state.FreeMargin)
	}
	if !state.UsedMargin.Equal(decimal.NewFromInt(900)) || state.SymbolLeverage != 500 {
		t.Fatalf("provider changed broker margin data: margin=%s leverage=%d", state.UsedMargin, state.SymbolLeverage)
	}
	if !state.DailyPnL.Equal(decimal.NewFromInt(321)) {
		t.Fatalf("profit=%s, want 321", state.DailyPnL)
	}
}

func TestProviderStaleSnapshotFailsClosed(t *testing.T) {
	pc := NewPositionCache(nil)
	snap := testAccountSnapshot()
	pc.PutSnapshot(snap, snap.CapturedAt)
	p := NewMTAccountStateProvider(nil, nil)
	p.SetPositionCache(pc)
	p.now = func() time.Time { return snap.CapturedAt.Add(AccountSnapshotMaxAge + time.Second) }
	state, err := p.GetAccountState(context.Background(), "acct-1")
	if err != nil || state != nil {
		t.Fatalf("stale snapshot must fail closed: state=%v err=%v", state, err)
	}
}

func TestProviderNonAuthoritativeSnapshotIgnored(t *testing.T) {
	pc := NewPositionCache(nil)
	snap := testAccountSnapshot()
	snap.FinancialsAuthoritative = false
	pc.PutSnapshot(snap, snap.CapturedAt)
	p := NewMTAccountStateProvider(nil, nil)
	p.SetPositionCache(pc)
	p.now = func() time.Time { return snap.CapturedAt.Add(time.Second) }
	state, err := p.GetAccountState(context.Background(), "acct-1")
	if err != nil || state != nil {
		t.Fatalf("non-authoritative snapshot must fail closed: state=%v err=%v", state, err)
	}
}

func TestProviderPeakEquityTracking(t *testing.T) {
	p := testProviderWithSnapshot()
	if _, err := p.GetAccountState(context.Background(), "acct-1"); err != nil {
		t.Fatal(err)
	}
	if !p.GetPeakEquity("acct-1").Equal(decimal.NewFromInt(12000)) {
		t.Fatalf("peak equity=%s, want 12000", p.GetPeakEquity("acct-1"))
	}
	p.ResetPeakEquity("acct-1")
	if !p.GetPeakEquity("acct-1").IsZero() {
		t.Fatal("peak equity was not reset")
	}
}
