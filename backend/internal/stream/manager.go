package stream

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type StreamType string

const (
	StreamTypeQuote   StreamType = "quote"
	StreamTypeOrder   StreamType = "order"
	StreamTypeProfit  StreamType = "profit"
	StreamTypeAccount StreamType = "account"
)

type StreamState int

const (
	StateInactive StreamState = iota
	StateStarting
	StateActive
	StateStopping
	StateError
)

func (s StreamState) String() string {
	switch s {
	case StateInactive:
		return "inactive"
	case StateStarting:
		return "starting"
	case StateActive:
		return "active"
	case StateStopping:
		return "stopping"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

type SubscriptionRecord struct {
	AccountID    uuid.UUID
	ClientID     string
	StreamType   StreamType
	Symbols      []string
	CreatedAt    time.Time
	LastActiveAt time.Time
}

type Manager struct {
	subscriptions map[string]*SubscriptionRecord
	mu            sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		subscriptions: make(map[string]*SubscriptionRecord),
	}
}

func (m *Manager) Start() error {
	return nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	m.subscriptions = make(map[string]*SubscriptionRecord)
	m.mu.Unlock()
}

func (m *Manager) RecordSubscription(accountID uuid.UUID, clientID string, streamType StreamType, symbols []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := accountID.String() + "_" + clientID + "_" + string(streamType)
	m.subscriptions[key] = &SubscriptionRecord{
		AccountID:    accountID,
		ClientID:     clientID,
		StreamType:   streamType,
		Symbols:      symbols,
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}

}

func (m *Manager) RemoveSubscription(accountID uuid.UUID, clientID string, streamType StreamType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := accountID.String() + "_" + clientID + "_" + string(streamType)
	delete(m.subscriptions, key)

}

func (m *Manager) GetSubscriptionStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_subscriptions": len(m.subscriptions),
		"stream_type":         "connect-rpc-server-stream",
		"note":                "Frontend connects via Connect-rpc streaming API",
	}
}

type QuoteData struct {
	AccountID string    `json:"account_id"`
	Symbol    string    `json:"symbol"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Time      time.Time `json:"time"`
}

type OrderData struct {
	AccountID string      `json:"account_id"`
	Ticket    int64       `json:"ticket"`
	Symbol    string      `json:"symbol"`
	OrderType string      `json:"order_type"`
	Volume    float64     `json:"volume"`
	Price     float64     `json:"price"`
	Profit    float64     `json:"profit"`
	Action    string      `json:"action"`
	MTType    string      `json:"mt_type"`
	RawOrder  interface{} `json:"-"`
}

type ProfitData struct {
	AccountID   string  `json:"account_id"`
	Balance     float64 `json:"balance"`
	Credit      float64 `json:"credit"`
	Profit      float64 `json:"profit"`
	Equity      float64 `json:"equity"`
	Margin      float64 `json:"margin"`
	FreeMargin  float64 `json:"free_margin"`
	MarginLevel float64 `json:"margin_level"`
}
