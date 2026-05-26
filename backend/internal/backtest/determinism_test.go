package backtest

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"anttrader/internal/clock"
)

// trendStrategy goes long when price trends up, flat otherwise.
func trendStrategy(bar Bar, pos float64) (int, float64) {
	if bar.Close > bar.Open && pos == 0 {
		return 1, 0.1
	}
	if bar.Close < bar.Open && pos > 0 {
		return -1, pos
	}
	return 0, 0
}

// computeResultHash produces a deterministic hash of backtest results.
func computeResultHash(m *Metrics) string {
	input := fmt.Sprintf("%.6f;%.6f;%.6f;%.4f;%.4f;%d",
		m.TotalReturn, m.SharpeRatio, m.MaxDrawdown,
		m.WinRate, m.ProfitFactor, m.TotalTrades)
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h[:16])
}

func TestBacktestDeterminism(t *testing.T) {
	// Create a deterministic bar dataset.
	bars := make([]Bar, 100)
	price := 1.0850
	for i := range bars {
		ts := int64(1_700_000_000 + i*3600)
		open := price
		close_ := price + float64(i%5-2)*0.0002
		high := max(open, close_) + 0.0001
		low := min(open, close_) - 0.0001
		bars[i] = Bar{
			OpenTime:  ts,
			CloseTime: ts + 3599,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close_,
			Volume:    100,
		}
		price = close_
	}

	fm := &FillModel{
		SlippagePips:    0.5,
		ContractSize:    100000,
		PartialFillProb: 0,
	}

	// Run 1.
	eng1 := NewEngine(bars, 10000, fm)
	result1 := eng1.Run(trendStrategy)
	hash1 := computeResultHash(result1)

	// Run 2 — same inputs, same seed (42), should produce identical results.
	eng2 := NewEngine(bars, 10000, fm)
	result2 := eng2.Run(trendStrategy)
	hash2 := computeResultHash(result2)

	if hash1 != hash2 {
		t.Fatalf("determinism violation: run1=%s run2=%s", hash1, hash2)
	}

	t.Logf("Determinism verified: hash=%s trades=%d return=%.4f", hash1, result1.TotalTrades, result1.TotalReturn)

	// Verify simulated clock determinism.
	startTime := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	simClk1 := clock.NewSimulatedClock(startTime)
	simClk2 := clock.NewSimulatedClock(startTime)

	if !simClk1.Now().Equal(simClk2.Now()) {
		t.Fatal("simulated clocks at same start should have same Now()")
	}

	simClk1.Sleep(time.Hour)
	simClk2.Sleep(time.Hour)
	if !simClk1.Now().Equal(simClk2.Now()) {
		t.Fatal("simulated clocks after same Sleep should have same Now()")
	}
}
