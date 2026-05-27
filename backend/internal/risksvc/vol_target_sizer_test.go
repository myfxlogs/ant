package risksvc

import (
	"context"
	"math"
	"testing"
)

func TestVolTargetSizer_EURUSD(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 0.01, MaxLots: 100}
	req := &SizerRequest{
		Symbol:       "EURUSD",
		Price:        1.0850,
		ATR:          0.0035,   // 35 pips daily ATR
		ContractSize: 100000,   // standard forex lot
		HoldingDays:  5,
		Equity:       100000,
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expected: 1000 / (0.0035 * 100000 * sqrt(5)) = 1000 / 782.6 = 1.28
	if res.Lots < 1.0 || res.Lots > 1.5 {
		t.Fatalf("EURUSD lots should be ~1.28, got %.4f", res.Lots)
	}
	if res.RiskUsed <= 0 || res.RiskUsed > 0.02 {
		t.Fatalf("risk used should be ~1%%, got %.4f", res.RiskUsed)
	}
	t.Logf("EURUSD: lots=%.4f risk_used=%.4f%%", res.Lots, res.RiskUsed*100)
}

func TestVolTargetSizer_BTCUSD(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 0.01, MaxLots: 100}
	req := &SizerRequest{
		Symbol:       "BTCUSD",
		Price:        50000,
		ATR:          2000,    // $2000 daily ATR
		ContractSize: 1,      // spot-like
		HoldingDays:  5,
		Equity:       100000,
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expected: 1000 / (2000 * 1 * sqrt(5)) = 1000 / 4472.1 = 0.2236
	if res.Lots < 0.15 || res.Lots > 0.35 {
		t.Fatalf("BTCUSD lots should be ~0.22, got %.4f", res.Lots)
	}
	t.Logf("BTCUSD: lots=%.4f risk_used=%.4f%%", res.Lots, res.RiskUsed*100)
}

func TestVolTargetSizer_EURUSDvsBTCUSD_Ratio(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 0.01, MaxLots: 100}
	eur := &SizerRequest{
		Symbol: "EURUSD", Price: 1.0850, ATR: 0.0035, ContractSize: 100000, HoldingDays: 5, Equity: 100000,
	}
	btc := &SizerRequest{
		Symbol: "BTCUSD", Price: 50000, ATR: 2000, ContractSize: 1, HoldingDays: 5, Equity: 100000,
	}
	eurRes, _ := s.Size(context.Background(), eur)
	btcRes, _ := s.Size(context.Background(), btc)

	ratio := eurRes.Lots / btcRes.Lots
	if ratio < 5 || ratio > 10 {
		t.Fatalf("EURUSD/BTCUSD lot ratio should be 5-10×, got %.2f", ratio)
	}
	t.Logf("EURUSD=%.4f lots, BTCUSD=%.4f lots, ratio=%.2f×", eurRes.Lots, btcRes.Lots, ratio)
}

func TestVolTargetSizer_ZeroEquity(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 0.01}
	req := &SizerRequest{Equity: 0, Price: 1.0850, ATR: 0.0035}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Lots != 0 {
		t.Fatalf("zero equity should give zero lots, got %.4f", res.Lots)
	}
}

func TestVolTargetSizer_MaxLotsCap(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 1.0, MaxLots: 0.5} // 100% risk budget → huge lot
	req := &SizerRequest{
		Symbol: "EURUSD", Price: 1.0850, ATR: 0.0001, ContractSize: 100000, HoldingDays: 1, Equity: 100000,
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Lots > 0.5 {
		t.Fatalf("lots should be capped at 0.5, got %.4f", res.Lots)
	}
}

func TestVolTargetSizer_MinLotsFloor(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 0.001, MinLots: 0.1} // tiny risk budget
	req := &SizerRequest{
		Symbol: "EURUSD", Price: 1.0850, ATR: 0.01, ContractSize: 100000, HoldingDays: 10, Equity: 10000,
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Lots != 0 {
		t.Fatalf("below min lots should give zero, got %.4f", res.Lots)
	}
}

func TestVolTargetSizer_DefaultRiskBudget(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{} // RiskBudgetPct defaults to 0.01
	req := &SizerRequest{
		Symbol: "EURUSD", Price: 1.0850, ATR: 0.0035, ContractSize: 100000, HoldingDays: 5, Equity: 100000,
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Lots <= 0 {
		t.Fatal("default risk budget should produce non-zero lots")
	}
}

func TestVolTargetSizer_FallbackATR(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 0.01}
	req := &SizerRequest{
		Symbol: "EURUSD", Price: 1.0850, ATR: 0, AnnualVol: 0.15, ContractSize: 100000, HoldingDays: 5, Equity: 100000,
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.IsNaN(res.Lots) || res.Lots <= 0 {
		t.Fatalf("fallback ATR should produce valid lots, got %.4f", res.Lots)
	}
	t.Logf("Fallback ATR: lots=%.4f", res.Lots)
}
