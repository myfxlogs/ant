package agent

import (
	_ "embed"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/service/systemai"
)

//go:embed prompts/analysis_user.prompt
var analysisUserPromptTmpl string

// Interpreter generates backtest analysis via LLM (injection point [4]).
// Input: backtest result + strategy profile → LLM → KEY: "value" lines → BacktestAnalysis proto.
type Interpreter struct {
	aiSvc *systemai.Service
	cache *LLCache
}

// NewInterpreter creates a backtest analysis generator.
func NewInterpreter(aiSvc *systemai.Service, cache *LLCache) *Interpreter {
	return &Interpreter{aiSvc: aiSvc, cache: cache}
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

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := i.aiSvc.ChatCompletion(llmCtx, uid, []systemai.ChatMessage{
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
	data := promptData{}
	if profile != nil {
		var sb strings.Builder
		writeProfileToPrompt(&sb, profile, "## Strategy Profile\n")
		data.ProfileBlock = sb.String()
	}
	var btSB strings.Builder
	fmt.Fprintf(&btSB, "Success: %v\n", result.Success)
	if result.Error != "" {
		fmt.Fprintf(&btSB, "Error: %s\n", result.Error)
	}
	totalReturn, _ := strconv.ParseFloat(result.TotalReturn, 64)
	annualReturn, _ := strconv.ParseFloat(result.AnnualReturn, 64)
	maxDrawdown, _ := strconv.ParseFloat(result.MaxDrawdown, 64)
	sharpeRatio, _ := strconv.ParseFloat(result.SharpeRatio, 64)
	winRate, _ := strconv.ParseFloat(result.WinRate, 64)
	profitFactor, _ := strconv.ParseFloat(result.ProfitFactor, 64)
	fmt.Fprintf(&btSB, "Total Return: %.2f%%\n", totalReturn*100)
	fmt.Fprintf(&btSB, "Annual Return: %.2f%%\n", annualReturn*100)
	fmt.Fprintf(&btSB, "Max Drawdown: %.2f%%\n", maxDrawdown*100)
	fmt.Fprintf(&btSB, "Sharpe Ratio: %.2f\n", sharpeRatio)
	fmt.Fprintf(&btSB, "Win Rate: %.1f%%\n", winRate*100)
	fmt.Fprintf(&btSB, "Profit Factor: %.2f\n", profitFactor)
	fmt.Fprintf(&btSB, "Total Trades: %d (Win: %d, Loss: %d)\n",
		result.TotalTrades, result.WinningTrades, result.LosingTrades)
	fmt.Fprintf(&btSB, "Total PnL: %s\n", result.TotalPnlAbsolute)
	if len(result.EquityCurve) > 0 {
		fmt.Fprintf(&btSB, "Equity Points: %d\n", len(result.EquityCurve))
		fmt.Fprintf(&btSB, "Start Equity: %s\n", result.EquityCurve[0])
		fmt.Fprintf(&btSB, "End Equity: %s\n", result.EquityCurve[len(result.EquityCurve)-1])
	}
	data.BacktestBlock = btSB.String()
	if len(result.Trades) > 0 && len(result.Trades) <= 50 {
		var trSB strings.Builder
		trSB.WriteString("\n## Trades\n")
		for _, t := range result.Trades {
			fmt.Fprintf(&trSB, "  #%d %s vol=%s pnl=%s reason=%s\n",
				t.Ticket, t.Side, t.Volume, t.Pnl, t.Reason)
		}
		data.TradesBlock = trSB.String()
	} else if len(result.Trades) > 50 {
		var trSB strings.Builder
		fmt.Fprintf(&trSB, "\n## Trades (showing first 10 of %d)\n", len(result.Trades))
		for _, t := range result.Trades[:10] {
			fmt.Fprintf(&trSB, "  #%d %s vol=%s pnl=%s\n", t.Ticket, t.Side, t.Volume, t.Pnl)
		}
		data.TradesBlock = trSB.String()
	}
	userPrompt, err := renderPrompt("analysis_user", analysisUserPromptTmpl, data)
	if err != nil {
		return fallbackAnalysisUserPrompt(result, profile)
	}
	return userPrompt
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
			analysis.KeyObservations = splitTrimmed(val, ";")
		case "improvement_suggestions":
			analysis.ImprovementSuggestions = splitTrimmed(val, ";")
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

func fallbackAnalysisUserPrompt(result *antv1.AgentBacktestResult, profile *antv1.StrategyProfile) string {
	var sb strings.Builder
	writeProfileToPrompt(&sb, profile, "## Strategy Profile\n")
	if profile != nil {
		sb.WriteString("\n")
	}
	sb.WriteString("## Backtest Results\n")
	fmt.Fprintf(&sb, "Success: %v\n", result.Success)
	if result.Error != "" {
		fmt.Fprintf(&sb, "Error: %s\n", result.Error)
	}
	tr2, _ := strconv.ParseFloat(result.TotalReturn, 64)
	md2, _ := strconv.ParseFloat(result.MaxDrawdown, 64)
	sr2, _ := strconv.ParseFloat(result.SharpeRatio, 64)
	wr2, _ := strconv.ParseFloat(result.WinRate, 64)
	fmt.Fprintf(&sb, "Total Return: %.2f%%\n", tr2*100)
	fmt.Fprintf(&sb, "Max Drawdown: %.2f%%\n", md2*100)
	fmt.Fprintf(&sb, "Sharpe Ratio: %.2f\n", sr2)
	fmt.Fprintf(&sb, "Win Rate: %.1f%%\n", wr2*100)
	fmt.Fprintf(&sb, "Total Trades: %d\n", result.TotalTrades)
	sb.WriteString("\nProduce the backtest analysis now.\n")
	return sb.String()
}
