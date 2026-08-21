// mutation_outcome.go — Typed mutation error classification for broker
// order operations (LIVE-ORDER-REENTRY-1 B4).
//
// The execution barrier must distinguish "order definitely did not reach
// the broker" (deterministic rejection → safe to release and retry) from
// "broker acceptance state is unknown" (outcome unknown → barrier stays
// locked, fail-closed). String-matching on error messages is forbidden —
// only typed sentinels and MutationError phase tags are authoritative.

package mthub

import (
	"errors"
	"fmt"
)

// MutationPhase identifies where in the mutation lifecycle an error occurred.
type MutationPhase int

const (
	// PhasePreBroker means the error occurred before any broker contact.
	// The order provably never reached the broker — deterministic rejection.
	PhasePreBroker MutationPhase = iota
	// PhaseBroker means the error occurred during or after broker contact.
	// The broker's acceptance state is unknown — outcome unknown (fail-closed).
	PhaseBroker
)

// MutationError wraps an error with its mutation phase. Returned by
// PlaceOrder/CloseOrder/ModifyOrder/DeleteOrder at phase boundaries so
// the execution barrier can classify the outcome without string guessing.
type MutationError struct {
	Phase MutationPhase
	Cause error
}

func (e *MutationError) Error() string {
	if e == nil || e.Cause == nil {
		return "mutation: <nil>"
	}
	return e.Cause.Error()
}

func (e *MutationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// IsPreBroker returns true if the error is a pre-broker rejection.
func (e *MutationError) IsPreBroker() bool {
	return e != nil && e.Phase == PhasePreBroker
}

// IsBroker returns true if the error occurred during/after broker contact.
func (e *MutationError) IsBroker() bool {
	return e != nil && e.Phase == PhaseBroker
}

// preBrokerError wraps err as a pre-broker phase mutation error. R6: does NOT
// double-wrap — if err is already a MutationError, returns it as-is.
func preBrokerError(err error) error {
	if err == nil {
		return nil
	}
	var me *MutationError
	if errors.As(err, &me) {
		return err // already typed — preserve original phase
	}
	return &MutationError{Phase: PhasePreBroker, Cause: err}
}

// brokerError wraps err as a broker-phase mutation error. R6: does NOT
// double-wrap — if err is already a MutationError, returns it as-is to
// preserve the original phase classification.
func brokerError(err error) error {
	if err == nil {
		return nil
	}
	var me *MutationError
	if errors.As(err, &me) {
		return err // already typed — preserve original phase
	}
	return &MutationError{Phase: PhaseBroker, Cause: err}
}

// ErrGateRejected is a sentinel for gate evaluation rejections (both place
// and close gates). Wrapped with fmt.Errorf("%w: %s", ErrGateRejected, reason)
// so errors.Is can detect it without string matching.
var ErrGateRejected = errors.New("mthub: gate rejected order")

// ErrBrokerRejected is a sentinel for broker application-level rejections.
// When the broker (MT4/MT5 via mtapi) returns a non-zero error code in its
// response, the order provably reached the broker and was definitively
// rejected — this is a deterministic rejection, not an unknown outcome.
// The order did NOT execute and no ticket was assigned.
// Wrapped with fmt.Errorf("%w: code=%d msg=%s", ErrBrokerRejected, ...) by
// the adapters so ClassifyMutationError can detect it without string matching.
var ErrBrokerRejected = errors.New("mthub: broker rejected order")

// ClassifyMutationError determines whether a mutation error is a deterministic
// pre-broker rejection (safe to release barrier) or a broker-phase unknown
// outcome (barrier must stay locked). Returns:
//
//	"deterministic_rejected" — order provably never reached broker, or broker
//	                           definitively rejected it at the application level
//	"outcome_unknown"         — broker acceptance state unknown (fail-closed)
//
// Non-MutationError errors default to "outcome_unknown" (fail-closed) unless
// they are known pre-broker sentinels.
func ClassifyMutationError(err error) string {
	if err == nil {
		return "confirmed"
	}
	// Check deterministic sentinels FIRST — they are authoritative regardless
	// of phase wrapping. A brokerError-wrapped ErrBrokerRejected is still a
	// deterministic rejection: the broker saw the request and said no.
	// Without this ordering, MutationError{PhaseBroker} would short-circuit
	// to "outcome_unknown" and lock the barrier forever on broker rejections
	// like MT4 code=130 (Invalid S/L or T/P), preventing strategy retry.
	if errors.Is(err, ErrBrokerRejected) ||
		errors.Is(err, ErrKillSwitchEngaged) ||
		errors.Is(err, ErrRateLimited) ||
		errors.Is(err, ErrDuplicateOrder) ||
		errors.Is(err, ErrReconciling) ||
		errors.Is(err, ErrAccountNotOwned) ||
		errors.Is(err, ErrCircuitOpen) ||
		errors.Is(err, ErrSessionNotFound) ||
		errors.Is(err, ErrGateRejected) {
		return "deterministic_rejected"
	}
	// If the error is wrapped with phase info, use it directly.
	var me *MutationError
	if errors.As(err, &me) {
		if me.IsPreBroker() {
			return "deterministic_rejected"
		}
		return "outcome_unknown"
	}
	// Default: unknown — fail-closed. Never assume an unclassified error
	// means "order didn't reach broker" (B4: no string guessing).
	return "outcome_unknown"
}

// wrapGateError wraps a gate rejection reason as a typed ErrGateRejected.
func wrapGateError(reason string) error {
	return fmt.Errorf("%w: %s", ErrGateRejected, reason)
}
