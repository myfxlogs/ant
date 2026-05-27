package mdgateway

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/internal/mdgateway/adapter/mdtick"
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

// --- F2: SessionClock Tests ---

func TestSessionClock_IsWeekend(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()

	// Saturday at noon UTC — weekend.
	sat := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	if !sc.IsWeekend(sat) {
		t.Fatal("Saturday noon should be weekend")
	}

	// Wed noon UTC — not weekend.
	wed := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	if sc.IsWeekend(wed) {
		t.Fatal("Wednesday noon should not be weekend")
	}

	// Friday 23:00 UTC is past close → weekend.
	friNight := time.Date(2026, 5, 22, 23, 0, 0, 0, time.UTC)
	if !sc.IsWeekend(friNight) {
		t.Fatal("Friday 23:00 UTC should be weekend")
	}
}

func TestSessionClock_IsHoliday(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	sc.AddHoliday("2026-01-01") // New Year

	holiday := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if !sc.IsHoliday(holiday) {
		t.Fatal("Jan 1 should be holiday")
	}

	normalDay := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	if sc.IsHoliday(normalDay) {
		t.Fatal("Jan 2 should not be holiday")
	}
}

func TestSessionClock_SessionPhase(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	sc.AddHoliday("2026-12-25")

	// Regular weekday.
	wed := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	if sc.SessionPhase(wed) != PhaseOpen {
		t.Fatalf("Wednesday should be open, got %s", sc.SessionPhase(wed))
	}

	// Holiday.
	xmas := time.Date(2026, 12, 25, 12, 0, 0, 0, time.UTC)
	if sc.SessionPhase(xmas) != PhaseHoliday {
		t.Fatalf("Christmas should be holiday, got %s", sc.SessionPhase(xmas))
	}

	// Weekend.
	sat := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	if sc.SessionPhase(sat) != PhaseWeekend {
		t.Fatalf("Saturday should be weekend, got %s", sc.SessionPhase(sat))
	}
}

func TestSessionClock_InSwapWindow(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()

	// 21:30 UTC is within swap window (21-23).
	inWindow := time.Date(2026, 5, 20, 21, 30, 0, 0, time.UTC)
	if !sc.InSwapWindow(inWindow) {
		t.Fatal("21:30 UTC should be in swap window")
	}

	// 12:00 UTC is outside swap window.
	outWindow := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	if sc.InSwapWindow(outWindow) {
		t.Fatal("12:00 UTC should not be in swap window")
	}
}

func TestSessionClock_BrokerOffset(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	sc.SetBrokerOffset(100) // broker is 100ms ahead
	if sc.BrokerOffsetMs() != 100 {
		t.Fatalf("broker offset: want 100, got %d", sc.BrokerOffsetMs())
	}
}

func TestSessionClock_BarBoundary(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	// 1m bar from 10:00:30 → boundary at 10:01:00.
	t0 := time.Date(2026, 5, 20, 10, 0, 30, 0, time.UTC)
	boundary := sc.BarBoundary(t0, 60_000)
	expected := t0.Truncate(time.Minute).Add(time.Minute).UnixMilli()
	if boundary != expected {
		t.Fatalf("bar boundary: want %d, got %d", expected, boundary)
	}
}

func TestSessionClock_ClockSkewMs(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	now := time.Now().UnixMilli()
	brokerTs := now - 100 // broker 100ms behind
	skew := sc.ClockSkewMs(brokerTs)
	if skew < 90 || skew > 200 {
		t.Fatalf("clock skew should be ~100ms, got %d", skew)
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
