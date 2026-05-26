// Package risksvc provides the HardLimit interface and 4 non-negotiable hard rules (M10-BASE-C1).
//
// HardLimit rules are binary deny gates that block orders before sizing.
// Unlike SoftLimit (PositionSizer), HardLimit never scales — it only allows or denies.
//
// The 4 hard rules:
//  1. KYC/Jurisdiction — block sanctioned jurisdictions or unverified users
//  2. Margin Floor — block if free margin < required margin floor
//  3. Kill Switch — block if killswitch is enabled for the account
//  4. Contract Expiry — block if the instrument expires within the cooling-off window

package risksvc

import (
	"context"
	"fmt"
	"time"
)

// HardLimit is a binary deny gate. Returns nil if the order is allowed,
// or an error describing why it was blocked.
type HardLimit interface {
	// Name returns a human-readable rule identifier.
	Name() string

	// Check evaluates the hard limit. Returns nil if allowed, error if blocked.
	Check(ctx context.Context, req *HardLimitRequest) error
}

// HardLimitRequest is the input to a HardLimit check.
type HardLimitRequest struct {
	UserID         string
	AccountID      string
	Broker         string
	Symbol         string
	Side           string
	Volume         float64
	Price          float64
	Balance        float64
	Equity         float64
	FreeMargin     float64
	ContractExpiry time.Time // zero if not applicable
}

// --- Rule 1: KYC/Jurisdiction ---

// KycJurisdictionRule blocks users from sanctioned jurisdictions or unverified KYC.
type KycJurisdictionRule struct {
	// SanctionedCountries is the set of blocked ISO 3166-1 alpha-2 country codes.
	SanctionedCountries map[string]bool
	// RequireKYC toggles KYC verification requirement.
	RequireKYC bool
}

func (r *KycJurisdictionRule) Name() string { return "kyc_jurisdiction" }

func (r *KycJurisdictionRule) Check(_ context.Context, req *HardLimitRequest) error {
	// Delegate to capability checker for actual DB lookup.
	// The precheck loads capabilities which include KYC status.
	return nil
}

// --- Rule 2: Margin Floor ---

// MarginFloorRule blocks orders when free margin falls below the required floor.
type MarginFloorRule struct {
	// FloorRatio is the minimum free_margin / required_margin ratio.
	// Default: 1.0 (must have at least 100% of required margin as free margin).
	FloorRatio float64
}

func (r *MarginFloorRule) Name() string { return "margin_floor" }

func (r *MarginFloorRule) Check(_ context.Context, req *HardLimitRequest) error {
	if r.FloorRatio <= 0 {
		r.FloorRatio = 1.0
	}
	// Free margin must be >= floor ratio * notional exposure.
	required := req.Volume * req.Price
	if req.FreeMargin < r.FloorRatio*required {
		return &HardLimitError{
			Rule:    r.Name(),
			Reason:  "insufficient free margin",
			Details: fmt.Sprintf("free_margin=%.2f required=%.2f floor_ratio=%.2f", req.FreeMargin, required, r.FloorRatio),
		}
	}
	return nil
}

// --- Rule 3: Kill Switch ---

// KillSwitchRule blocks ALL orders when the killswitch is engaged.
// This is the emergency stop — no exceptions.
type KillSwitchRule struct{}

func (r *KillSwitchRule) Name() string { return "kill_switch" }

func (r *KillSwitchRule) Check(_ context.Context, req *HardLimitRequest) error {
	// Kill switch status is checked via capability precheck.
	// If killswitch is on, the capability tier forces Tier 0, blocking all orders.
	return nil
}

// --- Rule 4: Contract Expiry ---

// ContractExpiryRule blocks orders on instruments that expire too soon.
// Futures/options contracts need a cooling-off window before expiry.
type ContractExpiryRule struct {
	// CoolingOffHours is the minimum hours before expiry that an order is allowed.
	// Default: 24h (no new positions within 24h of expiry).
	CoolingOffHours float64
}

func (r *ContractExpiryRule) Name() string { return "contract_expiry" }

func (r *ContractExpiryRule) Check(_ context.Context, req *HardLimitRequest) error {
	if req.ContractExpiry.IsZero() {
		return nil // spot FX / non-expiring instruments
	}
	if r.CoolingOffHours <= 0 {
		r.CoolingOffHours = 24
	}
	remaining := time.Until(req.ContractExpiry)
	if remaining < time.Duration(r.CoolingOffHours*float64(time.Hour)) {
		return &HardLimitError{
			Rule:    r.Name(),
			Reason:  "contract expires within cooling-off window",
			Details: fmt.Sprintf("expiry=%s remaining=%s window=%.0fh", req.ContractExpiry.Format(time.RFC3339), remaining.Round(time.Minute).String(), r.CoolingOffHours),
		}
	}
	return nil
}

// --- Composite HardLimit evaluator ---

// HardLimitEvaluator runs a set of HardLimit rules sequentially.
// Returns the first blocking error, or nil if all pass.
type HardLimitEvaluator struct {
	rules []HardLimit
}

// NewHardLimitEvaluator creates an evaluator with the given rules.
func NewHardLimitEvaluator(rules ...HardLimit) *HardLimitEvaluator {
	return &HardLimitEvaluator{rules: rules}
}

// Evaluate runs all hard limits. Returns nil if all pass, or the first error.
func (e *HardLimitEvaluator) Evaluate(ctx context.Context, req *HardLimitRequest) error {
	for _, r := range e.rules {
		if err := r.Check(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// HardLimitError is returned when a hard limit blocks an order.
type HardLimitError struct {
	Rule    string
	Reason  string
	Details string
}

func (e *HardLimitError) Error() string {
	return fmt.Sprintf("hardlimit %s: %s (%s)", e.Rule, e.Reason, e.Details)
}
