package mthub

import (
	"log"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// --- Position snapshots (full OpenedOrders list from OnOrderUpdate) ---

// PositionSnapshot is a complete account position list pushed from OnOrderUpdate stream.
type PositionSnapshot struct {
	AccountID   string
	UserID      string
	Platform    string
	Balance     decimal.Decimal
	Credit      decimal.Decimal
	Equity      decimal.Decimal
	Margin      decimal.Decimal
	FreeMargin  decimal.Decimal
	MarginLevel decimal.Decimal
	Profit      decimal.Decimal
	Positions   []PositionSnapshotItem
}

type PositionSnapshotItem struct {
	Ticket       int64
	Symbol       string
	Type         string
	Volume       decimal.Decimal
	OpenPrice    decimal.Decimal
	CurrentPrice decimal.Decimal
	StopLoss     decimal.Decimal
	TakeProfit   decimal.Decimal
	Profit       decimal.Decimal
	Swap         decimal.Decimal
	Commission   decimal.Decimal
	Comment      string
	OpenTime     int64
}

// PositionSnapshotBroker broadcasts full position snapshots per accountID.
type PositionSnapshotBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *PositionSnapshot
	allSubs     []chan *PositionSnapshot
}

func NewPositionSnapshotBroker() *PositionSnapshotBroker {
	return &PositionSnapshotBroker{subscribers: map[string][]chan *PositionSnapshot{}}
}

func (b *PositionSnapshotBroker) Publish(ev *PositionSnapshot) {
	b.mu.RLock()
	src := b.subscribers[ev.AccountID]
	all := b.allSubs
	b.mu.RUnlock()
	chs := make([]chan *PositionSnapshot, 0, len(src)+len(all))
	chs = append(chs, src...)
	chs = append(chs, all...)
	for _, ch := range chs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (b *PositionSnapshotBroker) Subscribe(accountID string) (<-chan *PositionSnapshot, func()) {
	ch := make(chan *PositionSnapshot, 8)
	b.mu.Lock()
	b.subscribers[accountID] = append(b.subscribers[accountID], ch)
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, c := range b.subscribers[accountID] {
			if c == ch {
				b.subscribers[accountID] = append(b.subscribers[accountID][:i], b.subscribers[accountID][i+1:]...)
				close(ch)
				return
			}
		}
	}
}

// SubscribeAll returns a channel receiving all snapshots regardless of accountID.
// Used by the SnapshotPersister for throttled PG persistence.
func (b *PositionSnapshotBroker) SubscribeAll() (<-chan *PositionSnapshot, func()) {
	ch := make(chan *PositionSnapshot, 64)
	b.mu.Lock()
	b.allSubs = append(b.allSubs, ch)
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, c := range b.allSubs {
			if c == ch {
				b.allSubs = append(b.allSubs[:i], b.allSubs[i+1:]...)
				close(ch)
				return
			}
		}
	}
}

// --- Bar updates (real-time K-line bar push) ---

// BarUpdate is a single OHLCV bar update pushed from the bar aggregator.
type BarUpdate struct {
	AccountID string
	Symbol    string
	Period    string
	OpenTime  int64 // unix milliseconds
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Bid       decimal.Decimal // latest bid for real-time quote display
	Ask       decimal.Decimal // latest ask for real-time quote display
	Volume    float64
	Closed    bool // true=finalized bar, false=in-progress candle
}

// BarBroker broadcasts bar updates per accountID.
type BarBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *BarUpdate
	drops       map[string]int64 // accountID → dropped bar count
}

func NewBarBroker() *BarBroker {
	return &BarBroker{
		subscribers: map[string][]chan *BarUpdate{},
		drops:       map[string]int64{},
	}
}
func (b *BarBroker) Publish(ev *BarUpdate) {
	b.mu.RLock()
	src := b.subscribers[ev.AccountID]
	b.mu.RUnlock()
	chs := make([]chan *BarUpdate, len(src))
	copy(chs, src)
	dropped := false
	for _, ch := range chs {
		select {
		case ch <- ev:
		default:
			dropped = true
		}
	}
	if dropped {
		b.mu.Lock()
		b.drops[ev.AccountID]++
		total := b.drops[ev.AccountID]
		b.mu.Unlock()
		// Log every 100th drop to avoid spam.
		if total%100 == 1 {
			log.Printf("WARNING: BarBroker dropped bars for account %s (total drops: %d, buffer: 64). Strategy is too slow or timeframe is too short.", ev.AccountID, total)
		}
	}
}

// DroppedBars returns the count of dropped bars for an account.
func (b *BarBroker) DroppedBars(accountID string) int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.drops[accountID]
}
func (b *BarBroker) Subscribe(accountID string) (<-chan *BarUpdate, func()) {
	ch := make(chan *BarUpdate, 64)
	b.mu.Lock()
	b.subscribers[accountID] = append(b.subscribers[accountID], ch)
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, c := range b.subscribers[accountID] {
			if c == ch {
				b.subscribers[accountID] = append(b.subscribers[accountID][:i], b.subscribers[accountID][i+1:]...)
				close(ch)
				return
			}
		}
	}
}

// --- Account status (per-account connection state push) ---

// AccountStatusEvent is emitted when a gateway's connection state changes
// (connected → reconnecting → disconnected). It carries the actual error
// message (e.g. "rpc error: code = Unauthenticated desc = token expired")
// so the frontend can display diagnostic information in real time.
type AccountStatusEvent struct {
	AccountID string
	UserID    string
	Status    string // "connected" | "reconnecting" | "disconnected"
	Message   string // error detail when status != connected; empty on connect
	Timestamp time.Time
}

// AccountStatusBroker broadcasts connection state changes per accountID.
type AccountStatusBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *AccountStatusEvent
}

func NewAccountStatusBroker() *AccountStatusBroker {
	return &AccountStatusBroker{subscribers: map[string][]chan *AccountStatusEvent{}}
}

func (b *AccountStatusBroker) Publish(ev *AccountStatusEvent) {
	b.mu.RLock()
	src := b.subscribers[ev.AccountID]
	b.mu.RUnlock()
	chs := make([]chan *AccountStatusEvent, len(src))
	copy(chs, src)
	for _, ch := range chs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (b *AccountStatusBroker) Subscribe(accountID string) (<-chan *AccountStatusEvent, func()) {
	ch := make(chan *AccountStatusEvent, 8)
	b.mu.Lock()
	b.subscribers[accountID] = append(b.subscribers[accountID], ch)
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, c := range b.subscribers[accountID] {
			if c == ch {
				b.subscribers[accountID] = append(b.subscribers[accountID][:i], b.subscribers[accountID][i+1:]...)
				close(ch)
				return
			}
		}
	}
}
