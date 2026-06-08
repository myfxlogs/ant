// Package notification provides in-process pubsub for SSE notification delivery.
// Each connected SSE stream subscribes to a per-user channel; SendNotification
// publishes to the matching channel so the stream pushes the event in real time.
package notification

import (
	"sync"

	"github.com/google/uuid"

	"anttrader/internal/repository"
)

// Subscriber receives notification rows pushed by SendNotification.
type Subscriber struct {
	mu   sync.Mutex
	subs map[uuid.UUID]map[chan repository.NotificationRow]struct{}
}

// NewSubscriber creates a Subscriber ready for use.
func NewSubscriber() *Subscriber {
	return &Subscriber{
		subs: make(map[uuid.UUID]map[chan repository.NotificationRow]struct{}),
	}
}

// Subscribe registers a channel for a user and returns it.
func (s *Subscriber) Subscribe(userID uuid.UUID) chan repository.NotificationRow {
	ch := make(chan repository.NotificationRow, 16)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.subs[userID] == nil {
		s.subs[userID] = make(map[chan repository.NotificationRow]struct{})
	}
	s.subs[userID][ch] = struct{}{}
	return ch
}

// Unsubscribe removes the channel for a user and closes it.
func (s *Subscriber) Unsubscribe(userID uuid.UUID, ch chan repository.NotificationRow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.subs[userID]; ok {
		delete(m, ch)
		if len(m) == 0 {
			delete(s.subs, userID)
		}
	}
	close(ch)
}

// Publish sends a notification row to all active subscribers for the user.
func (s *Subscriber) Publish(userID uuid.UUID, row repository.NotificationRow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs[userID] {
		select {
		case ch <- row:
		default:
			// drop if subscriber buffer is full (client too slow)
		}
	}
}
