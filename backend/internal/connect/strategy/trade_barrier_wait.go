// trade_barrier_wait.go — Wait/Reconcile/Release methods extracted from trade_barrier.go.
package strategy

import (
	"context"
	"time"
)

func (b *TradeBarrier) cacheEvent(key eventCacheKey, ticket int64, magic int32, updateType string) {
	if _, exists := b.eventCache[key]; !exists {
		b.eventCacheOrd = append(b.eventCacheOrd, key)
		// Evict oldest if over limit (R4: bounded).
		for len(b.eventCacheOrd) > maxEventCacheEntries {
			oldest := b.eventCacheOrd[0]
			b.eventCacheOrd = b.eventCacheOrd[1:]
			delete(b.eventCache, oldest)
		}
	}
	b.eventCache[key] = &cachedEvent{ticket: ticket, magic: magic, updateType: updateType}
}

// NotifyDeterministicRejected transitions submitting→deterministicRejected.
// Used when the order was rejected before reaching the broker (gate rejection,
// circuit open, etc.) — provably pre-broker.
func (b *TradeBarrier) NotifyDeterministicRejected() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state.isTerminal() {
		return
	}
	b.state = barrierDeterministicRejected
	b.cond.Broadcast()
}

// NotifyOutcomeUnknown transitions any non-terminal state→outcomeUnknown.
// The barrier stays locked — fail-closed. Does not override existing
// terminal states (confirmed/rejected stay as-is).
func (b *TradeBarrier) NotifyOutcomeUnknown() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state.isTerminal() && b.state != barrierOutcomeUnknown {
		return
	}
	if b.state == barrierOutcomeUnknown {
		return
	}
	b.state = barrierOutcomeUnknown
	b.cond.Broadcast()
}

// WaitConfirmed blocks until the barrier reaches a terminal state or ctx
// is cancelled. Returns the terminal (or current non-terminal) state.
func (b *TradeBarrier) WaitConfirmed(ctx context.Context) tradeBarrierState {
	b.mu.Lock()
	if b.state.isTerminal() {
		s := b.state
		b.mu.Unlock()
		return s
	}
	// Watch ctx cancellation to wake the cond.
	stopWatcher := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			b.mu.Lock()
			b.cond.Broadcast()
			b.mu.Unlock()
		case <-stopWatcher:
		}
	}()
	defer close(stopWatcher)
	for {
		if b.state.isTerminal() {
			s := b.state
			b.mu.Unlock()
			return s
		}
		if ctx.Err() != nil {
			s := b.state
			b.mu.Unlock()
			return s
		}
		b.cond.Wait()
	}
}

// Reconcile attempts to recover from outcomeUnknown state using an
// authoritative reconciliation result (e.g. a delayed OpenedOrders query).
// If the barrier is not in outcomeUnknown, this is a no-op.
// If confirmed=true → transitions to barrierConfirmed.
// If confirmed=false → transitions to barrierDeterministicRejected.
// Either way, the caller must subsequently call Release() to return to idle.
func (b *TradeBarrier) Reconcile(confirmed bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state != barrierOutcomeUnknown {
		return
	}
	if confirmed {
		b.state = barrierConfirmed
	} else {
		b.state = barrierDeterministicRejected
	}
	b.cond.Broadcast()
}

// Release transitions any state→idle. Called by the coordinator after
// WaitConfirmed returns a confirmed or deterministicRejected state.
// For outcomeUnknown, the caller must NOT call Release — barrier stays locked
// unless Reconcile has transitioned it to a recoverable terminal state.
func (b *TradeBarrier) Release() {
	b.mu.Lock()
	b.state = barrierIdle
	b.clientID = ""
	b.magic = 0
	b.action = ""
	b.expectedTicket = 0
	b.eventCache = make(map[eventCacheKey]*cachedEvent, 4)
	b.eventCacheOrd = b.eventCacheOrd[:0]
	b.cond.Broadcast()
	b.mu.Unlock()
}

// State returns the current state (for testing/diagnostics).
func (b *TradeBarrier) State() tradeBarrierState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

// Ticket returns the broker-accepted ticket (0 if not yet accepted).
func (b *TradeBarrier) Ticket() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.expectedTicket
}

// confirmationConfig holds tunable parameters for the confirmation wait.
// All fields are injectable for deterministic testing (no time.Sleep).
type confirmationConfig struct {
	// pushWait is the bounded duration to wait for a push confirmation
	// after broker acceptance before falling back to read-after-write.
	pushWait time.Duration
	// readAfterWriteTimeout is the context timeout for the single
	// OpenedOrders query.
	readAfterWriteTimeout time.Duration
	// mutationRPCTimeout is the deadline for the broker RPC call itself (B5).
	// If the broker doesn't respond within this duration, the outcome is
	// unknown and the barrier stays locked.
	mutationRPCTimeout time.Duration
	// recoveryDelay is the delay before attempting reconciliation-based
	// recovery from outcomeUnknown state (④-②). Only applies to mutations
	// with a known ticket (close/modify/cancel). Open mutations (ticket
	// unknown) stay fail-closed — no auto-recovery.
	recoveryDelay time.Duration
}

var defaultConfirmationConfig = confirmationConfig{
	pushWait:              5 * time.Second,
	readAfterWriteTimeout: 10 * time.Second,
	mutationRPCTimeout:    30 * time.Second,
	recoveryDelay:         10 * time.Second,
}
