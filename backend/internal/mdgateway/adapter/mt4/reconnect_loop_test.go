package mt4

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// TestEnsureConnected_ReturnsNilOnConnectError verifies QUOTE-RECONNECT-LOOP S1/T2:
// When Connect fails, ensureConnected must return nil (not an error) so that
// the caller loop continues to the next iteration instead of exiting permanently.
//
// Adversarial proof (P1): revert ensureConnected to return the Connect error
// → this test goes RED (ensureConnected returns non-nil error).
func TestEnsureConnected_ReturnsNilOnConnectError(t *testing.T) {
	t.Parallel()

	gw := New(mdtick.AccountConfig{
		AccountID:  "test-acc",
		MtapiHost:  "localhost:1", // unreachable — Connect will fail fast
		BrokerHost: "localhost:1",
		Login:      "1",
		Password:   "x",
	}, zap.NewNop())

	// Use a short-timeout context so grpc.DialContext fails quickly.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	backoff := 50 * time.Millisecond
	maxBackoff := 200 * time.Millisecond
	err := gw.ensureConnected(ctx, &backoff, maxBackoff)
	if err != nil {
		t.Fatalf("ensureConnected should return nil on Connect failure (S1), got error: %v — "+
			"RED: ensureConnected still returns error, loop will exit permanently", err)
	}
}

// TestRecvLoop_RetriesAfterConnectFailure verifies QUOTE-RECONNECT-LOOP S3/T1:
// When Connect fails, recvLoop must NOT exit — it should keep retrying until
// ctx is cancelled. We verify by checking that the loop goroutine is still
// alive after a Connect failure.
//
// Adversarial proof (P1): revert ensureConnected to return error + revert
// recvLoop to `if err != nil { return }` → the loop exits and the goroutine
// finishes before ctx cancel → this test goes RED.
func TestRecvLoop_RetriesAfterConnectFailure(t *testing.T) {
	t.Parallel()

	gw := New(mdtick.AccountConfig{
		AccountID:  "test-acc",
		MtapiHost:  "localhost:1", // unreachable — Connect will fail fast
		BrokerHost: "localhost:1",
		Login:      "1",
		Password:   "x",
	}, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var loopDone atomic.Bool
	go func() {
		gw.recvLoop(ctx, func(t *mdtick.Tick) {})
		loopDone.Store(true)
	}()

	// Wait long enough for at least one Connect attempt to fail.
	// With localhost:1, DialContext with a 500ms-ish timeout will fail,
	// but ensureConnected's sleep(backoff) keeps the loop alive.
	time.Sleep(2 * time.Second)

	if loopDone.Load() {
		t.Fatal("recvLoop exited before ctx cancel — RED: ensureConnected error caused loop to exit permanently")
	}
}

// TestDisconnect_DoesNotBlockOnSleep verifies QUOTE-RECONNECT-LOOP S2/T3:
// Disconnect on a cancelled context must return in <50ms (not sleep 200ms).
//
// Adversarial proof (P2): revert Disconnect to time.Sleep(200ms) →
// this test goes RED (>50ms).
func TestDisconnect_DoesNotBlockOnSleep(t *testing.T) {
	t.Parallel()

	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	// Set up a fake conn so Disconnect has something to close.
	gw.conn = &grpc.ClientConn{}

	// Cancel the context before calling Disconnect.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_ = gw.Disconnect(ctx)
	elapsed := time.Since(start)

	if elapsed > 50*time.Millisecond {
		t.Fatalf("Disconnect took %v on cancelled ctx, want <50ms — "+
			"RED: time.Sleep(200ms) not replaced with ctx-cancellable wait", elapsed)
	}
}

// TestProfitLoop_RetriesAfterConnectFailure verifies QUOTE-RECONNECT-LOOP S4/T4:
// profitRecvLoop must NOT exit on Connect failure — it retries until ctx cancel.
func TestProfitLoop_RetriesAfterConnectFailure(t *testing.T) {
	t.Parallel()

	gw := New(mdtick.AccountConfig{
		AccountID:  "test-acc",
		MtapiHost:  "localhost:1",
		BrokerHost: "localhost:1",
		Login:      "1",
		Password:   "x",
	}, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var loopDone atomic.Bool
	go func() {
		gw.profitRecvLoop(ctx, func(p *mdtick.ProfitUpdate) {})
		loopDone.Store(true)
	}()

	time.Sleep(2 * time.Second)

	if loopDone.Load() {
		t.Fatal("profitRecvLoop exited before ctx cancel — RED: ensureConnected error caused loop to exit permanently")
	}
}

// TestOrderLoop_RetriesAfterConnectFailure verifies QUOTE-RECONNECT-LOOP S4/T5:
// orderUpdateRecvLoop must NOT exit on Connect failure — it retries until ctx cancel.
func TestOrderLoop_RetriesAfterConnectFailure(t *testing.T) {
	t.Parallel()

	gw := New(mdtick.AccountConfig{
		AccountID:  "test-acc",
		MtapiHost:  "localhost:1",
		BrokerHost: "localhost:1",
		Login:      "1",
		Password:   "x",
	}, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var loopDone atomic.Bool
	go func() {
		gw.orderUpdateRecvLoop(ctx, func(u *mdtick.OrderUpdate) {})
		loopDone.Store(true)
	}()

	time.Sleep(2 * time.Second)

	if loopDone.Load() {
		t.Fatal("orderUpdateRecvLoop exited before ctx cancel — RED: ensureConnected error caused loop to exit permanently")
	}
}

// TestRecvLoop_ExitsOnCtxCancel verifies that recvLoop does exit when ctx
// is cancelled — the retry behavior must not prevent clean shutdown.
func TestRecvLoop_ExitsOnCtxCancel(t *testing.T) {
	t.Parallel()

	gw := New(mdtick.AccountConfig{
		AccountID:  "test-acc",
		MtapiHost:  "localhost:1",
		BrokerHost: "localhost:1",
		Login:      "1",
		Password:   "x",
	}, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())

	var loopDone atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		gw.recvLoop(ctx, func(t *mdtick.Tick) {})
		loopDone.Store(true)
	}()

	// Let the loop run briefly, then cancel.
	time.Sleep(500 * time.Millisecond)
	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// GREEN: loop exited after ctx cancel
	case <-time.After(5 * time.Second):
		t.Fatal("recvLoop did not exit after ctx cancel — RED: loop ignores ctx.Done()")
	}
}
