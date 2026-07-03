package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service/systemai"
)

// Interpreter generates backtest analysis via LLM (injection point [4]).
// Input: backtest result + strategy profile → LLM → KEY: "value" lines → BacktestAnalysis proto.
type Interpreter struct {
	aiSvc *systemai.Service
	log   *zap.Logger
	cache *LLCache
}

// NewInterpreter creates a backtest analysis generator.
func NewInterpreter(aiSvc *systemai.Service, log *zap.Logger, cache *LLCache) *Interpreter {
	return &Interpreter{aiSvc: aiSvc, log: log, cache: cache}
}

const analysisSystemPrompt = `You are a quantitative backtest analyst. Analyze the given backtest results and strategy profile.
Produce a backtest analysis in the following line-based format (one key per line):

summary: "1-2 sentence overall assessment"
performance_grade: "B"
sharpe_assessment: 0.7
drawdown_assessment: 0.6
win_rate_assessment: 0.8
profit_consistency: "consistent"
risk_adjusted_return: "good"
key_observations: "observation 1; observation 2; observation 3"
improvement_suggestions: "suggestion 1; suggestion 2"
overfitting_risk: "moderate"
recommended_action: "optimize"
detailed_analysis: "longer-form paragraph analyzing the strategy's backtest performance"

Rules:
- Each line is KEY: value (or KEY: "value" for strings with spaces)
- performance_grade is a single letter: A, B, C, D, F
- sharpe/drawdown/win_rate_assessment are numbers 0.0-1.0
- profit_consistency: consistent, inconsistent, streaky
- risk_adjusted_return: excellent, good, moderate, poor
- overfitting_risk: low, moderate, high
- recommended_action: deploy, optimize, refine, discard
- key_observations and improvement_suggestions are semicolon-separated
- Keep values concise and actionable
- Output ONLY the key-value lines, no markdown, no explanations`

// AnalyzeBacktest calls LLM to produce a BacktestAnalysis from backtest results + profile.
func (i *Interpreter) AnalyzeBacktest(
	ctx context.Context,
	userID string,
	result *antv1.AgentBacktestResult,
	profile *antv1.StrategyProfile,
) (*antv1.BacktestAnalysis, error) {
	userPrompt := buildAnalysisUserPrompt(result, profile)

	// Cache key: use result summary as source identifier + prompt.
	cacheSource := fmt.Sprintf("%v_%d_%d", result.Success, result.TotalTrades, len(result.EquityCurve))

	// Check cache — avoid redundant LLM calls during Agent iteration.
	if i.cache != nil {
		if cached, ok := i.cache.Get(cacheSource, userPrompt); ok {
			return parseAnalysisResponse(cached), nil
		}
	}

	uid, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	resp, err := i.aiSvc.ChatCompletion(ctx, uid, []systemai.ChatMessage{
		{Role: "system", Content: analysisSystemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return nil, fmt.Errorf("analysis LLM call: %w", err)
	}

	if i.cache != nil {
		i.cache.Set(cacheSource, userPrompt, resp)
	}

	return parseAnalysisResponse(resp), nil
}

func buildAnalysisUserPrompt(result *antv1.AgentBacktestResult, profile *antv1.StrategyProfile) string {
	var sb strings.Builder

	if profile != nil {
		sb.WriteString("## Strategy Profile\n")
		sb.WriteString(fmt.Sprintf("Type: %s\n", profile.StrategyType))
		sb.WriteString(fmt.Sprintf("Description: %s\n", profile.Description))
		sb.WriteString(fmt.Sprintf("Indicators: %s\n", strings.Join(profile.IndicatorsUsed, ", ")))
		sb.WriteString(fmt.Sprintf("Entry: %s\n", profile.EntryLogic))
		sb.WriteString(fmt.Sprintf("Exit: %s\n", profile.ExitLogic))
		sb.WriteString(fmt.Sprintf("Risk Management: %s\n", profile.RiskManagement))
		sb.WriteString("\n")
	}

	sb.WriteString("## Backtest Results\n")
	sb.WriteString(fmt.Sprintf("Success: %v\n", result.Success))
	if result.Error != "" {
		sb.WriteString(fmt.Sprintf("Error: %s\n", result.Error))
	}
	sb.WriteString(fmt.Sprintf("Total Return: %.2f%%\n", result.TotalReturn*100))
	sb.WriteString(fmt.Sprintf("Annual Return: %.2f%%\n", result.AnnualReturn*100))
	sb.WriteString(fmt.Sprintf("Max Drawdown: %.2f%%\n", result.MaxDrawdown*100))
	sb.WriteString(fmt.Sprintf("Sharpe Ratio: %.2f\n", result.SharpeRatio))
	sb.WriteString(fmt.Sprintf("Win Rate: %.1f%%\n", result.WinRate*100))
	sb.WriteString(fmt.Sprintf("Profit Factor: %.2f\n", result.ProfitFactor))
	sb.WriteString(fmt.Sprintf("Total Trades: %d (Win: %d, Loss: %d)\n",
		result.TotalTrades, result.WinningTrades, result.LosingTrades))
	sb.WriteString(fmt.Sprintf("Total PnL: %s\n", result.TotalPnlAbsolute))

	if len(result.EquityCurve) > 0 {
		sb.WriteString(fmt.Sprintf("Equity Points: %d\n", len(result.EquityCurve)))
		sb.WriteString(fmt.Sprintf("Start Equity: %s\n", result.EquityCurve[0]))
		sb.WriteString(fmt.Sprintf("End Equity: %s\n", result.EquityCurve[len(result.EquityCurve)-1]))
	}

	if len(result.Trades) > 0 && len(result.Trades) <= 50 {
		sb.WriteString("\n## Trades\n")
		for _, t := range result.Trades {
			sb.WriteString(fmt.Sprintf("  #%d %s vol=%s pnl=%s reason=%s\n",
				t.Ticket, t.Side, t.Volume, t.Pnl, t.Reason))
		}
	} else if len(result.Trades) > 50 {
		sb.WriteString(fmt.Sprintf("\n## Trades (showing first 10 of %d)\n", len(result.Trades)))
		for _, t := range result.Trades[:10] {
			sb.WriteString(fmt.Sprintf("  #%d %s vol=%s pnl=%s\n", t.Ticket, t.Side, t.Volume, t.Pnl))
		}
	}

	sb.WriteString("\nProduce the backtest analysis now.\n")
	return sb.String()
}

// parseAnalysisResponse parses KEY: "value" lines into BacktestAnalysis proto.
func parseAnalysisResponse(raw string) *antv1.BacktestAnalysis {
	analysis := &antv1.BacktestAnalysis{}

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
		case "summary":
			analysis.Summary = val
		case "performance_grade":
			analysis.PerformanceGrade = val
		case "sharpe_assessment":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				analysis.SharpeAssessment = f
			}
		case "drawdown_assessment":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				analysis.DrawdownAssessment = f
			}
		case "win_rate_assessment":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				analysis.WinRateAssessment = f
			}
		case "profit_consistency":
			analysis.ProfitConsistency = val
		case "risk_adjusted_return":
			analysis.RiskAdjustedReturn = val
		case "key_observations":
			analysis.KeyObservations = splitSemicolon(val)
		case "improvement_suggestions":
			analysis.ImprovementSuggestions = splitSemicolon(val)
		case "overfitting_risk":
			analysis.OverfittingRisk = val
		case "recommended_action":
			analysis.RecommendedAction = val
		case "detailed_analysis":
			analysis.DetailedAnalysis = val
		}
	}

	return analysis
}

// splitSemicolon splits a semicolon-separated string into trimmed fields.
func splitSemicolon(s string) []string {
	parts := strings.Split(s, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
