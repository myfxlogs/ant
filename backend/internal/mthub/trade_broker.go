// trade_broker.go — TradeBroker fans out trade events (fills, closes, modifies)
// to strategy runners. Follows the same channel-fanout pattern as BarBroker.

package mthub

import (
	"sync"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// BrokerTradeEventType classifies the type of trade broker event.
type BrokerTradeEventType int8

const (
	BrokerTradeFilled   BrokerTradeEventType = iota
	BrokerTradeClosed
	BrokerTradeModified
	BrokerTradeCancelled
)

// BrokerTradeEvent carries information about a completed trade action on an account.
type BrokerTradeEvent struct {
	AccountID  string
	Ticket     int64
	Symbol     string
	EventType  BrokerTradeEventType
	Side       string // "buy" | "sell"
	Volume     decimal.Decimal
	Price      decimal.Decimal
	StopLoss   decimal.Decimal
	TakeProfit decimal.Decimal
	Profit     decimal.Decimal
	Commission decimal.Decimal
	Swap       decimal.Decimal
}

// TradeBroker fans out BrokerTradeEvent events to per-account subscribers.
type TradeBroker struct {
	mu     sync.RWMutex
	subs   map[string][]chan *BrokerTradeEvent // accountID → subscribers
	maxBuf int
	log    *zap.Logger
}

// NewTradeBroker creates a TradeBroker with the given channel buffer size.
func NewTradeBroker(bufSize int, log *zap.Logger) *TradeBroker {
	return &TradeBroker{
		subs:   make(map[string][]chan *BrokerTradeEvent),
		maxBuf: bufSize,
		log:    log,
	}
}

// Subscribe returns a channel that receives BrokerTradeEvent events for the given account.
func (b *TradeBroker) Subscribe(accountID string) (<-chan *BrokerTradeEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan *BrokerTradeEvent, b.maxBuf)
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

// Publish sends a BrokerTradeEvent to all subscribers for the given account.
func (b *TradeBroker) Publish(evt *BrokerTradeEvent) {
	b.mu.RLock()
	subs := b.subs[evt.AccountID]
	b.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- evt:
		default:
			b.log.Warn("TradeBroker: dropping trade event for slow consumer",
				zap.String("account", evt.AccountID),
				zap.Int64("ticket", evt.Ticket))
		}
	}
}
