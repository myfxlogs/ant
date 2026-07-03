package agent

import (
	"fmt"
	"strings"

	antv1 "anttrader/gen/proto/ant/v1"
)

// planSystemPrompt instructs the LLM to produce a structured strategy plan (ADR-0025 §3).
// Output is line-based KEY: value format, parsed by parsePlanResponse into StrategyPlan proto.
const planSystemPrompt = `You are a quantitative strategy planner. Analyze the user's strategy request and produce a structured strategy plan.
Output the plan in the following line-based format (one key per line):

TYPE: dual_ema_crossover
ENTRY: EMA(10) cross above EMA(30) AND ADX(14) > 25
EXIT: EMA(10) cross below EMA(30) OR trailing_stop(ATR(14)*2)
RISK: 2%_per_trade, RR 1:2
MARKET: trending, ADX>25, H1

Rules:
- TYPE: concise strategy type identifier (snake_case)
- ENTRY: entry conditions in natural language, referencing specific indicators and parameters
- EXIT: exit conditions including stop-loss, take-profit, trailing, or signal-based exits
- RISK: risk management rules (position sizing, max risk per trade, risk-reward ratio)
- MARKET: suitable market regime, timeframe, and any filter conditions
- Keep each field to 1-2 sentences, concise and actionable
- Do NOT output Python code — only the plan
- Output ONLY the KEY: value lines, no markdown, no explanations`

// buildPlanPrompt constructs the user prompt for plan generation.
func buildPlanPrompt(msg *antv1.AgentGenerateStrategyRequest, profile *antv1.StrategyProfile, feedback string) string {
	var sb strings.Builder
	sb.WriteString("## Strategy Request\n")
	sb.WriteString(msg.Message)
	sb.WriteString("\n\n")

	writeProfileToPrompt(&sb, profile, "## Strategy Profile (AI-generated, for reference)\n")
	if profile != nil {
		sb.WriteString("\n")
	}

	writeRequestContext(&sb, msg)

	if feedback != "" {
		sb.WriteString("\n## User Feedback (modify the plan accordingly)\n")
		sb.WriteString(feedback)
		sb.WriteString("\n")
	}

	sb.WriteString("\nProduce the strategy plan now.\n")
	return sb.String()
}

// parsePlanResponse parses KEY: value lines into a StrategyPlan proto.
func parsePlanResponse(raw string) *antv1.StrategyPlan {
	plan := &antv1.StrategyPlan{}
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "```") {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		val := strings.TrimSpace(line[colonIdx+1:])
		val = unquote(val)
		switch key {
		case "TYPE":
			plan.Type = val
		case "ENTRY":
			plan.Entry = val
		case "EXIT":
			plan.Exit = val
		case "RISK":
			plan.Risk = val
		case "MARKET":
			plan.Market = val
		}
	}
	return plan
}

// buildGenerateFromPlanPrompt constructs the user prompt for code generation from a confirmed plan.
func buildGenerateFromPlanPrompt(msg *antv1.AgentGenerateStrategyRequest, plan *antv1.StrategyPlan, profile *antv1.StrategyProfile) (string, string) {
	var sb strings.Builder
	sb.WriteString("## Strategy Description\n")
	sb.WriteString(msg.Message)
	sb.WriteString("\n\n")

	sb.WriteString("## Confirmed Strategy Plan\n")
	sb.WriteString(fmt.Sprintf("Type: %s\n", plan.Type))
	sb.WriteString(fmt.Sprintf("Entry: %s\n", plan.Entry))
	sb.WriteString(fmt.Sprintf("Exit: %s\n", plan.Exit))
	sb.WriteString(fmt.Sprintf("Risk: %s\n", plan.Risk))
	sb.WriteString(fmt.Sprintf("Market: %s\n\n", plan.Market))

	writeProfileToPrompt(&sb, profile, "## Strategy Profile (use as guidance)\n")
	if profile != nil {
		sb.WriteString("\n")
	}

	writeRequestContext(&sb, msg)

	sb.WriteString("\n## Task\n")
	sb.WriteString("Generate a complete Python subset trading strategy based on the confirmed plan above.\n")
	sb.WriteString("Follow the plan's entry, exit, and risk rules precisely.\n")
	sb.WriteString("Use the SDK API mapping shown in the system prompt.\n")
	sb.WriteString("Output ONLY the Python source code.\n")

	return generateSystemPrompt, sb.String()
}
