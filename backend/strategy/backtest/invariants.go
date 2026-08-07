package backtest

import (
	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/sdk"
)

const severityFatal = "致命"

// CheckVolumeInvariant verifies that every trade has a strictly positive Volume.
// Returns a BlindSpot if any trade has Volume <= 0, nil otherwise.
// When there are no trades, the invariant is vacuously true (returns nil).
func CheckVolumeInvariant(trades []Trade) *antv1.BlindSpot {
	for _, t := range trades {
		if !t.Volume.GreaterThan(decimal.Zero) {
			return &antv1.BlindSpot{
				Id:          "zero_volume_trade",
				Category:    "invariant",
				Severity:    severityFatal,
				Description: "存在手数<=0的交易，回测结果不可信",
			}
		}
	}
	return nil
}

// CheckPricePositive verifies that every trade has EntryPrice > 0 and ExitPrice > 0.
// Returns a BlindSpot if any price is <= 0, nil otherwise.
// When there are no trades, the invariant is vacuously true (returns nil).
func CheckPricePositive(result *Result) *antv1.BlindSpot {
	for _, t := range result.Trades {
		if !t.EntryPrice.GreaterThan(decimal.Zero) || !t.ExitPrice.GreaterThan(decimal.Zero) {
			return &antv1.BlindSpot{
				Id:          "non_positive_price",
				Category:    "invariant",
				Severity:    severityFatal,
				Description: "存在开仓价或平仓价<=0的交易，回测结果不可信",
			}
		}
	}
	return nil
}

// CheckSideValid verifies that every trade has Side == sdk.SideBuy or sdk.SideSell.
// Returns a BlindSpot if any trade has an invalid side, nil otherwise.
// When there are no trades, the invariant is vacuously true (returns nil).
func CheckSideValid(result *Result) *antv1.BlindSpot {
	for _, t := range result.Trades {
		if t.Side != sdk.SideBuy && t.Side != sdk.SideSell {
			return &antv1.BlindSpot{
				Id:          "invalid_side",
				Category:    "invariant",
				Severity:    severityFatal,
				Description: "存在交易方向非法的交易（非 BUY/SELL），回测结果不可信",
			}
		}
	}
	return nil
}

// CheckTimeOrder verifies that every trade has EntryTime <= ExitTime.
// Returns a BlindSpot if any trade has EntryTime after ExitTime, nil otherwise.
// EntryTime == ExitTime is valid (same-bar entry and exit).
// When there are no trades, the invariant is vacuously true (returns nil).
func CheckTimeOrder(result *Result) *antv1.BlindSpot {
	for _, t := range result.Trades {
		if t.EntryTime.After(t.ExitTime) {
			return &antv1.BlindSpot{
				Id:          "time_order_violation",
				Category:    "invariant",
				Severity:    severityFatal,
				Description: "存在开仓时间晚于平仓时间的交易，回测结果不可信",
			}
		}
	}
	return nil
}

// CheckCapitalConservation verifies the capital conservation identity:
//
//	|FinalBalance − (本金 + ΣProfit − ΣCommission − ΣSwap)| < 容差
//
// FinalBalance is the realized balance (excludes unrealized PnL), so the invariant
// holds regardless of open positions at backtest end.
// Returns a BlindSpot if the identity is violated, nil otherwise.
// 容差 = max(0.01, 1e-4 × 本金) — covers floating-point accumulation and minor swap/commission model discrepancies.
func CheckCapitalConservation(result *Result) *antv1.BlindSpot {
	finalBalance := result.FinalBalance
	initialCapital := result.Config.InitialCapital

	var sumProfit, sumCommission, sumSwap decimal.Decimal
	for _, t := range result.Trades {
		sumProfit = sumProfit.Add(t.Profit)
		sumCommission = sumCommission.Add(t.Commission)
		sumSwap = sumSwap.Add(t.Swap)
	}

	expected := initialCapital.Add(sumProfit).Sub(sumCommission).Sub(sumSwap)
	diff := finalBalance.Sub(expected).Abs()

	tolerance := decimal.New(1, -2) // 0.01
	if scaled := initialCapital.Mul(decimal.New(1, -4)); scaled.GreaterThan(tolerance) {
		tolerance = scaled
	}

	if diff.GreaterThanOrEqual(tolerance) {
		return &antv1.BlindSpot{
			Id:          "capital_not_conserved",
			Category:    "invariant",
			Severity:    severityFatal,
			Description: "资金不守恒：期末净值与 本金+Σ盈亏−Σ手续费−Σswap 对不上，回测结果不可信",
		}
	}
	return nil
}

// ValidateInvariants runs all ADR-0028 Defense Line B invariant checks on a backtest result.
// Returns true if all checks pass (result is reliable), false if any check fails.
// Failed checks are returned as BlindSpots for feedback to the LLM.
func ValidateInvariants(result *Result) (bool, []*antv1.BlindSpot) {
	var blinds []*antv1.BlindSpot
	if bs := CheckVolumeInvariant(result.Trades); bs != nil {
		blinds = append(blinds, bs)
	}
	if bs := CheckCapitalConservation(result); bs != nil {
		blinds = append(blinds, bs)
	}
	if bs := CheckPricePositive(result); bs != nil {
		blinds = append(blinds, bs)
	}
	if bs := CheckSideValid(result); bs != nil {
		blinds = append(blinds, bs)
	}
	if bs := CheckTimeOrder(result); bs != nil {
		blinds = append(blinds, bs)
	}
	return len(blinds) == 0, blinds
}
