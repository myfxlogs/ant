package risksvc

import "context"

// Engine evaluates registered risk rules in order.
// First rule that returns Passed=false stops the pipeline.
type Engine struct {
	rules []RiskRule
}

// NewEngine creates a risk rule engine with the given rules.
func NewEngine(rules ...RiskRule) *Engine {
	return &Engine{rules: rules}
}

// Evaluate runs all registered rules sequentially.
// Returns the first BLOCK result, or nil if all pass.
func (e *Engine) Evaluate(ctx context.Context, req *CheckRequest) *CheckResult {
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
