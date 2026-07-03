package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service/systemai"
)

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

// retrospectSystemPrompt instructs the LLM to produce a concise experience entry.
const retrospectSystemPrompt = `You are a strategy retrospective analyst. Analyze the completed strategy generation and produce a concise experience entry for future reference.
Output a single paragraph (3-5 sentences) covering:
1. What strategy type was attempted
2. Key entry/exit/risk decisions
3. Backtest outcome (success/failure, key metrics)
4. What worked well or what went wrong
5. One actionable lesson for future strategy generation

Be specific and quantitative. Do NOT output markdown or code.`

// Run analyzes the generation result and stores an experience asynchronously.
// This is the PostStrategyGeneration lifecycle hook (ADR-0025 §6.2).
func (r *RetrospectAgent) Run(ctx context.Context, userID uuid.UUID, input retrospectInput) {
	if r.memory == nil {
		return
	}

	content, err := r.generateRetrospect(ctx, userID, input)
	if err != nil {
		r.log.Warn("retrospect: LLM analysis failed, using fallback", zap.Error(err))
		content = r.fallbackRetrospect(input)
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
	sb.WriteString("## Strategy Generation Summary\n")
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
		r := input.BacktestResult
		sb.WriteString(fmt.Sprintf("Backtest: success=%v, trades=%d, return=%.2f%%, drawdown=%.2f%%, sharpe=%.2f, win_rate=%.1f%%\n",
			r.Success, r.TotalTrades, r.TotalReturn, r.MaxDrawdown, r.SharpeRatio, r.WinRate))
		if r.Error != "" {
			sb.WriteString(fmt.Sprintf("Backtest Error: %s\n", r.Error))
		}
	}

	if input.Analysis != nil && input.Analysis.Summary != "" {
		sb.WriteString(fmt.Sprintf("Analysis: %s\n", input.Analysis.Summary))
	}

	sb.WriteString(fmt.Sprintf("Coverage Score: %.0f%%\n", input.CoverageScore*100))
	sb.WriteString("\nProduce the experience entry now.\n")
	return sb.String()
}

func (r *RetrospectAgent) fallbackRetrospect(input retrospectInput) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Strategy: %s on %s %s. ", input.Message[:min(60, len(input.Message))], input.Symbol, input.Timeframe))
	if input.BacktestResult != nil {
		r := input.BacktestResult
		if r.Success {
			sb.WriteString(fmt.Sprintf("Backtest succeeded: %d trades, %.1f%% return, %.1f%% drawdown. ", r.TotalTrades, r.TotalReturn, r.MaxDrawdown))
		} else {
			sb.WriteString(fmt.Sprintf("Backtest failed: %s. ", r.Error))
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
