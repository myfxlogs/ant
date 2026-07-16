// Package risksvc provides the VolTargetSizer (M10-BASE-C2).
//
// VolTargetSizer computes position size to achieve a target volatility:
//
//	lot = target_risk / (ATR × contract_size × √holding_period)
//
// where:
//   - target_risk = risk_budget_pct × equity (e.g. 1% of 100k = $1000)
//   - ATR = average true range (in quote currency)
//   - contract_size = lot multiplier (e.g. 100000 for standard forex)
//   - holding_period = expected holding time in days
//
// Example: BTCUSD at 50000 with ATR=2000, equity=100k, risk=1%, holding=5d
//
//	lot = 1000 / (2000 × 1 × √5) = 1000 / 4472 = 0.22 lots
//
// Example: EURUSD at 1.0850 with ATR=0.0035, equity=100k, risk=1%, holding=5d
//
//	lot = 1000 / (0.0035 × 100000 × √5) = 1000 / 783 = 1.28 lots
//
// BTCUSD lot should be ~5-10× smaller than EURUSD for the same risk budget.

package risksvc

import (
	"context"
	"math"

	"github.com/shopspring/decimal"
)

// VolTargetSizer sizes positions based on volatility targeting.
// Aligned with QuantConnect Lean MeanVarianceOptimization and freqtrade edge positioning.
type VolTargetSizer struct {
	// RiskBudgetPct is the fraction of equity to risk per trade (e.g. 0.01 = 1%).
	RiskBudgetPct float64

	// MaxLots caps the position size.
	MaxLots decimal.Decimal

	// MinLots is the minimum lot size (sizer returns 0 if below this).
	MinLots decimal.Decimal
}

func (s *VolTargetSizer) Name() string { return "vol_target" }

func (s *VolTargetSizer) Size(_ context.Context, req *SizerRequest) (*SizerResult, error) {
	riskBudgetPct := s.RiskBudgetPct
	if riskBudgetPct <= 0 {
		riskBudgetPct = 0.01 // default 1% risk per trade
	}
	maxLots := s.MaxLots
	if !maxLots.GreaterThan(decimal.Zero) {
		maxLots = decimal.NewFromInt(100) // generous cap
	}

	// Compute ATR in account currency terms: ATR × contract_size.
	atrValue := req.ATR
	if atrValue.LessThanOrEqual(decimal.Zero) {
		// Fallback: use price × annual_vol / √252 as daily vol proxy, then as ATR proxy.
		if req.AnnualVol > 0 {
			atrValue = req.Price.Mul(decimal.NewFromFloat(req.AnnualVol / math.Sqrt(252)))
		} else {
			atrValue = req.Price.Mul(decimal.NewFromFloat(0.01)) // 1% daily vol default
		}
	}

	contractSize := req.ContractSize
	if contractSize.LessThanOrEqual(decimal.Zero) {
		contractSize = decimal.NewFromInt(1) // spot-like or unit-less; use raw ATR
	}

	// Target risk in account currency.
	targetRisk := req.Equity.Mul(decimal.NewFromFloat(riskBudgetPct))
	if targetRisk.LessThanOrEqual(decimal.Zero) {
		return &SizerResult{Lots: decimal.Zero, RiskUsed: 0, Method: s.Name()}, nil
	}

	// Position risk per lot: ATR × contract_size × √holding_days.
	holdingDays := req.HoldingDays
	if holdingDays <= 0 {
		holdingDays = 5 // default 5-day holding period
	}
	sqrtHolding := math.Sqrt(holdingDays)
	riskPerLot := atrValue.Mul(contractSize).Mul(decimal.NewFromFloat(sqrtHolding))
	if riskPerLot.LessThanOrEqual(decimal.Zero) {
		return &SizerResult{Lots: decimal.Zero, RiskUsed: 0, Method: s.Name()}, nil
	}

	lots := targetRisk.Div(riskPerLot)

	// Clamp to limits.
	if s.MinLots.GreaterThan(decimal.Zero) && lots.LessThan(s.MinLots) {
		lots = decimal.Zero
	}
	if lots.GreaterThan(maxLots) {
		lots = maxLots
	}

	if req.Equity.LessThanOrEqual(decimal.Zero) {
		return &SizerResult{Lots: decimal.Zero, RiskUsed: 0, Method: s.Name()}, nil
	}
	riskUsed := lots.Mul(riskPerLot).Div(req.Equity).InexactFloat64()

	return &SizerResult{
		Lots:     lots,
		RiskUsed: riskUsed,
		Method:   s.Name(),
	}, nil
}
