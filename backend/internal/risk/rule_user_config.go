// rule_user_config.go — Gate rule that reads per-account risk config from DB.
//
// Bridges the frontend risk config form to the Gate.  At evaluation time,
// the Store function fetches the user's saved limits and enforces them.

package risk

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// UserRiskConfig holds per-account limits set by the user via the frontend.
type UserRiskConfig struct {
	MaxLotSize         decimal.Decimal
	MaxPositions       int
	MaxDailyLoss       decimal.Decimal
	MaxDrawdownPercent decimal.Decimal
	MaxRiskPercent     decimal.Decimal
	DailyLossUsed      decimal.Decimal
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

	if rc.MaxLotSize.GreaterThan(decimal.Zero) && vol.GreaterThan(rc.MaxLotSize) {
		return &RuleResult{Allowed: false, Reason: fmt.Sprintf(
			"volume %s exceeds max lot size %s", vol.String(), rc.MaxLotSize.String())}
	}

	if rc.MaxPositions > 0 && state != nil {
		t := intent.GetType()
		if (t == "buy" || t == "sell" || t == "market") && state.OpenPositions >= rc.MaxPositions {
			return &RuleResult{Allowed: false, Reason: fmt.Sprintf(
				"position count %d would exceed max %d", state.OpenPositions+1, rc.MaxPositions)}
		}
	}

	if rc.MaxDailyLoss.GreaterThan(decimal.Zero) && state != nil {
		if state.DailyPnL.LessThan(rc.MaxDailyLoss.Neg()) {
			return &RuleResult{Allowed: false, Reason: fmt.Sprintf(
				"daily loss %s exceeds limit %s", state.DailyPnL.String(), rc.MaxDailyLoss.String())}
		}
	}

	if rc.MaxDrawdownPercent.GreaterThan(decimal.Zero) && state != nil {
		if state.PeakEquity.GreaterThan(decimal.Zero) {
			dd := state.PeakEquity.Sub(state.Equity).Div(state.PeakEquity).Mul(decimal.NewFromInt(100))
			if dd.GreaterThan(rc.MaxDrawdownPercent) {
				return &RuleResult{Allowed: false, Reason: fmt.Sprintf(
					"drawdown %s%% exceeds limit %s%%", dd.String(), rc.MaxDrawdownPercent.String())}
			}
		}
	}

	if rc.MaxRiskPercent.GreaterThan(decimal.Zero) && state != nil {
		price := parseDecimal(intent.GetPrice())
		if state.Equity.GreaterThan(decimal.Zero) && price.GreaterThan(decimal.Zero) {
			pct := vol.Mul(price).Div(state.Equity).Mul(decimal.NewFromInt(100))
			if pct.GreaterThan(rc.MaxRiskPercent) {
				return &RuleResult{Allowed: false, Reason: fmt.Sprintf(
					"risk per trade %s%% exceeds limit %s%%", pct.String(), rc.MaxRiskPercent.String())}
			}
		}
	}

	return &RuleResult{Allowed: true}
}
