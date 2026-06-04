// Package ai — E4 Paper Gate tests.
package ai

import "testing"

func TestPaperGate_Passing(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{
		PaperDays:         14,
		BacktestNetReturn: 0.10,
		PaperNetReturn:    0.08,
		PaperNetPnL:       5000,
		PaperTradeCount:   20,
	}
	cfg := DefaultPaperGateConfig()
	result := PaperGate(metrics, cfg)
	if !result.Passed {
		t.Fatalf("should pass: %s", result.Reason)
	}
}

func TestPaperGate_InsufficientDays(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{
		PaperDays:      7,
		PaperNetReturn: 0.05,
		PaperNetPnL:    1000,
		PaperTradeCount: 10,
	}
	cfg := DefaultPaperGateConfig()
	result := PaperGate(metrics, cfg)
	if result.Passed {
		t.Fatal("insufficient paper days should not pass")
	}
}

func TestPaperGate_NegativePnL(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{
		PaperDays:      14,
		PaperNetPnL:    -500,
		PaperTradeCount: 10,
	}
	cfg := DefaultPaperGateConfig()
	result := PaperGate(metrics, cfg)
	if result.Passed {
		t.Fatal("negative P&L should not pass")
	}
}

func TestPaperGate_RegimeFail(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{
		PaperDays:         14,
		BacktestNetReturn: 0.20,
		PaperNetReturn:    0.05,
		PaperNetPnL:       1000,
		PaperTradeCount:   10,
	}
	cfg := DefaultPaperGateConfig()
	result := PaperGate(metrics, cfg)
	if result.Passed {
		t.Fatal("regime fail (paper return < 50% of backtest) should not pass")
	}
}

func TestPaperGate_TooFewTrades(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{
		PaperDays:      14,
		PaperNetPnL:    1000,
		PaperTradeCount: 3,
	}
	cfg := DefaultPaperGateConfig()
	result := PaperGate(metrics, cfg)
	if result.Passed {
		t.Fatal("too few paper trades should not pass")
	}
}
