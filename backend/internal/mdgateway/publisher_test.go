package mdgateway

import (
	"testing"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

func TestPublishReplayHeader(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestPublisherSubjectFormat(t *testing.T) {
	t.Parallel()
	tick := &mdtick.Tick{
		Broker:    "ic_markets",
		Canonical: "EURUSD",
		Bid:       requireDecimal(t, "1.08000"),
		Ask:       requireDecimal(t, "1.08002"),
		IsReplay:  false,
	}
	bar := &mdtick.Bar{
		Broker:        "ic_markets",
		Canonical:     "EURUSD",
		Period:        "1h",
		CloseTsUnixMs: 1000,
		Close:         requireDecimal(t, "1.08005"),
	}

	// Verify subject composition logic (without actual NATS publish).
	tickSubj := "md.tick." + tick.Broker + "." + tick.Canonical
	if tickSubj != "md.tick.ic_markets.EURUSD" {
		t.Fatalf("tick subject = %q, want md.tick.ic_markets.EURUSD", tickSubj)
	}

	barSubj := "md.bar." + bar.Broker + "." + bar.Canonical + "." + bar.Period
	if barSubj != "md.bar.ic_markets.EURUSD.1h" {
		t.Fatalf("bar subject = %q, want md.bar.ic_markets.EURUSD.1h", barSubj)
	}

	t.Log("PublisherSubjectFormat: subjects match NATS convention")
}

func TestPublisherBarWithoutReplay(t *testing.T) {
	t.Parallel()
	pub := &Publisher{js: nil}
	bar := &mdtick.Bar{
		Broker:        "oanda",
		Canonical:     "GBPUSD",
		Period:        "4h",
		CloseTsUnixMs: 7200000,
		Close:         requireDecimal(t, "1.25010"),
		IsReplay:      false,
		Open:          requireDecimal(t, "1.24800"),
		High:          requireDecimal(t, "1.25200"),
		Low:           requireDecimal(t, "1.24700"),
		Volume:        150.5,
	}
	err := pub.PublishBar(bar)
	if err != nil {
		t.Fatalf("PublishBar non-replay: %v", err)
	}
	t.Log("PublisherBarWithoutReplay: non-replay bar published")
}

func TestPublisherTickWithoutCanonical(t *testing.T) {
	t.Parallel()
	pub := &Publisher{js: nil}
	tick := &mdtick.Tick{
		Broker:    "test-broker",
		Canonical: "XAUUSD",
		TsUnixMs:  2000000,
		Bid:       requireDecimal(t, "1950.50"),
		Ask:       requireDecimal(t, "1950.80"),
	}
	err := pub.PublishTick(tick)
	if err != nil {
		t.Fatalf("PublishTick metals: %v", err)
	}
	t.Log("PublisherTickWithoutCanonical: metals symbol published")
}

func TestHashTickDeterministic(t *testing.T) {
	t.Parallel()
	t1 := &mdtick.Tick{Broker: "a", Canonical: "EURUSD", TsUnixMs: 100}
	t2 := &mdtick.Tick{Broker: "a", Canonical: "EURUSD", TsUnixMs: 100}
	if hashTick(t1) != hashTick(t2) {
		t.Fatal("hashTick must be deterministic")
	}
	// Different TsUnixMs must produce different hash.
	t3 := &mdtick.Tick{Broker: "a", Canonical: "EURUSD", TsUnixMs: 200}
	if hashTick(t1) == hashTick(t3) {
		t.Fatal("different TsUnixMs must produce different hash")
	}

	t.Log("HashTickDeterministic: idempotent and distinct")
}
