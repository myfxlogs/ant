// Package oms implements the Order Management System.
// State machine per ADR-0012 (BrokerAdapter) and AlfQ migration §M2.
//
// Order lifecycle:
//
//	NEW → VALIDATED → RISK_APPROVED → SUBMITTED
//	                                    ├── WORKING
//	                                    ├── PARTIALLY_FILLED → FILLED
//	                                    ├── FILLED
//	                                    ├── CANCELLED
//	                                    ├── EXPIRED
//	                                    └── FAILED
//	VALIDATED → REJECTED
//	RISK_APPROVED → REJECTED
package oms

import "fmt"

// OrderState represents the current state of an order in the OMS state machine.
type OrderState string

const (
	StateNew             OrderState = "NEW"
	StateValidated       OrderState = "VALIDATED"
	StateRiskApproved    OrderState = "RISK_APPROVED"
	StateSubmitted       OrderState = "SUBMITTED"
	StateWorking         OrderState = "WORKING"
	StatePartiallyFilled OrderState = "PARTIALLY_FILLED"
	StateFilled          OrderState = "FILLED"
	StateCancelled       OrderState = "CANCELLED"
	StateRejected        OrderState = "REJECTED"
	StateFailed          OrderState = "FAILED"
	StateExpired         OrderState = "EXPIRED"
)

// TerminalStates are the end states — no further transitions allowed.
var TerminalStates = map[OrderState]bool{
	StateFilled:    true,
	StateCancelled: true,
	StateRejected:  true,
	StateFailed:    true,
	StateExpired:   true,
}

// Transition validates and returns an error if the state transition is invalid.
func Transition(current, next OrderState) error {
	if !isValid(current, next) {
		return fmt.Errorf("oms: invalid transition %s → %s", current, next)
	}
	return nil
}

// IsTerminal returns true if the state is a final state.
func IsTerminal(s OrderState) bool {
	return TerminalStates[s]
}

// CanTransition checks if a transition is allowed without returning an error.
func CanTransition(current, next OrderState) bool {
	return isValid(current, next)
}

// isValid defines the allowed state transitions.
func isValid(current, next OrderState) bool {
	transitions := map[OrderState][]OrderState{
		StateNew: {
			StateValidated,
		},
		StateValidated: {
			StateRiskApproved,
			StateRejected,
		},
		StateRiskApproved: {
			StateSubmitted,
			StateRejected,
		},
		StateSubmitted: {
			StateWorking,
			StatePartiallyFilled,
			StateFilled,
			StateCancelled,
			StateExpired,
			StateFailed,
		},
		StateWorking: {
			StatePartiallyFilled,
			StateFilled,
			StateCancelled,
			StateExpired,
			StateFailed,
		},
		StatePartiallyFilled: {
			StatePartiallyFilled,
			StateFilled,
			StateCancelled,
			StateExpired,
			StateFailed,
		},
	}

	allowed, ok := transitions[current]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}
