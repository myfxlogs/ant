package interceptor

// Tests for StreamLimitInterceptor (G-POST2-1/2). All blocking is
// channel-synchronized — no sleeps (SCHEDULE-HOTLOOP-1a precedent).
//
// Over-cap probes resolve deterministically via a select on two events:
// the probe is either rejected (err arrives — acquire held the cap while
// the holder slots were still occupied) or it entered the handler (a
// mutated build with the acquire check bypassed — entry signal fires).
// This removes any dependence on goroutine scheduling order.
//
// T3 mutation (reviewer re-verifies independently): deleting the acquire
// check in WrapStreamingHandler turns both tests RED (probe enters handler
// over a full cap).

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	connectrpc "connectrpc.com/connect"
)

// blockingHandler returns a StreamingHandlerFunc that signals it holds a
// slot on `acquired`, then parks until `release` is closed.
func blockingHandler(acquired chan struct{}, release chan struct{}) connectrpc.StreamingHandlerFunc {
	return func(_ context.Context, _ connectrpc.StreamingHandlerConn) error {
		acquired <- struct{}{}
		<-release
		return nil
	}
}

// entrySignalingHandler is the over-cap probe body: it announces entry (so a
// mutated build that admits it is detectable, not just silently wrong), then
// parks until release. Buffer 1 is safe — each probe owns its channel.
func entrySignalingHandler(entered chan struct{}, release chan struct{}) connectrpc.StreamingHandlerFunc {
	return func(_ context.Context, _ connectrpc.StreamingHandlerConn) error {
		entered <- struct{}{}
		<-release
		return nil
	}
}

// expectRejected fires probe on ctx and deterministically asserts it was
// rejected with CodeResourceExhausted: the probe either reports its error
// (real build) or signals handler entry (mutated build admitting over cap).
// releaseFn is invoked on both paths before returning.
func expectRejected(t *testing.T, label string, probe connectrpc.StreamingHandlerFunc, ctx context.Context, entered chan struct{}, releaseFn func()) {
	t.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- probe(ctx, nil) }()
	select {
	case <-entered:
		releaseFn()
		t.Fatalf("%s: admitted over cap (acquire check ineffective)", label)
	case err := <-errCh:
		releaseFn()
		if connectrpc.CodeOf(err) != connectrpc.CodeResourceExhausted {
			t.Fatalf("%s: expected CodeResourceExhausted, got %v", label, err)
		}
	}
}

// gaugeValue reads the current value of an active-streams gauge label.
func gaugeValue(t *testing.T, keyType string) float64 {
	t.Helper()
	return testutil.ToFloat64(streamActiveGauge.WithLabelValues(keyType))
}

// rejectionValue reads the current value of a rejections counter label.
func rejectionValue(t *testing.T, keyType string) float64 {
	t.Helper()
	return testutil.ToFloat64(streamRejectionsTotal.WithLabelValues(keyType))
}

// TestStreamLimit_CapBlocksThirdConcurrent (T1): cap=2 — two blocked
// handlers hold both slots, the third concurrent call is rejected with
// CodeResourceExhausted; after release, new calls succeed again.
func TestStreamLimit_CapBlocksThirdConcurrent(t *testing.T) {
	lim := NewStreamLimitInterceptor(2, 2)
	acquired := make(chan struct{}, 2)
	release := make(chan struct{})
	handler := lim.WrapStreamingHandler(blockingHandler(acquired, release))
	ctx := context.Background()

	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { done <- handler(ctx, nil) }()
	}
	<-acquired
	<-acquired // both slots held deterministically

	// Third concurrent call: acquire check must reject before the handler runs.
	probeEntered := make(chan struct{}, 1)
	probe := lim.WrapStreamingHandler(entrySignalingHandler(probeEntered, release))
	expectRejected(t, "third concurrent call", probe, ctx, probeEntered, func() { close(release) })

	if e := <-done; e != nil {
		t.Fatalf("first handler: %v", e)
	}
	if e := <-done; e != nil {
		t.Fatalf("second handler: %v", e)
	}

	// Slot freed — a new call must succeed.
	if err := lim.WrapStreamingHandler(instantOK)(ctx, nil); err != nil {
		t.Fatalf("post-release call: expected success, got %v", err)
	}
}

// TestStreamLimit_KeyClasses (T2): user keys cap at userCap, IP keys at
// ipCap, and the keyless "anon" bucket is shared but isolated from the
// others. Metrics are asserted per key_type via testutil.
func TestStreamLimit_KeyClasses(t *testing.T) {
	lim := NewStreamLimitInterceptor(1, 2) // user=1, ip=2, anon→ip cap
	ctx := context.Background()

	rejUser0 := rejectionValue(t, "user")
	rejIP0 := rejectionValue(t, "ip")
	rejAnon0 := rejectionValue(t, "anon")

	// ── user bucket: cap 1, per-user isolation ──
	acq := make(chan struct{}, 1)
	rel := make(chan struct{})
	userHandler := lim.WrapStreamingHandler(blockingHandler(acq, rel))
	uctx := context.WithValue(ctx, UserIDKey, "user-a")

	done := make(chan error, 1)
	go func() { done <- userHandler(uctx, nil) }()
	<-acq
	if got := gaugeValue(t, "user"); got < 1 {
		t.Fatalf("active user gauge: expected ≥1 while slot held, got %v", got)
	}

	// Same user over cap → rejected, counter bumped.
	probeEntered := make(chan struct{}, 1)
	probe := lim.WrapStreamingHandler(entrySignalingHandler(probeEntered, rel))
	expectRejected(t, "same user over cap", probe, uctx, probeEntered, func() { close(rel) })
	if e := <-done; e != nil {
		t.Fatalf("user-a holder: %v", e)
	}
	if got := rejectionValue(t, "user") - rejUser0; got != 1 {
		t.Fatalf("user rejections: expected +1, got %v", got)
	}
	// Different user unaffected by user-a's held slot (per-user, not global).
	if err := lim.WrapStreamingHandler(instantOK)(context.WithValue(ctx, UserIDKey, "user-b"), nil); err != nil {
		t.Fatalf("different user: expected success, got %v", err)
	}

	// ── ip bucket: cap 2, independent of user bucket ──
	ipAcq := make(chan struct{}, 2)
	ipRel := make(chan struct{})
	ipHandler := lim.WrapStreamingHandler(blockingHandler(ipAcq, ipRel))
	ipCtx := context.WithValue(ctx, ClientIPKey, "10.0.0.1")

	holders := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { holders <- ipHandler(ipCtx, nil) }()
		<-ipAcq
	}
	// Full ip bucket must not block a user-keyed call (buckets isolated).
	if err := lim.WrapStreamingHandler(instantOK)(uctx, nil); err != nil {
		t.Fatalf("user call while ip bucket full: expected success, got %v", err)
	}
	ipEntered := make(chan struct{}, 1)
	ipProbe := lim.WrapStreamingHandler(entrySignalingHandler(ipEntered, ipRel))
	expectRejected(t, "same ip over cap", ipProbe, ipCtx, ipEntered, func() { close(ipRel) })
	<-holders
	<-holders
	if got := rejectionValue(t, "ip") - rejIP0; got != 1 {
		t.Fatalf("ip rejections: expected +1, got %v", got)
	}

	// ── anon bucket: keyless requests share one key at ipCap ──
	anonAcq := make(chan struct{}, 2)
	anonRel := make(chan struct{})
	anonHandler := lim.WrapStreamingHandler(blockingHandler(anonAcq, anonRel))

	anonHolders := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { anonHolders <- anonHandler(ctx, nil) }()
		<-anonAcq
	}
	anonEntered := make(chan struct{}, 1)
	anonProbe := lim.WrapStreamingHandler(entrySignalingHandler(anonEntered, anonRel))
	expectRejected(t, "anon over cap", anonProbe, ctx, anonEntered, func() { close(anonRel) })
	<-anonHolders
	<-anonHolders
	if got := rejectionValue(t, "anon") - rejAnon0; got != 1 {
		t.Fatalf("anon rejections: expected +1, got %v", got)
	}
}

// instantOK is a handler that opens and closes a stream successfully.
func instantOK(_ context.Context, _ connectrpc.StreamingHandlerConn) error { return nil }
