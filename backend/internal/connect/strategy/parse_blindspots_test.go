package strategy

import (
	"testing"

	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// TestParseBacktestResult_BlindSpots is the adversarial proof for ADR-0028 Part B:
// it verifies that parseBacktestResult extracts BlindSpots from the raw proto response,
// so they flow through the watch stream to the frontend diagnostic panel.
//
// Adversarial proof: if the BlindSpots extraction code in parseBacktestResult is deleted,
// this test will fail (bp.BlindSpots will be nil), proving the watch stream would be
// blind to diagnostics — the exact regression we're guarding against.
func TestParseBacktestResult_BlindSpots(t *testing.T) {
	resp := &antv1.ExecuteBacktestResponse{
		Success: true,
		BlindSpots: []*antv1.BlindSpot{
			{
				Id:          "zero_volume_trade",
				Description: "Trade with zero volume detected",
				Severity:    "致命",
				Category:    "invariant",
				Location:    "trade[5]",
			},
			{
				Id:          "no_stop_loss",
				Description: "Strategy does not set stop loss",
				Severity:    "警告",
				Category:    "defense_a",
				Location:    "",
			},
			{
				Id:          "overfitting_hint",
				Description: "High trade count relative to parameters",
				Severity:    "信息",
				Category:    "statistical",
				Location:    "",
			},
		},
	}
	raw, err := proto.Marshal(resp)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	bp := parseBacktestResult(raw)

	if len(bp.BlindSpots) != 3 {
		t.Fatalf("expected 3 blind spots, got %d — if 0, the extraction code in parseBacktestResult is missing", len(bp.BlindSpots))
	}

	// Verify fatal blind spot
	fatal := bp.BlindSpots[0]
	if fatal.GetId() != "zero_volume_trade" {
		t.Errorf("expected id=zero_volume_trade, got %s", fatal.GetId())
	}
	if fatal.GetSeverity() != "致命" {
		t.Errorf("expected severity=致命, got %s", fatal.GetSeverity())
	}
	if fatal.GetCategory() != "invariant" {
		t.Errorf("expected category=invariant, got %s", fatal.GetCategory())
	}
	if fatal.GetLocation() != "trade[5]" {
		t.Errorf("expected location=trade[5], got %s", fatal.GetLocation())
	}

	// Verify warning blind spot
	warning := bp.BlindSpots[1]
	if warning.GetSeverity() != "警告" {
		t.Errorf("expected severity=警告, got %s", warning.GetSeverity())
	}
	if warning.GetCategory() != "defense_a" {
		t.Errorf("expected category=defense_a, got %s", warning.GetCategory())
	}

	// Verify info blind spot
	info := bp.BlindSpots[2]
	if info.GetSeverity() != "信息" {
		t.Errorf("expected severity=信息, got %s", info.GetSeverity())
	}
	if info.GetCategory() != "statistical" {
		t.Errorf("expected category=statistical, got %s", info.GetCategory())
	}
}

// TestParseBacktestResult_NoBlindSpots verifies that a response without blind spots
// produces an empty (not nil) slice — the frontend checks length, not nil.
func TestParseBacktestResult_NoBlindSpots(t *testing.T) {
	resp := &antv1.ExecuteBacktestResponse{
		Success: true,
	}
	raw, err := proto.Marshal(resp)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	bp := parseBacktestResult(raw)
	if len(bp.BlindSpots) != 0 {
		t.Errorf("expected 0 blind spots for clean response, got %d", len(bp.BlindSpots))
	}
}

// TestParseBacktestResult_EmptyRaw verifies graceful handling of empty raw bytes.
func TestParseBacktestResult_EmptyRaw(t *testing.T) {
	bp := parseBacktestResult(nil)
	if len(bp.BlindSpots) != 0 {
		t.Errorf("expected 0 blind spots for nil raw, got %d", len(bp.BlindSpots))
	}
}
