package qualitygate

import (
	"context"
	"fmt"
)

// BacktestGate validates that backtest metrics meet minimum quality thresholds.
type BacktestGate struct {
	MinSharpeRatio     float64 // minimum Sharpe ratio (default 0.5)
	MinWinRate         float64 // minimum win rate (default 0.35)
	MinProfitFactor    float64 // minimum profit factor (default 1.1)
	MaxDrawdownAllowed float64 // max allowed drawdown (default 0.30)
	MinTotalTrades     int     // minimum number of trades (default 30)
	MinTotalReturn     float64 // minimum total return (default 0.0)
}

func (g *BacktestGate) Name() string { return "backtest" }

func (g *BacktestGate) Evaluate(_ context.Context, info *StrategyInfo) *GateResult {
	if g.MinSharpeRatio <= 0 {
		g.MinSharpeRatio = 0.5
	}
	if g.MinWinRate <= 0 {
		g.MinWinRate = 0.35
	}
	if g.MinProfitFactor <= 0 {
		g.MinProfitFactor = 1.1
	}
	if g.MaxDrawdownAllowed <= 0 {
		g.MaxDrawdownAllowed = 0.30
	}
	if g.MinTotalTrades <= 0 {
		g.MinTotalTrades = 30
	}

	result := &GateResult{
		Gate:    g.Name(),
		Passed:  true,
		Score:   1.0,
		Details: map[string]interface{}{},
	}

	bt := info.Backtest
	if bt == nil {
		result.Passed = false
		result.Severity = SeverityWarning
		result.Score = 0
		result.Reason = "no backtest data available"
		return result
	}

	warnings := make([]string, 0)
	errors := make([]string, 0)

	check := func(metric string, value, threshold float64, penalty float64, isError bool, desc string) {
		if value < threshold {
			if isError {
				errors = append(errors, desc)
			} else {
				warnings = append(warnings, desc)
			}
			result.Score -= penalty
		}
	}

	check("sharpe", bt.SharpeRatio, g.MinSharpeRatio, 0.25, true,
		fmt.Sprintf("Sharpe %.2f < minimum %.2f", bt.SharpeRatio, g.MinSharpeRatio))

	check("win_rate", bt.WinRate, g.MinWinRate, 0.15, false,
		fmt.Sprintf("win rate %.2f < minimum %.2f", bt.WinRate, g.MinWinRate))

	check("profit_factor", bt.ProfitFactor, g.MinProfitFactor, 0.20, true,
		fmt.Sprintf("profit factor %.2f < minimum %.2f", bt.ProfitFactor, g.MinProfitFactor))

	check("drawdown", 1.0-bt.MaxDrawdown, 1.0-g.MaxDrawdownAllowed, 0.20, true,
		fmt.Sprintf("max drawdown %.2f exceeds limit %.2f", bt.MaxDrawdown, g.MaxDrawdownAllowed))

	check("total_return", bt.TotalReturn, g.MinTotalReturn, 0.10, false,
		fmt.Sprintf("total return %.4f < minimum %.4f", bt.TotalReturn, g.MinTotalReturn))

	if bt.TotalTrades < g.MinTotalTrades {
		warnings = append(warnings, fmt.Sprintf("total trades %d < minimum %d (low confidence)",
			bt.TotalTrades, g.MinTotalTrades))
		result.Score -= 0.10
	}

	if result.Score < 0 {
		result.Score = 0
	}

	result.Warnings = append(warnings, errors...)
	if len(errors) > 0 {
		result.Passed = false
		result.Severity = SeverityError
		result.Reason = fmt.Sprintf("backtest check failed: %d errors", len(errors))
	} else if len(warnings) > 0 {
		result.Severity = SeverityWarning
	}

	result.Details["sharpe"] = bt.SharpeRatio
	result.Details["win_rate"] = bt.WinRate
	result.Details["profit_factor"] = bt.ProfitFactor
	result.Details["max_drawdown"] = bt.MaxDrawdown
	result.Details["total_trades"] = bt.TotalTrades

	return result
}
