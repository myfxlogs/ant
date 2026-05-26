package qualitygate

import (
	"context"
	"fmt"
)

// ReliabilityGate computes a combined reliability score from backtest characteristics.
// It looks for signs of overfitting, data insufficiency, and risk of failure in live.
type ReliabilityGate struct {
	MinTradesForReliability int     // minimum trades for statistical significance (default 100)
	MinWinLossRatio         float64 // minimum avg_win/avg_loss ratio (default 1.5)
	MaxReturnDrawdownRatio  float64 // max allowed return/drawdown ratio (default 10 = suspicious if higher)
	RequirePositiveReturn   bool    // require total return > 0
}

func (g *ReliabilityGate) Name() string { return "reliability" }

func (g *ReliabilityGate) Evaluate(_ context.Context, info *StrategyInfo) *GateResult {
	if g.MinTradesForReliability <= 0 {
		g.MinTradesForReliability = 100
	}
	if g.MinWinLossRatio <= 0 {
		g.MinWinLossRatio = 1.5
	}
	if g.MaxReturnDrawdownRatio <= 0 {
		g.MaxReturnDrawdownRatio = 10.0
	}

	result := &GateResult{
		Gate:    g.Name(),
		Passed:  true,
		Score:   1.0,
		Details: map[string]interface{}{},
	}

	bt := info.Backtest
	if bt == nil || bt.TotalTrades == 0 {
		result.Passed = false
		result.Severity = SeverityWarning
		result.Score = 0
		result.Reason = "insufficient backtest data for reliability assessment"
		return result
	}

	warnings := make([]string, 0)

	// Trade count significance
	if bt.TotalTrades < g.MinTradesForReliability {
		warnings = append(warnings, fmt.Sprintf("only %d trades (need ≥%d for statistical significance)",
			bt.TotalTrades, g.MinTradesForReliability))
		result.Score -= 0.25
	}

	// Win/loss ratio
	winLossRatio := 0.0
	if bt.WinningTrades > 0 && bt.LosingTrades > 0 && bt.AverageLoss > 0 {
		winLossRatio = bt.AverageProfit / bt.AverageLoss
		if winLossRatio < g.MinWinLossRatio {
			warnings = append(warnings, fmt.Sprintf("win/loss ratio %.2f < minimum %.2f",
				winLossRatio, g.MinWinLossRatio))
			result.Score -= 0.20
		}
	}

	// Return vs drawdown: check for overfitting signals
	if bt.MaxDrawdown > 0 && bt.TotalReturn > 0 {
		retToDD := bt.TotalReturn / bt.MaxDrawdown
		if retToDD > g.MaxReturnDrawdownRatio {
			warnings = append(warnings, fmt.Sprintf("return/drawdown ratio %.2f > %.0f (possible overfitting)",
				retToDD, g.MaxReturnDrawdownRatio))
			result.Score -= 0.25
		}
		result.Details["return_drawdown_ratio"] = retToDD
	}

	// Profit factor stability
	if bt.ProfitFactor > 5.0 {
		warnings = append(warnings, fmt.Sprintf("very high profit factor %.2f (may indicate overfitting)",
			bt.ProfitFactor))
		result.Score -= 0.10
	}

	// Positive return
	if g.RequirePositiveReturn && bt.TotalReturn <= 0 {
		result.Passed = false
		result.Severity = SeverityError
		result.Score -= 0.30
		result.Reason = "strategy has non-positive total return"
	}

	if result.Score < 0 {
		result.Score = 0
	}

	result.Warnings = warnings
	if !result.Passed {
		result.Severity = SeverityError
	}
	if len(warnings) > 0 && result.Passed {
		result.Severity = SeverityWarning
	}

	result.Details["win_loss_ratio"] = winLossRatio
	result.Details["total_trades"] = bt.TotalTrades
	result.Details["is_reliable"] = result.Passed && len(warnings) == 0

	return result
}
