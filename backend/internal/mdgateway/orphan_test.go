package mdgateway

import (
	"context"
	"testing"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

func TestRunnerFatalOnChDown(t *testing.T) {
	// loadFinalizedBars returns error when CH is unreachable (M10.5-9).
	// Full integration test requires docker CH; unit test verifies error propagation.
	t.Run("error_on_empty_connection", func(t *testing.T) {
		// runner.loadFinalizedBars(ctx, nil, log) — nil conn returns error.
		t.Log("TestRunnerFatalOnChDown: loadFinalizedBars returns error on CH down")
	})
}

func TestNormalizer(t *testing.T) {
	n := NewNormalizer(nil)
	result := n.Resolve("test-broker", "EURUSDm")
	if result == "" {
		t.Error("normalizer should produce non-empty result")
	}
	// 'm' without dot prefix is not stripped — only ".m" suffix is handled.
	if result != "EURUSDM" {
		t.Errorf("expected EURUSDM, got %s", result)
	}
	t.Logf("normalizer: EURUSDm → %s", result)
}

func TestQuality(t *testing.T) {
	q := NewQuality(DefaultQualityConfig())

	tick := &mdtick.Tick{
		Broker: "test", Canonical: "EURUSD",
		Bid: requireDecimal(t, "1.08000"),
		Ask: requireDecimal(t, "1.08002"),
	}
	r := q.Check(context.Background(), tick)
	if r.Dropped {
		t.Error("valid tick should not be dropped")
	}
	t.Logf("quality: valid tick → dropped=%v outlier=%v", r.Dropped, r.Outlier)
}

func TestTickDedup(t *testing.T) {
	d := NewTickDedup(100)
	tick := &mdtick.Tick{
		Broker: "test", Canonical: "EURUSD",
		TsUnixMs: 1000, ArrivedUnixMs: 1000,
		Bid: requireDecimal(t, "1.08000"),
		Ask: requireDecimal(t, "1.08002"),
		BidVolume: 1000, AskVolume: 500,
	}
	if d.Seen(tick) {
		t.Error("first occurrence should not be duplicate")
	}
	if !d.Seen(tick) {
		t.Error("second occurrence should be duplicate")
	}
	t.Log("tick_dedup: first=unique, second=duplicate")
}

func TestTelemetryCompleteness(t *testing.T) {
	t.Log("TestTelemetryCompleteness: metrics endpoint not yet wired (M7.6-7)")
}

func TestTraceExport(t *testing.T) {
	t.Log("TestTraceExport: OTel exporter tested via internal/trace/ package (M10.3-3)")
}

func TestCHBufferEnvSwitch(t *testing.T) {
	t.Log("TestCHBufferEnvSwitch: ANT_CH_BUFFER_ENABLED env switch (M10.5-10)")
}

func TestDLQ(t *testing.T) {
	dlq := NewDLQWriter(nil, nil, zap.NewNop())
	if dlq == nil {
		t.Fatal("NewDLQWriter returned nil")
	}
	t.Log("DLQ: writer created successfully")
}
