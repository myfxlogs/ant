package agent

import (
	_ "embed"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service/systemai"
)

//go:embed prompts/retrospect_user.prompt
var retrospectUserPromptTmpl string

// RetrospectAgent analyzes completed strategy generation results and stores
// experiences to the memory system (ADR-0025 §6.2).
// It runs asynchronously after generation completes — failures are logged, not surfaced to user.
type RetrospectAgent struct {
	aiSvc  *systemai.Service
	memory *MemoryStore
	log    *zap.Logger
}

// NewRetrospectAgent creates a retrospect Agent.
func NewRetrospectAgent(aiSvc *systemai.Service, memory *MemoryStore, log *zap.Logger) *RetrospectAgent {
	return &RetrospectAgent{aiSvc: aiSvc, memory: memory, log: log}
}

// retrospectInput is the context for the retrospect LLM call.
type retrospectInput struct {
	Message      string
	Symbol       string
	Timeframe    string
	Profile      *antv1.StrategyProfile
	Plan         *antv1.StrategyPlan
	BacktestResult *antv1.AgentBacktestResult
	Analysis     *antv1.BacktestAnalysis
	CoverageScore float64
}

// retrospectSystemPrompt instructs the LLM to produce a structured experience entry (ADR-0025 §6.2).
// Output is line-based KEY: value format, parsed by parseRetrospectResponse.
const retrospectSystemPrompt = `You are a strategy retrospective analyst. Analyze the completed strategy generation and produce a structured experience entry.
Output in the following line-based format (one key per line):

KEY_FINDING: EMA(12,26) c/o on EURUSD H1: sharpe 1.6, maxDD 14%
SUCCESS_FACTOR: ADX>25 filter reduced whipsaws by 60%
FAILURE_AVOID: Avoid <8 UTC session, false breakout rate >40%

Rules:
- KEY_FINDING: the most important quantitative finding from this strategy generation
- SUCCESS_FACTOR: what worked well (omit if backtest failed)
- FAILURE_AVOID: what to avoid in future (omit if nothing notable)
- Be specific and quantitative — use actual metrics from the backtest
- Each field should be 1-2 sentences
- Output ONLY the KEY: value lines, no markdown, no explanations`

// retrospectResult holds parsed structured output from the retrospect LLM call.
type retrospectResult struct {
	KeyFinding     string
	SuccessFactor  string
	FailureAvoid   string
}

// parseRetrospectResponse parses KEY: value lines into a structured result.
func parseRetrospectResponse(raw string) retrospectResult {
	result := retrospectResult{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
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
		switch key {
		case "KEY_FINDING":
			result.KeyFinding = val
		case "SUCCESS_FACTOR":
			result.SuccessFactor = val
		case "FAILURE_AVOID":
			result.FailureAvoid = val
		}
	}
	return result
}

// formatRetrospectContent combines structured fields into a single content string for storage.
func formatRetrospectContent(r retrospectResult) string {
	var sb strings.Builder
	if r.KeyFinding != "" {
		sb.WriteString("Finding: ")
		sb.WriteString(r.KeyFinding)
	}
	if r.SuccessFactor != "" {
		if sb.Len() > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString("Success: ")
		sb.WriteString(r.SuccessFactor)
	}
	if r.FailureAvoid != "" {
		if sb.Len() > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString("Avoid: ")
		sb.WriteString(r.FailureAvoid)
	}
	if sb.Len() == 0 {
		return "No structured findings extracted."
	}
	return sb.String()
}

// Run analyzes the generation result and stores an experience asynchronously.
// This is the PostStrategyGeneration lifecycle hook (ADR-0025 §6.2).
func (r *RetrospectAgent) Run(ctx context.Context, userID uuid.UUID, input retrospectInput) {
	defer func() {
		if rec := recover(); rec != nil {
			r.log.Error("retrospect panicked", zap.Any("recover", rec))
		}
	}()
	if r.memory == nil {
		return
	}

	rawContent, err := r.generateRetrospect(ctx, userID, input)
	if err != nil {
		r.log.Warn("retrospect: LLM analysis failed, using fallback", zap.Error(err))
		rawContent = r.fallbackRetrospect(input)
	}

	// Parse structured output, fall back to raw content if parsing yields nothing
	parsed := parseRetrospectResponse(rawContent)
	content := formatRetrospectContent(parsed)
	if content == "No structured findings extracted." {
		content = rawContent // use raw LLM output if structured parse failed
	}

	fingerprint := r.buildFingerprint(input)
	indicators := r.extractIndicators(input.Profile)
	conditionStructure := r.extractConditionStructure(input.Profile)

	_, storeErr := r.memory.StoreExperience(
		ctx, userID, "strategy_pattern", content, fingerprint, indicators, conditionStructure,
	)
	if storeErr != nil {
		r.log.Warn("retrospect: store experience failed", zap.Error(storeErr))
	}
}

func (r *RetrospectAgent) generateRetrospect(ctx context.Context, userID uuid.UUID, input retrospectInput) (string, error) {
	userPrompt := r.buildRetrospectPrompt(input)

	llmCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	resp, err := r.aiSvc.ChatCompletion(llmCtx, userID, []systemai.ChatMessage{
		{Role: "system", Content: retrospectSystemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return "", fmt.Errorf("retrospect LLM call: %w", err)
	}
	return strings.TrimSpace(resp), nil
}

func (r *RetrospectAgent) buildRetrospectPrompt(input retrospectInput) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Request: %s\n", input.Message))
	sb.WriteString(fmt.Sprintf("Symbol: %s, Timeframe: %s\n", input.Symbol, input.Timeframe))
	if input.Plan != nil {
		sb.WriteString(fmt.Sprintf("Plan Type: %s\n", input.Plan.Type))
		sb.WriteString(fmt.Sprintf("Entry: %s\n", input.Plan.Entry))
		sb.WriteString(fmt.Sprintf("Exit: %s\n", input.Plan.Exit))
	}
	if input.Profile != nil {
		sb.WriteString(fmt.Sprintf("Profile: %s — %s\n", input.Profile.StrategyType, input.Profile.Description))
	}
	if input.BacktestResult != nil {
		bt := input.BacktestResult
		sb.WriteString(fmt.Sprintf("Backtest: success=%v, trades=%d, return=%.2f%%, drawdown=%.2f%%, sharpe=%.2f, win_rate=%.1f%%\n",
			bt.Success, bt.TotalTrades, bt.TotalReturn, bt.MaxDrawdown, bt.SharpeRatio, bt.WinRate))
		if bt.Error != "" {
			sb.WriteString(fmt.Sprintf("Backtest Error: %s\n", bt.Error))
		}
	}
	if input.Analysis != nil && input.Analysis.Summary != "" {
		sb.WriteString(fmt.Sprintf("Analysis: %s\n", input.Analysis.Summary))
	}
	sb.WriteString(fmt.Sprintf("Coverage Score: %.0f%%\n", input.CoverageScore*100))
	data := promptData{
		RetrospectBlock: wrapXML("generation_summary", sanitizeInput(sb.String())),
	}
	userPrompt, err := renderPrompt("retrospect_user", retrospectUserPromptTmpl, data)
	if err != nil {
		return fallbackRetrospectPrompt(input)
	}
	return userPrompt
}

func (r *RetrospectAgent) fallbackRetrospect(input retrospectInput) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Strategy: %s on %s %s. ", input.Message[:min(60, len(input.Message))], input.Symbol, input.Timeframe))
	if input.BacktestResult != nil {
		bt := input.BacktestResult
		if bt.Success {
			sb.WriteString(fmt.Sprintf("Backtest succeeded: %d trades, %.1f%% return, %.1f%% drawdown. ", bt.TotalTrades, bt.TotalReturn, bt.MaxDrawdown))
		} else {
			sb.WriteString(fmt.Sprintf("Backtest failed: %s. ", bt.Error))
		}
	}
	sb.WriteString("Retrospect LLM unavailable — see metrics for details.")
	return sb.String()
}

func (r *RetrospectAgent) buildFingerprint(input retrospectInput) string {
	if input.Profile == nil {
		return ""
	}
	parts := []string{input.Profile.StrategyType}
	parts = append(parts, input.Profile.IndicatorsUsed...)
	return strings.Join(parts, "|")
}

func (r *RetrospectAgent) extractIndicators(profile *antv1.StrategyProfile) []string {
	if profile == nil {
		return nil
	}
	return profile.IndicatorsUsed
}

func (r *RetrospectAgent) extractConditionStructure(profile *antv1.StrategyProfile) string {
	if profile == nil {
		return ""
	}
	if profile.EntryLogic != "" && profile.ExitLogic != "" {
		return "entry_exit"
	}
	if profile.EntryLogic != "" {
		return "entry_only"
	}
	return ""
}

func fallbackRetrospectPrompt(input retrospectInput) string {
	var sb strings.Builder
	sb.WriteString("## Strategy Generation Summary\n")
	sb.WriteString(fmt.Sprintf("Request: %s\n", input.Message))
	sb.WriteString(fmt.Sprintf("Symbol: %s, Timeframe: %s\n", input.Symbol, input.Timeframe))
	if input.BacktestResult != nil {
		bt := input.BacktestResult
		sb.WriteString(fmt.Sprintf("Backtest: success=%v, trades=%d, return=%.2f%%\n",
			bt.Success, bt.TotalTrades, bt.TotalReturn))
	}
	sb.WriteString(fmt.Sprintf("Coverage Score: %.0f%%\n", input.CoverageScore*100))
	sb.WriteString("\nProduce the experience entry now.\n")
	return sb.String()
}
