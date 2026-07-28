// Package strategy — tests for shared gate proto builder functions.
package strategy

import (
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/ai"
	"alphaforge/internal/marketplace"
)

func TestGateResultToProto(t *testing.T) {
	t.Parallel()
	g := ai.GateStatus{
		Gate:     ai.GateCompliance,
		Passed:   true,
		Skipped:  false,
		Reason:   "ok",
		Score:    0.95,
		Duration: 42,
	}
	p := GateResultToProto(g)
	if p.Gate != "compliance" {
		t.Fatalf("gate: want compliance, got %s", p.Gate)
	}
	if !p.Passed {
		t.Fatal("should pass")
	}
	if p.Skipped {
		t.Fatal("should not be skipped")
	}
	if p.Reason != "ok" {
		t.Fatalf("reason: want ok, got %s", p.Reason)
	}
	if p.Score != 0.95 {
		t.Fatalf("score: want 0.95, got %f", p.Score)
	}
	if p.DurationMs != 42 {
		t.Fatalf("duration: want 42, got %d", p.DurationMs)
	}
}

func TestBuildGateSummaryProto_NilResult(t *testing.T) {
	t.Parallel()
	if p := BuildGateSummaryProto(nil); p != nil {
		t.Fatal("nil result should return nil")
	}
}

func TestBuildGateSummaryProto_ValidResult(t *testing.T) {
	t.Parallel()
	result := &ai.PipelineResult{
		Passed:        true,
		FirstFail:     "",
		Summary:       "all 7 gates passed",
		TotalDuration: 150,
	}
	p := BuildGateSummaryProto(result)
	if p == nil || p.Completed == nil {
		t.Fatal("should return non-nil summary")
	}
	if !p.Completed.Passed {
		t.Fatal("should pass")
	}
	if p.Completed.Summary != "all 7 gates passed" {
		t.Fatalf("summary: want 'all 7 gates passed', got %s", p.Completed.Summary)
	}
	if p.Completed.TotalDurationMs != 150 {
		t.Fatalf("duration: want 150, got %d", p.Completed.TotalDurationMs)
	}
}

func TestBuildGateListProto_NilResult(t *testing.T) {
	t.Parallel()
	if p := BuildGateListProto(nil); p != nil {
		t.Fatal("nil result should return nil")
	}
}

func TestBuildGateListProto_EmptyGates(t *testing.T) {
	t.Parallel()
	result := &ai.PipelineResult{Gates: nil}
	if p := BuildGateListProto(result); p != nil {
		t.Fatal("empty gates should return nil")
	}
}

func TestBuildGateListProto_ValidGates(t *testing.T) {
	t.Parallel()
	result := &ai.PipelineResult{
		Gates: []ai.GateStatus{
			{Gate: ai.GateCompliance, Passed: true, Duration: 10},
			{Gate: ai.GateLookAhead, Passed: false, Reason: "future ref", Duration: 5},
		},
	}
	list := BuildGateListProto(result)
	if list == nil {
		t.Fatal("should return non-nil list")
	}
	if len(list.Gates) != 2 {
		t.Fatalf("want 2 gates, got %d", len(list.Gates))
	}
	if list.Gates[0].Gate != "compliance" {
		t.Fatalf("first gate: want compliance, got %s", list.Gates[0].Gate)
	}
	if list.Gates[1].Gate != "lookahead" {
		t.Fatalf("second gate: want lookahead, got %s", list.Gates[1].Gate)
	}
	if list.Gates[1].Passed {
		t.Fatal("lookahead should not pass")
	}
}

func TestViolationsToPreview_Empty(t *testing.T) {
	t.Parallel()
	p := ViolationsToPreview(nil)
	if !p.Publishable {
		t.Fatal("no violations → should be publishable")
	}
	if len(p.Violations) != 0 {
		t.Fatalf("want 0 violations, got %d", len(p.Violations))
	}
}

func TestViolationsToPreview_WithViolations(t *testing.T) {
	t.Parallel()
	violations := []marketplace.QualityViolation{
		{Metric: "min_sharpe", Actual: "0.5", Threshold: "1.0"},
		{Metric: "min_trades", Actual: "10", Threshold: "30"},
	}
	p := ViolationsToPreview(violations)
	if p.Publishable {
		t.Fatal("with violations → should not be publishable")
	}
	if len(p.Violations) != 2 {
		t.Fatalf("want 2 violations, got %d", len(p.Violations))
	}
}

// Verify antv1 import is used (compile-time check).
var _ *antv1.GateResult = (*antv1.GateResult)(nil)
