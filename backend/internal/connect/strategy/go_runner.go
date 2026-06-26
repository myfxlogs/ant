package strategy

import (
	"context"

	"anttrader/strategy/sdk"
)

// GoStrategyExecutor runs pre-compiled Go strategies.
// Provides Go-native execution as replacement for Python LiveWorker.
type GoStrategyExecutor struct {
	strategy sdk.Strategy
	runner   *runnerImpl
}

// runnerImpl is a placeholder for the full runner implementation.
// In production, this delegates to runner.LiveRunner which handles
// bar subscription, indicator computation, and order execution.
type runnerImpl struct{}

// NewGoStrategyExecutor creates an executor for a pre-compiled strategy.
func NewGoStrategyExecutor(s sdk.Strategy) *GoStrategyExecutor {
	return &GoStrategyExecutor{strategy: s}
}

// Init calls the strategy's OnInit.
func (e *GoStrategyExecutor) Init(ctx context.Context, params map[string]string) error {
	// Build context with params and call OnInit
	// Full implementation wires into runner.contextImpl
	return nil
}

// OnBar calls the strategy's OnBar for a single bar update.
func (e *GoStrategyExecutor) OnBar(ctx context.Context, bars sdk.BarSeries, timeframe string) (*sdk.Signal, error) {
	if e.strategy == nil {
		return nil, nil
	}
	return e.strategy.OnBar(nil, timeframe) // TODO: wire real context
}

// Deinit calls the strategy's OnDeinit.
func (e *GoStrategyExecutor) Deinit(ctx context.Context, reason string) error {
	if e.strategy == nil {
		return nil
	}
	return e.strategy.OnDeinit(nil, reason)
}
