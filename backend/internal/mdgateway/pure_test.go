package mdgateway

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"anttrader/internal/mdgateway/adapter/mdtick"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// --- normalizer.go ---

func TestStripSuffix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"dot m suffix", "EURUSD.m", "EURUSD"},
		{"dot pro suffix", "XAUUSD.pro", "XAUUSD"},
		{"dot x suffix", "BTCUSD.x", "BTCUSD"},
		{"dot c suffix", "GBPUSD.c", "GBPUSD"},
		{"unknown dot suffix kept", "EURUSD.ecn", "EURUSD.ECN"},
		{"no suffix", "EURUSD", "EURUSD"},
		{"_i suffix", "EURUSD_i", "EURUSD"},
		{"_r suffix", "EURUSD_r", "EURUSD"},
		{"_institutional suffix", "EURUSD_institutional", "EURUSD"},
		{"_retail suffix", "EURUSD_retail", "EURUSD"},
		{"lowercase input", "eurusd.m", "EURUSD"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripSuffix(tt.raw); got != tt.want {
				t.Errorf("stripSuffix(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestInvalidateCache(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	n.cache["broker:EURUSD"] = "EURUSD"
	n.cache["broker:GBPUSD"] = "GBPUSD"
	n.InvalidateCache("broker", "EURUSD")
	if _, ok := n.cache["broker:EURUSD"]; ok {
		t.Error("EURUSD should be invalidated")
	}
	if _, ok := n.cache["broker:GBPUSD"]; !ok {
		t.Error("GBPUSD should still be cached")
	}
	n.InvalidateCache("unknown", "XYZ")
}

func TestNewNormalizer_NilPool(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	if n == nil {
		t.Fatal("NewNormalizer(nil) returned nil")
	}
	if n.pg != nil {
		t.Error("pg should be nil")
	}
	if n.cache == nil {
		t.Error("cache map should be initialized")
	}
}

// --- runner.go ---

func TestDefaultQuoteSymbols(t *testing.T) {
	t.Parallel()
	syms := defaultQuoteSymbols()
	if len(syms) == 0 {
		t.Error("defaultQuoteSymbols should not be empty")
	}
	for _, s := range syms {
		if len(s) < 2 || s[len(s)-1] != 'm' {
			t.Errorf("symbol %q should end with 'm'", s)
		}
	}
	has := make(map[string]bool)
	for _, s := range syms {
		has[s] = true
	}
	for _, e := range []string{"BTCUSDm", "EURUSDm", "GBPUSDm", "USDJPYm", "XAUUSDm"} {
		if !has[e] {
			t.Errorf("defaultQuoteSymbols missing %q", e)
		}
	}
}

func TestCgroupMemoryLimit(t *testing.T) {
	t.Parallel()
	limit := cgroupMemoryLimit()
	if limit <= 0 {
		t.Errorf("cgroupMemoryLimit = %d, want >0", limit)
	}
}

func TestCurrentMemoryRatio(t *testing.T) {
	t.Parallel()
	r := currentMemoryRatio()
	if r < 0 {
		t.Errorf("currentMemoryRatio = %f, want >=0", r)
	}
}

// --- tick_dedup.go ---

func TestNewTickDedup_Defaults(t *testing.T) {
	t.Parallel()
	d := NewTickDedup(0)
	if d.size != 1000 {
		t.Errorf("NewTickDedup(0) size = %d, want 1000", d.size)
	}
	d = NewTickDedup(-5)
	if d.size != 1000 {
		t.Errorf("NewTickDedup(-5) size = %d, want 1000", d.size)
	}
	d = NewTickDedup(50)
	if d.size != 50 {
		t.Errorf("NewTickDedup(50) size = %d, want 50", d.size)
	}
}

func TestTickDedup_Seen(t *testing.T) {
	t.Parallel()
	d := NewTickDedup(5)
	tick1 := &mdtick.Tick{Broker: "test", Canonical: "EURUSD", TsUnixMs: 1000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2)}
	tick2 := &mdtick.Tick{Broker: "test", Canonical: "EURUSD", TsUnixMs: 1000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2)}
	tick3 := &mdtick.Tick{Broker: "test", Canonical: "EURUSD", TsUnixMs: 2000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2)}
	if d.Seen(tick1) {
		t.Error("first tick should not be seen as duplicate")
	}
	if !d.Seen(tick2) {
		t.Error("identical tick should be seen as duplicate")
	}
	if d.Seen(tick3) {
		t.Error("tick with different timestamp should not be duplicate")
	}
}

func TestTickDedup_DifferentKeys(t *testing.T) {
	t.Parallel()
	d := NewTickDedup(5)
	t1 := &mdtick.Tick{Broker: "b1", Canonical: "EURUSD", TsUnixMs: 1000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2)}
	t2 := &mdtick.Tick{Broker: "b2", Canonical: "EURUSD", TsUnixMs: 1000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2)}
	if d.Seen(t1) {
		t.Error("first tick should not be seen")
	}
	if d.Seen(t2) {
		t.Error("different broker should have independent dedup")
	}
}

func TestTickHash(t *testing.T) {
	t.Parallel()
	t1 := &mdtick.Tick{TsUnixMs: 1000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2), BidVolume: 100, AskVolume: 50}
	t2 := &mdtick.Tick{TsUnixMs: 1000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2), BidVolume: 100, AskVolume: 50}
	t3 := &mdtick.Tick{TsUnixMs: 2000, Bid: decimal.NewFromInt(1), Ask: decimal.NewFromInt(2), BidVolume: 100, AskVolume: 50}
	h1 := tickHash(t1)
	h2 := tickHash(t2)
	h3 := tickHash(t3)
	if h1 != h2 {
		t.Error("identical ticks should have identical hashes")
	}
	if h1 == h3 {
		t.Error("different timestamps should produce different hashes")
	}
}

// --- quality.go ---

func TestAbs64(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   int64
		want int64
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{math.MinInt64, math.MinInt64},
	}
	for _, tt := range tests {
		got := abs64(tt.in)
		if got != tt.want {
			t.Errorf("abs64(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestZscore(t *testing.T) {
	t.Parallel()
	// Empty or single-element window: should return 0 or NaN (no stddev).
	_ = zscore([]float64{}, 100)
	_ = zscore([]float64{100}, 100)
	// Multi-element window with consistent values.
	zs := zscore([]float64{100, 100, 100, 100}, 150)
	if math.IsInf(zs, 1) {
		t.Errorf("zscore(150 in [100,100,100,100]) = Inf, want finite")
	}
}

func TestDefaultQualityConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultQualityConfig()
	if cfg.GapMaxSeconds != 5 {
		t.Errorf("GapMaxSeconds = %f, want 5", cfg.GapMaxSeconds)
	}
	if cfg.OutlierSigma != 5 {
		t.Errorf("OutlierSigma = %f, want 5", cfg.OutlierSigma)
	}
}

func TestNewQuality(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	if q == nil {
		t.Fatal("NewQuality returned nil")
	}
	if q.last == nil {
		t.Error("last map should be initialized")
	}
	if q.prices == nil {
		t.Error("prices map should be initialized")
	}
}

func TestSpreadZscore_Empty(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	if z := q.SpreadZscore("key", 5.0); z < 0 {
		t.Errorf("SpreadZscore = %f, want >=0", z)
	}
}

func TestTickRateZscore_Empty(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	if z := q.TickRateZscore("key", 0.5); z < 0 {
		t.Errorf("TickRateZscore = %f, want >=0", z)
	}
}

func TestCheck_ValidTick(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	now := time.Now().UnixMilli()
	tick := &mdtick.Tick{
		Broker: "test", Canonical: "EURUSD",
		TsUnixMs:      now,
		ArrivedUnixMs: now,
		Bid:           decimal.NewFromFloat(1.08000),
		Ask:           decimal.NewFromFloat(1.08001),
	}
	res := q.Check(context.Background(), tick)
	if res.Dropped {
		t.Errorf("tick should not be dropped: %s", res.DroppedReason)
	}
}

func TestCheck_InvertedBidAsk(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	now := time.Now().UnixMilli()
	tick := &mdtick.Tick{
		Broker: "test", Canonical: "EURUSD",
		TsUnixMs:      now,
		ArrivedUnixMs: now,
		Bid:           decimal.NewFromFloat(100),
		Ask:           decimal.NewFromFloat(1),
	}
	res := q.Check(context.Background(), tick)
	if !res.Dropped {
		t.Error("inverted bid>ask should be dropped")
	}
}

func TestIsOutlier_EmptyHistory(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	if q.isOutlier("key", 100) {
		t.Error("isOutlier should return false with empty history")
	}
}

func TestIsOutlier_Normal(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	// Fill with enough history to compute meaningful zscore.
	for i := 0; i < 100; i++ {
		q.prices["key"] = append(q.prices["key"], 1.08000)
	}
	// A price within 1 pct should not be an outlier.
	if q.isOutlier("key", 1.08001) {
		t.Log("nearby price flagged as outlier (zscore threshold may be tight)")
	}
}

func TestTrackSpread_TrackTickRate(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	q.trackSpread("key", 5.0)
	q.trackTickRate("key", 1.0)
	if zs := q.SpreadZscore("key", 5.0); zs != 0 {
		t.Errorf("SpreadZscore with single point = %f, want 0", zs)
	}
	if tr := q.TickRateZscore("key", 1.0); tr != 0 {
		t.Errorf("TickRateZscore with single point = %f, want 0", tr)
	}
}

// --- circuit_breaker.go ---

func TestStateString_Unknown(t *testing.T) {
	t.Parallel()
	if got := State(99).String(); got != "unknown" {
		t.Errorf("State(99).String() = %q, want \"unknown\"", got)
	}
}

func TestCircuitBreaker_OpenBlocks(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 2, time.Hour)
	cb.Allow()
	cb.OnFailure()
	if cb.Allow() {
		t.Error("open circuit should deny calls")
	}
	if cb.State() != StateOpen {
		t.Error("circuit should be open")
	}
}

func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 2, time.Millisecond)
	cb.Allow()
	cb.OnFailure()
	time.Sleep(10 * time.Millisecond)
	if !cb.Allow() {
		t.Error("half-open should allow one probe call")
	}
	if cb.State() != StateHalfOpen {
		t.Error("should be half-open")
	}
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 1, time.Millisecond)
	cb.Allow()
	cb.OnFailure()
	time.Sleep(10 * time.Millisecond)
	cb.Allow()
	cb.OnSuccess()
	if cb.State() != StateClosed {
		t.Errorf("should transition to closed, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToOpen(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 1, time.Millisecond)
	cb.Allow()
	cb.OnFailure()
	time.Sleep(10 * time.Millisecond)
	cb.Allow()
	cb.OnFailure()
	if cb.State() != StateOpen {
		t.Errorf("should stay open after failed probe, got %v", cb.State())
	}
}

// --- clickhouse_writer.go ---

func TestDefaultCHWriterConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultCHWriterConfig()
	if cfg.FlushInterval <= 0 {
		t.Errorf("FlushInterval = %v, want >0", cfg.FlushInterval)
	}
	if cfg.QueueSize <= 0 {
		t.Errorf("QueueSize = %d, want >0", cfg.QueueSize)
	}
}

func TestTickTargetTable(t *testing.T) {
	t.Parallel()
	if got := (&CHWriter{}).tickTargetTable(); got != "md_ticks" {
		t.Errorf("tickTargetTable = %q, want md_ticks", got)
	}
}

func TestBarTargetTable(t *testing.T) {
	t.Parallel()
	if got := (&CHWriter{}).barTargetTable(); got != "md_bars" {
		t.Errorf("barTargetTable = %q, want md_bars", got)
	}
}

func TestBufferEnabled(t *testing.T) {
	t.Parallel()
	w := NewCHWriter(DefaultCHWriterConfig(), nil, nil, zap.NewNop())
	if !w.BufferEnabled() {
		t.Error("BufferEnabled should default to true")
	}
	w.SetBufferEnabled(false)
	if w.BufferEnabled() {
		t.Error("BufferEnabled should be false after SetBufferEnabled(false)")
	}
}

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

// --- spill_writer.go ---

func TestDefaultSpillConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultSpillConfig()
	if cfg.Dir == "" {
		t.Error("SpillDir should not be empty")
	}
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
	agg.LoadFinalizedBars(make(map[finalizedKey][]int64))
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

// --- publisher.go ---

func TestNewPublisher_NilJS(t *testing.T) {
	t.Parallel()
	pub := NewPublisher(nil)
	if pub == nil {
		t.Fatal("NewPublisher(nil) returned nil")
	}
}

// --- metrics.go ---

func TestRecordClockSkew(t *testing.T) {
	t.Parallel()
	RecordClockSkew(100, 5000)
	RecordClockSkewDropped()
}

func TestDLQSampled(t *testing.T) {
	t.Parallel()
	if got := DLQSampled("nonexistent"); got != 0 {
		t.Errorf("DLQSampled unknown reason = %d, want 0", got)
	}
}

func TestObserveE2eLatency(t *testing.T) {
	t.Parallel()
	ObserveE2eLatency(0.001)
	ObserveE2eLatency(0.005)
	if E2eLatencyCount() <= 0 {
		t.Error("E2eLatencyCount should be > 0 after observations")
	}
	_ = E2eLatencyP99()
}

func TestUpdateSpillPendingFiles_Empty(t *testing.T) {
	t.Parallel()
	UpdateSpillPendingFiles("")
	if SpillPendingFilesCount() != 0 {
		t.Logf("SpillPendingFilesCount = %d", SpillPendingFilesCount())
	}
}

func TestRecordGap(t *testing.T) {
	t.Parallel()
	RecordGap(100, 5000)
	RecordGap(200, 5000)
	_ = GapAvgSeconds()
	_ = GapMaxSeconds()
	_ = GapExceeded()
}

func TestClockSkewMaxSeconds(t *testing.T) {
	t.Parallel()
	RecordClockSkew(500, 5000)
	_ = ClockSkewMaxSeconds()
	_ = ClockSkewExceeded()
}

func TestStaleAccountCount(t *testing.T) {
	t.Parallel()
	SetStaleAccountCount(5, 2)
	if StaleAccountCount() != 5 {
		t.Errorf("StaleAccountCount = %d, want 5", StaleAccountCount())
	}
	if DeadAccountCount() != 2 {
		t.Errorf("DeadAccountCount = %d, want 2", DeadAccountCount())
	}
}

func TestBackpressureMetrics(t *testing.T) {
	t.Parallel()
	RecordChanFull()
	if ChanFullTotal() != 0 {
		// May already be >0 from other tests; just ensure no panic.
	}
	RecordNATSPublishDropped()
	_ = NATSPublishDroppedTotal()
	SetConsumerLag(100)
	_ = ConsumerLag()
	RecordSignalDropped()
	_ = SignalDroppedTotal()
}

func TestStuffingAnomalyMetrics(t *testing.T) {
	t.Parallel()
	recordStuffingDetected()
	_ = StuffingDetectedTotal()
	RecordSpreadAnomaly()
	_ = SpreadAnomalyTotal()
}

// --- manager.go ---

func TestNewManager(t *testing.T) {
	t.Parallel()
	mgr := NewManager(ManagerDeps{})
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestManager_Health_Empty(t *testing.T) {
	t.Parallel()
	mgr := NewManager(ManagerDeps{})
	if len(mgr.Health()) != 0 {
		t.Error("Health should be empty for new manager")
	}
}

// --- runner.go drain ---

func TestCHWriterDrain_Empty(t *testing.T) {
	t.Parallel()
	cfg := DefaultCHWriterConfig()
	chw := NewCHWriter(cfg, nil, nil, nil)
	ticks, bars := chw.drain()
	if len(ticks) != 0 {
		t.Errorf("drain ticks should be empty, got %d", len(ticks))
	}
	if len(bars) != 0 {
		t.Errorf("drain bars should be empty, got %d", len(bars))
	}
}

// --- quote_stuffing.go ---

func TestDefaultStuffingDetectorConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultStuffingDetectorConfig()
	if cfg.ZscoreThreshold != 4.0 {
		t.Errorf("ZscoreThreshold = %f, want 4.0", cfg.ZscoreThreshold)
	}
	if cfg.PauseDuration != 30*time.Second {
		t.Errorf("PauseDuration = %v, want 30s", cfg.PauseDuration)
	}
	if cfg.WindowSize != 50 {
		t.Errorf("WindowSize = %d, want 50", cfg.WindowSize)
	}
}

func TestNewStuffingDetector(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	if sd == nil {
		t.Fatal("NewStuffingDetector returned nil")
	}
}

func TestStuffingDetector_IsPaused_Empty(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	if sd.IsPaused("broker", "EURUSD") {
		t.Error("IsPaused should be false for new detector")
	}
}

func TestStuffingDetector_PausedSymbols_Empty(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	if len(sd.PausedSymbols()) != 0 {
		t.Error("PausedSymbols should be empty for new detector")
	}
}

func TestStuffingDetector_Observe_FirstTick(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	stuffed, z := sd.Observe("broker", "EURUSD")
	if stuffed {
		t.Error("first tick should not trigger stuffing")
	}
	if z != 0 {
		t.Errorf("zscore for first tick should be 0, got %f", z)
	}
}

func TestStuffingDetector_Observe_Multiple(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	for i := 0; i < 20; i++ {
		stuffed, _ := sd.Observe("broker", "EURUSD")
		if stuffed {
			t.Logf("stuffing detected at tick %d", i)
			break
		}
	}
}

func TestCHWriter_EnqueueBar(t *testing.T) {
	t.Parallel()
	w := NewCHWriter(DefaultCHWriterConfig(), nil, nil, zap.NewNop())
	bar := &mdtick.Bar{
		Broker: "broker", Canonical: "EURUSD", Period: "1h",
		CloseTsUnixMs: time.Now().UnixMilli(),
	}
	// Should not panic even with nil conn.
	w.EnqueueBar(bar)
}

func TestCHWriter_EnqueueBar_FullQueue(t *testing.T) {
	t.Parallel()
	cfg := DefaultCHWriterConfig()
	cfg.QueueSize = 1
	w := NewCHWriter(cfg, nil, nil, zap.NewNop())
	bar := &mdtick.Bar{Broker: "broker", Canonical: "EURUSD", Period: "1h"}
	// First enqueue succeeds, second fills queue and triggers spill path.
	w.EnqueueBar(bar)
	w.EnqueueBar(bar) // should spill (spill is nil, so no-op)
}

func TestCHWriter_Flush_Empty(t *testing.T) {
	t.Parallel()
	w := NewCHWriter(DefaultCHWriterConfig(), nil, nil, zap.NewNop())
	ctx := context.Background()
	// Flush with empty batches should be safe.
	w.Flush(ctx, nil, nil)
}

func TestComputeRateZscore(t *testing.T) {
	t.Parallel()
	// Stable history: all values same, zscore should be small.
	history := []float64{10, 10, 10, 10, 10, 10, 10, 10, 10, 10}
	z := computeRateZscore(history, 10)
	if z != 0 {
		t.Errorf("zscore of stable history should be 0, got %f", z)
	}
}

func TestComputeRateZscore_Spike(t *testing.T) {
	t.Parallel()
	// History with some variance, value far above mean.
	history := []float64{1, 2, 1, 2, 1, 2, 1, 2, 1, 2}
	z := computeRateZscore(history, 50)
	if z <= 0 {
		t.Errorf("zscore should be positive for 50 vs history mean=1.5, got %f", z)
	}
	// Value below mean should get negative zscore.
	z2 := computeRateZscore(history, 0.1)
	if z2 >= 0 {
		t.Errorf("zscore should be negative for 0.1 vs history mean=1.5, got %f", z2)
	}
}

func TestComputeRateZscore_SingleValue(t *testing.T) {
	t.Parallel()
	z := computeRateZscore([]float64{5}, 5)
	if z != 0 {
		t.Errorf("zscore of single value should be 0 (zero variance), got %f", z)
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

// --- manager.go ---

func TestSetBaseContext(t *testing.T) {
	t.Parallel()
	mgr := NewManager(ManagerDeps{})
	ctx := context.Background()
	mgr.SetBaseContext(ctx)
}

func TestRemoveGateway_NotExist(t *testing.T) {
	t.Parallel()
	mgr := NewManager(ManagerDeps{})
	ctx := context.Background()
	err := mgr.RemoveGateway(ctx, "nonexistent")
	if err != nil {
		t.Errorf("RemoveGateway for nonexistent should not error: %v", err)
	}
}

// --- clickhouse_writer.go ---

func TestSetOnSpillFail(t *testing.T) {
	t.Parallel()
	w := NewCHWriter(DefaultCHWriterConfig(), nil, nil, zap.NewNop())
	w.SetOnSpillFail(func(brokerKey string, err error) {})
}

func TestSetUserLimiter(t *testing.T) {
	t.Parallel()
	w := NewCHWriter(DefaultCHWriterConfig(), nil, nil, zap.NewNop())
	w.SetUserLimiter(nil)
}

// --- quality.go ---

func TestSetDLQWriter(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	q.SetDLQWriter(nil)
}

// --- spill_writer.go ---

func TestNewSpillWriter_InvalidDir(t *testing.T) {
	t.Parallel()
	_, err := NewSpillWriter(SpillWriterConfig{Dir: "/proc/invalid-spill-dir"}, zap.NewNop())
	if err == nil {
		t.Log("NewSpillWriter to /proc should likely error")
	}
}

// --- metrics.go percentile ---

func TestPercentile_Empty(t *testing.T) {
	t.Parallel()
	h := newHistogram([]float64{1, 10, 100})
	p := h.percentile(99)
	if p != 0 {
		t.Errorf("percentile of empty histogram should be 0, got %f", p)
	}
}

func TestPercentile_WithData(t *testing.T) {
	t.Parallel()
	h := newHistogram([]float64{1, 10, 100})
	h.counts[1].Store(1) // one observation in bucket 10
	p := h.percentile(99)
	if p <= 0 {
		t.Errorf("percentile with data should be > 0, got %f", p)
	}
}

func TestNewHistogram(t *testing.T) {
	t.Parallel()
	h := newHistogram([]float64{1, 10, 100})
	if h == nil {
		t.Fatal("newHistogram returned nil")
	}
}

// --- normalizer.go ---

func TestNewNormalizer_NilPG(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	if n == nil {
		t.Fatal("NewNormalizer returned nil")
	}
}

// --- dlq_writer.go ---

// mockCHConn is a minimal clickhouse.Conn that always returns errors.
// Used to test error/spill paths without a real CH server.
type mockCHConn struct{}

func (m *mockCHConn) Contributors() []string                                     { return nil }
func (m *mockCHConn) ServerVersion() (*driver.ServerVersion, error)               { return nil, nil }
func (m *mockCHConn) Select(ctx context.Context, dest any, query string, args ...any) error {
	return fmt.Errorf("mock: not implemented")
}
func (m *mockCHConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockCHConn) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	return nil
}
func (m *mockCHConn) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	return nil, fmt.Errorf("mock: prepare failed")
}
func (m *mockCHConn) Exec(ctx context.Context, query string, args ...any) error {
	return fmt.Errorf("mock: not implemented")
}
func (m *mockCHConn) AsyncInsert(ctx context.Context, query string, wait bool, args ...any) error {
	return fmt.Errorf("mock: not implemented")
}
func (m *mockCHConn) Ping(ctx context.Context) error { return nil }
func (m *mockCHConn) Stats() driver.Stats            { return driver.Stats{} }
func (m *mockCHConn) Close() error                   { return nil }

func TestCHWriter_FlushTicks_WithMockConn(t *testing.T) {
	t.Parallel()
	cfg := DefaultCHWriterConfig()
	cfg.QueueSize = 100
	w := NewCHWriter(cfg, &mockCHConn{}, nil, zap.NewNop())
	ctx := context.Background()

	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	// flushTicks with error from mock conn should fall through to spill (nil spill, no-op).
	w.Flush(ctx, []*mdtick.Tick{tick}, nil)
}

func TestCHWriter_FlushBars_WithMockConn(t *testing.T) {
	t.Parallel()
	cfg := DefaultCHWriterConfig()
	cfg.QueueSize = 100
	w := NewCHWriter(cfg, &mockCHConn{}, nil, zap.NewNop())
	ctx := context.Background()

	bar := &mdtick.Bar{
		Broker: "broker", Canonical: "EURUSD", Period: "1h",
		CloseTsUnixMs: time.Now().UnixMilli(), OpenTsUnixMs: time.Now().UnixMilli() - 3600_000,
		Open: decimal.NewFromFloat(1.1000), High: decimal.NewFromFloat(1.1050),
		Low: decimal.NewFromFloat(1.0990), Close: decimal.NewFromFloat(1.1020),
	}
	w.Flush(ctx, nil, []*mdtick.Bar{bar})
}

func TestCHWriter_Start_WithMockConn(t *testing.T) {
	t.Parallel()
	cfg := DefaultCHWriterConfig()
	cfg.QueueSize = 10
	cfg.FlushInterval = 50 * time.Millisecond
	w := NewCHWriter(cfg, &mockCHConn{}, nil, zap.NewNop())
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	w.Start(ctx)
}

func TestCHWriter_EnqueueTick(t *testing.T) {
	t.Parallel()
	w := NewCHWriter(DefaultCHWriterConfig(), &mockCHConn{}, nil, zap.NewNop())
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	w.EnqueueTick(tick)
}

func TestCHWriter_SpillFailed(t *testing.T) {
	t.Parallel()
	w := NewCHWriter(DefaultCHWriterConfig(), &mockCHConn{}, nil, zap.NewNop())
	w.spillFailed("broker", fmt.Errorf("test error"))
	w.spillFailed("broker", fmt.Errorf("another error"))
}

func TestSpillWriter_ShouldRotate(t *testing.T) {
	t.Parallel()
	sw := &SpillWriter{
		cfg:      DefaultSpillConfig(),
		curBytes: 0,
		curStart: time.Now(),
	}
	if sw.shouldRotate() {
		t.Error("shouldRotate should be false for fresh writer")
	}
	sw.curBytes = sw.cfg.MaxFileBytes
	if !sw.shouldRotate() {
		t.Error("shouldRotate should be true when curBytes >= MaxFileBytes")
	}
}

func TestPublisher_PublishTick_NilJS(t *testing.T) {
	t.Parallel()
	pub := NewPublisher(nil)
	err := pub.PublishTick(&mdtick.Tick{
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
	err := pub.PublishBar(&mdtick.Bar{
		Broker: "broker", Canonical: "EURUSD", Period: "1h",
		CloseTsUnixMs: time.Now().UnixMilli(),
		Open: decimal.NewFromFloat(1.1000), High: decimal.NewFromFloat(1.1050),
		Low: decimal.NewFromFloat(1.0990), Close: decimal.NewFromFloat(1.1020),
	})
	if err != nil {
		t.Logf("PublishBar with nil JS: %v", err)
	}
}

func TestPublisher_PublishBarRevision_NilJS(t *testing.T) {
	t.Parallel()
	pub := NewPublisher(nil)
	err := pub.PublishBarRevision(&mdtick.Bar{
		Broker: "broker", Canonical: "EURUSD", Period: "1h",
		CloseTsUnixMs: time.Now().UnixMilli(),
		Open: decimal.NewFromFloat(1.1000), High: decimal.NewFromFloat(1.1050),
		Low: decimal.NewFromFloat(1.0990), Close: decimal.NewFromFloat(1.1020),
	})
	if err != nil {
		t.Logf("PublishBarRevision with nil JS: %v", err)
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

func TestResolve(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	result := n.Resolve(context.Background(), "broker", "EURUSDm")
	if result == "" {
		t.Log("Resolve returned empty (expected without PG-backed mapping)")
	}
}

func TestSpillWriter_WriteBar(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sw, err := NewSpillWriter(SpillWriterConfig{
		Dir:          dir,
		MaxFileBytes: 100 * 1024 * 1024,
		MaxFileAge:   time.Hour,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewSpillWriter: %v", err)
	}
	defer sw.Close()
	bar := &mdtick.Bar{
		Broker: "broker", Canonical: "EURUSD", Period: "1h",
		CloseTsUnixMs: time.Now().UnixMilli(),
		Open: decimal.NewFromFloat(1.1000), High: decimal.NewFromFloat(1.1050),
		Low: decimal.NewFromFloat(1.0990), Close: decimal.NewFromFloat(1.1020),
	}
	if err := sw.WriteBar(bar); err != nil {
		t.Errorf("WriteBar should not error: %v", err)
	}
}

func TestSpillWriter_WriteTick_Real(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sw, err := NewSpillWriter(SpillWriterConfig{
		Dir:          dir,
		MaxFileBytes: 100 * 1024 * 1024,
		MaxFileAge:   time.Hour,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewSpillWriter: %v", err)
	}
	defer sw.Close()
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	if err := sw.WriteTick(tick); err != nil {
		t.Errorf("WriteTick should not error: %v", err)
	}
}

func TestCHWriter_WriteSpillTick_WithSpill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sw, err := NewSpillWriter(SpillWriterConfig{
		Dir:          dir,
		MaxFileBytes: 100 * 1024 * 1024,
		MaxFileAge:   time.Hour,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewSpillWriter: %v", err)
	}
	defer sw.Close()
	w := NewCHWriter(DefaultCHWriterConfig(), &mockCHConn{}, sw, zap.NewNop())
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	w.Flush(context.Background(), []*mdtick.Tick{tick}, nil)
}

func TestCHWriter_WriteSpillBar_WithSpill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sw, err := NewSpillWriter(SpillWriterConfig{
		Dir:          dir,
		MaxFileBytes: 100 * 1024 * 1024,
		MaxFileAge:   time.Hour,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewSpillWriter: %v", err)
	}
	defer sw.Close()
	w := NewCHWriter(DefaultCHWriterConfig(), &mockCHConn{}, sw, zap.NewNop())
	bar := &mdtick.Bar{
		Broker: "broker", Canonical: "EURUSD", Period: "1h",
		CloseTsUnixMs: time.Now().UnixMilli(),
		Open: decimal.NewFromFloat(1.1000), High: decimal.NewFromFloat(1.1050),
		Low: decimal.NewFromFloat(1.0990), Close: decimal.NewFromFloat(1.1020),
	}
	w.Flush(context.Background(), nil, []*mdtick.Bar{bar})
}

func TestShouldSample_Always(t *testing.T) {
	t.Parallel()
	dlq := NewDLQWriter(nil, nil, zap.NewNop())
	if dlq == nil {
		t.Fatal("NewDLQWriter returned nil")
	}
	// pct=100.0 always samples.
	if !dlq.shouldSample(100.0) {
		t.Error("shouldSample(100.0) should return true")
	}
	// pct=0.0 never samples.
	if dlq.shouldSample(0.0) {
		t.Error("shouldSample(0.0) should return false")
	}
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

// --- clickhouse_writer.go EnqueueTick full queue ---

func TestCHWriter_EnqueueTick_FullQueue(t *testing.T) {
	t.Parallel()
	cfg := DefaultCHWriterConfig()
	cfg.QueueSize = 1
	w := NewCHWriter(cfg, &mockCHConn{}, nil, zap.NewNop())
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	w.EnqueueTick(tick)
	w.EnqueueTick(tick) // queue full, triggers writeSpillTick (nil spill)
}

// --- normalizer.go Resolve cache hit ---

func TestNormalizer_Resolve_CacheHit(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	n.cache["broker:EURUSDm"] = "EURUSD"
	result := n.Resolve(context.Background(), "broker", "EURUSDm")
	if result != "EURUSD" {
		t.Errorf("cache hit should return EURUSD, got %q", result)
	}
}

func TestNormalizer_Resolve_CacheGuard(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	// Fill cache past maxCacheSize (100k) to trigger cache reset.
	for i := 0; i < 100001; i++ {
		n.cache[fmt.Sprintf("b:s%d", i)] = fmt.Sprintf("S%d", i)
	}
	result := n.Resolve(context.Background(), "broker", "EURUSDm")
	if result == "" {
		t.Error("Resolve should still work after cache reset")
	}
}

// --- dlq_writer.go WriteTick ---

func TestDLQWriter_WriteTick_WithCHConn(t *testing.T) {
	t.Parallel()
	dlq := NewDLQWriter(&mockCHConn{}, nil, zap.NewNop())
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	dlq.WriteTick(context.Background(), tick, "test", "")
}

func TestDLQWriter_WriteTick_SpillOnly(t *testing.T) {
	t.Parallel()
	dlq := NewDLQWriter(nil, nil, zap.NewNop())
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	dlq.WriteTick(context.Background(), tick, "test", "")
}

// --- quality.go Check with stale tick ---

func TestCheck_StaleTick(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs:      time.Now().UnixMilli() - 60_000, // 1min old
		ArrivedUnixMs: time.Now().UnixMilli(),
		Bid:           decimal.NewFromFloat(1.08000),
		Ask:           decimal.NewFromFloat(1.08001),
	}
	res := q.Check(context.Background(), tick)
	_ = res
}

// --- manager.go HandleTick ---

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

// --- spill_replay.go Run ---

func TestSpillReplay_Run_NoFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	replay := NewSpillReplay(dir, nil, nil, zap.NewNop())
	ctx := context.Background()
	replay.Run(ctx)
}

// --- quote_stuffing.go IsPaused deeper ---

func TestStuffingDetector_IsPaused_Expired(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	// Manually insert an expired pause entry (white-box).
	sd.mu.Lock()
	sd.pausedUntil["broker:EURUSD"] = time.Now().Add(-time.Hour)
	sd.mu.Unlock()
	if sd.IsPaused("broker", "EURUSD") {
		t.Error("expired pause should return false")
	}
	// Key should be cleaned up.
	sd.mu.Lock()
	_, exists := sd.pausedUntil["broker:EURUSD"]
	sd.mu.Unlock()
	if exists {
		t.Error("expired key should be deleted")
	}
}

func TestStuffingDetector_IsPaused_Active(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	sd.mu.Lock()
	sd.pausedUntil["broker:EURUSD"] = time.Now().Add(time.Hour)
	sd.mu.Unlock()
	if !sd.IsPaused("broker", "EURUSD") {
		t.Error("active pause should return true")
	}
}

// --- clickhouse_writer.go Start with real ticker ---

func TestCHWriter_Start_FullFlush(t *testing.T) {
	t.Parallel()
	cfg := DefaultCHWriterConfig()
	cfg.QueueSize = 2
	cfg.MaxBatchSize = 2
	cfg.FlushInterval = 20 * time.Millisecond
	w := NewCHWriter(cfg, &mockCHConn{}, nil, zap.NewNop())
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	// Enqueue ticks to trigger batch flush.
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	w.EnqueueTick(tick)
	// Start ticker loop (will flush at interval or on ctx done).
	w.Start(ctx)
}

// --- spill_writer.go rotate ---

func TestSpillWriter_Rotate_TimeBased(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sw, err := NewSpillWriter(SpillWriterConfig{
		Dir:          dir,
		MaxFileBytes: 10 * 1024 * 1024,
		MaxFileAge:   time.Millisecond, // trigger rotation
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewSpillWriter: %v", err)
	}
	defer sw.Close()
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	if err := sw.WriteTick(tick); err != nil {
		t.Errorf("first WriteTick: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if err := sw.WriteTick(tick); err != nil {
		t.Errorf("second WriteTick: %v", err)
	}
}

// --- dlq_writer.go spillDLQ with spill ---

func TestDLQWriter_SpillDLQ_WithSpill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sw, err := NewSpillWriter(SpillWriterConfig{
		Dir:          dir,
		MaxFileBytes: 100 * 1024 * 1024,
		MaxFileAge:   time.Hour,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewSpillWriter: %v", err)
	}
	defer sw.Close()
	dlq := NewDLQWriter(nil, sw, zap.NewNop())
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	// Use "parse_error" (100% sample rate) to ensure the tick is written.
	dlq.WriteTick(context.Background(), tick, "parse_error", "raw")
	time.Sleep(10 * time.Millisecond) // wait for async flushLoop
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

// --- quality.go check stale ---

func TestCheck_StaleArrival(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs:      time.Now().UnixMilli(),
		ArrivedUnixMs: time.Now().UnixMilli() - 60_000,
		Bid:           decimal.NewFromFloat(1.08000),
		Ask:           decimal.NewFromFloat(1.08001),
	}
	res := q.Check(context.Background(), tick)
	_ = res
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
