package risksvc

import (
	"context"
	"testing"
)

func TestKellySizer_PositiveEdge(t *testing.T) {
	t.Parallel()
	// p=0.6, b=2.0 → f* = (0.6*2 - 0.4) / 2 = 0.8/2 = 0.4
	// half-Kelly = 0.2, cap at 0.25, so f=0.2
	s := &KellyFractionSizer{
		WinProb:      0.6,
		WinLossRatio: 2.0,
		Fraction:     0.5,
		KellyMax:     0.25,
	}
	req := &SizerRequest{
		Symbol: "EURUSD", Price: 1.0850, ContractSize: 100000, Equity: 100000,
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// riskCapital = 100000 * 0.2 = 20000
	// lots = 20000 / (1.085 * 100000) = 0.1843
	if res.Lots < 0.15 || res.Lots > 0.22 {
		t.Fatalf("lots should be ~0.184, got %.4f", res.Lots)
	}
	if res.RiskUsed < 0.19 || res.RiskUsed > 0.21 {
		t.Fatalf("risk used should be ~0.20, got %.4f", res.RiskUsed)
	}
	t.Logf("Positive edge: lots=%.4f risk_used=%.4f", res.Lots, res.RiskUsed)
}

func TestKellySizer_NegativeEdge(t *testing.T) {
	t.Parallel()
	// p=0.4, b=1.0 → f* = (0.4*1 - 0.6) / 1 = -0.2 → no bet
	s := &KellyFractionSizer{
		WinProb:      0.4,
		WinLossRatio: 1.0,
	}
	req := &SizerRequest{Price: 1.0850, Equity: 100000}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Lots != 0 {
		t.Fatalf("negative edge should give zero lots, got %.4f", res.Lots)
	}
}

func TestKellySizer_ZeroEdge(t *testing.T) {
	t.Parallel()
	// p=0.5, b=1.0 → f* = (0.5*1 - 0.5) / 1 = 0 → no bet
	s := &KellyFractionSizer{
		WinProb:      0.5,
		WinLossRatio: 1.0,
	}
	req := &SizerRequest{Price: 1.0850, Equity: 100000}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Lots != 0 {
		t.Fatalf("zero edge should give zero lots, got %.4f", res.Lots)
	}
}

func TestKellySizer_MaxCap(t *testing.T) {
	t.Parallel()
	// p=0.9, b=5.0 → f* = (0.9*5 - 0.1) / 5 = 4.4/5 = 0.88
	// half-Kelly = 0.44, but KellyMax cap at 0.25
	s := &KellyFractionSizer{
		WinProb:      0.9,
		WinLossRatio: 5.0,
		Fraction:     0.5,
		KellyMax:     0.25,
		MaxLots:      100,
	}
	req := &SizerRequest{Price: 1.0850, Equity: 100000}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RiskUsed > 0.251 {
		t.Fatalf("risk used should be capped at 0.25, got %.4f", res.RiskUsed)
	}
	t.Logf("KellyMax cap: risk_used=%.4f", res.RiskUsed)
}

func TestKellySizer_DefaultFraction(t *testing.T) {
	t.Parallel()
	s := &KellyFractionSizer{
		WinProb:      0.6,
		WinLossRatio: 2.0,
	}
	req := &SizerRequest{Price: 1.0850, ContractSize: 100000, Equity: 100000}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default half-Kelly: f* = 0.4, half = 0.2, capped at 0.25 → 0.2
	if res.RiskUsed < 0.19 || res.RiskUsed > 0.21 {
		t.Fatalf("default half-Kelly should use ~0.20 risk, got %.4f", res.RiskUsed)
	}
}

func TestKellySizer_InvalidWinProb(t *testing.T) {
	t.Parallel()
	s := &KellyFractionSizer{WinProb: 0, WinLossRatio: 2.0}
	req := &SizerRequest{Price: 1.0850, Equity: 100000}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Lots != 0 {
		t.Fatalf("zero win prob should give zero lots")
	}
}

func TestComputeKellyMetrics(t *testing.T) {
	t.Parallel()
	km := ComputeKellyMetrics(0.6, 2.0)
	// f* = (0.6*2 - 0.4)/2 = 0.8/2 = 0.4
	if km.FStar < 0.39 || km.FStar > 0.41 {
		t.Fatalf("FStar should be 0.4, got %.4f", km.FStar)
	}
	if km.HalfKelly < 0.19 || km.HalfKelly > 0.21 {
		t.Fatalf("HalfKelly should be 0.2, got %.4f", km.HalfKelly)
	}
	if !km.IsPositive {
		t.Fatal("edge should be positive")
	}

	// Negative case
	km2 := ComputeKellyMetrics(0.3, 1.0)
	if km2.IsPositive {
		t.Fatal("edge should be negative")
	}
	if km2.FStar != 0 {
		t.Fatalf("negative edge FStar should be 0, got %.4f", km2.FStar)
	}
}
