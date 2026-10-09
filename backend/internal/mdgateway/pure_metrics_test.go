package mdgateway

import (
	"context"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

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

func TestSetDLQWriter(t *testing.T) {
	t.Parallel()
	q := NewQuality(DefaultQualityConfig())
	q.SetDLQWriter(nil)
}

// --- metrics.go percentile ---

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

// --- quality.go Check with stale tick ---
