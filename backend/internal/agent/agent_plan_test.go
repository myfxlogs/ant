package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	connectai "alphaforge/internal/connect/ai"
)

// Adversarial proof: update_plan persists steps into generateState.
// Remove the t.result.PlanSteps = steps line and this test fails red.
func TestUpdatePlanTool_PersistsSteps(t *testing.T) {
	state := &generateState{}
	tool := &updatePlanTool{result: state}

	planJSON := `[{"step":"EMA entry","status":"done"},{"step":"ATR stop","status":"doing"},{"step":"position sizing","status":"pending"}]`
	out := tool.Run(context.Background(), connectai.ToolInput{
		RawArgs: map[string]any{"plan": planJSON},
	})

	if !out.Success {
		t.Fatalf("expected success, got error: %s", out.Error)
	}

	if len(state.PlanSteps) != 3 {
		t.Fatalf("expected 3 plan steps persisted, got %d", len(state.PlanSteps))
	}

	if state.PlanSteps[0].Step != "EMA entry" || state.PlanSteps[0].Status != "done" {
		t.Errorf("step 0 mismatch: %+v", state.PlanSteps[0])
	}
	if state.PlanSteps[1].Status != "doing" {
		t.Errorf("step 1 should be doing, got %s", state.PlanSteps[1].Status)
	}
}

// Adversarial proof: HasActivePlan returns true when steps are incomplete.
// Remove the HasActivePlan method and the agent loop won't inject step guidance.
func TestGenerateState_HasActivePlan(t *testing.T) {
	// All done → no active plan.
	state := &generateState{
		PlanSteps: []planStep{
			{Step: "a", Status: "done"},
			{Step: "b", Status: "done"},
		},
	}
	if state.HasActivePlan() {
		t.Fatal("all steps done should not have active plan")
	}

	// One pending → active plan.
	state.PlanSteps[1].Status = "pending"
	if !state.HasActivePlan() {
		t.Fatal("pending step should mean active plan")
	}

	// One doing → active plan.
	state.PlanSteps[1].Status = "doing"
	if !state.HasActivePlan() {
		t.Fatal("doing step should mean active plan")
	}
}

// Adversarial proof: CurrentStep returns the "doing" step first, then first "pending".
// Remove this logic and the agent loop can't guide the LLM to the right step.
func TestGenerateState_CurrentStep(t *testing.T) {
	state := &generateState{
		PlanSteps: []planStep{
			{Step: "entry", Status: "done"},
			{Step: "stop-loss", Status: "doing"},
			{Step: "sizing", Status: "pending"},
		},
	}

	// "doing" step should be returned first.
	if got := state.CurrentStep(); got != "stop-loss" {
		t.Fatalf("expected 'stop-loss' (doing), got %q", got)
	}

	// If no "doing", return first "pending".
	state.PlanSteps[1].Status = "done"
	if got := state.CurrentStep(); got != "sizing" {
		t.Fatalf("expected 'sizing' (first pending), got %q", got)
	}

	// All done → empty.
	state.PlanSteps[2].Status = "done"
	if got := state.CurrentStep(); got != "" {
		t.Fatalf("expected empty when all done, got %q", got)
	}
}

// Adversarial proof: CompletedSteps and TotalSteps return correct counts.
func TestGenerateState_StepCounts(t *testing.T) {
	state := &generateState{
		PlanSteps: []planStep{
			{Step: "a", Status: "done"},
			{Step: "b", Status: "doing"},
			{Step: "c", Status: "pending"},
			{Step: "d", Status: "done"},
		},
	}

	if state.CompletedSteps() != 2 {
		t.Errorf("expected 2 completed, got %d", state.CompletedSteps())
	}
	if state.TotalSteps() != 4 {
		t.Errorf("expected 4 total, got %d", state.TotalSteps())
	}
}

// Adversarial proof: update_plan with invalid JSON returns error.
func TestUpdatePlanTool_InvalidJSON(t *testing.T) {
	tool := &updatePlanTool{result: &generateState{}}
	out := tool.Run(context.Background(), connectai.ToolInput{
		RawArgs: map[string]any{"plan": "not-valid-json"},
	})

	if out.Success {
		t.Fatal("expected failure for invalid JSON")
	}
	if !strings.Contains(out.Error, "invalid plan JSON") {
		t.Errorf("expected 'invalid plan JSON' error, got %s", out.Error)
	}
}

// Adversarial proof: update_plan with empty plan returns error.
func TestUpdatePlanTool_EmptyPlan(t *testing.T) {
	tool := &updatePlanTool{result: &generateState{}}
	out := tool.Run(context.Background(), connectai.ToolInput{
		RawArgs: map[string]any{},
	})

	if out.Success {
		t.Fatal("expected failure for empty plan")
	}
}

// Adversarial proof: update_plan output contains step guidance for the LLM.
// Remove the summary building logic and the LLM won't know what to do next.
func TestUpdatePlanTool_ContainsStepGuidance(t *testing.T) {
	state := &generateState{}
	tool := &updatePlanTool{result: state}

	planJSON := `[{"step":"entry signals","status":"done"},{"step":"stop-loss","status":"doing"},{"step":"sizing","status":"pending"}]`
	out := tool.Run(context.Background(), connectai.ToolInput{
		RawArgs: map[string]any{"plan": planJSON},
	})

	summary, ok := out.Output.(string)
	if !ok {
		t.Fatal("expected string output")
	}

	// Must mention current step for the LLM.
	if !strings.Contains(summary, "Current step: stop-loss") {
		t.Errorf("output should mention current step, got: %s", summary)
	}

	// Must mention progress count.
	if !strings.Contains(summary, "1/3") {
		t.Errorf("output should mention 1/3 progress, got: %s", summary)
	}
}

// Adversarial proof: all steps done → output says "All steps complete".
func TestUpdatePlanTool_AllDone(t *testing.T) {
	state := &generateState{}
	tool := &updatePlanTool{result: state}

	planJSON := `[{"step":"a","status":"done"},{"step":"b","status":"done"}]`
	out := tool.Run(context.Background(), connectai.ToolInput{
		RawArgs: map[string]any{"plan": planJSON},
	})

	summary, ok := out.Output.(string)
	if !ok {
		t.Fatal("expected string output")
	}

	if !strings.Contains(summary, "All steps complete") {
		t.Errorf("output should say all complete, got: %s", summary)
	}

	if state.HasActivePlan() {
		t.Fatal("all done should not have active plan")
	}
}

// Adversarial proof: plan steps survive JSON round-trip.
func TestPlanStep_JSONRoundTrip(t *testing.T) {
	steps := []planStep{
		{Step: "入场信号", Status: "done"},
		{Step: "止损逻辑", Status: "doing"},
	}

	data, err := json.Marshal(steps)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded []planStep
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(decoded) != 2 || decoded[0].Step != "入场信号" || decoded[1].Status != "doing" {
		t.Errorf("round-trip mismatch: %+v", decoded)
	}
}
