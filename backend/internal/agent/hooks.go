package agent

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
)

// HookEvent identifies a lifecycle hook point (ADR-0025 §8).
type HookEvent string

const (
	HookPreStrategySubmit  HookEvent = "pre_strategy_submit"
	HookPostBacktest       HookEvent = "post_backtest"
	HookPreLiveDeploy      HookEvent = "pre_live_deploy"
	HookDegradationAlert   HookEvent = "degradation_alert"
	HookPostStrategyGen    HookEvent = "post_strategy_generation"
)

// HookContext carries data to lifecycle hooks.
type HookContext struct {
	Event       HookEvent
	UserID      uuid.UUID
	StrategyID  string
	Source      string
	BacktestResult *antv1.AgentBacktestResult
	Profile     *antv1.StrategyProfile
	Analysis    *antv1.BacktestAnalysis
	Error       error
}

// HookResult is the outcome of a hook execution.
type HookResult struct {
	Abort   bool   // if true, abort the operation
	Reason  string // abort reason
}

// HookHandler is a function called at a lifecycle hook point.
type HookHandler func(ctx context.Context, hc *HookContext) HookResult

// HookEngine manages lifecycle hooks (ADR-0025 §8).
type HookEngine struct {
	handlers map[HookEvent][]HookHandler
	log      *zap.Logger
}

// NewHookEngine creates a hook engine.
func NewHookEngine(log *zap.Logger) *HookEngine {
	return &HookEngine{
		handlers: make(map[HookEvent][]HookHandler),
		log:      log,
	}
}

// Register adds a handler for a lifecycle event.
func (e *HookEngine) Register(event HookEvent, handler HookHandler) {
	e.handlers[event] = append(e.handlers[event], handler)
}

// Fire executes all handlers for an event. If any handler returns Abort=true,
// subsequent handlers are skipped and the abort result is returned.
func (e *HookEngine) Fire(ctx context.Context, hc *HookContext) HookResult {
	handlers, ok := e.handlers[hc.Event]
	if !ok {
		return HookResult{}
	}

	for _, h := range handlers {
		result := h(ctx, hc)
		if result.Abort {
			e.log.Info("hook: operation aborted",
				zap.String("event", string(hc.Event)),
				zap.String("reason", result.Reason))
			return result
		}
	}
	return HookResult{}
}

// HasHandlers returns true if any handlers are registered for the event.
func (e *HookEngine) HasHandlers(event HookEvent) bool {
	return len(e.handlers[event]) > 0
}
