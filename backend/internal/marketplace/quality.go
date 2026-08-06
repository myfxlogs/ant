package marketplace

import (
	"context"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// QualityViolation describes a single failed quality gate check.
type QualityViolation struct {
	Metric    string
	Actual    string
	Threshold string
}

func (v QualityViolation) String() string {
	return fmt.Sprintf("%s: actual=%s, threshold=%s", v.Metric, v.Actual, v.Threshold)
}

// qualityGates holds the thresholds loaded from system_config.
type qualityGates struct {
	MinSharpeRatio      decimal.Decimal
	MaxDrawdownPct      decimal.Decimal
	MinTotalTrades      int32
	MinWinRate          decimal.Decimal
	MaxIsOosDegradation decimal.Decimal
	EnforceSnapshot     bool
}

// loadQualityGates reads thresholds from system_config in a single query.
// Disabled or missing keys are treated as "no gate" (zero value = always passes).
func (s *Service) loadQualityGates(ctx context.Context) (qualityGates, error) {
	if s.pg == nil {
		return qualityGates{}, nil // no DB = no gates configured
	}
	rows, err := s.pg.Query(ctx,
		`SELECT key, value FROM system_config
		 WHERE key LIKE 'marketplace.quality.%' AND enabled = true`)
	if err != nil {
		return qualityGates{}, nil // table missing or error = no gates
	}
	defer rows.Close()

	m := make(map[string]string, 8)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		m[k] = v
	}
	if err := rows.Err(); err != nil {
		return qualityGates{}, nil
	}

	var g qualityGates
	for k, v := range m {
		switch k {
		case "marketplace.quality.min_sharpe_ratio":
			g.MinSharpeRatio, _ = decimal.NewFromString(v)
		case "marketplace.quality.max_drawdown_pct":
			g.MaxDrawdownPct, _ = decimal.NewFromString(v)
		case "marketplace.quality.min_total_trades":
			n, _ := strconv.ParseInt(v, 10, 32)
			g.MinTotalTrades = int32(n)
		case "marketplace.quality.min_win_rate":
			g.MinWinRate, _ = decimal.NewFromString(v)
		case "marketplace.quality.max_is_oos_degradation":
			g.MaxIsOosDegradation, _ = decimal.NewFromString(v)
		case "marketplace.quality.enforce_backtest_snapshot":
			g.EnforceSnapshot = v == "true" || v == "1"
		}
	}
	return g, nil
}

// hasQualityWaiver checks if a strategy has an admin-granted quality waiver.
func (s *Service) hasQualityWaiver(ctx context.Context, strategyID string) (bool, error) {
	if strategyID == "" {
		return false, nil
	}
	if s.pg == nil {
		return false, nil // no DB = no waivers
	}
	var exists bool
	err := s.pg.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM marketplace_quality_waivers WHERE strategy_id = $1)`,
		strategyID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// checkDegradedStatus queries the latest backtest_run for a strategy.
// If the status is DEGRADED (invariant violation), returns a non-waivable violation.
// Returns nil if the strategy has no backtest runs or the latest is not DEGRADED.
func (s *Service) checkDegradedStatus(ctx context.Context, strategyID string) *QualityViolation {
	if strategyID == "" {
		return nil
	}
	var runStatus string
	err := s.pg.QueryRow(ctx,
		`SELECT status FROM backtest_runs
		  WHERE strategy_id = $1
		  ORDER BY created_at DESC
		  LIMIT 1`,
		strategyID,
	).Scan(&runStatus)
	if err != nil {
		return nil // no run found or query error = don't block
	}
	if runStatus == "DEGRADED" {
		return &QualityViolation{
			Metric:    "backtest_status",
			Actual:    "DEGRADED",
			Threshold: "SUCCEEDED (invariant checks must pass)",
		}
	}
	return nil
}

// ValidateBacktestQuality checks a BacktestSnapshot against configured quality gates.
// Returns a slice of violations (empty = passed). A nil snapshot with
// enforce_snapshot enabled returns a single violation.
//
// DEGRADED hard block: if the latest backtest_run for this strategy has status
// DEGRADED (invariant violation — result unreliable), publishing is blocked
// regardless of waivers. Fake data publishing = fraud, non-waivable.
func (s *Service) ValidateBacktestQuality(ctx context.Context, snapshotProto []byte, strategyID string) ([]QualityViolation, error) {
	gates, err := s.loadQualityGates(ctx)
	if err != nil {
		return nil, fmt.Errorf("marketplace: load quality gates: %w", err)
	}

	if len(snapshotProto) == 0 {
		if gates.EnforceSnapshot {
			return []QualityViolation{{Metric: "backtest_snapshot", Actual: "missing", Threshold: "required"}}, nil
		}
		return nil, nil
	}

	var snap antv1.BacktestSnapshot
	if err := proto.Unmarshal(snapshotProto, &snap); err != nil {
		return []QualityViolation{{Metric: "backtest_snapshot", Actual: "unmarshal_error", Threshold: "valid proto"}}, nil
	}

	// DEGRADED hard block — checked before waiver: fake data publishing is non-waivable.
	if v := s.checkDegradedStatus(ctx, strategyID); v != nil {
		return []QualityViolation{*v}, nil
	}

	waived, err := s.hasQualityWaiver(ctx, strategyID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: check waiver: %w", err)
	}
	if waived {
		return nil, nil
	}

	var violations []QualityViolation

	sharpe, err := decimal.NewFromString(snap.SharpeRatio)
	if err == nil && gates.MinSharpeRatio.IsPositive() && sharpe.LessThan(gates.MinSharpeRatio) {
		violations = append(violations, QualityViolation{
			Metric: "sharpe_ratio", Actual: sharpe.String(), Threshold: gates.MinSharpeRatio.String(),
		})
	}

	drawdown, err := decimal.NewFromString(snap.MaxDrawdown)
	if err == nil && gates.MaxDrawdownPct.IsPositive() && drawdown.GreaterThan(gates.MaxDrawdownPct) {
		violations = append(violations, QualityViolation{
			Metric: "max_drawdown", Actual: drawdown.String(), Threshold: gates.MaxDrawdownPct.String(),
		})
	}

	if gates.MinTotalTrades > 0 && snap.TotalTrades < gates.MinTotalTrades {
		violations = append(violations, QualityViolation{
			Metric: "total_trades", Actual: strconv.FormatInt(int64(snap.TotalTrades), 10),
			Threshold: strconv.FormatInt(int64(gates.MinTotalTrades), 10),
		})
	}

	winRate, err := decimal.NewFromString(snap.WinRate)
	if err == nil && gates.MinWinRate.IsPositive() && winRate.LessThan(gates.MinWinRate) {
		violations = append(violations, QualityViolation{
			Metric: "win_rate", Actual: winRate.String(), Threshold: gates.MinWinRate.String(),
		})
	}

	// Walk-forward OOS degradation check (only if OOS data is present).
	if gates.MaxIsOosDegradation.IsPositive() && snap.OosSharpeRatio != "" {
		isSharpe, errIS := decimal.NewFromString(snap.SharpeRatio)
		oosSharpe, errOOS := decimal.NewFromString(snap.OosSharpeRatio)
		if errIS == nil && errOOS == nil && isSharpe.IsPositive() {
			ratio := oosSharpe.Div(isSharpe)
			degradation := decimal.NewFromInt(1).Sub(ratio)
			if degradation.GreaterThan(gates.MaxIsOosDegradation) {
				violations = append(violations, QualityViolation{
					Metric:    "is_oos_sharpe_degradation",
					Actual:    degradation.String(),
					Threshold: gates.MaxIsOosDegradation.String(),
				})
			}
		}

		isReturn, errIS := decimal.NewFromString(snap.TotalReturn)
		oosReturn, errOOS := decimal.NewFromString(snap.OosTotalReturn)
		if errIS == nil && errOOS == nil && isReturn.IsPositive() {
			ratio := oosReturn.Div(isReturn)
			degradation := decimal.NewFromInt(1).Sub(ratio)
			if degradation.GreaterThan(gates.MaxIsOosDegradation) {
				violations = append(violations, QualityViolation{
					Metric:    "is_oos_return_degradation",
					Actual:    degradation.String(),
					Threshold: gates.MaxIsOosDegradation.String(),
				})
			}
		}
	}

	return violations, nil
}
