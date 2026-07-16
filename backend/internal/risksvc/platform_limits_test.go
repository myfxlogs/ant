package risksvc

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestPlatformLimits_AllPass(t *testing.T) {
	t.Parallel()
	limits := DefaultPlatformLimits()
	exposure := &PlatformExposure{
		NetExposureBySymbol: map[string]decimal.Decimal{"EURUSD": decF(1.0)},
		TotalGrossExposure:  decF(500_000),
		TotalNetExposure:    decF(200_000),
		TotalMarginUsed:     decF(50_000),
	}
	result := limits.Check(exposure)
	if !result.Allowed {
		t.Fatalf("expected pass, got blocked: %s", result.Reason)
	}
}

func TestPlatformLimits_GrossExposureBlocked(t *testing.T) {
	t.Parallel()
	limits := &PlatformLimits{MaxTotalGrossExposure: decF(1_000_000)}
	exposure := &PlatformExposure{
		TotalGrossExposure: decF(1_500_000),
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
	t.Parallel()
	limits := &PlatformLimits{MaxTotalNetExposure: decF(500_000)}
	exposure := &PlatformExposure{
		TotalNetExposure: decF(-800_000),
	}
	result := limits.Check(exposure)
	if result.Allowed {
		t.Fatal("should block on net exposure")
	}
}

func TestPlatformLimits_SymbolNetExposureBlocked(t *testing.T) {
	t.Parallel()
	limits := &PlatformLimits{MaxNetExposurePerSymbol: decF(1_000_000)}
	exposure := &PlatformExposure{
		NetExposureBySymbol: map[string]decimal.Decimal{"EURUSD": decF(1_500_000)},
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
	t.Parallel()
	limits := &PlatformLimits{MaxTotalMarginUsed: decF(100_000)}
	exposure := &PlatformExposure{
		TotalMarginUsed: decF(150_000),
	}
	result := limits.Check(exposure)
	if result.Allowed {
		t.Fatal("should block on margin")
	}
}

func TestPlatformLimits_NilLimits(t *testing.T) {
	t.Parallel()
	var limits *PlatformLimits
	exposure := &PlatformExposure{TotalGrossExposure: decF(100_000_000)}
	result := limits.Check(exposure)
	if !result.Allowed {
		t.Fatal("nil limits should always pass")
	}
}

func TestPlatformLimits_ZeroLimits(t *testing.T) {
	t.Parallel()
	limits := &PlatformLimits{}
	exposure := &PlatformExposure{TotalGrossExposure: decF(100_000_000)}
	result := limits.Check(exposure)
	if !result.Allowed {
		t.Fatal("zero-value limits should pass (disabled)")
	}
}
