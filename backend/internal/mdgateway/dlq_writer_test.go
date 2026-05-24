package mdgateway

import (
	"context"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

func TestDLQParseError(t *testing.T) {
	// DLQ with nil CH conn — writes should be no-ops (spill fallback or skip).
	dlq := NewDLQWriter(nil, nil, zap.NewNop())
	if dlq == nil {
		t.Fatal("NewDLQWriter returned nil")
	}

	tick := &mdtick.Tick{
		Broker: "test-broker", Canonical: "EURUSD",
		TsUnixMs: 1000, ArrivedUnixMs: 1000,
		Bid: requireDecimal(t, "1.08000"), Ask: requireDecimal(t, "1.08002"),
	}

	// parse_error: 100% sampling — all writes should be attempted.
	dlq.WriteTick(context.Background(), tick, "parse_error", `{"raw":"data"}`)
	dlq.WriteTick(context.Background(), tick, "parse_error", `{"raw":"data2"}`)
	dlq.WriteTick(context.Background(), tick, "parse_error", `{"raw":"data3"}`)

	t.Log("DLQParseError: 3 parse_error writes attempted (nil conn — no crash)")
}

func TestDLQSampling(t *testing.T) {
	dlq := NewDLQWriter(nil, nil, zap.NewNop())

	tick := &mdtick.Tick{
		Broker: "test-broker", Canonical: "EURUSD",
		TsUnixMs: 1000, ArrivedUnixMs: 1000,
		Bid: requireDecimal(t, "1.08000"), Ask: requireDecimal(t, "1.08002"),
	}

	// bid_gt_ask: 1% sampling — most should be skipped.
	for i := 0; i < 1000; i++ {
		dlq.WriteTick(context.Background(), tick, "bid_gt_ask", "")
	}

	// non_positive: 1% sampling.
	for i := 0; i < 500; i++ {
		dlq.WriteTick(context.Background(), tick, "non_positive", "")
	}

	t.Log("DLQSampling: 1500 writes attempted at 1% rate (nil conn — no crash)")
}

var _ clickhouse.Conn = nil
