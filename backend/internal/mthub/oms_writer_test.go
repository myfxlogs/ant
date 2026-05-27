package mthub

import (
	"testing"
)

func TestIsValidOMSTransition_HappyPath(t *testing.T) {
	// Full happy path: NEW → VALIDATED → RISK_APPROVED → SUBMITTED.
	if !isValidOMSTransition(OMSStateNew, OMSStateValidated) {
		t.Error("NEW → VALIDATED should be valid")
	}
	if !isValidOMSTransition(OMSStateValidated, OMSStateRiskApproved) {
		t.Error("VALIDATED → RISK_APPROVED should be valid")
	}
	if !isValidOMSTransition(OMSStateRiskApproved, OMSStateSubmitted) {
		t.Error("RISK_APPROVED → SUBMITTED should be valid")
	}
	// Broker execution path: SUBMITTED → WORKING → FILLED.
	if !isValidOMSTransition(OMSStateSubmitted, OMSStateWorking) {
		t.Error("SUBMITTED → WORKING should be valid")
	}
	if !isValidOMSTransition(OMSStateWorking, OMSStateFilled) {
		t.Error("WORKING → FILLED should be valid")
	}
	if !isValidOMSTransition(OMSStateWorking, OMSStatePartiallyFilled) {
		t.Error("WORKING → PARTIALLY_FILLED should be valid")
	}
	if !isValidOMSTransition(OMSStatePartiallyFilled, OMSStateFilled) {
		t.Error("PARTIALLY_FILLED → FILLED should be valid")
	}
}

func TestIsValidOMSTransition_RejectPaths(t *testing.T) {
	// Rejection paths.
	if !isValidOMSTransition(OMSStateValidated, OMSStateRejected) {
		t.Error("VALIDATED → REJECTED should be valid")
	}
	if !isValidOMSTransition(OMSStateRiskApproved, OMSStateRejected) {
		t.Error("RISK_APPROVED → REJECTED should be valid")
	}
	if !isValidOMSTransition(OMSStateSubmitted, OMSStateFailed) {
		t.Error("SUBMITTED → FAILED should be valid")
	}
}

func TestIsValidOMSTransition_Invalid(t *testing.T) {
	invalidTransitions := []struct{ from, to OMSState }{
		{OMSStateNew, OMSStateSubmitted},         // skip VALIDATED
		{OMSStateNew, OMSStateFilled},            // skip everything
		{OMSStateValidated, OMSStateSubmitted},   // skip RISK_APPROVED
		{OMSStateFilled, OMSStateWorking},        // terminal state
		{OMSStateCancelled, OMSStateWorking},     // terminal state
		{OMSStateSubmitted, OMSStateValidated},   // backward
		{OMSStateWorking, OMSStateNew},           // backward
	}
	for _, tc := range invalidTransitions {
		if isValidOMSTransition(tc.from, tc.to) {
			t.Errorf("%s → %s should be invalid", tc.from, tc.to)
		}
	}
}

func TestOrderLifecycleFullPath(t *testing.T) {
	// Verify the full path: NEW → VALIDATED → RISK_APPROVED → SUBMITTED.
	states := []OMSState{
		OMSStateNew,
		OMSStateValidated,
		OMSStateRiskApproved,
		OMSStateSubmitted,
	}
	for i := 0; i < len(states)-1; i++ {
		if !isValidOMSTransition(states[i], states[i+1]) {
			t.Errorf("transition %s → %s should be valid", states[i], states[i+1])
		}
	}
	t.Log("Full lifecycle path: NEW → VALIDATED → RISK_APPROVED → SUBMITTED — valid")
}
