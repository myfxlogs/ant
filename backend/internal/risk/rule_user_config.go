// rule_user_config.go — Gate rule that reads per-account risk config from DB.
//
// Bridges the frontend risk config form to the Gate.  At evaluation time,
// the Store function fetches the user's saved limits and enforces them.

package risk

import (
	"context"
	"fmt"

	antv1 "anttrader/gen/proto/ant/v1"
)

// UserRiskConfig holds per-account limits set by the user via the frontend.
type UserRiskConfig struct {
	MaxLotSize         float64
	MaxPositions       int
	MaxDailyLoss       float64
	MaxDrawdownPercent float64
	MaxRiskPercent     float64
	DailyLossUsed      float64
}

// UserRiskConfigRule enforces per-account risk limits stored in the DB.
// Store is nil-safe — pass-through when not configured.
type UserRiskConfigRule struct {
	Store func(ctx context.Context, accountID string) (*UserRiskConfig, error)
}

func (r *UserRiskConfigRule) Name() string { return "user_risk_config" }

func (r *UserRiskConfigRule) Check(ctx context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
	if r.Store == nil {
		return &RuleResult{Allowed: true}
	}

	rc, err := r.Store(ctx, intent.GetAccountId())
	if err != nil || rc == nil {
		return &RuleResult{Allowed: true}
	}

	vol := parseDecimal(intent.GetVolume())

	if rc.MaxLotSize > 0 && vol > rc.MaxLotSize {
		return &RuleResult{Allowed: false, Reason: fmt.Sprintf(
			"volume %.2f exceeds max lot size %.2f", vol, rc.MaxLotSize)}
	}

	if rc.MaxPositions > 0 && state != nil {
		t := intent.GetType()
		if (t == "buy" || t == "sell" || t == "market") && state.OpenPositions >= rc.MaxPositions {
			return &RuleResult{Allowed: false, Reason: fmt.Sprintf(
				"position count %d would exceed max %d", state.OpenPositions+1, rc.MaxPositions)}
		}
	}

	if rc.MaxDailyLoss > 0 {
		dailyLoss, _ := state.DailyPnL.Float64()
		if dailyLoss < -rc.MaxDailyLoss {
			return &RuleResult{Allowed: false, Reason: fmt.Sprintf(
				"daily loss %.2f exceeds limit %.2f", -dailyLoss, rc.MaxDailyLoss)}
		}
	}

	if rc.MaxDrawdownPercent > 0 && state != nil {
		peak, _ := state.PeakEquity.Float64()
		equity, _ := state.Equity.Float64()
		if peak > 0 {
			dd := (peak - equity) / peak * 100
			if dd > rc.MaxDrawdownPercent {
				return &RuleResult{Allowed: false, Reason: fmt.Sprintf(
					"drawdown %.1f%% exceeds limit %.1f%%", dd, rc.MaxDrawdownPercent)}
			}
		}
	}

	if rc.MaxRiskPercent > 0 && state != nil {
		equity, _ := state.Equity.Float64()
		price := parseDecimal(intent.GetPrice())
		if equity > 0 && price > 0 {
			pct := vol * price / equity * 100
			if pct > rc.MaxRiskPercent {
				return &RuleResult{Allowed: false, Reason: fmt.Sprintf(
					"risk per trade %.1f%% exceeds limit %.1f%%", pct, rc.MaxRiskPercent)}
			}
		}
	}

	return &RuleResult{Allowed: true}
}
