package event

import (
	"sync"
)

type EventType string

const (
	EventQuoteUpdate       EventType = "quote_update"
	EventOrderUpdate       EventType = "order_update"
	EventProfitUpdate      EventType = "profit_update"
	EventAccountStatus     EventType = "account_status"
	EventStrategyExecution EventType = "strategy_execution"
)

type Event struct {
	Type      EventType
	AccountID string
	Data      interface{}
}

type Handler func(event *Event)

type Bus struct {
	handlers map[string][]Handler
	mu       sync.RWMutex
}

var globalBus = &Bus{
	handlers: make(map[string][]Handler),
}

func GetBus() *Bus {
	return globalBus
}

func (b *Bus) Subscribe(accountID string, handler Handler) func() {
	b.mu.Lock()
	b.handlers[accountID] = append(b.handlers[accountID], handler)
	b.mu.Unlock()

	return func() {
		b.mu.Lock()
		handlers := b.handlers[accountID]
		for i, h := range handlers {
			if &h == &handler {
				b.handlers[accountID] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
		b.mu.Unlock()
	}
}

func (b *Bus) Publish(accountID string, event *Event) {
	b.mu.RLock()
	handlers := b.handlers[accountID]
	b.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}

	b.mu.RLock()
	globalHandlers := b.handlers["*"]
	b.mu.RUnlock()

	for _, handler := range globalHandlers {
		go handler(event)
	}
}

func (b *Bus) PublishAll(event *Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, handlers := range b.handlers {
		for _, handler := range handlers {
			go handler(event)
		}
	}
}

type StrategyExecutionData struct {
	TemplateID  string `json:"template_id"`
	ExecutionID string `json:"execution_id"`
	AccountID   string `json:"account_id"`
	Status      string `json:"status"`
	Symbol      string `json:"symbol"`
	Action      string `json:"action"`
	Ticket      int64  `json:"ticket"`
	Volume      string `json:"volume"`
	Price       string `json:"price"`
	Profit      string `json:"profit"`
	Message     string `json:"message"`
}
