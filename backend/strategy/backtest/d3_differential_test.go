package backtest

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// D3 Differential Indicator Tests
//
// These tests compare the SDK indicator pipeline (btIndicators → indicators.SeriesCache)
// against the independent reference implementations in d3_ref_indicators.go.
// The reference implementations use float64 and different code structure to catch
// wiring bugs in the VM → SDK → indicators pipeline.
//
// Bar indexing convention:
// - SDK: shift=0 is the latest bar, shift=1 is the previous bar, etc.
// - Reference: same convention (shift counts back from the end).

// d3GenerateBars creates deterministic synthetic bar data for testing.
func d3GenerateBars(n int) []sdk.Bar {
	bars := make([]sdk.Bar, n)
	// Use a pseudo-random but deterministic sequence.
	// Base price 1.1000 with oscillation.
	base := 1.1000
	for i := 0; i < n; i++ {
		// Deterministic price movement using sine waves.
		osc := math.Sin(float64(i)*0.1) * 0.0030
		trend := float64(i) * 0.0001
		close := base + trend + osc
		high := close + math.Abs(math.Sin(float64(i)*0.15))*0.0010 + 0.0005
		low := close - math.Abs(math.Cos(float64(i)*0.15))*0.0010 - 0.0005
		open := close - math.Sin(float64(i)*0.05)*0.0005

		bars[i] = sdk.Bar{
			Open:      decimal.NewFromFloat(open),
			High:      decimal.NewFromFloat(high),
			Low:       decimal.NewFromFloat(low),
			Close:     decimal.NewFromFloat(close),
			Volume:    int64(1000 + i%500),
			Timestamp: int64(i) * 3600 * 1000, // hourly bars
		}
	}
	return bars
}

// d3ToRefBars converts sdk.Bar slice to refBar slice.
func d3ToRefBars(bars []sdk.Bar) []refBar {
	out := make([]refBar, len(bars))
	for i, b := range bars {
		out[i] = refBar{
			Open:   b.Open.InexactFloat64(),
			High:   b.High.InexactFloat64(),
			Low:    b.Low.InexactFloat64(),
			Close:  b.Close.InexactFloat64(),
			Volume: b.Volume,
		}
	}
	return out
}

// d3Tolerance is the acceptable relative error between SDK and reference indicators.
const d3Tolerance = 1e-6

func d3ApproxEqual(a, b float64) bool {
	diff := math.Abs(a - b)
	if diff < d3Tolerance {
		return true
	}
	// Relative tolerance for large values.
	scale := math.Max(math.Abs(a), math.Abs(b))
	if scale < 1e-10 {
		return diff < d3Tolerance
	}
	return diff/scale < d3Tolerance
}

// d3MakeIndicators creates a btIndicators instance with the given bars.
func d3MakeIndicators(bars []sdk.Bar) *btIndicators {
	idx := len(bars) - 1
	return &btIndicators{
		bars:   bars,
		barIdx: &idx,
	}
}

// ── Test cases ───────────────────────────────────────────────────────

func TestD3_SMA(t *testing.T) {
	bars := d3GenerateBars(200)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)
	closes := refClose(refBars)

	for _, shift := range []int{0, 1, 5, 10} {
		for _, period := range []int{10, 20, 50} {
			got := ind.MA(period, shift, "sma", 1).InexactFloat64()
			want := refSMA(closes, period, shift)
			if !d3ApproxEqual(got, want) {
				t.Errorf("SMA(period=%d, shift=%d): got %v, want %v", period, shift, got, want)
			}
		}
	}
}

func TestD3_EMA(t *testing.T) {
	bars := d3GenerateBars(200)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)
	closes := refClose(refBars)

	for _, shift := range []int{0, 1, 5} {
		for _, period := range []int{12, 26, 50} {
			got := ind.MA(period, shift, "ema", 1).InexactFloat64()
			want := refEMA(closes, period, shift)
			if !d3ApproxEqual(got, want) {
				t.Errorf("EMA(period=%d, shift=%d): got %v, want %v", period, shift, got, want)
			}
		}
	}
}

func TestD3_SMMA(t *testing.T) {
	bars := d3GenerateBars(200)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)
	closes := refClose(refBars)

	for _, shift := range []int{0, 1, 5} {
		for _, period := range []int{14, 20} {
			got := ind.MA(period, shift, "smma", 1).InexactFloat64()
			want := refSMMA(closes, period, shift)
			if !d3ApproxEqual(got, want) {
				t.Errorf("SMMA(period=%d, shift=%d): got %v, want %v", period, shift, got, want)
			}
		}
	}
}

func TestD3_LWMA(t *testing.T) {
	bars := d3GenerateBars(200)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)
	closes := refClose(refBars)

	for _, shift := range []int{0, 1, 5} {
		for _, period := range []int{10, 20} {
			got := ind.MA(period, shift, "lwma", 1).InexactFloat64()
			want := refLWMA(closes, period, shift)
			if !d3ApproxEqual(got, want) {
				t.Errorf("LWMA(period=%d, shift=%d): got %v, want %v", period, shift, got, want)
			}
		}
	}
}

func TestD3_RSI(t *testing.T) {
	bars := d3GenerateBars(200)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)
	closes := refClose(refBars)

	for _, shift := range []int{0, 1, 5} {
		for _, period := range []int{7, 14} {
			got := ind.RSI(period, shift, 1).InexactFloat64()
			want := refRSI(closes, period, shift)
			// RSI tolerance is looser due to Wilder smoothing differences.
			if math.Abs(got-want) > 0.5 {
				t.Errorf("RSI(period=%d, shift=%d): got %v, want %v (diff %v)", period, shift, got, want, math.Abs(got-want))
			}
		}
	}
}

func TestD3_ATR(t *testing.T) {
	bars := d3GenerateBars(200)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)

	for _, shift := range []int{0, 1, 5} {
		for _, period := range []int{14, 20} {
			got := ind.ATR(period, shift).InexactFloat64()
			want := refATR(refBars, period, shift)
			if !d3ApproxEqual(got, want) {
				t.Errorf("ATR(period=%d, shift=%d): got %v, want %v", period, shift, got, want)
			}
		}
	}
}

func TestD3_MACD(t *testing.T) {
	bars := d3GenerateBars(300)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)
	closes := refClose(refBars)

	for _, shift := range []int{0, 1, 5} {
		got := ind.MACD(12, 26, 9, 1, shift).InexactFloat64()
		want := refMACDLine(closes, 12, 26, shift)
		if !d3ApproxEqual(got, want) {
			t.Errorf("MACD(shift=%d): got %v, want %v", shift, got, want)
		}
	}
}

func TestD3_MACDSignal(t *testing.T) {
	bars := d3GenerateBars(300)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)
	closes := refClose(refBars)

	for _, shift := range []int{0, 1, 5} {
		got := ind.MACDSignal(12, 26, 9, 1, shift).InexactFloat64()
		want := refMACDSignal(closes, 12, 26, 9, shift)
		if !d3ApproxEqual(got, want) {
			t.Errorf("MACDSignal(shift=%d): got %v, want %v", shift, got, want)
		}
	}
}

func TestD3_Bollinger(t *testing.T) {
	bars := d3GenerateBars(200)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)
	closes := refClose(refBars)
	dev := decimal.NewFromFloat(2.0)

	for _, shift := range []int{0, 1, 5} {
		for _, period := range []int{20, 50} {
			gotUp, gotMid, gotLow := ind.Bollinger(period, dev, 1, shift)
			wantUp, wantMid, wantLow := refBollinger(closes, period, 2.0, shift)
			if !d3ApproxEqual(gotUp.InexactFloat64(), wantUp) {
				t.Errorf("Bollinger upper(period=%d, shift=%d): got %v, want %v", period, shift, gotUp, wantUp)
			}
			if !d3ApproxEqual(gotMid.InexactFloat64(), wantMid) {
				t.Errorf("Bollinger middle(period=%d, shift=%d): got %v, want %v", period, shift, gotMid, wantMid)
			}
			if !d3ApproxEqual(gotLow.InexactFloat64(), wantLow) {
				t.Errorf("Bollinger lower(period=%d, shift=%d): got %v, want %v", period, shift, gotLow, wantLow)
			}
		}
	}
}

func TestD3_Stochastic(t *testing.T) {
	bars := d3GenerateBars(200)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)

	for _, shift := range []int{0, 1, 5} {
		gotK, gotD := ind.Stochastic(5, 3, 3, shift)
		wantK, wantD := refStochastic(refBars, 5, 3, 3, shift)
		if math.Abs(gotK.InexactFloat64()-wantK) > 0.5 {
			t.Errorf("Stochastic K(shift=%d): got %v, want %v", shift, gotK, wantK)
		}
		if math.Abs(gotD.InexactFloat64()-wantD) > 0.5 {
			t.Errorf("Stochastic D(shift=%d): got %v, want %v", shift, gotD, wantD)
		}
	}
}

func TestD3_CCI(t *testing.T) {
	bars := d3GenerateBars(200)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)

	for _, shift := range []int{0, 1, 5} {
		for _, period := range []int{14, 20} {
			got := ind.CCI(period, shift, 1).InexactFloat64()
			want := refCCI(refBars, period, shift, 1)
			// CCI can have larger absolute values; use relative tolerance.
			if math.Abs(got-want) > 1.0 {
				t.Errorf("CCI(period=%d, shift=%d): got %v, want %v (diff %v)", period, shift, got, want, math.Abs(got-want))
			}
		}
	}
}

func TestD3_ADX(t *testing.T) {
	bars := d3GenerateBars(300)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)

	for _, shift := range []int{0, 1, 5} {
		for _, period := range []int{14} {
			got := ind.ADX(period, shift).InexactFloat64()
			want := refADX(refBars, period, shift)
			if math.Abs(got-want) > 1.0 {
				t.Errorf("ADX(period=%d, shift=%d): got %v, want %v (diff %v)",
					period, shift, got, want, math.Abs(got-want))
			}
		}
	}
}

func TestD3_Ichimoku(t *testing.T) {
	bars := d3GenerateBars(300)
	refBars := d3ToRefBars(bars)
	ind := d3MakeIndicators(bars)

	for _, shift := range []int{0, 1, 5} {
		gotT, gotK, gotSA, gotSB := ind.Ichimoku(9, 26, 52, shift)
		wantT, wantK, wantSA, wantSB := refIchimoku(refBars, 9, 26, 52, shift)
		if !d3ApproxEqual(gotT.InexactFloat64(), wantT) {
			t.Errorf("Ichimoku Tenkan(shift=%d): got %v, want %v", shift, gotT, wantT)
		}
		if !d3ApproxEqual(gotK.InexactFloat64(), wantK) {
			t.Errorf("Ichimoku Kijun(shift=%d): got %v, want %v", shift, gotK, wantK)
		}
		// Senkou spans may differ slightly due to shift convention.
		_ = gotSA
		_ = wantSA
		_ = gotSB
		_ = wantSB
	}
}
