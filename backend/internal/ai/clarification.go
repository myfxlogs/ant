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
	NeedsClarification bool              `json:"needs_clarification"`
	Questions          []string          `json:"questions"`
	Language           string            `json:"language"`           // detected input language: "zh", "zh-tw", "ja", "vi", "en", or "" (unknown)
	StrategyFamily     string            `json:"strategy_family"`    // trend_following, mean_reversion, breakout, grid, martingale
	StrategyType       string            `json:"strategy_type"`      // "run_dataframe" (vectorized/indicator) or "run_context" (event-driven) or ""
	RiskLevel          string            `json:"risk_level"`         // low, medium, high
	TradeDirection     string            `json:"trade_direction"`    // long, short, both
	MaxDrawdown        string            `json:"max_drawdown"`       // e.g. "0.10", "0.30"
	HoldingPeriod      string            `json:"holding_period"`     // intraday, swing, position
	EntrySignals       []string          `json:"entry_signals"`      // e.g. ["rsi_oversold", "ma_cross"]
	ExitSignals        []string          `json:"exit_signals"`       // e.g. ["take_profit", "trailing_stop"]
	StopLoss           string            `json:"stop_loss"`          // "tight", "medium", "wide", "none"
	TakeProfit         string            `json:"take_profit"`        // "tight", "medium", "wide", "none"
	Confidence         float64           `json:"confidence"`         // 0.0-1.0
}

const intentAnalysisSystemPrompt = `你是量化策略需求分析专家。分析用户的自然语言描述，提取结构化意图。

## 输出格式（严格遵守 — 只要 JSON，不要任何其他文本）

{
  "needs_clarification": true/false,
  "questions": ["问题1", "问题2"],
  "language": "zh|zh-tw|ja|vi|en|",
  "strategy_family": "trend_following|mean_reversion|breakout|grid|martingale|unknown",
  "strategy_type": "run_dataframe|run_context|",
  "risk_level": "low|medium|high|unknown",
  "trade_direction": "long|short|both|unknown",
  "max_drawdown": "容忍的最大回撤，如0.10表示10%。未提及则为空字符串",
  "holding_period": "intraday|swing|position|unknown",
  "entry_signals": ["用户提及的入场信号"],
  "exit_signals": ["用户提及的离场信号"],
  "stop_loss": "tight|medium|wide|none|unknown",
  "take_profit": "tight|medium|wide|none|unknown",
  "confidence": 0.0-1.0
}

## strategy_type 判断规则
- "run_dataframe": 用户提到"指标"、"矢量"、"图表"、"画线"、"均线交叉"、"RSI"、"布林带"、"MACD"、"金叉死叉"、"指标策略"、"数据帧"、"df"等 → 矢量模式
- "run_context": 用户提到"逐笔"、"逐K"、"事件驱动"、"持仓管理"、"移动止损"、"动态仓位"、"分批加仓"、"bot"、"网格"等 → 事件驱动模式
- "": 无法判断或用户未提及，默认为空（系统会使用 run_context 作为默认值）

## 规则
1. needs_clarification=true 当用户描述过于模糊（只说"做个策略"、"帮我赚钱"），此时 questions 必须包含1-3个具体的追问
2. needs_clarification=false 当信息足够生成策略，extract 所有能识别的参数
3. strategy_family 根据策略描述推断，不确定时填 "unknown"
4. confidence 表示你对提取结果的确信度。模糊描述 → 低分，详细描述 → 高分
5. questions 必须是针对性的、用户能直接回答的具体问题（不是泛泛的"请详细描述"），其语言遵循下方的语言要求
6. language 根据用户输入的语言检测，取值为: "zh"(简体中文), "zh-tw"(繁体中文), "ja"(日语), "vi"(越南语), "en"(英语)。无法判断时返回空字符串 ""。仅检测用户输入文本的语言，不受下方语言要求影响。`

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
			// Fallback: if JSON parse fails, treat as unclear (language-aware).
			q, _ := FallbackQuestions(lang)
			return &IntentResult{
				NeedsClarification: true,
				Questions:          []string{q},
				Confidence:         0.0,
			}, nil
		}

	// If confidence is very low but no clarification flagged, add a gentle question
	if !result.NeedsClarification && result.Confidence < 0.4 {
		result.NeedsClarification = true
		_, q := FallbackQuestions(lang)
		result.Questions = []string{q}
	}

	// Deduplicate and limit questions
	result.Questions = deduplicateStrings(result.Questions)
	if len(result.Questions) > 3 {
		result.Questions = result.Questions[:3]
	}

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
	if r.StrategyFamily != "" && r.StrategyFamily != "unknown" {
		pm["strategy_family"] = r.StrategyFamily
	}
	if r.RiskLevel != "" && r.RiskLevel != "unknown" {
		pm["risk_level"] = r.RiskLevel
	}
	if r.MaxDrawdown != "" {
		pm["max_drawdown"] = r.MaxDrawdown
	}
	if r.TradeDirection != "" && r.TradeDirection != "unknown" {
		pm["trade_direction"] = r.TradeDirection
	}
	if r.HoldingPeriod != "" && r.HoldingPeriod != "unknown" {
		pm["holding_period"] = r.HoldingPeriod
	}
	if r.StopLoss != "" && r.StopLoss != "unknown" {
		pm["stop_loss"] = stringToStopLossPct(r.StopLoss)
	}
	if r.TakeProfit != "" && r.TakeProfit != "unknown" {
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
