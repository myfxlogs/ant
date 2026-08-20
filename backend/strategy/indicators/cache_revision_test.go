package indicators

import (
	"testing"

	"github.com/shopspring/decimal"
)

// mockRevisionedBarSource extends mockBarSource with a revision counter,
// implementing RevisionedBarSource. Used to test SeriesCache invalidation
// on rolling-window content changes at constant Len().
type mockRevisionedBarSource struct {
	mockBarSource
	rev uint64
}

func (m *mockRevisionedBarSource) Revision() uint64 { return m.rev }

// makeRevisionedBars creates n bars with an oscillating pattern and wraps
// them in a mockRevisionedBarSource at revision 1.
func makeRevisionedBars(n int) *mockRevisionedBarSource {
	inner := makeBars(n)
	return &mockRevisionedBarSource{
		mockBarSource: *inner,
		rev:           1,
	}
}

// rollWindow drops the oldest bar (highest index in BarSource = last element
// of the underlying closes slice) and appends a newest bar (index 0 = first
// element), keeping length constant at n. Increments revision.
func rollWindow(src *mockRevisionedBarSource, newClose float64) {
	n := len(src.closes)
	// Drop oldest (last element), prepend newest (first element).
	src.closes = append([]float64{newClose}, src.closes[:n-1]...)
	src.opens = append([]float64{newClose}, src.opens[:n-1]...)
	src.highs = append([]float64{newClose + 0.002}, src.highs[:n-1]...)
	src.lows = append([]float64{newClose - 0.001}, src.lows[:n-1]...)
	src.vols = append([]int64{1000}, src.vols[:n-1]...)
	src.rev++
}

// allCacheBackedIndicators returns a table of cache-backed indicator queries
// and their stateless equivalents, for table-driven comparison after mutation.
func allCacheBackedIndicators(cache *SeriesCache, src BarSource) []struct {
	name  string
	cache func() float64
	state func() float64
} {
	return []struct {
		name  string
		cache func() float64
		state func() float64
	}{
		{"EMA_14_0", func() float64 { return cache.EMA(14, 0) }, func() float64 { return ema(src, 14, 0) }},
		{"EMA_26_0", func() float64 { return cache.EMA(26, 0) }, func() float64 { return ema(src, 26, 0) }},
		{"SMMA_14_0", func() float64 { return cache.SMMA(14, 0) }, func() float64 { return smma(src, 14, 0) }},
		{"SMA_14_0", func() float64 { return cache.SMA(14, 0) }, func() float64 { return sma(src, 14, 0) }},
		{"LWMA_14_0", func() float64 { return cache.LWMA(14, 0) }, func() float64 { return lwma(src, 14, 0) }},
		{"RSI_14_0", func() float64 { return cache.RSI(14, 0) }, func() float64 { return rsiWilder(src, 14, 0) }},
		{"ATR_14_0", func() float64 { return cache.ATR(14, 0) }, func() float64 { return atrWilder(src, 14, 0) }},
		{"ADX_14_0", func() float64 { return cache.ADX(14, 0) }, func() float64 { return adxWilder(src, 14, 0) }},
		{"MACD_12_26_9_0", func() float64 { return cache.MACDLine(12, 26, 9, 0) }, func() float64 { return ema(src, 12, 0) - ema(src, 26, 0) }},
		{"MACDSignal_12_26_9_0", func() float64 { return cache.MACDSignal(12, 26, 9, 0) }, func() float64 {
			v, _ := MACDSignal(src, 12, 26, 9, 0, 1).Float64()
			return v
		}},
		{"DEMA_14_0", func() float64 { return cache.DEMA(14, 0) }, func() float64 {
			v, _ := DEMA(src, 14, 0, 1).Float64()
			return v
		}},
		{"TEMA_14_0", func() float64 { return cache.TEMA(14, 0) }, func() float64 {
			v, _ := TEMA(src, 14, 0, 1).Float64()
			return v
		}},
		{"Alligator_jaw", func() float64 { j, _, _ := cache.Alligator(13, 8, 8, 5, 5, 3, "smma", 0); return j }, func() float64 {
			j, _, _ := Alligator(src, 13, 8, 8, 5, 5, 3, "smma", 0, 0)
			v, _ := j.Float64()
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
		{"OBV_0", func() float64 { return cache.OBV(0) }, func() float64 {
			v, _ := OBV(src, 0, 1).Float64()
			return v
		}},
		{"SAR_0", func() float64 { return cache.SAR(0.02, 0.2, 0) }, func() float64 {
			v, _ := SAR(src, decimal.NewFromFloat(0.02), decimal.NewFromFloat(0.2), 0).Float64()
			return v
		}},
		{"Force_ema_13_0", func() float64 { return cache.Force(13, "ema", 0) }, func() float64 {
			v, _ := Force(src, 13, "ema", 0, 0).Float64()
			return v
		}},
		{"AMA_14_2_30_0", func() float64 { return cache.AMA(14, 2, 30, 0) }, func() float64 {
			v, _ := AMA(src, 14, 2, 30, 0, 1).Float64()
			return v
		}},
	}
}

// TestSeriesCache_RevisionedRollingWindow verifies that a RevisionedBarSource
// with a fixed-length rolling window (500→500) properly resets and rebuilds
// all cache-backed indicators when the revision changes. This is the core
// adversarial test for LIVE-INDICATOR-1: it MUST reuse the same source and
// same cache (creating a second cache would hide the bug).
//
// Adversarial proof: delete the revision-change reset branch in EnsureUpdated
// → this test goes RED (indicators stay frozen at first-window values).
func TestSeriesCache_RevisionedRollingWindow(t *testing.T) {
	src := makeRevisionedBars(500)
	cache := NewSeriesCache(src)

	// Phase 1: query all cache-backed indicators at revision 1.
	// This builds the incremental state that would freeze without the fix.
	// EnsureUpdated must be called first (same as runner.ensureCache) so c.n
	// is set correctly — without it, the lazy rebuild leaves c.n=0 and the
	// next EnsureUpdated would double-process all bars.
	cache.EnsureUpdated()
	queries := allCacheBackedIndicators(cache, src)
	for _, q := range queries {
		_ = q.cache()
	}

	// Phase 2: roll the window — drop oldest, append newest, length stays 500.
	// Use a distinctive close value so indicator values must change.
	rollWindow(src, 42.0)

	// Phase 3: query again with the SAME cache. EnsureUpdated must detect
	// the revision change and reset+rebuild. Compare every cache-backed
	// indicator against the stateless result computed from the new window.
	cache.EnsureUpdated() // caller's responsibility — same as runner.ensureCache
	queries2 := allCacheBackedIndicators(cache, src)
	for _, q := range queries2 {
		t.Run(q.name, func(t *testing.T) {
			got := q.cache()
			want := q.state()
			if !approxEqual(got, want) {
				t.Errorf("after roll: cache=%.10f stateless=%.10f diff=%.2e (indicator frozen — revision reset missing?)",
					got, want, got-want)
			}
		})
	}
}

// TestSeriesCache_RevisionUnchanged_NoRebuild verifies that when the revision
// is unchanged, EnsureUpdated does NOT reset — existing series pointers remain
// stable. This proves the tick hot path (thousands of ticks within one bar)
// does not rebuild the cache.
func TestSeriesCache_RevisionUnchanged_NoRebuild(t *testing.T) {
	src := makeRevisionedBars(200)
	cache := NewSeriesCache(src)

	// First query builds the EMA(14) series.
	cache.EnsureUpdated()
	v1 := cache.EMA(14, 0)

	// Capture the series pointer.
	series1 := cache.ema[14]

	// Second EnsureUpdated + query with same revision — must not rebuild.
	cache.EnsureUpdated()
	v2 := cache.EMA(14, 0)
	series2 := cache.ema[14]

	if series1 != series2 {
		t.Error("series pointer changed with unchanged revision — tick hot path would rebuild cache")
	}
	if !approxEqual(v1, v2) {
		t.Errorf("value changed with unchanged revision: %.10f vs %.10f", v1, v2)
	}

	// Third call — still stable.
	cache.EnsureUpdated()
	v3 := cache.EMA(14, 0)
	series3 := cache.ema[14]
	if series2 != series3 {
		t.Error("series pointer changed on third call with unchanged revision")
	}
	if !approxEqual(v2, v3) {
		t.Errorf("value changed on third call: %.10f vs %.10f", v2, v3)
	}
}
