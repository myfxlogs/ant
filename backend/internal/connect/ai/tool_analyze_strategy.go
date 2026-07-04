package ai

import (
	"context"
	"fmt"

	"anttrader/tools/mql2go"
	systemai "anttrader/internal/service/systemai"
)

// analyzeStrategyTool lets the AI agent analyze MQL source code:
//   - Compile MQL → Bytecode VM (same path as runtime)
//   - Extract extern/input parameters from AST
//   - Report coverage score + blind spots
//
// ADR-0023 Phase 3 #16/#17: Agent uses this to decide whether the
// strategy can run directly on the VM or needs AI translation/repair.
type analyzeStrategyTool struct{}

func (t *analyzeStrategyTool) Name() string { return "analyze_strategy" }
func (t *analyzeStrategyTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name:        "analyze_strategy",
			Description: "分析当前策略代码的编译状态、覆盖度和兼容性。返回：编译是否通过、覆盖度评分、盲区列表、参数列表、推荐操作。用于用户导入MQL代码后判断是否可直接在VM上运行。",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

func (t *analyzeStrategyTool) Run(_ context.Context, in ToolInput) ToolOutput {
	code := in.Code
	if code == "" {
		return ToolOutput{Success: false, Error: "no strategy code provided"}
	}

	// Compile MQL with coverage analysis — same pipeline as runtime.
	runner, coverage, err := mql2go.CompileMQLWithCoverage(code)
	if err != nil {
		return ToolOutput{
			Success: false,
			Error:   fmt.Sprintf("compile failed: %v", err),
			Output: map[string]any{
				"compiles":  false,
				"error":     err.Error(),
				"recommendation": "The MQL code has syntax or compilation errors. AI translation or manual fix is required.",
			},
		}
	}

	// Extract parameters from bytecode (no recompile needed).
	params := mql2go.ExtractParamInfos(runner.Bytecode())

	// Build blind spot summaries.
	var fatalBlindSpots, warningBlindSpots []string
	for _, bs := range coverage.BlindSpots {
		entry := fmt.Sprintf("%s (×%d, %s)", bs.Builtin, bs.Count, bs.Source)
		switch bs.Severity {
		case "fatal":
			fatalBlindSpots = append(fatalBlindSpots, entry)
		case "warning":
			warningBlindSpots = append(warningBlindSpots, entry)
		}
	}

	// Determine recommendation.
	recommendation := "ready_to_run"
	if len(fatalBlindSpots) > 0 {
		recommendation = "needs_ai_translation"
	} else if coverage.Score < 0.8 {
		recommendation = "needs_review"
	}

	// Build param list for the AI to see.
	paramList := make([]map[string]string, 0, len(params))
	for _, p := range params {
		paramList = append(paramList, map[string]string{
			"name":    p.Name,
			"type":    p.Type,
			"default": p.Default,
		})
	}

	return ToolOutput{
		Success: true,
		Output: map[string]any{
			"compiles":          true,
			"coverage_score":    coverage.Score,
			"total_calls":       coverage.TotalCalls,
			"supported_calls":   coverage.SupportedCalls,
			"exec_kind":         coverage.ExecKind,
			"version":           coverage.Version,
			"entry_rules":       coverage.EntryRules,
			"exit_rules":        coverage.ExitRules,
			"indicators":        coverage.Indicators,
			"params":            paramList,
			"fatal_blind_spots": fatalBlindSpots,
			"warn_blind_spots":  warningBlindSpots,
			"recommendation":    recommendation,
		},
	}
}
