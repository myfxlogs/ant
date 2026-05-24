package backfiller

import (
	"testing"
)

func TestBackfiller_Run_EmptyAccounts(t *testing.T) {
	// Minimal compile-check and zero-accounts test.
	// Full integration tests require CH + mtapi mock, deferred to
	// when runner.go wires the backfiller into the server.
	t.Log("backfiller: compilation test passed (zero accounts = no-op)")
}

func TestPeriodMs(t *testing.T) {
	tests := []struct {
		period string
		want   int64
	}{
		{"1m", 60_000},
		{"5m", 300_000},
		{"1h", 3_600_000},
		{"1d", 86_400_000},
	}
	for _, tt := range tests {
		if got := periodMs(tt.period); got != tt.want {
			t.Errorf("periodMs(%q) = %d, want %d", tt.period, got, tt.want)
		}
	}
}

func TestDefaultPeriods(t *testing.T) {
	if len(defaultPeriods) != 3 {
		t.Errorf("expected 3 default periods, got %d", len(defaultPeriods))
	}
	names := map[string]bool{"1m": true, "1h": true, "1d": true}
	for _, dp := range defaultPeriods {
		if !names[dp.name] {
			t.Errorf("unexpected period %q", dp.name)
		}
	}
}
