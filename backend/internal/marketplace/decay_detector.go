// Package marketplace — Phase 5.1a: Alpha decay detection.
//
// DetectDecay analyzes live performance data to identify strategies whose
// alpha is decaying. It compares recent performance (last 30 days) against
// the baseline (full history or first 90 days) using three signals:
//  1. Rolling Sharpe ratio decline (>30% drop)
//  2. Win rate decline (>15% drop)
//  3. Cumulative return plateau or drawdown increase
//
// A strategy is flagged as "decaying" when at least 2 of 3 signals fire.
package marketplace

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// dailyRow is a single day's performance data used for decay detection.
type dailyRow struct {
	date          time.Time
	dailyReturn   decimal.Decimal
	winningTrades int32
	totalTrades   int32
}

// decimalSqrt computes the square root of a decimal.Decimal using math.Sqrt.
// This is approximate but sufficient for Sharpe ratio calculations.
func decimalSqrt(d decimal.Decimal) decimal.Decimal {
	f, _ := d.Float64()
	return decimal.NewFromFloat(math.Sqrt(f))
}

// DecayResult holds the output of a decay detection analysis.
type DecayResult struct {
	StrategyID    string
	IsDecaying    bool
	DecayScore    int32  // 0-3, number of signals that fired
	TriggerReason string // human-readable summary
	// Signal tracking
	SharpeSignal  bool
	WinRateSignal bool
	ReturnSignal  bool
	// Baseline period metrics
	BaselineSharpe  *decimal.Decimal
	BaselineWinRate *decimal.Decimal
	BaselineReturn  decimal.Decimal
	// Recent period metrics
	RecentSharpe  *decimal.Decimal
	RecentWinRate *decimal.Decimal
	RecentReturn  decimal.Decimal
	// Computed deltas
	SharpeDeclinePct  decimal.Decimal
	WinRateDeclinePct decimal.Decimal
	ReturnDelta       decimal.Decimal
}

// DecayThresholds are the configurable thresholds for decay detection.
type DecayThresholds struct {
	SharpeDeclinePct  decimal.Decimal // 0.30 = 30% drop triggers signal
	WinRateDeclinePct decimal.Decimal // 0.15 = 15% drop triggers signal
	ReturnDeclinePct  decimal.Decimal // 0.70 = recent return < 70% of expected triggers signal
	MinDataPoints     int             // minimum daily records needed
	RecentWindowDays  int             // rolling window for recent period
}

// DefaultDecayThresholds returns the standard decay detection thresholds.
func DefaultDecayThresholds() DecayThresholds {
	return DecayThresholds{
		SharpeDeclinePct:  decimal.NewFromFloat(0.30),
		WinRateDeclinePct: decimal.NewFromFloat(0.15),
		ReturnDeclinePct:  decimal.NewFromFloat(0.70),
		MinDataPoints:     30,
		RecentWindowDays:  30,
	}
}

// loadDecayThresholds reads decay detection thresholds from system_config.
// Falls back to DefaultDecayThresholds for any missing or disabled keys.
func (s *Service) loadDecayThresholds(ctx context.Context) DecayThresholds {
	th := DefaultDecayThresholds()
	rows, err := s.pg.Query(ctx,
		`SELECT key, value FROM system_config
		 WHERE key LIKE 'marketplace.decay.%' AND enabled = true`)
	if err != nil {
		return th
	}
	defer rows.Close()
	for rows.Next() {
		var key, val string
		if err := rows.Scan(&key, &val); err != nil {
			continue
		}
		switch key {
		case "marketplace.decay.sharpe_decline_threshold":
			if d, err := decimal.NewFromString(val); err == nil {
				th.SharpeDeclinePct = d
			}
		case "marketplace.decay.winrate_decline_threshold":
			if d, err := decimal.NewFromString(val); err == nil {
				th.WinRateDeclinePct = d
			}
		case "marketplace.decay.return_decline_threshold":
			if d, err := decimal.NewFromString(val); err == nil {
				th.ReturnDeclinePct = d
			}
		case "marketplace.decay.lookback_days":
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				th.RecentWindowDays = n
			}
		case "marketplace.decay.min_live_days":
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				th.MinDataPoints = n
			}
		}
	}
	return th
}

// DetectDecay runs decay detection for a single strategy.
// It reads from marketplace_live_performance and returns a DecayResult.
// If the strategy has insufficient live performance data (< MinDataPoints days),
// it returns a result with IsDecaying=false.
func (s *Service) DetectDecay(ctx context.Context, strategyID string) (*DecayResult, error) {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: detect decay: invalid strategy_id: %w", err)
	}
	return s.detectDecayWithThresholds(ctx, sid, s.loadDecayThresholds(ctx))
}

// detectDecayWithThresholds is the core implementation, separated for testability.
func (s *Service) detectDecayWithThresholds(ctx context.Context, sid uuid.UUID, th DecayThresholds) (*DecayResult, error) {
	recentCutoff := time.Now().UTC().Add(-time.Duration(th.RecentWindowDays) * 24 * time.Hour)

	// Fetch all daily performance records.
	rows, err := s.pg.Query(ctx,
		`SELECT date, daily_return, winning_trades, total_trades
		 FROM marketplace_live_performance
		 WHERE strategy_id = $1
		 ORDER BY date ASC`,
		sid)
	if err != nil {
		return nil, fmt.Errorf("marketplace: detect decay: query: %w", err)
	}
	defer rows.Close()

	var allRows []dailyRow
	for rows.Next() {
		var r dailyRow
		if err := rows.Scan(&r.date, &r.dailyReturn, &r.winningTrades, &r.totalTrades); err != nil {
			return nil, fmt.Errorf("marketplace: detect decay: scan: %w", err)
		}
		allRows = append(allRows, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &DecayResult{StrategyID: sid.String()}

	if len(allRows) < th.MinDataPoints {
		return result, nil // insufficient data
	}

	// Split into baseline (all data before recentCutoff) and recent (after).
	var baseline, recent []dailyRow
	for _, r := range allRows {
		if r.date.Before(recentCutoff) {
			baseline = append(baseline, r)
		} else {
			recent = append(recent, r)
		}
	}

	if len(baseline) < 10 || len(recent) < 5 {
		return result, nil // not enough data in either period
	}

	// Compute metrics for each period.
	baselineMetrics := computePeriodMetrics(baseline)
	recentMetrics := computePeriodMetrics(recent)

	result.BaselineSharpe = baselineMetrics.sharpe
	result.BaselineWinRate = baselineMetrics.winRate
	result.BaselineReturn = baselineMetrics.totalReturn
	result.RecentSharpe = recentMetrics.sharpe
	result.RecentWinRate = recentMetrics.winRate
	result.RecentReturn = recentMetrics.totalReturn

	// Signal 1: Sharpe ratio decline.
	if baselineMetrics.sharpe != nil && recentMetrics.sharpe != nil {
		if baselineMetrics.sharpe.GreaterThan(decimal.Zero) {
			decline := baselineMetrics.sharpe.Sub(*recentMetrics.sharpe).Div(*baselineMetrics.sharpe)
			result.SharpeDeclinePct = decline
			if decline.GreaterThan(th.SharpeDeclinePct) {
				result.DecayScore++
				result.SharpeSignal = true
			}
		}
	}

	// Signal 2: Win rate decline.
	if baselineMetrics.winRate != nil && recentMetrics.winRate != nil {
		decline := baselineMetrics.winRate.Sub(*recentMetrics.winRate)
		result.WinRateDeclinePct = decline
		if decline.GreaterThan(th.WinRateDeclinePct) {
			result.DecayScore++
			result.WinRateSignal = true
		}
	}

	// Signal 3: Return plateau or reversal.
	// Compare recent period total return vs baseline period average daily return * recent days.
	// Signal fires when recent return < ReturnDeclinePct fraction of expected (default 70%).
	if baselineMetrics.avgDailyReturn.GreaterThan(decimal.Zero) {
		expectedRecent := baselineMetrics.avgDailyReturn.Mul(decimal.NewFromInt(int64(th.RecentWindowDays)))
		result.ReturnDelta = recentMetrics.totalReturn.Sub(expectedRecent)
		if recentMetrics.totalReturn.LessThanOrEqual(decimal.Zero) {
			result.DecayScore++
			result.ReturnSignal = true
		} else if recentMetrics.totalReturn.LessThan(expectedRecent.Mul(th.ReturnDeclinePct)) {
			result.DecayScore++
			result.ReturnSignal = true
		}
	} else if recentMetrics.totalReturn.LessThanOrEqual(decimal.Zero) {
		result.DecayScore++ // baseline was flat and recent is negative
		result.ReturnSignal = true
	}

	result.IsDecaying = result.DecayScore >= 2
	result.TriggerReason = formatDecayReason(result)
	return result, nil
}

type periodMetrics struct {
	sharpe         *decimal.Decimal
	winRate        *decimal.Decimal
	totalReturn    decimal.Decimal
	avgDailyReturn decimal.Decimal
}
