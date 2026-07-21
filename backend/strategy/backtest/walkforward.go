package backtest

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/sdk"
)

// WalkForwardResult holds IS (in-sample) and OOS (out-of-sample) metrics.
type WalkForwardResult struct {
	IS  *antv1.BacktestMetrics
	OOS *antv1.BacktestMetrics
}

// RunWalkForward splits bars into IS (70%) and OOS (30%) segments,
// runs the strategy on each, and returns both metric sets.
// The strategy is re-initialized for each segment (no state leakage).
func RunWalkForward(ctx context.Context, cfg Config, strategy sdk.Strategy, bars []sdk.Bar) (*WalkForwardResult, error) {
	if len(bars) < 10 {
		return nil, fmt.Errorf("walk-forward: insufficient bars (%d) for IS/OOS split", len(bars))
	}

	splitIdx := int(float64(len(bars)) * 0.7)
	if splitIdx < 5 || len(bars)-splitIdx < 5 {
		return nil, fmt.Errorf("walk-forward: split produces too few bars (IS=%d, OOS=%d)", splitIdx, len(bars)-splitIdx)
	}

	isBars := bars[:splitIdx]
	oosBars := bars[splitIdx:]

	isResult, err := runSegment(ctx, cfg, strategy, isBars)
	if err != nil {
		return nil, fmt.Errorf("walk-forward IS: %w", err)
	}

	oosResult, err := runSegment(ctx, cfg, strategy, oosBars)
	if err != nil {
		return nil, fmt.Errorf("walk-forward OOS: %w", err)
	}

	return &WalkForwardResult{
		IS:  isResult.Metrics,
		OOS: oosResult.Metrics,
	}, nil
}

// runSegment runs a single backtest on a bar subset and returns the result.
func runSegment(ctx context.Context, cfg Config, strategy sdk.Strategy, bars []sdk.Bar) (*Result, error) {
	segCfg := cfg
	segCfg.StartDate = timeFromMs(bars[0].Timestamp)
	segCfg.EndDate = timeFromMs(bars[len(bars)-1].Timestamp)

	engine := New(segCfg, strategy, bars)
	result, err := engine.Run(ctx)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func timeFromMs(ms int64) time.Time {
	return time.UnixMilli(ms)
}

// DegradationRatio computes OOS/IS ratio for a metric.
// Returns 0 if IS is zero or negative (can't compute meaningful ratio).
// A ratio < 1 means OOS is worse than IS (degradation).
func DegradationRatio(isVal, oosVal decimal.Decimal) decimal.Decimal {
	if !isVal.IsPositive() {
		return decimal.Zero
	}
	return oosVal.Div(isVal)
}
