// Package ai provides Paper Gate enforcement (M10-BASE-E4).
//
// 14-day mandatory paper trading before strategy can go live.
// Conditions:
//   - paper_return >= 0.5 * backtest_net_return (regime hasn't changed)
//   - Net P&L > 0 (strategy is profitable in current market)
//   - Minimum paper trading days met

package ai

import (
	"fmt"
	"math"
)

// PaperGateConfig holds parameters for paper trading validation.
type PaperGateConfig struct {
	MinPaperDays   int     // minimum paper trading days (default 14)
	MinReturnRatio float64 // paper_return / backtest_net_return minimum (default 0.5)
	MinPaperTrades int     // minimum paper trades for significance (default 5)
}

// DefaultPaperGateConfig returns standard paper gate parameters.
func DefaultPaperGateConfig() PaperGateConfig {
	return PaperGateConfig{
		MinPaperDays:   14,
		MinReturnRatio: 0.5,
		MinPaperTrades: 5,
	}
}

// PaperGateMetrics holds the paper trading performance metrics.
type PaperGateMetrics struct {
	PaperDays          int     `json:"paper_days"`
	BacktestNetReturn  float64 `json:"backtest_net_return"`
	BacktestGrossReturn float64 `json:"backtest_gross_return"`
	PaperNetReturn     float64 `json:"paper_net_return"`
	PaperNetPnL        float64 `json:"paper_net_pnl"`
	PaperTradeCount    int     `json:"paper_trade_count"`
}

// PaperGateResult is the outcome of the paper trading gate.
type PaperGateResult struct {
	Passed  bool             `json:"passed"`
	Metrics PaperGateMetrics `json:"metrics"`
	Reason  string           `json:"reason,omitempty"`
}

// validatePaperMetrics checks for NaN/Inf in critical paper metric fields.
func validatePaperMetrics(m PaperGateMetrics) bool {
	if math.IsNaN(m.PaperNetPnL) || math.IsInf(m.PaperNetPnL, 0) {
		return false
	}
	if math.IsNaN(m.PaperNetReturn) || math.IsInf(m.PaperNetReturn, 0) {
		return false
	}
	return true
}

// checkPaperPerformance validates paper trading performance against thresholds.
func checkPaperPerformance(metrics PaperGateMetrics, cfg PaperGateConfig) (bool, string) {
	// Net P&L must be positive.
	if metrics.PaperNetPnL <= 0 {
		return false, fmt.Sprintf("paper Net P&L %.2f <= 0 (must be profitable)", metrics.PaperNetPnL)
	}

	// Paper return ratio vs backtest.
	if metrics.BacktestNetReturn > 0 {
		if metrics.PaperNetReturn < 0 {
			return false, fmt.Sprintf(
				"paper return negative (%.4f) while backtest return positive — regime mismatch",
				metrics.PaperNetReturn,
			)
		}
		returnRatio := metrics.PaperNetReturn / metrics.BacktestNetReturn
		if returnRatio < cfg.MinReturnRatio {
			return false, fmt.Sprintf(
				"paper return %.4f below %.0f%% threshold of backtest return %.4f",
				metrics.PaperNetReturn, cfg.MinReturnRatio*100, metrics.BacktestNetReturn,
			)
		}
	}

	// Minimum trade count in paper.
	if metrics.PaperTradeCount < cfg.MinPaperTrades {
		return false, fmt.Sprintf(
			"paper trade count %d insufficient (min %d)",
			metrics.PaperTradeCount, cfg.MinPaperTrades,
		)
	}
	return true, ""
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

	// Reject NaN/Inf in critical metric fields.
	if !validatePaperMetrics(metrics) {
		result.Passed = false
		result.Reason = "paper metrics contain invalid values (NaN/Inf)"
		return result
	}

	// Performance checks.
	if passed, reason := checkPaperPerformance(metrics, cfg); !passed {
		result.Passed = false
		result.Reason = reason
		return result
	}

	return result
}
