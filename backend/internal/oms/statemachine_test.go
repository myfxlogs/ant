package oms

import (
	"testing"
	"time"
)

func TestOrderStateMachine_All15States(t *testing.T) {
	t.Parallel()
	all := []OrderState{
		StateNew, StateValidated, StateRiskApproved, StateSubmitted,
		StateWorking, StatePartiallyFilled, StateFilled,
		StateCancelled, StateRejected, StateFailed, StateExpired,
		StateRequoted, StateSlippageRejected, StateUnknown, StateReconciling, StateMarginCall,
	}
	if len(all) != 16 {
		t.Fatalf("expected 16 states (15 non-terminal + count), got %d", len(all))
	}

	seen := map[OrderState]bool{}
	for _, s := range all {
		if seen[s] {
			t.Fatalf("duplicate state: %s", s)
		}
		seen[s] = true
	}
	t.Logf("All 15 OMS states enumerated (PASS)")
}

func TestStateTransitions(t *testing.T) {
	t.Parallel()
	valid := []struct {
		current, next OrderState
	}{
		// Core path
		{StateNew, StateValidated},
		{StateValidated, StateRiskApproved},
		{StateValidated, StateRejected},
		{StateRiskApproved, StateSubmitted},
		{StateRiskApproved, StateRejected},
		{StateSubmitted, StateWorking},
		{StateSubmitted, StateFilled},
		{StateSubmitted, StateCancelled},
		{StateWorking, StateFilled},
		{StateWorking, StatePartiallyFilled},
		{StatePartiallyFilled, StatePartiallyFilled},
		{StatePartiallyFilled, StateFilled},

		// New states
		{StateSubmitted, StateUnknown},
		{StateSubmitted, StateRequoted},
		{StateSubmitted, StateSlippageRejected},
		{StateSubmitted, StateMarginCall},
		{StateRequoted, StateRiskApproved},
		{StateRequoted, StateCancelled},
		{StateSlippageRejected, StateRiskApproved},
		{StateSlippageRejected, StateCancelled},
		{StateUnknown, StateReconciling},
		{StateUnknown, StateWorking},
		{StateUnknown, StateFilled},
		{StateUnknown, StateCancelled},
		{StateReconciling, StateWorking},
		{StateReconciling, StateFilled},
		{StateReconciling, StateCancelled},
		{StateReconciling, StateFailed},
		{StateReconciling, StatePartiallyFilled},
		{StateMarginCall, StateRiskApproved},
		{StateMarginCall, StateCancelled},
		{StateMarginCall, StateFailed},
	}
	for _, tc := range valid {
		if err := Transition(tc.current, tc.next); err != nil {
			t.Errorf("valid transition %s → %s should pass: %v", tc.current, tc.next, err)
		}
	}
}

func TestInvalidTransitions(t *testing.T) {
	t.Parallel()
	invalid := []struct {
		current, next OrderState
	}{
		{StateNew, StateSubmitted},       // skip validated + risk
		{StateNew, StateFilled},          // skip all
		{StateSubmitted, StateValidated}, // backward
		{StateFilled, StateCancelled},    // terminal → anything
		{StateRejected, StateNew},        // terminal → anything
		{StateCancelled, StateWorking},   // terminal → anything
		{StateExpired, StateNew},         // terminal → anything
		{StateFailed, StateSubmitted},    // terminal → anything
		// New state invalid paths
		{StateRequoted, StateFilled},              // Requoted can't jump to Filled
		{StateSlippageRejected, StateWorking},     // must go through RiskApproved
		{StateUnknown, StateValidated},            // Unknown can't go backward
		{StateReconciling, StateNew},              // Reconciling can't restart
		{StateMarginCall, StateFilled},            // MarginCall can't jump to Filled
		{StateRequoted, StateSubmitted},           // must re-evaluate risk first
		{StateSlippageRejected, StateSubmitted},   // must re-evaluate risk first
		{StateUnknown, StateRiskApproved},         // Unknown can't go back to risk
		{StateReconciling, StateValidated},        // Reconciling can't go back
		{StateMarginCall, StateSubmitted},         // MarginCall must go through RiskApproved
	}
	for _, tc := range invalid {
		if err := Transition(tc.current, tc.next); err == nil {
			t.Errorf("invalid transition %s → %s should fail", tc.current, tc.next)
		}
	}
}

func TestIsTerminal(t *testing.T) {
	t.Parallel()
	terminals := []OrderState{StateFilled, StateCancelled, StateRejected, StateFailed, StateExpired}
	for _, s := range terminals {
		if !IsTerminal(s) {
			t.Errorf("%s should be terminal", s)
		}
	}
	nonTerminals := []OrderState{
		StateNew, StateValidated, StateRiskApproved, StateSubmitted,
		StateWorking, StatePartiallyFilled,
		StateRequoted, StateSlippageRejected, StateUnknown, StateReconciling, StateMarginCall,
	}
	for _, s := range nonTerminals {
		if IsTerminal(s) {
			t.Errorf("%s should NOT be terminal", s)
		}
	}
}

func TestTransitionRoundTrip(t *testing.T) {
	t.Parallel()
	// Full happy path: NEW → VALIDATED → RISK_APPROVED → SUBMITTED → WORKING → FILLED
	path := []OrderState{StateNew, StateValidated, StateRiskApproved, StateSubmitted, StateWorking, StateFilled}
	for i := 1; i < len(path); i++ {
		if err := Transition(path[i-1], path[i]); err != nil {
			t.Errorf("happy path transition %s → %s failed: %v", path[i-1], path[i], err)
		}
	}
}

func TestRequotedPath(t *testing.T) {
	t.Parallel()
	// Requoted → re-evaluate risk → re-submit
	path := []OrderState{StateRequoted, StateRiskApproved, StateSubmitted, StateWorking, StateFilled}
	for i := 1; i < len(path); i++ {
		if err := Transition(path[i-1], path[i]); err != nil {
			t.Errorf("requoted path %s → %s failed: %v", path[i-1], path[i], err)
		}
	}
}

func TestSlippageRejectedPath(t *testing.T) {
	t.Parallel()
	// SlippageRejected → re-evaluate risk → re-submit
	path := []OrderState{StateSlippageRejected, StateRiskApproved, StateSubmitted, StateWorking, StateFilled}
	for i := 1; i < len(path); i++ {
		if err := Transition(path[i-1], path[i]); err != nil {
			t.Errorf("slippage rejected path %s → %s failed: %v", path[i-1], path[i], err)
		}
	}
}

func TestUnknownToReconcilingPath(t *testing.T) {
	t.Parallel()
	// Unknown → Reconciling → resolved by reconciliation
	path := []OrderState{StateUnknown, StateReconciling, StateWorking, StateFilled}
	for i := 1; i < len(path); i++ {
		if err := Transition(path[i-1], path[i]); err != nil {
			t.Errorf("unknown→reconciling path %s → %s failed: %v", path[i-1], path[i], err)
		}
	}
}

func TestMarginCallPath(t *testing.T) {
	t.Parallel()
	// MarginCall → add margin → re-evaluate → re-submit
	path := []OrderState{StateMarginCall, StateRiskApproved, StateSubmitted, StateWorking, StateFilled}
	for i := 1; i < len(path); i++ {
		if err := Transition(path[i-1], path[i]); err != nil {
			t.Errorf("margin call path %s → %s failed: %v", path[i-1], path[i], err)
		}
	}
}

func TestSubmittedToUnknownPath(t *testing.T) {
	t.Parallel()
	// SUBMITTED → timeout → UNKNOWN → Reconciling → resolved
	path := []OrderState{StateSubmitted, StateUnknown, StateReconciling, StateFilled}
	for i := 1; i < len(path); i++ {
		if err := Transition(path[i-1], path[i]); err != nil {
			t.Errorf("submitted→unknown path %s → %s failed: %v", path[i-1], path[i], err)
		}
	}
}

func TestReconcilingAllResolutions(t *testing.T) {
	t.Parallel()
	// Reconciling can resolve to: Working, PartiallyFilled, Filled, Cancelled, Failed, Expired
	resolutions := []OrderState{StateWorking, StatePartiallyFilled, StateFilled, StateCancelled, StateFailed, StateExpired}
	for _, r := range resolutions {
		if err := Transition(StateReconciling, r); err != nil {
			t.Errorf("Reconciling → %s should be valid: %v", r, err)
		}
	}
}

func TestTimeoutTransition(t *testing.T) {
	t.Parallel()
	// Not timed out: should stay SUBMITTED.
	recent := time.Now()
	next, err := TimeoutTransition(StateSubmitted, recent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != StateSubmitted {
		t.Fatalf("recent submit should stay SUBMITTED, got %s", next)
	}

	// Timed out: should transition to UNKNOWN.
	old := time.Now().Add(-31 * time.Second)
	next, err = TimeoutTransition(StateSubmitted, old)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != StateUnknown {
		t.Fatalf("timed out submit should become UNKNOWN, got %s", next)
	}
}

func TestTimeoutTransition_OnlyFromSubmitted(t *testing.T) {
	t.Parallel()
	old := time.Now().Add(-31 * time.Second)
	_, err := TimeoutTransition(StateWorking, old)
	if err == nil {
		t.Fatal("TimeoutTransition from WORKING should error")
	}
}

func TestShouldTimeout(t *testing.T) {
	t.Parallel()
	if ShouldTimeout(time.Now()) {
		t.Fatal("ShouldTimeout should be false for recent time")
	}
	if !ShouldTimeout(time.Now().Add(-31 * time.Second)) {
		t.Fatal("ShouldTimeout should be true for >30s ago")
	}
}
