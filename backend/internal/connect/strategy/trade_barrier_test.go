// trade_barrier_test.go — Unit tests for the TradeBarrier state machine (B3).
// Production-wiring tests are in mutation_coordinator_test.go (B8).

package strategy

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

// I1: 100 concurrent goroutines, only 1 acquires the barrier.
func TestLIVE_ORDER_REENTRY_1_I1_ConcurrentAcquireOnlyOneSucceeds(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	var acquired atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if b.Acquire("client-1", 12345, "open") {
				acquired.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := acquired.Load(); got != 1 {
		t.Fatalf("I1: %d goroutines acquired barrier, want 1", got)
	}
}

// T6: NotifyOutcomeUnknown locks the barrier — subsequent Acquire fails.
func TestLIVE_ORDER_REENTRY_1_T6_OutcomeUnknownStaysLocked(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	if !b.Acquire("client-1", 12345, "open") {
		t.Fatal("T6: first Acquire failed")
	}
	b.NotifyOutcomeUnknown()
	if state := b.State(); state != barrierOutcomeUnknown {
		t.Fatalf("T6: state=%s, want outcome_unknown", state)
	}
	// Subsequent Acquire must fail (barrier locked).
	if b.Acquire("client-2", 12345, "open") {
		t.Fatal("T6: second Acquire succeeded — barrier should be locked")
	}
}

// Barrier state machine: pre-response event caching (B3).
func TestLIVE_ORDER_REENTRY_1_Barrier_PreResponseEventCached(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	// Event arrives before broker response.
	b.NotifyConfirmationEvent(42, 12345, "open")
	// State should still be submitting (event cached, not confirmed).
	if state := b.State(); state != barrierSubmitting {
		t.Fatalf("pre-response: state=%s, want submitting", state)
	}
	// Broker response arrives with matching ticket.
	b.NotifyBrokerAccepted(42)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("post-response: state=%s, want confirmed (cached event matched)", state)
	}
}

// Barrier: post-response event confirms (B1: listener alive after RPC).
func TestLIVE_ORDER_REENTRY_1_Barrier_PostResponseEventConfirms(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	if state := b.State(); state != barrierAcceptedUnconfirmed {
		t.Fatalf("post-response: state=%s, want accepted_unconfirmed", state)
	}
	// Event arrives after broker response — must confirm.
	b.NotifyConfirmationEvent(42, 12345, "open")
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("post-response event: state=%s, want confirmed", state)
	}
}

// Barrier: unrelated ticket does NOT confirm.
func TestLIVE_ORDER_REENTRY_1_Barrier_UnrelatedTicketNotConfirmed(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(99, 12345, "open") // wrong ticket
	if state := b.State(); state != barrierAcceptedUnconfirmed {
		t.Fatalf("unrelated ticket: state=%s, want accepted_unconfirmed (not confirmed)", state)
	}
}

// Barrier: matching ticket wrong magic does NOT confirm.
func TestLIVE_ORDER_REENTRY_1_Barrier_WrongMagicNotConfirmed(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 999, "open") // wrong magic
	if state := b.State(); state != barrierAcceptedUnconfirmed {
		t.Fatalf("wrong magic: state=%s, want accepted_unconfirmed (not confirmed)", state)
	}
}

// Barrier: deterministic rejection releases.
func TestLIVE_ORDER_REENTRY_1_Barrier_DeterministicRejectedReleases(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyDeterministicRejected()
	if state := b.State(); state != barrierDeterministicRejected {
		t.Fatalf("rejected: state=%s, want deterministic_rejected", state)
	}
	b.Release()
	if state := b.State(); state != barrierIdle {
		t.Fatalf("after Release: state=%s, want idle", state)
	}
	// Can acquire again after release.
	if !b.Acquire("client-2", 12345, "open") {
		t.Fatal("Acquire after Release failed")
	}
}

// Barrier: WaitConfirmed blocks until terminal state.
func TestLIVE_ORDER_REENTRY_1_Barrier_WaitConfirmed(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	// Confirm in a goroutine after a short delay.
	go func() {
		b.NotifyBrokerAccepted(42)
		b.NotifyConfirmationEvent(42, 12345, "open")
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	state := b.WaitConfirmed(ctx)
	if state != barrierConfirmed {
		t.Fatalf("WaitConfirmed: state=%s, want confirmed", state)
	}
}

// Barrier: WaitConfirmed returns non-terminal on context cancellation.
func TestLIVE_ORDER_REENTRY_1_Barrier_WaitConfirmedCtxCancel(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	state := b.WaitConfirmed(ctx)
	if state.isTerminal() {
		t.Fatalf("WaitConfirmed with cancelled ctx: state=%s, want non-terminal", state)
	}
}

// ── R3-③: Pending action compatibility (labels match real adapter output) ──

// pending_open confirms an open mutation (MT4 PendingOpen / MT5 PendingOpen).
func TestLIVE_ORDER_REENTRY_1_R3_PendingOpenConfirmsOpen(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, "pending_open")
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("pending_open confirms open: state=%s, want confirmed", state)
	}
}

// pending_close confirms a close mutation (MT4/MT5 PendingClose).
func TestLIVE_ORDER_REENTRY_1_R3_PendingCloseConfirmsClose(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "close")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, "pending_close")
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("pending_close confirms close: state=%s, want confirmed", state)
	}
}

// pending_modify confirms a modify mutation (MT4 PendingModify → "pending_modify").
func TestLIVE_ORDER_REENTRY_1_R3_PendingModifyConfirmsModify(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "modify")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, "pending_modify")
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("pending_modify confirms modify: state=%s, want confirmed", state)
	}
}

// pending_close confirms a cancel mutation (deleting pending order → PendingClose).
func TestLIVE_ORDER_REENTRY_1_R3_PendingCloseConfirmsCancel(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "cancel")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, "pending_close")
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("pending_close confirms cancel: state=%s, want confirmed", state)
	}
}

// ── R3-④: fail-closed for unknown action, empty updateType, magic=0 ──

// Unknown action must fail-closed (not accept all updateTypes).
func TestLIVE_ORDER_REENTRY_1_R3_UnknownActionFailClosed(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "bogus_action")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, "open")
	if state := b.State(); state != barrierAcceptedUnconfirmed {
		t.Fatalf("unknown action fail-closed: state=%s, want accepted_unconfirmed (not confirmed)", state)
	}
}

// Empty updateType must fail-closed (not accept).
func TestLIVE_ORDER_REENTRY_1_R3_EmptyUpdateTypeFailClosed(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, "")
	if state := b.State(); state != barrierAcceptedUnconfirmed {
		t.Fatalf("empty updateType fail-closed: state=%s, want accepted_unconfirmed (not confirmed)", state)
	}
}

// magic=0 event must NOT confirm when expected magic is non-zero (R3-④).
func TestLIVE_ORDER_REENTRY_1_R3_MagicZeroRejectedWhenExpectedNonZero(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	// Event with magic=0 — must NOT confirm even with matching ticket+updateType.
	b.NotifyConfirmationEvent(42, 0, "open")
	if state := b.State(); state != barrierAcceptedUnconfirmed {
		t.Fatalf("magic=0 rejected: state=%s, want accepted_unconfirmed (not confirmed)", state)
	}
}

// magic=0 event cached pre-response must NOT confirm on NotifyBrokerAccepted
// when expected magic is non-zero (R3-④: no magic=0 fallback).
func TestLIVE_ORDER_REENTRY_1_R3_PreResponseMagicZeroRejected(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	// Event arrives pre-response with magic=0.
	b.NotifyConfirmationEvent(42, 0, "open")
	if state := b.State(); state != barrierSubmitting {
		t.Fatalf("pre-response magic=0: state=%s, want submitting", state)
	}
	// Broker response arrives — magic=0 cached event must NOT confirm.
	b.NotifyBrokerAccepted(42)
	if state := b.State(); state != barrierAcceptedUnconfirmed {
		t.Fatalf("pre-response magic=0 rejected on accept: state=%s, want accepted_unconfirmed (not confirmed)", state)
	}
}

// magic=0 expected (legacy/no-magic strategy) accepts any magic event.
func TestLIVE_ORDER_REENTRY_1_R3_ZeroExpectedMagicAcceptsAny(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 0, "open")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 999, "open")
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("zero expected magic accepts any: state=%s, want confirmed", state)
	}
}
