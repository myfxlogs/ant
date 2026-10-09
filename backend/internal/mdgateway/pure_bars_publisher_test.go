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

// --- normalizer_invalidator.go ---

func TestNewPublisher_NilJS(t *testing.T) {
	t.Parallel()
	pub := NewPublisher(nil)
	if pub == nil {
		t.Fatal("NewPublisher(nil) returned nil")
	}
}

// --- metrics.go ---

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

// --- normalizer.go Resolve cache hit ---
