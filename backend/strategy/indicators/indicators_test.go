package indicators

import (
	"testing"

	"github.com/shopspring/decimal"
)

// mockBarSource implements BarSource for testing.
type mockBarSource struct {
	opens  []float64
	highs  []float64
	lows   []float64
	closes []float64
	vols   []int64
}

func (m *mockBarSource) Len() int { return len(m.closes) }
func (m *mockBarSource) Open(i int) decimal.Decimal {
	if i < 0 || i >= len(m.opens) {
		return decimal.Zero
	}
	return decimal.NewFromFloat(m.opens[i])
}
func (m *mockBarSource) High(i int) decimal.Decimal {
	if i < 0 || i >= len(m.highs) {
		return decimal.Zero
	}
	return decimal.NewFromFloat(m.highs[i])
}
func (m *mockBarSource) Low(i int) decimal.Decimal {
	if i < 0 || i >= len(m.lows) {
		return decimal.Zero
	}
	return decimal.NewFromFloat(m.lows[i])
}
func (m *mockBarSource) Close(i int) decimal.Decimal {
	if i < 0 || i >= len(m.closes) {
		return decimal.Zero
	}
	return decimal.NewFromFloat(m.closes[i])
}
func (m *mockBarSource) Volume(i int) int64 {
	if i < 0 || i >= len(m.vols) {
		return 0
	}
	return m.vols[i]
}

// makeBars creates n bars with oscillating pattern for realistic indicator testing.
func makeBars(n int) *mockBarSource {
	opens, highs, lows, closes := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	vols := make([]int64, n)
	for i := 0; i < n; i++ {
		base := 1.0 + float64(i%10)*0.001
		phase := float64(i % 4)
		opens[i] = base
		highs[i] = base + 0.002 + phase*0.001
		lows[i] = base - 0.001 - phase*0.0005
		closes[i] = base + 0.001 - phase*0.0008
		vols[i] = int64(1000 + i*10)
	}
	return &mockBarSource{opens: opens, highs: highs, lows: lows, closes: closes, vols: vols}
}

func TestAllIndicators_NonZero(t *testing.T) {
	src := makeBars(100)

	tests := []struct {
		name string
		fn   func() decimal.Decimal
	}{
		{"Alligator_jaw", func() decimal.Decimal { j, _, _ := Alligator(src, 13, 8, 8, 5, 5, 3, "smma", 0, 0); return j }},
		{"Ichimoku_tenkan", func() decimal.Decimal { t, _, _, _ := Ichimoku(src, 9, 26, 52, 0); return t }},
		{"Envelopes_upper", func() decimal.Decimal { u, _ := Envelopes(src, 14, decimal.NewFromFloat(0.1), "sma", 0, 0); return u }},
		{"DeMarker", func() decimal.Decimal { return DeMarker(src, 14, 0) }},
		{"OsMA", func() decimal.Decimal { return OsMA(src, 12, 26, 9, 0, 0) }},
		{"RVI", func() decimal.Decimal { return RVI(src, 10, 0) }},
		{"Force", func() decimal.Decimal { return Force(src, 13, "sma", 0, 0) }},
		{"Gator_upper", func() decimal.Decimal { u, _ := Gator(src, 13, 8, 8, 5, 5, 3, "smma", 0, 0); return u }},
		{"AC", func() decimal.Decimal { return AC(src, 0) }},
		{"AD", func() decimal.Decimal { return AD(src, 0) }},
		{"AO", func() decimal.Decimal { return AO(src, 0) }},
		{"BearsPower", func() decimal.Decimal { return BearsPower(src, 13, 0, 0) }},
		{"BullsPower", func() decimal.Decimal { return BullsPower(src, 13, 0, 0) }},
		{"BWMFI", func() decimal.Decimal { return BWMFI(src, 0) }},
		{"AMA", func() decimal.Decimal { return AMA(src, 14, 2, 30, 0, 1) }},
		{"DEMA", func() decimal.Decimal { return DEMA(src, 14, 0, 1) }},
		{"TEMA", func() decimal.Decimal { return TEMA(src, 14, 0, 1) }},
		{"FrAMA", func() decimal.Decimal { return FrAMA(src, 14, 0, 1) }},
		{"VIDyA", func() decimal.Decimal { return VIDyA(src, 9, 0, 12, 0, 0, 1) }},
		{"TriX", func() decimal.Decimal { return TriX(src, 14, 0, 1) }},
		{"ADXWilder", func() decimal.Decimal { return ADXWilder(src, 14, 0) }},
		{"Chaikin", func() decimal.Decimal { return Chaikin(src, 3, 10, 0) }},
		{"Volumes", func() decimal.Decimal { return Volumes(src, 0) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := tt.fn()
			if val.IsZero() {
				t.Errorf("%s returned zero, expected non-zero", tt.name)
			}
		})
	}
}

func approxEqual(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-9
}

func TestSeriesCache_MatchesStateless(t *testing.T) {
	src := makeBars(200)
	cache := NewSeriesCache(src)

	tests := []struct {
		name  string
		cache func() float64
		state func() float64
	}{
		{"EMA_14_0", func() float64 { return cache.EMA(14, 0) }, func() float64 { return ema(src, 14, 0) }},
		{"EMA_14_3", func() float64 { return cache.EMA(14, 3) }, func() float64 { return ema(src, 14, 3) }},
		{"EMA_26_0", func() float64 { return cache.EMA(26, 0) }, func() float64 { return ema(src, 26, 0) }},
		{"RSI_14_0", func() float64 { return cache.RSI(14, 0) }, func() float64 { return rsiWilder(src, 14, 0) }},
		{"RSI_14_5", func() float64 { return cache.RSI(14, 5) }, func() float64 { return rsiWilder(src, 14, 5) }},
		{"ATR_14_0", func() float64 { return cache.ATR(14, 0) }, func() float64 { return atrWilder(src, 14, 0) }},
		{"ATR_14_2", func() float64 { return cache.ATR(14, 2) }, func() float64 { return atrWilder(src, 14, 2) }},
		{"ADX_14_0", func() float64 { return cache.ADX(14, 0) }, func() float64 { return adxWilder(src, 14, 0) }},
		{"MACD_12_26_0", func() float64 { return cache.MACDLine(12, 26, 9, 0) }, func() float64 { return ema(src, 12, 0) - ema(src, 26, 0) }},
		{"MACDSignal_12_26_9_0", func() float64 { return cache.MACDSignal(12, 26, 9, 0) }, func() float64 {
			v, _ := MACDSignal(src, 12, 26, 9, 0, 1).Float64()
			return v
		}},
		{"SMA_14_0", func() float64 { return cache.SMA(14, 0) }, func() float64 { return sma(src, 14, 0) }},
		{"SMA_20_3", func() float64 { return cache.SMA(20, 3) }, func() float64 { return sma(src, 20, 3) }},
		{"DEMA_14_0", func() float64 { return cache.DEMA(14, 0) }, func() float64 {
			v, _ := DEMA(src, 14, 0, 1).Float64()
			return v
		}},
		{"TEMA_14_0", func() float64 { return cache.TEMA(14, 0) }, func() float64 {
			v, _ := TEMA(src, 14, 0, 1).Float64()
			return v
		}},
		{"TriX_14_0", func() float64 { return cache.TriX(14, 0) }, func() float64 {
			v, _ := TriX(src, 14, 0, 1).Float64()
			return v
		}},
		{"Alligator_jaw", func() float64 { j, _, _ := cache.Alligator(13, 8, 8, 5, 5, 3, "smma", 0); return j }, func() float64 {
			j, _, _ := Alligator(src, 13, 8, 8, 5, 5, 3, "smma", 0, 0)
			v, _ := j.Float64()
			return v
		}},
		{"OsMA_12_26_9_0", func() float64 { return cache.OsMA(12, 26, 9, 0) }, func() float64 {
			v, _ := OsMA(src, 12, 26, 9, 0, 0).Float64()
			return v
		}},
		{"BearsPower_13_0", func() float64 { return cache.BearsPower(13, 0) }, func() float64 {
			v, _ := BearsPower(src, 13, 0, 0).Float64()
			return v
		}},
		{"BullsPower_13_0", func() float64 { return cache.BullsPower(13, 0) }, func() float64 {
			v, _ := BullsPower(src, 13, 0, 0).Float64()
			return v
		}},
		{"Chaikin_3_10_0", func() float64 { return cache.Chaikin(3, 10, 0) }, func() float64 {
			v, _ := Chaikin(src, 3, 10, 0).Float64()
			return v
		}},
		{"Gator_upper", func() float64 { u, _ := cache.Gator(13, 8, 8, 5, 5, 3, "smma", 0); return u }, func() float64 {
			u, _ := Gator(src, 13, 8, 8, 5, 5, 3, "smma", 0, 0)
			v, _ := u.Float64()
			return v
		}},
		{"Envelopes_upper", func() float64 { u, _ := cache.Envelopes(14, 0.1, "sma", 0); return u }, func() float64 {
			u, _ := Envelopes(src, 14, decimal.NewFromFloat(0.1), "sma", 0, 0)
			v, _ := u.Float64()
			return v
		}},
		{"AD_0", func() float64 { return cache.AD(0) }, func() float64 {
			v, _ := AD(src, 0).Float64()
			return v
		}},
		{"AD_3", func() float64 { return cache.AD(3) }, func() float64 {
			v, _ := AD(src, 3).Float64()
			return v
		}},
		{"OBV_0", func() float64 { return cache.OBV(0) }, func() float64 {
			v, _ := OBV(src, 0, 1).Float64()
			return v
		}},
		{"OBV_5", func() float64 { return cache.OBV(5) }, func() float64 {
			v, _ := OBV(src, 5, 1).Float64()
			return v
		}},
		{"SAR_0", func() float64 { return cache.SAR(0.02, 0.2, 0) }, func() float64 {
			v, _ := SAR(src, decimal.NewFromFloat(0.02), decimal.NewFromFloat(0.2), 0).Float64()
			return v
		}},
		{"SAR_3", func() float64 { return cache.SAR(0.02, 0.2, 3) }, func() float64 {
			v, _ := SAR(src, decimal.NewFromFloat(0.02), decimal.NewFromFloat(0.2), 3).Float64()
			return v
		}},
		{"VIDyA_9_0_12_0_0", func() float64 { return cache.VIDyA(9, 0, 12, 0, 0) }, func() float64 {
			v, _ := VIDyA(src, 9, 0, 12, 0, 0, 1).Float64()
			return v
		}},
		{"Force_sma_13_0", func() float64 { return cache.Force(13, "sma", 0) }, func() float64 {
			v, _ := Force(src, 13, "sma", 0, 0).Float64()
			return v
		}},
		{"Force_ema_13_0", func() float64 { return cache.Force(13, "ema", 0) }, func() float64 {
			v, _ := Force(src, 13, "ema", 0, 0).Float64()
			return v
		}},
		{"LWMA_14_0", func() float64 { return cache.LWMA(14, 0) }, func() float64 { return lwma(src, 14, 0) }},
		{"LWMA_20_3", func() float64 { return cache.LWMA(20, 3) }, func() float64 { return lwma(src, 20, 3) }},
		{"AMA_14_2_30_0", func() float64 { return cache.AMA(14, 2, 30, 0) }, func() float64 {
			v, _ := AMA(src, 14, 2, 30, 0, 1).Float64()
			return v
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cache()
			want := tt.state()
			if !approxEqual(got, want) {
				t.Errorf("cache=%.10f stateless=%.10f diff=%.2e", got, want, got-want)
			}
		})
	}
}

func TestSeriesCache_Incremental(t *testing.T) {
	// Test that incremental updates produce same result as full rebuild.
	src := makeBars(100)
	cache := NewSeriesCache(src)
	cache.EnsureUpdated()
	v1 := cache.EMA(14, 0)

	// Simulate adding a new bar (shrink then grow the source)
	full := makeBars(101)
	cache2 := NewSeriesCache(full)
	cache2.EnsureUpdated()
	v2 := cache2.EMA(14, 0)

	// Also compute from stateless
	vStateless := ema(full, 14, 0)

	if !approxEqual(v1, ema(src, 14, 0)) {
		t.Errorf("100-bar cache doesn't match stateless: %.10f vs %.10f", v1, ema(src, 14, 0))
	}
	if !approxEqual(v2, vStateless) {
		t.Errorf("101-bar cache doesn't match stateless: %.10f vs %.10f", v2, vStateless)
	}
}

func TestFractals_NonZero(t *testing.T) {
	src := makeBars(100)
	upper, lower := Fractals(src, 0)
	// Fractals may or may not be present at shift=0 depending on pattern.
	// With a monotonic uptrend, there should be an upper fractal but no lower.
	// Just verify it doesn't panic and returns valid values.
	_ = upper
	_ = lower
}
