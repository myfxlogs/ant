//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// TestGetKlines_MultiBroker_Chronological verifies that GetKlines with an empty
// broker returns a single, globally chronological time series even when
// multiple brokers write the same canonical symbol.
//
// Regression: backtest ID 1ccad72a failed with "bars are not chronologically
// ordered at index 12" because the broker="" branch put broker in the DISTINCT
// ON key and ORDER BY, sorting by broker name first. Broker "Exness (VG) Ltd"
// (12 bars at the end of the range) preceded "Exness Technologies Ltd" (bars
// from the start of the range), so the series jumped backward in time at the
// broker boundary.
//
// This test seeds two brokers whose alphabetical order is the REVERSE of their
// chronological order, plus a shared timestamp to verify cross-broker dedup.
// With the old query the result is non-chronological; with the fix it is
// chronological and deduped.
func TestGetKlines_MultiBroker_Chronological(t *testing.T) {
	pool := getTestPool(t)
	store := NewPgMarketDataStore(pool, zap.NewNop())
	ctx := context.Background()

	// Use a unique canonical so we never collide with real data and can clean
	// up unconditionally.
	const canonical = "TEST_MULTIBROKER_CHRONO"
	const period = "1h"
	const periodMs = int64(3600_000)
	// Real, 1h-aligned timestamps inside existing monthly partitions (2026-06,
	// 2026-07, 2026-08) so the partitioned INSERT succeeds.
	hour := func(month, day, h int) uint64 {
		return uint64(time.Date(2026, time.Month(month), day, h, 0, 0, 0, time.UTC).UnixMilli())
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM md_bars WHERE canonical = $1`, canonical); err != nil {
			t.Logf("cleanup delete failed: %v", err)
		}
	})

	// Broker "A_first" writes LATE bars (August); broker "B_second" writes
	// EARLY bars (June). Alphabetical order (A < B) is the reverse of
	// chronological order, so any broker-first sort fails.
	mk := func(broker string, ts uint64, closePx string, tickCount uint32) KlineBar {
		return KlineBar{
			Broker:        broker,
			SymbolRaw:     canonical,
			Canonical:     canonical,
			Period:        period,
			OpenTsUnixMs:  ts,
			CloseTsUnixMs: ts + uint64(periodMs),
			Open:          decimal.NewFromInt(100),
			High:          decimal.NewFromInt(110),
			Low:           decimal.NewFromInt(90),
			Close:         decimal.RequireFromString(closePx),
			Volume:        1,
			TickCount:     tickCount,
		}
	}

	bars := []KlineBar{
		// "A_first" — alphabetically first, chronologically LATE (August).
		mk("A_first", hour(8, 1, 0), "800.0", 10),
		mk("A_first", hour(8, 1, 1), "801.0", 10),
		mk("A_first", hour(8, 1, 2), "802.0", 10),
		// "B_second" — alphabetically second, chronologically EARLY (June).
		mk("B_second", hour(6, 1, 0), "600.0", 10),
		mk("B_second", hour(6, 1, 1), "601.0", 10),
		mk("B_second", hour(6, 1, 2), "602.0", 10),
		// Shared timestamp (July 1 00:00): both brokers write it; A has higher
		// tick_count, so the cross-broker dedup must pick A_first.
		mk("A_first", hour(7, 1, 0), "700.0", 50),
		mk("B_second", hour(7, 1, 0), "700.5", 20),
	}
	if err := store.InsertBars(ctx, bars); err != nil {
		t.Fatalf("InsertBars: %v", err)
	}

	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	got, err := store.GetKlines(ctx, canonical, "" /*broker*/, period, &from, &to, 1000)
	if err != nil {
		t.Fatalf("GetKlines: %v", err)
	}

	// Expect 7 unique timestamps: June 1 00/01/02, July 1 00, Aug 1 00/01/02.
	wantTs := []uint64{
		hour(6, 1, 0), hour(6, 1, 1), hour(6, 1, 2),
		hour(7, 1, 0),
		hour(8, 1, 0), hour(8, 1, 1), hour(8, 1, 2),
	}
	if len(got) != len(wantTs) {
		t.Fatalf("expected %d deduped bars, got %d: %v", len(wantTs), len(got), tsList(got))
	}

	// 1. Chronological: each timestamp must be strictly greater than the previous.
	for i := 1; i < len(got); i++ {
		if got[i].OpenTsUnixMs <= got[i-1].OpenTsUnixMs {
			t.Fatalf("bars not chronologically ordered at index %d: %d <= %d (full: %v)",
				i, got[i].OpenTsUnixMs, got[i-1].OpenTsUnixMs, tsList(got))
		}
	}

	// 2. Deduped: exactly the expected timestamps in order.
	for i, w := range wantTs {
		if got[i].OpenTsUnixMs != w {
			t.Fatalf("index %d: timestamp %d, want %d (full: %v)",
				i, got[i].OpenTsUnixMs, w, tsList(got))
		}
	}

	// 3. Cross-broker dedup picks highest tick_count: July 1 00 → A_first (50).
	var julyBar *KlineBar
	for i := range got {
		if got[i].OpenTsUnixMs == hour(7, 1, 0) {
			julyBar = &got[i]
			break
		}
	}
	if julyBar == nil {
		t.Fatalf("July 1 00:00 bar missing")
	}
	if julyBar.TickCount != 50 {
		t.Errorf("July 1 00:00 tick_count = %d, want 50 (highest across brokers)", julyBar.TickCount)
	}
	if julyBar.Broker != "A_first" {
		t.Errorf("July 1 00:00 broker = %q, want %q (highest tick_count winner)", julyBar.Broker, "A_first")
	}
}

func tsList(bars []KlineBar) []uint64 {
	out := make([]uint64, len(bars))
	for i, b := range bars {
		out[i] = b.OpenTsUnixMs
	}
	return out
}
