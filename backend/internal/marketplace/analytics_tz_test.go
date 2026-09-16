package marketplace

import (
	"testing"
	"time"
)

// TestAnalyticsSince_IsUTC pins TZ-SWEEP-AFFECTED-1: the analytics window
// bound must be encoded as UTC wall clock. The compared columns
// (wallet_transactions/user_subscriptions/marketplace_strategies.created_at)
// are `timestamp` columns storing UTC, and pgx encodes a `timestamp`
// parameter by wall-clock components — a CST time.Local value shifts the
// bound +8h, shrinking every period window by 8h.
// Mutation proof: dropping .UTC() in analyticsSince turns this RED.
func TestAnalyticsSince_IsUTC(t *testing.T) {
	if loc := analyticsSince(0).Location(); loc != time.UTC {
		t.Fatalf("analyticsSince location = %v, want time.UTC", loc)
	}
}

// TestAnalyticsSince_UTCWallClock pins the encoded wall clock: with
// time.Local forced to CST the bound must still carry the UTC wall clock
// (e.g. 02:00), not the CST wall clock (10:00). Deterministic regardless of
// the host TZ.
func TestAnalyticsSince_UTCWallClock(t *testing.T) {
	old := time.Local
	defer func() { time.Local = old }()
	time.Local = time.FixedZone("CST", 8*3600)

	got := analyticsSince(0)
	// Re-interpret got's wall-clock components as UTC (what pgx will encode
	// for a `timestamp` param) and compare against the true UTC wall clock.
	encoded := time.Date(got.Year(), got.Month(), got.Day(),
		got.Hour(), got.Minute(), got.Second(), 0, time.UTC)
	want := time.Now().UTC().Truncate(time.Second)
	if diff := want.Sub(encoded); diff < -2*time.Second || diff > 2*time.Second {
		t.Fatalf("analyticsSince encoded wall clock = %v, want ~%v (diff %v; CST encoding would be +8h)",
			encoded, want, diff)
	}
}
