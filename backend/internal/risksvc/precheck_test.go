package risksvc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
)

func TestPreCheck_AllPassed(t *testing.T) {
	t.Parallel()
	limits := DefaultRiskLimits()
	req := &CheckRequest{Symbol: "EURUSD", Positions: 1, Volume: decF(0.1)}
	res := PreCheck(context.Background(), req, limits, 1, decF(10000), decF(1000))
	if !res.Allowed {
		t.Fatalf("expected allowed, got blocked by %s: %s", res.Rule, res.Reason)
	}
	if res.Rule != "all_passed" {
		t.Fatalf("expected rule=all_passed, got %s", res.Rule)
	}
}

func TestPreCheck_SymbolPositionLimit(t *testing.T) {
	t.Parallel()
	limits := &RiskLimits{MaxPositionsPerSymbol: 3}
	req := &CheckRequest{Symbol: "EURUSD", Positions: 5}
	res := PreCheck(context.Background(), req, limits, 3, decimal.Zero, decimal.Zero)
	if res.Allowed {
		t.Fatal("expected blocked by symbol_position_limit")
	}
	if res.Rule != "symbol_position_limit" {
		t.Fatalf("expected rule=symbol_position_limit, got %s", res.Rule)
	}
}

func TestPreCheck_TotalExposure(t *testing.T) {
	t.Parallel()
	limits := &RiskLimits{MaxTotalPositions: 10}
	req := &CheckRequest{Symbol: "EURUSD", Positions: 10}
	res := PreCheck(context.Background(), req, limits, 0, decimal.Zero, decimal.Zero)
	if res.Allowed {
		t.Fatal("expected blocked by total_exposure")
	}
	if res.Rule != "total_exposure" {
		t.Fatalf("expected rule=total_exposure, got %s", res.Rule)
	}
}

func TestPreCheck_AccountExposure(t *testing.T) {
	t.Parallel()
	limits := &RiskLimits{MaxExposurePerAccount: decF(50)}
	req := &CheckRequest{Symbol: "EURUSD", Volume: decF(100)}
	res := PreCheck(context.Background(), req, limits, 0, decimal.Zero, decimal.Zero)
	if res.Allowed {
		t.Fatal("expected blocked by account_exposure")
	}
	if res.Rule != "account_exposure" {
		t.Fatalf("expected rule=account_exposure, got %s", res.Rule)
	}
}

func TestPreCheck_MarginUtilization(t *testing.T) {
	t.Parallel()
	limits := &RiskLimits{MaxMarginUtilizationPct: 50}
	req := &CheckRequest{Symbol: "EURUSD"}
	// requiredMargin=8000, freeMargin=10000 → 80% > 50%
	res := PreCheck(context.Background(), req, limits, 0, decF(10000), decF(8000))
	if res.Allowed {
		t.Fatal("expected blocked by margin_utilization")
	}
	if res.Rule != "margin_utilization" {
		t.Fatalf("expected rule=margin_utilization, got %s", res.Rule)
	}
}

func TestPreCheck_NilLimits(t *testing.T) {
	t.Parallel()
	req := &CheckRequest{Symbol: "EURUSD", Positions: 100, Volume: decF(99999)}
	res := PreCheck(context.Background(), req, nil, 100, decimal.Zero, decimal.Zero)
	if !res.Allowed {
		t.Fatalf("nil limits should allow, got blocked by %s", res.Rule)
	}
}

func TestPreCheck_ZeroFreeMargin(t *testing.T) {
	t.Parallel()
	limits := &RiskLimits{MaxMarginUtilizationPct: 50}
	req := &CheckRequest{Symbol: "EURUSD"}
	// freeMargin=0 → skip margin check (avoids div-by-zero)
	res := PreCheck(context.Background(), req, limits, 0, decimal.Zero, decF(1000))
	if !res.Allowed {
		t.Fatalf("zero freeMargin should skip margin check, got blocked by %s", res.Rule)
	}
}

func TestDefaultRiskLimits(t *testing.T) {
	t.Parallel()
	d := DefaultRiskLimits()
	if d.MaxPositionsPerSymbol != 5 {
		t.Errorf("expected MaxPositionsPerSymbol=5, got %d", d.MaxPositionsPerSymbol)
	}
	if d.MaxTotalPositions != 20 {
		t.Errorf("expected MaxTotalPositions=20, got %d", d.MaxTotalPositions)
	}
	if !d.MaxExposurePerAccount.Equal(decF(100000)) {
		t.Errorf("expected MaxExposurePerAccount=100000, got %s", d.MaxExposurePerAccount.String())
	}
	if d.MaxMarginUtilizationPct != 80 {
		t.Errorf("expected MaxMarginUtilizationPct=80, got %f", d.MaxMarginUtilizationPct)
	}
}
