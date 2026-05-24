package mdgateway

import (
	"testing"

	"github.com/shopspring/decimal"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

func TestBarFinality(t *testing.T) {
	agg := NewBarAggregator()

	fk := finalizedKey{"test-broker", "EURUSD", "1m"}

	// Load finalized set with one existing bar.
	agg.LoadFinalizedBars(map[finalizedKey][]int64{
		fk: {1000, 2000, 3000},
	})

	// Replay same close_ts → should be skipped.
	bar1 := &mdtick.Bar{
		Broker: "test-broker", Canonical: "EURUSD", Period: "1m",
		CloseTsUnixMs: 1000,
		Close:         requireDecimal(t, "1.08000"),
	}
	if agg.IngestExternalBar(bar1) {
		t.Error("bar with existing close_ts=1000 should be rejected")
	}

	// New close_ts → should be accepted.
	bar2 := &mdtick.Bar{
		Broker: "test-broker", Canonical: "EURUSD", Period: "1m",
		CloseTsUnixMs: 4000,
		Close:         requireDecimal(t, "1.08010"),
	}
	if !agg.IngestExternalBar(bar2) {
		t.Error("bar with new close_ts=4000 should be accepted")
	}

	// Historical gap bar (e.g. backfill for 3 days ago) → should be accepted.
	bar3 := &mdtick.Bar{
		Broker: "test-broker", Canonical: "EURUSD", Period: "1m",
		CloseTsUnixMs: 500, // older than 1000 — historical gap
		Close:         requireDecimal(t, "1.07990"),
	}
	if !agg.IngestExternalBar(bar3) {
		t.Error("historical gap bar with close_ts=500 should be accepted (exact-match dedup)")
	}

	// Repeat the same accepted bar → should now be rejected.
	if agg.IngestExternalBar(bar3) {
		t.Error("repeating accepted bar should be rejected on second attempt")
	}

	if BarSkippedFinalized() < 2 {
		t.Errorf("expected at least 2 skipped bars, got %d", BarSkippedFinalized())
	}

	t.Log("BarFinality: exact-match dedup works correctly for both replay and historical gap bars")
}

// requireDecimal is a test helper that panics on invalid decimal strings.
func requireDecimal(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("invalid decimal %q: %v", s, err)
	}
	return d
}
