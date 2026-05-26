// Package oms implements the Order Management System.
// State machine aligned with NautilusTrader OrderStatus enum (ADR-0012, M11-2).
//
// Order lifecycle (15 states):
//
//	NEW → VALIDATED → RISK_APPROVED → SUBMITTED
//	                                    ├── WORKING
//	                                    ├── PARTIALLY_FILLED → FILLED
//	                                    ├── FILLED
//	                                    ├── CANCELLED
//	                                    ├── EXPIRED
//	                                    ├── FAILED
//	                                    ├── UNKNOWN (timeout: 30s no response)
//	                                    ├── REQUOTED
//	                                    ├── SLIPPAGE_REJECTED
//	                                    └── MARGIN_CALL
//	VALIDATED → REJECTED
//	RISK_APPROVED → REJECTED
//	UNKNOWN → RECONCILING (reconcile-before-accept gate)
//	RECONCILING → WORKING | FILLED | CANCELLED | FAILED (resolved by reconciliation)
package oms

import (
	"fmt"
	"time"
)

// OrderState represents the current state of an order in the OMS state machine.
type OrderState string

const (
	StateNew              OrderState = "NEW"
	StateValidated        OrderState = "VALIDATED"
	StateRiskApproved     OrderState = "RISK_APPROVED"
	StateSubmitted        OrderState = "SUBMITTED"
	StateWorking          OrderState = "WORKING"
	StatePartiallyFilled  OrderState = "PARTIALLY_FILLED"
	StateFilled           OrderState = "FILLED"
	StateCancelled        OrderState = "CANCELLED"
	StateRejected         OrderState = "REJECTED"
	StateFailed           OrderState = "FAILED"
	StateExpired          OrderState = "EXPIRED"
	StateRequoted         OrderState = "REQUOTED"
	StateSlippageRejected OrderState = "SLIPPAGE_REJECTED"
	StateUnknown          OrderState = "UNKNOWN"
	StateReconciling      OrderState = "RECONCILING"
	StateMarginCall       OrderState = "MARGIN_CALL"
)

// TerminalStates are the end states — no further transitions allowed.
var TerminalStates = map[OrderState]bool{
	StateFilled:    true,
	StateCancelled: true,
	StateRejected:  true,
	StateFailed:    true,
	StateExpired:   true,
}

// TimeoutTransitionDuration is the max time a SUBMITTED order waits before
// transitioning to UNKNOWN when no broker response is received.
const TimeoutTransitionDuration = 30 * time.Second

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

// ShouldTimeout returns true if the order has been in SUBMITTED state
// longer than TimeoutTransitionDuration and should transition to UNKNOWN.
func ShouldTimeout(submittedAt time.Time) bool {
	return time.Since(submittedAt) > TimeoutTransitionDuration
}

// TimeoutTransition is a convenience that checks the timeout condition
// and returns the UNKNOWN state if the submitted_at timestamp exceeds the limit.
func TimeoutTransition(current OrderState, submittedAt time.Time) (OrderState, error) {
	if current != StateSubmitted {
		return current, fmt.Errorf("oms: timeout transition only valid from SUBMITTED, got %s", current)
	}
	if !ShouldTimeout(submittedAt) {
		return current, nil
	}
	return StateUnknown, nil
}

// isValid defines the allowed state transitions (15-state machine).
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
			StateUnknown,
			StateRequoted,
			StateSlippageRejected,
			StateMarginCall,
		},
		StateWorking: {
			StatePartiallyFilled,
			StateFilled,
			StateCancelled,
			StateExpired,
			StateFailed,
			StateRequoted,
		},
		StatePartiallyFilled: {
			StatePartiallyFilled,
			StateFilled,
			StateCancelled,
			StateExpired,
			StateFailed,
		},
		StateRequoted: {
			StateRiskApproved,
			StateCancelled,
			StateExpired,
		},
		StateSlippageRejected: {
			StateRiskApproved,
			StateCancelled,
			StateExpired,
		},
		StateUnknown: {
			StateReconciling,
			StateWorking,
			StateFilled,
			StateCancelled,
			StateFailed,
			StateExpired,
		},
		StateReconciling: {
			StateWorking,
			StatePartiallyFilled,
			StateFilled,
			StateCancelled,
			StateFailed,
			StateExpired,
		},
		StateMarginCall: {
			StateRiskApproved,
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
