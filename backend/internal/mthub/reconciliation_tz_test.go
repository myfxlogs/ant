package mthub

import (
	"testing"
	"time"

	"alphaforge/internal/clock"
)

// TestReconcileCutoff_IsUTC pins RECONCILE-TZ-WINDOW-1: the ant-side query
// cutoff must be encoded as UTC wall clock. orders.created_at /
// trade_records.close_time are `timestamp` columns storing UTC, and pgx
// encodes a `timestamp` parameter by its wall-clock components — a CST
// time.Local value shifts the cutoff +8h, shrinking the 24h window to ~16h.
// Mutation proof: dropping .UTC() in reconcileCutoff turns this RED.
func TestReconcileCutoff_IsUTC(t *testing.T) {
	if loc := reconcileCutoff().Location(); loc != time.UTC {
		t.Fatalf("reconcileCutoff location = %v, want time.UTC", loc)
	}
}

// TestReconcileCutoff_UTCValue pins the instant semantics: with a CST clock
// the cutoff must still be exactly now-24h expressed in UTC — not the CST
// wall clock minus 24h.
func TestReconcileCutoff_UTCValue(t *testing.T) {
	old := Clk
	defer func() { Clk = old }()
	cst := time.FixedZone("CST", 8*3600)
	Clk = clock.NewSimulatedClock(time.Date(2026, 9, 16, 10, 0, 0, 0, cst)) // 02:00 UTC

	got := reconcileCutoff()
	want := time.Date(2026, 9, 15, 2, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("reconcileCutoff = %v, want instant %v", got, want)
	}
	// The encoded wall clock must be 02:00 UTC, not 10:00 CST.
	if got.Hour() != 2 {
		t.Fatalf("reconcileCutoff wall-clock hour = %d, want 2 (UTC encoding)", got.Hour())
	}
}
