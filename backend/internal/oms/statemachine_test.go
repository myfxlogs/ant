package oms

import "testing"

func TestStateTransitions(t *testing.T) {
	valid := []struct {
		current, next OrderState
	}{
		{StateNew, StateValidated},
		{StateValidated, StateRiskApproved},
		{StateValidated, StateRejected},
		{StateRiskApproved, StateSubmitted},
		{StateRiskApproved, StateRejected},
		{StateSubmitted, StateWorking},
		{StateSubmitted, StateFilled},
		{StateSubmitted, StateCancelled},
		{StateWorking, StateFilled},
		{StatePartiallyFilled, StatePartiallyFilled},
		{StatePartiallyFilled, StateFilled},
	}
	for _, tc := range valid {
		if err := Transition(tc.current, tc.next); err != nil {
			t.Errorf("valid transition %s → %s should pass: %v", tc.current, tc.next, err)
		}
	}
}

func TestInvalidTransitions(t *testing.T) {
	invalid := []struct {
		current, next OrderState
	}{
		{StateNew, StateSubmitted},       // skip validated + risk
		{StateNew, StateFilled},          // skip all
		{StateSubmitted, StateValidated}, // backward
		{StateFilled, StateCancelled},    // terminal → anything
		{StateRejected, StateNew},        // terminal → anything
		{StateCancelled, StateWorking},   // terminal → anything
	}
	for _, tc := range invalid {
		if err := Transition(tc.current, tc.next); err == nil {
			t.Errorf("invalid transition %s → %s should fail", tc.current, tc.next)
		}
	}
}

func TestIsTerminal(t *testing.T) {
	terminals := []OrderState{StateFilled, StateCancelled, StateRejected, StateFailed, StateExpired}
	for _, s := range terminals {
		if !IsTerminal(s) {
			t.Errorf("%s should be terminal", s)
		}
	}
	nonTerminals := []OrderState{StateNew, StateValidated, StateRiskApproved, StateSubmitted, StateWorking}
	for _, s := range nonTerminals {
		if IsTerminal(s) {
			t.Errorf("%s should NOT be terminal", s)
		}
	}
}

func TestTransitionRoundTrip(t *testing.T) {
	// Full happy path: NEW → VALIDATED → RISK_APPROVED → SUBMITTED → WORKING → FILLED
	path := []OrderState{StateNew, StateValidated, StateRiskApproved, StateSubmitted, StateWorking, StateFilled}
	for i := 1; i < len(path); i++ {
		if err := Transition(path[i-1], path[i]); err != nil {
			t.Errorf("happy path transition %s → %s failed: %v", path[i-1], path[i], err)
		}
	}
}
