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
	sb.WriteString(fmt.Sprintf("Type: %s\n", profile.StrategyType))
	sb.WriteString(fmt.Sprintf("Description: %s\n", profile.Description))
	if len(profile.IndicatorsUsed) > 0 {
		sb.WriteString(fmt.Sprintf("Indicators: %s\n", strings.Join(profile.IndicatorsUsed, ", ")))
	}
	sb.WriteString(fmt.Sprintf("Entry: %s\n", profile.EntryLogic))
	sb.WriteString(fmt.Sprintf("Exit: %s\n", profile.ExitLogic))
	sb.WriteString(fmt.Sprintf("Risk: %s\n", profile.RiskManagement))
}

