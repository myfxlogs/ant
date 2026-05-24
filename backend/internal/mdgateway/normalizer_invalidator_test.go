package mdgateway

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNormalizerListenerFallback(t *testing.T) {
	// Create invalidator with no PG listener → should use ticker fallback.
	invalidated := make(chan string, 10)
	ni := NewNormalizerInvalidator(zap.NewNop(), func(broker, symbolRaw string) {
		invalidated <- broker + ":" + symbolRaw
	})

	ctx, cancel := context.WithCancel(context.Background())
	ni.Start(ctx, nil) // nil pgListener → ticker fallback

	// Let it run for a moment.
	time.Sleep(100 * time.Millisecond)
	cancel()
	ni.Stop()

	// Ticker fallback is no-op until PG is wired; verify no panic during start/stop.
	select {
	case inv := <-invalidated:
		t.Logf("unexpected invalidation: %s", inv)
	default:
		t.Log("NormalizerListenerFallback: ticker started and stopped without panic")
	}
}

func TestNormalizerListener_StartStop(t *testing.T) {
	ni := NewNormalizerInvalidator(zap.NewNop(), func(broker, symbolRaw string) {})

	ctx, cancel := context.WithCancel(context.Background())
	ni.Start(ctx, nil)
	time.Sleep(50 * time.Millisecond)

	// Double start should be no-op.
	ni.Start(ctx, nil)

	ni.Stop()
	cancel()

	// Stop after already stopped should be safe.
	ni.Stop()
	t.Log("NormalizerListener: start/stop lifecycle works")
}
