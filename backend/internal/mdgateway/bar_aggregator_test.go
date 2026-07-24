package mdgateway

import (
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/repository"
)

func TestBarFinality(t *testing.T) {
	t.Parallel()
	agg := NewBarAggregator()

	fk := repository.FinalizedKey{Broker: "test-broker", Canonical: "EURUSD", Period: "1m"}

	// Load finalized set with one existing bar.
	agg.LoadFinalizedBars(map[repository.FinalizedKey][]int64{
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

func TestRestoreOpenBars(t *testing.T) {
	t.Parallel()
	agg := NewBarAggregator()

	// Simulate a finalized 1m bar that closed at ts=60_000 (bucket 1).
	// The next bucket (bucket 2) spans [120_000, 180_000).
	// If now is within bucket 2 (e.g. 150_000), the open bar should be restored.
	bars := []repository.KlineBar{
		{
			Broker: "test-broker", Canonical: "EURUSD", Period: "1m",
			OpenTsUnixMs:  0,
			CloseTsUnixMs: 60_000,
			Open:          requireDecimal(t, "1.08000"),
			High:          requireDecimal(t, "1.08010"),
			Low:           requireDecimal(t, "1.07990"),
			Close:         requireDecimal(t, "1.08005"),
		},
	}

	// nowMs = 150_000 → bucket 2 → should restore
	restored := agg.RestoreOpenBars(bars, 150_000)
	if restored != 1 {
		t.Fatalf("expected 1 restored bar, got %d", restored)
	}

	// Verify the open bar was created with the finalized bar's close as initial OHLC.
	openBars := agg.GetOpenBars()
	var found bool
	for _, ob := range openBars {
		if ob.Broker == "test-broker" && ob.Canonical == "EURUSD" && ob.Period == "1m" {
			found = true
			if !ob.Open.Equal(requireDecimal(t, "1.08005")) {
				t.Errorf("expected open=1.08005, got %s", ob.Open)
			}
			if !ob.Close.Equal(requireDecimal(t, "1.08005")) {
				t.Errorf("expected close=1.08005, got %s", ob.Close)
			}
			if ob.OpenTsUnixMs != 120_000 {
				t.Errorf("expected startTs=120000, got %d", ob.OpenTsUnixMs)
			}
		}
	}
	if !found {
		t.Error("restored open bar not found in GetOpenBars")
	}

	// nowMs = 300_000 → bucket 5 → too far from finalized bucket 1 → should NOT restore
	agg2 := NewBarAggregator()
	restored2 := agg2.RestoreOpenBars(bars, 300_000)
	if restored2 != 0 {
		t.Errorf("expected 0 restored bars (bucket gap too large), got %d", restored2)
	}

	// Double restore should not duplicate
	restored3 := agg.RestoreOpenBars(bars, 150_000)
	if restored3 != 0 {
		t.Errorf("expected 0 restored on second call (already exists), got %d", restored3)
	}

	t.Log("RestoreOpenBars: correctly restores in-progress bar state on restart")
}
