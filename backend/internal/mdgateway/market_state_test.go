package mdgateway

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/internal/mdgateway/adapter/mdtick"
)

// --- F1: MarketState Tests ---

func TestMarketStateTracker_Update(t *testing.T) {
	t.Parallel()
	cfg := DefaultMarketStateConfig()
	tracker := NewMarketStateTracker(cfg)

	tick := makeTick("test", "EURUSD", "1.1000", "1.1002")
	ms := tracker.Update(tick)
	if ms == nil {
		t.Fatal("Update should return non-nil MarketState")
	}
	if ms.IsTradeable != true {
		t.Fatalf("fresh quote should be tradeable, got IsTradeable=%v reason=%s", ms.IsTradeable, ms.FreezeReason)
	}
}

func TestMarketStateTracker_StaleQuote(t *testing.T) {
	t.Parallel()
	cfg := DefaultMarketStateConfig()
	cfg.MaxQuoteAgeMs = 100
	tracker := NewMarketStateTracker(cfg)

	tick := makeTick("test", "EURUSD", "1.1000", "1.1002")
	ms := tracker.Update(tick)
	if ms == nil {
		t.Fatal("expected non-nil MarketState")
	}

	// Simulate aging.
	time.Sleep(200 * time.Millisecond)
	tracker.RefreshAges(time.Now())
	ms = tracker.Get("test", "EURUSD")
	if ms == nil {
		t.Fatal("expected existing MarketState")
	}
	if ms.IsTradeable {
		t.Fatalf("stale quote should not be tradeable, age=%dms", ms.QuoteAgeMs)
	}
}

func TestMarketStateTracker_GetAndAll(t *testing.T) {
	t.Parallel()
	cfg := DefaultMarketStateConfig()
	tracker := NewMarketStateTracker(cfg)

	tick := makeTick("test", "EURUSD", "1.1000", "1.1002")
	tracker.Update(tick)

	ms := tracker.Get("test", "EURUSD")
	if ms == nil {
		t.Fatal("Get should return existing state")
	}
	if ms.Broker != "test" || ms.Symbol != "EURUSD" {
		t.Fatalf("unexpected market state: broker=%s symbol=%s", ms.Broker, ms.Symbol)
	}

	all := tracker.All()
	if len(all) != 1 {
		t.Fatalf("All should return 1 state, got %d", len(all))
	}

	// Get non-existent.
	if tracker.Get("nonexistent", "X") != nil {
		t.Fatal("non-existent symbol should return nil")
	}
}
// --- F3: NTP Clock Skew Drop Tests ---

func TestQuality_NTPSkewDrop(t *testing.T) {
	t.Parallel()
	cfg := DefaultQualityConfig()
	cfg.MaxClockSkewMs = 5000
	q := NewQuality(cfg)

	// Normal tick within skew bounds.
	tick := &mdtick.Tick{
		Broker:    "test",
		SymbolRaw: "EURUSD",
		Bid:       decimal.NewFromFloat(1.1000),
		Ask:       decimal.NewFromFloat(1.1002),
	}
	tick.TsUnixMs = time.Now().UnixMilli()
	tick.ArrivedUnixMs = tick.TsUnixMs + 100 // 100ms skew

	qr := q.Check(nil, tick)
	if qr.Dropped {
		t.Fatalf("100ms skew should not drop: reason=%s", qr.DroppedReason)
	}

	// Tick with extreme clock skew → should drop.
	badTick := &mdtick.Tick{
		Broker:    "test",
		SymbolRaw: "EURUSD",
		Bid:       decimal.NewFromFloat(1.1000),
		Ask:       decimal.NewFromFloat(1.1002),
	}
	badTick.TsUnixMs = time.Now().UnixMilli()
	badTick.ArrivedUnixMs = badTick.TsUnixMs + 10000 // 10s skew

	qrBad := q.Check(nil, badTick)
	if !qrBad.Dropped {
		t.Fatal("10s clock skew should be dropped")
	}
	if qrBad.DroppedReason != "clock_skew" {
		t.Fatalf("drop reason should be clock_skew, got %s", qrBad.DroppedReason)
	}
	if ClockSkewDroppedTotal() < 1 {
		t.Fatal("metric should record clock skew drop")
	}
}

func TestQuality_SpreadComputation(t *testing.T) {
	t.Parallel()
	cfg := DefaultQualityConfig()
	q := NewQuality(cfg)

	tick := &mdtick.Tick{
		Broker:    "test",
		SymbolRaw: "EURUSD",
		Bid:       decimal.NewFromFloat(1.1000),
		Ask:       decimal.NewFromFloat(1.1002),
	}
	tick.TsUnixMs = time.Now().UnixMilli()
	tick.ArrivedUnixMs = tick.TsUnixMs

	qr := q.Check(nil, tick)
	// Spread = (1.1002 - 1.1000) / 1.1000 * 10000 ≈ 1.818 bps.
	if qr.SpreadBps < 1.5 || qr.SpreadBps > 2.0 {
		t.Fatalf("spread should be ~1.818 bps, got %.4f", qr.SpreadBps)
	}
}

// --- F4: Quote Stuffing Detection Tests ---

func TestStuffingDetector_NormalTickRate(t *testing.T) {
	t.Parallel()
	cfg := DefaultStuffingDetectorConfig()
	cfg.WindowSize = 20
	detector := NewStuffingDetector(cfg)

	// Feed ~5 ticks/sec → normal rate.
	for i := 0; i < 50; i++ {
		stuffed, _ := detector.Observe("test", "EURUSD")
		if stuffed {
			t.Fatal("normal tick rate should not trigger stuffing")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestStuffingDetector_IsPaused(t *testing.T) {
	t.Parallel()
	cfg := DefaultStuffingDetectorConfig()
	detector := NewStuffingDetector(cfg)

	if detector.IsPaused("test", "EURUSD") {
		t.Fatal("should not be paused initially")
	}

	paused := detector.PausedSymbols()
	if len(paused) != 0 {
		t.Fatalf("no symbols paused, got %d", len(paused))
	}
}

func TestStuffingDetector_ObserveTracksRate(t *testing.T) {
	t.Parallel()
	cfg := DefaultStuffingDetectorConfig()
	cfg.WindowSize = 50
	detector := NewStuffingDetector(cfg)

	// First observation sets baseline.
	stuffed, _ := detector.Observe("test", "EURUSD")
	if stuffed {
		t.Fatal("first observation should not trigger stuffing")
	}

	// Second observation also OK.
	time.Sleep(50 * time.Millisecond)
	stuffed, _ = detector.Observe("test", "EURUSD")
	if stuffed {
		t.Fatal("normal rate should not trigger stuffing")
	}
}

// --- F5: Spread Anomaly Tests ---

func TestQuality_SpreadZscore(t *testing.T) {
	t.Parallel()
	cfg := DefaultQualityConfig()
	cfg.HistorySize = 50
	q := NewQuality(cfg)

	key := "test:EURUSD"
	// Feed 20 identical spreads.
	for i := 0; i < 20; i++ {
		q.trackSpread(key, 2.0)
	}
	// Z-score of same value ≈ 0.
	z := q.SpreadZscore(key, 2.0)
	if z > 1.0 {
		t.Fatalf("identical spread should have z-score ~0, got %.4f", z)
	}

	// Extreme spread → high Z-score.
	zExtreme := q.SpreadZscore(key, 10.0)
	if zExtreme < 2.0 {
		t.Fatalf("extreme spread should have high z-score, got %.4f", zExtreme)
	}
}

func TestQuality_TickRateZscore(t *testing.T) {
	t.Parallel()
	cfg := DefaultQualityConfig()
	cfg.HistorySize = 50
	q := NewQuality(cfg)

	key := "test:EURUSD"
	// Feed normal rates.
	for i := 0; i < 20; i++ {
		q.trackTickRate(key, 5.0)
	}
	// Z-score of same rate ≈ 0.
	z := q.TickRateZscore(key, 5.0)
	if z > 1.0 {
		t.Fatalf("same rate should have z-score ~0, got %.4f", z)
	}
}

// --- Helpers ---

func makeTick(broker, canonical, bidStr, askStr string) *mdtick.Tick {
	now := time.Now().UnixMilli()
	return &mdtick.Tick{
		Broker:        broker,
		Canonical:     canonical,
		SymbolRaw:     canonical,
		Bid:           decimal.RequireFromString(bidStr),
		Ask:           decimal.RequireFromString(askStr),
		TsUnixMs:      now,
		ArrivedUnixMs: now,
	}
}
