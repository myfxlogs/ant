package risksvc

import (
	"context"

	"anttrader/internal/usermgr"
)

// Engine evaluates registered risk rules in order.
// First rule that returns Passed=false stops the pipeline.
type Engine struct {
	rules       []RiskRule
	userLimiter *usermgr.UserLimiter
}

// NewEngine creates a risk rule engine with the given rules.
func NewEngine(rules ...RiskRule) *Engine {
	return &Engine{rules: rules}
}

// SetUserLimiter injects the per-user rate limiter for signal generation (nil-safe).
func (e *Engine) SetUserLimiter(l *usermgr.UserLimiter) { e.userLimiter = l }

// Evaluate runs all registered rules sequentially.
// Returns the first BLOCK result, or nil if all pass.
// Also enforces per-user signal rate limit when a limiter is configured.
func (e *Engine) Evaluate(ctx context.Context, req *CheckRequest) *CheckResult {
	if e.userLimiter != nil && req.UserID != "" {
		if !e.userLimiter.AllowSignal(req.UserID) {
			return &CheckResult{
				Passed: false,
				Rule:   "rate_limit",
				Reason: "signal rate limit exceeded",
				Detail: req.UserID,
			}
		}
	}

	for _, rule := range e.rules {
		result := rule.Check(ctx, req)
		if !result.Passed {
			return result
		}
	}
	return &CheckResult{Passed: true, Rule: "all_passed"}
}

// Rules returns the list of registered rule names.
func (e *Engine) Rules() []string {
	names := make([]string, len(e.rules))
	for i, r := range e.rules {
		names[i] = r.Name()
	}
	return names
}
