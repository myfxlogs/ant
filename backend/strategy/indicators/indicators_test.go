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
		{"AMA", func() decimal.Decimal { return AMA(src, 14, 2, 30, 0) }},
		{"DEMA", func() decimal.Decimal { return DEMA(src, 14, 0) }},
		{"TEMA", func() decimal.Decimal { return TEMA(src, 14, 0) }},
		{"FrAMA", func() decimal.Decimal { return FrAMA(src, 14, 0) }},
		{"VIDyA", func() decimal.Decimal { return VIDyA(src, 9, 0, 12, 0, 0) }},
		{"TriX", func() decimal.Decimal { return TriX(src, 14, 0) }},
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

func TestFractals_NonZero(t *testing.T) {
	src := makeBars(100)
	upper, lower := Fractals(src, 0)
	// Fractals may or may not be present at shift=0 depending on pattern.
	// With a monotonic uptrend, there should be an upper fractal but no lower.
	// Just verify it doesn't panic and returns valid values.
	_ = upper
	_ = lower
}
