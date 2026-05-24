package mdgateway

import (
	"context"
	"errors"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// errCHConn is a clickhouse.Conn stub whose Query always fails. It exercises
// loadFinalizedBars's error path (M10.5-9 M-3): runner must NOT silently swallow
// CH-unreachable errors at startup — bar finality would be disabled otherwise.
type errCHConn struct {
	clickhouse.Conn
}

func (e *errCHConn) Query(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
	return nil, errors.New("simulated CH unreachable")
}

func TestRunnerFatalOnChDown(t *testing.T) {
	log := zap.NewNop()
	// Real assertion: loadFinalizedBars must propagate the underlying CH error.
	rows, err := loadFinalizedBars(context.Background(), &errCHConn{}, log)
	if err == nil {
		t.Fatal("expected non-nil error when CH Query fails; got nil (would silently disable bar finality)")
	}
	if rows != nil {
		t.Fatalf("expected nil result on error, got %d keys", len(rows))
	}
	if err.Error() == "" {
		t.Fatalf("error should be non-empty")
	}
	t.Logf("loadFinalizedBars correctly returned error on CH-down: %v", err)
}

func TestCHBufferEnvSwitch(t *testing.T) {
	// M10.5-10 S-2: ANT_CH_BUFFER_ENABLED env switches CH INSERT target tables.
	// default → md_ticks_buffer / md_bars_buffer (Buffer engine, ADR-0011)
	// =false  → md_ticks / md_bars (direct write, Buffer bypass)
	t.Run("default_uses_buffer", func(t *testing.T) {
		t.Setenv("ANT_CH_BUFFER_ENABLED", "")
		if got := tickTargetTable(); got != "md_ticks_buffer" {
			t.Errorf("default tickTargetTable=%q, want md_ticks_buffer", got)
		}
		if got := barTargetTable(); got != "md_bars_buffer" {
			t.Errorf("default barTargetTable=%q, want md_bars_buffer", got)
		}
	})
	t.Run("false_bypasses_buffer", func(t *testing.T) {
		t.Setenv("ANT_CH_BUFFER_ENABLED", "false")
		if got := tickTargetTable(); got != "md_ticks" {
			t.Errorf("env=false tickTargetTable=%q, want md_ticks", got)
		}
		if got := barTargetTable(); got != "md_bars" {
			t.Errorf("env=false barTargetTable=%q, want md_bars", got)
		}
	})
	t.Run("any_other_value_uses_buffer", func(t *testing.T) {
		t.Setenv("ANT_CH_BUFFER_ENABLED", "true")
		if got := tickTargetTable(); got != "md_ticks_buffer" {
			t.Errorf("env=true tickTargetTable=%q, want md_ticks_buffer", got)
		}
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

func TestDLQ(t *testing.T) {
	dlq := NewDLQWriter(nil, nil, zap.NewNop())
	if dlq == nil {
		t.Fatal("NewDLQWriter returned nil")
	}
	t.Log("DLQ: writer created successfully")
}
