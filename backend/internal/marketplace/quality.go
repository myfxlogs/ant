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
	MinSharpeRatio       decimal.Decimal
	MaxDrawdownPct       decimal.Decimal
	MinTotalTrades       int32
	MinWinRate           decimal.Decimal
	MaxIsOosDegradation  decimal.Decimal
	EnforceSnapshot      bool
}

// loadQualityGates reads thresholds from system_config in a single query.
// Disabled or missing keys are treated as "no gate" (zero value = always passes).
func (s *Service) loadQualityGates(ctx context.Context) (qualityGates, error) {
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

// ValidateBacktestQuality checks a BacktestSnapshot against configured quality gates.
// Returns a slice of violations (empty = passed). A nil snapshot with
// enforce_snapshot enabled returns a single violation.
func (s *Service) ValidateBacktestQuality(ctx context.Context, snapshotProto []byte, strategyID string) ([]QualityViolation, error) {
	gates, err := s.loadQualityGates(ctx)
	if err != nil {
		return nil, fmt.Errorf("marketplace: load quality gates: %w", err)
	}

	waived, err := s.hasQualityWaiver(ctx, strategyID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: check waiver: %w", err)
	}
	if waived {
		return nil, nil
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

	return violations, nil
}
