// clarification.go: LLM-driven intent analysis for strategy generation (spec/26 Phase 1).
//
// Instead of hardcoded keyword matching, uses a lightweight non-streaming LLM call
// to extract structured intent from natural language. Returns either:
//   - Clarification questions (when input is too vague)
//   - Extracted parameters (strategy family, risk level, trade direction, etc.)
//
// This replaces the previous keyword-based ClarificationEngine.
package ai

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

// IntentResult is the output of LLM-driven intent analysis.
type IntentResult struct {
	NeedsClarification bool     `json:"needs_clarification"`
	Questions          []string `json:"questions"`
	Language           string   `json:"language"`        // detected input language
	StrategyFamily     string   `json:"strategy_family"` // trend_following, mean_reversion, breakout, grid, martingale
	StrategyType       string   `json:"strategy_type"`   // "run_dataframe" or "run_context"
	RiskLevel          string   `json:"risk_level"`      // low, medium, high
	TradeDirection     string   `json:"trade_direction"` // long, short, both
	MaxDrawdown        string   `json:"max_drawdown"`
	HoldingPeriod      string   `json:"holding_period"`
	EntrySignals       []string `json:"entry_signals"`
	ExitSignals        []string `json:"exit_signals"`
	StopLoss           string   `json:"stop_loss"`
	TakeProfit         string   `json:"take_profit"`
	Confidence         float64  `json:"confidence"` // 0.0-1.0
	Plan               string   `json:"plan"`       // brief execution plan (1-2 sentences)
}

const intentAnalysisSystemPrompt = `You are a quantitative strategy analyst. Extract structured intent from the user's natural language description.

## Output format (strict — JSON only, no other text)

{
  "needs_clarification": true/false,
  "questions": ["question 1", "question 2"],
  "language": "zh|zh-tw|ja|vi|en|",
  "strategy_family": "trend_following|mean_reversion|breakout|grid|martingale|unknown",
  "strategy_type": "run_dataframe|run_context|",
  "risk_level": "low|medium|high|unknown",
  "trade_direction": "long|short|both|unknown",
  "max_drawdown": "e.g. 0.10 for 10%. Empty string if not mentioned.",
  "holding_period": "intraday|swing|position|unknown",
  "entry_signals": ["signals the user mentioned"],
  "exit_signals": ["signals the user mentioned"],
  "stop_loss": "tight|medium|wide|none|unknown",
  "take_profit": "tight|medium|wide|none|unknown",
  "confidence": 0.0-1.0,
  "plan": "Brief 1-2 sentence execution plan: what kind of strategy, key indicators to use, risk management approach. Use defaults for anything not specified."
}

## CRITICAL RULES (Claude Code style — generate first, refine later)
1. **Only set needs_clarification=true when**: the input is TRULY empty (e.g. "？", "help") or completely nonsensical. Even then, ask at most ONE specific question.
2. **For vague but well-intentioned input** (e.g. "做个策略", "帮我写个EURUSD策略"): set needs_clarification=FALSE. Pick reasonable defaults (mean_reversion, medium risk, swing holding) and generate a plan. The user will refine via feedback — that's faster than a Q&A loop.
3. **For detailed input**: extract everything, set needs_clarification=false, write a concrete plan.
4. **Plan field**: Always write a brief plan summarizing what you'll build. Mention key indicators, entry/exit logic, and risk controls. Fill gaps with sensible defaults.
5. **Confidence**: low confidence is fine — the system will iterate. Do NOT ask questions just because confidence < 0.4.`
// IntentAnalyzer uses an LLM to extract structured intent from NL descriptions.
type IntentAnalyzer struct {
	chatFn func(ctx context.Context, userID uuid.UUID, messages []ChatMessage, model string) (string, error)
}

// ChatMessage mirrors systemai.ChatMessage to avoid circular imports.
type ChatMessage struct {
	Role    string
	Content string
}

// NewIntentAnalyzer creates an analyzer backed by an LLM chat function.
// chatFn should be a lightweight non-streaming chat completion.
func NewIntentAnalyzer(chatFn func(ctx context.Context, userID uuid.UUID, messages []ChatMessage, model string) (string, error)) *IntentAnalyzer {
	return &IntentAnalyzer{chatFn: chatFn}
}

// Analyze sends the user message to the LLM for intent extraction.
// Returns either clarification questions or extracted strategy parameters.
func (a *IntentAnalyzer) Analyze(ctx context.Context, userID uuid.UUID, message, symbol, timeframe, lang, langDirective string) (*IntentResult, error) {
	userMsg := buildIntentUserMessage(message, symbol, timeframe)
	sysPrompt := intentAnalysisSystemPrompt
	if langDirective != "" {
		sysPrompt += "\n\n## 语言要求\n" + langDirective
	}
	resp, err := a.chatFn(ctx, userID, []ChatMessage{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: userMsg},
	}, "") // empty model = use default
	if err != nil {
		return nil, err
	}

	// Extract JSON from response (LLM may wrap in markdown code blocks)
	jsonStr := extractJSON(resp)
	var result IntentResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// JSON parse failed — don't block the user. Generate with defaults.
		return &IntentResult{NeedsClarification: false, Confidence: 0.0}, nil
	}

	// Deduplicate and limit questions (only when truly needed per new prompt rules).
	result.Questions = deduplicateStrings(result.Questions)
	if len(result.Questions) > 1 {
		result.Questions = result.Questions[:1]
	}
	// Never auto-escalate low confidence to clarification — let the user
	// iterate via feedback. This is the Claude Code "ship early, refine later" approach.

	return &result, nil
}

// AnalyzeForFeedback analyzes user feedback for the Phase 3 loop.
func (a *IntentAnalyzer) AnalyzeForFeedback(ctx context.Context, userID uuid.UUID, feedback, previousCode, metricsSummary string) (*IntentResult, error) {
	userMsg := buildFeedbackUserMessage(feedback, previousCode, metricsSummary)
	resp, err := a.chatFn(ctx, userID, []ChatMessage{
		{Role: "system", Content: intentAnalysisSystemPrompt},
		{Role: "user", Content: userMsg},
	}, "")
	if err != nil {
		return nil, err
	}
	jsonStr := extractJSON(resp)
	var result IntentResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return &IntentResult{}, nil
	}
	return &result, nil
}

// ToParamMap converts extracted intent into a parameter map for prompt building.
func (r *IntentResult) ToParamMap() map[string]string {
	pm := make(map[string]string)
	if r.StrategyFamily != "" && r.StrategyFamily != unknownStr {
		pm["strategy_family"] = r.StrategyFamily
	}
	if r.RiskLevel != "" && r.RiskLevel != unknownStr {
		pm["risk_level"] = r.RiskLevel
	}
	if r.MaxDrawdown != "" {
		pm["max_drawdown"] = r.MaxDrawdown
	}
	if r.TradeDirection != "" && r.TradeDirection != unknownStr {
		pm["trade_direction"] = r.TradeDirection
	}
	if r.HoldingPeriod != "" && r.HoldingPeriod != unknownStr {
		pm["holding_period"] = r.HoldingPeriod
	}
	if r.StopLoss != "" && r.StopLoss != unknownStr {
		pm["stop_loss"] = stringToStopLossPct(r.StopLoss)
	}
	if r.TakeProfit != "" && r.TakeProfit != unknownStr {
		pm["take_profit"] = stringToTakeProfitPct(r.TakeProfit)
	}
	return pm
}

func buildIntentUserMessage(message, symbol, timeframe string) string {
	var sb strings.Builder
	sb.WriteString("【用户输入】\n")
	sb.WriteString(message)
	if symbol != "" || timeframe != "" {
		sb.WriteString("\n\n【交易上下文】\n")
		if symbol != "" {
			sb.WriteString("品种: " + symbol + "\n")
		}
		if timeframe != "" {
			sb.WriteString("周期: " + timeframe + "\n")
		}
	}
	return sb.String()
}

func buildFeedbackUserMessage(feedback, previousCode, metricsSummary string) string {
	var sb strings.Builder
	sb.WriteString("【用户反馈】\n")
	sb.WriteString(feedback)
	sb.WriteString("\n\n【回测结果】\n")
	sb.WriteString(metricsSummary)
	sb.WriteString("\n\n【当前策略代码】\n")
	sb.WriteString(truncateCode(previousCode, 500))
	return sb.String()
}

func extractJSON(raw string) string {
	// Remove markdown code fences if present
	s := raw
	if i := strings.Index(s, "```json"); i >= 0 {
		s = s[i+7:]
		if j := strings.LastIndex(s, "```"); j >= 0 {
			s = s[:j]
		}
	} else if i := strings.Index(s, "```"); i >= 0 {
		s = s[i+3:]
		if j := strings.LastIndex(s, "```"); j >= 0 {
			s = s[:j]
		}
	}
	// Find JSON object
	if i := strings.Index(s, "{"); i >= 0 {
		s = s[i:]
		if j := strings.LastIndex(s, "}"); j >= 0 {
			s = s[:j+1]
		}
	}
	return strings.TrimSpace(s)
}

func deduplicateStrings(ss []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func stringToStopLossPct(s string) string {
	switch s {
	case "tight":
		return "0.01"
	case "medium":
		return "0.03"
	case "wide":
		return "0.07"
	default:
		return ""
	}
}

func stringToTakeProfitPct(s string) string {
	switch s {
	case "tight":
		return "0.02"
	case "medium":
		return "0.05"
	case "wide":
		return "0.10"
	default:
		return ""
	}
}

func truncateCode(code string, maxLen int) string {
	if len(code) <= maxLen {
		return code
	}
	return code[:maxLen] + "..."
}

// ── Backward-compatible ClarificationEngine for tests ──

// ClarificationRule defines a fuzzy keyword pattern and its follow-up questions.
// Kept for test compatibility; new code should use IntentAnalyzer.
type ClarificationRule struct {
	Keywords  []string
	Questions []string
	ParamMap  map[string]string
	Priority  int
}

// ClarificationResult is the output of the clarification check.
// Kept for backward compatibility.
type ClarificationResult struct {
	NeedsClarification bool
	Questions          []string
	ParamMap           map[string]string
	MatchedKeyword     string
}

// ClarificationEngine is the legacy keyword-based engine.
// Kept for test backward compatibility. New code uses IntentAnalyzer.
type ClarificationEngine struct {
	rules []ClarificationRule
}

func NewClarificationEngine(rules []ClarificationRule) *ClarificationEngine {
	return &ClarificationEngine{rules: rules}
}

func (e *ClarificationEngine) Check(message string) *ClarificationResult {
	msg := strings.ToLower(message)
	for _, rule := range e.rules {
		if matchAnyKeyword(msg, rule.Keywords) {
			return &ClarificationResult{
				NeedsClarification: true,
				Questions:          rule.Questions,
				ParamMap:           rule.ParamMap,
				MatchedKeyword:     firstMatch(msg, rule.Keywords),
			}
		}
	}
	return nil
}

func matchAnyKeyword(msg string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(msg, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

func firstMatch(msg string, keywords []string) string {
	for _, kw := range keywords {
		if strings.Contains(msg, strings.ToLower(kw)) {
			return kw
		}
	}
	return ""
}
