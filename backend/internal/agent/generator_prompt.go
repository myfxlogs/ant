package agent

import (
	"fmt"
	"strings"

	antv1 "anttrader/gen/proto/ant/v1"
)

// generateSystemPrompt instructs the LLM to generate a Python subset strategy from natural language.
// Shares Python subset rules + SDK API mapping via pythonSubsetRules (prompts_shared.go).
const generateSystemPrompt = `You are a quantitative strategy generator. Your task is to generate a Python subset trading strategy from a natural language description.

` + pythonSubsetRules + `

### Trading
- Buy → ctx.broker().buy(lot=Decimal("0.1"))
- Sell → ctx.broker().sell(lot=Decimal("0.1"))
- Close position → ctx.broker().close(ticket)
- Modify position → ctx.broker().modify(ticket, sl, tp)
- Delete order → ctx.broker().delete(ticket)
- Buy Limit → ctx.broker().buy_limit(lot, price, sl, tp)
- Sell Limit → ctx.broker().sell_limit(lot, price, sl, tp)
- Buy Stop → ctx.broker().buy_stop(lot, price, sl, tp)
- Sell Stop → ctx.broker().sell_stop(lot, price, sl, tp)
- Position count → ctx.positions().count()
- Iterate positions → for pos in ctx.positions: pos.ticket, pos.profit, pos.volume, pos.sl, pos.tp

## Output Format
Output ONLY the Python source code, no markdown fences, no explanations.
The code must be a complete, compilable Python subset strategy.
All parameters must have concrete default values — no TODOs or placeholders.
For unspecified parameters, use reasonable defaults based on the strategy description.`

func buildGeneratePrompt(msg *antv1.AgentGenerateStrategyRequest, profile *antv1.StrategyProfile) (string, string) {
	var sb strings.Builder
	sb.WriteString("## Strategy Description\n")
	sb.WriteString(msg.Message)
	sb.WriteString("\n\n")

	writeProfileToPrompt(&sb, profile, "## Strategy Profile (use as guidance)\n")
	if profile != nil {
		sb.WriteString("\n")
	}

	writeRequestContext(&sb, msg)

	sb.WriteString("\n## Task\n")
	sb.WriteString("Generate a complete Python subset trading strategy based on the description above.\n")
	sb.WriteString("Use the SDK API mapping shown in the system prompt.\n")
	sb.WriteString("Output ONLY the Python source code.\n")

	return generateSystemPrompt, sb.String()
}

func buildGenerateRetryPrompt(msg *antv1.AgentGenerateStrategyRequest, prevCode, compileErr, btErr string, profile *antv1.StrategyProfile) (string, string) {
	var sb strings.Builder
	sb.WriteString("## Previous Attempt (failed validation)\n```\n")
	sb.WriteString(prevCode)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Error\n")
	if compileErr != "" {
		sb.WriteString("Compile error:\n")
		sb.WriteString(compileErr)
	} else if btErr != "" {
		sb.WriteString("Backtest error:\n")
		sb.WriteString(btErr)
	}

	if profile != nil {
		sb.WriteString("\n\n## Strategy Profile (for context)\n")
		sb.WriteString(fmt.Sprintf("Type: %s\n", profile.StrategyType))
		sb.WriteString(fmt.Sprintf("Indicators: %s\n", strings.Join(profile.IndicatorsUsed, ", ")))
		sb.WriteString(fmt.Sprintf("Entry: %s\n", profile.EntryLogic))
		sb.WriteString(fmt.Sprintf("Exit: %s\n", profile.ExitLogic))
	}

	sb.WriteString("\n\n## Original Strategy Description\n")
	sb.WriteString(msg.Message)

	writeRequestContext(&sb, msg)

	sb.WriteString("\n\n## Task\n")
	sb.WriteString("The previous Python strategy failed validation (compile error or backtest failure).\n")
	sb.WriteString("Fix the issue above and output the corrected Python source code.\n")
	sb.WriteString("Output ONLY the Python source code.\n")

	return generateSystemPrompt, sb.String()
}

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
	var sb strings.Builder
	sb.WriteString("## Strategy Description\n")
	sb.WriteString(msg.Message)
	sb.WriteString("\n\n")
	writeRequestContext(&sb, msg)
	sb.WriteString("\nProduce the strategy profile now.\n")
	return sb.String()
}
