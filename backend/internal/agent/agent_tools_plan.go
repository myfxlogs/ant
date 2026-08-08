package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	connectai "alphaforge/internal/connect/ai"
	systemai "alphaforge/internal/service/systemai"
)

// updatePlanTool is the plan-driven state machine tool.
// The LLM calls it to create/update a multi-step plan. The plan is persisted
// into generateState.PlanSteps so the agent loop can drive step-by-step execution.
type updatePlanTool struct {
	result *generateState
}

func (t *updatePlanTool) Name() string { return "update_plan" }

func (t *updatePlanTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name:        "update_plan",
			Description: "Plan-driven step tracker for complex strategies. Call this FIRST with a JSON plan [{step, status}] before writing code for multi-step strategies. Update step statuses as you progress. Simple strategies can skip this and call write_strategy directly.",
			Parameters: map[string]any{
				schemaKeyType: schemaTypeObject,
				"required":    []string{"plan"},
				schemaKeyProperties: map[string]any{
					"plan": map[string]any{
						schemaKeyType: schemaTypeString,
						"description": "JSON array string, each item {step, status}. status: pending|doing|done. Example: [{\"step\":\"EMA entry signals\",\"status\":\"done\"},{\"step\":\"ATR stop-loss\",\"status\":\"doing\"},{\"step\":\"position sizing\",\"status\":\"pending\"}]",
					},
				},
			},
		},
	}
}

func (t *updatePlanTool) Run(_ context.Context, in connectai.ToolInput) connectai.ToolOutput {
	planJSON, _ := in.RawArgs["plan"].(string)
	if planJSON == "" {
		return connectai.ToolOutput{Success: false, Error: "plan is required (JSON array of {step, status})"}
	}

	var steps []planStep
	if err := json.Unmarshal([]byte(planJSON), &steps); err != nil {
		return connectai.ToolOutput{Success: false, Error: fmt.Sprintf("invalid plan JSON: %v", err)}
	}

	// Persist plan into generateState for agent loop step-driven execution.
	if t.result != nil {
		t.result.PlanSteps = steps
	}

	// Build summary for LLM feedback.
	var done, pending int
	for _, s := range steps {
		switch s.Status {
		case "done":
			done++
		case "pending", "doing":
			pending++
		}
	}

	formatted := strings.ReplaceAll(planJSON, `","`, `", "`)
	summary := fmt.Sprintf("Plan updated (%d/%d steps done):\n%s\n", done, len(steps), formatted)

	if pending > 0 {
		// Find the current "doing" or first "pending" step.
		for _, s := range steps {
			if s.Status == "doing" {
				summary += fmt.Sprintf("\nCurrent step: %s — proceed with write_strategy or edit_code.", s.Step)
				break
			}
		}
		if !strings.Contains(summary, "Current step:") {
			for _, s := range steps {
				if s.Status == "pending" {
					summary += fmt.Sprintf("\nNext step: %s — proceed with write_strategy or edit_code.", s.Step)
					break
				}
			}
		}
	} else {
		summary += "\nAll steps complete — strategy is ready."
	}

	return connectai.ToolOutput{
		Success: true,
		Output:  summary,
	}
}
