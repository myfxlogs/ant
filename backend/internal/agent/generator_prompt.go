package agent

import (
	_ "embed"
	"fmt"
	"strings"

	antv1 "anttrader/gen/proto/ant/v1"
)

//go:embed prompts/profile_from_nl_user.prompt
var profileFromNLUserPromptTmpl string

// profileFromNLSystemPrompt instructs the LLM to produce a strategy profile from NL description.
const profileFromNLSystemPrompt = `You are a quantitative strategy analyst. Analyze the given natural language strategy description and produce a strategy profile in the following line-based format (one key per line):

strategy_type: "trend_following"
description: "1-2 sentence summary"
indicators_used: "EMA,RSI,ATR"
entry_logic: "description of entry conditions"
exit_logic: "description of exit conditions"
risk_management: "stop-loss, take-profit, trailing, position sizing"
timeframe_preference: "H1"
market_regime: "trending"
strengths: "what the strategy does well"
weaknesses: "potential issues"

Rules:
- Each line is KEY: value (or KEY: "value" for strings with spaces)
- indicators_used is comma-separated
- Keep values concise (1-2 sentences max per field)
- Output ONLY the key-value lines, no markdown, no explanations`

func buildProfileFromNLPrompt(msg *antv1.AgentGenerateStrategyRequest) string {
	data := buildPromptData(msg, nil, nil)
	userPrompt, err := renderPrompt("profile_from_nl_user", profileFromNLUserPromptTmpl, data)
	if err != nil {
		return fallbackProfileFromNLPrompt(msg)
	}
	return userPrompt
}

// buildPromptData constructs the shared prompt template data from request + profile + memory.
// User-controlled fields are sanitized via sanitizeInput and wrapped in XML tags.
func buildPromptData(msg *antv1.AgentGenerateStrategyRequest, profile *antv1.StrategyProfile, mem *SessionMemory) promptData {
	data := promptData{
		Message: wrapXML("user_input", sanitizeInput(msg.Message)),
	}
	if len(msg.Params) > 0 {
		var sb strings.Builder
		sb.WriteString("## Parameter Overrides\n")
		for k, v := range msg.Params {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
		data.Params = sb.String()
	}
	if profile != nil {
		var sb strings.Builder
		writeProfileToPrompt(&sb, profile, "## Strategy Profile (use as guidance)\n")
		data.ProfileBlock = sb.String()
	}
	if mem != nil {
		var sb strings.Builder
		mem.InjectIntoPrompt(&sb)
		data.MemoryBlock = sb.String()
	}
	return data
}

func fallbackProfileFromNLPrompt(msg *antv1.AgentGenerateStrategyRequest) string {
	var sb strings.Builder
	sb.WriteString("## Strategy Description\n")
	sb.WriteString(msg.Message)
	sb.WriteString("\n\n")
	writeRequestContext(&sb, msg)
	sb.WriteString("\nProduce the strategy profile now.\n")
	return sb.String()
}
