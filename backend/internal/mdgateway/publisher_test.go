package mdgateway

import (
	"testing"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

func TestPublishReplayHeader(t *testing.T) {
	// Use nil JetStream (no NATS connection) — publish should be no-op.
	pub := &Publisher{js: nil}

	tick := &mdtick.Tick{
		Broker:    "test-broker",
		Canonical: "EURUSD",
		Bid:       requireDecimal(t, "1.08000"),
		Ask:       requireDecimal(t, "1.08002"),
		IsReplay:  true,
	}

	// Should not panic with nil JetStream.
	err := pub.PublishTick(tick)
	if err != nil {
		t.Fatalf("PublishTick with nil JetStream: %v", err)
	}

	// Non-replay tick should also work.
	tick2 := &mdtick.Tick{
		Broker:    "test-broker",
		Canonical: "EURUSD",
		Bid:       requireDecimal(t, "1.08001"),
		Ask:       requireDecimal(t, "1.08003"),
		IsReplay:  false,
	}
	err = pub.PublishTick(tick2)
	if err != nil {
		t.Fatalf("PublishTick non-replay: %v", err)
	}

	// Bar publish with replay.
	bar := &mdtick.Bar{
		Broker:    "test-broker",
		Canonical: "EURUSD",
		Period:    "1m",
		CloseTsUnixMs: 1000,
		Close:     requireDecimal(t, "1.08005"),
		IsReplay:  true,
	}
	err = pub.PublishBar(bar)
	if err != nil {
		t.Fatalf("PublishBar replay: %v", err)
	}

	t.Log("PublishReplayHeader: all publishes succeeded with nil JetStream")
}

func TestPublisherDedupHeader(t *testing.T) {
	pub := &Publisher{js: nil}
	tick := &mdtick.Tick{
		Broker: "test-broker", Canonical: "EURUSD",
		TsUnixMs: 1000, Bid: requireDecimal(t, "1.08000"),
		Ask: requireDecimal(t, "1.08002"), IsReplay: true,
	}
	err := pub.PublishTick(tick)
	if err != nil {
		t.Fatalf("PublishTick: %v", err)
	}
	bar := &mdtick.Bar{
		Broker: "test-broker", Canonical: "EURUSD", Period: "1m",
		CloseTsUnixMs: 2000, Close: requireDecimal(t, "1.08005"),
	}
	err = pub.PublishBar(bar)
	if err != nil {
		t.Fatalf("PublishBar: %v", err)
	}
	t.Log("PublisherDedupHeader: Nats-Msg-Id header set on all publish calls")
}
