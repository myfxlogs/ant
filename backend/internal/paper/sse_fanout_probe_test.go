package paper

// POST-2 capacity probe (S2): SSE fan-out cost curve.
//
// Measures per-client cost of the production fan-out chain — PaperEngine
// broadcast → per-client chan(8) → stream handler goroutine → HTTP write →
// client read — at N ∈ {10,100,500} concurrent streams:
//
//	goroutines/stream, heap bytes/stream, delivery latency p50/p95/p99.
//
// 口径 (honest scope):
//   - Handler mirrors the WatchPaperAccount select loop
//     (internal/connect/paper/handler.go) over the real PaperEngine
//     Subscribe/broadcast primitives; payload is a minimal SSE line, so
//     ConnectRPC framing + proto marshal cost is NOT included.
//   - Per-stream PG LISTEN / NATS subscription adders are NOT included
//     (probe runs DB-free; registered in the staging list of
//     docs/benchmarks/post2-capacity-baseline-2026-09.md).
//   - Production wraps SSE with SSEKeepaliveMiddleware (+1 goroutine per
//     active stream, sse_keepalive.go) — adder noted in the doc, not measured.
//   - Structural assertions only; latency numbers are measurements, not gates.
//
// Numbers land in docs/benchmarks/post2-capacity-baseline-2026-09.md.

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// percentile returns the p-th percentile of ds (p ∈ (0,1]).
func percentile(ds []time.Duration, p float64) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	s := append([]time.Duration(nil), ds...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[int(float64(len(s)-1)*p)]
}

// sampleGoroutinesAndHeap returns (goroutines, heapAlloc bytes) after a GC
// quiesce, to make per-stream deltas comparable across N levels.
func sampleGoroutinesAndHeap() (int, uint64) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return runtime.NumGoroutine(), ms.HeapAlloc
}

// sseProbeStreamHandler mirrors the WatchPaperAccount loop shape: subscribe a
// chan, select { ctx.Done | ch }, write one SSE line per event, flush. The
// event line embeds its publish timestamp (unix ns) so the client side can
// compute delivery latency.
func sseProbeStreamHandler(engine *PaperEngine, accountID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		ch, unsubscribe := engine.Subscribe(accountID)
		defer unsubscribe()

		for {
			select {
			case <-r.Context().Done():
				return
			case acct, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "data: %s %d\n\n", acct.ID, time.Now().UnixNano())
				flusher.Flush()
			}
		}
	}
}

// readSSEStream consumes one stream body, converting "data: <id> <sentNs>"
// lines into delivery latencies, and reports them through res.
func readSSEStream(res *latencySamples, body io.Reader) {
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 0, 64*1024), 64*1024)
	var lat []time.Duration
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "data: "))
		if len(fields) != 2 {
			continue
		}
		sentNs, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}
		lat = append(lat, time.Duration(time.Now().UnixNano()-sentNs))
	}
	res.add(lat)
}

// latencySamples accumulates delivery latencies from reader goroutines.
type latencySamples struct {
	mu  sync.Mutex
	all []time.Duration
}

func (s *latencySamples) add(ds []time.Duration) {
	s.mu.Lock()
	s.all = append(s.all, ds...)
	s.mu.Unlock()
}

func (s *latencySamples) drain() []time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.all
	s.all = nil
	return out
}

// TestSSEFanoutCostCurve is the POST-2 S2 probe. See file doc for scope.
func TestSSEFanoutCostCurve(t *testing.T) {
	if testing.Short() {
		t.Skip("POST-2 probe: skipped in -short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	repo := newStubPaperRepo()
	repo.seedAccount("pa-fanout", 100000)
	engine := New(repo, nil, testLogger())

	srv := httptest.NewServer(http.HandlerFunc(sseProbeStreamHandler(engine, "pa-fanout")))
	defer srv.Close()

	// Generous idle pool so each in-flight request owns a connection
	// (one stream = one conn, matching 1 tab = 1 stream).
	client := &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 600}}

	baseGoroutines, baseHeap := sampleGoroutinesAndHeap()
	const events = 200
	samples := &latencySamples{}

	for _, n := range []int{10, 100, 500} {
		// Connect n concurrent streams.
		conns := make([]*http.Response, 0, n)
		var connMu sync.Mutex
		for i := 0; i < n; i++ {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/stream", nil)
			if err != nil {
				t.Fatalf("N=%d: request %d: %v", n, i, err)
			}
			req.Header.Set("Accept", "text/event-stream")
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("N=%d: connect %d: %v", n, i, err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("N=%d: stream %d: status %d", n, i, resp.StatusCode)
			}
			conns = append(conns, resp)
		}
		// One reader goroutine per stream.
		for _, resp := range conns {
			go readSSEStream(samples, resp.Body)
		}

		got, heap := sampleGoroutinesAndHeap()
		perConnGoroutines := float64(got-baseGoroutines) / float64(n)
		perConnHeap := float64(heap-baseHeap) / float64(n)

		// Publish `events` broadcasts paced at 5ms — near the real paper-fill
		// rhythm — so we measure delivery latency, not chan(8) overflow drops.
		// (At unpaced burst rate broadcast's drop-slow-consumer semantics drop
		// most events per conn: ~9-10 of 200 arrive = chan depth + in-flight.
		// Recorded as a doc note; not re-measured per level.)
		var publishWall time.Duration
		for e := 0; e < events; e++ {
			start := time.Now()
			engine.broadcast(ctx, "pa-fanout")
			publishWall += time.Since(start)
			time.Sleep(5 * time.Millisecond)
		}
		// Let readers drain, then tear the streams down before measuring.
		time.Sleep(300 * time.Millisecond)
		connMu.Lock()
		for _, resp := range conns {
			_ = resp.Body.Close() //nolint:errcheck // probe teardown
		}
		connMu.Unlock()
		time.Sleep(200 * time.Millisecond)

		lat := samples.drain()
		expected := n * events
		if len(lat) == 0 {
			t.Fatalf("N=%d: no deliveries observed", n)
		}
		t.Logf("POST-2 SSE fanout N=%d: goroutines/conn=%.1f heap/conn=%.0fB "+
			"deliveries=%d/%d (%.1f%%) latency p50=%s p95=%s p99=%s max=%s publish-avg=%s",
			n, perConnGoroutines, perConnHeap, len(lat), expected,
			100*float64(len(lat))/float64(expected),
			percentile(lat, 0.50), percentile(lat, 0.95), percentile(lat, 0.99),
			percentile(lat, 1.0), publishWall/time.Duration(events))
	}
}
