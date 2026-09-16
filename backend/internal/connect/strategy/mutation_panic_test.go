package strategy

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
)

// ============================================================================
// QS-2.5: panic recovery — coordinateMutation + dispatchLiveSignal.
// An uncaught panic on the VM event-loop goroutine crashes the whole process.
// Recovery must converge fail-closed: outcomeUnknown + circuit open when the
// broker RPC may already be in flight, deterministicRejected + release when
// it provably was not, and must never lock an idle barrier.
// ============================================================================

func testPanicConf() confirmationConfig {
	return confirmationConfig{
		pushWait:              100 * time.Millisecond,
		readAfterWriteTimeout: 2 * time.Second,
		mutationRPCTimeout:    5 * time.Second,
		recoveryDelay:         50 * time.Millisecond,
	}
}

// S3a: panic inside brokerCall (post-RPC boundary) → outcomeUnknown lock.
func TestQS25_CoordinateMutationPanic_BrokerCall_OutcomeUnknown(t *testing.T) {
	srv, _, _ := testCoordinatorSetup(&prodMockExecutor{})
	cfg := testLiveCfg()
	sess := testActiveSess()
	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}

	res := srv.coordinateMutation(context.Background(), cfg, sess, mutationSpec{
		action:        actionOpen,
		clientID:      "panic-open-1",
		expectedMagic: strategyMagic(cfg.ScheduleID),
		brokerCall: func(ctx context.Context) (int64, error) {
			panic("broker boom")
		},
	}, "buy", sig, testPanicConf())

	if res.state != barrierOutcomeUnknown {
		t.Fatalf("result state=%s, want outcome_unknown", res.state)
	}
	if st := sess.barrier.State(); st != barrierOutcomeUnknown {
		t.Fatalf("barrier state=%s, want outcome_unknown (fail-closed lock)", st)
	}
	if !sess.IsCircuitOpen() {
		t.Fatal("circuit must be open after post-broker panic")
	}
	if !strings.Contains(sess.LastError, "panic") {
		t.Fatalf("LastError=%q, want panic record", sess.LastError)
	}
}

// S3b: panic before the broker RPC (nil mtHub → SubscribePositionSnapshots
// nil-receiver panic at mutation_coordinator.go:139) → deterministicRejected
// + Release. The broker was never called, so the outcome is knowable and the
// barrier must go back to idle — no circuit trip.
func TestQS25_CoordinateMutationPanic_PreBroker_DeterministicRejected(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()} // mtHub nil → panic at SubscribePositionSnapshots
	cfg := testLiveCfg()
	sess := testActiveSess()
	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}

	res := srv.coordinateMutation(context.Background(), cfg, sess, mutationSpec{
		action:   actionOpen,
		clientID: "panic-pre-1",
		brokerCall: func(ctx context.Context) (int64, error) {
			t.Fatal("brokerCall must not be reached — panic is pre-RPC")
			return 0, nil
		},
	}, "buy", sig, testPanicConf())

	if res.state != barrierDeterministicRejected {
		t.Fatalf("result state=%s, want deterministic_rejected", res.state)
	}
	if st := sess.barrier.State(); st != barrierIdle {
		t.Fatalf("barrier state=%s, want idle (Release must run)", st)
	}
	if sess.IsCircuitOpen() {
		t.Fatal("circuit must stay closed for a pre-broker panic")
	}
	if !strings.Contains(sess.LastError, "panic") {
		t.Fatalf("LastError=%q, want panic record", sess.LastError)
	}
}

// S3c: Acquire-rejection path — coverage equivalent of the !acquired recover
// branch. A panic with acquired==false can only originate in the nil-barrier
// check or inside Acquire itself; neither is injectable without nil'ing
// s.log, which the recover's own logging depends on. This asserts the same
// observable outcome the !acquired branch produces: {state: barrierIdle},
// no circuit trip, no barrier state change.
func TestQS25_CoordinateMutation_AcquireRejected_Idle(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()}
	cfg := testLiveCfg()
	sess := testActiveSess()
	// Pre-lock the barrier as if a previous mutation is still in flight.
	sess.barrier.Acquire("in-flight-other", 0, "open")
	defer sess.barrier.Release()
	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}

	res := srv.coordinateMutation(context.Background(), cfg, sess, mutationSpec{
		action:   actionOpen,
		clientID: "rejected-1",
		brokerCall: func(ctx context.Context) (int64, error) {
			t.Fatal("brokerCall must not be reached — Acquire rejected")
			return 0, nil
		},
	}, "buy", sig, testPanicConf())

	if res.state != barrierIdle {
		t.Fatalf("result state=%s, want idle", res.state)
	}
	if sess.IsCircuitOpen() {
		t.Fatal("circuit must stay closed on acquire rejection")
	}
	if st := sess.barrier.State(); st != barrierSubmitting {
		t.Fatalf("barrier state=%s, want submitting (still held by the other op)", st)
	}
}

// S3d: dispatchLiveSignal panic with an IDLE barrier → circuit opens, error
// recorded, but the barrier must NOT be locked (an idle barrier converged to
// outcomeUnknown would permanently reject all subsequent mutations — a
// fail-open regression in reverse).
// Injection: runRepo = zero-value repository with nil pgxpool →
// persistSignal → StrategyRunRepository.InsertSignal → r.db.Exec nil-deref
// panic inside the dispatchLiveSignal frame (live_helpers.go:94).
func TestQS25_DispatchLiveSignalPanic_IdleBarrierNotLocked(t *testing.T) {
	srv, _, _ := testCoordinatorSetup(&prodMockExecutor{})
	srv.runRepo = &repository.StrategyRunRepository{} // nil db pool → panic in persistSignal
	cfg := testLiveCfg()
	sess := testActiveSess()
	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}

	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess) // must not crash

	if st := sess.barrier.State(); st != barrierIdle {
		t.Fatalf("barrier state=%s, want idle — idle barrier must never be locked", st)
	}
	if !sess.IsCircuitOpen() {
		t.Fatal("circuit must be open after dispatch panic")
	}
	if !strings.Contains(sess.LastError, "panic") {
		t.Fatalf("LastError=%q, want panic record", sess.LastError)
	}
}

// S3e: dispatchLiveSignal panic with an IN-FLIGHT barrier (submitting) →
// converged to outcomeUnknown (conservative: an in-flight mutation whose
// supervising frame died has an unknowable outcome).
func TestQS25_DispatchLiveSignalPanic_InFlightBarrier_OutcomeUnknown(t *testing.T) {
	srv, _, _ := testCoordinatorSetup(&prodMockExecutor{})
	srv.runRepo = &repository.StrategyRunRepository{} // nil db pool → panic in persistSignal
	cfg := testLiveCfg()
	sess := testActiveSess()
	sess.barrier.Acquire("inflight-1", 0, "open") // pre-set in-flight
	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}

	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess) // must not crash

	if st := sess.barrier.State(); st != barrierOutcomeUnknown {
		t.Fatalf("barrier state=%s, want outcome_unknown (in-flight → fail-closed lock)", st)
	}
	if !sess.IsCircuitOpen() {
		t.Fatal("circuit must be open after dispatch panic")
	}
}
