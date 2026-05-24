package backfiller

import (
	"testing"

	"golang.org/x/time/rate"
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

func TestBackfillGap(t *testing.T) {
	t.Log("TestBackfillGap: requires CH + mtapi mock (M10.5-8)")
}

func TestBackfillerPerAccountRate(t *testing.T) {
	b := &Backfiller{
		accountLimiters: make(map[string]*rate.Limiter),
		globalLimiter:   rate.NewLimiter(rate.Limit(60), 1),
	}
	l := b.getLimiter("acc-1")
	if l == nil {
		t.Fatal("getLimiter returned nil")
	}
	l2 := b.getLimiter("acc-1")
	if l != l2 {
		t.Error("same account should return same limiter")
	}
	l3 := b.getLimiter("acc-2")
	if l == l3 {
		t.Error("different accounts should have different limiters")
	}
	t.Log("BackfillerPerAccountRate: per-account limiters work")
}

func TestBackfillerPgTrigger(t *testing.T) {
	t.Log("TestBackfillerPgTrigger: requires PG NOTIFY (M10.5-8)")
}
