package agent

import (
	"fmt"
	"strings"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
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

// writeRequestContext writes symbol, timeframe, and params from a generate request.
// Shared by plan_prompt and buildProfileFromNLPrompt.
func writeRequestContext(sb *strings.Builder, msg *antv1.AgentGenerateStrategyRequest) {
	if msg.Symbol != "" {
		sb.WriteString(fmt.Sprintf("## Trading Symbol\n%s\n", msg.Symbol))
	}
	if msg.Timeframe != "" {
		sb.WriteString(fmt.Sprintf("## Timeframe\n%s\n", msg.Timeframe))
	}
	if len(msg.Params) > 0 {
		sb.WriteString("## Parameter Overrides\n")
		for k, v := range msg.Params {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
	}
}
