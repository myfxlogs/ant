package qualitygate

import (
	"context"
	"fmt"
)

// RiskGate scores a strategy's risk parameters against safe thresholds.
type RiskGate struct {
	MaxRiskPerTrade  float64 // max risk per trade as % of equity (default 0.05 = 5%)
	MaxDrawdownLimit float64 // max allowed drawdown (default 0.30 = 30%)
	MaxLeverage      float64 // max allowed leverage (default 100)
	MinStopLoss      float64 // minimum stop loss required as % (0 = not required)
	MaxDailyTrades   int     // max trades per day (0 = unlimited)
	MaxPositions     int     // max concurrent positions (0 = unlimited)
}

func (g *RiskGate) Name() string { return "risk" }

func (g *RiskGate) Evaluate(_ context.Context, info *StrategyInfo) *GateResult {
	if g.MaxRiskPerTrade <= 0 {
		g.MaxRiskPerTrade = 0.05
	}
	if g.MaxDrawdownLimit <= 0 {
		g.MaxDrawdownLimit = 0.30
	}
	if g.MaxLeverage <= 0 {
		g.MaxLeverage = 100
	}

	result := &GateResult{
		Gate:     g.Name(),
		Passed:   true,
		Score:    1.0,
		Details:  map[string]interface{}{},
	}

	sched := info.Schedule
	if sched == nil {
		sched = DefaultSchedule()
	}

	warnings := make([]string, 0)
	criticalCount := 0

	// Check risk per trade
	if sched.RiskPerTradePct > g.MaxRiskPerTrade {
		criticalCount++
		result.Score -= 0.3
		warnings = append(warnings, fmt.Sprintf("risk_per_trade %.1f%% exceeds max %.1f%%",
			sched.RiskPerTradePct*100, g.MaxRiskPerTrade*100))
	}

	// Check max drawdown
	if sched.MaxDrawdownPct > g.MaxDrawdownLimit {
		criticalCount++
		result.Score -= 0.2
		warnings = append(warnings, fmt.Sprintf("max_drawdown %.1f%% exceeds limit %.1f%%",
			sched.MaxDrawdownPct*100, g.MaxDrawdownLimit*100))
	}

	// Check leverage
	if sched.LeverageAllowed > g.MaxLeverage {
		result.Score -= 0.2
		warnings = append(warnings, fmt.Sprintf("leverage %.0f exceeds max %.0f",
			sched.LeverageAllowed, g.MaxLeverage))
	}

	// Check stop loss
	if g.MinStopLoss > 0 && sched.StopLossPct < g.MinStopLoss {
		warnings = append(warnings, fmt.Sprintf("stop_loss %.1f%% below required minimum %.1f%%",
			sched.StopLossPct*100, g.MinStopLoss*100))
		result.Score -= 0.15
	}

	// Check daily trade limit
	if g.MaxDailyTrades > 0 && sched.DailyTradeLimit > g.MaxDailyTrades {
		warnings = append(warnings, fmt.Sprintf("daily_trade_limit %d exceeds max %d",
			sched.DailyTradeLimit, g.MaxDailyTrades))
		result.Score -= 0.1
	}

	// Check positions
	if g.MaxPositions > 0 && sched.MaxPositions > g.MaxPositions {
		warnings = append(warnings, fmt.Sprintf("max_positions %d exceeds limit %d",
			sched.MaxPositions, g.MaxPositions))
		result.Score -= 0.1
	}

	// Score floor
	if result.Score < 0 {
		result.Score = 0
	}

	if criticalCount > 0 {
		result.Passed = false
		result.Severity = SeverityCritical
		result.Reason = fmt.Sprintf("risk check failed: %d critical issues", criticalCount)
	} else if len(warnings) > 0 {
		result.Severity = SeverityWarning
	}

	result.Warnings = warnings
	result.Details["critical_count"] = criticalCount
	result.Details["warning_count"] = len(warnings) - criticalCount
	result.Details["risk_per_trade"] = sched.RiskPerTradePct
	result.Details["max_drawdown"] = sched.MaxDrawdownPct

	return result
}
