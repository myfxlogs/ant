package mthub

import (
	"context"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// Session holds a broker connection session with renewal logic.
type Session struct {
	AccountID string
	CreatedAt time.Time
	MaxAge    time.Duration // default 4h (Q-014)
}

// IsExpired checks if the session has exceeded its max age.
func (s *Session) IsExpired() bool {
	if s.MaxAge <= 0 { s.MaxAge = 4 * time.Hour }
	return time.Since(s.CreatedAt) > s.MaxAge
}

// Hub is the session registry. All MT account sessions are registered here.
type Hub struct {
	mu       sync.RWMutex
	sessions map[string]*Session    // accountID → session
	executors map[string]OrderExecutor // accountID → executor
}

// NewHub creates a session hub.
func NewHub() *Hub {
	return &Hub{sessions: make(map[string]*Session), executors: make(map[string]OrderExecutor)}
}

// Register adds a session and executor for the given account.
func (h *Hub) Register(accountID string, sess *Session, exec OrderExecutor) {
	h.mu.Lock(); defer h.mu.Unlock()
	h.sessions[accountID] = sess
	h.executors[accountID] = exec
}

// Get returns the executor for an account.
func (h *Hub) Get(accountID string) OrderExecutor {
	h.mu.RLock(); defer h.mu.RUnlock()
	return h.executors[accountID]
}

// EnsureSession returns a valid session, renewing if expired.
func (h *Hub) EnsureSession(ctx context.Context, accountID string) (*Session, error) {
	h.mu.RLock()
	sess := h.sessions[accountID]
	h.mu.RUnlock()
	if sess == nil { return nil, ErrSessionNotFound }
	if sess.IsExpired() {
		sess.CreatedAt = time.Now() // renewal: extend lifetime
	}
	return sess, nil
}

// CloseSession removes a session from the registry.
func (h *Hub) CloseSession(ctx context.Context, accountID string) error {
	h.mu.Lock(); defer h.mu.Unlock()
	delete(h.sessions, accountID)
	delete(h.executors, accountID)
	return nil
}

// ErrSessionNotFound is returned when no session is registered.
var ErrSessionNotFound = &HubError{Msg: "session not found"}

// HubError is a sentinel error for hub operations.
type HubError struct{ Msg string }
func (e *HubError) Error() string { return "mthub: " + e.Msg }

// OrderRequest is a normalized order placement request.
type OrderRequest struct {
	AccountID  string
	Canonical  string
	Side       Side
	OrderType  OrderType
	Volume     decimal.Decimal
	Price      decimal.Decimal
	StopLoss   decimal.Decimal
	TakeProfit decimal.Decimal
	Comment    string
	ClientID   string
	Magic      int32
}

// OrderRecord represents a broker order snapshot.
type OrderRecord struct {
	Ticket     int64
	AccountID  string
	SymbolRaw  string
	Canonical  string
	Side       Side
	OrderType  OrderType
	Volume     decimal.Decimal
	OpenPrice  decimal.Decimal
	OpenTime   time.Time
	ClosePrice decimal.Decimal
	CloseTime  time.Time
	Profit     decimal.Decimal
	Commission decimal.Decimal
	Swap       decimal.Decimal
	Comment    string
	Magic      int32
	State      OrderState
}

// SymbolParam holds broker trading parameters for a symbol.
type SymbolParam struct {
	Canonical   string
	SymbolRaw   string
	Digits      int32
	PointValue  decimal.Decimal
	LotSize     decimal.Decimal
	LotStep     decimal.Decimal
	LotMin      decimal.Decimal
	LotMax      decimal.Decimal
	SpreadFloat bool
	TradeMode   int32
	StopLevel   int32
}

type Side int8
const (
	SideBuy  Side = 1
	SideSell Side = -1
)

type OrderType int8
const (
	OrderMarket    OrderType = iota
	OrderLimit
	OrderStop
	OrderStopLimit
)

type OrderState int8
const (
	OrderStatePending   OrderState = iota
	OrderStateOpen
	OrderStateClosed
	OrderStateCancelled
	OrderStateRejected
)

// OrderEvent is a broker order lifecycle event.
type OrderEvent struct {
	AccountID string
	Ticket    int64
	EventType string // "open"|"close"|"modify"|"delete"
	Order     *OrderRecord
	Timestamp time.Time
}

// OrderEventHandler is called for each broker order event.
type OrderEventHandler func(*OrderEvent)

// OrderExecutor is the interface for broker order operations.
type OrderExecutor interface {
	Platform() string
	PlaceOrder(ctx context.Context, req *OrderRequest) (ticket int64, err error)
	CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error
	ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error
	FetchOpenedOrders(ctx context.Context) ([]*OrderRecord, error)
	FetchOrderHistory(ctx context.Context, from, to time.Time) ([]*OrderRecord, error)
	FetchSymbolParams(ctx context.Context, canonicals []string) ([]*SymbolParam, error)
	SubscribeOrderEvents(ctx context.Context, h OrderEventHandler) error
}

// OrderEventBroker fans-in broker order events to subscribed clients.
type OrderEventBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *OrderEvent // userID → channels
}

// NewOrderEventBroker creates an event broker.
func NewOrderEventBroker() *OrderEventBroker {
	return &OrderEventBroker{subscribers: make(map[string][]chan *OrderEvent)}
}

// PublishEvent fans-in an order event to all subscribers of the user.
func (b *OrderEventBroker) PublishEvent(userID string, ev *OrderEvent) {
	b.mu.RLock()
	chs := b.subscribers[userID]
	b.mu.RUnlock()
	for _, ch := range chs {
		select {
		case ch <- ev:
		default:
			// channel full, drop oldest
		}
	}
}

// Subscribe returns a channel for receiving order events for a user.
func (b *OrderEventBroker) Subscribe(userID string) (<-chan *OrderEvent, func()) {
	ch := make(chan *OrderEvent, 64)
	b.mu.Lock()
	b.subscribers[userID] = append(b.subscribers[userID], ch)
	b.mu.Unlock()
	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		list := b.subscribers[userID]
		for i, c := range list {
			if c == ch {
				b.subscribers[userID] = append(list[:i], list[i+1:]...)
				close(ch)
				return
			}
		}
	}
	return ch, cancel
}
