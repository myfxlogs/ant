package risksvc

import "fmt"

// PlatformLimits defines aggregate risk boundaries across all accounts.
type PlatformLimits struct {
	MaxTotalGrossExposure float64
	MaxTotalNetExposure   float64
	MaxNetExposurePerSymbol float64
	MaxTotalMarginUsed    float64
}

// DefaultPlatformLimits returns sensible platform-wide defaults.
func DefaultPlatformLimits() *PlatformLimits {
	return &PlatformLimits{
		MaxTotalGrossExposure:  10_000_000,
		MaxTotalNetExposure:    5_000_000,
		MaxNetExposurePerSymbol: 2_000_000,
		MaxTotalMarginUsed:     1_000_000,
	}
}

// PlatformLimitResult holds the outcome of a platform limit check.
type PlatformLimitResult struct {
	Allowed  bool
	Rule     string
	Reason   string
	Current  float64
	Limit    float64
}

// Check evaluates platform exposure against configured limits.
func (l *PlatformLimits) Check(exposure *PlatformExposure) *PlatformLimitResult {
	if l == nil {
		return &PlatformLimitResult{Allowed: true, Rule: "no_limits"}
	}
	if l.MaxTotalGrossExposure > 0 && exposure.TotalGrossExposure > l.MaxTotalGrossExposure {
		return &PlatformLimitResult{
			Allowed: false, Rule: "platform_gross_exposure",
			Reason:  fmt.Sprintf("total gross %.0f > limit %.0f", exposure.TotalGrossExposure, l.MaxTotalGrossExposure),
			Current: exposure.TotalGrossExposure, Limit: l.MaxTotalGrossExposure,
		}
	}
	if l.MaxTotalNetExposure > 0 && abs(exposure.TotalNetExposure) > l.MaxTotalNetExposure {
		return &PlatformLimitResult{
			Allowed: false, Rule: "platform_net_exposure",
			Reason:  fmt.Sprintf("total net %.0f > limit %.0f", abs(exposure.TotalNetExposure), l.MaxTotalNetExposure),
			Current: abs(exposure.TotalNetExposure), Limit: l.MaxTotalNetExposure,
		}
	}
	if l.MaxNetExposurePerSymbol > 0 {
		for sym, net := range exposure.NetExposureBySymbol {
			if abs(net) > l.MaxNetExposurePerSymbol {
				return &PlatformLimitResult{
					Allowed: false, Rule: "platform_symbol_net_exposure",
					Reason:  fmt.Sprintf("%s net %.4f > limit %.0f", sym, abs(net), l.MaxNetExposurePerSymbol),
					Current: abs(net), Limit: l.MaxNetExposurePerSymbol,
				}
			}
		}
	}
	if l.MaxTotalMarginUsed > 0 && exposure.TotalMarginUsed > l.MaxTotalMarginUsed {
		return &PlatformLimitResult{
			Allowed: false, Rule: "platform_margin",
			Reason:  fmt.Sprintf("total margin %.0f > limit %.0f", exposure.TotalMarginUsed, l.MaxTotalMarginUsed),
			Current: exposure.TotalMarginUsed, Limit: l.MaxTotalMarginUsed,
		}
	}
	return &PlatformLimitResult{Allowed: true, Rule: "all_passed"}
}
