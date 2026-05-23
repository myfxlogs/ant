// Package risksvc implements the pluggable risk rule engine (M3).
// Each rule implements the RiskRule interface and is registered with the Engine.
// Rules are evaluated in order; first BLOCK stops the pipeline.
package risksvc

import "context"

// RiskRule is a single risk check rule. Name must be unique.
type RiskRule interface {
	Name() string
	Check(ctx context.Context, req *CheckRequest) *CheckResult
}

// CheckRequest is the input to a risk rule evaluation.
type CheckRequest struct {
	UserID    string  `json:"user_id"`
	AccountID string  `json:"account_id"`
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`   // buy / sell
	Volume    float64 `json:"volume"`
	Price     float64 `json:"price"`

	// Account state for margin / position checks
	Balance  float64 `json:"balance"`
	Equity   float64 `json:"equity"`
	Margin   float64 `json:"margin"`
	Positions int    `json:"positions"` // current open position count
}

// CheckResult is the output of a risk rule evaluation.
type CheckResult struct {
	Passed bool   `json:"passed"`
	Reason string `json:"reason,omitempty"`
	Rule   string `json:"rule"`
	Detail string `json:"detail,omitempty"`
}
