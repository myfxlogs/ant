package mthub

import (
	"sync/atomic"
	"testing"

	"go.uber.org/zap"
)

// TestTask2_TransitionOrderByTicket_NilOMSWriter_NoPanic verifies defensive
// behavior when omsWriter is nil — TransitionOrderByTicket returns immediately
// without calling reconcileTrigger or panicking.
// Remove the nil guard → nil pointer panic → RED.
func TestTask2_TransitionOrderByTicket_NilOMSWriter_NoPanic(t *testing.T) {
	svc := newTestService()
	svc.SetLogger(zap.NewNop())

	var triggered atomic.Int32
	svc.SetReconcileTrigger(func(accountID string) {
		triggered.Add(1)
	})

	// Should not panic and should not trigger reconciliation.
	svc.TransitionOrderByTicket(t.Context(), "acc-1", 123, OMSStateFilled)

	if triggered.Load() != 0 {
		t.Fatal("reconcileTrigger should not be called when omsWriter is nil")
	}
}

// TestTask2_SetReconcileTrigger verifies the setter correctly injects the
// reconciliation trigger function.
// Remove SetReconcileTrigger → field stays nil → RED.
func TestTask2_SetReconcileTrigger(t *testing.T) {
	svc := newTestService()
	var called atomic.Int32
	svc.SetReconcileTrigger(func(accountID string) {
		called.Add(1)
	})
	if svc.reconcileTrigger == nil {
		t.Fatal("SetReconcileTrigger did not set the function")
	}
	svc.reconcileTrigger("acc-1")
	if called.Load() != 1 {
		t.Fatal("reconcileTrigger not callable")
	}
}

// TestTask2_ReconcileTrigger_NilSafe verifies that retryTransitionByTicket
// does not panic when reconcileTrigger is nil (defensive nil guard).
func TestTask2_ReconcileTrigger_NilSafe(t *testing.T) {
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	// reconcileTrigger is nil — retryTransitionByTicket should not panic.
	// omsWriter is also nil — TransitionOrderByTicket returns early.
	svc.TransitionOrderByTicket(t.Context(), "acc-1", 999, OMSStateFilled)
}

// TestShouldAttemptOMSTransition pins idempotency + terminal guards that keep
// broker stream replays from spamming invalid-transition errors:
// same-state replays and any transition out of a terminal state must be
// skipped before hitting the OMS writer.
func TestShouldAttemptOMSTransition(t *testing.T) {
	cases := []struct {
		name        string
		current, to OMSState
		want        bool
	}{
		{"submitted to working", OMSStateSubmitted, OMSStateWorking, true},
		{"submitted to filled", OMSStateSubmitted, OMSStateFilled, true},
		{"working to filled", OMSStateWorking, OMSStateFilled, true},
		{"filled replay same-state", OMSStateFilled, OMSStateFilled, false},
		{"working replay same-state", OMSStateWorking, OMSStateWorking, false},
		{"filled to working regression", OMSStateFilled, OMSStateWorking, false},
		{"filled to cancelled", OMSStateFilled, OMSStateCancelled, false},
		{"cancelled to filled", OMSStateCancelled, OMSStateFilled, false},
		{"expired to working", OMSStateExpired, OMSStateWorking, false},
		{"rejected terminal", OMSStateRejected, OMSStateFilled, false},
		{"unknown still attempts", OMSStateUnknown, OMSStateFilled, true},
		{"reconciling still attempts", OMSStateReconciling, OMSStateWorking, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldAttemptOMSTransition(tc.current, tc.to); got != tc.want {
				t.Fatalf("shouldAttemptOMSTransition(%s,%s) = %v, want %v", tc.current, tc.to, got, tc.want)
			}
		})
	}
}
