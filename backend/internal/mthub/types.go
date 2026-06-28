package mthub

import (
	"context"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type Session struct {
	AccountID string
	CreatedAt time.Time
	MaxAge    time.Duration
}

func (s *Session) IsExpired() bool {
	if s.MaxAge <= 0 {
		s.MaxAge = 4 * time.Hour
	}
	return time.Since(s.CreatedAt) > s.MaxAge
}

type Hub struct {
	mu        sync.RWMutex
	sessions  map[string]*Session
	executors map[string]OrderExecutor
	waiters   map[string][]chan struct{} // signaled on Register
}

func NewHub() *Hub {
	return &Hub{
		sessions:  map[string]*Session{},
		executors: map[string]OrderExecutor{},
		waiters:   map[string][]chan struct{}{},
	}
}

// Register adds a session and signals any WaitSession callers for this account.
func (h *Hub) Register(id string, s *Session, e OrderExecutor) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessions[id] = s
	h.executors[id] = e
	for _, ch := range h.waiters[id] {
		close(ch)
	}
	delete(h.waiters, id)
}

// WaitSession returns a channel that closes when a session is registered for id,
// or immediately if the session already exists. The caller should select on the
// returned channel and ctx.Done() — no polling needed.
func (h *Hub) WaitSession(id string) <-chan struct{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.sessions[id]; ok {
		ch := make(chan struct{})
		close(ch) // already ready
		return ch
	}
	ch := make(chan struct{})
	h.waiters[id] = append(h.waiters[id], ch)
	return ch
}

func (h *Hub) Get(id string) OrderExecutor {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.executors[id]
}

func (h *Hub) EnsureSession(ctx context.Context, id string) (*Session, error) {
	h.mu.RLock()
	s := h.sessions[id]
	h.mu.RUnlock()
	if s != nil {
		if s.IsExpired() {
			return nil, ErrSessionNotFound
		}
		return s, nil
	}
	return nil, ErrSessionNotFound
}

// RemoveSession closes a session and removes its executor.
// Called by the gateway manager on disconnect.
func (h *Hub) RemoveSession(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sessions, id)
	delete(h.executors, id)
}

func (h *Hub) CloseSession(ctx context.Context, id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sessions, id)
	delete(h.executors, id)
	return nil
}

func (h *Hub) ActiveAccountIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.sessions))
	for id := range h.sessions {
		ids = append(ids, id)
	}
	return ids
}

var ErrSessionNotFound = &HubError{Msg: "session not found"}

type HubError struct{ Msg string }

func (e *HubError) Error() string { return "mthub: " + e.Msg }

// --- Account profit events ---

type AccountProfitEvent struct {
	AccountID                               string
	UserID                                  string
	Platform                                string
	Status                                  string
	Balance, Credit, Equity                 decimal.Decimal
	Margin, FreeMargin, MarginLevel, Profit decimal.Decimal
	ProfitPercent                           float64
	Timestamp                               time.Time
	Positions                               []AccountProfitPosition
}

type AccountProfitPosition struct {
	Ticket       int64
	Symbol       string
	Profit       decimal.Decimal
	Volume       decimal.Decimal
	CurrentPrice decimal.Decimal
}

type AccountProfitBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *AccountProfitEvent
}

func NewAccountProfitBroker() *AccountProfitBroker {
	return &AccountProfitBroker{subscribers: map[string][]chan *AccountProfitEvent{}}
}

func (b *AccountProfitBroker) Publish(ev *AccountProfitEvent) {
	b.mu.RLock()
	src := b.subscribers[ev.AccountID]
	b.mu.RUnlock()
	chs := make([]chan *AccountProfitEvent, len(src))
	copy(chs, src)
	for _, ch := range chs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (b *AccountProfitBroker) Subscribe(accountID string) (<-chan *AccountProfitEvent, func()) {
	ch := make(chan *AccountProfitEvent, 64)
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

// --- Order events ---

type OrderEventBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *OrderEvent
}

func NewOrderEventBroker() *OrderEventBroker {
	return &OrderEventBroker{subscribers: map[string][]chan *OrderEvent{}}
}
func (b *OrderEventBroker) PublishEvent(userID string, ev *OrderEvent) {
	b.mu.RLock()
	src := b.subscribers[userID]
	b.mu.RUnlock()
	chs := make([]chan *OrderEvent, len(src))
	copy(chs, src)
	for _, ch := range chs {
		select {
		case ch <- ev:
		default:
		}
	}
}
func (b *OrderEventBroker) Subscribe(userID string) (<-chan *OrderEvent, func()) {
	ch := make(chan *OrderEvent, 64)
	b.mu.Lock()
	b.subscribers[userID] = append(b.subscribers[userID], ch)
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, c := range b.subscribers[userID] {
			if c == ch {
				b.subscribers[userID] = append(b.subscribers[userID][:i], b.subscribers[userID][i+1:]...)
				close(ch)
				return
			}
		}
	}
}
