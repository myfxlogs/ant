package mt4

import (
	"sync"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// RECONNECT-RACE: the quote, profit and order-update loops all call
// ensureConnected on the same Gateway. Before the single-flight guard they
// raced Connect() after a shared-connection teardown, each creating a separate
// mtapi session; the losers kept a stale/empty sessionID and every
// SubscribeMany was rejected with "Client with id = ... not found", silently
// starving live strategies of prices.
//
// Adversarial proof: delete `g.connectMu.Lock()` from beginConnect (or the
// `if !g.beginConnect()` call in ensureConnected) → concurrent callers all
// enter the connect section at once → this test goes RED.
func TestRECONNECT_RACE_BeginConnect_IsSingleFlight(t *testing.T) {
	t.Parallel()

	gw := New(mdtick.AccountConfig{AccountID: "acc-1"}, zap.NewNop())

	const goroutines = 8
	var (
		mu        sync.Mutex
		inside    int
		maxInside int
	)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if !gw.beginConnect() {
				return
			}
			defer gw.connectMu.Unlock()

			mu.Lock()
			inside++
			if inside > maxInside {
				maxInside = inside
			}
			mu.Unlock()

			// Hold the slot long enough that any missing mutual exclusion
			// is observed deterministically rather than by luck.
			time.Sleep(10 * time.Millisecond)

			mu.Lock()
			inside--
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()

	if maxInside != 1 {
		t.Fatalf("RECONNECT-RACE: %d goroutines were inside the connect section concurrently, want 1 — "+
			"RED: single-flight guard missing, concurrent Connect() calls will create duplicate mtapi "+
			"sessions and leave losers with a stale sessionID", maxInside)
	}
}

// beginConnect must not hand out the slot at all once another goroutine has
// already restored the connection — otherwise a redundant Connect() would
// replace a healthy session and invalidate every subscribed symbol.
func TestRECONNECT_RACE_BeginConnect_SkipsWhenAlreadyConnected(t *testing.T) {
	t.Parallel()

	gw := New(mdtick.AccountConfig{AccountID: "acc-1"}, zap.NewNop())
	gw.conn = &grpc.ClientConn{}

	if gw.beginConnect() {
		gw.connectMu.Unlock()
		t.Fatal("RECONNECT-RACE: beginConnect granted the slot while conn was already established — " +
			"RED: a redundant Connect() would discard the live session and drop all symbol subscriptions")
	}

	// The slot must be released, not leaked, on the skip path.
	done := make(chan struct{})
	go func() {
		gw.connectMu.Lock()
		gw.connectMu.Unlock()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RECONNECT-RACE: connectMu still held after beginConnect returned false — " +
			"RED: lock leaked on the skip path, all future reconnects would deadlock")
	}
}
