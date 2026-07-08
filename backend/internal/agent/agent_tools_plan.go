package agent

import (
	"context"
	"fmt"
	"strings"

	connectai "anttrader/internal/connect/ai"
	systemai "anttrader/internal/service/systemai"
)

// updatePlanTool allows the LLM to track and display a multi-step strategy plan.
type updatePlanTool struct{}

func (t *updatePlanTool) Name() string { return "update_plan" }

func (t *updatePlanTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name: "update_plan",
			Description: "复杂策略的分步计划工具（JSON [{step, status}]）。仅多步策略使用，简单策略请直接调用write_strategy。",
			Parameters: map[string]any{
				"type":     "object",
				"required": []string{"plan"},
				"properties": map[string]any{
					"plan": map[string]any{
						"type": "string",
						"description": "JSON数组字符串，每项 {step, status}。例如: [{\"step\":\"EMA入场\",\"status\":\"done\"},{\"step\":\"ATR止损\",\"status\":\"doing\"}]",
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
	// Parse the plan for nice display
	formatted := strings.ReplaceAll(planJSON, `","`, `", "`)
	return connectai.ToolOutput{
		Success: true,
		Output:  fmt.Sprintf("Plan updated:\n%s", formatted),
	}
}
