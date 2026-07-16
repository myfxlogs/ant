package risksvc

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// PlatformLimits defines aggregate risk boundaries across all accounts.
type PlatformLimits struct {
	MaxTotalGrossExposure decimal.Decimal
	MaxTotalNetExposure   decimal.Decimal
	MaxNetExposurePerSymbol decimal.Decimal
	MaxTotalMarginUsed    decimal.Decimal
}

// DefaultPlatformLimits returns liberal platform-wide defaults.
// These are high-water safety nets, not daily trading gates.
// Individual account limits are enforced by the broker (MT5).
func DefaultPlatformLimits() *PlatformLimits {
	return &PlatformLimits{
		MaxTotalGrossExposure:   decimal.NewFromInt(500_000_000),
		MaxTotalNetExposure:     decimal.NewFromInt(250_000_000),
		MaxNetExposurePerSymbol: decimal.NewFromInt(100_000_000),
		MaxTotalMarginUsed:      decimal.NewFromInt(10_000_000),
	}
}

// PlatformLimitResult holds the outcome of a platform limit check.
type PlatformLimitResult struct {
	Allowed  bool
	Rule     string
	Reason   string
	Current  decimal.Decimal
	Limit    decimal.Decimal
}

// Check evaluates platform exposure against configured limits.
func (l *PlatformLimits) Check(exposure *PlatformExposure) *PlatformLimitResult {
	if l == nil {
		return &PlatformLimitResult{Allowed: true, Rule: "no_limits"}
	}
	if l.MaxTotalGrossExposure.GreaterThan(decimal.Zero) && exposure.TotalGrossExposure.GreaterThan(l.MaxTotalGrossExposure) {
		return &PlatformLimitResult{
			Allowed: false, Rule: "platform_gross_exposure",
			Reason:  fmt.Sprintf("total gross %s > limit %s", exposure.TotalGrossExposure.String(), l.MaxTotalGrossExposure.String()),
			Current: exposure.TotalGrossExposure, Limit: l.MaxTotalGrossExposure,
		}
	}
	if l.MaxTotalNetExposure.GreaterThan(decimal.Zero) && abs(exposure.TotalNetExposure).GreaterThan(l.MaxTotalNetExposure) {
		return &PlatformLimitResult{
			Allowed: false, Rule: "platform_net_exposure",
			Reason:  fmt.Sprintf("total net %s > limit %s", abs(exposure.TotalNetExposure).String(), l.MaxTotalNetExposure.String()),
			Current: abs(exposure.TotalNetExposure), Limit: l.MaxTotalNetExposure,
		}
	}
	if l.MaxNetExposurePerSymbol.GreaterThan(decimal.Zero) {
		for sym, net := range exposure.NetExposureBySymbol {
			if abs(net).GreaterThan(l.MaxNetExposurePerSymbol) {
				return &PlatformLimitResult{
					Allowed: false, Rule: "platform_symbol_net_exposure",
					Reason:  fmt.Sprintf("%s net %s > limit %s", sym, abs(net).String(), l.MaxNetExposurePerSymbol.String()),
					Current: abs(net), Limit: l.MaxNetExposurePerSymbol,
				}
			}
		}
	}
	if l.MaxTotalMarginUsed.GreaterThan(decimal.Zero) && exposure.TotalMarginUsed.GreaterThan(l.MaxTotalMarginUsed) {
		return &PlatformLimitResult{
			Allowed: false, Rule: "platform_margin",
			Reason:  fmt.Sprintf("total margin %s > limit %s", exposure.TotalMarginUsed.String(), l.MaxTotalMarginUsed.String()),
			Current: exposure.TotalMarginUsed, Limit: l.MaxTotalMarginUsed,
		}
	}
	return &PlatformLimitResult{Allowed: true, Rule: "all_passed"}
}
