package mthub

import "sync"

// Precision note: financial fields in SSE push types (PositionSnapshot, BarUpdate)
// use float64 for display efficiency. These are real-time visual updates, NOT used
// for trading calculations or persistent storage. For price-sensitive operations:
//   - Trading execution: uses decimal.Decimal (mthub/service.go PlaceOrder)
//   - Persistent storage: PG NUMERIC(20,8) / CH Decimal(18,6)
//   - Backtest computation: Python decimal.Decimal
// float64 provides ~15 significant digits — sufficient for Forex price display
// (typical quote: 1.12345 has 6 significant digits, well within safe range).
// Cumulative rounding errors could appear for large trade counts (>10^6) shown
// in position summaries; use PG/CH for authoritative P&L computation.

// --- Position snapshots (full OpenedOrders list from OnOrderUpdate) ---

// PositionSnapshot is a complete account position list pushed from OnOrderUpdate stream.
type PositionSnapshot struct {
	AccountID   string
	UserID      string
	Platform    string
	Balance     float64
	Credit      float64
	Equity      float64
	Margin      float64
	FreeMargin  float64
	MarginLevel float64
	Profit      float64
	Positions   []PositionSnapshotItem
}

type PositionSnapshotItem struct {
	Ticket       int64
	Symbol       string
	Type         string
	Volume       float64
	OpenPrice    float64
	CurrentPrice float64
	StopLoss     float64
	TakeProfit   float64
	Profit       float64
	Swap         float64
	Commission   float64
	Comment      string
	OpenTime     int64
}

// PositionSnapshotBroker broadcasts full position snapshots per accountID.
type PositionSnapshotBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *PositionSnapshot
}

func NewPositionSnapshotBroker() *PositionSnapshotBroker {
	return &PositionSnapshotBroker{subscribers: map[string][]chan *PositionSnapshot{}}
}

func (b *PositionSnapshotBroker) Publish(ev *PositionSnapshot) {
	b.mu.RLock()
	src := b.subscribers[ev.AccountID]
	b.mu.RUnlock()
	chs := make([]chan *PositionSnapshot, len(src))
	copy(chs, src)
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

// --- Bar updates (real-time K-line bar push) ---

// BarUpdate is a single OHLCV bar update pushed from the bar aggregator.
type BarUpdate struct {
	AccountID string
	Symbol    string
	Period    string
	OpenTime  int64 // unix milliseconds
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Bid       float64 // latest bid for real-time quote display
	Ask       float64 // latest ask for real-time quote display
	Volume    float64
	Closed    bool // true=finalized bar, false=in-progress candle
}

// BarBroker broadcasts bar updates per accountID.
type BarBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *BarUpdate
}

func NewBarBroker() *BarBroker {
	return &BarBroker{subscribers: map[string][]chan *BarUpdate{}}
}
func (b *BarBroker) Publish(ev *BarUpdate) {
	b.mu.RLock()
	src := b.subscribers[ev.AccountID]
	b.mu.RUnlock()
	chs := make([]chan *BarUpdate, len(src))
	copy(chs, src)
	for _, ch := range chs {
		select {
		case ch <- ev:
		default:
		}
	}
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
