package marketplace

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/tools/mql2go"
	"alphaforge/tools/mql2go/interp"
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

// checkUnreliableCoverage queries the latest backtest_run's proto_response for
// IsReliable and fatal coverage blind spots. If IsReliable=false or any fatal
// blind spot exists, returns a non-waivable QualityViolation with actionable
// guidance. This closes the "orphan detection" gap: HONESTY-3 sets
// IsReliable=false but the quality gate didn't check it.
func (s *Service) checkUnreliableCoverage(ctx context.Context, strategyID string) *QualityViolation {
	if strategyID == "" {
		return nil
	}
	var protoResponse []byte
	err := s.pg.QueryRow(ctx,
		`SELECT proto_response FROM backtest_runs
		  WHERE strategy_id = $1 AND status = 'SUCCEEDED'
		  ORDER BY created_at DESC
		  LIMIT 1`,
		strategyID,
	).Scan(&protoResponse)
	if err != nil || len(protoResponse) == 0 {
		return nil // no succeeded run or no proto = don't block
	}

	var resp antv1.ExecuteBacktestResponse
	if err := proto.Unmarshal(protoResponse, &resp); err != nil {
		return nil // unmarshal error = don't block (let other gates handle)
	}

	// Collect fatal blind spots with actionable suggestions.
	var fatalDescs []string
	for _, bs := range resp.BlindSpots {
		if isFatalSeverity(bs.Severity) {
			suggestion := blindSpotSuggestion(bs.Id, bs.Description)
			fatalDescs = append(fatalDescs, suggestion)
			// K3: Record demand signal for this unsupported builtin.
			// Best-effort: errors don't block the quality gate.
			if s.demandRecorder != nil {
				uid := uuid.Nil
				if raw := interceptor.GetUserID(ctx); raw != "" {
					if parsed, parseErr := uuid.Parse(raw); parseErr == nil {
						uid = parsed
					}
				}
				_ = s.demandRecorder.RecordDemandSignal(ctx, bs.Id, uid)
			}
		}
	}

	isReliable := resp.GetRisk().GetIsReliable()
	if !isReliable || len(fatalDescs) > 0 {
		actual := "IsReliable=false"
		if isReliable && len(fatalDescs) > 0 {
			actual = "fatal coverage blind spots detected"
		}
		if len(fatalDescs) > 0 {
			actual += ": " + strings.Join(fatalDescs, "; ")
		}
		return &QualityViolation{
			Metric:    "coverage_reliability",
			Actual:    actual,
			Threshold: "IsReliable=true with zero fatal blind spots",
		}
	}
	return nil
}

// isFatalSeverity returns true for fatal severity values (Chinese + English).
func isFatalSeverity(severity string) bool {
	return severity == "\u81f4\u547d" || severity == "fatal"
}

// blindSpotSuggestion maps a fatal blind spot to actionable guidance.
// Returns "人话 + 怎么办" (human-readable + what to do).
func blindSpotSuggestion(id, description string) string {
	switch {
	case strings.Contains(id, "iCustom") || strings.Contains(description, "iCustom"):
		return "iCustom (custom indicator) is not supported — replace with a built-in indicator (iMA/iRSI/iMACD etc.) or implement the logic manually"
	case strings.HasPrefix(id, "i") && len(id) > 1 && id[1] >= 'A' && id[1] <= 'Z':
		return fmt.Sprintf("%s: unknown indicator not supported by the VM — use a supported built-in indicator instead", id)
	case strings.HasPrefix(id, "Order") || strings.HasPrefix(id, "Position"):
		return fmt.Sprintf("%s: trade function not fully supported — check that all order/position operations use supported MQL4/5 APIs", id)
	case strings.Contains(description, "DLL"):
		return "DLL imports are not supported — remove external DLL calls and use built-in MQL functions"
	case strings.Contains(description, "unknown constant"):
		return fmt.Sprintf("unknown constant detected (%s) — replace with a known MQL constant or define it explicitly", description)
	default:
		if description != "" {
			return fmt.Sprintf("%s: %s — fix the issue or remove the unsupported feature", id, description)
		}
		return fmt.Sprintf("%s: unsupported feature — fix or remove from strategy", id)
	}
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
