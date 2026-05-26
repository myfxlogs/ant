package risksvc

import (
	"testing"
)

func TestPlatformLimits_AllPass(t *testing.T) {
	limits := DefaultPlatformLimits()
	exposure := &PlatformExposure{
		NetExposureBySymbol: map[string]float64{"EURUSD": 1.0},
		TotalGrossExposure:  500_000,
		TotalNetExposure:    200_000,
		TotalMarginUsed:     50_000,
	}
	result := limits.Check(exposure)
	if !result.Allowed {
		t.Fatalf("expected pass, got blocked: %s", result.Reason)
	}
}

func TestPlatformLimits_GrossExposureBlocked(t *testing.T) {
	limits := &PlatformLimits{MaxTotalGrossExposure: 1_000_000}
	exposure := &PlatformExposure{
		TotalGrossExposure: 1_500_000,
	}
	result := limits.Check(exposure)
	if result.Allowed {
		t.Fatal("should block on gross exposure")
	}
	if result.Rule != "platform_gross_exposure" {
		t.Fatalf("want platform_gross_exposure, got %s", result.Rule)
	}
}

func TestPlatformLimits_NetExposureBlocked(t *testing.T) {
	limits := &PlatformLimits{MaxTotalNetExposure: 500_000}
	exposure := &PlatformExposure{
		TotalNetExposure: -800_000,
	}
	result := limits.Check(exposure)
	if result.Allowed {
		t.Fatal("should block on net exposure")
	}
}

func TestPlatformLimits_SymbolNetExposureBlocked(t *testing.T) {
	limits := &PlatformLimits{MaxNetExposurePerSymbol: 1_000_000}
	exposure := &PlatformExposure{
		NetExposureBySymbol: map[string]float64{"EURUSD": 1_500_000},
	}
	result := limits.Check(exposure)
	if result.Allowed {
		t.Fatal("should block on symbol net exposure")
	}
	if result.Rule != "platform_symbol_net_exposure" {
		t.Fatalf("want platform_symbol_net_exposure, got %s", result.Rule)
	}
}

func TestPlatformLimits_MarginBlocked(t *testing.T) {
	limits := &PlatformLimits{MaxTotalMarginUsed: 100_000}
	exposure := &PlatformExposure{
		TotalMarginUsed: 150_000,
	}
	result := limits.Check(exposure)
	if result.Allowed {
		t.Fatal("should block on margin")
	}
}

func TestPlatformLimits_NilLimits(t *testing.T) {
	var limits *PlatformLimits
	exposure := &PlatformExposure{TotalGrossExposure: 100_000_000}
	result := limits.Check(exposure)
	if !result.Allowed {
		t.Fatal("nil limits should always pass")
	}
}

func TestPlatformLimits_ZeroLimits(t *testing.T) {
	limits := &PlatformLimits{}
	exposure := &PlatformExposure{TotalGrossExposure: 100_000_000}
	result := limits.Check(exposure)
	if !result.Allowed {
		t.Fatal("zero-value limits should pass (disabled)")
	}
}
