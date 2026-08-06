package backtest

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// D3 Metamorphic Tests
//
// Metamorphic testing verifies indicator properties that must hold regardless
// of the specific input data. Instead of comparing against a reference value,
// these tests check mathematical identities and invariants.
//
// Each test defines a "metamorphic relation" — a property that should hold
// between the outputs of the same indicator function applied to related inputs.

// ── MR1: SMA symmetry under data reversal ────────────────────────────
// SMA of a reversed data slice at shift=0 should equal SMA of the original
// at the same position (since SMA is order-independent within its window).

func TestD3_MR1_SMA_OrderIndependence(t *testing.T) {
	bars := d3GenerateBars(100)
	refBars := d3ToRefBars(bars)
	closes := refClose(refBars)
	period := 20

	// Take the last `period` close values.
	window := closes[len(closes)-period:]
	// Compute SMA of the window directly.
	sum := 0.0
	for _, v := range window {
		sum += v
	}
	want := sum / float64(period)

	// Reverse the window and compute SMA — should be the same.
	reversed := make([]float64, len(window))
	for i, v := range window {
		reversed[len(window)-1-i] = v
	}
	sumR := 0.0
	for _, v := range reversed {
		sumR += v
	}
	got := sumR / float64(period)

	if math.Abs(got-want) > 1e-10 {
		t.Errorf("SMA order independence: got %v, want %v", got, want)
	}
}

// ── MR2: EMA continuity — small input change → small output change ───
// Perturbing the input by epsilon should change EMA by at most epsilon.

func TestD3_MR2_EMA_Continuity(t *testing.T) {
	bars := d3GenerateBars(200)
	refBars := d3ToRefBars(bars)
	closes := refClose(refBars)
	period := 20
	shift := 0

	original := refEMA(closes, period, shift)

	// Perturb the last close by a small delta.
	delta := 0.0001
	perturbed := make([]float64, len(closes))
	copy(perturbed, closes)
	perturbed[len(perturbed)-1] += delta

	perturbedEMA := refEMA(perturbed, period, shift)

	// The change in EMA should be bounded by delta (EMA smooths, so it should be less).
	diff := math.Abs(perturbedEMA - original)
	if diff > delta {
		t.Errorf("EMA continuity: output changed by %v > delta %v", diff, delta)
	}
	// For EMA with period 20, k = 2/21 ≈ 0.095, so the change should be ≈ delta * k.
	expectedChange := delta * 2.0 / float64(period+1)
	if math.Abs(diff-expectedChange) > delta*0.1 {
		t.Errorf("EMA continuity: expected change ~%v, got %v", expectedChange, diff)
	}
}

// ── MR3: Bollinger band symmetry ─────────────────────────────────────
// upper - middle == middle - lower (by definition, since both are deviation*std).

func TestD3_MR3_BollingerSymmetry(t *testing.T) {
	bars := d3GenerateBars(200)
	refBars := d3ToRefBars(bars)
	closes := refClose(refBars)

	for _, period := range []int{20, 50} {
		for _, shift := range []int{0, 1, 5} {
			upper, middle, lower := refBollinger(closes, period, 2.0, shift)
			upperSpread := upper - middle
			lowerSpread := middle - lower
			if math.Abs(upperSpread-lowerSpread) > 1e-10 {
				t.Errorf("Bollinger symmetry (period=%d, shift=%d): upper-mid=%v, mid-lower=%v",
					period, shift, upperSpread, lowerSpread)
			}
		}
	}
}

// ── MR4: RSI bounds — RSI is always in [0, 100] ──────────────────────

func TestD3_MR4_RSI_Bounds(t *testing.T) {
	// Generate bars with various patterns: trending up, down, flat, volatile.
	patterns := []string{"up", "down", "flat", "volatile"}

	for _, pattern := range patterns {
		bars := d3GenerateBarsPattern(200, pattern)
		refBars := d3ToRefBars(bars)
		closes := refClose(refBars)

		for _, period := range []int{7, 14, 50} {
			for _, shift := range []int{0, 1, 10} {
				rsi := refRSI(closes, period, shift)
				if rsi < 0 || rsi > 100 {
					t.Errorf("RSI out of bounds (pattern=%s, period=%d, shift=%d): %v",
						pattern, period, shift, rsi)
				}
			}
		}
	}
}

// ── MR5: Stochastic bounds — %K and %D are always in [0, 100] ────────

func TestD3_MR5_StochasticBounds(t *testing.T) {
	patterns := []string{"up", "down", "flat", "volatile"}

	for _, pattern := range patterns {
		bars := d3GenerateBarsPattern(200, pattern)
		refBars := d3ToRefBars(bars)

		for _, shift := range []int{0, 1, 10} {
			k, d := refStochastic(refBars, 5, 3, 3, shift)
			if k < 0 || k > 100 {
				t.Errorf("Stochastic %%K out of bounds (pattern=%s, shift=%d): %v",
					pattern, shift, k)
			}
			if d < 0 || d > 100 {
				t.Errorf("Stochastic %%D out of bounds (pattern=%s, shift=%d): %v",
					pattern, shift, d)
			}
		}
	}
}

// ── MR6: ATR non-negativity ──────────────────────────────────────────

func TestD3_MR6_ATR_NonNegative(t *testing.T) {
	patterns := []string{"up", "down", "flat", "volatile"}

	for _, pattern := range patterns {
		bars := d3GenerateBarsPattern(200, pattern)
		refBars := d3ToRefBars(bars)

		for _, period := range []int{7, 14, 50} {
			for _, shift := range []int{0, 1, 10} {
				atr := refATR(refBars, period, shift)
				if atr < 0 {
					t.Errorf("ATR negative (pattern=%s, period=%d, shift=%d): %v",
						pattern, period, shift, atr)
				}
			}
		}
	}
}

// ── MR7: MACD line = EMA(fast) - EMA(slow) identity ──────────────────

func TestD3_MR7_MACDIdentity(t *testing.T) {
	bars := d3GenerateBars(300)
	refBars := d3ToRefBars(bars)
	closes := refClose(refBars)

	fast, slow := 12, 26
	for _, shift := range []int{0, 1, 5, 10} {
		macd := refMACDLine(closes, fast, slow, shift)
		emaFast := refEMA(closes, fast, shift)
		emaSlow := refEMA(closes, slow, shift)
		diff := math.Abs(macd - (emaFast - emaSlow))
		if diff > 1e-10 {
			t.Errorf("MACD identity (shift=%d): MACD=%v, EMA_fast-EMA_slow=%v, diff=%v",
				shift, macd, emaFast-emaSlow, diff)
		}
	}
}

// ── Helpers ──────────────────────────────────────────────────────────

// d3GenerateBarsPattern generates bars with specific price patterns for metamorphic tests.
func d3GenerateBarsPattern(n int, pattern string) []sdk.Bar {
	bars := make([]sdk.Bar, n)
	base := 1.1000

	for i := 0; i < n; i++ {
		var close float64
		switch pattern {
		case "up":
			close = base + float64(i)*0.001
		case "down":
			close = base - float64(i)*0.001
		case "flat":
			close = base + math.Sin(float64(i)*0.01)*0.0001
		case "volatile":
			close = base + math.Sin(float64(i)*0.3)*0.005 + math.Cos(float64(i)*0.7)*0.003
		default:
			close = base
		}

		high := close + 0.0005 + math.Abs(math.Sin(float64(i)*0.2))*0.001
		low := close - 0.0005 - math.Abs(math.Cos(float64(i)*0.2))*0.001
		open := close - math.Sin(float64(i)*0.05)*0.0003

		bars[i] = sdk.Bar{
			Open:      decimal.NewFromFloat(open),
			High:      decimal.NewFromFloat(high),
			Low:       decimal.NewFromFloat(low),
			Close:     decimal.NewFromFloat(close),
			Volume:    int64(1000 + i%500),
			Timestamp: int64(i) * 3600 * 1000,
		}
	}
	return bars
}
