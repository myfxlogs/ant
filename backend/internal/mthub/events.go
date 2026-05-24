// V1-LEGACY: will be replaced by M7.1-7.4 cards. Do not extend; new code goes alongside.
package mthub

import (
	"sync"
)

// OrderEventBroker multiplexes order events from MT sessions
// to multiple subscribers.
type OrderEventBroker struct {
	mu          sync.RWMutex
	subscribers map[string]chan *OrderEvent // accountID → event chan
}

// NewOrderEventBroker creates a new event broker.
func NewOrderEventBroker() *OrderEventBroker {
	return &OrderEventBroker{
		subscribers: make(map[string]chan *OrderEvent),
	}
}

// Subscribe registers a subscriber for the given account and returns
// a channel that receives OrderEvents. Call Unsubscribe when done.
func (b *OrderEventBroker) Subscribe(accountID string) chan *OrderEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan *OrderEvent, 64)
	b.subscribers[accountID] = ch
	return ch
}

// Unsubscribe removes a subscriber. The channel is closed by the broker.
func (b *OrderEventBroker) Unsubscribe(accountID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.subscribers[accountID]; ok {
		delete(b.subscribers, accountID)
		close(ch)
	}
}

// Publish sends an event to the subscriber for the given account.
// Non-blocking — drops if the subscriber's buffer is full.
func (b *OrderEventBroker) Publish(ev *OrderEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	ch, ok := b.subscribers[ev.AccountId]
	if !ok {
		return
	}
	select {
	case ch <- ev:
	default:
		// drop if channel is full — subscriber is too slow
	}
}

// SubscriberCount returns the number of active subscribers.
func (b *OrderEventBroker) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
