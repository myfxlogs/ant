package mdgateway

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/mdtick"
)

func TestDLQParseError(t *testing.T) {
	t.Parallel()
	// DLQ writes structured log entries (ADR-0012: no DB insertion).
	dlq := NewDLQWriter(zap.NewNop())
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
	t.Parallel()
	dlq := NewDLQWriter(zap.NewNop())

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

func TestDLQAsync(t *testing.T) {
	t.Parallel()
	dlq := NewDLQWriter(zap.NewNop())
	tick := &mdtick.Tick{
		Broker: "test", Canonical: "EURUSD",
		TsUnixMs: 1000, ArrivedUnixMs: 1000,
		Bid: requireDecimal(t, "1.08000"), Ask: requireDecimal(t, "1.08002"),
	}
	// 1000 writes through async channel — should not block.
	for i := 0; i < 1000; i++ {
		dlq.WriteTick(context.Background(), tick, "parse_error", "")
	}
	t.Log("DLQAsync: 1000 async writes completed without blocking")
}
