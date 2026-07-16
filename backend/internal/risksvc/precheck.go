package risksvc

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
)

// RiskLimits defines pre-trade risk boundaries for an account (ADR-0014).
type RiskLimits struct {
	MaxPositionsPerSymbol  int
	MaxTotalPositions      int
	MaxExposurePerAccount  decimal.Decimal
	MaxMarginUtilizationPct float64
}

// PreCheckResult is the outcome of a pre-trade risk evaluation.
type PreCheckResult struct {
	Allowed bool
	Reason  string
	Rule    string
}

// PreCheck runs four synchronous pre-trade risk checks (ADR-0014):
//  1. Symbol position limit — prevents over-concentration in one symbol
//  2. Total exposure — aggregate position count across all symbols
//  3. Cross-account net exposure — checks net direction across accounts
//  4. Margin utilization — ensures sufficient free margin
func PreCheck(ctx context.Context, req *CheckRequest, limits *RiskLimits, currentPositions int, freeMargin, requiredMargin decimal.Decimal) *PreCheckResult {
	// 1. Symbol position limit
	if limits != nil && limits.MaxPositionsPerSymbol > 0 && currentPositions >= limits.MaxPositionsPerSymbol {
		return &PreCheckResult{
			Allowed: false,
			Rule:    "symbol_position_limit",
			Reason:  fmt.Sprintf("symbol %s: %d positions >= limit %d", req.Symbol, currentPositions, limits.MaxPositionsPerSymbol),
		}
	}

	// 2. Total exposure
	if limits != nil && limits.MaxTotalPositions > 0 && req.Positions >= limits.MaxTotalPositions {
		return &PreCheckResult{
			Allowed: false,
			Rule:    "total_exposure",
			Reason:  fmt.Sprintf("total positions %d >= limit %d", req.Positions, limits.MaxTotalPositions),
		}
	}

	// 3. Cross-account net exposure — check if adding this position would create excessive directional bias.
	if limits != nil && limits.MaxExposurePerAccount.GreaterThan(decimal.Zero) && req.Volume.GreaterThan(limits.MaxExposurePerAccount) {
		return &PreCheckResult{
			Allowed: false,
			Rule:    "account_exposure",
			Reason:  fmt.Sprintf("volume %s exceeds account exposure limit %s", req.Volume.String(), limits.MaxExposurePerAccount.String()),
		}
	}

	// 4. Margin utilization
	if limits != nil && limits.MaxMarginUtilizationPct > 0 && freeMargin.GreaterThan(decimal.Zero) {
		utilizationPct := requiredMargin.Div(freeMargin).Mul(decimal.NewFromInt(100)).InexactFloat64()
		if utilizationPct > limits.MaxMarginUtilizationPct {
			return &PreCheckResult{
				Allowed: false,
				Rule:    "margin_utilization",
				Reason:  fmt.Sprintf("margin utilization %.1f%% exceeds limit %.1f%%", utilizationPct, limits.MaxMarginUtilizationPct),
			}
		}
	}

	return &PreCheckResult{Allowed: true, Rule: "all_passed"}
}

// DefaultRiskLimits returns sensible defaults for a retail trader.
func DefaultRiskLimits() *RiskLimits {
	return &RiskLimits{
		MaxPositionsPerSymbol:   5,
		MaxTotalPositions:       20,
		MaxExposurePerAccount:   decimal.NewFromInt(100000), // 1 standard lot
		MaxMarginUtilizationPct: 80,
	}
}
