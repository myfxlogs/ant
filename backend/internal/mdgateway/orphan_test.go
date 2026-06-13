package mdgateway

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// TestLoadFinalizedBarsErrorPropagation verifies that loadFinalizedBars
// propagates errors from the MarketDataStore (PG or CH) rather than
// silently swallowing them. Bar finality would be silently disabled otherwise.
func TestLoadFinalizedBarsErrorPropagation(t *testing.T) {
	t.Parallel()
	log := zap.NewNop()
	// Use a nil store — LoadFinalizedBars will fail with a nil pointer.
	rows, err := loadFinalizedBars(context.Background(), nil, log)
	if err == nil {
		t.Fatal("expected non-nil error when store is nil; got nil (would silently disable bar finality)")
	}
	if rows != nil {
		t.Fatalf("expected nil result on error, got %d keys", len(rows))
	}
	if err.Error() == "" {
		t.Fatalf("error should be non-empty")
	}
	t.Logf("loadFinalizedBars correctly returned error on store failure: %v", err)
}

func TestNormalizer(t *testing.T) {
	t.Parallel()
	n := NewNormalizer(nil)
	result := n.Resolve(context.Background(), "test-broker", "EURUSDm")
	if result == "" {
		t.Error("normalizer should produce non-empty result")
	}
	// K-line suffix fix (98918d7): raw broker symbol IS canonical.
	// Suffix stripping caused mismatches — brokers don't recognize
	// stripped forms for historical queries.
	if result != "EURUSDm" {
		t.Errorf("expected EURUSDm (raw symbol preserved), got %s", result)
	}
	t.Logf("normalizer: EURUSDm → %s", result)
}

func TestQuality(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	t.Log("TestTelemetryCompleteness: metrics endpoint not yet wired (M7.6-7)")
}

func TestTraceExport(t *testing.T) {
	t.Parallel()
	t.Log("TestTraceExport: OTel exporter tested via internal/trace/ package (M10.3-3)")
}

func TestDLQ(t *testing.T) {
	t.Parallel()
	dlq := NewDLQWriter(zap.NewNop())
	if dlq == nil {
		t.Fatal("NewDLQWriter returned nil")
	}
	t.Log("DLQ: writer created successfully")
}
