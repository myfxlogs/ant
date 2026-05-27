package factor

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

func TestSubscriber_Push(t *testing.T) {
	t.Parallel()
	cfg := DefaultSubscriberConfig()
	cfg.BufferSize = 2
	// Disable finality gate for this test.
	cfg.FinalityDelay = 0
	s := NewSubscriber(cfg, zap.NewNop())

	// Fill buffer.
	for i := 0; i < 2; i++ {
		if !s.Push(&mdtick.Bar{Canonical: "EURUSD"}) {
			t.Fatalf("Push #%d: expected true", i+1)
		}
	}

	// Channel full — should drop.
	if s.Push(&mdtick.Bar{Canonical: "EURUSD"}) {
		t.Fatal("Push on full channel: expected false (backpressure drop)")
	}

	if ChanFullTotal() < 1 {
		t.Fatalf("ChanFullTotal: want >=1, got %d", ChanFullTotal())
	}
}

func TestSubscriber_Chan(t *testing.T) {
	t.Parallel()
	cfg := DefaultSubscriberConfig()
	cfg.FinalityDelay = 0
	s := NewSubscriber(cfg, zap.NewNop())

	bar := &mdtick.Bar{Canonical: "GBPUSD"}
	s.Push(bar)

	select {
	case received := <-s.Chan():
		if received.Canonical != "GBPUSD" {
			t.Fatalf("received wrong bar: %s", received.Canonical)
		}
	default:
		t.Fatal("expected to receive bar from channel")
	}
}

func TestSubscriber_StartStop(t *testing.T) {
	t.Parallel()
	cfg := DefaultSubscriberConfig()
	s := NewSubscriber(cfg, zap.NewNop())

	ctx := t.Context()
	s.Start(ctx)
	s.Stop()
}

func TestSubscriber_BarFinalityGate_SkipRecent(t *testing.T) {
	t.Parallel()
	cfg := DefaultSubscriberConfig()
	cfg.BufferSize = 10
	cfg.FinalityDelay = 10 * time.Second // bars must be at least 10s old
	s := NewSubscriber(cfg, zap.NewNop())

	// Bar closed 1 second ago — too recent, should be skipped.
	recentBar := &mdtick.Bar{
		Canonical:     "EURUSD",
		CloseTsUnixMs: time.Now().Add(-1 * time.Second).UnixMilli(),
		IsReplay:      false,
	}
	if s.Push(recentBar) {
		t.Fatal("recent bar should be skipped by finality gate")
	}
	if BarFinalitySkipTotal() < 1 {
		t.Fatalf("BarFinalitySkipTotal: want >=1, got %d", BarFinalitySkipTotal())
	}
}

func TestSubscriber_BarFinalityGate_OldPasses(t *testing.T) {
	t.Parallel()
	cfg := DefaultSubscriberConfig()
	cfg.BufferSize = 10
	cfg.FinalityDelay = 10 * time.Second
	s := NewSubscriber(cfg, zap.NewNop())

	// Bar closed 20s ago — old enough, should pass.
	oldBar := &mdtick.Bar{
		Canonical:     "EURUSD",
		CloseTsUnixMs: time.Now().Add(-20 * time.Second).UnixMilli(),
		IsReplay:      false,
	}
	if !s.Push(oldBar) {
		t.Fatal("old bar should pass finality gate")
	}
	select {
	case received := <-s.Chan():
		if received.Canonical != "EURUSD" {
			t.Fatalf("wrong bar: %s", received.Canonical)
		}
	default:
		t.Fatal("expected to receive old bar")
	}
}

func TestSubscriber_BarFinalityGate_ReplayAlwaysPasses(t *testing.T) {
	t.Parallel()
	cfg := DefaultSubscriberConfig()
	cfg.BufferSize = 10
	cfg.FinalityDelay = 10 * time.Second
	s := NewSubscriber(cfg, zap.NewNop())

	// Replay bar closed 1ms ago — should always pass finality gate.
	replayBar := &mdtick.Bar{
		Canonical:     "EURUSD",
		CloseTsUnixMs: time.Now().UnixMilli(),
		IsReplay:      true,
	}
	if !s.Push(replayBar) {
		t.Fatal("replay bar should always pass finality gate")
	}
}

func TestSubscriber_BarFinalityGate_Disabled(t *testing.T) {
	t.Parallel()
	cfg := DefaultSubscriberConfig()
	cfg.BufferSize = 10
	cfg.FinalityDelay = 0 // disabled
	s := NewSubscriber(cfg, zap.NewNop())

	recentBar := &mdtick.Bar{
		Canonical:     "EURUSD",
		CloseTsUnixMs: time.Now().UnixMilli(),
		IsReplay:      false,
	}
	if !s.Push(recentBar) {
		t.Fatal("with zero finality delay, all bars should pass")
	}
}
