// param_proposer.go: AI multi-round parameter proposal via LLM.
// The LLM proposes candidate parameter sets based on previous round results.
// Temperature anneals from 0.7 to 0.8 across rounds to balance exploit/explore.
// Adapted from QuantDinger experiment/prompts.py + runner.py.

package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ProposeRequest contains all inputs for the AI parameter proposer.
type ProposeRequest struct {
	IndicatorCode string              // strategy code with @param annotations
	Params        []TunableParam      // extracted tunable parameters
	RiskParams    map[string]float64  // risk/position params (optional)
	Regime        string              // detected market regime
	Round         int                 // 1-based round number
	MaxCandidates int                 // candidates per round
	PrevResults   []ProposePrevResult // previous round results (empty for round 1)
}

// ProposePrevResult holds a single candidate's result from a previous round.
type ProposePrevResult struct {
	Params      map[string]interface{} `json:"params"`
	Score       float64                `json:"score"`
	Grade       string                 `json:"grade"`
	TotalReturn float64                `json:"total_return"`
	MaxDrawdown float64                `json:"max_drawdown"`
	SharpeRatio float64                `json:"sharpe_ratio"`
	TotalTrades int                    `json:"total_trades"`
}

// ProposeResponse contains the LLM's proposed candidate parameter sets.
type ProposeResponse struct {
	Candidates []map[string]interface{} `json:"candidates"`
}

// Temperature returns the LLM temperature for the given round.
// Rounds 1→0.70, 2→0.75, 3→0.80 (capped at 0.85).
func RoundTemperature(round int) float64 {
	t := 0.7 + float64(round-1)*0.05
	if t > 0.85 {
		t = 0.85
	}
	return t
}

// BuildProposePrompt constructs the system + user prompts for AI parameter proposal.
func BuildProposePrompt(req *ProposeRequest) (systemPrompt, userPrompt string) {
	var sb strings.Builder

	// System prompt
	sb.WriteString("You are a quantitative trading strategy optimization expert.\n")
	sb.WriteString("Your task is to propose diverse parameter sets for backtesting.\n")
	sb.WriteString("Return ONLY a JSON array of objects. No explanations, no markdown.\n\n")
	sb.WriteString("Each object must have keys matching the parameter names exactly.\n")
	systemPrompt = sb.String()

	// User prompt
	sb.Reset()
	if len(req.Params) > 0 {
		sb.WriteString("## Tunable Parameters\n\n")
		for _, p := range req.Params {
			sb.WriteString(fmt.Sprintf("- **%s** (type=%s, default=%.2f, range=%.2f:%.2f:%.2f)\n",
				p.Name, p.Type, p.Default, p.Min, p.Max, p.Step))
		}
		sb.WriteString("\n")
	}

	if len(req.RiskParams) > 0 {
		sb.WriteString("## Risk/Position Parameters\n\n")
		for k, v := range req.RiskParams {
			sb.WriteString(fmt.Sprintf("- %s = %.4f\n", k, v))
		}
		sb.WriteString("\n")
	}

	if req.Regime != "" {
		sb.WriteString(fmt.Sprintf("## Market Regime: %s\n\n", req.Regime))
	}

	if len(req.PrevResults) > 0 {
		sb.WriteString(fmt.Sprintf("## Previous Round Results (Round %d)\n\n", req.Round-1))
		sb.WriteString("| Score | Grade | Return% | Drawdown% | Sharpe | Trades | Params |\n")
		sb.WriteString("|-------|-------|---------|-----------|--------|--------|--------|\n")
		for _, r := range req.PrevResults {
			paramStr, _ := json.Marshal(r.Params)
			sb.WriteString(fmt.Sprintf("| %.1f | %s | %.1f | %.1f | %.2f | %d | %s |\n",
				r.Score, r.Grade, r.TotalReturn, r.MaxDrawdown, r.SharpeRatio, r.TotalTrades, string(paramStr)))
		}
		sb.WriteString("\n")
		if req.Round >= 2 {
			sb.WriteString("Analyze which parameter ranges performed well in the previous round.\n")
			sb.WriteString("Explore promising directions while maintaining diversity.\n\n")
		}
	}

	sb.WriteString(fmt.Sprintf("Propose %d diverse parameter sets for Round %d.\n", req.MaxCandidates, req.Round))
	sb.WriteString("Return as JSON array: [{\"param_name\": value, ...}, ...]")

	return systemPrompt, sb.String()
}

// ParseProposeResponse extracts candidate parameter sets from LLM JSON output.
func ParseProposeResponse(raw string) ([]map[string]interface{}, error) {
	raw = strings.TrimSpace(raw)
	// Strip markdown fences if present
	if strings.HasPrefix(raw, "```") {
		if idx := strings.Index(raw, "\n"); idx >= 0 {
			raw = raw[idx+1:]
		}
		if idx := strings.LastIndex(raw, "```"); idx >= 0 {
			raw = raw[:idx]
		}
	}
	raw = strings.TrimSpace(raw)

	// Try JSON array first
	var candidates []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &candidates); err == nil {
		return candidates, nil
	}

	// Try JSON object with "candidates" key
	var wrapper struct {
		Candidates []map[string]interface{} `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapper); err == nil && len(wrapper.Candidates) > 0 {
		return wrapper.Candidates, nil
	}

	// Try single object → wrap in array
	var single map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &single); err == nil {
		return []map[string]interface{}{single}, nil
	}

	return nil, fmt.Errorf("failed to parse LLM response as JSON array or object")
}

// EarlyStopScore is the threshold for early stopping (QuantDinger: 82.0).
const EarlyStopScore = 82.0

// MaxAIRounds is the maximum number of AI optimization rounds.
const MaxAIRounds = 3
