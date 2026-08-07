package strategy

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/internal/repository"
)

func TestComputeSharpeRatio(t *testing.T) {
	t.Parallel()
	returns := []float64{2.0, 1.0, 3.0, 1.5, 2.5, 1.0, 3.0, 2.0, 1.5, 2.5}
	sr := computeSharpeRatio(returns)
	if sr <= 0 {
		t.Fatalf("positive returns: want positive SR, got %.4f", sr)
	}

	negReturns := []float64{-2.0, -1.0, -3.0, -1.5, -2.5, -1.0, -3.0, -2.0, -1.5, -2.5}
	srNeg := computeSharpeRatio(negReturns)
	if srNeg >= 0 {
		t.Fatalf("negative returns: want negative SR, got %.4f", srNeg)
	}

	if computeSharpeRatio([]float64{}) != 0 {
		t.Fatal("empty returns should return 0")
	}
	if computeSharpeRatio([]float64{1.0}) != 0 {
		t.Fatal("single return should return 0")
	}
	if computeSharpeRatio([]float64{1.0, 1.0, 1.0}) != 0 {
		t.Fatal("zero variance returns should return 0")
	}
}

func TestComputeMaxDrawdown(t *testing.T) {
	t.Parallel()
	returns := []float64{100, -50, -30, 20, 50}
	dd := computeMaxDrawdown(returns)
	if dd < 0 {
		t.Fatal("max DD should be >= 0")
	}
	if math.Abs(dd-0.8) > 0.01 {
		t.Fatalf("max DD: want ~0.8, got %.2f", dd)
	}

	if computeMaxDrawdown([]float64{}) != 0 {
		t.Fatal("empty returns should return 0")
	}

	underwater := []float64{-10, -20, -30}
	if computeMaxDrawdown(underwater) != 1.0 {
		t.Fatal("underwater strategy should return 1.0")
	}
}

func TestSplitTradesMetrics_Empty(t *testing.T) {
	t.Parallel()
	isM, oosM := splitTradesMetrics(nil, nil)
	if isM == nil || oosM == nil {
		t.Fatal("metrics should not be nil")
	}
	if isM.TradeCount != 0 || oosM.TradeCount != 0 {
		t.Fatal("empty trades should have zero count")
	}
}

func TestSplitTradesMetrics_Basic(t *testing.T) {
	t.Parallel()
	trades := make([]*repository.BacktestRunTrade, 10)
	for i := range trades {
		pnl := 10.0
		if i >= 7 {
			pnl = -5.0
		}
		trades[i] = &repository.BacktestRunTrade{
			OpenTs:  int64(i) * 3600000,
			CloseTs: int64(i)*3600000 + 1800000,
			PnL:     decimal.NewFromFloat(pnl),
		}
	}
	returns := make([]float64, 20)
	for i := range returns {
		returns[i] = 1.0
	}

	isM, oosM := splitTradesMetrics(trades, returns)
	if isM.TradeCount != 7 {
		t.Fatalf("IS trade count: want 7, got %d", isM.TradeCount)
	}
	if oosM.TradeCount != 3 {
		t.Fatalf("OOS trade count: want 3, got %d", oosM.TradeCount)
	}
	if isM.NetPnl <= 0 {
		t.Fatalf("IS net PnL should be positive, got %.2f", isM.NetPnl)
	}
	if oosM.NetPnl >= 0 {
		t.Fatalf("OOS net PnL should be negative, got %.2f", oosM.NetPnl)
	}
	if !almostEqual(isM.WinRate, 100.0, 0.001) {
		t.Fatalf("IS win rate: want 100, got %.2f", isM.WinRate)
	}
	if !almostEqual(oosM.WinRate, 0.0, 0.001) {
		t.Fatalf("OOS win rate: want 0, got %.2f", oosM.WinRate)
	}
}

func TestSplitTradesMetrics_SingleTrade(t *testing.T) {
	t.Parallel()
	trades := []*repository.BacktestRunTrade{
		{OpenTs: 1000, CloseTs: 2000, PnL: decimal.NewFromFloat(50.0)},
	}
	isM, oosM := splitTradesMetrics(trades, []float64{1.0, 1.0})
	if isM.TradeCount != 0 {
		t.Fatalf("IS trade count: want 0 (too few trades for IS split), got %d", isM.TradeCount)
	}
	if oosM.TradeCount != 1 {
		t.Fatalf("OOS trade count: want 1, got %d", oosM.TradeCount)
	}
}

func TestComputeWalkForwardSegmentMetrics_PeriodBounds(t *testing.T) {
	t.Parallel()
	trades := []*repository.BacktestRunTrade{
		{OpenTs: 5000, CloseTs: 6000, PnL: decimal.NewFromFloat(10.0)},
		{OpenTs: 3000, CloseTs: 9000, PnL: decimal.NewFromFloat(20.0)},
	}
	m := computeWalkForwardSegmentMetrics(trades, []float64{1.0, 2.0, 3.0})
	if m.PeriodStartMs != 3000 {
		t.Fatalf("period start: want 3000, got %d", m.PeriodStartMs)
	}
	if m.PeriodEndMs != 9000 {
		t.Fatalf("period end: want 9000, got %d", m.PeriodEndMs)
	}
	if !almostEqual(m.NetPnl, 30.0, 0.001) {
		t.Fatalf("net pnl: want 30, got %.2f", m.NetPnl)
	}
	if !almostEqual(m.AvgTradePnl, 15.0, 0.001) {
		t.Fatalf("avg trade pnl: want 15, got %.2f", m.AvgTradePnl)
	}
}

func TestComputeWalkForwardSegmentMetrics_NoReturns(t *testing.T) {
	t.Parallel()
	trades := []*repository.BacktestRunTrade{
		{OpenTs: 1000, CloseTs: 2000, PnL: decimal.NewFromFloat(10.0)},
	}
	m := computeWalkForwardSegmentMetrics(trades, nil)
	if m.SharpeRatio != 0 {
		t.Fatalf("sharpe with no returns: want 0, got %.4f", m.SharpeRatio)
	}
	if m.MaxDrawdown != 0 {
		t.Fatalf("max DD with no returns: want 0, got %.4f", m.MaxDrawdown)
	}
}

func TestComputeWalkForwardSegmentMetrics_Empty(t *testing.T) {
	t.Parallel()
	m := computeWalkForwardSegmentMetrics(nil, nil)
	if m == nil {
		t.Fatal("metrics should not be nil")
	}
	if m.TradeCount != 0 {
		t.Fatalf("empty: want 0 trades, got %d", m.TradeCount)
	}
}

func TestSplitTradesMetrics_SplitBoundary(t *testing.T) {
	t.Parallel()
	trades := make([]*repository.BacktestRunTrade, 3)
	for i := range trades {
		trades[i] = &repository.BacktestRunTrade{
			OpenTs:  int64(i) * 1000,
			CloseTs: int64(i)*1000 + 500,
			PnL:     decimal.NewFromFloat(float64(i + 1)),
		}
	}
	isM, oosM := splitTradesMetrics(trades, []float64{1.0, 1.0, 1.0})
	if isM.TradeCount != 2 {
		t.Fatalf("IS: want 2 trades (70%% of 3), got %d", isM.TradeCount)
	}
	if oosM.TradeCount != 1 {
		t.Fatalf("OOS: want 1 trade (30%% of 3), got %d", oosM.TradeCount)
	}
}
