// quality_validation.go — ValidateBacktestQuality and coverage checks extracted from quality.go.
package marketplace

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/tools/mql2go"
	"alphaforge/tools/mql2go/interp"
)

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

	// Unreliable coverage hard block — checked before waiver: strategies with
	// IsReliable=false or fatal coverage blind spots cannot be published.
	// This closes the "orphan detection" gap (HONESTY-3 sets IsReliable but
	// the quality gate didn't check it). Non-waivable: unreliable results = fraud.
	if v := s.checkUnreliableCoverage(ctx, strategyID); v != nil {
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

// CheckLiveCoverage checks whether a strategy is safe to run on a real account.
// T5: Symmetric with the publish gate — strategies with fatal blind spots must
// not run live. Two paths:
//  1. If a recent SUCCEEDED backtest exists, check its IsReliable + fatal blind spots.
//  2. If no backtest exists, do a compile+coverage analysis on the source code.
//
// Returns nil if safe (no fatal blind spots), or an error describing the fatal
// blind spots if unsafe. Paper mode is the caller's responsibility (caller skips).
func (s *Service) CheckLiveCoverage(ctx context.Context, strategyID, sourceCode string) error {
	// Path 1: Check latest SUCCEEDED backtest.
	if strategyID != "" {
		if v := s.checkUnreliableCoverage(ctx, strategyID); v != nil {
			return fmt.Errorf("live coverage gate: %s (actual=%s, threshold=%s)",
				v.Metric, v.Actual, v.Threshold)
		}
		// Also check DEGRADED status — invariant violations mean unreliable.
		if v := s.checkDegradedStatus(ctx, strategyID); v != nil {
			return fmt.Errorf("live coverage gate: %s (actual=%s, threshold=%s)",
				v.Metric, v.Actual, v.Threshold)
		}
		return nil
	}

	// Path 2: No strategy ID or no backtest — do compile+coverage analysis.
	if sourceCode == "" {
		return nil // no code to check = can't block (let other gates handle)
	}
	return checkSourceCoverage(sourceCode)
}

// checkSourceCoverage compiles the MQL source and checks for fatal blind spots.
// Used when no backtest exists yet (e.g. user just wrote code in workspace).
func checkSourceCoverage(sourceCode string) error {
	_, cov, err := mql2go.CompileMQLWithCoverage(sourceCode)
	if err != nil {
		// Compile error = honest failure, not a fatal blind spot. Allow live
		// (the VM will report the error at runtime). Only fatal blind spots block.
		return nil
	}
	if cov == nil {
		return nil
	}
	var fatalDescs []string
	for _, bs := range cov.BlindSpots {
		if bs.Severity == interp.SeverityFatal {
			suggestion := blindSpotSuggestion(bs.Builtin, "")
			fatalDescs = append(fatalDescs, suggestion)
		}
	}
	if len(fatalDescs) > 0 {
		return fmt.Errorf("fatal coverage blind spots: %s", strings.Join(fatalDescs, "; "))
	}
	return nil
}
