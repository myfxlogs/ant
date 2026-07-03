package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service/systemai"
	"anttrader/tools/mql2go"
)

// Profiler generates strategy profiles via LLM (injection point [1]).
// Input: source code + coverage report → LLM → KEY: "value" lines → StrategyProfile proto.
type Profiler struct {
	aiSvc *systemai.Service
	cache *LLCache
}

// NewProfiler creates a strategy profile generator.
func NewProfiler(aiSvc *systemai.Service, cache *LLCache) *Profiler {
	return &Profiler{aiSvc: aiSvc, cache: cache}
}

const profileSystemPrompt = `You are a quantitative strategy analyst. Analyze the given trading strategy source code and coverage report.
Produce a strategy profile in the following line-based format (one key per line):

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
coverage_score: 0.85
blind_spots: "iCustom,WebRequest"

Rules:
- Each line is KEY: value (or KEY: "value" for strings with spaces)
- indicators_used is comma-separated
- coverage_score is a number 0.0-1.0
- blind_spots is comma-separated (empty if none)
- Keep values concise (1-2 sentences max per field)
- Output ONLY the key-value lines, no markdown, no explanations`

// GenerateProfile calls LLM to produce a StrategyProfile from source + coverage.
func (p *Profiler) GenerateProfile(
	ctx context.Context,
	userID string,
	sourceCode string,
	coverage *mql2go.CoverageResult,
) (*antv1.StrategyProfile, error) {
	userPrompt := buildProfileUserPrompt(sourceCode, coverage)

	// Check cache — avoid redundant LLM calls during Agent iteration.
	if p.cache != nil {
		if cached, ok := p.cache.Get(sourceCode, userPrompt); ok {
			return parseProfileResponse(cached, coverage), nil
		}
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := p.aiSvc.ChatCompletion(llmCtx, uid, []systemai.ChatMessage{
		{Role: "system", Content: profileSystemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return nil, fmt.Errorf("profile LLM call: %w", err)
	}

	if p.cache != nil {
		p.cache.Set(sourceCode, userPrompt, resp)
	}

	return parseProfileResponse(resp, coverage), nil
}

func buildProfileUserPrompt(source string, cov *mql2go.CoverageResult) string {
	var sb strings.Builder
	sb.WriteString("## Strategy Source Code\n```\n")
	sb.WriteString(source)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Coverage Report\n")
	sb.WriteString(fmt.Sprintf("Coverage Score: %.2f\n", cov.Score))
	sb.WriteString(fmt.Sprintf("Total Calls: %d, Supported: %d\n", cov.TotalCalls, cov.SupportedCalls))
	sb.WriteString(fmt.Sprintf("Execution Model: %s\n", cov.ExecKind))
	sb.WriteString(fmt.Sprintf("MQL Version: %s\n", cov.Version))
	sb.WriteString(fmt.Sprintf("Entry Rules: %d, Exit Rules: %d\n", cov.EntryRules, cov.ExitRules))

	if len(cov.Indicators) > 0 {
		sb.WriteString("Indicators: " + strings.Join(cov.Indicators, ", ") + "\n")
	}
	if len(cov.BlindSpots) > 0 {
		sb.WriteString("Blind Spots:\n")
		for _, bs := range cov.BlindSpots {
			sb.WriteString(fmt.Sprintf("  - %s (severity=%s, count=%d)\n", bs.Builtin, bs.Severity, bs.Count))
		}
	}

	sb.WriteString("\nProduce the strategy profile now.\n")
	return sb.String()
}

// parseProfileResponse parses KEY: "value" lines into StrategyProfile proto.
func parseProfileResponse(raw string, cov *mql2go.CoverageResult) *antv1.StrategyProfile {
	profile := parseProfileLines(raw)
	profile.CoverageScore = cov.Score
	if cov.Indicators != nil {
		profile.IndicatorsUsed = append([]string(nil), cov.Indicators...)
	}
	if len(cov.BlindSpots) > 0 {
		for _, bs := range cov.BlindSpots {
			profile.BlindSpots = append(profile.BlindSpots, bs.Builtin)
		}
	}
	return profile
}
