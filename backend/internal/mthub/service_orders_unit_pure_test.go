package mthub

import (
	"testing"
)

func TestHashToNegative_Deterministic(t *testing.T) {
	t.Parallel()
	a := hashToNegative("order-abc-123")
	b := hashToNegative("order-abc-123")
	if a != b {
		t.Fatalf("hashToNegative must be deterministic: %d vs %d", a, b)
	}
}

func TestHashToNegative_Different(t *testing.T) {
	t.Parallel()
	a := hashToNegative("order-1")
	b := hashToNegative("order-2")
	if a == b {
		t.Fatal("different IDs should produce different hashes")
	}
}

func TestHashToNegative_AlwaysNegative(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"", "a", "order-1", "very-long-order-id-12345"} {
		h := hashToNegative(id)
		if h >= 0 {
			t.Errorf("hashToNegative(%q) = %d, expected negative", id, h)
		}
	}
}

func TestAdvisoryLockKey_Deterministic(t *testing.T) {
	t.Parallel()
	k1, k2 := advisoryLockKey("acc-1", "client-1")
	k1b, k2b := advisoryLockKey("acc-1", "client-1")
	if k1 != k1b || k2 != k2b {
		t.Fatalf("advisoryLockKey must be deterministic: (%d,%d) vs (%d,%d)", k1, k2, k1b, k2b)
	}
}

func TestAdvisoryLockKey_DifferentInputs(t *testing.T) {
	t.Parallel()
	a1, a2 := advisoryLockKey("acc-1", "client-1")
	b1, b2 := advisoryLockKey("acc-1", "client-2")
	c1, c2 := advisoryLockKey("acc-2", "client-1")
	sameAB := a1 == b1 && a2 == b2
	sameAC := a1 == c1 && a2 == c2
	if sameAB || sameAC {
		t.Fatal("different inputs should produce different lock keys")
	}
}

// --- OMS state machine: full coverage of all valid transitions ---

func TestIsValidOMSTransition_AllValid(t *testing.T) {
	t.Parallel()
	valid := []struct{ from, to OMSState }{
		{OMSStateNew, OMSStateValidated},
		{OMSStateValidated, OMSStateRiskApproved},
		{OMSStateRiskApproved, OMSStateSubmitted},
		{OMSStateSubmitted, OMSStateWorking},
		{OMSStateSubmitted, OMSStatePartiallyFilled},
		{OMSStateSubmitted, OMSStateFilled},
		{OMSStateSubmitted, OMSStateCancelled},
		{OMSStateSubmitted, OMSStateExpired},
		{OMSStateSubmitted, OMSStateFailed},
		{OMSStateSubmitted, OMSStateUnknown},
		{OMSStateSubmitted, OMSStateRequoted},
		{OMSStateSubmitted, OMSStateSlippageRejected},
		{OMSStateSubmitted, OMSStateMarginCall},
		{OMSStateWorking, OMSStatePartiallyFilled},
		{OMSStateWorking, OMSStateFilled},
		{OMSStateWorking, OMSStateCancelled},
		{OMSStateWorking, OMSStateExpired},
		{OMSStateWorking, OMSStateFailed},
		{OMSStateWorking, OMSStateRequoted},
		{OMSStatePartiallyFilled, OMSStatePartiallyFilled},
		{OMSStatePartiallyFilled, OMSStateFilled},
		{OMSStatePartiallyFilled, OMSStateCancelled},
		{OMSStatePartiallyFilled, OMSStateExpired},
		{OMSStatePartiallyFilled, OMSStateFailed},
		{OMSStateValidated, OMSStateRejected},
		{OMSStateRiskApproved, OMSStateRejected},
		{OMSStateRiskApproved, OMSStateFailed},
		{OMSStateRequoted, OMSStateRiskApproved},
		{OMSStateRequoted, OMSStateCancelled},
		{OMSStateRequoted, OMSStateExpired},
		{OMSStateSlippageRejected, OMSStateRiskApproved},
		{OMSStateSlippageRejected, OMSStateCancelled},
		{OMSStateSlippageRejected, OMSStateExpired},
		{OMSStateUnknown, OMSStateReconciling},
		{OMSStateUnknown, OMSStateWorking},
		{OMSStateUnknown, OMSStateFilled},
		{OMSStateUnknown, OMSStateCancelled},
		{OMSStateUnknown, OMSStateFailed},
		{OMSStateUnknown, OMSStateExpired},
		{OMSStateReconciling, OMSStateWorking},
		{OMSStateReconciling, OMSStatePartiallyFilled},
		{OMSStateReconciling, OMSStateFilled},
		{OMSStateReconciling, OMSStateCancelled},
		{OMSStateReconciling, OMSStateFailed},
		{OMSStateReconciling, OMSStateExpired},
		{OMSStateMarginCall, OMSStateRiskApproved},
		{OMSStateMarginCall, OMSStateCancelled},
		{OMSStateMarginCall, OMSStateExpired},
		{OMSStateMarginCall, OMSStateFailed},
	}
	for _, tc := range valid {
		if !isValidOMSTransition(tc.from, tc.to) {
			t.Errorf("%s → %s should be valid", tc.from, tc.to)
		}
	}
}

func TestIsValidOMSTransition_TerminalStates_NoExit(t *testing.T) {
	t.Parallel()
	terminal := []OMSState{OMSStateFilled, OMSStateCancelled, OMSStateExpired, OMSStateRejected, OMSStateFailed}
	allStates := []OMSState{
		OMSStateNew, OMSStateValidated, OMSStateRiskApproved, OMSStateSubmitted,
		OMSStateWorking, OMSStatePartiallyFilled, OMSStateFilled,
		OMSStateCancelled, OMSStateRejected, OMSStateFailed, OMSStateExpired,
		OMSStateRequoted, OMSStateSlippageRejected, OMSStateUnknown,
		OMSStateReconciling, OMSStateMarginCall,
	}
	for _, term := range terminal {
		for _, next := range allStates {
			if isValidOMSTransition(term, next) {
				t.Errorf("terminal state %s → %s should be invalid", term, next)
			}
		}
	}
}

// --- Helpers ---

// mockKillSwitch implements KillSwitchGate for testing.
