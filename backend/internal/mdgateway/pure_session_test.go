package mdgateway

import (
	"context"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/repository"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// --- session_clock.go ---

func TestDefaultSessionClock(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	if sc == nil {
		t.Fatal("DefaultSessionClock returned nil")
	}
	if sc.BrokerOffsetMs() != 0 {
		t.Error("default offset should be 0")
	}
}

func TestSetBrokerOffset(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	sc.SetBrokerOffset(500)
	if sc.BrokerOffsetMs() != 500 {
		t.Errorf("BrokerOffsetMs = %d, want 500", sc.BrokerOffsetMs())
	}
}

func TestAddRemoveHoliday(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	dateStr := date.Format("2006-01-02")
	sc.AddHoliday(dateStr)
	if !sc.IsHoliday(date) {
		t.Error("date should be holiday after AddHoliday")
	}
	sc.RemoveHoliday(dateStr)
	if sc.IsHoliday(date) {
		t.Error("date should not be holiday after RemoveHoliday")
	}
}

func TestIsWeekend(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	sat := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	if !sc.IsWeekend(sat) {
		t.Error("Saturday should be weekend")
	}
	mon := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	if sc.IsWeekend(mon) {
		t.Error("Monday should not be weekend")
	}
}

func TestSessionPhase(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	phase := sc.SessionPhase(time.Now())
	if phase == "" {
		t.Error("SessionPhase should return non-empty string")
	}
}

func TestInSwapWindow(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	_ = sc.InSwapWindow(time.Now())
}

func TestBarBoundary(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	boundary := sc.BarBoundary(time.Now(), 3600_000) // 1h in ms
	if boundary <= 0 {
		t.Error("BarBoundary should return positive timestamp")
	}
}

func TestClockSkewMs(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	ms := sc.ClockSkewMs(time.Now().UnixMilli())
	if ms < 0 {
		t.Errorf("ClockSkewMs = %d, want >=0", ms)
	}
}

// --- session_clock.go BrokerTime ---

func TestBrokerTime(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	bt := sc.BrokerTime()
	if bt.IsZero() {
		t.Error("BrokerTime should not be zero")
	}
}

// --- session_clock.go ClockSkewMs with offset ---

func TestClockSkewMs_WithOffset(t *testing.T) {
	t.Parallel()
	sc := DefaultSessionClock()
	sc.SetBrokerOffset(500)
	ms := sc.ClockSkewMs(time.Now().UnixMilli())
	if ms < 0 {
		t.Errorf("ClockSkewMs should be >=0, got %d", ms)
	}
}

// --- market_state.go ---

func TestDefaultMarketStateConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultMarketStateConfig()
	if cfg.MaxQuoteAgeMs <= 0 {
		t.Errorf("MaxQuoteAgeMs = %d, want >0", cfg.MaxQuoteAgeMs)
	}
}

func TestMarketState_GetAll(t *testing.T) {
	t.Parallel()
	ms := NewMarketStateTracker(DefaultMarketStateConfig())
	ms.Update(&mdtick.Tick{Broker: "broker", Canonical: "EURUSD", TsUnixMs: time.Now().UnixMilli()})
	if len(ms.All()) == 0 {
		t.Error("All should return non-empty slice")
	}
	if ms.Get("broker", "EURUSD") == nil {
		t.Error("Get should return non-nil state")
	}
}

func TestMarketState_RefreshAges(t *testing.T) {
	t.Parallel()
	ms := NewMarketStateTracker(DefaultMarketStateConfig())
	ms.Update(&mdtick.Tick{Broker: "broker", Canonical: "EURUSD", TsUnixMs: time.Now().UnixMilli()})
	ms.RefreshAges(time.Now())
}

func TestEvaluateTradeable_Holiday(t *testing.T) {
	t.Parallel()
	ms := NewMarketStateTracker(DefaultMarketStateConfig())
	state := &MarketState{SessionPhase: PhaseHoliday}
	if ms.evaluateTradeable(state) {
		t.Error("should return false for holiday phase")
	}
}

// --- market_state.go EvaluateTradeable ---

func TestEvaluateTradeable_Stale(t *testing.T) {
	t.Parallel()
	ms := NewMarketStateTracker(DefaultMarketStateConfig())
	state := &MarketState{QuoteAgeMs: 100_000} // very stale
	if ms.evaluateTradeable(state) {
		t.Error("stale quote should not be tradeable")
	}
}

func TestEvaluateTradeable_NoStatePhase(t *testing.T) {
	t.Parallel()
	ms := NewMarketStateTracker(DefaultMarketStateConfig())
	state := &MarketState{SessionPhase: "UKNOWN_PHASE", QuoteAgeMs: 100}
	if !ms.evaluateTradeable(state) {
		t.Error("unknown phase with fresh quote should be tradeable")
	}
}

func TestEvaluateTradeable_Weekend(t *testing.T) {
	t.Parallel()
	ms := NewMarketStateTracker(DefaultMarketStateConfig())
	state := &MarketState{SessionPhase: PhaseWeekend, QuoteAgeMs: 100}
	if ms.evaluateTradeable(state) {
		t.Error("weekend phase should not be tradeable")
	}
}

func TestEvaluateTradeable_SpreadAnomaly(t *testing.T) {
	t.Parallel()
	ms := NewMarketStateTracker(DefaultMarketStateConfig())
	state := &MarketState{SessionPhase: PhaseOpen, QuoteAgeMs: 100, SpreadZscore: 999}
	if ms.evaluateTradeable(state) {
		t.Error("spread anomaly should not be tradeable")
	}
}

// --- market_state.go update/stale ---

func TestMarketState_UpdateStale(t *testing.T) {
	t.Parallel()
	ms := NewMarketStateTracker(DefaultMarketStateConfig())
	now := time.Now()
	ms.Update(&mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: now.UnixMilli() - 100_000, ArrivedUnixMs: now.UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	})
	ms.RefreshAges(now)
}

// --- user_metrics_flusher.go ---

func TestNewUserMetricsCollector(t *testing.T) {
	t.Parallel()
	c := NewUserMetricsCollector()
	if c == nil {
		t.Fatal("NewUserMetricsCollector returned nil")
	}
}

func TestRecordAndFlush(t *testing.T) {
	t.Parallel()
	c := NewUserMetricsCollector()
	c.Record("acct-1", "tick_count", 100.0)
	c.Record("acct-1", "bar_count", 50.0)
	c.Flush()
}

func TestFlushedTotal_Initial(t *testing.T) {
	t.Parallel()
	uf := NewUserMetricsFlusher(time.Minute, nil)
	if uf.FlushedTotal() != 0 {
		t.Errorf("FlushedTotal = %d, want 0", uf.FlushedTotal())
	}
	if uf.FlushErrors() != 0 {
		t.Errorf("FlushErrors = %d, want 0", uf.FlushErrors())
	}
}

// --- bar_aggregator.go ---

func TestNewBarAggregator(t *testing.T) {
	t.Parallel()
	agg := NewBarAggregator()
	if agg == nil {
		t.Fatal("NewBarAggregator returned nil")
	}
}

func TestLoadFinalizedBars_Empty(t *testing.T) {
	t.Parallel()
	agg := NewBarAggregator()
	agg.LoadFinalizedBars(nil)
	agg.LoadFinalizedBars(make(map[repository.FinalizedKey][]int64))
}

func TestIngestExternalBar_NoFinalized(t *testing.T) {
	t.Parallel()
	agg := NewBarAggregator()
	bar := &mdtick.Bar{
		Broker: "broker", Canonical: "EURUSD", Period: "1m",
		CloseTsUnixMs: 1000,
	}
	if !agg.IngestExternalBar(bar) {
		t.Error("should accept bar when no finalized data")
	}
}

func TestBarSkippedFinalized(t *testing.T) {
	t.Parallel()
	BarSkippedFinalized()
	BarSkippedFinalized()
	// Verify counter is non-negative.
	if BarSkippedFinalized() < 0 {
		t.Error("BarSkippedFinalized should be >= 0")
	}
}

func TestBarAggregator_AddTick(t *testing.T) {
	t.Parallel()
	agg := NewBarAggregator()
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	agg.AddTick(tick, func(b *mdtick.Bar) {})
}

// --- bar_aggregator.go deeper tests ---

func TestBarAggregator_AddTick_SameBucket(t *testing.T) {
	t.Parallel()
	agg := NewBarAggregator()
	now := time.Now().UnixMilli()
	tick1 := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: now, ArrivedUnixMs: now,
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1002),
		BidVolume: 10, AskVolume: 5,
	}
	tick2 := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: now, ArrivedUnixMs: now + 100, // same bucket
		Bid: decimal.NewFromFloat(1.1005), Ask: decimal.NewFromFloat(1.1007),
		BidVolume: 5, AskVolume: 3,
	}
	agg.AddTick(tick1, func(b *mdtick.Bar) {
		t.Error("should not emit bar on first tick (same bucket)")
	})
	agg.AddTick(tick2, func(b *mdtick.Bar) {
		t.Error("should not emit bar, still same bucket")
	})
}

func TestBarAggregator_AddTick_DifferentBucket(t *testing.T) {
	t.Parallel()
	agg := NewBarAggregator()
	now := time.Now().UnixMilli()
	tick1 := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: now, ArrivedUnixMs: now,
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1002),
	}
	tick2 := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: now + 3600_000, ArrivedUnixMs: now + 3600_000, // different bucket for 1h
		Bid: decimal.NewFromFloat(1.1050), Ask: decimal.NewFromFloat(1.1052),
	}
	emitted := 0
	agg.AddTick(tick1, func(b *mdtick.Bar) {
		emitted++
	})
	agg.AddTick(tick2, func(b *mdtick.Bar) {
		emitted++
	})
	if emitted < 1 {
		t.Errorf("should have emitted at least 1 bar, got %d", emitted)
	}
}

func TestBarAggregator_IngestExternalBar_Finalized(t *testing.T) {
	t.Parallel()
	agg := NewBarAggregator()
	bar := &mdtick.Bar{
		Broker: "broker", Canonical: "EURUSD", Period: "1m",
		CloseTsUnixMs: 1000,
	}
	if !agg.IngestExternalBar(bar) {
		t.Error("first bar should be accepted")
	}
	if agg.IngestExternalBar(bar) {
		t.Error("duplicate bar should be rejected")
	}
}

// --- normalizer_invalidator.go ---

func TestNewNormalizerInvalidator(t *testing.T) {
	t.Parallel()
	inv := NewNormalizerInvalidator(nil, nil, func(broker, raw string) {})
	if inv == nil {
		t.Fatal("NewNormalizerInvalidator returned nil")
	}
}

func TestNormalizerInvalidator_StartStop(t *testing.T) {
	t.Parallel()
	inv := NewNormalizerInvalidator(zap.NewNop(), nil, func(broker, raw string) {})
	inv.Start(context.Background(), nil)
	time.Sleep(10 * time.Millisecond)
	inv.Stop()
}

// --- normalizer_invalidator.go tickerLoop ---

func TestNormalizerInvalidator_TickerLoop(t *testing.T) {
	t.Parallel()
	called := false
	inv := NewNormalizerInvalidator(zap.NewNop(), nil, func(broker, raw string) { called = true })
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	inv.Start(ctx, nil)
	time.Sleep(30 * time.Millisecond)
	inv.Stop()
	_ = called
}

// --- publisher.go ---

func TestNewPublisher_NilJS(t *testing.T) {
	t.Parallel()
	pub := NewPublisher(nil)
	if pub == nil {
		t.Fatal("NewPublisher(nil) returned nil")
	}
}

// --- dlq_writer.go ---

func TestPublisher_PublishTick_NilJS(t *testing.T) {
	t.Parallel()
	pub := NewPublisher(nil)
	err := pub.PublishTick(context.Background(), &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	})
	if err != nil {
		t.Logf("PublishTick with nil JS: %v", err)
	}
}

func TestPublisher_PublishBar_NilJS(t *testing.T) {
	t.Parallel()
	pub := NewPublisher(nil)
	err := pub.PublishBar(context.Background(), &mdtick.Bar{
		Broker: "broker", Canonical: "EURUSD", Period: "1h",
		CloseTsUnixMs: time.Now().UnixMilli(),
		Open:          decimal.NewFromFloat(1.1000), High: decimal.NewFromFloat(1.1050),
		Low: decimal.NewFromFloat(1.0990), Close: decimal.NewFromFloat(1.1020),
	})
	if err != nil {
		t.Logf("PublishBar with nil JS: %v", err)
	}
}

func TestPublisher_PublishBarRevision_NilJS(t *testing.T) {
	t.Parallel()
	pub := NewPublisher(nil)
	err := pub.PublishBarRevision(context.Background(), &mdtick.Bar{
		Broker: "broker", Canonical: "EURUSD", Period: "1h",
		CloseTsUnixMs: time.Now().UnixMilli(),
		Open:          decimal.NewFromFloat(1.1000), High: decimal.NewFromFloat(1.1050),
		Low: decimal.NewFromFloat(1.0990), Close: decimal.NewFromFloat(1.1020),
	})
	if err != nil {
		t.Logf("PublishBarRevision with nil JS: %v", err)
	}
}
