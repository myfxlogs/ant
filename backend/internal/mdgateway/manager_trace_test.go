package mdgateway

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"

	anttrace "anttrader/internal/trace"
	"anttrader/internal/mdgateway/adapter/mdtick"
)

// TestHandleTickCreatesSixSpans verifies the ADR-0010 §2.3 requirement:
// HandleTick creates one OTel span for each of the 6 pipeline stages.
// L-2: this test proves tracing is not dead code — spans are observed and verifiable.
func TestHandleTickCreatesSixSpans(t *testing.T) {
	// 1. Create in-memory exporter + TracerProvider with AlwaysSample.
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithSampler(trace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())

	tr := anttrace.NewWithProvider(tp)
	if !tr.Enabled() {
		t.Fatal("NewWithProvider: tracer should be enabled")
	}

	// 2. Wire tracer into a minimal Manager.
	mgr := testManager()
	mgr.SetOTelTracer(tr)

	// 3. Feed a valid tick through the pipeline.
	now := time.Now().UnixMilli()
	bid := decimal.NewFromFloat(1.0800)
	ask := decimal.NewFromFloat(1.0802)
	tk := &mdtick.Tick{
		UserID:        "test-user",
		AccountID:     "test-acc",
		Broker:        "test-broker",
		SymbolRaw:     "EURUSD",
		Canonical:     "EURUSD",
		TsUnixMs:      now,
		ArrivedUnixMs: now,
		Bid:           bid,
		Ask:           ask,
		BidVolume:     1000,
		AskVolume:     1000,
	}
	mgr.HandleTick(tk)

	// 4. Force flush + collect spans.
	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("ForceFlush: %v", err)
	}

	spans := exp.GetSpans()
	spanNames := make(map[string]int)
	for _, s := range spans {
		spanNames[s.Name]++
	}
	t.Logf("Observed spans: %v", spanNames)

	// 5. Assert all 6 pipeline stages appear.
	want := []string{"normalize", "quality", "dedup", "aggregate", "publish", "enqueue"}
	for _, name := range want {
		n := spanNames[name]
		if n == 0 {
			t.Errorf("missing span %q — pipeline stage not traced", name)
		}
	}
	if t.Failed() {
		t.Log("This means the OTel tracer is NOT wired through HandleTick. Found spans:")
		for _, s := range spans {
			t.Logf("  span: %s", s.Name)
		}
	}
}

// HandleTick helper: minimal deps that won't panic.
func testManager() *Manager {
	nopLog := zap.NewNop()
	chCfg := DefaultCHWriterConfig()
	chCfg.QueueSize = 100
	return NewManager(ManagerDeps{
		Normalizer: NewNormalizer(nil),
		Quality:    NewQuality(DefaultQualityConfig()),
		Dedup:      NewTickDedup(10),
		Aggregator: NewBarAggregator(),
		Publisher:  NewPublisher(nil),
		CHWriter:   NewCHWriter(chCfg, nil, nil, nopLog),
	})
}

// TestHandleTickNoOpWhenTracerNil verifies that HandleTick does not panic
// when no OTel tracer is set (nil = no-op, not crash).
func TestHandleTickNoOpWhenTracerNil(t *testing.T) {
	mgr := testManager()
	// No SetOTelTracer call — tracer is nil.

	now := time.Now().UnixMilli()
	tk := &mdtick.Tick{
		UserID:        "test-user",
		AccountID:     "test-acc",
		Broker:        "test-broker",
		SymbolRaw:     "EURUSD",
		Canonical:     "EURUSD",
		TsUnixMs:      now,
		ArrivedUnixMs: now,
		Bid:           decimal.NewFromFloat(1.0800),
		Ask:           decimal.NewFromFloat(1.0802),
		BidVolume:     1000,
		AskVolume:     1000,
	}

	// Must not panic.
	mgr.HandleTick(tk)
	t.Log("HandleTick with nil tracer: no panic (PASS)")
}

// TestTraceSamplingHonored verifies that when sampling is disabled (0%),
// no spans are exported — confirming the sampling config actually works.
func TestTraceSamplingHonored(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	// DropAllSampler = never sample (0%).
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithSampler(trace.NeverSample()),
	)
	defer tp.Shutdown(context.Background())

	tr := anttrace.NewWithProvider(tp)
	if !tr.Enabled() {
		t.Fatal("NewWithProvider: tracer should be enabled even with NeverSample")
	}

	mgr := testManager()
	mgr.SetOTelTracer(tr)

	now := time.Now().UnixMilli()
	tk := &mdtick.Tick{
		UserID:        "test-user",
		AccountID:     "test-acc",
		Broker:        "test-broker",
		SymbolRaw:     "EURUSD",
		Canonical:     "EURUSD",
		TsUnixMs:      now,
		ArrivedUnixMs: now,
		Bid:           decimal.NewFromFloat(1.0800),
		Ask:           decimal.NewFromFloat(1.0802),
		BidVolume:     1000,
		AskVolume:     1000,
	}
	mgr.HandleTick(tk)
	_ = tp.ForceFlush(context.Background())

	spans := exp.GetSpans()
	if len(spans) > 0 {
		t.Errorf("NeverSample produced %d spans, want 0 — sampling not honored", len(spans))
		for _, s := range spans {
			t.Logf("  unexpected span: %s", s.Name)
		}
	} else {
		t.Log("NeverSample: 0 spans (PASS)")
	}
}
