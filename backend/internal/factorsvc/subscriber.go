package factorsvc

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// FactorValue is a computed factor value event.
type FactorValue struct {
	UserID string
	Factor   string
	Symbol   string
	TsUnixMs int64
	Value    float64
}

// FactorHandler is a callback invoked for each computed factor value.
type FactorHandler func(ctx context.Context, fv FactorValue)

// Subscriber manages fan-out distribution of factor values to registered handlers.
type Subscriber struct {
	mu       sync.RWMutex
	handlers map[string]FactorHandler // keyed by subscription id
	nextID   int
	log      *zap.Logger
}

// NewSubscriber creates a new Subscriber.
func NewSubscriber(log *zap.Logger) *Subscriber {
	return &Subscriber{
		handlers: make(map[string]FactorHandler),
		log:     log,
	}
}

// Subscribe registers a handler and returns a subscription ID for later removal.
func (s *Subscriber) Subscribe(h FactorHandler) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	id := string(rune(s.nextID))
	s.handlers[id] = h
	s.log.Debug("subscriber: handler registered", zap.String("id", id))
	return id
}

// Unsubscribe removes a previously registered handler.
func (s *Subscriber) Unsubscribe(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.handlers, id)
}

// M7.8-10: TODO — switch from in-process dispatch to NATS subscription.
// Current: factor values are delivered via in-process handler fan-out.
// Target: subscribe to NATS subject "md_events.bar" and dispatch to handlers.
// This decouples factorsvc from mdgateway and enables horizontal scaling.
//
// Dispatch sends a factor value to all registered handlers concurrently.
func (s *Subscriber) Dispatch(ctx context.Context, fv FactorValue) {
	s.mu.RLock()
	handlers := make([]FactorHandler, 0, len(s.handlers))
	for _, h := range s.handlers {
		handlers = append(handlers, h)
	}
	s.mu.RUnlock()

	var wg sync.WaitGroup
	for _, h := range handlers {
		wg.Add(1)
		go func(h FactorHandler) {
			defer wg.Done()
			h(ctx, fv)
		}(h)
	}
	wg.Wait()
}

// DispatchBatch sends multiple factor values to all registered handlers.
func (s *Subscriber) DispatchBatch(ctx context.Context, fvs []FactorValue) {
	for _, fv := range fvs {
		s.Dispatch(ctx, fv)
	}
}

// HandlerCount returns the number of registered handlers.
func (s *Subscriber) HandlerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.handlers)
}
