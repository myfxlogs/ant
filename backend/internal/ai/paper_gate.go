// Package ai provides Paper Gate enforcement (M10-BASE-E4).
//
// 14-day mandatory paper trading before strategy can go live.
// Conditions:
//   - paper_return >= 0.5 * backtest_net_return (regime hasn't changed)
//   - Net P&L > 0 (strategy is profitable in current market)
//   - Minimum paper trading days met

package ai

import "fmt"

// PaperGateConfig holds parameters for paper trading validation.
type PaperGateConfig struct {
	MinPaperDays         int     // minimum paper trading days (default 14)
	MinReturnRatio       float64 // paper_return / backtest_net_return minimum (default 0.5)
}

// DefaultPaperGateConfig returns standard paper gate parameters.
func DefaultPaperGateConfig() PaperGateConfig {
	return PaperGateConfig{
		MinPaperDays:   14,
		MinReturnRatio: 0.5,
	}
}

// PaperGateMetrics holds the paper trading performance metrics.
type PaperGateMetrics struct {
	PaperDays         int     `json:"paper_days"`
	BacktestNetReturn float64 `json:"backtest_net_return"`
	BacktestGrossReturn float64 `json:"backtest_gross_return"`
	PaperNetReturn    float64 `json:"paper_net_return"`
	PaperNetPnL       float64 `json:"paper_net_pnl"`
	PaperTradeCount   int     `json:"paper_trade_count"`
}

// PaperGateResult is the outcome of the paper trading gate.
type PaperGateResult struct {
	Passed     bool              `json:"passed"`
	Metrics    PaperGateMetrics  `json:"metrics"`
	Reason     string            `json:"reason,omitempty"`
}

// PaperGate evaluates paper trading performance against backtest expectations.
func PaperGate(metrics PaperGateMetrics, cfg PaperGateConfig) PaperGateResult {
	result := PaperGateResult{Metrics: metrics, Passed: true}

	// Check minimum paper trading days.
	if metrics.PaperDays < cfg.MinPaperDays {
		result.Passed = false
		result.Reason = fmt.Sprintf("paper days %d < minimum %d", metrics.PaperDays, cfg.MinPaperDays)
		return result
	}

	// Check Net P&L > 0.
	if metrics.PaperNetPnL <= 0 {
		result.Passed = false
		result.Reason = fmt.Sprintf("paper Net P&L %.2f <= 0 (must be profitable)", metrics.PaperNetPnL)
		return result
	}

	// Check paper return ratio vs backtest.
	if metrics.BacktestNetReturn > 0 {
		returnRatio := metrics.PaperNetReturn / metrics.BacktestNetReturn
		if returnRatio < cfg.MinReturnRatio {
			result.Passed = false
			result.Reason = fmt.Sprintf(
				"paper return %.4f is < %.0f%% of backtest return %.4f (regime fail)",
				metrics.PaperNetReturn, cfg.MinReturnRatio*100, metrics.BacktestNetReturn,
			)
			return result
		}
	}

	// Check minimum trade count in paper.
	if metrics.PaperTradeCount < 5 {
		result.Passed = false
		result.Reason = fmt.Sprintf("paper trade count %d insufficient for evaluation", metrics.PaperTradeCount)
		return result
	}

	return result
}
