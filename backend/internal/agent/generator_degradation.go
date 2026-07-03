package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service/systemai"
)

// fireDegradationAlert fires the DegradationAlert hook (ADR-0025 §8).
// Pushes a warning to the frontend via stream (SSE push type) and fires HookEngine for command/webhook handlers.
func (g *Generator) fireDegradationAlert(ctx context.Context, userID uuid.UUID, reason string, stream func(*antv1.AgentGenerateStrategyChunk) error) {
	g.log.Warn("generator: degradation alert", zap.String("reason", reason))

	// SSE push to frontend (ADR-0025 §8: DegradationAlert type = SSE push)
	if stream != nil {
		_ = stream(&antv1.AgentGenerateStrategyChunk{
			Phase: "degradation_alert",
			Error: reason,
		})
	}

	// Fire HookEngine for command/webhook handlers
	if g.hooks != nil && g.hooks.HasHandlers(HookDegradationAlert) {
		go g.hooks.Fire(context.Background(), &HookContext{
			Event:             HookDegradationAlert,
			UserID:            userID,
			DegradationReason: reason,
		})
	}
}

// degradedAnalysis returns a template BacktestAnalysis when LLM analysis fails (ADR-0024 §5.4).
func degradedAnalysis(r *antv1.AgentBacktestResult) *antv1.BacktestAnalysis {
	summary := "Backtest completed. LLM analysis unavailable — see metrics above."
	if r != nil && r.TotalTrades > 0 {
		summary = fmt.Sprintf(
			"Backtest completed with %d trades. Total return: %.2f%%, Max drawdown: %.2f%%, Win rate: %.1f%%. LLM analysis unavailable.",
			r.TotalTrades, r.TotalReturn, r.MaxDrawdown, r.WinRate,
		)
	}
	return &antv1.BacktestAnalysis{Summary: summary}
}

// generatePlan calls LLM to produce a structured StrategyPlan from NL + profile (ADR-0025 §3).
func (g *Generator) generatePlan(
	ctx context.Context,
	userID uuid.UUID,
	msg *antv1.AgentGenerateStrategyRequest,
	profile *antv1.StrategyProfile,
	sessionMem *SessionMemory,
) (*antv1.StrategyPlan, error) {
	userPrompt := buildPlanPrompt(msg, profile, msg.PlanFeedback, sessionMem)

	llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := g.aiSvc.ChatCompletion(llmCtx, userID, []systemai.ChatMessage{
		{Role: "system", Content: planSystemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return nil, fmt.Errorf("plan LLM call: %w", err)
	}

	return parsePlanResponse(resp), nil
}
