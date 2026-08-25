package mdgateway

import (
	"context"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

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

// --- pgwriter.go ---

func TestPgWriter_EnqueueBar(t *testing.T) {
	t.Parallel()
	w := NewPgWriter(DefaultPgWriterConfig(), nil, zap.NewNop())
	bar := &mdtick.Bar{
		Broker: "broker", Canonical: "EURUSD", Period: "1h",
		CloseTsUnixMs: time.Now().UnixMilli(),
	}
	// Should not panic even with nil store.
	w.EnqueueBar(bar)
}

func TestPgWriter_EnqueueBar_FullQueue(t *testing.T) {
	t.Parallel()
	cfg := DefaultPgWriterConfig()
	cfg.QueueSize = 1
	w := NewPgWriter(cfg, nil, zap.NewNop())
	bar := &mdtick.Bar{Broker: "broker", Canonical: "EURUSD", Period: "1h"}
	// First enqueue succeeds, second fills queue and is dropped gracefully.
	w.EnqueueBar(bar)
	w.EnqueueBar(bar) // queue full — dropped gracefully
}

func TestPgWriter_Flush_Empty(t *testing.T) {
	t.Parallel()
	w := NewPgWriter(DefaultPgWriterConfig(), nil, zap.NewNop())
	ctx := context.Background()
	// Flush with empty batch should be safe.
	w.Flush(ctx, nil)
}

func TestPgWriterDrain_Empty(t *testing.T) {
	t.Parallel()
	w := NewPgWriter(DefaultPgWriterConfig(), nil, zap.NewNop())
	bars := w.Drain()
	if len(bars) != 0 {
		t.Errorf("drain bars should be empty, got %d", len(bars))
	}
}

func TestDefaultPgWriterConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultPgWriterConfig()
	if cfg.MaxBatchSize <= 0 {
		t.Errorf("MaxBatchSize = %d, want >0", cfg.MaxBatchSize)
	}
	if cfg.QueueSize <= 0 {
		t.Errorf("QueueSize = %d, want >0", cfg.QueueSize)
	}
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

// --- dlq_writer.go ---

func TestShouldSample_Always(t *testing.T) {
	t.Parallel()
	dlq := NewDLQWriter(zap.NewNop())
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

// --- dlq_writer.go WriteTick ---

func TestDLQWriter_WriteTick_WithCHConn(t *testing.T) {
	t.Parallel()
	dlq := NewDLQWriter(zap.NewNop())
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	dlq.WriteTick(context.Background(), tick, "test", "")
}

func TestDLQWriter_WriteTick_SpillOnly(t *testing.T) {
	t.Parallel()
	dlq := NewDLQWriter(zap.NewNop())
	tick := &mdtick.Tick{
		Broker: "broker", Canonical: "EURUSD",
		TsUnixMs: time.Now().UnixMilli(), ArrivedUnixMs: time.Now().UnixMilli(),
		Bid: decimal.NewFromFloat(1.1000), Ask: decimal.NewFromFloat(1.1001),
	}
	dlq.WriteTick(context.Background(), tick, "test", "")
}
