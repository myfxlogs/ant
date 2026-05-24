package mthub

import (
	"context"
	"sync"
	"time"
	"github.com/shopspring/decimal"
)

type Session struct {
	AccountID string; CreatedAt time.Time; MaxAge time.Duration
}
func (s *Session) IsExpired() bool {
	if s.MaxAge <= 0 { s.MaxAge = 4 * time.Hour }
	return time.Since(s.CreatedAt) > s.MaxAge
}

type Hub struct {
	mu sync.RWMutex; sessions map[string]*Session; executors map[string]OrderExecutor
}
func NewHub() *Hub { return &Hub{sessions: map[string]*Session{}, executors: map[string]OrderExecutor{}} }
func (h *Hub) Register(id string, s *Session, e OrderExecutor) { h.mu.Lock(); defer h.mu.Unlock(); h.sessions[id]=s; h.executors[id]=e }
func (h *Hub) Get(id string) OrderExecutor { h.mu.RLock(); defer h.mu.RUnlock(); return h.executors[id] }
func (h *Hub) EnsureSession(ctx context.Context, id string) (*Session, error) {
	h.mu.RLock(); s := h.sessions[id]; h.mu.RUnlock()
	if s == nil { return nil, ErrSessionNotFound }
	if s.IsExpired() { s.CreatedAt = time.Now() }
	return s, nil
}
func (h *Hub) CloseSession(ctx context.Context, id string) error {
	h.mu.Lock(); defer h.mu.Unlock(); delete(h.sessions, id); delete(h.executors, id); return nil
}

var ErrSessionNotFound = &HubError{Msg: "session not found"}
type HubError struct{ Msg string }
func (e *HubError) Error() string { return "mthub: " + e.Msg }

type OrderRequest struct {
	AccountID, Canonical string; Side Side; OrderType OrderType
	Volume, Price, StopLoss, TakeProfit decimal.Decimal
	Comment, ClientID string; Magic int32
}
type OrderRecord struct {
	Ticket int64; AccountID, SymbolRaw, Canonical string
	Side Side; OrderType OrderType
	Volume, OpenPrice, ClosePrice, Profit, Commission, Swap decimal.Decimal
	OpenTime, CloseTime time.Time; Comment string; Magic int32; State OrderState
}
type SymbolParam struct {
	Canonical, SymbolRaw string; Digits, TradeMode, StopLevel int32
	PointValue, LotSize, LotStep, LotMin, LotMax decimal.Decimal; SpreadFloat bool
}

type Side int8
const (SideBuy Side=1; SideSell Side=-1)
type OrderType int8
const (OrderMarket OrderType=iota; OrderLimit; OrderStop; OrderStopLimit)
type OrderState int8
const (OrderStatePending OrderState=iota; OrderStateOpen; OrderStateClosed; OrderStateCancelled; OrderStateRejected)

type OrderEvent struct {
	AccountID string; Ticket int64; EventType string; Order *OrderRecord; Timestamp time.Time
}
type OrderEventHandler func(*OrderEvent)

type OrderExecutor interface {
	Platform() string
	PlaceOrder(ctx context.Context, req *OrderRequest) (int64, error)
	CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error
	ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error
	FetchOpenedOrders(ctx context.Context) ([]*OrderRecord, error)
	FetchOrderHistory(ctx context.Context, from, to time.Time) ([]*OrderRecord, error)
	FetchSymbolParams(ctx context.Context, canonicals []string) ([]*SymbolParam, error)
	SubscribeOrderEvents(ctx context.Context, h OrderEventHandler) error
}

type OrderEventBroker struct {
	mu sync.RWMutex; subscribers map[string][]chan *OrderEvent
}
func NewOrderEventBroker() *OrderEventBroker { return &OrderEventBroker{subscribers: map[string][]chan *OrderEvent{}} }
func (b *OrderEventBroker) PublishEvent(userID string, ev *OrderEvent) {
	b.mu.RLock(); chs := b.subscribers[userID]; b.mu.RUnlock()
	for _, ch := range chs { select { case ch <- ev: default: } }
}
func (b *OrderEventBroker) Subscribe(userID string) (<-chan *OrderEvent, func()) {
	ch := make(chan *OrderEvent, 64)
	b.mu.Lock(); b.subscribers[userID] = append(b.subscribers[userID], ch); b.mu.Unlock()
	return ch, func() {
		b.mu.Lock(); defer b.mu.Unlock()
		for i, c := range b.subscribers[userID] { if c == ch { b.subscribers[userID] = append(b.subscribers[userID][:i], b.subscribers[userID][i+1:]...); close(ch); return } }
	}
}
