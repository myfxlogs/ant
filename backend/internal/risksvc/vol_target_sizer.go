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
)

// VolTargetSizer sizes positions based on volatility targeting.
// Aligned with QuantConnect Lean MeanVarianceOptimization and freqtrade edge positioning.
type VolTargetSizer struct {
	// RiskBudgetPct is the fraction of equity to risk per trade (e.g. 0.01 = 1%).
	RiskBudgetPct float64

	// MaxLots caps the position size.
	MaxLots float64

	// MinLots is the minimum lot size (sizer returns 0 if below this).
	MinLots float64
}

func (s *VolTargetSizer) Name() string { return "vol_target" }

func (s *VolTargetSizer) Size(_ context.Context, req *SizerRequest) (*SizerResult, error) {
	if s.RiskBudgetPct <= 0 {
		s.RiskBudgetPct = 0.01 // default 1% risk per trade
	}
	if s.MaxLots <= 0 {
		s.MaxLots = 100 // generous cap
	}

	// Compute ATR in account currency terms: ATR × contract_size.
	atrValue := req.ATR
	if atrValue <= 0 {
		// Fallback: use price × annual_vol / √252 as daily vol proxy, then as ATR proxy.
		if req.AnnualVol > 0 {
			atrValue = req.Price * req.AnnualVol / math.Sqrt(252)
		} else {
			atrValue = req.Price * 0.01 // 1% daily vol default
		}
	}

	contractSize := req.ContractSize
	if contractSize <= 0 {
		contractSize = 1 // spot-like or unit-less; use raw ATR
	}

	// Target risk in account currency.
	targetRisk := req.Equity * s.RiskBudgetPct
	if targetRisk <= 0 {
		return &SizerResult{Lots: 0, RiskUsed: 0, Method: s.Name()}, nil
	}

	// Position risk per lot: ATR × contract_size × √holding_days.
	holdingDays := req.HoldingDays
	if holdingDays <= 0 {
		holdingDays = 5 // default 5-day holding period
	}
	riskPerLot := atrValue * contractSize * math.Sqrt(holdingDays)
	if riskPerLot <= 0 {
		return &SizerResult{Lots: 0, RiskUsed: 0, Method: s.Name()}, nil
	}

	lots := targetRisk / riskPerLot

	// Clamp to limits.
	if lots < s.MinLots {
		lots = 0
	}
	if lots > s.MaxLots {
		lots = s.MaxLots
	}

	riskUsed := lots * riskPerLot / req.Equity

	return &SizerResult{
		Lots:     lots,
		RiskUsed: riskUsed,
		Method:   s.Name(),
	}, nil
}
