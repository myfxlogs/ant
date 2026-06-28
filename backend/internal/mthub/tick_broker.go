// tick_broker.go — TickBroker fans out real-time quote (Bid/Ask) updates
// from mdgateway to strategy runners. Follows the same channel-fanout pattern
// as BarBroker.

package mthub

import (
	"sync"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// TickUpdate carries a single quote update for a symbol on an account.
type TickUpdate struct {
	AccountID string
	Symbol    string
	Bid       decimal.Decimal
	Ask       decimal.Decimal
}

// TickBroker fans out TickUpdate events to per-account subscribers.
// Each subscriber gets a buffered channel; slow consumers are dropped.
type TickBroker struct {
	mu      sync.RWMutex
	subs    map[string][]chan *TickUpdate // accountID → subscribers
	maxBuf  int
	log     *zap.Logger
}

// NewTickBroker creates a TickBroker with the given channel buffer size.
func NewTickBroker(bufSize int, log *zap.Logger) *TickBroker {
	return &TickBroker{
		subs:   make(map[string][]chan *TickUpdate),
		maxBuf: bufSize,
		log:    log,
	}
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
				close(c)
				return
			}
		}
	}
	return ch, cancel
}

// Publish sends a TickUpdate to all subscribers for the given account.
func (b *TickBroker) Publish(u *TickUpdate) {
	b.mu.RLock()
	subs := b.subs[u.AccountID]
	b.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- u:
		default:
			b.log.Warn("TickBroker: dropping tick for slow consumer",
				zap.String("account", u.AccountID),
				zap.String("symbol", u.Symbol))
		}
	}
}
