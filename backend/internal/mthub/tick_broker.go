// tick_broker.go — TickBroker fans out real-time quote (Bid/Ask) updates
// from mdgateway to strategy runners. Follows the same channel-fanout pattern
// as BarBroker.

package mthub

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// TickUpdate carries a single quote update for a symbol on an account.
type TickUpdate struct {
	AccountID string
	Symbol    string
	Bid       decimal.Decimal
	Ask       decimal.Decimal
	Time      time.Time // server-side timestamp of receipt
}

// TickBroker fans out TickUpdate events to per-account subscribers.
// Each subscriber gets a buffered channel; slow consumers are dropped.
// Also caches the latest tick per (accountID, symbol) for market order
// price resolution in the risk gate (RISK-MARGIN1).
type TickBroker struct {
	mu       sync.RWMutex
	subs     map[string][]chan *TickUpdate // accountID → subscribers
	watchers []chan *TickUpdate            // global watchers (WatchAll)
	latest   map[string]*TickUpdate        // "accountID:symbol" → latest tick
	maxBuf   int
	log      *zap.Logger
}

// NewTickBroker creates a TickBroker with the given channel buffer size.
func NewTickBroker(bufSize int, log *zap.Logger) *TickBroker {
	return &TickBroker{
		subs:   make(map[string][]chan *TickUpdate),
		latest: make(map[string]*TickUpdate),
		maxBuf: bufSize,
		log:    log,
	}
}

// LatestTick returns the most recent tick for the given account+symbol.
// Returns nil if no tick has been received yet.
func (b *TickBroker) LatestTick(accountID, symbol string) *TickUpdate {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.latest[accountID+":"+symbol]
}

// Subscribe returns a channel that receives TickUpdate events for the given account.
// The returned cancel function removes the subscription.
func (b *TickBroker) Subscribe(accountID string) (<-chan *TickUpdate, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan *TickUpdate, b.maxBuf)
	b.subs[accountID] = append(b.subs[accountID], ch)
	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subs := b.subs[accountID]
		for i, c := range subs {
			if c == ch {
				b.subs[accountID] = append(subs[:i], subs[i+1:]...)
				// Do NOT close the channel — Publish() may still be sending on it after RUnlock
				return
			}
		}
	}
	return ch, cancel
}

// Publish sends a TickUpdate to all subscribers for the given account.
// Also caches the latest tick per (accountID, symbol) for LatestTick lookups.
func (b *TickBroker) Publish(u *TickUpdate) {
	if u.Time.IsZero() {
		u.Time = time.Now()
	}
	b.mu.Lock()
	b.latest[u.AccountID+":"+u.Symbol] = u
	subs := b.subs[u.AccountID]
	watchers := b.watchers
	b.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- u:
		default:
			b.log.Warn("TickBroker: dropping tick for slow consumer",
				zap.String("account", u.AccountID),
				zap.String("symbol", u.Symbol))
		}
	}
	for _, ch := range watchers {
		select {
		case ch <- u:
		default:
		}
	}
}

// WatchAll returns a channel that receives ALL tick updates across all accounts.
// Used by WatchActiveStrategies to push real-time prices to the Active Runs table.
// The returned cancel function removes the subscription.
func (b *TickBroker) WatchAll() (<-chan *TickUpdate, func()) {
	ch := make(chan *TickUpdate, 128)
	b.mu.Lock()
	b.watchers = append(b.watchers, ch)
	b.mu.Unlock()
	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, c := range b.watchers {
			if c == ch {
				b.watchers = append(b.watchers[:i], b.watchers[i+1:]...)
				return
			}
		}
	}
	return ch, cancel
}
