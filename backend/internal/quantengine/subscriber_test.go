package quantengine

import (
	"testing"

	"go.uber.org/zap"
)

func TestSignalSubscriber_Push(t *testing.T) {
	cfg := DefaultSubscriberConfig()
	cfg.BufferSize = 2
	s := NewSubscriber(cfg, zap.NewNop())

	// Fill buffer.
	for i := 0; i < 2; i++ {
		if !s.Push(&Signal{Symbol: "EURUSD"}) {
			t.Fatalf("Push #%d: expected true", i+1)
		}
	}

	// Channel full — should drop and record backpressure metric.
	if s.Push(&Signal{Symbol: "EURUSD"}) {
		t.Fatal("Push on full channel: expected false (backpressure drop)")
	}

	if SignalDroppedTotal() < 1 {
		t.Fatalf("SignalDroppedTotal: want >=1, got %d", SignalDroppedTotal())
	}
}

func TestSignalSubscriber_Chan(t *testing.T) {
	cfg := DefaultSubscriberConfig()
	s := NewSubscriber(cfg, zap.NewNop())

	sig := &Signal{Symbol: "GBPUSD", Side: "buy", Volume: 0.1}
	s.Push(sig)

	select {
	case received := <-s.Chan():
		if received.Symbol != "GBPUSD" {
			t.Fatalf("received wrong signal: %s", received.Symbol)
		}
	default:
		t.Fatal("expected to receive signal from channel")
	}
}

func TestSignalSubscriber_StartStop(t *testing.T) {
	cfg := DefaultSubscriberConfig()
	s := NewSubscriber(cfg, zap.NewNop())

	ctx := t.Context()
	s.Start(ctx)
	s.Stop()
}
