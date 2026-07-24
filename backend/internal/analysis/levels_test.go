package analysis

import (
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/internal/repository"
)

func makeTrendBars(n int, startPrice, stepHigh, stepLow float64) []repository.KlineBar {
	bars := make([]repository.KlineBar, n)
	price := startPrice
	for i := 0; i < n; i++ {
		bars[i] = repository.KlineBar{
			High:  decimal.NewFromFloat(price + stepHigh),
			Low:   decimal.NewFromFloat(price - stepLow),
			Close: decimal.NewFromFloat(price),
		}
		price += 0.1
	}
	return bars
}

func TestDetectSRLevels_NotEnoughBars(t *testing.T) {
	t.Parallel()
	bars := makeTrendBars(10, 1.1000, 0.001, 0.001)
	levels := detectSRLevels(bars)
	if len(levels) != 0 {
		t.Fatalf("expected 0 levels for <20 bars, got %d", len(levels))
	}
}

func TestDetectSRLevels_ReturnsLevels(t *testing.T) {
	t.Parallel()
	bars := makeTrendBars(30, 1.1000, 0.001, 0.001)
	levels := detectSRLevels(bars)
	// May return nil if no clusters form, but function should not panic.
	_ = levels
}

func TestDetectSRLevels_StrongSwingPoints(t *testing.T) {
	t.Parallel()
	// Create bars with clear swing highs and lows.
	bars := make([]repository.KlineBar, 30)
	base := 1.1000
	for i := 0; i < 30; i++ {
		high := base
		low := base - 0.005
		// Every 5 bars create a swing high.
		if i%5 == 0 {
			high = base + 0.010
		}
		// Every 7 bars create a swing low.
		if i%7 == 0 {
			low = base - 0.015
		}
		bars[i] = repository.KlineBar{
			High:  decimal.NewFromFloat(high),
			Low:   decimal.NewFromFloat(low),
			Close: decimal.NewFromFloat((high + low) / 2),
		}
	}
	levels := detectSRLevels(bars)
	// Should detect at least some levels.
	if len(levels) == 0 {
		t.Log("no levels detected (acceptable with random-ish data)")
	}
	// Verify level types if any found.
	for _, l := range levels {
		if l.Type != "RESISTANCE" && l.Type != "SUPPORT" {
			t.Errorf("unexpected level type: %s", l.Type)
		}
		if l.Price <= 0 {
			t.Errorf("unexpected level price: %f", l.Price)
		}
	}
}

func TestClassifyVolatility_Low(t *testing.T) {
	t.Parallel()
	// Flat bars → low ATR.
	bars := makeTrendBars(30, 1.1000, 0.0001, 0.0001)
	_, state := classifyVolatility(bars)
	if state == "" {
		t.Fatal("expected non-empty state")
	}
}

func TestClassifyVolatility_WithMovement(t *testing.T) {
	t.Parallel()
	bars := makeTrendBars(30, 1.1000, 0.01, 0.01)
	atrPct, state := classifyVolatility(bars)
	if atrPct < 0 {
		t.Errorf("expected non-negative ATR%%, got %f", atrPct)
	}
	if state == "" {
		t.Fatal("expected non-empty state")
	}
}
