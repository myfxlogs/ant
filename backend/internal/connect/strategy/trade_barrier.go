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
	"sync"

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
