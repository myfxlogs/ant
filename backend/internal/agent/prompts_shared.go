package agent

import (
	"fmt"
	"strings"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/ai"
)

// pythonSubsetRules is the shared Python subset language rules block.
// Aliased from internal/ai — single source of truth.
const pythonSubsetRules = ai.PythonSubsetRules

// writeProfileToPrompt writes a StrategyProfile as context to a prompt builder.
// Shared by buildBridgeRetryPrompt, buildBridgeUserPrompt, and buildAnalysisUserPrompt.
func writeProfileToPrompt(sb *strings.Builder, profile *antv1.StrategyProfile, header string) {
	if profile == nil {
		return
	}
	sb.WriteString(header)
	fmt.Fprintf(sb, "Type: %s\n", profile.StrategyType)
	fmt.Fprintf(sb, "Description: %s\n", profile.Description)
	if len(profile.IndicatorsUsed) > 0 {
		fmt.Fprintf(sb, "Indicators: %s\n", strings.Join(profile.IndicatorsUsed, ", "))
	}
	fmt.Fprintf(sb, "Entry: %s\n", profile.EntryLogic)
	fmt.Fprintf(sb, "Exit: %s\n", profile.ExitLogic)
	fmt.Fprintf(sb, "Risk: %s\n", profile.RiskManagement)
}

