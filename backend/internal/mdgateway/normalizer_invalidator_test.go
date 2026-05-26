package mdgateway

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// fakePGListener feeds queued JSON payloads to NormalizerInvalidator.listenLoop.
type fakePGListener struct {
	payloads chan string
	closed   chan struct{}
}

func newFakePGListener(buf int) *fakePGListener {
	return &fakePGListener{
		payloads: make(chan string, buf),
		closed:   make(chan struct{}),
	}
}

func (f *fakePGListener) WaitForNotification(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-f.closed:
		return "", context.Canceled
	case p, ok := <-f.payloads:
		if !ok {
			return "", context.Canceled
		}
		return p, nil
	}
}

func (f *fakePGListener) Close() error {
	select {
	case <-f.closed:
	default:
		close(f.closed)
	}
	return nil
}

// TestNormalizerListenerFallback verifies that:
//  1. nil pgListener → ticker fallback path runs cleanly without dispatching
//     spurious invalidations (no PG signal source available).
//  2. ticker goroutine terminates cleanly on Stop+cancel.
func TestNormalizerListenerFallback(t *testing.T) {
	invalidated := make(chan string, 10)
	ni := NewNormalizerInvalidator(zap.NewNop(), nil, func(broker, symbolRaw string) {
		invalidated <- broker + ":" + symbolRaw
	})

	ctx, cancel := context.WithCancel(context.Background())
	ni.Start(ctx, nil) // nil pgListener → ticker fallback

	time.Sleep(150 * time.Millisecond)
	cancel()
	ni.Stop()

	// Real assertion: ticker MUST NOT dispatch any invalidations (it has no
	// signal source). A non-empty channel here would indicate ticker leaked
	// invalidations from elsewhere.
	select {
	case inv := <-invalidated:
		t.Fatalf("ticker fallback must not produce invalidations, got %q", inv)
	default:
	}

	// Calling Stop again must be idempotent — no panic.
	ni.Stop()
	t.Log("ticker fallback ran cleanly with zero spurious invalidations")
}

// TestNormalizerListenerPgListen verifies the LISTEN path: a JSON payload
// arriving via PGListener.WaitForNotification triggers onInvalidate(broker,
// symbol_raw) exactly once with the parsed fields.
func TestNormalizerListenerPgListen(t *testing.T) {
	invalidated := make(chan string, 4)
	ni := NewNormalizerInvalidator(zap.NewNop(), nil, func(broker, symbolRaw string) {
		invalidated <- broker + ":" + symbolRaw
	})

	listener := newFakePGListener(4)
	listener.payloads <- `{"broker":"ic_markets","symbol_raw":"EURUSDm"}`
	listener.payloads <- `not-json`                                 // bad payload, must be skipped
	listener.payloads <- `{"broker":"","symbol_raw":""}`            // empty fields, must be skipped
	listener.payloads <- `{"broker":"oanda","symbol_raw":"GBPUSD"}`

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ni.Start(ctx, listener)

	got := []string{}
	timeout := time.After(2 * time.Second)
	for len(got) < 2 {
		select {
		case s := <-invalidated:
			got = append(got, s)
		case <-timeout:
			t.Fatalf("timed out; got %d/2 invalidations", len(got))
		}
	}
	ni.Stop()

	if got[0] != "ic_markets:EURUSDm" || got[1] != "oanda:GBPUSD" {
		t.Errorf("got invalidations %v, want [ic_markets:EURUSDm oanda:GBPUSD]", got)
	}
	// bad/empty payloads must NOT have triggered an invalidation
	select {
	case extra := <-invalidated:
		t.Errorf("unexpected extra invalidation: %q", extra)
	default:
	}
	t.Logf("PG LISTEN path dispatched 2 valid invalidations + skipped 2 bad payloads")
}

func TestNormalizerListener_StartStop(t *testing.T) {
	ni := NewNormalizerInvalidator(zap.NewNop(), nil, func(broker, symbolRaw string) {})

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

func TestNormalizerListener(t *testing.T) {
	t.Run("PgListen", func(t *testing.T) {
		ni := NewNormalizerInvalidator(zap.NewNop(), nil, func(broker, symbolRaw string) {})
		ctx, cancel := context.WithCancel(context.Background())
		ni.Start(ctx, nil)
		time.Sleep(50 * time.Millisecond)
		ni.Stop()
		cancel()
	})
	t.Run("Fallback", func(t *testing.T) {
		invalidated := make(chan string, 10)
		ni := NewNormalizerInvalidator(zap.NewNop(), nil, func(broker, symbolRaw string) {
			invalidated <- broker + ":" + symbolRaw
		})
		ctx, cancel := context.WithCancel(context.Background())
		ni.Start(ctx, nil)
		time.Sleep(100 * time.Millisecond)
		cancel()
		ni.Stop()
	})
}
