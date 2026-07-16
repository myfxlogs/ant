// Package risksvc implements the pluggable risk rule engine (M3).
// Each rule implements the RiskRule interface and is registered with the Engine.
// Rules are evaluated in order; first BLOCK stops the pipeline.
package risksvc


import (
	"context"

	"github.com/shopspring/decimal"
)

// RiskRule is a single risk check rule. Name must be unique.
type RiskRule interface {
	Name() string
	Check(ctx context.Context, req *CheckRequest) *CheckResult
}

// CheckRequest is the input to a risk rule evaluation.
type CheckRequest struct {
	UserID    string
	AccountID string
	Symbol    string
	Side      string // buy / sell
	Volume    decimal.Decimal
	Price     decimal.Decimal

	// Account state for margin / position checks
	Balance   decimal.Decimal
	Equity    decimal.Decimal
	Margin    decimal.Decimal
	Positions int // current open position count
}

// CheckResult is the output of a risk rule evaluation.
type CheckResult struct {
	Passed bool
	Reason string
	Rule   string
	Detail string
}
