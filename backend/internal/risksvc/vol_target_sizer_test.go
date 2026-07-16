package risksvc

import (
	"context"
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

func TestVolTargetSizer_EURUSD(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 0.01, MaxLots: decF(100)}
	req := &SizerRequest{
		Symbol:       "EURUSD",
		Price:        decF(1.0850),
		ATR:          decF(0.0035),   // 35 pips daily ATR
		ContractSize: decF(100000),   // standard forex lot
		HoldingDays:  5,
		Equity:       decF(100000),
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expected: 1000 / (0.0035 * 100000 * sqrt(5)) = 1000 / 782.6 = 1.28
	if res.Lots.LessThan(decF(1.0)) || res.Lots.GreaterThan(decF(1.5)) {
		t.Fatalf("EURUSD lots should be ~1.28, got %s", res.Lots.String())
	}
	if res.RiskUsed <= 0 || res.RiskUsed > 0.02 {
		t.Fatalf("risk used should be ~1%%, got %.4f", res.RiskUsed)
	}
	t.Logf("EURUSD: lots=%s risk_used=%.4f%%", res.Lots.String(), res.RiskUsed*100)
}

func TestVolTargetSizer_BTCUSD(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 0.01, MaxLots: decF(100)}
	req := &SizerRequest{
		Symbol:       "BTCUSD",
		Price:        decF(50000),
		ATR:          decF(2000),    // $2000 daily ATR
		ContractSize: decF(1),      // spot-like
		HoldingDays:  5,
		Equity:       decF(100000),
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expected: 1000 / (2000 * 1 * sqrt(5)) = 1000 / 4472.1 = 0.2236
	if res.Lots.LessThan(decF(0.15)) || res.Lots.GreaterThan(decF(0.35)) {
		t.Fatalf("BTCUSD lots should be ~0.22, got %s", res.Lots.String())
	}
	t.Logf("BTCUSD: lots=%s risk_used=%.4f%%", res.Lots.String(), res.RiskUsed*100)
}

func TestVolTargetSizer_EURUSDvsBTCUSD_Ratio(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 0.01, MaxLots: decF(100)}
	eur := &SizerRequest{
		Symbol: "EURUSD", Price: decF(1.0850), ATR: decF(0.0035), ContractSize: decF(100000), HoldingDays: 5, Equity: decF(100000),
	}
	btc := &SizerRequest{
		Symbol: "BTCUSD", Price: decF(50000), ATR: decF(2000), ContractSize: decF(1), HoldingDays: 5, Equity: decF(100000),
	}
	eurRes, _ := s.Size(context.Background(), eur)
	btcRes, _ := s.Size(context.Background(), btc)

	ratio := eurRes.Lots.Div(btcRes.Lots).InexactFloat64()
	if ratio < 5 || ratio > 10 {
		t.Fatalf("EURUSD/BTCUSD lot ratio should be 5-10×, got %.2f", ratio)
	}
	t.Logf("EURUSD=%s lots, BTCUSD=%s lots, ratio=%.2f×", eurRes.Lots.String(), btcRes.Lots.String(), ratio)
}

func TestVolTargetSizer_ZeroEquity(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 0.01}
	req := &SizerRequest{Equity: decimal.Zero, Price: decF(1.0850), ATR: decF(0.0035)}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Lots.Equal(decimal.Zero) {
		t.Fatalf("zero equity should give zero lots, got %s", res.Lots.String())
	}
}

func TestVolTargetSizer_MaxLotsCap(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 1.0, MaxLots: decF(0.5)} // 100% risk budget → huge lot
	req := &SizerRequest{
		Symbol: "EURUSD", Price: decF(1.0850), ATR: decF(0.0001), ContractSize: decF(100000), HoldingDays: 1, Equity: decF(100000),
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Lots.GreaterThan(decF(0.5)) {
		t.Fatalf("lots should be capped at 0.5, got %s", res.Lots.String())
	}
}

func TestVolTargetSizer_MinLotsFloor(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 0.001, MinLots: decF(0.1)} // tiny risk budget
	req := &SizerRequest{
		Symbol: "EURUSD", Price: decF(1.0850), ATR: decF(0.01), ContractSize: decF(100000), HoldingDays: 10, Equity: decF(10000),
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Lots.Equal(decimal.Zero) {
		t.Fatalf("below min lots should give zero, got %s", res.Lots.String())
	}
}

func TestVolTargetSizer_DefaultRiskBudget(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{} // RiskBudgetPct defaults to 0.01
	req := &SizerRequest{
		Symbol: "EURUSD", Price: decF(1.0850), ATR: decF(0.0035), ContractSize: decF(100000), HoldingDays: 5, Equity: decF(100000),
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Lots.LessThanOrEqual(decimal.Zero) {
		t.Fatal("default risk budget should produce non-zero lots")
	}
}

func TestVolTargetSizer_FallbackATR(t *testing.T) {
	t.Parallel()
	s := &VolTargetSizer{RiskBudgetPct: 0.01}
	req := &SizerRequest{
		Symbol: "EURUSD", Price: decF(1.0850), ATR: decimal.Zero, AnnualVol: 0.15, ContractSize: decF(100000), HoldingDays: 5, Equity: decF(100000),
	}
	res, err := s.Size(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lotsFloat := res.Lots.InexactFloat64()
	if math.IsNaN(lotsFloat) || res.Lots.LessThanOrEqual(decimal.Zero) {
		t.Fatalf("fallback ATR should produce valid lots, got %s", res.Lots.String())
	}
	t.Logf("Fallback ATR: lots=%s", res.Lots.String())
}
