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
			Description: "复杂策略先拆解为分步计划（入场/仓位/加仓/出场），每步完成后调用 write_strategy 提交代码验证。plan 是 JSON 数组 [{step, status}]，status: pending|doing|done。简单策略可跳过直接 write_strategy。",
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
