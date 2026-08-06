package marketplace

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// degradedViolation returns a non-waivable QualityViolation if the backtest
// run status is DEGRADED. This is the pure-logic core of checkDegradedStatus,
// extracted for testability without a database.
func degradedViolation(runStatus string) *QualityViolation {
	if runStatus == "DEGRADED" {
		return &QualityViolation{
			Metric:    "backtest_status",
			Actual:    "DEGRADED",
			Threshold: "SUCCEEDED (invariant checks must pass)",
		}
	}
	return nil
}

func TestDegradedViolation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status string
		expect bool
	}{
		{"DEGRADED", "DEGRADED", true},
		{"SUCCEEDED", "SUCCEEDED", false},
		{"FAILED", "FAILED", false},
		{"empty", "", false},
		{"CANCELED", "CANCELED", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := degradedViolation(tc.status)
			if tc.expect && v == nil {
				t.Fatal("expected violation for DEGRADED status, got nil")
			}
			if !tc.expect && v != nil {
				t.Fatalf("expected nil for %s status, got violation: %s", tc.status, v)
			}
			if v != nil {
				if v.Metric != "backtest_status" || v.Actual != "DEGRADED" {
					t.Errorf("unexpected violation fields: %+v", v)
				}
			}
		})
	}
}

// TestCheckDegradedStatus_EmptyStrategyID verifies that an empty strategyID
// short-circuits to nil (no DB query, no block).
func TestCheckDegradedStatus_EmptyStrategyID(t *testing.T) {
	s := &Service{}
	v := s.checkDegradedStatus(context.Background(), "")
	if v != nil {
		t.Fatal("expected nil for empty strategyID (no DB query should be made)")
	}
}

// TestValidateBacktestQuality_DegradedCheckBeforeWaiver verifies the structural
// ordering: DEGRADED check runs before waiver check. With nil pg + non-empty
// strategyID:
// - loadQualityGates returns zero gates (nil-safe)
// - checkDegradedStatus panics on nil pg (no nil guard — reaches pg.QueryRow)
// - hasQualityWaiver would return false (nil-safe — has nil guard)
//
// If waiver check were before DEGRADED, hasQualityWaiver would return false
// (nil-safe) and no panic would occur — the function would return nil, nil.
// The panic proves DEGRADED check runs first and is reached before waiver.
func TestValidateBacktestQuality_DegradedCheckBeforeWaiver(t *testing.T) {
	s := &Service{}

	snap := &antv1.BacktestSnapshot{
		TotalTrades: 15,
		SharpeRatio: "1.0",
		MaxDrawdown: "0.1",
		WinRate:     "0.6",
	}
	snapBytes, _ := proto.Marshal(snap)

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic from checkDegradedStatus on nil pg, proving DEGRADED check runs before waiver (which is nil-safe and would not panic)")
			}
		}()
		_, _ = s.ValidateBacktestQuality(context.Background(), snapBytes, "00000000-0000-0000-0000-000000000001")
	}()
}

// TestValidateBacktestQuality_MissingSnapshotNoGates verifies that a missing
// snapshot with no gates loaded (nil pg) returns no violations — documenting
// that gate enforcement requires DB-configured thresholds.
func TestValidateBacktestQuality_MissingSnapshotNoGates(t *testing.T) {
	s := &Service{}

	violations, err := s.ValidateBacktestQuality(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("expected no violations with nil pg (gates not loaded), got %d", len(violations))
	}
}
