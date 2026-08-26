// trade_barrier.go — Session-scoped execution barrier that serializes broker
// mutations per ActiveSession, restoring MT4 EA single-threaded OrderSend
// semantics (LIVE-ORDER-REENTRY-1).
//
// R3 rework: ALL state transitions, ticket/magic reservation, and the
// pre-response event cache are managed under a single mutex. No atomic +
// mutex mix — the entire barrier state is one linearizable critical section.
// The event cache is a bounded map (max 16 entries, evict oldest) keyed by
// (ticket, magic) so multiple unrelated events arriving before the broker
// response don't overwrite each other. Acquire clears the reservation +
// cache for the new mutation. R3: updateType action compatibility is
// validated — a matching ticket+magic event with an incompatible updateType
// (e.g. "modify" event confirming a "close" action) is NOT accepted.
package strategy

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// maxEventCacheEntries bounds the event cache to prevent unbounded growth
// from a flood of unrelated events (R4).
const maxEventCacheEntries = 16

// tradeBarrierState tracks the lifecycle of a single in-flight broker mutation.
type tradeBarrierState int32

const (
	barrierIdle tradeBarrierState = iota
	barrierSubmitting
	barrierAcceptedUnconfirmed
	barrierConfirmed
	barrierDeterministicRejected
	barrierOutcomeUnknown
)

func (s tradeBarrierState) String() string {
	switch s {
	case barrierIdle:
		return "idle"
	case barrierSubmitting:
		return "submitting"
	case barrierAcceptedUnconfirmed:
		return "accepted_unconfirmed"
	case barrierConfirmed:
		return "confirmed"
	case barrierDeterministicRejected:
		return "deterministic_rejected"
	case barrierOutcomeUnknown:
		return "outcome_unknown"
	default:
		return "unknown"
	}
}

func (s tradeBarrierState) isTerminal() bool {
	return s == barrierConfirmed || s == barrierDeterministicRejected || s == barrierOutcomeUnknown
}

// cachedEvent is a pre-response confirmation event stored in the barrier's
// bounded cache. Keyed by (ticket, magic) so multiple unrelated events
// with the same ticket but different magic don't collide (R4).
type cachedEvent struct {
	ticket     int64
	magic      int32
	updateType string
}

// eventCacheKey is the composite key for the event cache (R4: ticket+magic).
type eventCacheKey struct {
	ticket int64
	magic  int32
}

// actionCompatibleUpdateTypes maps each mutation action to the set of
// OnOrderUpdate updateType values that are compatible with confirming it.
// R3: a matching ticket+magic event with an incompatible updateType must
// NOT confirm the mutation (e.g. a "modify" event cannot confirm a "close").
//
// R3-③ (third-round rework): labels MUST match the real adapter output.
// MT4 mt4UpdateActionLabel produces: open, close, modify, pending_open,
// pending_close, pending_modify, balance, credit, unknown.
// MT5 mt5UpdateTypeLabel produces: open, close, pending_open,
// pending_close, modify, balance, unknown.
// Previous table had phantom labels (add/pending_add/buy/sell/delete/
// pending_delete) that no adapter ever emits — pending-order confirmations
// never matched and always fell back to the 5s pushWait + read-after-write.
var actionCompatibleUpdateTypes = map[string]map[string]bool{
	string(actionOpen): {
		"open": true, "pending_open": true, // PendingFill → "open" (MT4)
	},
	string(actionClose): {
		"close": true, "pending_close": true, // PartialClose → "close" (MT5)
	},
	string(actionModify): {
		"modify": true, "pending_modify": true, // MT5 PendingModify → "modify"
	},
	"cancel": {
		// Deleting a pending order emits PendingClose from both adapters.
		"close": true, "pending_close": true,
	},
}

// isUpdateTypeCompatible checks whether the given updateType is compatible
// with the specified action. R3-④ (third-round rework): fail-closed for
// unknown actions and empty updateType — fund-boundary precise confirmation
// requires an explicit, compatible label. Brokers that don't fill updateType
// fall back to the bounded read-after-write (authoritative anyway).
func isUpdateTypeCompatible(action, updateType string) bool {
	compatible, ok := actionCompatibleUpdateTypes[action]
	if !ok {
		return false // unknown action — fail-closed
	}
	if updateType == "" {
		return false // empty updateType — fail-closed
	}
	return compatible[updateType]
}

// TradeBarrier serializes broker mutations for one ActiveSession.
// All state is managed under a single mutex (R3: no atomic + mutex mix).
type TradeBarrier struct {
	mu   sync.Mutex
	cond *sync.Cond

	state tradeBarrierState

	// Reservation for the current mutation.
	clientID       string
	magic          int32
	action         string // R3: action for updateType compatibility checking
	expectedTicket int64  // 0 until NotifyBrokerAccepted sets it

	// Bounded event cache: events that arrived before the broker response.
	// Cleared on Acquire. Keyed by (ticket, magic) so unrelated events
	// coexist (R4). Bounded to maxEventCacheEntries — oldest evicted.
	eventCache    map[eventCacheKey]*cachedEvent
	eventCacheOrd []eventCacheKey // insertion order for eviction

	log *zap.Logger
}

// NewTradeBarrier creates a barrier in the idle state.
func NewTradeBarrier(log *zap.Logger) *TradeBarrier {
	if log == nil {
		log = zap.NewNop()
	}
	b := &TradeBarrier{
		log:        log,
		eventCache: make(map[eventCacheKey]*cachedEvent, 4),
	}
	b.cond = sync.NewCond(&b.mu)
	return b
}

// Acquire atomically transitions idle→submitting under the mutex. Returns
// false if the barrier is not idle. Clears the reservation + event cache
// for the new mutation. R3: stores the action for updateType compatibility.
func (b *TradeBarrier) Acquire(clientID string, magic int32, action string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state != barrierIdle {
		return false
	}
	b.state = barrierSubmitting
	b.clientID = clientID
	b.magic = magic
	b.action = action
	b.expectedTicket = 0
	b.eventCache = make(map[eventCacheKey]*cachedEvent, 4)
	b.eventCacheOrd = b.eventCacheOrd[:0]
	return true
}

// NotifyBrokerAccepted transitions submitting→acceptedUnconfirmed with the
// broker-returned ticket. If a matching confirmation event was already cached
// (order-event-before-response race), transitions directly to confirmed.
// R3: updateType compatibility is checked — a cached event with incompatible
// updateType does NOT confirm. R3-④: when expected magic is non-zero, only
// events with matching magic confirm — magic=0 events are rejected (fund
// boundary precise confirmation). All under the single mutex.
func (b *TradeBarrier) NotifyBrokerAccepted(ticket int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state != barrierSubmitting {
		return
	}
	b.expectedTicket = ticket
	// Check if a matching event was already cached (R4: composite key).
	// R3-④: strict magic match — no magic=0 fallback when expected magic
	// is non-zero. An event with magic=0 cannot prove it belongs to this
	// strategy's order on a shared account.
	key := eventCacheKey{ticket: ticket, magic: b.magic}
	if ev, ok := b.eventCache[key]; ok {
		if ticket != 0 && isUpdateTypeCompatible(b.action, ev.updateType) {
			b.state = barrierConfirmed
			b.cond.Broadcast()
			return
		}
	}
	b.state = barrierAcceptedUnconfirmed
	b.cond.Broadcast()
}

// NotifyConfirmationEvent caches an incoming OnOrderUpdate event. If the
// barrier is in acceptedUnconfirmed state and the event matches the expected
// ticket+magic AND updateType is compatible (R3), transitions to confirmed.
// If in submitting state, the event is cached for later matching when
// NotifyBrokerAccepted arrives. R4: cache is bounded to maxEventCacheEntries.
// R3-④: when expected magic is non-zero, magic=0 events do NOT confirm
// (fund boundary precise confirmation) — they are cached but never matched.
//
// All under the single mutex — the accepted/confirmed transition and
// cache read/write are one atomic critical section.
func (b *TradeBarrier) NotifyConfirmationEvent(ticket int64, magic int32, updateType string) {
	if ticket == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	key := eventCacheKey{ticket: ticket, magic: magic}
	switch b.state {
	case barrierSubmitting:
		b.cacheEvent(key, ticket, magic, updateType)
	case barrierAcceptedUnconfirmed:
		// R3-④: strict magic match. When expected magic is non-zero,
		// reject magic=0 events (can't prove ownership on shared account).
		// When expected magic is zero (legacy/no-magic strategy), accept
		// any magic.
		magicMatch := b.magic == 0 || magic == b.magic
		if ticket == b.expectedTicket && magicMatch &&
			isUpdateTypeCompatible(b.action, updateType) {
			b.state = barrierConfirmed
			b.cond.Broadcast()
		} else {
			b.cacheEvent(key, ticket, magic, updateType)
		}
	}
}

// cacheEvent stores an event in the bounded cache, evicting the oldest
// entry if the cache exceeds maxEventCacheEntries (R4).
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

// WaitState blocks until the barrier reaches the target state or ctx is
// cancelled. Returns the final state. Used by tests for deterministic
// synchronization without time.Sleep (LIVE-ORDER-REENTRY-1 R4 S3).
func (b *TradeBarrier) WaitState(ctx context.Context, target tradeBarrierState) tradeBarrierState {
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

	b.mu.Lock()
	for {
		if b.state == target || ctx.Err() != nil {
			s := b.state
			b.mu.Unlock()
			return s
		}
		b.cond.Wait()
	}
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
